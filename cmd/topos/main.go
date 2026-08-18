// Command topos is the kernel binary: `topos serve` and
// `topos sync`.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/httpapi"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/pluginhost"
	"github.com/davison/topos/kernel/supervisor"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "serve":
		path := resolveConfigPath(parseConfigFlag("serve", os.Args[2:]))
		if err := runServe(path); err != nil {
			fatal(err)
		}
	case "sync":
		path := resolveConfigPath(parseConfigFlag("sync", os.Args[2:]))
		if err := runSync(path); err != nil {
			fatal(err)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: topos <serve|sync> [--config <path>]")
	fmt.Fprintln(os.Stderr, "  TOPOS_CONFIG can also set the config path (lower precedence than --config)")
}

// parseConfigFlag parses the given subcommand's own argv tail (os.Args[2:])
// for an optional --config flag, using a per-subcommand flag.FlagSet so
// "serve" and "sync" each report their own name on a parse error. Both
// "--config <path>" and "--config=<path>" are accepted (a FlagSet gives
// both forms for free). An unrecognised flag or a --config with no value
// prints this package's own usage() (naming TOPOS_CONFIG alongside the
// flag) and exits 2, the same contract main already applies to an unknown
// subcommand — flag.ContinueOnError's own message goes to os.Stderr first,
// via fs.SetOutput, so the operator sees both the specific parse error and
// the general usage line.
func parseConfigFlag(cmd string, args []string) string {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configFlag := fs.String("config", "", "path to the config file (overrides TOPOS_CONFIG and the XDG default)")
	if err := fs.Parse(args); err != nil {
		usage()
		os.Exit(2)
	}
	return *configFlag
}

// resolveConfigPath computes the config file path setup() loads, in
// precedence order: flagValue (the --config flag, highest); then the
// TOPOS_CONFIG environment variable; then configPath()'s own unchanged
// XDG_CONFIG_HOME / $HOME/.config / bare-"config.toml" fallback chain
// (lowest). flagValue is the empty string when --config was not given.
//
// A non-empty flagValue or TOPOS_CONFIG value is returned VERBATIM — a
// relative value is resolved later against the process's own working
// directory when the path is opened, never joined against
// os.Executable()'s directory the way pluginsDir/externalPluginsDir
// resolve cfg.Plugins.Dir/ExternalDir. That executable-relative
// convention exists so a built binary's relative plugin directory always
// resolves next to the binary regardless of the caller's cwd — the right
// rule for a config-declared path. A --config value is different in
// kind: it is typed on a command line (or set in TOPOS_CONFIG) by
// whoever is running the process, and the dev loop starts the kernel via
// `go run`, whose own os.Executable() lives in a throwaway build
// directory under the Go build cache that never holds a config file.
// Routing --config through the executable-relative rule would silently
// resolve `go run ./cmd/topos serve --config config.dev.toml` (run from
// a checkout root) into a nonexistent path in that temp directory
// instead of the config.dev.toml sitting right there. A path typed on a
// command line is expected to mean what the shell means by it: relative
// to the directory the command was run from — do NOT call
// os.Executable() from this function.
func resolveConfigPath(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("TOPOS_CONFIG"); env != "" {
		return env
	}
	return configPath()
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
// when it is not already absolute. The thin wrapper exists so
// resolvePluginsDir stays pure (no os.Executable call) and every one of
// its branches can be pinned by tests that build real directory trees.
func pluginsDir(cfg *config.Config) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return resolvePluginsDir(cfg, filepath.Dir(exe)), nil
}

