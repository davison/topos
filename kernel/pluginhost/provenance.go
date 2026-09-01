// provenance.go implements the signed release-manifest trust arm
// (16-01-PLAN.md Task 1, D-01/D-03/D-05/D-06/D-07/D-08/D-10/D-12/D-13):
// an ed25519-signed release manifest that vouches for a plugin binary's
// name and SHA-256 without needing an entry in this kernel build's
// link-time build manifest (manifest.go). EvaluateTrust, below, is the
// single authority launch (host.go) consults — it asks BOTH evidence
// sources (the link-time arm and this file's signed arm) and neither can
// silently substitute for the other.
//
// Stale-manifest cleanup (Claude's discretion, decided here, D-08):
// NONE. This package is a READER of the plugins directory, never a
// writer — a superseded release manifest (naming an older binary this
// operator may still have) is tolerated forever, costs one extra
// signature verification per evaluation, and remains useful evidence for
// whatever it names. A component that deletes or rewrites operator files
// to keep its own state tidy is a component that can destroy evidence; a
// future reader should not add a cleanup pass here believing it a
// tidiness improvement.
package pluginhost

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/davison/topos/kernel/config"
)

// ProvenanceManifestSuffix and ProvenanceSignatureSuffix name the two
// sibling files a signed release places beside its plugin binaries
// (D-05/D-07): a manifest at <base>ProvenanceManifestSuffix is signed by
// <base>ProvenanceSignatureSuffix. Versioned basenames (e.g.
// "topos-plugins-v0.1.0.provenance.json") let multiple release manifests
// coexist per D-08 — VerifySignedProvenance scans every candidate
// present, never just one.
const (
	ProvenanceManifestSuffix  = ".provenance.json"
	ProvenanceSignatureSuffix = ".provenance.sig"
)

// ProvenanceSchema and ProvenanceSignatureSchema are the schema markers
// BuildProvenanceManifest/SignProvenanceManifest stamp into each file's
// top-level "schema" field, and VerifySignedProvenance requires — a
// version marker for the ON-DISK FORMAT itself, distinct from each
// plugin entry's own "contract" field (the gRPC contract generation the
// binary was built against, D-06).
const (
	ProvenanceSchema          = "topos.provenance.v1"
	ProvenanceSignatureSchema = "topos.provenance.sig.v1"
)

// MaxProvenanceManifestBytes caps a candidate manifest or signature
// file's size before it is ever read into memory (T-16-06): a hostile
// oversized file sitting in a plugin directory must not be able to
// exhaust memory. Exceeding this cap makes the candidate INVALID (a
// named diagnostic), never a scan-aborting error — one hostile file must
// not veto a valid manifest sitting beside it (D-08's own discipline,
// extended).
const MaxProvenanceManifestBytes = 1 << 20 // 1 MiB

// ErrProvenanceUnverified is returned (always wrapped, never bare) by
// VerifySignedProvenance/EvaluateTrust when no validly-signed release
// manifest vouches for a binary. This sentinel collapses every one of
// the following named causes — no provenance manifest file present in
// scope at all, a manifest whose sibling signature file is missing or
// malformed, a signature naming an unknown key id, a signature that does
// not verify against the manifest's raw bytes, a manifest whose
// release.os/release.arch does not match runtime.GOOS/runtime.GOARCH,
// and a manifest that DOES verify but names this binary with a digest
// that differs from what is on disk (tamper) — because the caller's
// remedy is identical in every case (D-13): this binary's only path to
// running remains the existing external-tier consent and pin flow.
// Verification NEVER demotes a trusted-tier binary and runs it anyway at
// a lower tier, silently or otherwise.
var ErrProvenanceUnverified = errors.New("pluginhost: plugin binary is not verified by any signed release manifest")

// ProvenanceKey pairs an accepted verification key id with its ed25519
// public key (D-03). Signatures name the key that made them by ID, so
// key rotation is purely additive: a new kernel release ships a new key
// beside the old one, and releases signed with either verify during the
// overlap.
type ProvenanceKey struct {
	ID        string
	PublicKey ed25519.PublicKey
	// Word records WHOSE word this key speaks (M2-R4, davison/topos#49):
	// the kernel author's (KeyWordEmbedded — compiled in; KeyWordBuild —
	// the link-time provenanceKeysExtra seam) or the operator's
	// (KeyWordOperator — [[plugins.trusted_keys]] in the operator's own
	// config, installed by SetOperatorProvenanceKeys). EvaluateTrust
	// turns the word into the tier: an operator key earns
	// TierOperatorTrusted, never TierTrusted, so the chip and the API
	// say on whose word a plugin runs.
	Word KeyWord
}

// KeyWord names whose trust decision a ProvenanceKey carries.
type KeyWord string

const (
	KeyWordEmbedded KeyWord = "embedded"
	KeyWordBuild    KeyWord = "build"
	KeyWordOperator KeyWord = "operator"
)

// KeyOffer is what an unknown-but-self-describing signature yields
// (davison/topos#49, "an unknown key is an offer"): the signature file
// carried a public key, the signature VERIFIED against it, and the
// manifest it signs names the binary with a matching digest — so the
// operator can be offered "trust this key for future releases" for a
// key that demonstrably signed this release. It is NOT evidence: the
// binary is external until the operator trusts the key. Reused is set
// when KeyID matches a key already in the accepted set but the carried
// public key differs — a reused id is exactly the impersonation the
// fingerprint guards against, and the offer must say so.
type KeyOffer struct {
	KeyID       string
	PublicKey   ed25519.PublicKey
	Fingerprint string
	Reused      bool
}

