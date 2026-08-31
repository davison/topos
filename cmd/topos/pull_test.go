package main

// Committed proof of PULL-01/PULL-02 (M1-R8, davison/topos#19; GSD
// phase-18 criteria 1 and 2): the tier is earned, never chosen, and
// every failed verification aborts with both plugin directories
// byte-identical to their pre-attempt state — asserted with recursive
// digest manifests, not inspection.

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/pluginhost"
)

// pullTestEnv is one test's fixture: a release directory served over a
// real HTTP server, a throwaway accepted signing key, and a config
// whose two tier directories are fresh temp dirs.
type pullTestEnv struct {
	t          *testing.T
	releaseDir string
	server     *httptest.Server
	cfg        *config.Config
	keyID      string
	priv       ed25519.PrivateKey
}

func newPullTestEnv(t *testing.T) *pullTestEnv {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	restore := pluginhost.OverrideProvenanceKeys([]pluginhost.ProvenanceKey{{ID: "pull-test", PublicKey: pub}})
	t.Cleanup(restore)

	releaseDir := t.TempDir()
	server := httptest.NewServer(http.FileServer(http.Dir(releaseDir)))
	t.Cleanup(server.Close)

	return &pullTestEnv{
		t:          t,
		releaseDir: releaseDir,
		server:     server,
		cfg: &config.Config{Plugins: config.PluginsConfig{
			Dir:         filepath.Join(t.TempDir(), "trusted"),
			ExternalDir: filepath.Join(t.TempDir(), "external"),
		}},
		keyID: "pull-test",
		priv:  priv,
	}
}

func (e *pullTestEnv) url(name string) string { return e.server.URL + "/" + name }

func (e *pullTestEnv) writeBinary(name, content string) {
	e.t.Helper()
	if err := os.WriteFile(filepath.Join(e.releaseDir, name), []byte(content), 0o755); err != nil {
		e.t.Fatal(err)
	}
}

// sign writes a signed manifest pair (basename release-vX) vouching for
// the named files' CURRENT on-disk bytes, for the given platform.
func (e *pullTestEnv) sign(base, goos, goarch string, names ...string) {
	e.t.Helper()
	entries := make([]pluginhost.ProvenanceEntry, 0, len(names))
	for _, n := range names {
		h, err := pluginhost.HashBinary(filepath.Join(e.releaseDir, n))
		if err != nil {
			e.t.Fatal(err)
		}
		entries = append(entries, pluginhost.ProvenanceEntry{Name: n, SHA256: h, Version: "v9.9.9", Contract: "topos.v2"})
	}
	manifest, err := pluginhost.BuildProvenanceManifest(pluginhost.ProvenanceRelease{Repo: "example/pull-test", Tag: "v9.9.9", OS: goos, Arch: goarch}, entries)
	if err != nil {
		e.t.Fatal(err)
	}
	sig, err := pluginhost.SignProvenanceManifest(manifest, e.keyID, e.priv)
	if err != nil {
		e.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.releaseDir, base+pluginhost.ProvenanceManifestSuffix), manifest, 0o644); err != nil {
		e.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.releaseDir, base+pluginhost.ProvenanceSignatureSuffix), sig, 0o644); err != nil {
		e.t.Fatal(err)
	}
}

