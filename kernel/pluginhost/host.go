// Package pluginhost discovers, launches, and supervises source plugin
// subprocesses over the go-plugin gRPC transport. It is the only package
// permitted to import hashicorp/go-plugin's client side; kernel/httpapi
// must never import this package (see kernel/httpapi/stream.go).
package pluginhost

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/sdk"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// ErrItemNotFound is returned by Host.Fetch when the plugin reports the
// requested source id does not exist (gRPC codes.NotFound), or when no
// launched plugin is registered for the item's source type.
var ErrItemNotFound = errors.New("pluginhost: item not found")

// ErrSourceUnavailable is returned by Host.Fetch when the plugin's live
// call to its source system failed transiently (gRPC codes.Unavailable,
// or any other transport-level failure calling the plugin).
var ErrSourceUnavailable = errors.New("pluginhost: source unavailable")

// ErrPinMismatch is returned by launch (wrapped inside *pinMismatchError,
// which Discover/Reconcile extract via errors.As) when an external-tier
// binary's recomputed SHA-256 does not equal its pinned value in
// [plugins.pins] — including the "no pin recorded at all" case, which is
// treated identically to a mismatch (the empty pinned value never matches a
// real digest). T-11-07/PLUG-07/D-03: the refusal happens BEFORE
// exec.Command is ever constructed, so a tampered or unpinned external
// binary is never executed. Trusted-tier binaries never reach this check at
// all (D-04) — see launch's own doc comment.
var ErrPinMismatch = errors.New("pluginhost: external plugin binary does not match its pinned hash")

// LaunchFailurePinMismatch is the one populated LaunchFailure.Reason value
// this phase introduces — kept as a named constant (rather than an inline
// string literal at every call site) so a future second failure-reason
// class has an established place to add its own constant beside this one.
const LaunchFailurePinMismatch = "pin_mismatch"

// LaunchFailureManifestUnverified is LaunchFailure.Reason's second
// populated value (13-05-PLAN.md, D-12/D-13): a trusted-directory binary
// whose name is absent from, or whose on-disk hash does not match, the
// kernel's link-time build manifest. Unlike LaunchFailurePinMismatch, this
// reason's remedy is NOT "trust and re-pin" — a trusted-tier binary is
// never demoted to external tier and run anyway (D-13); its only path to
// running at all is the existing explicit external-tier consent and pin
// flow.
const LaunchFailureManifestUnverified = "manifest_unverified"

// ErrContractIncompatible is wrapped inside the *launchError a
// contract-generation refusal returns (M1-R6/DIST-03, davison/topos#17):
// the plugin completed the go-plugin handshake and answered Describe, but
// the contract generation it declared (DescribeResponse.contract_version)
// is outside sdk.SupportedContractVersions — including the empty string,
// which names NO generation and is never treated as compatible. The
// subprocess is killed before any further RPC; the refusal names both
// generations.
var ErrContractIncompatible = errors.New("pluginhost: plugin declares a contract generation this kernel does not support")

// LaunchFailureHandshakeIncompatible names a binary that failed the
// go-plugin handshake for a VERSION reason — the subprocess started but
// spoke a different protocol version than sdk.Handshake's (a plugin
// built against a different contract era; the wrapped go-plugin error
// text names both versions). Distinct from LaunchFailureLaunchFailed so
// a client can say "incompatible" rather than merely "broken"
// (M1-R6/DIST-03).
const LaunchFailureHandshakeIncompatible = "handshake_incompatible"

// LaunchFailureContractIncompatible names a binary that handshook
// cleanly but whose Describe declared a contract generation outside
// sdk.SupportedContractVersions (or none at all). The message names the
// declared and supported generations; the remedy is updating the plugin
// fleet or the kernel so the generations agree — never running the
// mismatch anyway.
const LaunchFailureContractIncompatible = "contract_incompatible"

// LaunchFailureLaunchFailed is the generic never-became-a-plugin class
// (M1-R6/DIST-03): the binary is missing from both directories, exited
// before the handshake, emitted something other than a go-plugin
// handshake line, failed Dispense/Describe, or did not implement the
// source interface. Before davison/topos#17 every one of these aborted
// the WHOLE Discover/Reconcile call — a dead boot or a rejected apply
// with nothing on the source list naming the culprit; now each is a
// per-instance record like every other launch-failure class, and the
// message carries the specific cause.
const LaunchFailureLaunchFailed = "launch_failed"

// LaunchAdvisoryShadowed is SourceHealth.LaunchAdvisory's one populated
// value (D-14): a LAUNCHED, verified trusted-tier instance whose binary
// name also exists as a regular file in the configured external
// directory — the trusted copy won the launch (D-11's shadow rule
// unchanged), but the operator's consented-to external plugin is not the
// one actually running. Distinct from LaunchFailure/LaunchFailureReason:
// a shadowed instance DID launch successfully; this is an advisory about
// provenance ambiguity, never a launch refusal.
const LaunchAdvisoryShadowed = "shadowed"

// LaunchFailure is one instance's SOFT, per-instance launch failure.
// Since M1-R6/DIST-03 (davison/topos#17) EVERY launch-failure class is
// recorded here — the two trust refusals (pin_mismatch,
// manifest_unverified), the handshake-version refusal
// (handshake_incompatible), the contract-generation refusal
// (contract_incompatible), and the generic never-became-a-plugin class
// (launch_failed) — so the instance is simply absent from
// Host.Plugins() rather than taking down every other configured
// source's boot or apply.
type LaunchFailure struct {
	Instance    string
	Plugin      string
	DisplayName string
	Tier        Tier
	Reason      string
	PinnedHash  string
	CurrentHash string
	Message     string
	// OfferedKey is the unknown self-describing key that signed a
	// manifest naming this external binary (davison/topos#49) — carried
	// on a pin-mismatch failure so the UI can offer "trust this key"
	// beside "trust updated binary". Nil otherwise.
	OfferedKey *KeyOffer
}

// pinMismatchError carries the structured facts a caller (Discover,
// Reconcile) needs to build a LaunchFailure record WITHOUT re-hashing the
// binary a second time (the "small helper" this phase's action text calls
// for) — launch computes the current hash exactly once, and this type
// threads both hashes back out through the returned error rather than
// forcing the caller to re-derive them from the error string.
type pinMismatchError struct {
	instance, plugin, displayName string
	tier                          Tier
	pinnedHash, currentHash       string
	offeredKey                    *KeyOffer
}

// Error names the instance, the binary, the pinned value (or "not pinned"
// when absent) and the on-disk value — the fail-loudly-by-name convention
// this package's matchconfig.go already establishes, extended here.
func (e *pinMismatchError) Error() string {
	pinnedDisplay := e.pinnedHash
	if pinnedDisplay == "" {
		pinnedDisplay = "not pinned"
	}
	return fmt.Sprintf(
		"pluginhost: instance %q binary %q hash mismatch: pinned=%s current=%s",
		e.instance, e.plugin, pinnedDisplay, e.currentHash,
	)
}

// Unwrap makes errors.Is(err, ErrPinMismatch) true for every
// *pinMismatchError, so a caller that only cares about the failure CLASS
// (not its structured fields) can test for it with the plain sentinel.
func (e *pinMismatchError) Unwrap() error { return ErrPinMismatch }

// toLaunchFailure converts e into the exported LaunchFailure shape
// Host.launchFailures stores, filling in the fixed Reason/message fields
// this phase's one failure class always carries.
func (e *pinMismatchError) toLaunchFailure() LaunchFailure {
	return LaunchFailure{
		Instance:    e.instance,
		Plugin:      e.plugin,
		DisplayName: e.displayName,
		Tier:        e.tier,
		Reason:      LaunchFailurePinMismatch,
		PinnedHash:  e.pinnedHash,
		CurrentHash: e.currentHash,
		Message:     e.Error(),
		OfferedKey:  e.offeredKey,
	}
}

// manifestUnverifiedError carries the structured facts a caller (Discover,
// Reconcile) needs to build a LaunchFailure record for a trusted-directory
// binary that failed trust verification (13-05-PLAN.md Task 3, widened by
// 16-01-PLAN.md Task 1 to the two-arm coexistence rule, D-10/D-13) —
// mirroring pinMismatchError's shape (instance, plugin, displayName,
// tier, currentHash), minus a "pinned/expected" field: unlike a pin
// mismatch, there is nothing for an operator to re-pin here — the only
// field this failure class has to show is what's actually on disk. cause
// carries EvaluateTrust's own diagnostic text (when evaluation returned
// an error) so Error() names WHICH arm failed and why, rather than only
// the generic wire-vocabulary reason (LaunchFailureManifestUnverified,
// value "manifest_unverified" — deliberately reused rather than widened,
// so web/src/lib/api.ts's closed union, docs/api.md, and the existing
// chip-health precedence chain stay untouched, TRUST-03).
type manifestUnverifiedError struct {
	instance, plugin, displayName string
	tier                          Tier
	currentHash                   string
	// cause is EvaluateTrust's own returned error (nil for the plain
	// "neither arm names this binary at all" case) — carried as a real
	// error, not just its text, so Unwrap below can expose whichever
	// sentinel (ErrManifestUnverified from the link-time arm,
	// ErrProvenanceUnverified from the signed arm) the underlying tamper
	// refusal actually wrapped.
	cause error
}

// Error names the instance, the binary, the on-disk digest, and (when
// available) EvaluateTrust's own cause text — the fail-loudly-by-name
// convention pinMismatchError's own Error method establishes, extended
// here to name which of the two trust arms (D-10) refused and why.
func (e *manifestUnverifiedError) Error() string {
	msg := fmt.Sprintf(
		"pluginhost: instance %q binary %q is not verified by the kernel's build manifest or any signed release manifest (current=%s)",
		e.instance, e.plugin, e.currentHash,
	)
	if e.cause != nil {
		msg += ": " + e.cause.Error()
	}
	return msg
}

