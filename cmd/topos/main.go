// Command topos is the kernel binary: `topos serve` and
// `topos sync`.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/httpapi"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/supervisor"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "serve":
		if err := runServe(); err != nil {
			fatal(err)
		}
	case "sync":
		if err := runSync(); err != nil {
			fatal(err)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: topos <serve|sync>")
}

// shutdownSignals are the signals that must run the kernel's teardown
// rather than kill it outright. Ctrl-C on `make dev` sends SIGINT;
// `kill <pid>`, pkill and service-manager stops send SIGTERM. SIGKILL is
// deliberately absent because it cannot be caught — see runServe.
var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

// serverShutdownTimeout bounds how long the HTTP server is given to
// finish in-flight requests before the kernel proceeds to tear the plugin
// subprocesses down anyway. Reaping the subprocesses matters more than
// letting a slow request finish: a missed request is retried by the UI, a
// missed reap leaves a live orphan holding a source's store lock. No
// handler streams (no SSE/websocket on this listener), so in practice
// this completes in milliseconds and the timeout is a backstop only.
const serverShutdownTimeout = 10 * time.Second

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "topos:", err)
	os.Exit(1)
}

func configPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "topos", "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.toml"
	}
	return filepath.Join(home, ".config", "topos", "config.toml")
}

// pluginsDir resolves cfg.Plugins.Dir relative to the running executable
// when it is not already absolute.
func pluginsDir(cfg *config.Config) (string, error) {
	if filepath.IsAbs(cfg.Plugins.Dir) {
		return cfg.Plugins.Dir, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), cfg.Plugins.Dir), nil
}

func setupLogger() hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Name:  "topos",
		Level: hclog.Info,
	})
}

// setup loads the config store and opens the index. Callers must call
// store.Close(); the plugin host/coordinator/scheduler triple is built
// separately, by supervisor.NewSupervisor (below) — setup no longer
// builds it itself (07-02-PLAN.md Task 1: that boot sequence moved into
// kernel/supervisor so it is the ONE construction sequence runServe,
// runSync, and a future hot-apply all share, rather than being
// duplicated here and re-derived again inside Supervisor.Apply).
func setup(ctx context.Context, logger hclog.Logger) (*config.Store, *index.Store, error) {
	cfgStore, err := config.NewStore(configPath())
	if err != nil {
		return nil, nil, err
	}
	cfg := cfgStore.Expanded()

	store, err := index.Open(cfg.Index.Path)
	if err != nil {
		return nil, nil, err
	}

	return cfgStore, store, nil
}

func runSync() error {
	// Same reachability requirement as runServe: `defer sup.Shutdown()`
	// below is what kills the plugin subprocesses, and a signal that
	// terminates the process outright never runs it. NotifyContext is
	// enough here (unlike runServe, which must also drain a listener)
	// because cancelling ctx is what unblocks RefreshAll. Aborting a sync
	// mid-flight does NOT strand its sync_runs row: the coordinator
	// finalises on a detached context (kernel/syncer/coordinator.go,
	// context.WithoutCancel) precisely so a cancellation cannot destroy
	// the record of work that already happened.
	ctx, stop := signal.NotifyContext(context.Background(), shutdownSignals...)
	defer stop()
	logger := setupLogger()

	cfgStore, store, err := setup(ctx, logger)
	if err != nil {
		return err
	}
	defer store.Close()

	cfg := cfgStore.Expanded()
	pdir, err := pluginsDir(cfg)
	if err != nil {
		return err
	}

	sup, err := supervisor.NewSupervisor(ctx, store, cfgStore, pdir, logger)
	if err != nil {
		return err
	}
	defer sup.Shutdown()

	results := sup.Coordinator().RefreshAll(ctx)

	for _, r := range results {
		if r.Status == "error" {
			fmt.Printf("%s: error: %s\n", r.Source, r.Error)
			continue
		}
		fmt.Printf("%s: %d items\n", r.Source, r.ItemCount)
	}
	return nil
}

