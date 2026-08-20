// provenance_test.go pins the signed release-manifest trust arm
// (16-01-PLAN.md Task 1): every named provenance failure cause, and the
// tracer success path — a fixture plugin binary launching TierTrusted
// with an EMPTY link-time build manifest, using only signed-release
// evidence.
package pluginhost

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
)

// installProvenanceTestKey generates a fresh ed25519 keypair, installs
// its public half as the CURRENT accepted key set via
// OverrideProvenanceKeys (t.Cleanup-restored), and returns the key id
// used plus the private key for signing fixtures.
func installProvenanceTestKey(t *testing.T) (keyID string, priv ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	keyID = "test-key"
	restore := OverrideProvenanceKeys([]ProvenanceKey{{ID: keyID, PublicKey: pub}})
	t.Cleanup(restore)
	return keyID, priv
}

// writeSignedManifest builds and writes a validly-signed
// <dir>/<basename>.provenance.json + .provenance.sig pair via
// BuildProvenanceManifest/SignProvenanceManifest — the ONE producer path,
// never hand-formatted JSON — and returns the manifest's full path.
func writeSignedManifest(t *testing.T, dir, basename string, release ProvenanceRelease, entries []ProvenanceEntry, keyID string, priv ed25519.PrivateKey) string {
	t.Helper()

	manifestBytes, err := BuildProvenanceManifest(release, entries)
	if err != nil {
		t.Fatalf("BuildProvenanceManifest: %v", err)
	}
	sigBytes, err := SignProvenanceManifest(manifestBytes, keyID, priv)
	if err != nil {
		t.Fatalf("SignProvenanceManifest: %v", err)
	}

	manifestPath := filepath.Join(dir, basename+ProvenanceManifestSuffix)
	sigPath := filepath.Join(dir, basename+ProvenanceSignatureSuffix)
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(sigPath, sigBytes, 0o644); err != nil {
		t.Fatalf("write signature: %v", err)
	}
	return manifestPath
}