// KeyFingerprint is the SHA-256 of the raw ed25519 public key bytes,
// lowercase hex — the value an operator compares against a developer's
// published fingerprint before trusting a key. Callers abbreviate for
// display; this is the full digest.
func KeyFingerprint(pub ed25519.PublicKey) string {
	sum := sha256.Sum256([]byte(pub))
	return hex.EncodeToString(sum[:])
}

// ProvenanceRelease is a signed manifest's "release" object (D-06): the
// repo and tag this release was cut from, plus the OS/arch its plugin
// binaries were built for. VerifySignedProvenance compares OS/Arch
// against runtime.GOOS/runtime.GOARCH — a manifest built for a different
// platform grants no trust here even when correctly signed.
type ProvenanceRelease struct {
	Repo string `json:"repo"`
	Tag  string `json:"tag"`
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// ProvenanceEntry is one plugin binary's name -> sha256 binding (D-05),
// plus the version and gRPC contract generation ("topos.v1" today) it
// was built against (D-06) — Contract is the field Phase 17's DIST-03
// mismatch check consumes; this plan records it but does not itself
// enforce a match.
type ProvenanceEntry struct {
	Name     string `json:"name"`
	SHA256   string `json:"sha256"`
	Version  string `json:"version"`
	Contract string `json:"contract"`
}

// ProvenanceManifest is the on-disk shape of a
// <base>ProvenanceManifestSuffix file (D-05/D-06/D-07): a schema marker,
// the release descriptor, and every plugin binary this release vouches
// for.
type ProvenanceManifest struct {
	Schema  string            `json:"schema"`
	Release ProvenanceRelease `json:"release"`
	Plugins []ProvenanceEntry `json:"plugins"`
}

// ProvenanceSignature is the on-disk shape of a
// <base>ProvenanceSignatureSuffix file: which key signed, the algorithm
// (only "ed25519" is ever accepted), and the standard-base64-encoded
// 64-byte ed25519 signature over the sibling manifest file's raw bytes
// VERBATIM — never a re-serialized or canonicalized form, so
// verification never depends on JSON field ordering or whitespace.
type ProvenanceSignature struct {
	Schema    string `json:"schema"`
	KeyID     string `json:"key_id"`
	Algorithm string `json:"algorithm"`
	Signature string `json:"signature"`
	// PublicKey is the signer's ed25519 public key, standard base64
	// (davison/topos#49, "the key travels with the signature"). Optional
	// on the wire — the schema stays topos.provenance.sig.v1, an added
	// field an older kernel ignores — and written by every signature
	// SignProvenanceManifest produces. It never makes a key ACCEPTED:
	// when the key id is unknown, the kernel verifies the signature
	// against this key to decide whether it can OFFER the key to the
	// operator; acceptance is still the accepted set's.
	PublicKey string `json:"public_key,omitempty"`
}

// Trust is EvaluateTrust's result (D-10): the tier a binary evaluated
// to, its on-disk SHA-256, which evidence source vouched for it
// ("build-manifest", or a signed manifest's filename — empty when Tier
// is TierExternal or when evaluation refused), and every named
// diagnostic collected while scanning candidate manifests (T-16-07) — a
// caller holding a logger (launch, in host.go) emits these one by one;
// EvaluateTrust itself performs no logging and caches nothing.
//
// On a tamper refusal (16-06-PLAN.md Task 1, 16-VERIFICATION.md gap 1),
// Tier is TierTrusted — NEVER the empty zero value — because a refusal
// only ever occurs when a trust arm positively NAMED this binary with a
// digest that no longer matches. Tier is a REPORTING field consumed
// after the gate decision has already been made: it is the returned
// error, not Tier, that actually refuses the launch.
type Trust struct {
	Tier        Tier
	Hash        string
	Evidence    string
	Diagnostics []string
	// KeyID and Word name the accepted key that signed the vouching
	// manifest (empty for the build-manifest arm and for TierExternal):
	// Word KeyWordOperator is what makes Tier TierOperatorTrusted.
	KeyID string
	Word  KeyWord
	// Offer is set on a TierExternal result when a manifest naming this
	// binary with a matching digest was signed by an unknown key that
	// carried its public key and verified against it — the operator may
	// be offered that key. Never set alongside Evidence.
	Offer *KeyOffer
}

// embeddedProvenanceKeys is the compiled-in accepted verification key SET
// (D-03). This is the ONE real key this kernel accepts as of this release
// (16-04-PLAN.md, D-04):
//
//   - key id "topos-plugins-2026a"
//   - private half held ONLY as the TOPOS_PROVENANCE_SIGNING_KEY GitHub
//     Actions secret in https://github.com/davison/topos-plugins — never
//     committed to any repository, never printed by any tool, never
//     logged by any workflow step
//   - signs every tagged release (v*.*.*) that repository's own
//     .github/workflows/release.yml publishes
//
// Rotation (D-03): adding a second key here — for a new signing identity,
// or because this one is believed compromised — is purely additive. Ship
// a new kernel release with both entries in this slice; releases signed
// with EITHER key verify during the overlap; once every operator has
// upgraded past the last release trusting the retired key, its entry can
// be deleted in a later release. Like buildManifest (manifest.go), this
// slice is NEVER read from a file, environment variable, or configuration
// at run time — the only way to change it is a new compiled kernel.
var embeddedProvenanceKeys = []ProvenanceKey{
	{
		ID:        "topos-plugins-2026a",
		PublicKey: mustDecodeProvenancePublicKey("hHDvrubE9EkNOl81FY2obRUaeeeBgmZcdWNXy93xntQ="),
	},
}

// mustDecodeProvenancePublicKey decodes a standard-base64-encoded ed25519
// public key literal for embeddedProvenanceKeys above. A malformed base64
// literal is a build-time defect this function panics on immediately —
// far louder than letting a garbage []byte silently reach
// AcceptedProvenanceKeys. A WRONG-SIZE (but validly base64-encoded) key
// is deliberately NOT panicked on here: TestEmbeddedProvenanceKeys_WellFormed
// asserts the size instead, so a size defect fails the build's own test
// run with a clear test name and message (T-16-24) rather than a bare
// panic with no test attribution.
func mustDecodeProvenancePublicKey(b64 string) ed25519.PublicKey {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic(fmt.Sprintf("pluginhost: embeddedProvenanceKeys: malformed base64 public key literal: %v", err))
	}
	return ed25519.PublicKey(raw)
}