// Unwrap makes BOTH errors.Is(err, ErrManifestUnverified) (the
// wire-vocabulary-preserving sentinel this failure class has always
// carried, TRUST-03) AND, when the underlying refusal came from the
// signed arm, errors.Is(err, ErrProvenanceUnverified) true for every
// *manifestUnverifiedError — the multi-error Unwrap() []error form
// (Go 1.20+) lets both sentinels resolve through one wrapped error
// without picking a single "primary" cause.
func (e *manifestUnverifiedError) Unwrap() []error {
	if e.cause != nil {
		return []error{ErrManifestUnverified, e.cause}
	}
	return []error{ErrManifestUnverified}
}

// toLaunchFailure converts e into the exported LaunchFailure shape
// Host.launchFailures stores — mirroring pinMismatchError's own method.
// PinnedHash is deliberately left empty (D-13: a trusted-tier binary is
// never pinned, and this failure's only remedy is the existing
// external-tier consent flow, not a re-pin action).
func (e *manifestUnverifiedError) toLaunchFailure() LaunchFailure {
	return LaunchFailure{
		Instance:    e.instance,
		Plugin:      e.plugin,
		DisplayName: e.displayName,
		Tier:        e.tier,
		Reason:      LaunchFailureManifestUnverified,
		CurrentHash: e.currentHash,
		Message:     e.Error(),
	}
}

// launchError is the per-instance record shape for every launch-failure
// class beyond the two trust refusals (which keep their own richer types
// above): the generic never-became-a-plugin cases, the handshake version
// refusal, and the contract-generation refusal (M1-R6/DIST-03,
// davison/topos#17). Constructed INSIDE launch(), where instance,
// binary, display name and tier are all at hand, so Discover/Reconcile
// record it with one errors.As branch and never re-derive facts from
// error text. tier may be the empty Tier for a binary that was never
// found on disk at all — the one class with no bytes to derive a tier
// from; the wire simply carries "" there.
type launchError struct {
	instance, plugin, displayName string
	tier                          Tier
	reason                        string
	message                       string
	cause                         error
}

func (e *launchError) Error() string { return e.message }

// Unwrap exposes the underlying cause (ErrContractIncompatible for the
// contract class; the wrapped connect/describe error otherwise) so
// errors.Is keeps working through the recorded shape.
func (e *launchError) Unwrap() error { return e.cause }

// toLaunchFailure converts e into the exported LaunchFailure shape —
// mirroring pinMismatchError's and manifestUnverifiedError's own
// methods. CurrentHash/PinnedHash stay empty: none of these classes is
// about bytes-vs-evidence, and the hash facts either never existed
// (missing binary) or add nothing to the named cause.
func (e *launchError) toLaunchFailure() LaunchFailure {
	return LaunchFailure{
		Instance:    e.instance,
		Plugin:      e.plugin,
		DisplayName: e.displayName,
		Tier:        e.tier,
		Reason:      e.reason,
		Message:     e.message,
	}
}

// connectFailureReason classifies a go-plugin client.Client() error into
// the closed LaunchFailure vocabulary: the two protocol-version refusal
// texts go-plugin produces (client.go's "incompatible API version with
// plugin" and "incompatible core API version with plugin" — it exports
// no typed error for either) are LaunchFailureHandshakeIncompatible;
// everything else (a pre-handshake fatal, an unrecognized handshake
// line from a non-plugin binary, a timeout) is the generic
// LaunchFailureLaunchFailed. String matching a dependency's error text
// is fragile by nature, so kernel/pluginhost's handshake fixture test
// launches a REAL plugin served at ProtocolVersion 1 and asserts the
// classification — if go-plugin ever rewords, that test fails by name
// rather than this silently degrading to the generic class.
func connectFailureReason(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "incompatible API version with plugin") ||
		strings.Contains(msg, "incompatible core API version with plugin") {
		return LaunchFailureHandshakeIncompatible
	}
	return LaunchFailureLaunchFailed
}

// hashAndLookupPin computes path's current SHA-256 (via HashBinary) and
// looks up binary's pinned value in pins in one call, so launch's pin-check
// and the LaunchFailure record it may need to build never hash the same
// file twice. pins may be nil (no [plugins.pins] table at all) — a nil map
// read is the Go zero value "", which is exactly the "not pinned" case this
// function's caller already treats identically to a mismatch.
func hashAndLookupPin(path, binary string, pins map[string]string) (currentHash, pinnedHash string, err error) {
	currentHash, err = HashBinary(path)
	if err != nil {
		return "", "", err
	}
	return currentHash, pins[binary], nil
}

// MaxIconBytes is the size ceiling enforced on DescribeResponse.icon at
// capture time (09-01-PLAN.md Task 2, T-09-03): an icon larger than this
// is dropped — never truncated — and never fails the launch.
const MaxIconBytes = 65536

// allowedIconMIME is the kernel-side mime allowlist enforced on
// DescribeResponse.icon_mime at capture time (T-09-02): a plugin cannot
// choose the Content-Type the kernel later serves outside this set. A mime
// not in this set, or an empty mime paired with non-empty icon bytes, has
// its icon dropped and treated identically to "no icon declared."
var allowedIconMIME = map[string]bool{
	"image/svg+xml": true,
	"image/png":     true,
}

// captureIcon validates a Describe response's icon/icon_mime fields against
// MaxIconBytes and allowedIconMIME, returning ok=false whenever the icon is
// absent or fails validation — every failure is treated as "this plugin
// declared no icon," never as a launch failure (09-01-PLAN.md Task 2).
func captureIcon(desc *toposv1.DescribeResponse) (iconBytes []byte, iconMIME string, ok bool) {
	b := desc.GetIcon()
	m := desc.GetIconMime()

	if len(b) == 0 {
		return nil, "", false
	}
	if len(b) > MaxIconBytes {
		return nil, "", false
	}
	if m == "" {
		return nil, "", false
	}
	if !allowedIconMIME[m] {
		return nil, "", false
	}
	return b, m, true
}

// Plugin is one launched, handshaken source plugin subprocess.
type Plugin struct {
	name            string // config key under [sources.<name>] — THE instance identity (D-08)
	sourceType      string // learned via Describe, not trusted from the filename — plugin kind only, never identity
	pluginName      string // Describe-learned display name (e.g. "paperless-ngx") — the plugin KIND's own label
	displayName     string // resolved instance display name (config display_name, or name if unset) — D-09
	matchVocabulary []string
	// iconBytes/iconMIME are captured from the same Describe call launch()
	// already makes (no new RPC), validated by captureIcon. Both are the
	// zero value when the plugin declared no icon, declared one that
	// failed validation, or was built against the pre-Phase-9 contract
	// (09-01-PLAN.md Task 2).
	iconBytes []byte
	iconMIME  string
	// src is the config.Source this instance was launched with — Reconcile
	// (07-02-PLAN.md Task 1, D-06/D-07) compares a currently-configured
	// instance's config.Source against this value to decide whether the
	// instance is unchanged (subprocess left running untouched) or has a
	// different connection config (subprocess relaunched). Never used for
	// anything beyond that diff.
	src    config.Source
	client *goplugin.Client
	impl   sdk.SourcePlugin
	// tier is the provenance Tier this instance's binary resolved to at
	// launch time, via ResolveBinary (T-11-01) — set exactly once here
	// and never overwritten from anything the plugin process itself
	// later reports (Describe, Health, or any other RPC).
	tier Tier
	// pinnedHash is the SHA-256 this instance's binary matched in
	// [plugins.pins] at launch time (Phase 11 plan 11-03) — populated only
	// for a real (non-trial), TierExternal launch that passed the pin
	// check in launch() below; empty for every trusted-tier instance
	// (D-04: never pinned) and for a trial-launched instance (pin
	// verification is skipped entirely for describeOnly launches). Read by
	// ProbeSources to populate SourceHealth.PinnedHash, so a healthy
	// external-tier source's GET /api/sources entry can show the pin it is
	// currently running under, not only a broken one.
	pinnedHash string
	// binaryHash is this instance's on-disk SHA-256, computed for EVERY
	// TierExternal launch (11-03-PLAN.md Task 2) — including a describeOnly
	// trial launch, where it is the ONLY way to learn an external binary's
	// identity/hash before any pin can exist for it (T-11-14). Empty for
	// TierTrusted (nothing is pinned for the trusted tier, D-04).
	binaryHash string
	// manifestHash is this instance's on-disk SHA-256 as verified against
	// the kernel's link-time build manifest (13-05-PLAN.md Task 3,
	// VerifyTrustedBinary) — populated for EVERY TierTrusted launch that
	// passed verification (a launch that failed verification never
	// reaches this point at all — see launch's own manifest-gate branch).
	// Distinct from binaryHash (external-tier-only, pin-related); kept for
	// provenance/debugging symmetry with the external-tier hash fields
	// above, not currently surfaced through any public getter.
	manifestHash string
	// shadowed is true when this TierTrusted instance's binary name also
	// exists as a regular file in the configured external directory
	// (D-14, resolveBinaryDetailed) — carried onto ProbeSources'
	// SourceHealth.LaunchAdvisory. Always false for a TierExternal
	// instance (only a trusted copy can shadow, never the reverse).
	shadowed   bool
	trustedKey string    // the operator key id that vouched (TierOperatorTrusted only, davison/topos#49)
	offeredKey *KeyOffer // an unknown self-describing key's offer (TierExternal only)
	// extras mirrors this instance's Describe-declared
	// DescribeResponse.extras (D-15), filtered to drop any entry with an
	// empty key (filterExtras) — a plugin must not be able to inject a
	// nameless field into the operator's add-source form.
	extras []*toposv1.ExtrasField
}