func runServe() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := setupLogger()

	cfgStore, store, err := setup(ctx, logger)
	if err != nil {
		return err
	}
	cfg := cfgStore.Expanded()
	defer store.Close()

	// Repair sync_runs rows stranded at "running" by a previous kernel that
	// died or was cancelled mid-sync. A freshly-started kernel has no
	// in-flight runs, so any such row is orphaned by definition — and
	// nothing else in the system ever finalises it, so without this it
	// survives every restart and keeps reporting its source as syncing.
	if n, err := store.ReconcileInterruptedSyncRuns(ctx); err != nil {
		// A failed repair must not stop the kernel booting: the worst case
		// is the pre-existing stale indicator, not a broken server.
		logger.Error("could not reconcile interrupted sync runs", "error", err.Error())
	} else if n > 0 {
		logger.Info("reconciled interrupted sync runs from a previous kernel session", "rows", n)
	}

	pdir, err := pluginsDir(cfg)
	if err != nil {
		return err
	}

	// supervisor.NewSupervisor performs the boot sequence Phase 1-6 built
	// directly into this function (Discover, ValidateMatchConfig, build
	// the coordinator, start the scheduler) — see kernel/supervisor for
	// why: a config save now needs to repeat this same sequence in place
	// (D-06/D-07), so it lives in one package the HTTP layer can also
	// call into via Supervisor.Apply, rather than being duplicated here.
	sup, err := supervisor.NewSupervisor(ctx, store, cfgStore, pdir, logger)
	if err != nil {
		return err
	}
	defer sup.Shutdown()

	// sup itself satisfies Fetcher/HealthProber/Refresher/Applier/Suspender
	// (delegating to its CURRENT host/coordinator on every call) — never
	// sup.Host()/sup.Coordinator() called once here, which would freeze
	// Router's refresher in particular at the coordinator Apply later
	// replaces wholesale (see Supervisor.Refresh's doc comment). pdir and
	// logger feed the plugin-type discovery/describe routes. Router's
	// second return value is the link-session store backing
	// POST/GET/DELETE /api/config/whatsapp-link (08-03-PLAN.md Task 3,
	// D-01) — its Shutdown, deferred below, terminates every live link
	// subprocess on kernel shutdown so none is ever left orphaned holding
	// a source's store lock, the same guarantee sup.Shutdown() already
	// gives every pluginhost-launched subprocess.
	router, linkStore := httpapi.Router(store, cfgStore, sup, sup, sup, sup, sup, pdir, logger)
	defer linkStore.Shutdown()

	listen := cfg.Server.Listen
	if !isLoopback(listen) {
		logger.Warn("kernel HTTP listener is not bound to loopback — this exposes the API beyond this machine", "listen", listen)
	}

	// The kernel MUST reach the deferred teardown above on every ordinary
	// exit path, which is why this is no longer a bare
	// `return http.ListenAndServe(...)`.
	//
	// hashicorp/go-plugin subprocesses do not die with their parent: the
	// plugin side explicitly swallows SIGINT ("Eat the interrupts",
	// go-plugin server.go) and has no parent-death watchdog, so a child
	// only exits when this process kills it. With no signal handler
	// installed, SIGINT/SIGTERM terminated the kernel with the Go
	// runtime's default disposition — and Go does not run deferred
	// functions on signal death, so `defer sup.Shutdown()` and
	// `defer linkStore.Shutdown()` above were unreachable and every
	// plugin subprocess was left alive, reparented to init.
	//
	// This was masked on the most-travelled path: `make dev`'s
	// `trap 'kill 0' INT TERM` turns Ctrl-C into a process-group SIGTERM,
	// which go-plugin does NOT swallow, so the children happened to die on
	// their own signal rather than by any cleanup here. Every other exit
	// path (bare `topos serve` + Ctrl-C, `kill <pid>`, a service-manager
	// stop) orphaned them.
	//
	// SIGKILL, a panic and an OOM-kill remain uncoverable — nothing
	// in-process can catch them. Those paths still orphan by design.
	srv := &http.Server{Addr: listen, Handler: router}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, shutdownSignals...)
	defer signal.Stop(sigCh)

	serveErr := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveErr <- err
	}()

	logger.Info("topos serving", "listen", listen)

	select {
	case err := <-serveErr:
		// The listener failed on its own (port in use, bind refused).
		// Returning the error preserves the existing contract `make dev`'s
		// startup guard relies on: a kernel that cannot listen exits non-zero.
		return err

	case sig := <-sigCh:
		// Restore default disposition immediately, so a SECOND Ctrl-C from
		// an operator who thinks the kernel has hung kills it outright
		// instead of being swallowed by this handler.
		signal.Reset(shutdownSignals...)
		logger.Info("shutdown signal received, terminating plugin subprocesses", "signal", sig.String())

		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), serverShutdownTimeout)
		defer cancelShutdown()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			// Never return here: an HTTP server that would not drain must
			// not stop the plugin teardown below, which is the whole point
			// of catching the signal.
			logger.Warn("HTTP server did not shut down cleanly, continuing to plugin teardown", "error", err.Error())
		}
		return nil
	}
}

// isLoopback reports whether listen (a host:port string) binds only to the
// loopback interface — the security default for Phase 1 (T-01-01): no LAN
// exposure ships without an explicit, logged warning.
func isLoopback(listen string) bool {
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		return false
	}
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}