// provenanceKeysExtra is the link-time-only EXTENSION to
// embeddedProvenanceKeys (D-12), populated ONLY via:
//
//	go build -ldflags "-X github.com/davison/topos/kernel/pluginhost.provenanceKeysExtra=<spec>"
//
// <spec> is FormatProvenanceKeys's comma-separated "keyid=<base64
// pubkey>" output. This variable is NEVER read from a file on disk, an
// environment variable, or configuration at run time — it is link-time
// data only, mirroring buildManifest's own discipline exactly. This is
// the D-12 seam a dev/e2e build uses to trust its own locally signed
// fixtures (see scripts/provenance-smoke.sh); production release builds
// leave it unset and rely on embeddedProvenanceKeys alone.
var provenanceKeysExtra string

// provenanceKeyIDPattern is the restricted charset ValidateProvenanceKeyID
// enforces (16-REVIEW.md WR-02): letters, digits, ".", "_", and "-" only.
// Crucially this EXCLUDES "," and "=" — the two characters
// FormatProvenanceKeys's comma-separated "keyid=<base64 pubkey>" spec uses
// as its own delimiters. A key id containing either would silently break
// the format's own round trip (a "," splits one segment into bogus ones; a
// bare "=" shifts where ParseProvenanceKeys cuts the id from the base64
// value) rather than being rejected at the point the id is actually
// chosen.
var provenanceKeyIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidateProvenanceKeyID rejects a key id outright — before it is ever
// embedded in a FormatProvenanceKeys spec or accepted by ParseProvenanceKeys
// — if it is empty or contains any character outside
// provenanceKeyIDPattern. cmd/topos-provenance's keygen and sign
// subcommands call this on their own --key-id flag so a typo (most
// concretely a stray "," or "=") fails loudly with a clear, actionable
// error naming the exact key id chosen, rather than surfacing later as an
// opaque "malformed provenance key segment" parse failure deep inside
// AcceptedProvenanceKeys — a failure mode whose actual effect
// (AcceptedProvenanceKeys fails safe and simply trusts nothing extra, per
// its own doc comment) is easy to miss during a release-key rotation.
func ValidateProvenanceKeyID(id string) error {
	if id == "" {
		return errors.New("pluginhost: provenance key id must not be empty")
	}
	if !provenanceKeyIDPattern.MatchString(id) {
		return fmt.Errorf("pluginhost: provenance key id %q contains a character outside the allowed charset (letters, digits, \".\", \"_\", \"-\") — in particular \",\" and \"=\" are the spec's own delimiters and would silently break FormatProvenanceKeys/ParseProvenanceKeys's round trip", id)
	}
	return nil
}

// ParseProvenanceKeys parses spec — the comma-separated "keyid=<base64
// pubkey>" format FormatProvenanceKeys produces — into a []ProvenanceKey
// sorted by key id. An empty (or whitespace-only) spec is NOT an error:
// it returns an empty, non-nil slice. Every other malformed shape — a
// segment with no "=", an empty or out-of-charset key id (see
// ValidateProvenanceKeyID), or a base64 value whose decoded length is not
// ed25519.PublicKeySize — is REJECTED outright, naming the offending
// segment; nothing is ever silently dropped, mirroring ParseManifest's own
// never-silently-drop discipline (manifest.go).
func ParseProvenanceKeys(spec string) ([]ProvenanceKey, error) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return []ProvenanceKey{}, nil
	}

	keys := make([]ProvenanceKey, 0)
	for _, segment := range strings.Split(trimmed, ",") {
		idx := strings.Index(segment, "=")
		if idx < 0 {
			return nil, fmt.Errorf("pluginhost: malformed provenance key segment %q: missing \"=\"", segment)
		}
		id := segment[:idx]
		encoded := segment[idx+1:]
		if err := ValidateProvenanceKeyID(id); err != nil {
			return nil, fmt.Errorf("pluginhost: malformed provenance key segment %q: %w", segment, err)
		}
		raw, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("pluginhost: malformed provenance key segment %q: invalid base64: %w", segment, err)
		}
		if len(raw) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("pluginhost: malformed provenance key segment %q: decoded key is %d bytes, want %d", segment, len(raw), ed25519.PublicKeySize)
		}
		keys = append(keys, ProvenanceKey{ID: id, PublicKey: ed25519.PublicKey(raw)})
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].ID < keys[j].ID })
	return keys, nil
}

