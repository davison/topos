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
// path->sha256 manifest over BOTH tier directories (absent directories
// digest as empty).
func (e *pullTestEnv) tierDigests() string {
	e.t.Helper()
	var b strings.Builder
	for _, dir := range []string{e.cfg.Plugins.Dir, e.cfg.Plugins.ExternalDir} {
		var lines []string
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				e.t.Fatal(err)
			}
			sum := sha256.Sum256(raw)
			lines = append(lines, p+" "+hex.EncodeToString(sum[:]))
			return nil
		})
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