// Name returns the config key this plugin was launched under (under
// [sources.<name>]) — the source instance's identity (D-08).
func (p *Plugin) Name() string { return p.name }

// SourceType returns the source_type learned from the plugin's own
// Describe RPC response — never trusted from the filename (T-01-07). This
// is the plugin KIND, never an identity key: two launched instances of one
// plugin binary share the same SourceType but always have distinct Name().
func (p *Plugin) SourceType() string { return p.sourceType }

// DisplayName returns this source INSTANCE's resolved display name (D-09):
// its config-authored [sources.<name>] display_name, or the instance id
// itself when that key is omitted. This is what the health/sources API and
// the UI show for this specific instance — distinct from PluginDisplayName,
// which names the plugin KIND regardless of how many instances of it are
// configured.
func (p *Plugin) DisplayName() string { return p.displayName }

// Tier returns this instance's launch-time provenance Tier — TierTrusted
// or TierExternal, set once at launch by ResolveBinary and never
// re-derived from anything the plugin process reports (T-11-01).
func (p *Plugin) Tier() Tier { return p.tier }

// TrustedKey returns the operator key id that vouched for this instance's
// binary — non-empty only for TierOperatorTrusted.
func (p *Plugin) TrustedKey() string { return p.trustedKey }

// OfferedKey returns the unknown self-describing key's offer carried by
// this external instance's binary, or nil.
func (p *Plugin) OfferedKey() *KeyOffer { return p.offeredKey }

// trustKeysChanged reports whether the operator's accepted keys in raw
// would change this running instance's tier, so Reconcile relaunches it
// under the new keys (M2-R4, davison/topos#49): a key withdrawn from
// [[plugins.trusted_keys]] demotes its plugins at the apply that removed
// it, not at some later launch; a key trusted while its plugin runs
// external (pinned) promotes it at that apply. Cheap and filesystem-free:
// it compares the instance's launch-time facts (the key that vouched, or
// the key that was offered) with the table; the relaunch's own
// EvaluateTrust makes the actual decision.
func trustKeysChanged(p *Plugin, raw *config.Config) bool {
	var keys []config.TrustedKey
	if raw != nil {
		keys = raw.Plugins.TrustedKeys
	}
	switch {
	case p.tier == TierOperatorTrusted:
		for _, k := range keys {
			if k.ID == p.trustedKey {
				return false
			}
		}
		return true // the vouching key is gone
	case p.tier == TierExternal && p.offeredKey != nil:
		for _, k := range keys {
			if k.ID == p.offeredKey.KeyID && k.PublicKey == base64.StdEncoding.EncodeToString(p.offeredKey.PublicKey) {
				return true // the offered key is now trusted
			}
		}
		return false
	default:
		return false
	}
}

// operatorKeyID is the key id to report for a trust result: only the
// operator's word is named on the wire (the kernel author's is implicit
// in TierTrusted).
func operatorKeyID(t Trust) string {
	if t.Word == KeyWordOperator {
		return t.KeyID
	}
	return ""
}

// BinaryHash returns this instance's on-disk SHA-256, computed at launch
// time for TierExternal only — empty for TierTrusted (11-03-PLAN.md Task 2).
func (p *Plugin) BinaryHash() string { return p.binaryHash }

// Extras returns this instance's Describe-declared extras field
// declarations (D-15), already filtered to drop any entry with an empty
// key.
func (p *Plugin) Extras() []*toposv1.ExtrasField { return p.extras }

// PluginDisplayName returns the Describe-learned display name for the
// plugin KIND this instance runs (e.g. "paperless-ngx", "SilverBullet") —
// the same value every instance of this plugin binary reports, regardless
// of its own instance display name. Provenance/diagnostic use only; never
// a substitute for DisplayName in anything the UI renders as a source
// label (02-01-PLAN.md's sourceDisplayName fix is the local-mapping
// predecessor this makes obsolete).
func (p *Plugin) PluginDisplayName() string { return p.pluginName }

// MatchVocabulary returns the field-name vocabulary this plugin declared in
// its Describe response (DescribeResponse.match_vocabulary) — the set of
// keys correlate.matchFieldsFor may populate in a Match request sent to
// this instance. Satisfies correlate.Source.
func (p *Plugin) MatchVocabulary() []string { return p.matchVocabulary }

// Match calls the plugin's Match RPC, wrapping each declared field's value
// list in a StringList (proto3 map values cannot be repeated fields
// directly). Satisfies correlate.Source.
func (p *Plugin) Match(ctx context.Context, fields map[string][]string) (*toposv1.MatchResponse, error) {
	matchFields := make(map[string]*toposv1.StringList, len(fields))
	for k, v := range fields {
		matchFields[k] = &toposv1.StringList{Values: v}
	}
	return p.impl.Match(ctx, &toposv1.MatchRequest{MatchFields: matchFields})
}

// Fetch calls the plugin's Fetch RPC.
func (p *Plugin) Fetch(ctx context.Context, sourceID string, variant toposv1.ContentVariant) (*toposv1.FetchResponse, error) {
	return p.impl.Fetch(ctx, &toposv1.FetchRequest{SourceId: sourceID, Variant: variant})
}

// Health calls the plugin's Health RPC.
func (p *Plugin) Health(ctx context.Context) (*toposv1.HealthResponse, error) {
	return p.impl.Health(ctx, &toposv1.HealthRequest{})
}

// Kill terminates the plugin subprocess.
func (p *Plugin) Kill() {
	p.client.Kill()
}

// Icon returns this plugin instance's validated icon bytes and mime,
// captured from its own Describe response (09-01-PLAN.md Task 2). ok is
// false when the plugin declared no icon, declared one that failed
// captureIcon's validation, or was built against the pre-Phase-9 contract.
func (p *Plugin) Icon() ([]byte, string, bool) {
	if len(p.iconBytes) == 0 || p.iconMIME == "" {
		return nil, "", false
	}
	return p.iconBytes, p.iconMIME, true
}

// Host owns the lifecycle of every launched plugin subprocess.
type Host struct {
	// dirs is the two configured plugin directories Discover launched
	// every plugin binary from (Phase 11, widening the pre-existing
	// single pluginsDir field) — retained (07-02-PLAN.md Task 1) so
	// Reconcile can launch a replacement/added instance later without a
	// caller having to thread the directories through a second time. Set
	// once at construction and never written again, so it needs no lock.
	dirs Dirs

	// mu guards ONLY the plugins field below (08-13-PLAN.md Task 1(C),
	// closing G-08-5's second, latent defect: kernel/supervisor.Supervisor
	// already called Host.Fetch with no lock held, concurrently with a
	// Reconcile writing this field in place — a genuine data race at HEAD,
	// merely narrow enough that no test provoked it). This lock makes
	// concurrent READERS (Plugins/ProbeSources/Fetch/byInstance, via
	// snapshot below) safe against a single in-flight Reconcile; two
	// concurrent Reconcile calls remain excluded by kernel/supervisor's own
	// mutation lock (Supervisor.mu), which every caller of Reconcile in
	// this repo already holds for the call's whole duration.
	mu      sync.RWMutex
	plugins []*Plugin
	// launchFailures holds one entry per instance CURRENTLY refused at
	// launch by a soft failure class (today: pin mismatch only) — guarded
	// by mu alongside plugins, since both are written together at the same
	// commit points (Discover's construction, Reconcile's commit region).
	// Rebuilt wholesale on every Reconcile call (never merged), which is
	// what clears a stale entry the instant its pin is repaired or the
	// instance is removed from config — see Reconcile's own doc comment.
	launchFailures map[string]LaunchFailure
	// raw is the latest raw (unexpanded) *config.Config Discover/Reconcile
	// were called with — stored under mu alongside plugins/launchFailures
	// because both pins ([plugins.pins]) and Task 2's env-reference
	// allowlist are read from the RAW document, never the expanded one
	// (D-05's secret-value discipline: raw never holds a resolved secret
	// VALUE, only the ${VAR} reference).
	raw *config.Config
}

// snapshot returns a defensive copy of the currently launched plugin set
// under mu.RLock() — the one place the read lock is taken for the plugin
// set, so every read path below shares it rather than reimplementing its
// own lock/copy. A copy, not the live backing array, so a caller ranging
// over the result after the lock is released can never observe a
// concurrent Reconcile's commit mid-iteration.
func (h *Host) snapshot() []*Plugin {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]*Plugin, len(h.plugins))
	copy(out, h.plugins)
	return out
}