// FormatProvenanceKeys renders keys as the comma-separated "keyid=<base64
// pubkey>" spec ParseProvenanceKeys parses, sorted by key id — the ONE
// producer of this shape (cmd/topos-provenance's keygen subcommand and
// this package's own tests go through this, never hand-format a spec),
// mirroring FormatManifest's round-trip contract.
func FormatProvenanceKeys(keys []ProvenanceKey) string {
	sorted := make([]ProvenanceKey, len(keys))
	copy(sorted, keys)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	parts := make([]string, 0, len(sorted))
	for _, k := range sorted {
		parts = append(parts, k.ID+"="+base64.StdEncoding.EncodeToString(k.PublicKey))
	}
	return strings.Join(parts, ",")
}

var (
	// provenanceExtraOnce/provenanceExtraParsed cache provenanceKeysExtra's
	// ONE parse — go build's own -ldflags -X mechanism sets it before
	// main() ever runs, so it never changes for the lifetime of a real
	// kernel process (mirrors manifest.go's trustManifestOnce exactly).
	provenanceExtraOnce   sync.Once
	provenanceExtraParsed []ProvenanceKey

	// provenanceOverrideMu guards provenanceOverride/provenanceOverrideSet
	// — the TEST-ONLY seam OverrideProvenanceKeys installs below. Never
	// written outside that function and this file's own
	// AcceptedProvenanceKeys reader.
	provenanceOverrideMu  sync.RWMutex
	provenanceOverride    []ProvenanceKey
	provenanceOverrideSet bool
)

// AcceptedProvenanceKeys returns a defensive copy of the currently
// effective accepted key SET (D-03): an installed OverrideProvenanceKeys
// value when a test has set one, otherwise embeddedProvenanceKeys plus
// provenanceKeysExtra's parsed entries. A malformed provenanceKeysExtra
// value (a hand-corrupted -ldflags spec, never one FormatProvenanceKeys
// itself would produce) is treated as contributing no extra keys at all
// — trusting nothing is the fail-safe state, mirroring TrustManifest's
// own malformed-link-time-input discipline (manifest.go).
// operatorProvenanceKeys is the operator's own accepted keys —
// [[plugins.trusted_keys]] — installed by SetOperatorProvenanceKeys from
// the kernel's config at load, at every config apply, and at every
// launch (host.go reads them from the raw config it already holds for
// the pins). Decision D-12 (revised at davison/topos#49): pluginhost
// never reads a file, an environment variable or configuration itself
// — the KERNEL hands it the operator's keys, exactly as it hands it the
// operator's pins.
var (
	operatorProvenanceMu   sync.RWMutex
	operatorProvenanceKeys []ProvenanceKey
)

// SetOperatorProvenanceKeys replaces the operator's accepted keys with
// keys (copied; Word forced to KeyWordOperator). Nil or empty clears.
func SetOperatorProvenanceKeys(keys []ProvenanceKey) {
	cp := make([]ProvenanceKey, 0, len(keys))
	for _, k := range keys {
		k.Word = KeyWordOperator
		cp = append(cp, k)
	}
	operatorProvenanceMu.Lock()
	operatorProvenanceKeys = cp
	operatorProvenanceMu.Unlock()
}

// OperatorProvenanceKeysFromConfig converts the operator's
// [[plugins.trusted_keys]] entries into ProvenanceKeys. Config validation
// (kernel/config.validateTrustedKeys) has already refused a malformed
// entry, so an undecodable key here is skipped rather than fatal — the
// fail-safe state is trusting nothing.
func OperatorProvenanceKeysFromConfig(entries []config.TrustedKey) []ProvenanceKey {
	out := make([]ProvenanceKey, 0, len(entries))
	for _, e := range entries {
		raw, err := base64.StdEncoding.DecodeString(e.PublicKey)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			continue
		}
		out = append(out, ProvenanceKey{ID: e.ID, PublicKey: ed25519.PublicKey(raw), Word: KeyWordOperator})
	}
	return out
}

func AcceptedProvenanceKeys() []ProvenanceKey {
	provenanceOverrideMu.RLock()
	if provenanceOverrideSet {
		out := make([]ProvenanceKey, len(provenanceOverride))
		copy(out, provenanceOverride)
		provenanceOverrideMu.RUnlock()
		return out
	}
	provenanceOverrideMu.RUnlock()

	provenanceExtraOnce.Do(func() {
		parsed, err := ParseProvenanceKeys(provenanceKeysExtra)
		if err != nil {
			provenanceExtraParsed = []ProvenanceKey{}
			return
		}
		provenanceExtraParsed = parsed
	})

	operatorProvenanceMu.RLock()
	operator := operatorProvenanceKeys
	operatorProvenanceMu.RUnlock()

	out := make([]ProvenanceKey, 0, len(embeddedProvenanceKeys)+len(provenanceExtraParsed)+len(operator))
	for _, k := range embeddedProvenanceKeys {
		k.Word = KeyWordEmbedded
		out = append(out, k)
	}
	for _, k := range provenanceExtraParsed {
		k.Word = KeyWordBuild
		out = append(out, k)
	}
	out = append(out, operator...)
	return out
}

