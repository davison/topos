package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// modulePin records the minimum acceptable version for a module path and
// the advisory identifier that motivates the floor.
type modulePin struct {
	MinVersion string
	Advisory   string
}

// minimumModuleVersions is the declared-dependency-floor table: for each
// module path listed here, any go.mod anywhere in the workspace declaring
// a version below MinVersion is a reported offence. Widening this table
// (adding an entry, or raising an existing floor) is a deliberate change:
// update it here, plus its comment, as part of that change — mirroring
// the sanctionedEgressFiles idiom outbound_hosts_test.go already
// established.
//
// This table exists at all because this repo is a go.work workspace of
// six modules (see go.work's `use` block), and Go's MVS resolves the
// *maximum* requirement across every listed module for any workspace
// build. That masking is exactly how plugins/proton's go.mod declared
// golang.org/x/net v0.26.0 — affected by CVE-2024-45338 / GO-2024-3333, a
// CPU/memory-exhaustion DoS from non-linear HTML-tokenization cost in
// golang.org/x/net/html, the tokenizer behind
// github.com/microcosm-cc/bluemonday and reachable from
// plugins/proton/body.go's RenderSanitizedEmail over an arbitrary inbound
// email's HTML body — while every workspace build actually compiled
// v0.56.0, unnoticed across three verification rounds (03-07-PLAN.md).
// The declared manifest is nonetheless the artefact a security scanner,
// an SBOM generator, or a third-party plugin author copying this repo's
// go.mod as a template all consume; this table is the permanent,
// mechanical guard against that declared contract silently regressing in
// any of the six modules again.
var minimumModuleVersions = map[string]modulePin{
	"golang.org/x/net": {MinVersion: "v0.33.0", Advisory: "CVE-2024-45338 / GO-2024-3333"},
}

// scanGoModForBelowFloorPins parses the go.mod file at path and returns
// one human-readable offence string per requirement whose module path is
// tracked in minimumModuleVersions and whose declared version is below
// (or unorderable against) that floor. Only tracked module paths are ever
// version-compared — the many pseudo-versions elsewhere in these
// manifests are never parsed at all.
//
// Both require shapes a real go.mod can use are handled: a parenthesised
// `require ( ... )` block (tab-indented entries) and a standalone
// single-line `require <path> <version> // indirect` (plugins/proton's
// own golang.org/x/text requirement uses exactly this shape — a scanner
// that only handled the block form would silently miss it).
func scanGoModForBelowFloorPins(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var offenses []string
	inBlock := false

	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if line == "require (" {
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}

		var reqLine string
		switch {
		case inBlock:
			reqLine = line
		case strings.HasPrefix(line, "require "):
			reqLine = strings.TrimPrefix(line, "require ")
		default:
			continue
		}

		// Strip a trailing "// indirect" (or any other trailing comment)
		// before splitting into module path and version fields.
		if idx := strings.Index(reqLine, "//"); idx >= 0 {
			reqLine = reqLine[:idx]
		}

		fields := strings.Fields(reqLine)
		if len(fields) < 2 {
			continue
		}
		modPath, declared := fields[0], fields[1]

		floor, tracked := minimumModuleVersions[modPath]
		if !tracked {
			continue
		}

		cmp, ok := compareGoVersions(declared, floor.MinVersion)
		if !ok {
			offenses = append(offenses, fmt.Sprintf(
				"%s: %s declares version %q, which cannot be ordered against the required floor %q (%s) — treat as an offence requiring explicit review, never as silently satisfying the floor",
				path, modPath, declared, floor.MinVersion, floor.Advisory,
			))
			continue
		}
		if cmp < 0 {
			offenses = append(offenses, fmt.Sprintf(
				"%s: %s declares %s, below the required floor %s (%s)",
				path, modPath, declared, floor.MinVersion, floor.Advisory,
			))
		}
	}

	return offenses, nil
}

