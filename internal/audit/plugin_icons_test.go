package audit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// maxIconBytes mirrors kernel/pluginhost.MaxIconBytes (09-01-PLAN.md) — an
// icon asset must fit under the same ceiling the kernel enforces at
// capture time, or a plugin that "passes" this audit would still have its
// icon silently dropped at launch.
const maxIconBytes = 65536

// provenanceKeys are the four keys a plugin's go:embed provenance comment
// must carry, in order, for Task 3's audit to parse it mechanically
// (09-02-PLAN.md Task 1/2/3). Order is documented here for humans; the
// scanner below checks presence, not sequence, since Go comment wrapping
// can legitimately vary the exact column each key starts on.
var provenanceKeys = []string{"Source-Project", "Source-File", "Source-Version", "Source-License"}

// goEmbedIconDirective matches a //go:embed directive whose target is
// this repo's own icon asset convention — "assets/icon.svg" — with no
// other embed patterns (glob, multiple paths) permitted for an icon
// directive.
var goEmbedIconDirective = regexp.MustCompile(`^//go:embed\s+assets/icon\.svg\s*$`)

// pluginIconOffenses runs every <behavior> check from 09-02-PLAN.md Task 3
// against one plugin module directory (a directory directly under
// plugins/ containing its own go.mod) and returns one human-readable
// offense string per violation found. An empty slice means the plugin
// fully satisfies the icon contract.
func pluginIconOffenses(pluginDir string) []string {
	var offenses []string
	name := filepath.Base(pluginDir)

	iconPath := filepath.Join(pluginDir, "assets", "icon.svg")
	iconBytes, err := os.ReadFile(iconPath)
	switch {
	case os.IsNotExist(err):
		offenses = append(offenses, fmt.Sprintf("%s: missing assets/icon.svg", name))
	case err != nil:
		offenses = append(offenses, fmt.Sprintf("%s: reading assets/icon.svg: %v", name, err))
	default:
		if len(iconBytes) >= maxIconBytes {
			offenses = append(offenses, fmt.Sprintf(
				"%s: assets/icon.svg is %d bytes, must be under %d (kernel/pluginhost.MaxIconBytes)",
				name, len(iconBytes), maxIconBytes,
			))
		}
		if strings.Contains(string(iconBytes), "currentColor") {
			offenses = append(offenses, fmt.Sprintf(
				"%s: assets/icon.svg contains \"currentColor\" — an img-loaded SVG cannot inherit page CSS; bake the literal --muted-foreground hex instead (09-UI-SPEC.md Fix 10)",
				name,
			))
		}
	}

	goFiles, globErr := filepath.Glob(filepath.Join(pluginDir, "*.go"))
	if globErr != nil {
		offenses = append(offenses, fmt.Sprintf("%s: glob *.go: %v", name, globErr))
		return offenses
	}

	foundDirective := false
	foundDescribeResponse := false

	for _, goFile := range goFiles {
		if strings.HasSuffix(goFile, "_test.go") {
			continue
		}

		data, readErr := os.ReadFile(goFile)
		if readErr != nil {
			offenses = append(offenses, fmt.Sprintf("%s: reading %s: %v", name, filepath.Base(goFile), readErr))
			continue
		}
		lines := strings.Split(string(data), "\n")

		for i, line := range lines {
			if !goEmbedIconDirective.MatchString(strings.TrimRight(line, "\r")) {
				continue
			}
			foundDirective = true

			// Walk upward collecting the contiguous "//"-prefixed comment
			// block directly above the directive (stop at the first
			// non-comment line, or the top of the file).
			var block []string
			for j := i - 1; j >= 0; j-- {
				trimmed := strings.TrimSpace(lines[j])
				if !strings.HasPrefix(trimmed, "//") {
					break
				}
				block = append(block, trimmed)
			}
			blockText := strings.Join(block, "\n")

			for _, key := range provenanceKeys {
				if !strings.Contains(blockText, key) {
					offenses = append(offenses, fmt.Sprintf(
						"%s: %s's go:embed directive at line %d is missing provenance key %q in its preceding comment block",
						name, filepath.Base(goFile), i+1, key,
					))
				}
			}
		}

		// Icon/IconMime completeness: parse the file's AST and inspect
		// every composite literal that builds a *DescribeResponse,
		// regardless of the selector's package alias.
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, goFile, data, parser.SkipObjectResolution)
		if parseErr != nil {
			offenses = append(offenses, fmt.Sprintf("%s: parse %s: %v", name, filepath.Base(goFile), parseErr))
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			sel, ok := lit.Type.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil || sel.Sel.Name != "DescribeResponse" {
				return true
			}
			foundDescribeResponse = true

			hasIcon, hasIconMime := false, false
			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				switch key.Name {
				case "Icon":
					hasIcon = true
				case "IconMime":
					hasIconMime = true
				}
			}
			if !hasIcon || !hasIconMime {
				offenses = append(offenses, fmt.Sprintf(
					"%s: %s: DescribeResponse literal at %s sets Icon=%v IconMime=%v — a plugin declaring one without the other fails the audit",
					name, filepath.Base(goFile), fset.Position(lit.Pos()), hasIcon, hasIconMime,
				))
			}
			return true
		})
	}

	if !foundDirective {
		offenses = append(offenses, fmt.Sprintf("%s: no //go:embed assets/icon.svg directive found in any non-test .go file", name))
	}
	if !foundDescribeResponse {
		offenses = append(offenses, fmt.Sprintf("%s: no DescribeResponse composite literal found in any non-test .go file", name))
	}

	return offenses
}

