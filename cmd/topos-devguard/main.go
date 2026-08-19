// Command topos-devguard is the pre-flight isolation gate for dev runs
// (ISOL-01/ISOL-02). It exists to prevent one concrete defect class: a
// dev kernel started from a checkout reading the INSTALLED instance's
// config (which resolves real credentials from the environment),
// writing its kernel index, or re-linking a chat session whose store
// the installed instance is using — damage that is invisible until it
// has already happened.
//
// The guard loads the dev config through kernel/config's own store
// constructor — the same parser, defaulting, environment expansion and
// validation the kernel itself uses, so the config the kernel would
// accept and the config the guard inspects can never be two different
// documents. It then derives the two topos-owned roots exactly as the
// kernel derives its own locations (cmd/topos's configPath and
// defaultExternalPluginsDir):
//
//   - the config root: $XDG_CONFIG_HOME/topos, else ~/.config/topos
//   - the state root:  $XDG_DATA_HOME/topos,  else ~/.local/share/topos
//
// and refuses (exit 1) when any writable path the config declares — the
// config file itself, the index, the trusted or external plugin
// directory (INCLUDING the omitted-key default, which resolves inside
// the state root), or any source's own store path — resolves inside
// either root. Containment is decided on resolved path components,
// never string prefixes, so a sibling like ".../topos-extra" is clean
// while a path equal to a root is a violation. Every violation is
// reported in one pass, sorted by config key path for byte-identical
// output across runs.
//
// A source's read-only data location OUTSIDE the topos roots (a chat
// application's own directory, a document tree) is deliberately never a
// violation — only the topos-owned roots are in scope.
//
// Modes: --warn-only prints the identical findings to stderr behind a
// loud bypass banner and exits 0 (there is no partial or per-key form —
// the guard either refuses or announces everything it permits);
// --expected-port N additionally reports a violation when the config's
// [server] listen port differs from N (ISOL-02's stale-dev-config
// catch).
package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/davison/topos/kernel/config"
)

// root is one topos-owned protected root.
type root struct {
	name string // human name, e.g. "topos config root"
	path string // absolute, cleaned
}

// violation is one finding, keyed by the config key path it names.
type violation struct {
	key     string // config key path, e.g. "[index] path" — the sort key
	message string // full report line (without the "devguard: " prefix)
}

// deriveRoots computes the two protected roots from the same
// environment variables the kernel reads, with the same fallbacks
// (cmd/topos's configPath and defaultExternalPluginsDir Linux branch),
// so a machine with a non-default config or data home is protected at
// its real locations.
func deriveRoots() (configRoot, stateRoot root, err error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		configRoot = root{name: "topos config root", path: filepath.Join(xdg, "topos")}
	} else {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return root{}, root{}, fmt.Errorf("resolve home directory: %w", herr)
		}
		configRoot = root{name: "topos config root", path: filepath.Join(home, ".config", "topos")}
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		stateRoot = root{name: "topos state root", path: filepath.Join(xdg, "topos")}
	} else {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return root{}, root{}, fmt.Errorf("resolve home directory: %w", herr)
		}
		stateRoot = root{name: "topos state root", path: filepath.Join(home, ".local", "share", "topos")}
	}
	return configRoot, stateRoot, nil
}

// containedIn reports whether candidate is the root itself or sits
// under it, decided on path components after both are cleaned and made
// absolute — never on string prefixes, so "/x/topos-extra" is NOT
// inside "/x/topos". This is the single containment decision every
// candidate goes through; no call site implements its own variant.
func containedIn(candidate string, r root) bool {
	c := filepath.Clean(candidate)
	rp := filepath.Clean(r.path)
	if c == rp {
		return true
	}
	rel, err := filepath.Rel(rp, c)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// expandHome expands a leading "~" the way the kernel's own loader does
// for the fields it covers — the guard applies it to candidates the
// loader deliberately does not expand (a source's own path is expanded
// by the plugin subprocess, not the kernel).
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return strings.Replace(path, "~", home, 1)
}

// absolutize expands, absolutizes (against baseDir when relative), and
// cleans a candidate path.
func absolutize(path, baseDir string) string {
	p := expandHome(path)
	if !filepath.IsAbs(p) {
		p = filepath.Join(baseDir, p)
	}
	return filepath.Clean(p)
}

