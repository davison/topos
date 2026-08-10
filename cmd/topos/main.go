// Command topos is the kernel binary: `topos serve` and
// `topos sync`.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

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
	ctx := context.Background()
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

	logger.Info("topos serving", "listen", listen)
	return http.ListenAndServe(listen, router)
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