// OverrideProvenanceKeys is a TEST-ONLY seam mirroring
// OverrideBuildManifest (manifest.go) exactly: it installs keys as the
// CURRENT effective accepted key set (AcceptedProvenanceKeys returns
// exactly this slice, ignoring embeddedProvenanceKeys/provenanceKeysExtra
// entirely, until restore is called) and returns a restore func that
// puts the previous override state — installed or not — back exactly.
//
// MUST NEVER be called from production code — production code has no
// legitimate reason to override the accepted key set at run time; doing
// so would recreate exactly the "trust whatever key is configured"
// bypass D-12's link-time-only discipline exists to close. This is also
// the D-12 seam a dev/e2e build's link-time -ldflags injection (never
// this function) uses to trust its own locally signed fixtures.
func OverrideProvenanceKeys(keys []ProvenanceKey) (restore func()) {
	provenanceOverrideMu.Lock()
	prevOverride := provenanceOverride
	prevSet := provenanceOverrideSet

	cp := make([]ProvenanceKey, len(keys))
	copy(cp, keys)
	provenanceOverride = cp
	provenanceOverrideSet = true
	provenanceOverrideMu.Unlock()

	return func() {
		provenanceOverrideMu.Lock()
		provenanceOverride = prevOverride
		provenanceOverrideSet = prevSet
		provenanceOverrideMu.Unlock()
	}
}

// BuildProvenanceManifest renders release and entries as deterministic
// manifest bytes (D-05/D-06/D-07): entries sorted by name, two-space
// indented JSON, trailing newline — the ONE producer of this shape
// (cmd/topos-provenance and this package's own tests share this, never
// hand-format the JSON), so a given release always produces
// byte-identical manifest bytes, matching FormatManifest's own
// reproducibility discipline. Every entry's SHA256 is validated with the
// existing isHexDigest (manifest.go) — a malformed digest is rejected by
// name, never silently written.
func BuildProvenanceManifest(release ProvenanceRelease, entries []ProvenanceEntry) ([]byte, error) {
	sorted := make([]ProvenanceEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	for _, e := range sorted {
		if !isHexDigest(e.SHA256) {
			return nil, fmt.Errorf("pluginhost: build provenance manifest: entry %q has a malformed sha256 %q — must be %d lowercase hex characters", e.Name, e.SHA256, manifestDigestLen)
		}
	}

	manifest := ProvenanceManifest{
		Schema:  ProvenanceSchema,
		Release: release,
		Plugins: sorted,
	}

	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("pluginhost: build provenance manifest: %w", err)
	}
	return append(out, '\n'), nil
}

// SignProvenanceManifest signs manifestBytes — EXACTLY the bytes
// BuildProvenanceManifest returned, never a re-serialized or re-parsed
// form — with priv under keyID, and returns the signature-file bytes
// VerifySignedProvenance's sibling reader expects. The signature covers
// manifestBytes VERBATIM (D-05): the signer and verifier both operate on
// raw bytes, never a parsed-then-re-emitted structure, so verification
// never depends on JSON field ordering or whitespace.
func SignProvenanceManifest(manifestBytes []byte, keyID string, priv ed25519.PrivateKey) ([]byte, error) {
	sig := ed25519.Sign(priv, manifestBytes)

	pub, ok := priv.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("pluginhost: sign provenance manifest: private key has no ed25519 public half")
	}
	out, err := json.MarshalIndent(ProvenanceSignature{
		Schema:    ProvenanceSignatureSchema,
		KeyID:     keyID,
		Algorithm: "ed25519",
		Signature: base64.StdEncoding.EncodeToString(sig),
		PublicKey: base64.StdEncoding.EncodeToString(pub),
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("pluginhost: sign provenance manifest: %w", err)
	}
	return append(out, '\n'), nil
}