// TestPluginIcons_EveryInRepoPluginDeclaresOne walks plugins/ (relative to
// repoRoot, reusing the same repoRoot outbound_hosts_test.go already
// established) for directories containing their own go.mod — never a
// hardcoded plugin-name list, per D-05 ("the kernel holds no built-in
// table of known plugin types") applied here to the audit itself, so a
// future eighth plugin is caught automatically — and asserts every
// <behavior> case from 09-02-PLAN.md Task 3 for each one discovered:
//
//   - assets/icon.svg exists and is under maxIconBytes
//   - assets/icon.svg never contains "currentColor"
//   - the plugin's go:embed directive carries all four provenance keys
//     directly above it
//   - every DescribeResponse literal sets both Icon and IconMime
func TestPluginIcons_EveryInRepoPluginDeclaresOne(t *testing.T) {
	pluginsDir := filepath.Join(repoRoot, "plugins")
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		t.Fatalf("read %s: %v", pluginsDir, err)
	}

	discovered := 0
	var offenses []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginDir := filepath.Join(pluginsDir, entry.Name())
		if _, statErr := os.Stat(filepath.Join(pluginDir, "go.mod")); statErr != nil {
			continue
		}
		discovered++
		offenses = append(offenses, pluginIconOffenses(pluginDir)...)
	}

	if discovered < 2 {
		t.Fatalf(
			// The floor dropped from 7 to 2 when the functional plugins
			// moved to davison/topos-plugins (mock and mockstrict remain).
			"expected to discover at least 2 plugin modules under %s (found %d) — "+
				"a wrong pluginsDir would make this test vacuously pass",
			pluginsDir, discovered,
		)
	}

	if len(offenses) > 0 {
		t.Fatalf(
			"plugin icon contract violation(s) found (09-UI-SPEC.md Fix 10, 09-02-PLAN.md Task 3):\n%s",
			strings.Join(offenses, "\n"),
		)
	}
}

// TestPluginIconsScanner_FixtureReportsAllOffenseKinds is the negative
// control: it proves pluginIconOffenses is non-vacuous by running it
// against testdata/plugin_icons_violation, a fixture module deliberately
// missing the provenance block, using currentColor instead of a baked
// stroke, and setting Icon without IconMime — and failing unless all
// three offense kinds are reported.
func TestPluginIconsScanner_FixtureReportsAllOffenseKinds(t *testing.T) {
	fixture := filepath.Join("testdata", "plugin_icons_violation")
	offenses := pluginIconOffenses(fixture)

	wantSubstrings := []string{
		"missing provenance key",
		"currentColor",
		"sets Icon=true IconMime=false",
	}
	for _, want := range wantSubstrings {
		found := false
		for _, offense := range offenses {
			if strings.Contains(offense, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected an offense containing %q, got: %v", want, offenses)
		}
	}
}