// Discover launches one subprocess per configured source, resolving each
// binary through dirs (Phase 11's two-tier ResolveBinary), and returns the
// resulting Host. Every launched plugin's Describe RPC is called
// immediately to learn its source_type — the kernel never trusts a
// plugin's identity from its filename (T-01-07).
//
// raw is the RAW (unexpanded) config sources was derived from — Phase 11's
// pin map ([plugins.pins]) and env-reference allowlist (Task 2) both come
// from this document, never the expanded one. raw may be nil (no
// call site outside this package's own tests should pass nil for a real
// boot, but a nil raw degrades gracefully: no pins are ever known, so every
// external-tier launch behaves as if unpinned — a mismatch, per D-01's
// empty-pin-is-a-mismatch rule).
//
// Sources are launched in SORTED name order (deterministic run to run,
// matching this package's established convention — see matchconfig.go) —
// pure ordering, not a correctness requirement, since every source is
// launched independently either way.
//
// EVERY launch-failure class is a SOFT, per-instance failure
// (M1-R6/DIST-03, davison/topos#17, widening 11-RESEARCH.md Pitfall 1's
// pin-mismatch precedent to the whole family): a pin mismatch, a trust
// refusal, a missing binary, a handshake-version refusal, a
// contract-generation refusal, or a plugin that never became a plugin
// at all is recorded in the returned Host's LaunchFailures(), logged by
// name, and Discover CONTINUES to the next source — the failed instance
// is absent from Host.Plugins() while every OTHER configured source
// still boots normally. One stale binary must never cost the operator a
// dead kernel. The hard-abort path (kill what this call launched,
// return the named error) remains for exactly two things: a
// cancellation of ctx itself, and machinery errors that are not
// per-instance launch outcomes (config marshalling, hashing I/O).
func Discover(ctx context.Context, dirs Dirs, raw *config.Config, sources map[string]config.Source, logger hclog.Logger) (*Host, error) {
	h := &Host{dirs: dirs, raw: raw, launchFailures: make(map[string]LaunchFailure)}

	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		src := sources[name]
		p, err := launch(ctx, dirs, name, src, raw, logger, false)
		if err != nil {
			var pmErr *pinMismatchError
			if errors.As(err, &pmErr) {
				h.launchFailures[name] = pmErr.toLaunchFailure()
				logger.Error("plugin launch refused: pinned binary hash mismatch (T-11-07)",
					"instance", name, "plugin", src.Plugin,
					"pinned_hash", pmErr.pinnedHash, "current_hash", pmErr.currentHash)
				continue
			}
			var muErr *manifestUnverifiedError
			if errors.As(err, &muErr) {
				h.launchFailures[name] = muErr.toLaunchFailure()
				logger.Error("plugin launch refused: trusted binary not verified by the build manifest (D-12/D-13)",
					"instance", name, "plugin", src.Plugin, "current_hash", muErr.currentHash)
				continue
			}
			var lErr *launchError
			if errors.As(err, &lErr) && ctx.Err() == nil {
				// M1-R6/DIST-03: every never-became-a-plugin class is a
				// per-instance record — the boot proceeds and the failure
				// is on the source list by name. The ctx.Err() guard keeps
				// a shutdown/cancellation mid-boot an abort, not a page of
				// bogus per-source "failures" that are really one
				// cancellation.
				h.launchFailures[name] = lErr.toLaunchFailure()
				logger.Error("plugin launch failed: recorded as a named per-instance launch failure (M1-R6/DIST-03)",
					"instance", name, "plugin", src.Plugin, "reason", lErr.reason, "error", lErr.message)
				continue
			}
			h.Shutdown()
			return nil, fmt.Errorf("pluginhost: launch source %q: %w", name, err)
		}
		h.plugins = append(h.plugins, p)
	}

	return h, nil
}

// Reconcile brings the launched plugin set in line with sources, in place
// (07-02-PLAN.md Task 1, D-06/D-07's hot-apply): for every configured
// instance not currently launched, or currently launched with a different
// config.Source than sources now declares, it launches a replacement via
// the same unexported launch() Discover uses and kills the instance it
// replaces; every currently launched instance absent from sources is
// killed and dropped; every other instance's subprocess is left running
// completely untouched, so a save that edits one source's config never
// blips an unrelated source's reachability.
//
// Every new/changed launch is attempted before anything currently running
// is torn down, and the previously running set is left fully intact by a
// failure either way — per instance for the recorded launch-failure
// classes (the paragraph below), or wholesale for the ctx-cancellation/
// machinery abort, which kills only what THIS call itself launched and
// returns an error naming the offending instance (T-07-11's surviving
// core). Instance names are iterated in sorted order so multi-failure
// output is deterministic run to run, matching this package's existing
// discipline (matchconfig.go).
//
// Locking (08-13-PLAN.md Task 1(C), G-08-5): this call takes h.mu.RLock()
// only to build existing from a snapshot of the CURRENT plugin set, then
// releases it and performs the whole toLaunch launch loop with NO LOCK
// HELD, then takes h.mu.Lock() for the kill-and-commit region only (the
// existing kill loop plus the h.plugins = next assignment). Holding the
// write lock across the launches is exactly the defect this fix closes —
// a plugin subprocess slow to complete its handshake used to freeze every
// concurrent reader (Plugins/ProbeSources/Fetch/byInstance) kernel-wide,
// contradicting phase success criterion 4's "every other source is
// unaffected" — so it must never be done here again.
//
// raw is the RAW (unexpanded) config sources was derived from — see
// Discover's doc comment for why this is needed (pins, and Task 2's env
// allowlist) and why a nil raw degrades gracefully rather than panicking.
//
// EVERY launch-failure class is a SOFT, per-instance failure exactly
// like Discover's (M1-R6/DIST-03, davison/topos#17): a pin mismatch, a
// trust refusal, a missing binary, a handshake-version refusal, a
// contract-generation refusal, or a generic never-became-a-plugin
// error is recorded, logged by name, and this call does NOT abort —
// every other new/changed instance still launches, every unchanged
// instance is left running untouched, and Reconcile still returns nil,
// so an apply whose only casualties are per-instance launch failures
// COMMITS with those failures surfaced on the source list. T-07-11's
// hard-fail (kill only what THIS call launched, return the named
// error, previously running set fully intact) remains for a ctx
// cancellation and for machinery errors that are not per-instance
// launch outcomes.
//
// Host.launchFailures is rebuilt WHOLESALE from this call's own toLaunch
// loop (never merged with the previous round) — an instance no longer in
// sources at all, or one that now launches successfully because its pin
// was repaired, is simply absent from the new map, which is what clears a
// stale "binary changed" state the instant the operator fixes it. An
// instance still pin-mismatched is never "kept" (it never has a *Plugin in
// h.plugins to keep — see the kept/toLaunch split below), so it is always
// retried on every Reconcile call, exactly the retry a repaired pin needs.
func (h *Host) Reconcile(ctx context.Context, raw *config.Config, sources map[string]config.Source, logger hclog.Logger) error {
	h.mu.RLock()
	existing := make(map[string]*Plugin, len(h.plugins))
	for _, p := range h.plugins {
		existing[p.name] = p
	}
	h.mu.RUnlock()

	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)

	kept := make(map[string]*Plugin, len(names))
	var toLaunch []string
	for _, name := range names {
		p, ok := existing[name]
		if ok && reflect.DeepEqual(p.src, sources[name]) && !trustKeysChanged(p, raw) {
			kept[name] = p
			continue
		}
		toLaunch = append(toLaunch, name)
	}

	launched := make(map[string]*Plugin, len(toLaunch))
	newFailures := make(map[string]LaunchFailure, len(toLaunch))
	for _, name := range toLaunch {
		p, err := launch(ctx, h.dirs, name, sources[name], raw, logger, false)
		if err != nil {
			var pmErr *pinMismatchError
			if errors.As(err, &pmErr) {
				newFailures[name] = pmErr.toLaunchFailure()
				logger.Error("plugin launch refused: pinned binary hash mismatch (T-11-07)",
					"instance", name, "plugin", sources[name].Plugin,
					"pinned_hash", pmErr.pinnedHash, "current_hash", pmErr.currentHash)
				continue
			}
			var muErr *manifestUnverifiedError
			if errors.As(err, &muErr) {
				newFailures[name] = muErr.toLaunchFailure()
				logger.Error("plugin launch refused: trusted binary not verified by the build manifest (D-12/D-13)",
					"instance", name, "plugin", sources[name].Plugin, "current_hash", muErr.currentHash)
				continue
			}
			var lErr *launchError
			if errors.As(err, &lErr) && ctx.Err() == nil {
				// M1-R6/DIST-03: mirror of Discover's own branch — an
				// apply whose only casualties are per-instance launch
				// failures commits with those failures on the source
				// list, instead of a whole-save 500 with no named
				// culprit. The ctx.Err() guard keeps a cancellation an
				// abort, exactly as in Discover.
				newFailures[name] = lErr.toLaunchFailure()
				logger.Error("plugin launch failed: recorded as a named per-instance launch failure (M1-R6/DIST-03)",
					"instance", name, "plugin", sources[name].Plugin, "reason", lErr.reason, "error", lErr.message)
				continue
			}
			for _, lp := range launched {
				lp.Kill()
			}
			return fmt.Errorf("pluginhost: reconcile: launch source %q: %w", name, err)
		}
		launched[name] = p
	}

	// Every launch this call needed has now succeeded — only now take the
	// write lock, kill the instances being replaced or removed, and commit
	// the new plugin set.
	h.mu.Lock()
	defer h.mu.Unlock()

	for name, p := range existing {
		if _, stillKept := kept[name]; stillKept {
			continue
		}
		p.Kill()
	}

	next := make([]*Plugin, 0, len(kept)+len(launched))
	for _, name := range names {
		if p, ok := kept[name]; ok {
			next = append(next, p)
			continue
		}
		if p, ok := launched[name]; ok {
			next = append(next, p)
		}
	}
	h.plugins = next
	h.raw = raw
	h.launchFailures = newFailures
	return nil
}

// LaunchFailures returns a defensive copy of every instance CURRENTLY
// refused or failed at launch (every class — see LaunchFailure's doc
// comment), sorted by Instance for deterministic output.
func (h *Host) LaunchFailures() []LaunchFailure {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]LaunchFailure, 0, len(h.launchFailures))
	for _, lf := range h.launchFailures {
		out = append(out, lf)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Instance < out[j].Instance })
	return out
}

// stderrTailCap bounds stderrTail's retained byte count. Chosen in the low
// kilobytes: comfortably larger than a single fatal line or the last few
// frames of a panic stack trace (the shapes this capture exists to
// surface — see the signal plugin's fatal helper, in
// davison/topos-plugins, for the worked example), while staying irrelevant to kernel memory even if every
// launched plugin instance held its own tail for its whole lifetime.
const stderrTailCap = 4096