// readAndVerifyProvenanceManifest reads manifestPath and its sibling
// signature file, verifies the ed25519 signature against keyByID, and
// validates the manifest's schema and release platform — every step a
// single candidate in VerifySignedProvenance's scan must pass before its
// entries are consulted at all. A failure at ANY step returns ok=false
// with a diagnostic string naming manifestPath's base name and the
// specific cause — it is the caller's job to treat this as "this
// candidate contributes no evidence", never a scan-aborting error (D-08:
// one junk or hostile file must not veto a valid manifest beside it).
func readAndVerifyProvenanceManifest(manifestPath string, keyByID map[string]ProvenanceKey) (manifest ProvenanceManifest, key ProvenanceKey, offer *KeyOffer, diagnostic string, ok bool) {
	fname := filepath.Base(manifestPath)
	sigPath := strings.TrimSuffix(manifestPath, ProvenanceManifestSuffix) + ProvenanceSignatureSuffix

	info, err := os.Stat(manifestPath)
	if err != nil {
		return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: cannot stat manifest: %v", fname, err), false
	}
	if info.Size() > MaxProvenanceManifestBytes {
		return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: manifest exceeds %d bytes, refusing to read", fname, MaxProvenanceManifestBytes), false
	}
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: cannot read manifest: %v", fname, err), false
	}

	sigInfo, err := os.Stat(sigPath)
	if err != nil {
		return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: missing or unreadable sibling signature file %s: %v", fname, filepath.Base(sigPath), err), false
	}
	if sigInfo.Size() > MaxProvenanceManifestBytes {
		return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: sibling signature file exceeds %d bytes, refusing to read", fname, MaxProvenanceManifestBytes), false
	}
	sigBytes, err := os.ReadFile(sigPath)
	if err != nil {
		return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: cannot read sibling signature file %s: %v", fname, filepath.Base(sigPath), err), false
	}

	var sig ProvenanceSignature
	if err := json.Unmarshal(sigBytes, &sig); err != nil {
		return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: sibling signature file %s is malformed JSON: %v", fname, filepath.Base(sigPath), err), false
	}
	if sig.Algorithm != "ed25519" {
		return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: signature algorithm %q is not \"ed25519\"", fname, sig.Algorithm), false
	}
	sigRaw, err := base64.StdEncoding.DecodeString(sig.Signature)
	if err != nil || len(sigRaw) != ed25519.SignatureSize {
		return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: signature is not a valid base64-encoded ed25519 signature", fname), false
	}
	key, known := keyByID[sig.KeyID]
	var carried ed25519.PublicKey
	if sig.PublicKey != "" {
		raw, decErr := base64.StdEncoding.DecodeString(sig.PublicKey)
		if decErr != nil || len(raw) != ed25519.PublicKeySize {
			return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: signature file carries a public_key that is not a valid base64-encoded ed25519 public key", fname), false
		}
		carried = ed25519.PublicKey(raw)
	}
	// A known id whose carried key differs is NOT the known key: it is an
	// unknown key wearing a trusted name (davison/topos#49) — treated as
	// unknown below, and if it verifies, offered with Reused set.
	reused := false
	if known && carried != nil && !carried.Equal(key.PublicKey) {
		known = false
		reused = true
	}
	if !known {
		if carried == nil {
			return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: signature names unknown key id %q", fname, sig.KeyID), false
		}
		if !ed25519.Verify(carried, manifestBytes, sigRaw) {
			return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: signature names unknown key id %q and does not verify against the public key it carries", fname, sig.KeyID), false
		}
		// The signature verifies against the carried key: this manifest
		// was demonstrably signed by that key. Still NO evidence — but the
		// manifest is parsed so the caller can decide whether it names the
		// binary, in which case the key becomes an offer.
		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: manifest is malformed JSON despite a valid (unknown-key) signature", fname), false
		}
		if manifest.Schema != ProvenanceSchema {
			return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: manifest schema %q is not %q", fname, manifest.Schema, ProvenanceSchema), false
		}
		if manifest.Release.OS != runtime.GOOS || manifest.Release.Arch != runtime.GOARCH {
			return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: manifest release os/arch %s/%s does not match this kernel's %s/%s", fname, manifest.Release.OS, manifest.Release.Arch, runtime.GOOS, runtime.GOARCH), false
		}
		offer = &KeyOffer{KeyID: sig.KeyID, PublicKey: carried, Fingerprint: KeyFingerprint(carried), Reused: reused}
		what := "unknown key id"
		if reused {
			what = "a key id already trusted, but with a DIFFERENT public key — a reused id"
		}
		return manifest, ProvenanceKey{}, offer, fmt.Sprintf("%s: signature names %s %q (verifies against the public key it carries; fingerprint %s) — no evidence; the operator may be offered the key", fname, what, sig.KeyID, offer.Fingerprint), false
	}
	if !ed25519.Verify(key.PublicKey, manifestBytes, sigRaw) {
		return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: signature does not verify against key %q — manifest bytes do not match what was signed", fname, sig.KeyID), false
	}

	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: manifest is malformed JSON despite a valid signature", fname), false
	}
	if manifest.Schema != ProvenanceSchema {
		return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: manifest schema %q is not %q", fname, manifest.Schema, ProvenanceSchema), false
	}
	if manifest.Release.OS != runtime.GOOS || manifest.Release.Arch != runtime.GOARCH {
		return ProvenanceManifest{}, ProvenanceKey{}, nil, fmt.Sprintf("%s: manifest release os/arch %s/%s does not match this kernel's %s/%s", fname, manifest.Release.OS, manifest.Release.Arch, runtime.GOOS, runtime.GOARCH), false
	}

	return manifest, key, nil, "", true
}

// findProvenanceEntry returns the entry in manifest.Plugins named name,
// if any. Absent means this manifest simply does not cover this binary
// — not a failure of the manifest itself.
func findProvenanceEntry(manifest ProvenanceManifest, name string) (ProvenanceEntry, bool) {
	for _, e := range manifest.Plugins {
		if e.Name == name {
			return e, true
		}
	}
	return ProvenanceEntry{}, false
}

