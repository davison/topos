// escalation_test.go closes the standing security todo by test
// (.planning/todos/completed/2026-08-13-plugin-trust-tier-is-directory-location-not-provenance.md,
// TRUST-04, ROADMAP.md Phase 16 success criterion 4: "Every escalation
// path named in the standing security todo is closed by a committed
// test that fails if its gate is removed"). Three cases, one per named
// escalation path — config edit, file drop, name shadowing — each
// drives the REAL resolver (ResolveBinary/resolveBinaryDetailed) and the
// REAL launch gate (launch), never a stand-in. No case in this file may
// pass without exercising EvaluateTrust through one of those two entry
// points (16-03-PLAN.md Task 1).
//
// Falsifiability (the fail-first proof this file's own discipline
// requires — a test that can never fail proves nothing): each of
// TestEscalation_ConfigEditCannotGrantTrust and
// TestEscalation_FileDropCannotGrantTrust asserts tier == TierExternal
// for a binary that, under the pre-16-02 directory-derived-trust bug,
// would have resolved TierTrusted purely by sitting in Dirs.Trusted.
// Introducing a mockable evaluation hook to prove this mechanically
// would require a NEW production-code seam in discover_binaries.go's
// resolveBinaryDetailed — today it calls EvaluateTrust directly, with no
// indirection — and this plan declines to add one solely to serve a
// test. Instead, to see this suite go red for real:
//
//  1. Temporarily replace the trusted-branch evaluation in
//     kernel/pluginhost/discover_binaries.go's resolveBinaryDetailed —
//     the line
//
//     case trustedPath != "":
//     trust, err = EvaluateTrust(dirs, name, trustedPath)
//
//     — with an unconditional grant:
//
//     case trustedPath != "":
//     trust, err = Trust{Tier: TierTrusted}, nil
//
//  2. Run:
//
//     TOPOS_ESCALATION_FAILFIRST_PROOF=1 go test ./kernel/pluginhost/ -run TestEscalation -count=1 -v
//
//  3. Expect TestEscalation_ConfigEditCannotGrantTrust and
//     TestEscalation_FileDropCannotGrantTrust to FAIL (tier now comes
//     back TierTrusted), and
//     TestEscalation_FailFirstProofWeakenedGateWouldGrantTrust — skipped
//     by default, see its own doc comment — to flip from skip to PASS.
//     The suite overall exits non-zero.
//
//  4. Revert the edit before committing anything; `git diff` must be
//     empty afterward.
package pluginhost

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
)

// TestEscalation_ConfigEditCannotGrantTrust closes escalation path 1, in
// the todo's own words: "pointing plugins.dir at any other directory ...
// silently promotes everything there to trusted." A real, unsigned
// plugin binary with no link-time manifest entry and no signed manifest
// beside it is reached through BOTH Dirs.Trusted and Dirs.External in
// turn — proving path-independence, not one lucky placement — and in
// each case resolves TierExternal, and a real (non-trial) launch refuses
// with ErrPinMismatch, creating no subprocess. Changing which directory
// name a plugin binary sits under changes nothing about its trust.
func TestEscalation_ConfigEditCannotGrantTrust(t *testing.T) {
	restore := OverrideBuildManifest(map[string]string{}) // no link-time evidence at all
	t.Cleanup(restore)

	// A real built binary, in its own private directory, carrying no
	// manifest entry and no signed provenance manifest beside it — the
	// "attacker points plugins.dir at a directory of their choosing"
	// shape.
	attackerDir := copyMockBinaryToFreshDir(t)
	src := config.Source{Plugin: "topos-plugin-mock"}

	cases := []struct {
		name string
		dirs Dirs
	}{
		{"reached via Dirs.Trusted (plugins.dir)", Dirs{Trusted: attackerDir}},
		{"reached via Dirs.External (plugins.external_dir)", Dirs{External: attackerDir}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, tier, err := ResolveBinary(tc.dirs, "topos-plugin-mock", hclog.NewNullLogger())
			if err != nil {
				t.Fatalf("ResolveBinary: %v", err)
			}
			if tier != TierExternal {
				t.Fatalf("expected tier %q after a config-edit escalation attempt, got %q (path %q) — pointing a search path at an attacker-chosen directory must grant nothing", TierExternal, tier, path)
			}

			p, err := launch(context.Background(), tc.dirs, "attacker-demo", src, nil, hclog.NewNullLogger(), false)
			if err == nil {
				if p != nil {
					p.Kill()
				}
				t.Fatal("expected launch to refuse an unpinned, evidence-free binary reached via a config-edit escalation attempt")
			}
			if p != nil {
				t.Fatalf("expected no subprocess to be created, got %+v", p)
			}
			if !errors.Is(err, ErrPinMismatch) {
				t.Fatalf("expected errors.Is(err, ErrPinMismatch) — no evidence means external tier, refused for lack of a pin, got: %v", err)
			}
		})
	}
}

