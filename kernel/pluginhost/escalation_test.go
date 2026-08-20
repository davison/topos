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
		if tier == TierTrusted {
			t.Fatalf("must never resolve %q for a digest-mismatched shadow", TierTrusted)
		}
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

	t.Run("cross-directory shadow resolves to whichever copy carries evidence and logs the collision by name", func(t *testing.T) {
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
		if err != nil {
			t.Fatalf("ResolveBinary: %v", err)
		}
		if want := filepath.Join(externalDir, "topos-plugin-crossshadow"); path != want {
			t.Errorf("expected the copy WITH evidence (%s) to win the collision, got %q", want, path)
		}
		if tier != TierTrusted {
			t.Errorf("expected tier %q, got %q", TierTrusted, tier)
		}
		if !strings.Contains(buf.String(), "topos-plugin-crossshadow") {
			t.Errorf("expected the collision to be logged by name, got: %s", buf.String())
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