// nativeRelease returns a ProvenanceRelease already matching this test
// binary's own runtime.GOOS/runtime.GOARCH, so a fixture manifest built
// from it verifies platform-wise by default.
func nativeRelease() ProvenanceRelease {
	return ProvenanceRelease{
		Repo: "davison/topos-plugins",
		Tag:  "v0.1.0-test",
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}

// TestLaunch_Provenance_TracerSuccessPath is the tracer proof: a real
// fixture binary in a directory with a validly-signed manifest naming
// it, an EMPTY link-time manifest installed, launched through launch()
// -> tier is TierTrusted and the process actually runs.
func TestLaunch_Provenance_TracerSuccessPath(t *testing.T) {
	dir := buildMockPluginDir(t)
	restore := OverrideBuildManifest(map[string]string{}) // empty link-time manifest
	defer restore()

	keyID, priv := installProvenanceTestKey(t)
	hash := mustHashBinary(t, dir+"/topos-plugin-mock")
	writeSignedManifest(t, dir, "topos-plugins-v0.1.0", nativeRelease(),
		[]ProvenanceEntry{{Name: "topos-plugin-mock", SHA256: hash, Version: "0.1.0", Contract: "topos.v1"}},
		keyID, priv)

	dirs := Dirs{Trusted: dir}
	src := config.Source{Plugin: "topos-plugin-mock"}

	p, err := launch(context.Background(), dirs, "demo", src, nil, hclog.NewNullLogger(), false)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	defer p.Kill()

	if p.Tier() != TierTrusted {
		t.Errorf("expected tier %q, got %q", TierTrusted, p.Tier())
	}
}

// TestVerifySignedProvenance_UnknownKeyIDGrantsNoTrust proves a
// signature naming a key id absent from the accepted set contributes no
// evidence, names the unknown key id in a diagnostic, and is not an
// error (the binary simply falls to external tier).
func TestVerifySignedProvenance_UnknownKeyIDGrantsNoTrust(t *testing.T) {
	dir := buildMockPluginDir(t)

	// Sign with a key that is NEVER installed via OverrideProvenanceKeys —
	// the accepted set (test default: no override at all, so it is
	// whatever embeddedProvenanceKeys/provenanceKeysExtra resolve to,
	// which in this plan is always empty) will not recognise it.
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	hash := mustHashBinary(t, dir+"/topos-plugin-mock")
	writeSignedManifest(t, dir, "topos-plugins-v0.1.0", nativeRelease(),
		[]ProvenanceEntry{{Name: "topos-plugin-mock", SHA256: hash, Version: "0.1.0", Contract: "topos.v1"}},
		"unknown-key", priv)

	dirs := Dirs{Trusted: dir}
	gotHash, evidence, diagnostics, err := VerifySignedProvenance(dirs, "topos-plugin-mock", dir+"/topos-plugin-mock")
	if err != nil {
		t.Fatalf("expected nil error (no evidence, not a refusal), got: %v", err)
	}
	if evidence != "" {
		t.Fatalf("expected no evidence, got %q", evidence)
	}
	if gotHash != hash {
		t.Errorf("expected hash %q, got %q", hash, gotHash)
	}
	if !anyContains(diagnostics, "unknown-key") {
		t.Errorf("expected a diagnostic naming the unknown key id, got %v", diagnostics)
	}
}

// TestVerifySignedProvenance_CorruptedSignatureBytesGrantsNoTrust proves
// a signature file whose bytes are not a valid signature contributes no
// evidence and is not an error.
func TestVerifySignedProvenance_CorruptedSignatureBytesGrantsNoTrust(t *testing.T) {
	dir := buildMockPluginDir(t)
	keyID, priv := installProvenanceTestKey(t)
	hash := mustHashBinary(t, dir+"/topos-plugin-mock")
	manifestPath := writeSignedManifest(t, dir, "topos-plugins-v0.1.0", nativeRelease(),
		[]ProvenanceEntry{{Name: "topos-plugin-mock", SHA256: hash, Version: "0.1.0", Contract: "topos.v1"}},
		keyID, priv)

	sigPath := strings.TrimSuffix(manifestPath, ProvenanceManifestSuffix) + ProvenanceSignatureSuffix
	sigBytes, err := os.ReadFile(sigPath)
	if err != nil {
		t.Fatalf("read signature: %v", err)
	}
	// Flip one base64 character inside the signature value itself — still
	// valid base64, still decodes to ed25519.SignatureSize bytes, but no
	// longer verifies against the manifest — exercises the "signature
	// does not verify" path, not the "malformed JSON" path.
	mutated := mutateSignatureValue(t, string(sigBytes))
	if err := os.WriteFile(sigPath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("write corrupted signature: %v", err)
	}

	dirs := Dirs{Trusted: dir}
	_, evidence, diagnostics, err := VerifySignedProvenance(dirs, "topos-plugin-mock", dir+"/topos-plugin-mock")
	if err != nil {
		t.Fatalf("expected nil error (no evidence, not a refusal), got: %v", err)
	}
	if evidence != "" {
		t.Fatalf("expected no evidence, got %q", evidence)
	}
	if len(diagnostics) == 0 {
		t.Errorf("expected a diagnostic naming the signature failure, got none")
	}
}

// TestVerifySignedProvenance_SignatureOverDifferentBytesGrantsNoTrust
// proves a signature that validly signs SOME bytes, but not the sibling
// manifest file's actual bytes, contributes no evidence.
func TestVerifySignedProvenance_SignatureOverDifferentBytesGrantsNoTrust(t *testing.T) {
	dir := buildMockPluginDir(t)
	keyID, priv := installProvenanceTestKey(t)
	hash := mustHashBinary(t, dir+"/topos-plugin-mock")

	manifestBytes, err := BuildProvenanceManifest(nativeRelease(),
		[]ProvenanceEntry{{Name: "topos-plugin-mock", SHA256: hash, Version: "0.1.0", Contract: "topos.v1"}})
	if err != nil {
		t.Fatalf("BuildProvenanceManifest: %v", err)
	}
	// Sign DIFFERENT bytes than what gets written as the manifest file.
	sigBytes, err := SignProvenanceManifest(append([]byte{}, append(manifestBytes, ' ')...), keyID, priv)
	if err != nil {
		t.Fatalf("SignProvenanceManifest: %v", err)
	}

	manifestPath := filepath.Join(dir, "topos-plugins-v0.1.0"+ProvenanceManifestSuffix)
	sigPath := filepath.Join(dir, "topos-plugins-v0.1.0"+ProvenanceSignatureSuffix)
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(sigPath, sigBytes, 0o644); err != nil {
		t.Fatalf("write signature: %v", err)
	}

	dirs := Dirs{Trusted: dir}
	_, evidence, diagnostics, err := VerifySignedProvenance(dirs, "topos-plugin-mock", dir+"/topos-plugin-mock")
	if err != nil {
		t.Fatalf("expected nil error (no evidence, not a refusal), got: %v", err)
	}
	if evidence != "" {
		t.Fatalf("expected no evidence, got %q", evidence)
	}
	if len(diagnostics) == 0 {
		t.Errorf("expected a diagnostic naming the signature-verify failure, got none")
	}
}

// TestVerifySignedProvenance_PlatformMismatchGrantsNoTrust proves a
// validly-signed manifest whose release.os/arch do not match this
// kernel's runtime.GOOS/runtime.GOARCH grants no trust.
func TestVerifySignedProvenance_PlatformMismatchGrantsNoTrust(t *testing.T) {
	dir := buildMockPluginDir(t)
	keyID, priv := installProvenanceTestKey(t)
	hash := mustHashBinary(t, dir+"/topos-plugin-mock")

	release := nativeRelease()
	release.Arch = release.Arch + "-bogus" // guaranteed to differ from runtime.GOARCH
	writeSignedManifest(t, dir, "topos-plugins-v0.1.0", release,
		[]ProvenanceEntry{{Name: "topos-plugin-mock", SHA256: hash, Version: "0.1.0", Contract: "topos.v1"}},
		keyID, priv)

	dirs := Dirs{Trusted: dir}
	trust, err := EvaluateTrust(dirs, "topos-plugin-mock", dir+"/topos-plugin-mock")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if trust.Tier != TierExternal {
		t.Errorf("expected tier %q, got %q", TierExternal, trust.Tier)
	}
}

// TestVerifySignedProvenance_DigestMismatchRefusesAndCreatesNoSubprocess
// proves a validly-signed manifest naming a binary whose on-disk SHA-256
// differs from the signed digest refuses the launch by name and creates
// no subprocess (D-05/D-13).
func TestVerifySignedProvenance_DigestMismatchRefusesAndCreatesNoSubprocess(t *testing.T) {
	dir := copyMockBinaryToFreshDir(t)
	keyID, priv := installProvenanceTestKey(t)
	hash := mustHashBinary(t, dir+"/topos-plugin-mock")
	writeSignedManifest(t, dir, "topos-plugins-v0.1.0", nativeRelease(),
		[]ProvenanceEntry{{Name: "topos-plugin-mock", SHA256: hash, Version: "0.1.0", Contract: "topos.v1"}},
		keyID, priv)

	// Tamper the binary AFTER the manifest above was computed from the
	// original bytes — one byte appended.
	mutateLastByte(t, dir+"/topos-plugin-mock")

	dirs := Dirs{Trusted: dir}
	restore := OverrideBuildManifest(map[string]string{}) // empty link-time manifest
	defer restore()
	src := config.Source{Plugin: "topos-plugin-mock"}

	p, err := launch(context.Background(), dirs, "demo", src, nil, hclog.NewNullLogger(), false)
	if err == nil {
		if p != nil {
			p.Kill()
		}
		t.Fatal("expected an error for a tampered binary carrying a mismatched signed manifest")
	}
	if p != nil {
		t.Fatalf("expected a nil *Plugin (no subprocess should ever be created), got %+v", p)
	}
	if !errors.Is(err, ErrProvenanceUnverified) {
		t.Fatalf("expected errors.Is(err, ErrProvenanceUnverified), got: %v", err)
	}
}

// TestVerifySignedProvenance_BinaryNamedInNoManifestIsExternalTier proves
// a binary named in no manifest at all evaluates to TierExternal with a
// nil error — "no evidence" is not a refusal.
func TestVerifySignedProvenance_BinaryNamedInNoManifestIsExternalTier(t *testing.T) {
	dir := buildMockPluginDir(t)
	keyID, priv := installProvenanceTestKey(t)
	hash := mustHashBinary(t, dir+"/topos-plugin-mock")
	// A manifest present, validly signed, but naming a DIFFERENT binary.
	writeSignedManifest(t, dir, "topos-plugins-v0.1.0", nativeRelease(),
		[]ProvenanceEntry{{Name: "topos-plugin-unrelated", SHA256: hash, Version: "0.1.0", Contract: "topos.v1"}},
		keyID, priv)

	dirs := Dirs{Trusted: dir}
	trust, err := EvaluateTrust(dirs, "topos-plugin-mock", dir+"/topos-plugin-mock")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if trust.Tier != TierExternal {
		t.Errorf("expected tier %q, got %q", TierExternal, trust.Tier)
	}
	if trust.Evidence != "" {
		t.Errorf("expected no evidence, got %q", trust.Evidence)
	}
}

// TestVerifySignedProvenance_TwoManifestsOnlySecondNamesBinary proves
// D-08: a second, later manifest naming the binary succeeds even though
// the first manifest present does not mention it at all.
func TestVerifySignedProvenance_TwoManifestsOnlySecondNamesBinary(t *testing.T) {
	dir := buildMockPluginDir(t)
	keyID, priv := installProvenanceTestKey(t)
	hash := mustHashBinary(t, dir+"/topos-plugin-mock")

	writeSignedManifest(t, dir, "a-unrelated", nativeRelease(),
		[]ProvenanceEntry{{Name: "topos-plugin-unrelated", SHA256: hash, Version: "0.1.0", Contract: "topos.v1"}},
		keyID, priv)
	writeSignedManifest(t, dir, "b-covers-mock", nativeRelease(),
		[]ProvenanceEntry{{Name: "topos-plugin-mock", SHA256: hash, Version: "0.1.0", Contract: "topos.v1"}},
		keyID, priv)

	dirs := Dirs{Trusted: dir}
	trust, err := EvaluateTrust(dirs, "topos-plugin-mock", dir+"/topos-plugin-mock")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if trust.Tier != TierTrusted {
		t.Fatalf("expected tier %q, got %q", TierTrusted, trust.Tier)
	}
	if trust.Evidence != "b-covers-mock"+ProvenanceManifestSuffix {
		t.Errorf("expected evidence naming b-covers-mock's manifest, got %q", trust.Evidence)
	}
}

// TestVerifySignedProvenance_MultipleManifestsCoexistExhaustiveScanOrderIndependent
// proves D-08's exhaustive-scan requirement directly: an OLDER manifest
// naming the binary with a SUPERSEDED digest coexists with a NEWER
// manifest naming its CURRENT digest, and evaluation succeeds regardless
// of which filename sorts first in the directory listing.
func TestVerifySignedProvenance_MultipleManifestsCoexistExhaustiveScanOrderIndependent(t *testing.T) {
	cases := []struct {
		name        string
		oldBasename string
		newBasename string
	}{
		{name: "old sorts first", oldBasename: "a-old", newBasename: "b-new"},
		{name: "new sorts first", oldBasename: "b-old", newBasename: "a-new"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := buildMockPluginDir(t)
			keyID, priv := installProvenanceTestKey(t)
			currentHash := mustHashBinary(t, dir+"/topos-plugin-mock")
			supersededHash := strings.Repeat("0", 64) // never equals a real hash

			writeSignedManifest(t, dir, tc.oldBasename, nativeRelease(),
				[]ProvenanceEntry{{Name: "topos-plugin-mock", SHA256: supersededHash, Version: "0.0.9", Contract: "topos.v1"}},
				keyID, priv)
			writeSignedManifest(t, dir, tc.newBasename, nativeRelease(),
				[]ProvenanceEntry{{Name: "topos-plugin-mock", SHA256: currentHash, Version: "0.1.0", Contract: "topos.v1"}},
				keyID, priv)

			dirs := Dirs{Trusted: dir}
			trust, err := EvaluateTrust(dirs, "topos-plugin-mock", dir+"/topos-plugin-mock")
			if err != nil {
				t.Fatalf("expected nil error, got: %v", err)
			}
			if trust.Tier != TierTrusted {
				t.Fatalf("expected tier %q, got %q (diagnostics: %v)", TierTrusted, trust.Tier, trust.Diagnostics)
			}
		})
	}
}

