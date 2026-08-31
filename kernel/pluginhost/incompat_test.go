package pluginhost

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/sdk"
)

// buildHandshakeV1PluginDir builds testdata/handshakev1 — a real
// go-plugin server at ProtocolVersion 1 (see its own doc comment) —
// once per test binary run, mirroring buildMockPluginDir's own caching
// discipline, and returns the directory it was written to.
var (
	handshakeV1DirOnce sync.Once
	handshakeV1Dir     string
	handshakeV1DirErr  error
)

func buildHandshakeV1PluginDir(t *testing.T) string {
	t.Helper()
	handshakeV1DirOnce.Do(func() {
		out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/davison/topos").Output()
		if err != nil {
			handshakeV1DirErr = fmt.Errorf("resolve module root: %w", err)
			return
		}
		root := strings.TrimSpace(string(out))

		dir, err := os.MkdirTemp("", "topos-pluginhost-handshakev1-test-*")
		if err != nil {
			handshakeV1DirErr = err
			return
		}

		bin := filepath.Join(dir, "topos-plugin-handshakev1")
		cmd := exec.Command("go", "build", "-o", bin, "./kernel/pluginhost/testdata/handshakev1")
		cmd.Dir = root
		if buildOut, err := cmd.CombinedOutput(); err != nil {
			handshakeV1DirErr = fmt.Errorf("build handshakev1 fixture: %w\n%s", err, buildOut)
			return
		}
		handshakeV1Dir = dir
	})
	if handshakeV1DirErr != nil {
		t.Fatalf("build handshakev1 fixture: %v", handshakeV1DirErr)
	}
	return handshakeV1Dir
}

// TestDiscover_MissingBinaryRecordsLaunchFailedAndBootContinues proves
// the R6 core (M1-R6/DIST-03): a source naming a binary present in
// NEITHER directory no longer costs the whole boot — it becomes a named
// launch_failed record while the healthy sibling launches normally.
func TestDiscover_MissingBinaryRecordsLaunchFailedAndBootContinues(t *testing.T) {
	dir := buildMockPluginDir(t)
	installTrustedManifest(t, dir)

	h, err := Discover(context.Background(), Dirs{Trusted: dir}, nil, map[string]config.Source{
		"demo":  {Plugin: "topos-plugin-mock"},
		"bogus": {Plugin: "topos-plugin-does-not-exist", DisplayName: "Bogus Source"},
	}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover must boot around a missing binary, got: %v", err)
	}
	defer h.Shutdown()

	if got := len(h.Plugins()); got != 1 {
		t.Fatalf("expected the healthy sibling to launch, got %d plugins", got)
	}
	failures := h.LaunchFailures()
	if len(failures) != 1 {
		t.Fatalf("expected exactly one launch failure, got %d: %+v", len(failures), failures)
	}
	f := failures[0]
	if f.Instance != "bogus" || f.Reason != LaunchFailureLaunchFailed {
		t.Fatalf("expected instance %q reason %q, got %+v", "bogus", LaunchFailureLaunchFailed, f)
	}
	if f.DisplayName != "Bogus Source" {
		t.Errorf("expected the operator's display name on the record, got %q", f.DisplayName)
	}
	if !strings.Contains(f.Message, "topos-plugin-does-not-exist") || !strings.Contains(f.Message, "not found") {
		t.Errorf("expected the message to name the missing binary, got: %s", f.Message)
	}
}