// stderrTail is a bounded, mutex-guarded tail of a plugin subprocess's
// stderr output, wired into goplugin.ClientConfig.Stderr in launch below.
//
// Why it exists: go-plugin defaults ClientConfig.Stderr to io.Discard, so a
// plugin that calls fatal (writes one line, exits) before ever reaching
// goplugin.Serve writes its only explanation into nothing — the kernel log
// records only "exit status 1", and DescribePluginType's caller sees only
// go-plugin's own four-item handshake diagnostic list, every item of which
// was false in the case this type was written to fix (07-13-PLAN.md,
// closing 07-UAT.md G-07-5: a blank Signal path field). This is a
// recurrence guard for the whole pre-handshake-fatal class, not a fix
// scoped to one field — it fires for any plugin, any config field, any
// future pre-Serve fatal, on both trial launches and boot-time launches,
// since both go through this same launch function.
//
// It is bounded because a plugin looping on stderr before eventually
// exiting must never grow kernel memory without limit — a memory-safety
// requirement, not a cosmetic one. It is mutex-guarded because go-plugin's
// own logStderr goroutine (client.go) writes to this type concurrently
// with the kernel's own goroutine that will eventually read it; an
// unsynchronized read/write pair here would be a genuine data race, not a
// theoretical one. It discards from the FRONT once the cap is exceeded
// (never truncates the incoming write, never stops accepting bytes)
// because the explanatory line is the LAST thing a dying plugin writes —
// stopping at the cap would keep the least useful bytes and discard the
// one line this whole type exists to preserve.
type stderrTail struct {
	mu  sync.Mutex
	buf []byte
}

// Write satisfies io.Writer for goplugin.ClientConfig.Stderr. It always
// reports the full input length written and a nil error — go-plugin's own
// stderr-draining loop (logStderr) does nothing special with a short-write
// report beyond logging it, but this type must never be the reason that
// loop believes a write failed; the tail's own bounding happens entirely
// inside this call, invisibly to the caller.
func (t *stderrTail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > stderrTailCap {
		t.buf = t.buf[len(t.buf)-stderrTailCap:]
	}
	return len(p), nil
}

// lastLine returns the final non-empty line currently held, with
// surrounding whitespace (including any trailing newline(s)) trimmed, or
// an empty string when nothing has been written. Safe to call concurrently
// with Write, but the caller in launch below only ever calls it AFTER
// client.Kill() has returned — Kill waits on go-plugin's own client wait
// group, which the stderr-draining goroutine belongs to, so calling this
// any earlier would race that goroutine and could miss the very line this
// type exists to surface.
func (t *stderrTail) lastLine() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	trimmed := strings.TrimRight(string(t.buf), "\r\n")
	if trimmed == "" {
		return ""
	}
	if idx := strings.LastIndexByte(trimmed, '\n'); idx >= 0 {
		trimmed = trimmed[idx+1:]
	}
	return strings.TrimSpace(trimmed)
}

// sourceConfigEnvelope is the WEBSPACES_SOURCE_CONFIG JSON document every
// launched plugin subprocess receives — a NAMED struct (Phase 11 Task 3),
// replacing the pre-Phase-11 flat map[string]string literal so a nested
// sub-table (Extras) can be represented at all: a flat map can only ever
// hold string values, never a nested object. Every pre-existing key's JSON
// tag is byte-identical to what the flat map produced, so every already-
// shipped plugin binary keeps reading the same top-level document
// unchanged — this type is purely additive.
//
// Flattening Extras into this envelope's own top-level key space is
// PROHIBITED (D-12): the sub-table boundary — kernel-known fields stay
// strictly typed here, provider-specific keys stay opaque inside their own
// nested object — is the entire point of D-12. A plugin reads its own
// provider keys from config["extras"]["key"], never from a top-level key
// the kernel itself could someday also want to use.
type sourceConfigEnvelope struct {
	BaseURL        string `json:"base_url"`
	Token          string `json:"token"`
	APIVersion     string `json:"api_version"`
	CACert         string `json:"ca_cert"`
	Username       string `json:"username"`
	WebmailBaseURL string `json:"webmail_base_url"`
	Path           string `json:"path"`
	// Recursive carries config.Source.Recursive verbatim (12-03-PLAN.md
	// Task 1) — a kernel-known typed field, always emitted like every
	// other scalar above (never omitempty: false is a meaningful, present
	// value distinct from "field absent", unlike Extras' legitimate
	// absent state below), and never folded into the nested Extras object
	// — the sub-table boundary (D-12) is between kernel-known typed
	// fields and opaque provider keys, and this is a kernel-known field.
	Recursive bool `json:"recursive"`
	// Extras carries config.Source.Extras verbatim (D-12/D-13) — omitted
	// entirely (no "extras" key at all, never an empty object) when this
	// source declares no extras, so a plugin's own JSON decode sees exactly
	// "no extras configured" rather than an ambiguous empty-vs-absent case.
	Extras map[string]string `json:"extras,omitempty"`
}

// allowedDesktopSessionEnvVars is the fixed, documented allowlist copied
// into every plugin subprocess's environment (D-14), regardless of which
// ${VAR} references that instance's own config happens to declare:
//
//   - PATH, HOME — needed by any subprocess at all, and by "~" expansion
//     (the signal plugin resolves its own Path field's leading "~" itself,
//     not the kernel — see kernel/config/types.go's Source.Path comment).
//   - LANG, LC_ALL, LC_CTYPE, TZ, TMPDIR — the locale/timezone/scratch-
//     space group, so a plugin's date and text handling stays identical to
//     the kernel's own, rather than silently falling back to a C-locale
//     default inside the subprocess.
//   - XDG_RUNTIME_DIR, DBUS_SESSION_BUS_ADDRESS — desktop-session plumbing
//     the Signal plugin's Secret Service key retrieval requires
//     (its secretservice.go D-Bus call, in topos-plugins, to unwrap Signal
//     Desktop's SQLCipher key). These are SESSION ADDRESSES, not secret
//     values — withholding them would not meaningfully contain a plugin
//     that can already read the session bus socket path directly off the
//     filesystem; this phase implements honest disclosure of what a
//     plugin subprocess can see (D-14's whole point), not sandboxing
//     (11-RESEARCH.md's explicit no-containment framing, unchanged here).
var allowedDesktopSessionEnvVars = []string{
	"PATH", "HOME", "LANG", "LC_ALL", "LC_CTYPE", "TZ", "TMPDIR",
	"XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS",
}

// allowedEnv builds the COMPLETE environment for a launched plugin
// subprocess (D-14) — the sole producer of exec.Cmd.Env in launch below,
// which never reads the kernel process's own environment wholesale
// (os.Environ()) again. Three, and only three, sources contribute:
//
//  1. allowedDesktopSessionEnvVars above, copied present-only (an unset
//     allowlisted variable contributes nothing — never an empty-string
//     entry).
//  2. Every name in config.EnvRefNames(rawSrc) that is CURRENTLY SET in the
//     kernel process — the values behind the ${VAR} references THIS
//     instance's own raw config declares, and nothing else. A variable set
//     on the kernel process but referenced nowhere in rawSrc is never
//     copied, no matter what it's named — this is the T-11-08 mitigation:
//     the kernel's remaining environment (any credential, any unrelated
//     secret sitting in the operator's shell) is structurally invisible to
//     a plugin subprocess that never referenced it.
//  3. The WEBSPACES_SOURCE_CONFIG envelope (always) and
//     WEBSPACES_DESCRIBE_ONLY=1 (trial launches only) — unchanged from
//     before this phase.
func allowedEnv(rawSrc config.Source, sourceConfigJSON []byte, describeOnly bool) []string {
	env := make([]string, 0, len(allowedDesktopSessionEnvVars)+4)

	for _, name := range allowedDesktopSessionEnvVars {
		if v, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+v)
		}
	}

	for _, name := range config.EnvRefNames(rawSrc) {
		if v, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+v)
		}
	}

	env = append(env, "WEBSPACES_SOURCE_CONFIG="+string(sourceConfigJSON))
	if describeOnly {
		env = append(env, "WEBSPACES_DESCRIBE_ONLY=1")
	}
	return env
}

