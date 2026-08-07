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
	"github.com/davison/topos/kernel/correlate"
	"github.com/davison/topos/kernel/httpapi"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/pluginhost"
	"github.com/davison/topos/kernel/syncer"
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

// setup loads the config store, opens the index, and launches every
// configured plugin. Callers must call host.Shutdown() and store.Close().
//
// setup returns a *config.Store rather than a bare *config.Config
// (assumption-delta decision, 07-01-PLAN.md: the running configuration is
// now a live, versioned resource — see kernel/config/store.go). Every
// caller below that still needs a plain *config.Config (pluginsDir,
// pluginhost.Discover, pluginhost.ValidateMatchConfig, newCoordinator,
// syncer.Scheduler) takes cfgStore.Expanded() for now — a deliberately
// temporary boot-time snapshot the same way Router's own three legacy
// handlers do (routes.go), not yet re-launched or re-registered on a
// config save (D-06/D-07's hot-apply reconcile is 07-02+ scope). Only
// httpapi.Router receives the *config.Store itself, so the HTTP surface's
// stream/config routes see a save immediately.
func setup(ctx context.Context, logger hclog.Logger) (*config.Store, *index.Store, *pluginhost.Host, error) {
	cfgStore, err := config.NewStore(configPath())
	if err != nil {
		return nil, nil, nil, err
	}
	cfg := cfgStore.Expanded()

	store, err := index.Open(cfg.Index.Path)
	if err != nil {
		return nil, nil, nil, err
	}

	pdir, err := pluginsDir(cfg)
	if err != nil {
		store.Close()
		return nil, nil, nil, err
	}

	host, err := pluginhost.Discover(ctx, pdir, cfg.Sources, logger)
	if err != nil {
		store.Close()
		return nil, nil, nil, err
	}

	// D-05's second validation phase: cross-check every webspace's match
	// configuration against each launched plugin's own declared
	// vocabulary. This must happen after Discover (it needs the launched
	// *Host) and before any sync — a rejected config must never leave
	// subprocesses running, so both the host and the store are torn down
	// on failure here, exactly as the two error paths above already do.
	if err := pluginhost.ValidateMatchConfig(cfg, host); err != nil {
		host.Shutdown()
		store.Close()
		return nil, nil, nil, err
	}

	return cfgStore, store, host, nil
}

func sourcesFromHost(host *pluginhost.Host) []correlate.Source {
	plugins := host.Plugins()
	sources := make([]correlate.Source, len(plugins))
	for i, p := range plugins {
		sources[i] = p
	}
	return sources
}

// newCoordinator builds the correlate.Engine + syncer.Coordinator pair
// every sync in the system — the scheduler, the manual refresh routes,
// and this CLI — must go through. Neither runSync nor runServe may call
// the correlation engine's sync methods directly; the coordinator is the
// only entry point (D-06).
func newCoordinator(store *index.Store, cfg *config.Config, host *pluginhost.Host) *syncer.Coordinator {
	engine := &correlate.Engine{Store: store, Config: cfg}
	return syncer.NewCoordinator(store, engine, sourcesFromHost(host))
}

func runSync() error {
	ctx := context.Background()
	logger := setupLogger()

	cfgStore, store, host, err := setup(ctx, logger)
	if err != nil {
		return err
	}
	defer host.Shutdown()
	defer store.Close()

	coord := newCoordinator(store, cfgStore.Expanded(), host)
	results := coord.RefreshAll(ctx)

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

	cfgStore, store, host, err := setup(ctx, logger)
	if err != nil {
		return err
	}
	cfg := cfgStore.Expanded()
	defer host.Shutdown()
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

	coord := newCoordinator(store, cfg, host)

	// Background scheduler (KERN-04, D-05): one goroutine per configured
	// source, first run immediate (replacing Phase 1's one-shot startup
	// goroutine), then repeating at each source's resolved sync_interval.
	// Cancelled via the same ctx that's cancelled when runServe returns.
	sched := &syncer.Scheduler{Coordinator: coord, Config: cfg, Logger: logger}
	go sched.Run(ctx)

	router := httpapi.Router(store, cfgStore, host, host, coord)

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
