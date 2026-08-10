package audit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// repoRoot is the repository root, relative to this package's own
// directory (internal/audit). Go always runs a test binary with its own
// package directory as the working directory, so this relative path is
// stable regardless of how "go test" is invoked. A filesystem walk (not
// package loading) is used deliberately: it crosses all three go.work
// modules (root, sdk, plugins/paperless) in a single pass, which
// `go test ./...` from any one module cannot.
const repoRoot = "../.."

// sanctionedEgressFiles is the set of files in the repo allowed to
// construct outbound HTTP requests — one per source plugin's REST/HTTP
// client. Widening this set is a deliberate change: update it here, plus
// this comment, as part of that change. Widened in 02-01-PLAN.md Task 1 to
// add the SilverBullet plugin's client alongside paperless's (previously a
// single sanctionedEgressFile constant, when only one plugin existed).
var sanctionedEgressFiles = map[string]bool{
	"plugins/paperless/client.go":    true,
	"plugins/silverbullet/client.go": true,
}

// sanctionedDeepLinkLiteralFiles is a narrower, DIFFERENT-IN-KIND
// allowlist from sanctionedEgressFiles above: files permitted to contain
// a foreign https(s)/ws(s) URL LITERAL because it is deep-link METADATA
// this Go process itself never dials — the literal is handed back
// through Item.DeepLink to the frontend, which the USER (not this
// process) may click to open in their own browser/app. This is a
// materially different privacy shape than sanctionedEgressFiles' REST/
// HTTP client files (which the plugin process itself uses to make
// outbound calls on ITS OWN initiative, no user click required):
// widening THIS list does NOT grant a file permission to construct an
// http.Client or issue an outbound request — the outbound-HTTP-
// construction check in scanFileForForeignEgress below still applies to
// these files unconditionally (that check is gated on sanctionedEgressFiles
// alone, checked separately from this list).
//
// plugins/whatsapp/deeplink.go (08-01-PLAN.md Task 3's real-device spike,
// 2026-08-10): WhatsApp's only documented, reliable click-to-chat web API
// is "https://wa.me/<phone>" (1:1), with "https://web.whatsapp.com/" as
// the honest best-effort group fallback (WhatsApp has no per-group web
// URL) — a non-functional bare "whatsapp://" scheme (this file's PRIOR
// literal) does nothing at all on a desktop with no WhatsApp Linux
// client installed, confirmed live against the real spike machine. This
// project's own privacy constraint ("no personal content leaves the
// user's machines") is about data this plugin's OWN background process
// transmits without the user's direct action — an Item's own deep_link
// field is inert until the user themselves clicks "Open in WhatsApp",
// the identical shape plugins/signal/deeplink.go's own
// "sgnl://signal.me/#p/<e164>" scheme URI already uses (that one simply
// isn't an http(s)/ws(s) scheme, so it never tripped this scanner in the
// first place).
var sanctionedDeepLinkLiteralFiles = map[string]bool{
	"plugins/whatsapp/deeplink.go": true,
}

// skipDirs are directories (relative to repoRoot, slash-separated) whose
// entire subtree is skipped: vendored/generated/build output that is
// never shipped as source, plus .git. "testdata" is skipped anywhere it
// occurs (matched by base name, not full path) since Go itself never
// compiles it and it deliberately holds this package's own negative
// control.
var skipDirs = map[string]bool{
	".git":               true,
	"node_modules":       true,
	"web":                true,
	"bin":                true,
	"kernel/webui/build": true,
}

// outboundHTTPIdents are net/http package-level identifiers that
// construct or issue an outbound request or reference the shared default
// client/transport. Referencing any of these outside a file in
// sanctionedEgressFiles would open a second, unaudited egress point.
var outboundHTTPIdents = map[string]bool{
	"Get":                   true,
	"Head":                  true,
	"Post":                  true,
	"PostForm":              true,
	"NewRequest":            true,
	"NewRequestWithContext": true,
	"DefaultClient":         true,
	"DefaultTransport":      true,
}

// outboundHTTPTypes are net/http composite-literal types that build a
// client capable of issuing outbound requests.
var outboundHTTPTypes = map[string]bool{
	"Client":    true,
	"Transport": true,
}

// Deliberately NOT flagged (inbound-server usage, out of this scan's
// scope): http.ListenAndServe (cmd/topos/main.go), and
// kernel/httpapi's use of http.Handler, http.HandlerFunc, http.Request,
// http.ResponseWriter, http.FileServer, http.FS, http.Error, and the
// http.StatusXxx / header constants — none of those names appear in
// outboundHTTPIdents or outboundHTTPTypes above.

var schemeAuthority = regexp.MustCompile(`^(?:https?|wss?)://`)