// launch resolves src.Plugin against dirs via ResolveBinary (Phase 11's
// two-tier provenance authority), execs the resolved path, performs the
// go-plugin handshake, and calls Describe to learn the launched instance's
// identity. Every caller —
// Discover, Reconcile, and DescribePluginType's trial launch — goes
// through this one function, so the bounded stderr capture wired in below
// (stderrTail) covers a pre-handshake plugin fatal identically whether the
// launch is a UI-driven trial or a real boot-time/hot-apply launch: the
// class of failure this guards against has nothing to do with which caller
// triggered it.
//
// describeOnly (CR-01, 08-REVIEW.md), when true, sets the
// WEBSPACES_DESCRIBE_ONLY=1 environment variable alongside the usual
// WEBSPACES_SOURCE_CONFIG one — set only by DescribePluginType's trial
// launch, never by Discover/Reconcile's real boot-time/hot-apply launches.
// A launched plugin binary that recognises this variable (currently only
// the whatsapp plugin, in topos-plugins) may use it to skip acquiring any exclusive,
// process-lifetime resource (e.g. WhatsApp's storelock.go flock) it would
// otherwise hold for as long as this trial-launched subprocess is alive,
// since Describe's answer never depends on that resource. A plugin binary
// that does not recognise the variable simply ignores it and launches
// exactly as before — this is additive, never a behavior change for any
// plugin type that hasn't opted in.
//
// raw is the RAW (unexpanded) config Discover/Reconcile were called with —
// nil for DescribePluginType's trial-launch path, where the SUBMITTED src
// itself already IS the raw form (nothing loaded it from disk). launch
// resolves rawSrc, this instance's own raw config.Source, as raw.Sources[name]
// when raw is non-nil and the name is present there, falling back to src
// otherwise — this is what lets a real boot-time launch see the operator's
// literal ${VAR} references (Task 2's env allowlist) while a trial launch
// still works with no raw document to draw from at all.
//
// Pin verification (T-11-07, D-01/D-02/D-03/D-04): immediately after
// ResolveBinary resolves an EXTERNAL-tier binary, and BEFORE exec.Command is
// ever constructed, launch recomputes the binary's SHA-256 (HashBinary) and
// compares it to raw.Plugins.Pins[src.Plugin]. A mismatch — including "no
// pin recorded at all", which compares equal to the zero-value empty
// string and therefore never matches a real 64-character digest — returns
// a *pinMismatchError (wrapping ErrPinMismatch) and creates NO subprocess
// at all. TierTrusted skips this entire block (D-04: a trusted-dir binary
// is rebuilt constantly by `make build`/`make dev`, so pinning it would
// false-alarm on every rebuild). describeOnly launches ALSO skip this
// block deliberately (T-11-14, accepted risk, 11-PLAN.md threat register):
// the add-source flow's trial launch is the ONLY way to learn an external
// binary's identity (and therefore compute the hash an operator can choose
// to pin) BEFORE any pin can exist for it — gating the trial launch on a
// pin that cannot yet exist would make it structurally impossible to ever
// add a first external source.
//
// Trust verification (13-05-PLAN.md Task 3, D-12/D-13; widened by
// 16-01-PLAN.md Task 1 to the two-arm coexistence rule, D-10; collapsed by
// 16-02-PLAN.md Task 1, D-11, into resolveBinaryDetailed itself): tier is
// no longer a directory-derived fact launch re-checks — resolveBinaryDetailed
// ALREADY evaluated the resolved binary's provenance via EvaluateTrust
// (provenance.go), the single authority consulting BOTH the link-time
// build manifest AND every validly-signed release manifest in dirs;
// neither arm can silently substitute for the other, and EITHER arm
// succeeding grants trusted-tier eligibility, wherever the binary sits
// (D-11: directories are pure search paths). launch gates directly on the
// Trust value the resolver already produced. A binary verified by neither
// arm, or refused as tampered by either, returns a *manifestUnverifiedError
// (wrapping ErrManifestUnverified) and creates NO subprocess at all —
// UNLIKE the pin-mismatch block below, this gate runs for describeOnly
// (trial) launches TOO: the external-tier pin check skips trial launches
// because a first pin cannot exist before the trial ever runs, but a
// trusted binary either verifies or it doesn't — that fact doesn't depend
// on whether this is a real or a trial launch, and letting the add-source
// picker's trial launch execute an unverified dropped binary would hand an
// attacker code execution through the describe path (T-13-06).
// Verification never demotes-and-runs: an unverifiable binary's only path
// to running remains the existing explicit external-tier consent and pin
// flow (TRUST-03) — trust is evaluated exactly once per launch, by
// resolveBinaryDetailed, and still before exec.Command is constructed.
func launch(ctx context.Context, dirs Dirs, name string, src config.Source, raw *config.Config, logger hclog.Logger, describeOnly bool) (*Plugin, error) {
	if raw != nil {
		// The operator's accepted keys come from the same raw config the
		// pins do, installed before this launch evaluates trust — the
		// kernel hands pluginhost the operator's keys; pluginhost never
		// reads config itself (D-12 revised, davison/topos#49).
		SetOperatorProvenanceKeys(OperatorProvenanceKeysFromConfig(raw.Plugins.TrustedKeys))
	}
	binPath, trust, shadowed, resolveErr := resolveBinaryDetailed(dirs, src.Plugin, logger)
	if resolveErr != nil && binPath == "" {
		// binPath is empty ONLY when the binary was not found in either
		// directory at all (or its name failed validatePluginBinaryName) —
		// never when resolveBinaryDetailed found bytes but EvaluateTrust
		// refused them (that case is handled below, once instanceDisplayName
		// is available, as a *manifestUnverifiedError, not a "not found").
		// A *launchError, not a bare error (M1-R6/DIST-03): a fleet
		// updated independently of the kernel can legitimately stop
		// shipping a binary the config still names, and that must show as
		// a named per-source failure, never a dead boot. Tier is the
		// empty Tier — there are no bytes to derive one from.
		displayName := src.DisplayName
		if displayName == "" {
			displayName = name
		}
		return nil, &launchError{
			instance:    name,
			plugin:      src.Plugin,
			displayName: displayName,
			reason:      LaunchFailureLaunchFailed,
			message:     fmt.Sprintf("pluginhost: instance %q plugin binary %q not found in the trusted or external plugin directory: %v", name, src.Plugin, resolveErr),
			cause:       resolveErr,
		}
	}
	tier := trust.Tier

	// The instance display name resolves from the operator's own config
	// (D-09), falling back to the instance id itself when display_name is
	// omitted — never from anything the plugin process asserts. Resolved
	// here (before the pin check below, and well before Describe returns)
	// so a pin-mismatch LaunchFailure record can carry it too — mirrors
	// config.Config.DisplayNameFor without needing the whole *config.Config
	// here, since launch already holds this instance's own config.Source.
	instanceDisplayName := src.DisplayName
	if instanceDisplayName == "" {
		instanceDisplayName = name
	}

	// manifestHash carries the trusted-tier binary's verified on-disk
	// SHA-256 out of this block into the returned *Plugin — see the
	// Plugin.manifestHash field's own doc comment. D-11 (16-02-PLAN.md
	// Task 1): resolveBinaryDetailed already evaluated trust exactly once
	// above (and already logged every Trust.Diagnostics entry) — launch
	// gates directly on that result instead of re-evaluating.
	//
	// A non-nil resolveErr at this point means resolveBinaryDetailed found
	// bytes at binPath but EvaluateTrust refused them (a tamper refusal —
	// binPath being non-empty already ruled out the "not found" case
	// above). This is treated identically to "trust was expected but the
	// binary didn't earn it": *manifestUnverifiedError, no subprocess ever
	// created (D-13).
	var manifestHash string
	if resolveErr != nil {
		return nil, &manifestUnverifiedError{
			instance:    name,
			plugin:      src.Plugin,
			displayName: instanceDisplayName,
			tier:        tier,
			currentHash: trust.Hash,
			cause:       resolveErr,
		}
	}
	if tier.Vouched() {
		manifestHash = trust.Hash
	}

	// rawSrc is this instance's own RAW (unexpanded) config.Source — the
	// operator's literal ${VAR} references, never a resolved secret value
	// (D-05) — read from raw.Sources[name] when raw is non-nil and the name
	// is present there, falling back to src itself for DescribePluginType's
	// trial-launch path (raw is nil there: the SUBMITTED src already IS the
	// raw form, since nothing loaded it from disk). allowedEnv below is the
	// only consumer — it decides which ${VAR} values THIS instance may see.
	rawSrc := src
	if raw != nil {
		if rs, ok := raw.Sources[name]; ok {
			rawSrc = rs
		}
	}

	// binaryHash and launchPinnedHash carry TierExternal launch-time hash
	// facts out of this block into the returned *Plugin (11-03-PLAN.md
	// Tasks 1/2) — declared at this scope, rather than inside the
	// if-block, so the values survive past it. binaryHash is this
	// instance's on-disk SHA-256, computed for EVERY external-tier launch
	// including a describeOnly trial launch (DescribePluginType needs it
	// to report binary_hash before any pin can exist for a not-yet-added
	// source); launchPinnedHash is only ever set once a REAL (non-trial)
	// launch's pin actually matched.
	var binaryHash, launchPinnedHash string
	if tier == TierExternal {
		var pins map[string]string
		if raw != nil {
			pins = raw.Plugins.Pins
		}
		currentHash, pinnedHash, hashErr := hashAndLookupPin(binPath, src.Plugin, pins)
		if hashErr != nil {
			return nil, fmt.Errorf("pluginhost: hash external plugin binary for instance %q: %w", name, hashErr)
		}
		binaryHash = currentHash

		if !describeOnly {
			if currentHash != pinnedHash {
				return nil, &pinMismatchError{
					instance:    name,
					plugin:      src.Plugin,
					displayName: instanceDisplayName,
					tier:        tier,
					pinnedHash:  pinnedHash,
					currentHash: currentHash,
					offeredKey:  trust.Offer,
				}
			}
			launchPinnedHash = pinnedHash
		}
	}

	sourceConfig, err := json.Marshal(sourceConfigEnvelope{
		BaseURL:        src.BaseURL,
		Token:          src.Token,
		APIVersion:     src.APIVersion,
		CACert:         src.CACert,
		Username:       src.Username,
		WebmailBaseURL: src.WebmailBaseURL,
		// Path: the local filesystem path source config field (Source.Path
		// — kernel/config/types.go), needed by a local-path source like
		// SRC-02's Signal plugin, which has no base_url/token at all.
		Path: src.Path,
		// Recursive (12-03-PLAN.md Task 1): src is the EXPANDED
		// config.Source, matching every other typed field here.
		Recursive: src.Recursive,
		// Extras (D-12/D-13, Phase 11 Task 3): src is the EXPANDED
		// config.Source (never rawSrc) — expandEnv runs over the raw TOML
		// text before decode (kernel/config/config.go's LoadRaw doc
		// comment), so a ${VAR} reference inside an extras value is
		// already resolved here exactly like base_url/token are, with no
		// separate expansion step needed in this package.
		Extras: src.Extras,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal source config: %w", err)
	}

	// The exec.Cmd's own Stderr field is deliberately left unset: go-plugin
	// builds its own stderr pipe from this *exec.Cmd internally, and
	// os/exec rejects a command whose stderr writer is already set when
	// the caller (go-plugin, in this case) tries to set it a second time.
	// The capture goes through ClientConfig.Stderr below instead, which
	// go-plugin's own logStderr goroutine writes to.
	cmd := exec.Command(binPath)
	cmd.Env = allowedEnv(rawSrc, sourceConfig, describeOnly)

	tail := &stderrTail{}

	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig:  sdk.Handshake,
		Plugins:          sdk.PluginMap,
		Cmd:              cmd,
		AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolGRPC},
		Logger:           logger.Named("plugin." + name),
		// SkipHostEnv is D-14's actual enforcement point, not merely
		// cmd.Env above: go-plugin's own Client() (client.go) UNCONDITIONALLY
		// appends the ENTIRE os.Environ() onto cmd.Env before exec'ing —
		// "if !c.config.SkipHostEnv { cmd.Env = append(cmd.Env,
		// os.Environ()...) }" — regardless of what cmd.Env already holds.
		// Without this field, allowedEnv's restricted allowlist above would
		// be silently widened right back out to the kernel's full
		// environment by go-plugin itself, one layer below where this
		// package's own code stops looking. Discovered live while writing
		// this phase's own supervisor tests (kernel/supervisor/
		// launchlatency_test.go): a variable NOT in cmd.Env still reached
		// the subprocess until this field was added.
		SkipHostEnv: true,
		// Raises the gRPC message-size ceiling to match sdk.GRPCServer on
		// the plugin side, so a unary Fetch response carrying a full
		// rendition doesn't hit the 4 MB default (D-Task1, 01-01).
		GRPCDialOptions: []grpc.DialOption{
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(sdk.MaxMessageSize),
				grpc.MaxCallSendMsgSize(sdk.MaxMessageSize),
			),
		},
		// Captures a bounded tail of the child's stderr (07-13-PLAN.md
		// Task 3, closing 07-UAT.md G-07-5's second cause) — without this,
		// go-plugin defaults to io.Discard and a plugin that fatals before
		// the handshake explains itself to nobody.
		Stderr: tail,
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		// The read must happen strictly after Kill() returns: Kill waits
		// on go-plugin's client wait group, which the stderr-draining
		// goroutine (logStderr) belongs to — reading any earlier races
		// that goroutine and can miss the very line this capture exists
		// to surface. When nothing was captured (a genuinely silent
		// pre-handshake exit, or a failure that isn't a pre-handshake
		// fatal at all), the error text is left byte-identical to before
		// this capture existed — no empty parenthetical, no dangling
		// separator.
		connectErr := fmt.Errorf("connect to plugin subprocess: %w", err)
		if last := tail.lastLine(); last != "" {
			connectErr = fmt.Errorf("connect to plugin subprocess: %w (plugin stderr: %s)", err, last)
		}
		// M1-R6/DIST-03: a handshake refusal (go-plugin's own error text
		// names both protocol versions) or any other pre-RPC failure is a
		// per-instance *launchError — connectFailureReason picks the
		// closed-vocabulary class; the message carries go-plugin's text
		// whole, so a version refusal names its versions on the wire.
		return nil, &launchError{
			instance:    name,
			plugin:      src.Plugin,
			displayName: instanceDisplayName,
			tier:        tier,
			reason:      connectFailureReason(err),
			message:     fmt.Sprintf("pluginhost: instance %q binary %q: %v", name, src.Plugin, connectErr),
			cause:       connectErr,
		}
	}

	// newLaunchFailed wraps a post-handshake failure (dispense, a wrong
	// interface, a Describe error) into the generic per-instance class —
	// same shape as the connect branch above (M1-R6/DIST-03).
	newLaunchFailed := func(cause error) *launchError {
		return &launchError{
			instance:    name,
			plugin:      src.Plugin,
			displayName: instanceDisplayName,
			tier:        tier,
			reason:      LaunchFailureLaunchFailed,
			message:     fmt.Sprintf("pluginhost: instance %q binary %q: %v", name, src.Plugin, cause),
			cause:       cause,
		}
	}

	dispensed, err := rpcClient.Dispense("source")
	if err != nil {
		client.Kill()
		return nil, newLaunchFailed(fmt.Errorf("dispense source plugin: %w", err))
	}

	impl, ok := dispensed.(sdk.SourcePlugin)
	if !ok {
		client.Kill()
		return nil, newLaunchFailed(fmt.Errorf("plugin %s does not implement sdk.SourcePlugin", name))
	}

	desc, err := impl.Describe(ctx, &toposv1.DescribeRequest{})
	if err != nil {
		client.Kill()
		return nil, newLaunchFailed(fmt.Errorf("describe plugin %s: %w", name, err))
	}

	// The contract-generation gate (M1-R6/DIST-03, davison/topos#17):
	// Describe's contract_version is the additive-compatibility signal
	// the contract has carried since Phase 5, consumed HERE for the
	// first time. A declared generation outside
	// sdk.SupportedContractVersions — or no declaration at all, which
	// names nothing and is never assumed compatible — kills the
	// subprocess before any further RPC and refuses by name, naming both
	// sides. This runs for describe-only trial launches too: an
	// incompatible plugin is refused at add time with the same named
	// message, not discovered later.
	if declared := desc.GetContractVersion(); !sdk.ContractSupported(declared) {
		client.Kill()
		declaredText := fmt.Sprintf("%q", declared)
		if declared == "" {
			declaredText = "none (the plugin declared no contract generation)"
		}
		return nil, &launchError{
			instance:    name,
			plugin:      src.Plugin,
			displayName: instanceDisplayName,
			tier:        tier,
			reason:      LaunchFailureContractIncompatible,
			message: fmt.Sprintf(
				"pluginhost: instance %q binary %q declares contract generation %s; this kernel supports %s — update the plugin fleet or the kernel so the generations agree",
				name, src.Plugin, declaredText, strings.Join(sdk.SupportedContractVersions, ", ")),
			cause: ErrContractIncompatible,
		}
	}

	iconBytes, iconMIME, _ := captureIcon(desc)

	return &Plugin{
		name:            name,
		sourceType:      desc.GetSourceType(),
		pluginName:      desc.GetDisplayName(),
		displayName:     instanceDisplayName,
		matchVocabulary: desc.GetMatchVocabulary(),
		iconBytes:       iconBytes,
		iconMIME:        iconMIME,
		src:             src,
		client:          client,
		impl:            impl,
		tier:            tier,
		pinnedHash:      launchPinnedHash,
		binaryHash:      binaryHash,
		manifestHash:    manifestHash,
		shadowed:        shadowed,
		trustedKey:      operatorKeyID(trust),
		offeredKey:      trust.Offer,
		extras:          filterExtras(desc.GetExtras()),
	}, nil
}

