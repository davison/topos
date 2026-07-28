// Command webspaces is the kernel binary: `webspaces serve` and
// `webspaces sync`.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/webspaces/kernel/config"
	"github.com/davison/webspaces/kernel/correlate"
	"github.com/davison/webspaces/kernel/httpapi"
	"github.com/davison/webspaces/kernel/index"
	"github.com/davison/webspaces/kernel/pluginhost"
	"github.com/davison/webspaces/kernel/syncer"
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
	fmt.Fprintln(os.Stderr, "usage: webspaces <serve|sync>")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "webspaces:", err)
	os.Exit(1)
}

func configPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "webspaces", "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.toml"
	}
	return filepath.Join(home, ".config", "webspaces", "config.toml")
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
		Name:  "webspaces",
		Level: hclog.Info,
	})
}

// setup loads config, opens the index, and launches every configured
// plugin. Callers must call host.Shutdown() and store.Close().
func setup(ctx context.Context, logger hclog.Logger) (*config.Config, *index.Store, *pluginhost.Host, error) {
	cfg, err := config.Load(configPath())
	if err != nil {
		return nil, nil, nil, err
	}

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

	return cfg, store, host, nil
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

	cfg, store, host, err := setup(ctx, logger)
	if err != nil {
		return err
	}
	defer host.Shutdown()
	defer store.Close()

	coord := newCoordinator(store, cfg, host)
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

	cfg, store, host, err := setup(ctx, logger)
	if err != nil {
		return err
	}
	defer host.Shutdown()
	defer store.Close()

	coord := newCoordinator(store, cfg, host)

	// Background scheduler (KERN-04, D-05): one goroutine per configured
	// source, first run immediate (replacing Phase 1's one-shot startup
	// goroutine), then repeating at each source's resolved sync_interval.
	// Cancelled via the same ctx that's cancelled when runServe returns.
	sched := &syncer.Scheduler{Coordinator: coord, Config: cfg, Logger: logger}
	go sched.Run(ctx)

	router := httpapi.Router(store, cfg, host)

	listen := cfg.Server.Listen
	if !isLoopback(listen) {
		logger.Warn("kernel HTTP listener is not bound to loopback — this exposes the API beyond this machine", "listen", listen)
	}

	logger.Info("webspaces serving", "listen", listen)
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