// resolvePluginsDir computes the trusted plugins directory from
// cfg.Plugins.Dir and the running executable's directory (exeDir), in
// three ordered branches:
//
//  1. An absolute cfg.Plugins.Dir is returned verbatim — an operator who
//     typed an absolute path gets exactly that path, regardless of what
//     exists on disk or what the executable's directory is named.
//  2. Otherwise the executable-relative join (exeDir + the relative
//     configured value) is used when it exists as a directory — the
//     checkout layout, where bin/topos sits beside bin/plugins/.
//  3. Otherwise, when exeDir itself is named "bin", the installed-layout
//     sibling formed from exeDir's parent plus "lib", "topos", and the
//     configured relative value is used if it exists as a directory.
//     This second probe exists because `make install` writes the kernel
//     to <prefix>/bin and its plugins to <prefix>/lib/topos/plugins
//     (INST-01) — so an installed $PREFIX/bin/topos with the stock
//     relative `[plugins] dir = "plugins"` value would otherwise look
//     for $PREFIX/bin/plugins, a directory the installer never creates
//     (INST-03). The existence probes — not a compiled-in prefix — are
//     what let the same published binary serve both a checkout build and
//     an installed instance.
//
// When neither candidate exists, the executable-relative join is
// returned anyway, so a "no plugins found" failure still names the
// primary, documented location rather than the installed-layout
// fallback.
func resolvePluginsDir(cfg *config.Config, exeDir string) string {
	if filepath.IsAbs(cfg.Plugins.Dir) {
		return cfg.Plugins.Dir
	}
	primary := filepath.Join(exeDir, cfg.Plugins.Dir)
	if isDir(primary) {
		return primary
	}
	if filepath.Base(exeDir) == "bin" {
		sibling := filepath.Join(filepath.Dir(exeDir), "lib", "topos", cfg.Plugins.Dir)
		if isDir(sibling) {
			return sibling
		}
	}
	return primary
}

// isDir reports whether path exists and is a directory. Any stat error
// (not-exist, permission, anything else) counts as "not a directory" —
// resolvePluginsDir's probes must never turn a stat failure into a
// startup error, only into falling through to the next candidate.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// defaultExternalPluginsDir resolves the per-OS platform data directory
// Phase 11's external plugin tier defaults to when
// PluginsConfig.ExternalDir is unset (D-09): "$XDG_DATA_HOME/topos/
// plugins-external" (falling back to "~/.local/share/topos/
// plugins-external" when XDG_DATA_HOME is unset or empty) on Linux and
// every other non-darwin, non-windows GOOS; "~/Library/Application
// Support/topos/plugins-external" on macOS; "%LOCALAPPDATA%\topos\
// plugins-external" on Windows. Portable resolution without committing to
// Windows support — the project stays Linux-anchored (D-Bus keyring,
// desktop chat DBs) — this helper only avoids a hard-coded Linux-only
// path assumption leaking into a config key an operator on another OS
// would have to override by hand. Never creates the directory: a missing
// external directory is a legitimate empty tier, exactly like a missing
// trusted directory already is.
func defaultExternalPluginsDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "topos", "plugins-external"), nil
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "topos", "plugins-external"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, "AppData", "Local", "topos", "plugins-external"), nil
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, "topos", "plugins-external"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, ".local", "share", "topos", "plugins-external"), nil
	}
}

// externalPluginsDir resolves cfg.Plugins.ExternalDir: an explicitly
// configured value is used verbatim when absolute, or resolved relative
// to the running executable when relative — the identical convention
// pluginsDir already applies to cfg.Plugins.Dir, extended here so the two
// directories behave identically under an operator-authored relative
// path. An empty (omitted) ExternalDir resolves to
// defaultExternalPluginsDir() instead — the zero-config, drop-a-binary-in
// default (D-09).
func externalPluginsDir(cfg *config.Config) (string, error) {
	if cfg.Plugins.ExternalDir == "" {
		return defaultExternalPluginsDir()
	}
	if filepath.IsAbs(cfg.Plugins.ExternalDir) {
		return cfg.Plugins.ExternalDir, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), cfg.Plugins.ExternalDir), nil
}