// filterExtras drops any declared extras field whose key is empty
// (11-03-PLAN.md Task 2) — a plugin must not be able to inject a nameless
// field into the operator's add-source form via a malformed Describe
// response.
func filterExtras(fields []*toposv1.ExtrasField) []*toposv1.ExtrasField {
	out := make([]*toposv1.ExtrasField, 0, len(fields))
	for _, f := range fields {
		if f.GetKey() == "" {
			continue
		}
		out = append(out, f)
	}
	return out
}

// Plugins returns every launched plugin — a defensive copy (via snapshot)
// taken under h.mu.RLock(), never the live backing array, so a caller
// ranging over the result cannot observe a concurrent Reconcile's commit
// mid-iteration.
func (h *Host) Plugins() []*Plugin {
	return h.snapshot()
}

// PluginIcon resolves a plugin BINARY's icon (src.Plugin, e.g.
// "topos-plugin-mock") — never an instance name — over every currently
// launched instance, returning the first one that has a validated icon
// (09-01-PLAN.md Task 2, 09-UI-SPEC.md Fix 10). Two instances of the same
// binary have byte-identical icons (both came from the same Describe
// implementation), so "first launched instance with an icon" is
// equivalent to "this binary's icon." ok is false when no launched
// instance of binary has one — an undescribed plugin type (never
// launched), or every launch attempt so far predates a successful
// Describe.
func (h *Host) PluginIcon(binary string) (iconBytes []byte, iconMIME string, ok bool) {
	for _, p := range h.snapshot() {
		if p.src.Plugin != binary {
			continue
		}
		if b, m, iok := p.Icon(); iok {
			return b, m, true
		}
	}
	return nil, "", false
}

// DescribeInfo is the three Describe-derived facts the "+" chip picker's
// two-step modal needs to build a match-vocabulary-driven form for a
// plugin type that has no configured instance yet (D-11 step 1 -> step 2,
// 07-02-PLAN.md Task 3).
type DescribeInfo struct {
	SourceType        string
	PluginDisplayName string
	MatchVocabulary   []string
	// Tier is the launch-time provenance Tier the trial-launched binary
	// resolved to (T-11-01) — set from ResolveBinary via the trial
	// launch's own *Plugin, never from anything the plugin's Describe
	// response itself asserts.
	Tier Tier
	// OfferedKey is the trial-launched binary's offer, if an unknown
	// self-describing key signed a manifest naming it (davison/topos#49)
	// — learned here BEFORE the source is added, so the add-source
	// interstitial can offer the key.
	OfferedKey *KeyOffer
	// BinaryHash is the SHA-256 of the trial-launched binary, computed by
	// launch() at the exact point ResolveBinary hands back TierExternal
	// (11-03-PLAN.md Task 2) — the fact the add-source flow's confirm
	// interstitial displays and, on save, the value written to
	// [plugins.pins]. Empty for TierTrusted (D-04: never pinned).
	BinaryHash string
	// Extras mirrors the plugin's declared DescribeResponse.extras (D-15),
	// in declaration order, with any entry carrying an empty key already
	// dropped (filterExtras) — a plugin must not be able to inject a
	// nameless field into the operator's add-source form.
	Extras []*toposv1.ExtrasField
}