// externalDirDefault mirrors cmd/topos's defaultExternalPluginsDir for
// the omitted-key case, per-OS branches included, so an omitted
// external_dir is checked at the path the kernel would actually use.
func externalDirDefault() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "topos", "plugins-external"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "topos", "plugins-external"), nil
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("topos-devguard", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configFlag := fs.String("config", "", "path to the dev config file to inspect (required)")
	warnOnly := fs.Bool("warn-only", false, "report findings as warnings on stderr and exit 0 — the explicit, loud isolation bypass")
	expectedPort := fs.Int("expected-port", 0, "when non-zero, additionally require [server] listen to name this port")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *configFlag == "" {
		fmt.Fprintln(stderr, "devguard: --config is required")
		fs.Usage()
		return 1
	}

	configPath, err := filepath.Abs(*configFlag)
	if err != nil {
		fmt.Fprintf(stderr, "devguard: resolve config path: %v\n", err)
		return 1
	}
	configDir := filepath.Dir(configPath)

	// ONE config-loading call site, through the kernel's own store — the
	// guard never grows a second TOML parser.
	store, err := config.NewStore(configPath)
	if err != nil {
		fmt.Fprintf(stderr, "devguard: load config: %v\n", err)
		return 1
	}
	cfg := store.Expanded()

	configRoot, stateRoot, err := deriveRoots()
	if err != nil {
		fmt.Fprintf(stderr, "devguard: %v\n", err)
		return 1
	}
	roots := []root{configRoot, stateRoot}

	// Candidates: config key path -> resolved absolute path. The
	// trusted and external plugin directories follow the kernel's
	// resolution as far as a guard without the kernel's executable path
	// can: absolute values verbatim, relative values against the config
	// file's directory (the checkout — for a dev run the kernel's own
	// executable lives in a throwaway go-run build directory that is
	// never inside a topos root).
	type candidate struct{ key, path string }
	candidates := []candidate{
		{"config", configPath},
		{"[index] path", absolutize(cfg.Index.Path, configDir)},
		{"[plugins] dir", absolutize(cfg.Plugins.Dir, configDir)},
	}
	if cfg.Plugins.ExternalDir == "" {
		// The omitted key is NOT a pass: the kernel resolves it to the
		// platform data default inside the state root.
		def, derr := externalDirDefault()
		if derr != nil {
			fmt.Fprintf(stderr, "devguard: %v\n", derr)
			return 1
		}
		candidates = append(candidates, candidate{"[plugins] external_dir", def})
	} else {
		candidates = append(candidates, candidate{"[plugins] external_dir", absolutize(cfg.Plugins.ExternalDir, configDir)})
	}
	// A source's own path is resolved by the PLUGIN SUBPROCESS against
	// its working directory — the kernel launches plugins with no
	// cmd.Dir override, so the real resolution base is the kernel
	// process's own cwd, not the config file's directory. The guard
	// mirrors that: relative source paths resolve against the guard's
	// own working directory, which under `make dev` is the same
	// checkout root the kernel is started from. Resolving against the
	// config directory here produced a false clear when the guard was
	// invoked from a cwd inside a topos root against a config kept
	// elsewhere (15-VERIFICATION.md gap 1).
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "devguard: resolve working directory: %v\n", err)
		return 1
	}
	for name, src := range cfg.Sources {
		if strings.TrimSpace(src.Path) == "" {
			continue
		}
		candidates = append(candidates, candidate{
			fmt.Sprintf("[sources.%s] path", name),
			absolutize(src.Path, cwd),
		})
	}

	var violations []violation
	for _, c := range candidates {
		for _, r := range roots {
			if containedIn(c.path, r) {
				violations = append(violations, violation{
					key: c.key,
					message: fmt.Sprintf("%s -> %s (inside %s %s)",
						c.key, c.path, r.name, r.path),
				})
				break
			}
		}
	}

	// Expected-port comparison (ISOL-02): a stale dev config still
	// naming the installed instance's port fails here by name, before
	// the dev recipe's readiness gate could mask it as a timeout.
	if *expectedPort != 0 {
		_, portStr, perr := net.SplitHostPort(cfg.Server.Listen)
		if perr != nil {
			violations = append(violations, violation{
				key:     "[server] listen",
				message: fmt.Sprintf("[server] listen -> %q is not host:port: %v", cfg.Server.Listen, perr),
			})
		} else if portStr != fmt.Sprintf("%d", *expectedPort) {
			violations = append(violations, violation{
				key: "[server] listen",
				message: fmt.Sprintf("[server] listen -> port %s differs from the expected dev port %d — a dev config generated before the port move still names the old port; delete the generated dev config and re-run so the generator writes a fresh one",
					portStr, *expectedPort),
			})
		}
	}

	// One pass, deterministic order: sorted by config key path so two
	// runs over one config are byte-identical despite Go's randomised
	// map iteration.
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].key != violations[j].key {
			return violations[i].key < violations[j].key
		}
		return violations[i].message < violations[j].message
	})

	if len(violations) == 0 {
		fmt.Fprintf(stdout, "devguard: OK — %s declares no path inside %s or %s\n",
			configPath, configRoot.path, stateRoot.path)
		return 0
	}

	if *warnOnly {
		// The bypass is loud and total: there is no per-key
		// suppression — every permitted path is announced.
		fmt.Fprintln(stderr, "devguard: ############################################################")
		fmt.Fprintln(stderr, "devguard: WARNING: ISOLATION BYPASSED (warn-only mode)")
		fmt.Fprintln(stderr, "devguard: This dev run is being PERMITTED to reach paths the installed")
		fmt.Fprintln(stderr, "devguard: topos instance owns. Every permitted path is listed below.")
		fmt.Fprintln(stderr, "devguard: ############################################################")
		for _, v := range violations {
			fmt.Fprintf(stderr, "devguard: PERMITTED VIOLATION: %s\n", v.message)
		}
		return 0
	}

	for _, v := range violations {
		fmt.Fprintf(stdout, "devguard: VIOLATION: %s\n", v.message)
	}
	fmt.Fprintf(stdout, "devguard: %d violation(s) — refusing to let a dev run reach the installed instance's config or state\n", len(violations))
	return 1
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