// registerManifestMimeType registers ".webmanifest" -> "application/
// manifest+json" with the Go stdlib's mime package once, before the HTTP
// server is constructed. spaHandler (kernel/httpapi/routes.go) serves the
// embedded SPA build — including the PWA's generated manifest.webmanifest
// — via http.FileServer, which infers Content-Type through
// mime.TypeByExtension; that function falls through to the HOST OS's own
// mime.types database for any extension Go doesn't hard-code, and
// ".webmanifest" is not universally pre-registered there across every
// Linux distro, macOS, or Windows install (13-RESEARCH.md Pitfall 4).
// Browsers can be strict about a manifest's declared Content-Type when
// deciding installability, so this line must not be deleted as
// redundant even on a machine where it happens to be a no-op today. A
// registration failure is logged and otherwise ignored — the worst case
// is falling back to the host's own (possibly-correct) mime database,
// never a reason to fail kernel startup.
func registerManifestMimeType(logger hclog.Logger) {
	if err := mime.AddExtensionType(".webmanifest", "application/manifest+json"); err != nil {
		logger.Warn("could not register .webmanifest mime type, falling back to the host's own mime database", "error", err.Error())
	}
}

func setupLogger() hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Name:  "topos",
		Level: hclog.Info,
	})
}

// bootstrapConfig makes a genuinely-missing config.toml a first-run event
// instead of a fatal error (09.1-BOOTSTRAP): it writes a canonical default
// config at path and reports whether it did so. Kept as its own function
// (rather than inlined in setup) specifically so it is unit-testable —
// setup itself also opens the index at the configured path, which a unit
// test must not do on a developer's real machine.
//
// The single errors.Is(loadErr, os.ErrNotExist) gate below is the whole
// safety property (RESEARCH Pitfall 3, threat T-09.1-B1): a malformed
// TOML file, a permission-denied read, a directory sitting at path, or a
// validation failure on an existing file are all NOT os.ErrNotExist, so
// they fall straight through to (false, nil) and reach fatal(err) at the
// call site exactly as they do today — bootstrap fires only when the file
// genuinely does not exist, never as a substitute for a broken one.
func bootstrapConfig(path string, loadErr error, logger hclog.Logger) (bool, error) {
	if !errors.Is(loadErr, os.ErrNotExist) {
		return false, nil
	}

	// The config directory is a genuine gap, not a safety net that already
	// exists: nothing in kernel/config or cmd/topos creates it today, and
	// WriteCanonical's own os.CreateTemp(dir, ...) fails with ENOENT on a
	// machine that has never had a ~/.config/topos/. 0o700 (not 0o755) is
	// deliberate — this directory can later hold a config.toml carrying
	// ${VAR} references and per-source connection details (T-09.1-B2).
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, fmt.Errorf("bootstrap config dir %s: %w", filepath.Dir(path), err)
	}

	// Only the raw, secret-free DefaultConfig() is ever passed here — never
	// any expanded/secret-resolved config, per Phase 7 D-05 and
	// WriteCanonical's own doc comment (threat T-09.1-B3).
	if err := config.WriteCanonical(path, config.DefaultConfig()); err != nil {
		return false, fmt.Errorf("bootstrap config write %s: %w", path, err)
	}

	logger.Info("no config file found, wrote a default config", "path", path)
	return true, nil
}

// setup loads the config store and opens the index. Callers must call
// store.Close(); the plugin host/coordinator/scheduler triple is built
// separately, by supervisor.NewSupervisor (below) — setup no longer
// builds it itself (07-02-PLAN.md Task 1: that boot sequence moved into
// kernel/supervisor so it is the ONE construction sequence runServe,
// runSync, and a future hot-apply all share, rather than being
// duplicated here and re-derived again inside Supervisor.Apply).
//
// setup shares one bootstrapConfig gate for both runServe and runSync
// (09.1-BOOTSTRAP, planner_resolutions R2) — a fresh `topos sync` reports
// zero sources honestly rather than dying, and the INFO log line
// bootstrapConfig emits is what makes that self-explaining rather than
// silent.
//
// path is the already-resolved config path (resolveConfigPath's result,
// threaded down from main via runServe/runSync) — setup no longer
// computes it itself; every other behaviour (the bootstrapConfig
// first-run gate, the re-load through config.NewStore after a bootstrap
// write, returning the ORIGINAL load error when the failure was not
// os.ErrNotExist) is unchanged.
func setup(ctx context.Context, logger hclog.Logger, path string) (*config.Store, *index.Store, error) {
	cfgStore, err := config.NewStore(path)
	if err != nil {
		wrote, bootErr := bootstrapConfig(path, err, logger)
		if bootErr != nil {
			return nil, nil, bootErr
		}
		if !wrote {
			// Not a missing-file case (or bootstrapConfig itself declined) —
			// the ORIGINAL load error is the one that names the actual
			// problem (malformed TOML, permission denied, etc.) and must
			// reach the caller unchanged rather than a generic one.
			return nil, nil, err
		}
		// Re-load through the normal path rather than constructing a Store
		// in memory: this proves the just-written file actually round-trips
		// through real validation on the user's own machine, so a default
		// that somehow failed validation surfaces here at first run instead
		// of at some later reload.
		cfgStore, err = config.NewStore(path)
		if err != nil {
			return nil, nil, err
		}
	}
	cfg := cfgStore.Expanded()

	store, err := index.Open(cfg.Index.Path)
	if err != nil {
		return nil, nil, err
	}

	return cfgStore, store, nil
}