// VerifySignedProvenance computes path's current SHA-256 (via HashBinary
// — reused, never a second hashing implementation) and scans every
// non-empty directory in dirs for candidate release manifests (files
// ending in ProvenanceManifestSuffix). The scan is EXHAUSTIVE before it
// decides — this ordering is load-bearing for D-08: completing the scan
// over every candidate before resolving precedence is what lets an
// independently upgraded plugin (an old manifest naming its superseded
// digest sitting beside a new manifest naming its current one) launch
// regardless of directory read order. Precedence, once the scan is
// complete:
//
//  1. any candidate MATCH (validly-signed manifest naming name with the
//     on-disk digest) -> success, evidence names that manifest's file;
//  2. otherwise, any candidate NAMED MISMATCH (validly-signed manifest
//     naming name with a DIFFERENT digest) -> tamper refusal, a wrapped
//     ErrProvenanceUnverified naming the binary, the mismatching
//     manifest file, and the on-disk digest;
//  3. otherwise -> no evidence (hash is still returned; evidence is
//     empty; err is nil — the caller treats this as external tier, D-11).
//
// A candidate that fails signature/schema/platform verification (see
// readAndVerifyProvenanceManifest) contributes NO evidence either way —
// it is recorded as a named diagnostic string in diagnostics, never an
// error that aborts the scan, because one junk or hostile file in a
// directory must not veto a valid manifest beside it. diagnostics is
// always returned (even alongside a nil error) so a caller holding a
// logger (EvaluateTrust, then launch in host.go) can surface every
// collapsed candidate failure by name — this widens the return shape
// 16-01-PLAN.md's action text sketches (hash, evidence, err) with an
// explicit diagnostics slice, because otherwise no candidate-level
// diagnostic could ever reach Trust.Diagnostics, which the same task
// requires EvaluateTrust to carry out.
func VerifySignedProvenance(dirs Dirs, name, path string) (hash string, evidence string, diagnostics []string, err error) {
	res, err := VerifySignedProvenanceDetailed(dirs, name, path)
	return res.Hash, res.Evidence, res.Diagnostics, err
}

// ProvenanceResult is VerifySignedProvenanceDetailed's result: the
// three values VerifySignedProvenance returns, plus WHICH accepted key
// vouched (KeyID/Word — the word decides the tier) and, when nothing
// vouched, an Offer from an unknown self-describing key that signed a
// manifest naming this binary with a matching digest.
type ProvenanceResult struct {
	Hash        string
	Evidence    string
	KeyID       string
	Word        KeyWord
	Offer       *KeyOffer
	Diagnostics []string
}

// VerifySignedProvenanceDetailed is VerifySignedProvenance with the
// vouching key and any KeyOffer surfaced (davison/topos#49). Precedence
// is unchanged; an offer never outranks anything — it is reported only
// in the no-evidence case.
func VerifySignedProvenanceDetailed(dirs Dirs, name, path string) (res ProvenanceResult, err error) {
	hash, err := HashBinary(path)
	if err != nil {
		return ProvenanceResult{}, err
	}
	res.Hash = hash
	var diagnostics []string

	acceptedKeys := AcceptedProvenanceKeys()
	keyByID := make(map[string]ProvenanceKey, len(acceptedKeys))
	for _, k := range acceptedKeys {
		keyByID[k.ID] = k
	}

	var matchFile string
	var matchKey ProvenanceKey
	var mismatchFile, mismatchDigest string
	haveMismatch := false
	var offer *KeyOffer

	for _, dir := range []string{dirs.Trusted, dirs.External} {
		if dir == "" {
			continue
		}
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			// A missing/unreadable directory contributes nothing — this
			// is not itself a diagnostic because it names no candidate
			// manifest file at all, mirroring DiscoverAllBinaries' own
			// missing-directory-is-empty-state contract.
			continue
		}
		for _, e := range entries {
			fname := e.Name()
			if !strings.HasSuffix(fname, ProvenanceManifestSuffix) {
				continue
			}
			if !isRegularFileFollowingSymlinks(dir, e) {
				continue
			}
			manifestPath := filepath.Join(dir, fname)

			manifest, key, candidateOffer, diag, ok := readAndVerifyProvenanceManifest(manifestPath, keyByID)
			if !ok {
				diagnostics = append(diagnostics, diag)
				if candidateOffer != nil && offer == nil {
					// An unknown self-describing key signed this manifest:
					// it is an offer for THIS binary only if the manifest
					// names it with the on-disk digest — otherwise the key
					// signed something else entirely.
					if entry, found := findProvenanceEntry(manifest, name); found && entry.SHA256 == hash {
						offer = candidateOffer
					}
				}
				continue
			}

			entry, found := findProvenanceEntry(manifest, name)
			if !found {
				// This manifest simply does not cover this binary — not
				// a failure, not a diagnostic.
				continue
			}
			if entry.SHA256 == hash {
				if matchFile == "" {
					matchFile = fname
					matchKey = key
				}
				continue
			}
			haveMismatch = true
			if mismatchFile == "" {
				mismatchFile = fname
				mismatchDigest = entry.SHA256
			}
			diagnostics = append(diagnostics, fmt.Sprintf("%s: names %q with digest %s, on-disk digest is %s", fname, name, entry.SHA256, hash))
		}
	}

	res.Diagnostics = diagnostics
	switch {
	case matchFile != "":
		res.Evidence, res.KeyID, res.Word = matchFile, matchKey.ID, matchKey.Word
		return res, nil
	case haveMismatch:
		return res, fmt.Errorf("%w: %q — manifest %s names a digest that does not match the on-disk binary (signed digest %s, on-disk %s)", ErrProvenanceUnverified, name, mismatchFile, mismatchDigest, hash)
	default:
		res.Offer = offer
		return res, nil
	}
}