// TestEvaluateTrust_ManifestNamingDifferentBinaryDoesNotAffectFirst is
// the suggested invariant test from 16-01-PLAN.md's
// assumption_delta_decision: adding a second validly-signed manifest
// naming a DIFFERENT binary does not change the first binary's
// evaluation — evidence sources compose, they do not compete.
func TestEvaluateTrust_ManifestNamingDifferentBinaryDoesNotAffectFirst(t *testing.T) {
	dir := buildMockPluginDir(t)
	keyID, priv := installProvenanceTestKey(t)
	hash := mustHashBinary(t, dir+"/topos-plugin-mock")

	dirs := Dirs{Trusted: dir}

	// Baseline: no manifest naming a different binary yet.
	before, err := EvaluateTrust(dirs, "topos-plugin-mock", dir+"/topos-plugin-mock")
	if err != nil {
		t.Fatalf("baseline EvaluateTrust: %v", err)
	}
	if before.Tier != TierExternal {
		t.Fatalf("expected baseline tier %q, got %q", TierExternal, before.Tier)
	}

	// Add a manifest naming an entirely different binary.
	writeSignedManifest(t, dir, "unrelated-release", nativeRelease(),
		[]ProvenanceEntry{{Name: "topos-plugin-unrelated", SHA256: strings.Repeat("a", 64), Version: "1.0.0", Contract: "topos.v1"}},
		keyID, priv)

	after, err := EvaluateTrust(dirs, "topos-plugin-mock", dir+"/topos-plugin-mock")
	if err != nil {
		t.Fatalf("after EvaluateTrust: %v", err)
	}
	if after.Tier != before.Tier {
		t.Errorf("expected tier unchanged (%q), got %q", before.Tier, after.Tier)
	}
	if after.Hash != hash {
		t.Errorf("expected hash %q, got %q", hash, after.Hash)
	}
}