func runSync(path string) error {
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

	cfgStore, store, err := setup(ctx, logger, path)
	if err != nil {
		return err
	}
	defer store.Close()

	cfg := cfgStore.Expanded()
	pdir, err := pluginsDir(cfg)
	if err != nil {
		return err
	}
	extdir, err := externalPluginsDir(cfg)
	if err != nil {
		return err
	}
	dirs := pluginhost.Dirs{Trusted: pdir, External: extdir}

	sup, err := supervisor.NewSupervisor(ctx, store, cfgStore, dirs, logger)
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

func runServe(path string) error {
	// Register the shutdown-signal handler BEFORE anything below can spawn
	// a plugin subprocess — supervisor.NewSupervisor launches every
	// configured plugin during construction, and hashicorp/go-plugin
	// children do not die with their parent. With registration at the old
	// position (just before the accept loop), there was a real startup
	// window — plugins alive, handler not yet installed — in which a
	// SIGINT/SIGTERM killed the kernel with the Go runtime's default
	// disposition: no deferred teardown ran and any child that had already
	// written its go-plugin handshake line survived as a live orphan.
	// That is exactly the window a service-manager stop lands in when it
	// arrives during startup, and the window the shutdown_signal_test
	// regression gate hits on a slow CI runner (it signals at the first
	// observable plugin child). A signal arriving mid-boot is simply held
	// in sigCh until the select at the bottom of this function, so boot
	// completes and the ordinary graceful teardown runs — the same
	// register-before-spawn ordering runSync has always had via
	// NotifyContext.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, shutdownSignals...)
	defer signal.Stop(sigCh)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := setupLogger()

	registerManifestMimeType(logger)

	cfgStore, store, err := setup(ctx, logger, path)
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
	extdir, err := externalPluginsDir(cfg)
	if err != nil {
		return err
	}
	dirs := pluginhost.Dirs{Trusted: pdir, External: extdir}

	// supervisor.NewSupervisor performs the boot sequence Phase 1-6 built
	// directly into this function (Discover, ValidateMatchConfig, build
	// the coordinator, start the scheduler) — see kernel/supervisor for
	// why: a config save now needs to repeat this same sequence in place
	// (D-06/D-07), so it lives in one package the HTTP layer can also
	// call into via Supervisor.Apply, rather than being duplicated here.
	sup, err := supervisor.NewSupervisor(ctx, store, cfgStore, dirs, logger)
	if err != nil {
		return err
	}
	defer sup.Shutdown()

	// sup itself satisfies Fetcher/HealthProber/Refresher/Applier/Suspender/
	// PluginIconProvider (09-01-PLAN.md Task 2) (delegating to its CURRENT
	// host/coordinator on every call) — never
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
	router, linkStore := httpapi.Router(store, cfgStore, sup, sup, sup, sup, sup, sup, dirs, logger)
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
	//
	// sigCh itself is registered at the TOP of this function — before the
	// supervisor spawns any plugin — so no signal can slip through the
	// startup window; see the comment there.
	srv := &http.Server{Addr: listen, Handler: router}

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