// TestNoForeignEgressOutsideSanctionedClient walks the Go AST (not text: a
// comment or a string built by concatenation cannot trip or defeat this
// check) of every non-test .go file in the repository and fails if it
// finds a foreign absolute URL literal, or outbound HTTP construction
// outside a file in sanctionedEgressFiles. This is the mechanical enforcement
// of the prohibition that plugin outbound traffic — and the kernel, which
// must have no egress at all — MUST NOT reach any host other than the
// user's own configured paperless-ngx instance and the loopback interface
// (PROJECT.md Constraints; 01-01-PLAN.md's third prohibition, closed by
// gap G-01-6).
//
// *_test.go files are excluded: test binaries are never shipped, and the
// existing kernel/config, kernel/correlate, kernel/httpapi and
// kernel/index suites legitimately use example hosts as fixtures. The
// guarantee being enforced here is about the binaries the user runs.
func TestNoForeignEgressOutsideSanctionedClient(t *testing.T) {
	var offenses []string

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
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		offenses = append(offenses, scanFileForForeignEgress(t, path, sanctionedEgressFiles[rel], sanctionedDeepLinkLiteralFiles[rel])...)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", repoRoot, err)
	}

	if len(offenses) > 0 {
		t.Fatalf(
			"PROJECT.md Constraints / 01-01-PLAN.md prohibition (\"MUST NOT transmit indexed "+
				"item data, config file contents, or usage telemetry to any host other than the "+
				"user's own configured paperless-ngx instance and the loopback interface\"): "+
				"found outbound-egress violation(s):\n%s\n\n"+
				"To widen the sanctioned-egress list as part of a deliberate change, update "+
				"sanctionedEgressFiles in internal/audit/outbound_hosts_test.go.",
			strings.Join(offenses, "\n"),
		)
	}
}

// TestScanner_FixtureReportsBothOffenseKinds is the negative control: it
// proves the scanner is non-vacuous by running it against
// testdata/foreign_host_violation.go.txt, a fixture containing one
// instance of each offense kind, and failing unless both are reported.
func TestScanner_FixtureReportsBothOffenseKinds(t *testing.T) {
	fixture := filepath.Join("testdata", "foreign_host_violation.go.txt")
	offenses := scanFileForForeignEgress(t, fixture, false, false)
	if len(offenses) < 2 {
		t.Fatalf("expected at least 2 offenses from the negative-control fixture, got %d: %v", len(offenses), offenses)
	}
}

// shouldSkipDir reports whether the directory at rel (relative to
// repoRoot, slash-separated, never ".") should be skipped in its
// entirety.
func shouldSkipDir(rel string) bool {
	if skipDirs[rel] {
		return true
	}
	return filepath.Base(rel) == "testdata"
}

// scanFileForForeignEgress parses path and walks its AST, returning one
// human-readable offense string per finding. sanctioned is true only for
// sanctionedEgressFiles, in which case outbound-HTTP-construction offenses
// are not flagged (foreign URL literals still are). deepLinkSanctioned is
// true only for sanctionedDeepLinkLiteralFiles, in which case foreign URL
// LITERALS are not flagged — but outbound-HTTP-construction offenses
// remain flagged regardless (gated on sanctioned alone), since a
// deep-link file being allowed to contain a URL string is not the same
// permission as being allowed to dial one.
func scanFileForForeignEgress(t *testing.T, path string, sanctioned, deepLinkSanctioned bool) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	var offenses []string
	ast.Inspect(file, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.BasicLit:
			if deepLinkSanctioned {
				return true
			}
			if expr.Kind != token.STRING {
				return true
			}
			val, unquoteErr := strconv.Unquote(expr.Value)
			if unquoteErr != nil {
				return true
			}
			if host, foreign := foreignURLHost(val); foreign {
				offenses = append(offenses, fmt.Sprintf(
					"%s: foreign absolute URL literal %q (host %q)", fset.Position(expr.Pos()), val, host,
				))
			}
		case *ast.SelectorExpr:
			if sanctioned {
				return true
			}
			if pkgIdent, ok := expr.X.(*ast.Ident); ok && pkgIdent.Name == "http" && outboundHTTPIdents[expr.Sel.Name] {
				offenses = append(offenses, fmt.Sprintf(
					"%s: outbound HTTP construction http.%s referenced outside a sanctioned client file",
					fset.Position(expr.Pos()), expr.Sel.Name,
				))
			}
		case *ast.CompositeLit:
			if sanctioned {
				return true
			}
			sel, ok := expr.Type.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if pkgIdent, ok := sel.X.(*ast.Ident); ok && pkgIdent.Name == "http" && outboundHTTPTypes[sel.Sel.Name] {
				offenses = append(offenses, fmt.Sprintf(
					"%s: outbound HTTP construction http.%s{} referenced outside a sanctioned client file",
					fset.Position(expr.Pos()), sel.Sel.Name,
				))
			}
		}
		return true
	})
	return offenses
}

// foreignURLHost reports whether val is a string with an http(s)/ws(s)
// scheme-and-authority prefix whose host is neither loopback nor
// "localhost". Returns the lowercased host either way (empty when val has
// no such prefix).
func foreignURLHost(val string) (host string, foreign bool) {
	if !schemeAuthority.MatchString(val) {
		return "", false
	}
	u, err := url.Parse(val)
	if err != nil {
		return "", false
	}
	h := strings.ToLower(u.Hostname())
	if h == "" {
		return "", false
	}
	if h == "localhost" {
		return h, false
	}
	if ip := net.ParseIP(h); ip != nil && ip.IsLoopback() {
		return h, false
	}
	return h, true
}