// TestEscalation_FileDropCannotGrantTrust closes escalation path 2:
// "copying any third-party binary into the trusted dir makes it trusted
// instantly." One directory holds TWO binaries: one covered by an
// installed link-time manifest entry, one not. Both are resolved from
// the SAME Dirs value in the same call, and DiscoverAllTiered lists
// both — the discriminator is evidence, not the directory they share.
// The dropped binary's launch refuses with no subprocess created.
func TestEscalation_FileDropCannotGrantTrust(t *testing.T) {
	dir := t.TempDir()

	// The covered binary: a real copy of the reference mock plugin,
	// covered below by an explicit link-time manifest entry naming it.
	coveredSrc := buildMockPluginDir(t)
	copyBinaryInto(t, filepath.Join(coveredSrc, "topos-plugin-mock"), filepath.Join(dir, "topos-plugin-mock"))

	// The dropped binary: a real, DIFFERENTLY NAMED build of the same
	// reference plugin source, placed in the SAME directory, deliberately
	// left out of the manifest override below.
	droppedSrc := buildRenamedMockPluginDirForManifestGate(t, "topos-plugin-dropped")
	copyBinaryInto(t, filepath.Join(droppedSrc, "topos-plugin-dropped"), filepath.Join(dir, "topos-plugin-dropped"))

	coveredHash := mustHashBinary(t, filepath.Join(dir, "topos-plugin-mock"))
	restore := OverrideBuildManifest(map[string]string{"topos-plugin-mock": coveredHash}) // covers ONLY the mock binary
	t.Cleanup(restore)

	logger := hclog.NewNullLogger()
	dirs := Dirs{Trusted: dir}

	_, coveredTier, err := ResolveBinary(dirs, "topos-plugin-mock", logger)
	if err != nil {
		t.Fatalf("ResolveBinary(topos-plugin-mock): %v", err)
	}
	if coveredTier != TierTrusted {
		t.Fatalf("expected the covered binary to resolve tier %q, got %q", TierTrusted, coveredTier)
	}

	_, droppedTier, err := ResolveBinary(dirs, "topos-plugin-dropped", logger)
	if err != nil {
		t.Fatalf("ResolveBinary(topos-plugin-dropped): %v", err)
	}
	if droppedTier != TierExternal {
		t.Fatalf("expected the dropped binary to resolve tier %q from the SAME directory as a covered binary, got %q", TierExternal, droppedTier)
	}

	tiered, err := DiscoverAllTiered(dirs)
	if err != nil {
		t.Fatalf("DiscoverAllTiered: %v", err)
	}
	tierByName := make(map[string]Tier, len(tiered))
	for _, tb := range tiered {
		tierByName[tb.Name] = tb.Tier
	}
	if tierByName["topos-plugin-mock"] != TierTrusted {
		t.Errorf("expected DiscoverAllTiered to tag topos-plugin-mock %q, got %q", TierTrusted, tierByName["topos-plugin-mock"])
	}
	if tierByName["topos-plugin-dropped"] != TierExternal {
		t.Errorf("expected DiscoverAllTiered to tag topos-plugin-dropped %q, got %q", TierExternal, tierByName["topos-plugin-dropped"])
	}

	src := config.Source{Plugin: "topos-plugin-dropped"}
	p, err := launch(context.Background(), dirs, "dropped-demo", src, nil, logger, false)
	if err == nil {
		if p != nil {
			p.Kill()
		}
		t.Fatal("expected launch to refuse the dropped binary")
	}
	if p != nil {
		t.Fatalf("expected no subprocess to be created, got %+v", p)
	}
	if !errors.Is(err, ErrPinMismatch) {
		t.Fatalf("expected errors.Is(err, ErrPinMismatch), got: %v", err)
	}
}