// compareGoVersions compares two go.mod-style version strings of the
// vMAJOR.MINOR.PATCH form and reports -1/0/1 for a<b, a==b, a>b. ok is
// false for any input this cannot parse — a pseudo-version, a
// "+incompatible" suffix, a prerelease suffix, or any other non-numeric
// dotted-triple form — and the caller (scanGoModForBelowFloorPins) turns
// that into a reported offence rather than guessing that it satisfies the
// floor.
//
// This is a small local helper rather than a dependency on an external
// semver-comparison package: such a package is present in the workspace
// build list transitively, but is not a declared requirement of the root
// module, and adding one — in the very plan whose subject is dependency
// hygiene — would be the wrong trade for saving a few lines of version
// parsing.
func compareGoVersions(a, b string) (int, bool) {
	pa, ok := parseGoVersion(a)
	if !ok {
		return 0, false
	}
	pb, ok := parseGoVersion(b)
	if !ok {
		return 0, false
	}
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			if pa[i] < pb[i] {
				return -1, true
			}
			return 1, true
		}
	}
	return 0, true
}

// parseGoVersion parses a version string of the exact "vMAJOR.MINOR.PATCH"
// form into its three integer components. Anything else — a pseudo-version
// (v0.0.0-20240101000000-abcdef123456), a "+incompatible" suffix, a
// prerelease/build-metadata suffix, or a non-numeric component — is
// reported as unparseable via ok=false.
func parseGoVersion(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(v, "v")
	if strings.ContainsAny(v, "-+") {
		return out, false
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// TestPinScanner_FixtureReportsTheBelowFloorDeclaration is the negative
// control: it proves scanGoModForBelowFloorPins is non-vacuous by running
// it directly against testdata/vulnerable_pin_go.mod.txt, a fixture
// containing one below-floor golang.org/x/net declaration (in the
// standalone-require shape) plus one above-floor tracked-adjacent
// requirement and one requirement for a module path absent from the
// floor table — and failing unless the below-floor declaration is
// reported. Without this, TestNoModuleDeclaresAKnownVulnerablePin's
// silence over the real manifests could mean either "no offence" or
// "scanner matched nothing", and those are not the same thing.
func TestPinScanner_FixtureReportsTheBelowFloorDeclaration(t *testing.T) {
	fixture := filepath.Join("testdata", "vulnerable_pin_go.mod.txt")
	offenses, err := scanGoModForBelowFloorPins(fixture)
	if err != nil {
		t.Fatalf("scan %s: %v", fixture, err)
	}
	if len(offenses) < 1 {
		t.Fatalf("expected at least 1 offense from the negative-control fixture, got %d", len(offenses))
	}
	t.Logf("negative control reported: %v", offenses)
}

// TestNoModuleDeclaresAKnownVulnerablePin walks repoRoot (reusing the
// same repoRoot, skipDirs, and shouldSkipDir that
// TestNoForeignEgressOutsideSanctionedClient in outbound_hosts_test.go
// already established — same package, referenced directly, never copied
// or widened), parses every go.mod it finds outside the skipped
// directories, and fails if any declares a version below its documented
// floor in minimumModuleVersions.
//
// This is the standing, mechanical form of 03-VERIFICATION.md's one-time
// instruction to "check plugins/silverbullet for the same stale pin": that
// instruction was a manual check which expired the moment it was
// performed, whereas this covers all six go.work modules on every
// root-module `go test` run, permanently.
//
// The walk must also discover at least six go.mod files before asserting
// over the offences it found, so a wrong repoRoot or an over-broad
// skipDirs entry can never make this test vacuously green.
func TestNoModuleDeclaresAKnownVulnerablePin(t *testing.T) {
	var offenses []string
	scanned := 0

	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel != "." && shouldSkipDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(path) != "go.mod" {
			return nil
		}

		scanned++
		found, scanErr := scanGoModForBelowFloorPins(path)
		if scanErr != nil {
			return fmt.Errorf("scan %s: %w", path, scanErr)
		}
		offenses = append(offenses, found...)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", repoRoot, err)
	}

	if scanned < 6 {
		t.Fatalf(
			"expected to discover at least 6 go.mod files across the workspace (repoRoot=%q), got %d — "+
				"a wrong repoRoot or an over-broad skipDirs entry would make this test vacuously pass",
			repoRoot, scanned,
		)
	}

	if len(offenses) > 0 {
		t.Fatalf(
			"declared-dependency-floor violation(s) found — a workspace module declares a "+
				"version below its documented security floor:\n%s\n\n"+
				"To widen the floor table as a deliberate change, update minimumModuleVersions "+
				"in internal/audit/module_pins_test.go.",
			strings.Join(offenses, "\n"),
		)
	}
}