// TestOverrideProvenanceKeys_InstallsAndRestoresExactly mirrors
// TestOverrideBuildManifest_InstallsAndRestoresExactly (manifest_test.go)
// for the new provenance key seam.
func TestOverrideProvenanceKeys_InstallsAndRestoresExactly(t *testing.T) {
	before := AcceptedProvenanceKeys()

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	restore := OverrideProvenanceKeys([]ProvenanceKey{{ID: "k1", PublicKey: pub}})

	got := AcceptedProvenanceKeys()
	if len(got) != 1 || got[0].ID != "k1" {
		t.Fatalf("expected override to install exactly one key %q, got %+v", "k1", got)
	}

	restore()
	after := AcceptedProvenanceKeys()
	if len(after) != len(before) {
		t.Fatalf("expected restore to return to the pre-override key set (%d keys), got %d", len(before), len(after))
	}
}

// TestParseProvenanceKeys_FormatProvenanceKeys_RoundTrips mirrors
// TestFormatManifest_ParseManifest_RoundTrips (manifest_test.go) for the
// new key-spec shape.
func TestParseProvenanceKeys_FormatProvenanceKeys_RoundTrips(t *testing.T) {
	pub1, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	pub2, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	keys := []ProvenanceKey{{ID: "b", PublicKey: pub2}, {ID: "a", PublicKey: pub1}}

	spec := FormatProvenanceKeys(keys)
	parsed, err := ParseProvenanceKeys(spec)
	if err != nil {
		t.Fatalf("ParseProvenanceKeys(%q): %v", spec, err)
	}
	if len(parsed) != 2 || parsed[0].ID != "a" || parsed[1].ID != "b" {
		t.Fatalf("expected sorted round-trip [a b], got %+v", parsed)
	}
}