// TestEscalation_ShadowingCannotInheritTrust closes escalation path 3
// (D-11): "a same-named binary dropped in the trusted dir shadows a
// pinned external plugin ... trusted-tier binaries launch unpinned." A
// binary placed under the NAME of a plugin that legitimately carries
// provenance does not inherit that plugin's trust — D-05 binds names to
// hashes, so a signature vouching for one set of bytes says nothing
// about different bytes wearing the same name.
func TestEscalation_ShadowingCannotInheritTrust(t *testing.T) {
	t.Run("digest mismatch under a legitimately-named manifest entry is a refusal, never a demotion", func(t *testing.T) {
		dir := t.TempDir()
		keyID, priv := installProvenanceTestKey(t)

		realDir := buildMockPluginDir(t)
		realHash := mustHashBinary(t, filepath.Join(realDir, "topos-plugin-mock"))
		writeSignedManifest(t, dir, "topos-plugins-v0.1.0", nativeRelease(),
			[]ProvenanceEntry{{Name: "topos-plugin-mock", SHA256: realHash, Version: "0.1.0", Contract: "topos.v1"}},
			keyID, priv)

		// An attacker's bytes, placed under the NAME a valid manifest
		// entry vouches for — the shadow attempt itself.
		if err := os.WriteFile(filepath.Join(dir, "topos-plugin-mock"), []byte("attacker-controlled-bytes-shadowing-a-real-name"), 0o755); err != nil {
			t.Fatalf("write shadowing binary: %v", err)
		}

		_, tier, err := ResolveBinary(Dirs{Trusted: dir}, "topos-plugin-mock", hclog.NewNullLogger())
		if err == nil {
			t.Fatalf("expected a refusal for a name-shadowed binary whose bytes differ from the vouched digest, got tier %q with no error", tier)
		}
		// 16-06-PLAN.md Task 1 (16-VERIFICATION.md gap 1 / CR-01 / WR-01):
		// tier is now DELIBERATELY TierTrusted on this refusal — a signed
		// manifest positively named "topos-plugin-mock" with a digest that
		// no longer matches, so this is a trusted-tier refusal on the
		// wire (matching docs/api.md's dropped-binary worked example),
		// exactly as EvaluateTrust's own doc comment now states. Tier is
		// a REPORTING field only; err (still non-nil, still
		// ErrProvenanceUnverified below) is what actually refuses this
		// binary — no subprocess is ever created, and D-13's
		// never-demote-and-run rule is unchanged: this assertion no
		// longer treats tier==TierTrusted as itself an escalation,
		// because the returned error is what launch() gates on.
		if tier == TierExternal {
			t.Fatalf("expected a REFUSAL, not a demotion to %q — a manifest positively naming this binary with a different digest must never fall back to external tier", TierExternal)
		}
		if !errors.Is(err, ErrProvenanceUnverified) {
			t.Fatalf("expected errors.Is(err, ErrProvenanceUnverified), got: %v", err)
		}
	})

	t.Run("sibling: a manifest present but not naming the shadowing binary is external tier, never a refusal", func(t *testing.T) {
		dir := t.TempDir()
		keyID, priv := installProvenanceTestKey(t)

		writeSignedManifest(t, dir, "topos-plugins-v0.1.0", nativeRelease(),
			[]ProvenanceEntry{{Name: "topos-plugin-unrelated", SHA256: strings.Repeat("a", 64), Version: "0.1.0", Contract: "topos.v1"}},
			keyID, priv)
		if err := os.WriteFile(filepath.Join(dir, "topos-plugin-mock"), []byte("attacker-controlled-bytes-no-manifest-entry"), 0o755); err != nil {
			t.Fatalf("write binary: %v", err)
		}

		_, tier, err := ResolveBinary(Dirs{Trusted: dir}, "topos-plugin-mock", hclog.NewNullLogger())
		if err != nil {
			t.Fatalf("expected no error (no evidence means external tier, not a refusal), got: %v", err)
		}
		if tier != TierExternal {
			t.Fatalf("expected tier %q, got %q", TierExternal, tier)
		}
	})

	// 16-REVIEW.md CR-01 (this iteration's fix) corrected this subtest's
	// own expectation. It was originally written to prove "whichever
	// copy carries evidence wins," expecting err == nil and the
	// external copy's path back. That passed only because of the exact
	// bug CR-01 closes: VerifySignedProvenance scans EVERY manifest in
	// BOTH dirs.Trusted and dirs.External before deciding (D-08,
	// provenance.go), so the signed manifest living in externalDir —
	// which correctly names the external copy's own digest — is ALSO
	// consulted while evaluating the trusted copy, and the trusted
	// copy's unrelated bytes necessarily mismatch that same entry. That
	// makes the trusted copy a genuine tamper refusal in its own right
	// (not "no evidence" — VerifySignedProvenance's own precedence rule
	// has no "not applicable to this candidate" outcome once a manifest
	// names the collision's filename: only match or mismatch). The
	// pre-fix collision resolver's two independent `if err == nil`
	// checks silently discarded that trusted-side refusal and returned
	// the external copy instead — exactly CR-01's scenario 1. With the
	// fix, the trusted-side refusal now wins outright, matching
	// docs/plugin-contract.md's invariant that a tamper refusal never
	// falls back to launching the other copy.
	t.Run("cross-directory shadow: a same-named manifest that vouches for one copy makes the OTHER copy's differing bytes a tamper refusal, which wins the collision", func(t *testing.T) {
		trustedDir := t.TempDir()
		externalDir := t.TempDir()

		if err := os.WriteFile(filepath.Join(trustedDir, "topos-plugin-crossshadow"), []byte("trusted-side-no-evidence"), 0o755); err != nil {
			t.Fatalf("write trusted fixture: %v", err)
		}
		if err := os.WriteFile(filepath.Join(externalDir, "topos-plugin-crossshadow"), []byte("external-side-carries-evidence"), 0o755); err != nil {
			t.Fatalf("write external fixture: %v", err)
		}
		keyID, priv := installProvenanceTestKey(t)
		hash := mustHashBinary(t, filepath.Join(externalDir, "topos-plugin-crossshadow"))
		writeSignedManifest(t, externalDir, "topos-plugins-v0.1.0", nativeRelease(),
			[]ProvenanceEntry{{Name: "topos-plugin-crossshadow", SHA256: hash, Version: "0.1.0", Contract: "topos.v1"}},
			keyID, priv)

		var buf bytes.Buffer
		logger := hclog.New(&hclog.LoggerOptions{Output: &buf})

		path, tier, err := ResolveBinary(Dirs{Trusted: trustedDir, External: externalDir}, "topos-plugin-crossshadow", logger)
		if err == nil {
			t.Fatalf("expected the trusted-side mismatch (against the externally-vouched digest) to refuse the collision (path=%q, tier=%q), not launch either copy silently", path, tier)
		}
		if !errors.Is(err, ErrProvenanceUnverified) {
			t.Fatalf("expected errors.Is(err, ErrProvenanceUnverified), got: %v", err)
		}
		if want := filepath.Join(trustedDir, "topos-plugin-crossshadow"); path != want {
			t.Errorf("expected the refusal to be reported against the trusted-side path %q (it is checked first), got %q", want, path)
		}
		if tier != TierTrusted {
			t.Errorf("expected tier %q (a refusal is a trusted-tier refusal on the wire — err is what actually refuses), got %q", TierTrusted, tier)
		}
		if !strings.Contains(buf.String(), "topos-plugin-crossshadow") {
			t.Errorf("expected the collision to be logged by name, got: %s", buf.String())
		}
	})

	// 16-REVIEW.md CR-01 (this iteration's fix): a tamper refusal on
	// EITHER side of a cross-directory collision must win outright and
	// refuse — never be silently overridden by the other candidate's
	// clean resolution. Before the fix, resolveBinaryDetailed's two
	// independent `if trustedErr == nil && ...` / `if externalErr == nil
	// && ...` checks meant a tampered candidate satisfied neither branch,
	// so nothing refused on its behalf: direction (a) fell through to
	// the OTHER candidate's clean win (launching bytes the docs promise
	// can never launch), and direction (b) fell through to the final
	// `return trustedPath, trustedTrust, shadowed, trustedErr` with
	// trustedErr == nil, silently dropping the external tamper.
	t.Run("cross-directory: trusted-side tamper refusal wins even though the external copy independently resolves clean", func(t *testing.T) {
		const name = "topos-plugin-crossshadow-trustedtamper"
		trustedDir := t.TempDir()
		externalDir := t.TempDir()

		// The external copy's bytes are what the (shared, name-keyed)
		// link-time manifest actually vouches for — a genuine clean win
		// via the link-time arm.
		externalPath := filepath.Join(externalDir, name)
		if err := os.WriteFile(externalPath, []byte("external-side-correct-bytes"), 0o755); err != nil {
			t.Fatalf("write external fixture: %v", err)
		}
		correctHash := mustHashBinary(t, externalPath)
		restore := OverrideBuildManifest(map[string]string{name: correctHash})
		defer restore()

		// The trusted copy shares the same NAME but different bytes — a
		// genuine tamper: the manifest positively names this binary with
		// a digest that no longer matches what's on disk at the trusted
		// path.
		trustedPath := filepath.Join(trustedDir, name)
		if err := os.WriteFile(trustedPath, []byte("trusted-side-tampered-bytes"), 0o755); err != nil {
			t.Fatalf("write trusted fixture: %v", err)
		}

		var buf bytes.Buffer
		logger := hclog.New(&hclog.LoggerOptions{Output: &buf})

		path, tier, err := ResolveBinary(Dirs{Trusted: trustedDir, External: externalDir}, name, logger)
		if err == nil {
			t.Fatalf("expected the trusted-side tamper refusal to win the collision (path=%q, tier=%q) — it must never fall back to launching the external copy instead", path, tier)
		}
		if !errors.Is(err, ErrManifestUnverified) {
			t.Fatalf("expected errors.Is(err, ErrManifestUnverified), got: %v", err)
		}
		if path != trustedPath {
			t.Errorf("expected the refusal to be reported against the tampered trusted-side path %q, got %q — a caller must never be handed the external copy's path alongside a trusted-side refusal", trustedPath, path)
		}
	})

	// Mirror of the previous subtest, with the roles of trusted/external
	// swapped: here the TRUSTED copy is the one that resolves clean, and
	// the EXTERNAL copy is the tamper refusal. This is deliberately NOT
	// constructed as "external tampered, trusted has literally zero
	// evidence" (16-REVIEW.md CR-01's second numbered scenario) — that
	// exact combination turns out to be unreachable given how
	// VerifySignedProvenance/VerifyTrustedBinary actually work: both
	// arms look up their manifest entry purely by NAME (never scoped to
	// a directory or a specific candidate's path — see
	// VerifySignedProvenance's own doc comment on scanning being
	// EXHAUSTIVE across dirs.Trusted and dirs.External before deciding,
	// D-08), so once any manifest names the colliding binary at all,
	// EVERY same-named candidate is classified match-or-mismatch against
	// that one entry — never "not evaluated." A trusted copy can only
	// have "zero evidence" if the name is absent from every manifest
	// everywhere, but then the external copy's tamper (which requires
	// the name to be PRESENT with a mismatching digest) could not exist
	// either. This mirrored construction (trusted CLEAN WIN, external
	// TAMPER REFUSAL, both against the shared link-time entry) is the
	// nearest reachable equivalent, and still exercises the exact code
	// path this fix touches: the `if externalErr != nil` check must fire
	// and win BEFORE `if trustedTrust.Tier == TierTrusted` ever gets a
	// chance to return the trusted copy while silently dropping evidence
	// that the external copy sharing its name was tampered.
	t.Run("cross-directory: external-side tamper refusal wins even though the trusted copy independently resolves clean", func(t *testing.T) {
		const name = "topos-plugin-crossshadow-externaltamper"
		trustedDir := t.TempDir()
		externalDir := t.TempDir()

		// The trusted copy's bytes are what the (shared, name-keyed)
		// link-time manifest actually vouches for — a genuine clean win
		// via the link-time arm.
		trustedPath := filepath.Join(trustedDir, name)
		if err := os.WriteFile(trustedPath, []byte("trusted-side-correct-bytes"), 0o755); err != nil {
			t.Fatalf("write trusted fixture: %v", err)
		}
		correctHash := mustHashBinary(t, trustedPath)
		restore := OverrideBuildManifest(map[string]string{name: correctHash})
		defer restore()

		// The external copy shares the same NAME but different bytes —
		// a genuine tamper against the very same manifest entry that
		// just cleanly vouched for the trusted copy.
		externalPath := filepath.Join(externalDir, name)
		if err := os.WriteFile(externalPath, []byte("external-side-tampered-bytes"), 0o755); err != nil {
			t.Fatalf("write external fixture: %v", err)
		}

		path, tier, err := ResolveBinary(Dirs{Trusted: trustedDir, External: externalDir}, name, hclog.NewNullLogger())
		if err == nil {
			t.Fatalf("expected the external-side tamper refusal to win the collision (path=%q, tier=%q) — it must never be silently dropped in favor of the trusted copy's own clean resolution", path, tier)
		}
		if !errors.Is(err, ErrManifestUnverified) {
			t.Fatalf("expected errors.Is(err, ErrManifestUnverified), got: %v", err)
		}
		if path != externalPath {
			t.Errorf("expected the refusal to be reported against the tampered external-side path %q, got %q — a caller must never be handed the trusted copy's path while the external copy sharing its name is a tamper refusal", externalPath, path)
		}
	})
}