// EvaluateTrust is the single authority launch (host.go) consults to
// decide a binary's trust tier (D-10 coexistence: one verifier with two
// evidence sources, not two verifiers behind an interface — Phase 17
// retires the link-time arm by deleting one branch inside this function,
// rather than unwinding an interface and its implementations):
//
//   - Ask the link-time arm (VerifyTrustedBinary, manifest.go). Success
//     sets Tier=TierTrusted, Evidence="build-manifest".
//   - A link-time arm digest MISMATCH (name present, bytes differ) is a
//     tamper refusal exactly as before this phase — returned
//     IMMEDIATELY, wrapped, never falling through to the signed arm or
//     silently demoting to TierExternal (D-13). This refusal carries
//     Tier=TierTrusted (16-06-PLAN.md Task 1, 16-VERIFICATION.md gap 1):
//     the link-time manifest positively NAMED this binary, so the
//     refusal is a trusted-tier refusal on the wire — but it is the
//     returned error, not the tier, that actually stops the launch.
//   - Otherwise (the link-time arm simply does not name this binary — no
//     manifest embedded at all, or a manifest that just doesn't mention
//     it) ask the signed arm (VerifySignedProvenance). A tamper refusal
//     from THAT arm also returns immediately, wrapped, and also carries
//     Tier=TierTrusted for the identical reason: a signed manifest
//     positively named this binary.
//   - Success from the signed arm sets Tier=TierTrusted, Evidence to the
//     vouching manifest's filename.
//   - Both arms declining because the binary is simply not named
//     anywhere is NOT an error: Tier=TierExternal, Evidence="", err=nil
//     — the "no evidence, therefore no trust" state (D-11), external-tier
//     semantics, never a trusted-tier refusal.
//
// EvaluateTrust holds no cross-call mutable state of its own beyond the
// guarded test-only override seams (OverrideBuildManifest,
// OverrideProvenanceKeys) — every call re-hashes path and re-reads every
// on-disk manifest, so concurrent launches and an interrupted launch
// each start from the same on-disk facts and never observe a
// partially-updated trust decision (TRUST-04). Trust.Diagnostics carries
// every candidate-level diagnostic string out so a caller holding a
// logger can emit them; EvaluateTrust itself performs no logging.
//
// The residual window between EvaluateTrust returning and the caller
// constructing exec.Command is narrowed by this gate running immediately
// before that call, not eliminated — matching the accepted TOCTOU
// posture this project's Phase 12 EvalSymlinks precedent already
// documents (T-16-04).
func EvaluateTrust(dirs Dirs, name, path string) (Trust, error) {
	buildHash, buildErr := VerifyTrustedBinary(name, path)
	if buildErr == nil {
		return Trust{Tier: TierTrusted, Hash: buildHash, Evidence: "build-manifest"}, nil
	}

	if expected, ok := TrustManifest()[name]; ok && expected != buildHash {
		// The link-time arm found a real entry whose digest disagrees
		// with what's on disk — a tamper refusal (D-13). The signed arm
		// never gets a chance to override this: only "this binary is
		// simply absent from the link-time manifest" falls through to
		// it below. Tier is set to TierTrusted here (16-06-PLAN.md Task
		// 1, 16-VERIFICATION.md gap 1): a refusal is a TRUSTED-tier
		// refusal — the link-time manifest positively named this
		// binary — so the wire tier field a caller (launch, host.go)
		// surfaces is never the empty zero value. Tier is a REPORTING
		// field only; the returned buildErr, untouched, is what
		// actually refuses the launch.
		return Trust{Tier: TierTrusted, Hash: buildHash}, buildErr
	}

	prov, provErr := VerifySignedProvenanceDetailed(dirs, name, path)
	provHash, evidence, diagnostics := prov.Hash, prov.Evidence, prov.Diagnostics
	if provErr != nil {
		// Same reasoning as the link-time branch above: a signed
		// manifest positively named this binary with a digest that no
		// longer matches, so this refusal is also TRUSTED-tier on the
		// wire. Tier is a reporting field only; provErr, untouched, is
		// what actually refuses the launch.
		return Trust{Tier: TierTrusted, Hash: provHash, Diagnostics: diagnostics}, provErr
	}
	if evidence != "" {
		tier := TierTrusted
		if prov.Word == KeyWordOperator {
			// The operator's word, not the kernel author's: a distinct
			// tier, so the chip and the API say whose word it is
			// (davison/topos#49).
			tier = TierOperatorTrusted
		}
		return Trust{Tier: tier, Hash: provHash, Evidence: evidence, Diagnostics: diagnostics, KeyID: prov.KeyID, Word: prov.Word}, nil
	}

	return Trust{Tier: TierExternal, Hash: provHash, Diagnostics: diagnostics, Offer: prov.Offer}, nil
}