// writeChecksums writes checksums.txt over the named files' current
// bytes — the release's own asset manifest, sha256sum shape.
func (e *pullTestEnv) writeChecksums(names ...string) {
	e.t.Helper()
	var b strings.Builder
	for _, n := range names {
		raw, err := os.ReadFile(filepath.Join(e.releaseDir, n))
		if err != nil {
			e.t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		fmt.Fprintf(&b, "%s  %s\n", hex.EncodeToString(sum[:]), n)
	}
	if err := os.WriteFile(filepath.Join(e.releaseDir, "checksums.txt"), []byte(b.String()), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

// tierDigests is the byte-identical proof's instrument: a sorted
// manifest over BOTH tier directories recording every regular file's
// SHA-256 AND every directory's existence (PR #20 review finding 1: an
// absent directory and a freshly created empty one must never snapshot
// identically), with an absent root recorded as ABSENT and every other
// traversal error fatal — a proof that cannot read what it is proving
// is no proof.
func (e *pullTestEnv) tierDigests() string {
	e.t.Helper()
	var b strings.Builder
	for _, dir := range []string{e.cfg.Plugins.Dir, e.cfg.Plugins.ExternalDir} {
		if _, err := os.Lstat(dir); os.IsNotExist(err) {
			b.WriteString("ABSENT " + dir + "\n---\n")
			continue
		} else if err != nil {
			e.t.Fatal(err)
		}
		var lines []string
		walkErr := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				lines = append(lines, "DIR "+p)
				return nil
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(raw)
			lines = append(lines, p+" "+hex.EncodeToString(sum[:]))
			return nil
		})
		if walkErr != nil {
			e.t.Fatal(walkErr)
		}
		sort.Strings(lines)
		b.WriteString(strings.Join(lines, "\n") + "\n---\n")
	}
	return b.String()
}

// expectAbortUnplaced runs a pull that must fail, asserts the error
// names wantInMessage, and proves both tier directories byte-identical
// across the attempt.
func (e *pullTestEnv) expectAbortUnplaced(url, wantInMessage string) {
	e.t.Helper()
	before := e.tierDigests()
	err := pullPlugin(url, e.cfg, io_Discard())
	if err == nil {
		e.t.Fatalf("expected the pull to abort (%s), got success", wantInMessage)
	}
	if !strings.Contains(err.Error(), wantInMessage) {
		e.t.Fatalf("expected the abort to name %q, got: %v", wantInMessage, err)
	}
	if after := e.tierDigests(); after != before {
		e.t.Fatalf("a failed pull changed a tier directory (PULL-02):\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// io_Discard sidesteps importing io just for Discard in half the tests.
func io_Discard() *strings.Builder { return &strings.Builder{} }

func TestPull_ValidProvenanceEarnsTrusted(t *testing.T) {
	e := newPullTestEnv(t)
	e.writeBinary("topos-plugin-demo", "demo bytes v1")
	e.sign("example-v9.9.9", runtime.GOOS, runtime.GOARCH, "topos-plugin-demo")
	e.writeChecksums("topos-plugin-demo", "example-v9.9.9"+pluginhost.ProvenanceManifestSuffix, "example-v9.9.9"+pluginhost.ProvenanceSignatureSuffix)

	var out strings.Builder
	if err := pullPlugin(e.url("topos-plugin-demo"), e.cfg, &out); err != nil {
		t.Fatalf("pull: %v", err)
	}
	if !strings.Contains(out.String(), "TRUSTED") {
		t.Errorf("expected the report to name the trusted tier, got:\n%s", out.String())
	}
	bin := filepath.Join(e.cfg.Plugins.Dir, "topos-plugin-demo")
	if raw, err := os.ReadFile(bin); err != nil || string(raw) != "demo bytes v1" {
		t.Fatalf("expected the binary placed in the trusted directory, got err=%v", err)
	}
	if info, _ := os.Stat(bin); info.Mode().Perm() != 0o755 {
		t.Errorf("expected mode 0755 on the binary, got %v", info.Mode().Perm())
	}
	for _, f := range []string{"example-v9.9.9" + pluginhost.ProvenanceManifestSuffix, "example-v9.9.9" + pluginhost.ProvenanceSignatureSuffix} {
		p := filepath.Join(e.cfg.Plugins.Dir, f)
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("expected the vouching pair placed beside the binary: %v", err)
		}
		if info.Mode().Perm() != 0o644 {
			t.Errorf("expected mode 0644 on %s, got %v", f, info.Mode().Perm())
		}
	}
	if entries, _ := os.ReadDir(e.cfg.Plugins.ExternalDir); len(entries) != 0 {
		t.Errorf("expected the external directory untouched, got %v", entries)
	}
	// The launch gate reaches the same verdict from what was placed.
	if _, evidence, _, err := pluginhost.VerifySignedProvenance(pluginhost.Dirs{Trusted: e.cfg.Plugins.Dir}, "topos-plugin-demo", bin); err != nil || evidence == "" {
		t.Fatalf("expected the placed directory to verify exactly as the pull did, got evidence=%q err=%v", evidence, err)
	}
}

func TestPull_NoEvidenceEarnsExternal(t *testing.T) {
	e := newPullTestEnv(t)
	e.writeBinary("topos-plugin-demo", "unsigned bytes")
	// No checksums.txt at all: the clean no-evidence state.
	var out strings.Builder
	if err := pullPlugin(e.url("topos-plugin-demo"), e.cfg, &out); err != nil {
		t.Fatalf("pull: %v", err)
	}
	if !strings.Contains(out.String(), "EXTERNAL") || !strings.Contains(out.String(), "consent") {
		t.Errorf("expected the report to name the external tier and the consent flow, got:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(e.cfg.Plugins.ExternalDir, "topos-plugin-demo")); err != nil {
		t.Fatalf("expected the binary placed in the external directory: %v", err)
	}
	if _, err := os.Stat(e.cfg.Plugins.Dir); !os.IsNotExist(err) {
		t.Errorf("expected the trusted directory never created for an external pull, got err=%v", err)
	}
}

func TestPull_TamperedBinaryAbortsUnplaced(t *testing.T) {
	e := newPullTestEnv(t)
	e.writeBinary("topos-plugin-demo", "honest bytes")
	e.sign("example-v9.9.9", runtime.GOOS, runtime.GOARCH, "topos-plugin-demo")
	// Tamper AFTER signing, regenerating checksums to match the tampered
	// bytes — the attacker who controls the pipeline but not the key.
	e.writeBinary("topos-plugin-demo", "tampered bytes")
	e.writeChecksums("topos-plugin-demo", "example-v9.9.9"+pluginhost.ProvenanceManifestSuffix, "example-v9.9.9"+pluginhost.ProvenanceSignatureSuffix)
	e.expectAbortUnplaced(e.url("topos-plugin-demo"), "provenance verification refused")
}

func TestPull_UnknownKeyAbortsUnplaced(t *testing.T) {
	e := newPullTestEnv(t)
	// Sign with a key the kernel does NOT accept.
	_, otherPriv, _ := ed25519.GenerateKey(rand.Reader)
	e.priv = otherPriv // keyID stays "pull-test", but the signature won't verify against the accepted key
	e.writeBinary("topos-plugin-demo", "bytes")
	e.sign("example-v9.9.9", runtime.GOOS, runtime.GOARCH, "topos-plugin-demo")
	e.writeChecksums("topos-plugin-demo", "example-v9.9.9"+pluginhost.ProvenanceManifestSuffix, "example-v9.9.9"+pluginhost.ProvenanceSignatureSuffix)
	e.expectAbortUnplaced(e.url("topos-plugin-demo"), "none of it vouches")
}

func TestPull_WrongPlatformAbortsUnplaced(t *testing.T) {
	e := newPullTestEnv(t)
	e.writeBinary("topos-plugin-demo", "bytes")
	e.sign("example-v9.9.9", "plan9", "mips", "topos-plugin-demo")
	e.writeChecksums("topos-plugin-demo", "example-v9.9.9"+pluginhost.ProvenanceManifestSuffix, "example-v9.9.9"+pluginhost.ProvenanceSignatureSuffix)
	e.expectAbortUnplaced(e.url("topos-plugin-demo"), "none of it vouches")
}

func TestPull_ManifestOmittingBinaryAbortsUnplaced(t *testing.T) {
	e := newPullTestEnv(t)
	e.writeBinary("topos-plugin-demo", "bytes")
	e.writeBinary("topos-plugin-other", "other bytes")
	e.sign("example-v9.9.9", runtime.GOOS, runtime.GOARCH, "topos-plugin-other")
	e.writeChecksums("topos-plugin-demo", "topos-plugin-other", "example-v9.9.9"+pluginhost.ProvenanceManifestSuffix, "example-v9.9.9"+pluginhost.ProvenanceSignatureSuffix)
	e.expectAbortUnplaced(e.url("topos-plugin-demo"), "none of it vouches")
}

func TestPull_ChecksumsMismatchAbortsUnplaced(t *testing.T) {
	e := newPullTestEnv(t)
	e.writeBinary("topos-plugin-demo", "actual bytes")
	e.writeChecksums("topos-plugin-demo")
	e.writeBinary("topos-plugin-demo", "different bytes now") // served bytes drift after checksums were written
	e.expectAbortUnplaced(e.url("topos-plugin-demo"), "SHA-256 mismatch")
}

func TestPull_ChecksumsOmittingBinaryAbortsUnplaced(t *testing.T) {
	e := newPullTestEnv(t)
	e.writeBinary("topos-plugin-demo", "bytes")
	e.writeBinary("topos-plugin-other", "other")
	e.writeChecksums("topos-plugin-other")
	e.expectAbortUnplaced(e.url("topos-plugin-demo"), "does not name")
}

func TestPull_DownloadFailureAbortsUnplaced(t *testing.T) {
	e := newPullTestEnv(t)
	e.expectAbortUnplaced(e.url("topos-plugin-absent"), "download")
}

func TestPull_BadBasenameRefused(t *testing.T) {
	e := newPullTestEnv(t)
	e.expectAbortUnplaced(e.url("topos-plugin-UPPER"), "not a plugin binary name")
	e.expectAbortUnplaced(e.url("not-a-plugin"), "not a plugin binary name")
}

func TestPull_RepullReplacesInPlace(t *testing.T) {
	e := newPullTestEnv(t)
	e.writeBinary("topos-plugin-demo", "v1 bytes")
	e.sign("example-v1", runtime.GOOS, runtime.GOARCH, "topos-plugin-demo")
	e.writeChecksums("topos-plugin-demo", "example-v1"+pluginhost.ProvenanceManifestSuffix, "example-v1"+pluginhost.ProvenanceSignatureSuffix)
	if err := pullPlugin(e.url("topos-plugin-demo"), e.cfg, io_Discard()); err != nil {
		t.Fatalf("first pull: %v", err)
	}
	e.writeBinary("topos-plugin-demo", "v2 bytes")
	e.sign("example-v2", runtime.GOOS, runtime.GOARCH, "topos-plugin-demo")
	e.writeChecksums("topos-plugin-demo", "example-v2"+pluginhost.ProvenanceManifestSuffix, "example-v2"+pluginhost.ProvenanceSignatureSuffix)
	if err := pullPlugin(e.url("topos-plugin-demo"), e.cfg, io_Discard()); err != nil {
		t.Fatalf("second pull: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(e.cfg.Plugins.Dir, "topos-plugin-demo"))
	if err != nil || string(raw) != "v2 bytes" {
		t.Fatalf("expected the re-pull to replace the binary in place, got %q err=%v", raw, err)
	}
	// The newer pair sits beside the older one — coexistence the kernel's
	// scan already handles (D-08); the launch gate still verifies.
	if _, evidence, _, err := pluginhost.VerifySignedProvenance(pluginhost.Dirs{Trusted: e.cfg.Plugins.Dir}, "topos-plugin-demo", filepath.Join(e.cfg.Plugins.Dir, "topos-plugin-demo")); err != nil || evidence == "" {
		t.Fatalf("expected the updated directory to verify, got evidence=%q err=%v", evidence, err)
	}
}

func TestPull_ChecksumsWithoutProvenanceEarnsExternal(t *testing.T) {
	// The unsigned third-party release shape (amended Decision on #19):
	// a clean checksums.txt naming the binary with matching bytes but
	// naming no provenance pair publishes integrity without
	// authenticity — the external tier, never an abort.
	e := newPullTestEnv(t)
	e.writeBinary("topos-plugin-demo", "unsigned but checksummed")
	e.writeChecksums("topos-plugin-demo")
	var out strings.Builder
	if err := pullPlugin(e.url("topos-plugin-demo"), e.cfg, &out); err != nil {
		t.Fatalf("pull: %v", err)
	}
	if !strings.Contains(out.String(), "EXTERNAL") {
		t.Errorf("expected the external tier, got:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(e.cfg.Plugins.ExternalDir, "topos-plugin-demo")); err != nil {
		t.Fatalf("expected the binary placed in the external directory: %v", err)
	}
	if _, err := os.Stat(e.cfg.Plugins.Dir); !os.IsNotExist(err) {
		t.Errorf("expected the trusted directory never created, got err=%v", err)
	}
}

func TestPull_ForeignPlatformPairBesideMatchingPairStillTrusted(t *testing.T) {
	// A release publishing per-platform manifests: the pair for THIS
	// platform vouches, the foreign one contributes nothing — any valid
	// match wins, exactly the kernel's own scan rule (D-08).
	e := newPullTestEnv(t)
	e.writeBinary("topos-plugin-demo", "multi-platform release bytes")
	e.sign("example-linux", runtime.GOOS, runtime.GOARCH, "topos-plugin-demo")
	e.sign("example-darwin", "darwin", "arm64", "topos-plugin-demo")
	e.writeChecksums("topos-plugin-demo",
		"example-linux"+pluginhost.ProvenanceManifestSuffix, "example-linux"+pluginhost.ProvenanceSignatureSuffix,
		"example-darwin"+pluginhost.ProvenanceManifestSuffix, "example-darwin"+pluginhost.ProvenanceSignatureSuffix)
	var out strings.Builder
	if err := pullPlugin(e.url("topos-plugin-demo"), e.cfg, &out); err != nil {
		t.Fatalf("pull: %v", err)
	}
	if !strings.Contains(out.String(), "TRUSTED") || !strings.Contains(out.String(), "example-linux") {
		t.Errorf("expected the matching pair's evidence to earn trusted, got:\n%s", out.String())
	}
}

func TestPull_QueryBearingURLKeepsQueryForBinaryOnly(t *testing.T) {
	// The binary downloads with the operator's query intact; sibling
	// discovery addresses plain path siblings with no query.
	e := newPullTestEnv(t)
	e.writeBinary("topos-plugin-demo", "queried bytes")
	e.writeChecksums("topos-plugin-demo")
	var sawBinaryQuery, sawChecksumsQuery bool
	inner := e.server.Config.Handler
	e.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "topos-plugin-demo") && r.URL.RawQuery == "token=abc" {
			sawBinaryQuery = true
		}
		if strings.HasSuffix(r.URL.Path, "checksums.txt") && r.URL.RawQuery != "" {
			sawChecksumsQuery = true
		}
		inner.ServeHTTP(w, r)
	})
	if err := pullPlugin(e.url("topos-plugin-demo")+"?token=abc", e.cfg, io_Discard()); err != nil {
		t.Fatalf("pull: %v", err)
	}
	if !sawBinaryQuery {
		t.Error("expected the binary request to carry the operator's query")
	}
	if sawChecksumsQuery {
		t.Error("expected sibling discovery to drop the query")
	}
}

func TestPullPlace_FailureUnwindsCreatedDirectories(t *testing.T) {
	// A post-mkdir failure (here: a staged source that does not exist)
	// must remove every directory this attempt itself created — the
	// non-recursive unwind, from the destination up to the topmost
	// created ancestor — while a pre-existing ancestor survives.
	root := t.TempDir()
	dest := filepath.Join(root, "a", "b", "c")
	stage := t.TempDir()
	_, err := pullPlace(stage, dest, []pullPlacement{{name: "topos-plugin-ghost", mode: 0o755}})
	if err == nil {
		t.Fatal("expected the placement to fail for a missing staged source")
	}
	if _, statErr := os.Lstat(filepath.Join(root, "a")); !os.IsNotExist(statErr) {
		t.Fatalf("expected the created directory chain removed, got err=%v", statErr)
	}
	if _, statErr := os.Lstat(root); statErr != nil {
		t.Fatalf("expected the pre-existing root untouched: %v", statErr)
	}
}

func TestPullCheckRedirect_Policy(t *testing.T) {
	mk := func(rawurl string) *http.Request {
		u, err := url.Parse(rawurl)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Request{URL: u}
	}
	// https -> http downgrade: refused, naming the downgrade.
	err := pullCheckRedirect(mk("http://evil.example/x"), []*http.Request{mk("https://github.com/a")})
	if err == nil || !strings.Contains(err.Error(), "downgrade") {
		t.Fatalf("expected the https->http downgrade refused by name, got %v", err)
	}
	// https -> https cross-origin: allowed (the GitHub assets shape).
	if err := pullCheckRedirect(mk("https://objects.githubusercontent.com/x"), []*http.Request{mk("https://github.com/a")}); err != nil {
		t.Fatalf("expected the cross-origin https redirect allowed, got %v", err)
	}
	// http origin -> http: allowed (the operator chose http).
	if err := pullCheckRedirect(mk("http://mirror.example/x"), []*http.Request{mk("http://host.example/a")}); err != nil {
		t.Fatalf("expected an http-origin redirect allowed, got %v", err)
	}
	// The round-2 chain: http origin -> https hop -> http again. The
	// LAST hop was https, so the fallback is refused even though the
	// ORIGIN was http — the predicate reads the immediately preceding
	// request, never via[0].
	err = pullCheckRedirect(mk("http://cdn.example/x"), []*http.Request{mk("http://host.example/a"), mk("https://secure.example/b")})
	if err == nil || !strings.Contains(err.Error(), "downgrade") {
		t.Fatalf("expected the http->https->http chain refused at the downgrade hop, got %v", err)
	}
	// And the inverse chain stays legal: http -> http -> https.
	if err := pullCheckRedirect(mk("https://secure.example/x"), []*http.Request{mk("http://host.example/a"), mk("http://mirror.example/b")}); err != nil {
		t.Fatalf("expected an upgrade at any hop allowed, got %v", err)
	}
	// the hop cap.
	via := make([]*http.Request, 10)
	for i := range via {
		via[i] = mk("https://github.com/a")
	}
	if err := pullCheckRedirect(mk("https://github.com/b"), via); err == nil {
		t.Fatal("expected the ten-redirect cap enforced")
	}
}