// escalationFailFirstProofEnvVar gates
// TestEscalation_FailFirstProofWeakenedGateWouldGrantTrust — see that
// test's own doc comment and this file's header comment.
const escalationFailFirstProofEnvVar = "TOPOS_ESCALATION_FAILFIRST_PROOF"

// TestEscalation_FailFirstProofWeakenedGateWouldGrantTrust is TRUST-04's
// falsifiability proof, SKIPPED BY DEFAULT. It asserts the OPPOSITE
// outcome of TestEscalation_ConfigEditCannotGrantTrust /
// TestEscalation_FileDropCannotGrantTrust — that a binary found only in
// Dirs.Trusted, with zero provenance evidence, resolves TierTrusted — so
// it can only PASS while the real gate those tests depend on has been
// temporarily removed. It exists to demonstrate this suite is
// falsifiable one way or the other (a test that can never fail proves
// nothing), without adding a production-code seam:
// resolveBinaryDetailed/EvaluateTrust call each other directly today
// (discover_binaries.go), and a mockable indirection purely to serve
// this test would itself be new production surface this plan declines
// to add. See this file's own header comment for the exact edit and
// command that exercise the proof for real.
//
// Under normal CI this test is always skipped and can never contribute a
// false green. Enabling it against an UNMODIFIED discover_binaries.go
// (the env var set, but the documented temporary edit NOT applied) is
// expected to FAIL — that is itself the intended tripwire: this test
// must never accidentally pass against secure code.
func TestEscalation_FailFirstProofWeakenedGateWouldGrantTrust(t *testing.T) {
	if os.Getenv(escalationFailFirstProofEnvVar) == "" {
		t.Skipf("skipped by default — see this test's doc comment and this file's header comment for how to exercise TRUST-04's fail-first proof; set %s=1 only after applying the documented temporary edit to discover_binaries.go", escalationFailFirstProofEnvVar)
	}

	dir := t.TempDir()
	writeFixtureFile(t, dir, "topos-plugin-failfirst-proof")
	restore := OverrideBuildManifest(map[string]string{}) // zero evidence
	t.Cleanup(restore)

	_, tier, err := ResolveBinary(Dirs{Trusted: dir}, "topos-plugin-failfirst-proof", hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("ResolveBinary: %v", err)
	}
	if tier != TierTrusted {
		t.Fatalf("expected the weakened gate (per the documented temporary edit) to grant tier %q to a binary present in Dirs.Trusted with zero evidence; got %q instead — either the documented edit was not applied to discover_binaries.go, or the real gate is intact (correct for a normal run; this test must only be enabled after applying the edit)", TierTrusted, tier)
	}
}