// DescribePluginType trial-launches src (a config.Source naming the
// plugin binary to run, submitted but not yet persisted anywhere) under a
// fixed, non-persisted instance name, calls its Describe RPC, and kills
// the subprocess before returning — writing nothing to disk and
// registering nothing on any *Host (D-11 step 1 -> step 2). It reuses the
// existing unexported launch() verbatim (the same binary-stat, handshake,
// dispense and Describe-call sequence Discover already performs per
// configured instance) rather than duplicating any of it, and calls no
// RPC beyond the one Describe call launch() already makes — the trial-
// launch path must never become a general plugin-invocation surface for
// request-supplied input (PLUG-02, T-07-10); see
// kernel/httpapi/config_test.go's AST guard pinning exactly this over
// this function's own body.
//
// Every trial launch here passes describeOnly=true to launch() (CR-01,
// 08-REVIEW.md): this call site is ALWAYS a describe-only launch, never a
// real boot-time/hot-apply one, regardless of whether src names a plugin
// type with no configured instance yet, or an already-configured,
// already-running instance's own stored connection fields (the "+"
// picker's one-step existing-instance add flow, and every source chip's
// "Edit match settings…" entry, both reuse this same call — see
// kernel/httpapi/config.go's DescribePluginHandler doc comment). For most
// plugin types this changes nothing observable (they open-and-close per
// RPC call and hold no cross-instance exclusive resource); for WhatsApp —
// the one plugin in this repo that holds a persistent connection and an
// exclusive store lock for its entire process lifetime — this is what
// stops the trial-launch from always losing that lock race against a real
// running instance and failing before Describe is ever reached.
func DescribePluginType(ctx context.Context, dirs Dirs, src config.Source, logger hclog.Logger) (DescribeInfo, error) {
	p, err := launch(ctx, dirs, "__trial__", src, nil, logger, true)
	if err != nil {
		return DescribeInfo{}, fmt.Errorf("pluginhost: trial-launch for describe: %w", err)
	}
	defer p.Kill()

	return DescribeInfo{
		SourceType:        p.SourceType(),
		PluginDisplayName: p.PluginDisplayName(),
		MatchVocabulary:   p.MatchVocabulary(),
		Tier:              p.Tier(),
		BinaryHash:        p.BinaryHash(),
		OfferedKey:        p.OfferedKey(),
		Extras:            p.Extras(),
	}, nil
}

// SourceHealth is one plugin's live reachability probe result, keyed to
// its config name and Describe-learned identity. 02-02-PLAN.md's D-08:
// this is a liveness signal only — last-sync time and last-error are
// deliberately NOT part of this type, because those come from the
// kernel's own sync_runs history (kernel/index.Store), never from a
// plugin's self-reported HealthResponse fields (A-PLUG-04). A plugin that
// reports a rosier history than the kernel actually recorded must not be
// able to turn its own health chip green.
type SourceHealth struct {
	Name        string // config key under [sources.<name>]
	SourceType  string // Describe-learned
	DisplayName string // Describe-learned
	// Plugin is the launched instance's plugin BINARY name (src.Plugin,
	// e.g. "topos-plugin-mock") — never the instance id or SourceType.
	// Threaded through to GET /api/sources (09-01-PLAN.md Task 3) so the
	// SPA can address GET /api/plugins/{plugin}/icon per row without a
	// second lookup.
	Plugin string
	// Tier is the launched instance's launch-time provenance Tier
	// (T-11-01) — set from the same *Plugin.Tier() every other field on
	// this line reads, never re-derived from a live RPC.
	Tier Tier
	// PinnedHash is the SHA-256 this instance's external-tier binary
	// matched in [plugins.pins] at launch time (11-03-PLAN.md Task 1) —
	// empty for a trusted-tier instance (D-04: never pinned) or a
	// trial-launched one (pin verification skipped). Lets a healthy
	// external-tier GET /api/sources entry show its current pin, not only
	// a pin-mismatched one.
	PinnedHash string
	// TrustedKey is the operator key id that vouched (TierOperatorTrusted
	// only); OfferedKey is an unknown self-describing key's offer carried
	// by an external binary (davison/topos#49). Empty/nil otherwise.
	TrustedKey string
	OfferedKey *KeyOffer
	// LaunchAdvisory is a closed-vocabulary, non-fatal fact about a
	// LAUNCHED instance's provenance (13-05-PLAN.md Task 3, D-14) — today
	// only LaunchAdvisoryShadowed, set when this trusted-tier instance's
	// binary name also exists as a regular file in the configured
	// external directory. Empty for every instance with nothing to
	// advise about. Distinct from a LaunchFailure: an instance carrying
	// an advisory DID launch.
	LaunchAdvisory string
	Reachable      bool
	ProbeError     string
}

// launchAdvisoryFor returns LaunchAdvisoryShadowed when p's binary shadowed
// a same-named external-directory file at launch time (D-14), or an empty
// string otherwise — the one mapping from Plugin.shadowed onto
// SourceHealth.LaunchAdvisory's closed vocabulary, kept as its own
// function so a future second advisory class has an established place to
// extend this mapping.
func launchAdvisoryFor(p *Plugin) string {
	if p.shadowed {
		return LaunchAdvisoryShadowed
	}
	return ""
}

// ProbeSources calls every launched plugin's Health RPC concurrently — a
// live reachability probe, not a cached value — and returns one
// SourceHealth per plugin, in launch order (so the response order is
// stable run to run, matching the deterministic ordering
// kernel/httpapi.SourcesHandler sorts on top of). A transport error or a
// Reachable:false response both map to Reachable:false with the error
// text in ProbeError; ProbeSources itself never returns a Go error — one
// plugin being unreachable is data, not a failure of the whole probe.
func (h *Host) ProbeSources(ctx context.Context) []SourceHealth {
	plugins := h.snapshot()
	out := make([]SourceHealth, len(plugins))
	var wg sync.WaitGroup
	for i, p := range plugins {
		wg.Add(1)
		go func(i int, p *Plugin) {
			defer wg.Done()
			health := SourceHealth{Name: p.Name(), SourceType: p.SourceType(), DisplayName: p.DisplayName(), Plugin: p.src.Plugin, Tier: p.tier, PinnedHash: p.pinnedHash, TrustedKey: p.trustedKey, OfferedKey: p.offeredKey, LaunchAdvisory: launchAdvisoryFor(p)}
			resp, err := p.Health(ctx)
			switch {
			case err != nil:
				health.Reachable = false
				health.ProbeError = err.Error()
			case !resp.GetReachable():
				health.Reachable = false
				health.ProbeError = resp.GetLastError()
			default:
				health.Reachable = true
			}
			out[i] = health
		}(i, p)
	}
	wg.Wait()
	return out
}

// FetchResult is the kernel-domain translation of a plugin's unary Fetch
// response. Body is nil when there is no rendition (available=false, or a
// full-variant fetch that carried only extracted text).
type FetchResult struct {
	Available         bool
	UnavailableReason string
	MimeType          string
	SizeBytes         int64
	Text              string
	Provenance        map[string]string
	Body              io.ReadCloser
	// ContentShape carries FetchResponse.content_shape (D-11) into the
	// kernel domain — meaningful only when MimeType is "text/html";
	// kernel/httpapi.sanitizeAndWrapRendition uses it to select the
	// sanitize/wrap/theme policy for this rendition. Zero value
	// (CONTENT_SHAPE_UNSPECIFIED) for every other MimeType.
	ContentShape toposv1.ContentShape
}

// Fetch calls the Fetch RPC on the plugin registered for source (the
// source INSTANCE id supplied by an index row — item.Item.Source, D-08),
// request-time only (never sync-time), and translates the result into
// kernel-domain types and errors. gRPC codes.NotFound maps to
// ErrItemNotFound, codes.Unavailable and any other transport failure maps
// to ErrSourceUnavailable.
func (h *Host) Fetch(ctx context.Context, source, sourceID string, variant toposv1.ContentVariant) (FetchResult, error) {
	p := h.byInstance(source)
	if p == nil {
		return FetchResult{}, fmt.Errorf("%w: no plugin registered for source %q", ErrItemNotFound, source)
	}

	resp, err := p.impl.Fetch(ctx, &toposv1.FetchRequest{SourceId: sourceID, Variant: variant})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return FetchResult{}, fmt.Errorf("%w: %s", ErrItemNotFound, st.Message())
			case codes.Unavailable:
				return FetchResult{}, fmt.Errorf("%w: %s", ErrSourceUnavailable, st.Message())
			}
		}
		return FetchResult{}, fmt.Errorf("%w: %v", ErrSourceUnavailable, err)
	}

	var body io.ReadCloser
	if len(resp.GetData()) > 0 {
		body = io.NopCloser(bytes.NewReader(resp.GetData()))
	}

	return FetchResult{
		Available:         resp.GetAvailable(),
		UnavailableReason: resp.GetUnavailableReason(),
		MimeType:          resp.GetMimeType(),
		SizeBytes:         resp.GetSizeBytes(),
		Text:              resp.GetText(),
		Provenance:        resp.GetProvenance(),
		Body:              body,
		ContentShape:      resp.GetContentShape(),
	}, nil
}

// byInstance returns the launched plugin whose config-key instance id
// (Name()) matches, or nil if none is registered. Keyed on instance id, not
// on the Describe-learned plugin kind, so two launched instances of one
// plugin binary resolve to two distinct *Plugin values (D-08). Takes
// h.mu.RLock() itself (via snapshot) — Fetch, its only caller, makes no
// other access to h.plugins, so there is no nested-lock risk here.
func (h *Host) byInstance(source string) *Plugin {
	for _, p := range h.snapshot() {
		if p.name == source {
			return p
		}
	}
	return nil
}

// Shutdown kills every launched plugin subprocess.
func (h *Host) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, p := range h.plugins {
		p.Kill()
	}
}