// TestParseProvenanceKeys_MalformedSegmentsAreRejected mirrors
// TestParseManifest_MalformedSegmentsAreRejected's table shape.
func TestParseProvenanceKeys_MalformedSegmentsAreRejected(t *testing.T) {
	cases := []string{
		"noequalssign",
		"=abc",
		"id=not-valid-base64!!!",
		"id=" + base64.StdEncoding.EncodeToString([]byte("too-short")),
	}
	for _, spec := range cases {
		if _, err := ParseProvenanceKeys(spec); err == nil {
			t.Errorf("ParseProvenanceKeys(%q): expected an error, got nil", spec)
		}
	}
}

// anyContains reports whether any string in ss contains substr.
func anyContains(ss []string, substr string) bool {
	for _, s := range ss {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// TestEmbeddedProvenanceKeys_WellFormed pins embeddedProvenanceKeys
// (16-04-PLAN.md Task 2, T-16-24): the compiled-in accepted key set this
// kernel build ships MUST be non-empty (the whole point of this release
// is that it names a real key), every entry's public key MUST decode to
// exactly ed25519.PublicKeySize bytes, and every key id MUST be unique —
// a malformed embedded key is a defect this test catches at build/test
// time, never something a user discovers only when their launch silently
// fails to trust a genuinely valid release. This test reads
// embeddedProvenanceKeys directly (not AcceptedProvenanceKeys) so it is
// unaffected by provenanceKeysExtra or any test-only OverrideProvenanceKeys
// installed elsewhere in this package's test run.
func TestEmbeddedProvenanceKeys_WellFormed(t *testing.T) {
	if len(embeddedProvenanceKeys) == 0 {
		t.Fatal("embeddedProvenanceKeys is empty — this kernel build embeds no accepted signing key at all")
	}

	seenIDs := make(map[string]bool, len(embeddedProvenanceKeys))
	for _, k := range embeddedProvenanceKeys {
		if k.ID == "" {
			t.Errorf("embeddedProvenanceKeys contains an entry with an empty key id")
		}
		if seenIDs[k.ID] {
			t.Errorf("embeddedProvenanceKeys contains a duplicate key id %q", k.ID)
		}
		seenIDs[k.ID] = true

		if len(k.PublicKey) != ed25519.PublicKeySize {
			t.Errorf("embeddedProvenanceKeys entry %q: public key is %d bytes, want %d (ed25519.PublicKeySize)", k.ID, len(k.PublicKey), ed25519.PublicKeySize)
		}
	}
}

// TestEmbeddedProvenanceKeys_NamesToposPlugins2026a pins the specific key
// this plan embeds (16-04-PLAN.md Task 2, D-04): key id
// "topos-plugins-2026a" must be present in the compiled-in accepted key
// set, so a regression that accidentally clears or renames the entry
// fails the build's own test run by name.
func TestEmbeddedProvenanceKeys_NamesToposPlugins2026a(t *testing.T) {
	for _, k := range embeddedProvenanceKeys {
		if k.ID == "topos-plugins-2026a" {
			return
		}
	}
	t.Fatalf("embeddedProvenanceKeys does not contain key id %q", "topos-plugins-2026a")
}

// mutateSignatureValue flips one character inside sigJSON's "signature"
// base64 value, leaving the surrounding JSON structure and key_id/schema
// fields intact — used to exercise "signature does not verify" without
// producing a JSON parse error.
func mutateSignatureValue(t *testing.T, sigJSON string) string {
	t.Helper()
	const marker = `"signature": "`
	idx := strings.Index(sigJSON, marker)
	if idx < 0 {
		t.Fatalf("mutateSignatureValue: could not find %q in signature JSON", marker)
	}
	valueStart := idx + len(marker)
	// Flip the first base64 character of the signature value.
	b := []byte(sigJSON)
	if b[valueStart] == 'A' {
		b[valueStart] = 'B'
	} else {
		b[valueStart] = 'A'
	}
	return string(b)
}