// TestDiscover_ContractMismatchRecordsContractIncompatible proves the
// contract-generation gate end to end with a REAL launched plugin: the
// mock's fixture-only override (WEBSPACES_MOCK_CONTRACT_VERSION,
// plugins/mock/contractfixture.go) makes it DECLARE topos.v1 at
// Describe; the kernel kills it and records contract_incompatible
// naming both generations, while a second instance of the SAME binary —
// no override reference in its config, so the env var never reaches its
// subprocess (allowedEnv's per-instance ${VAR} allowlist) — launches
// normally.
func TestDiscover_ContractMismatchRecordsContractIncompatible(t *testing.T) {
	dir := buildMockPluginDir(t)
	installTrustedManifest(t, dir)
	t.Setenv("WEBSPACES_MOCK_CONTRACT_VERSION", "topos.v1")

	h, err := Discover(context.Background(), Dirs{Trusted: dir}, nil, map[string]config.Source{
		"stale": {
			Plugin: "topos-plugin-mock",
			// The ${VAR} reference is what admits the env var into THIS
			// instance's subprocess environment — the key name itself is
			// arbitrary (the mock reads the env var, not this extras
			// value), mirroring the e2e harness's own rendition-fixture
			// pattern.
			Extras: map[string]string{"contract": "${WEBSPACES_MOCK_CONTRACT_VERSION}"},
		},
		"healthy": {Plugin: "topos-plugin-mock"},
	}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover must boot around a contract mismatch, got: %v", err)
	}
	defer h.Shutdown()

	if got := len(h.Plugins()); got != 1 {
		t.Fatalf("expected only the healthy instance to launch, got %d plugins", got)
	}
	if h.Plugins()[0].Name() != "healthy" {
		t.Fatalf("expected the healthy instance to be the launched one, got %q", h.Plugins()[0].Name())
	}
	failures := h.LaunchFailures()
	if len(failures) != 1 {
		t.Fatalf("expected exactly one launch failure, got %d: %+v", len(failures), failures)
	}
	f := failures[0]
	if f.Instance != "stale" || f.Reason != LaunchFailureContractIncompatible {
		t.Fatalf("expected instance %q reason %q, got %+v", "stale", LaunchFailureContractIncompatible, f)
	}
	if f.Tier != TierTrusted {
		t.Errorf("expected the trusted tier on the record (the binary IS trusted; its contract is not), got %q", f.Tier)
	}
	if !strings.Contains(f.Message, `"topos.v1"`) || !strings.Contains(f.Message, sdk.ContractVersion) {
		t.Errorf("expected the message to name BOTH generations (declared topos.v1, supported %s), got: %s", sdk.ContractVersion, f.Message)
	}
}

// TestDiscover_HandshakeVersionMismatchRecordsHandshakeIncompatible
// proves the handshake arm with a REAL go-plugin server built at
// ProtocolVersion 1 (testdata/handshakev1): the refusal is classified
// handshake_incompatible — not the generic class — and the recorded
// message carries go-plugin's own text naming both protocol versions.
// This test is ALSO the tripwire connectFailureReason's doc comment
// promises: if go-plugin ever rewords its version-refusal error, this
// fails by name instead of the classification silently degrading.
func TestDiscover_HandshakeVersionMismatchRecordsHandshakeIncompatible(t *testing.T) {
	dir := buildHandshakeV1PluginDir(t)
	installTrustedManifest(t, dir)

	h, err := Discover(context.Background(), Dirs{Trusted: dir}, nil, map[string]config.Source{
		"stale": {Plugin: "topos-plugin-handshakev1"},
	}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover must boot around a handshake mismatch, got: %v", err)
	}
	defer h.Shutdown()

	failures := h.LaunchFailures()
	if len(failures) != 1 {
		t.Fatalf("expected exactly one launch failure, got %d: %+v", len(failures), failures)
	}
	f := failures[0]
	if f.Reason != LaunchFailureHandshakeIncompatible {
		t.Fatalf("expected reason %q (go-plugin's version-refusal text may have changed — see connectFailureReason), got %+v", LaunchFailureHandshakeIncompatible, f)
	}
	if !strings.Contains(f.Message, "1") || !strings.Contains(f.Message, "2") {
		t.Errorf("expected the message to carry both protocol versions, got: %s", f.Message)
	}
}

// TestReconcile_ContractMismatchCommitsSiblings proves the apply-path
// half of R6: a reconcile adding one contract-incompatible instance and
// one healthy instance returns nil, commits the healthy launch, and
// records the mismatch — the whole-save failure is gone.
func TestReconcile_ContractMismatchCommitsSiblings(t *testing.T) {
	dir := buildMockPluginDir(t)
	installTrustedManifest(t, dir)
	t.Setenv("WEBSPACES_MOCK_CONTRACT_VERSION", "topos.v9")

	h, err := Discover(context.Background(), Dirs{Trusted: dir}, nil, map[string]config.Source{}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	defer h.Shutdown()

	if err := h.Reconcile(context.Background(), nil, map[string]config.Source{
		"stale": {
			Plugin: "topos-plugin-mock",
			Extras: map[string]string{"contract": "${WEBSPACES_MOCK_CONTRACT_VERSION}"},
		},
		"healthy": {Plugin: "topos-plugin-mock"},
	}, hclog.NewNullLogger()); err != nil {
		t.Fatalf("Reconcile must commit around a contract mismatch, got: %v", err)
	}

	if got := len(h.Plugins()); got != 1 {
		t.Fatalf("expected the healthy instance committed, got %d plugins", got)
	}
	failures := h.LaunchFailures()
	if len(failures) != 1 || failures[0].Reason != LaunchFailureContractIncompatible {
		t.Fatalf("expected one contract_incompatible record, got %+v", failures)
	}
	if !strings.Contains(failures[0].Message, `"topos.v9"`) {
		t.Errorf("expected the declared generation in the message, got: %s", failures[0].Message)
	}
}
