package supervisor

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
)

// TestResume_SlowRelaunchDoesNotFreezeOtherSources is the hermetic,
// real-subprocess gate for phase success criterion 4's "every other
// source is unaffected" clause and 08-VERIFICATION.md gap G-08-5: while a
// suspended source instance's resume closure is relaunching a plugin
// subprocess slow to complete the go-plugin handshake, every OTHER
// configured source's health-probe and manual-refresh path must still
// answer promptly — the supervisor's reader path (Host()/Coordinator())
// must never wait behind a plugin subprocess launch, no matter how long
// that launch takes.
//
// The "demo" webspace carries an explicit match block for EACH instance —
// the same shape suspend_test.go's own participating-webspace fixture
// documents as load-bearing: a webspace that does not participate would
// never touch the plugin handles at all, and this test would pass against
// the defective code while proving nothing.
func TestResume_SlowRelaunchDoesNotFreezeOtherSources(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	// A long [sync] interval so a scheduled tick can never interleave with
	// this test's own timing assertions below.
	cfgStore := newTestConfigStore(t, `
[sync]
interval = "1h"

[sources.slow-one]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.control]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.demo]
sources = ["slow-one", "control"]

[webspaces.demo.match.slow-one]
labels = ["demo"]

[webspaces.demo.match.control]
labels = ["demo"]
`)

	// Boot with the launch-delay variable UNSET: boot must be fast.
	sup, err := NewSupervisor(ctx, idx, cfgStore, dir, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	if got := len(sup.Host().Plugins()); got != 2 {
		t.Fatalf("expected 2 launched plugins at boot, got %d", got)
	}

	// SuspendInstance only kills — fast, regardless of the delay variable,
	// which is not set yet at this point.
	resume, err := sup.SuspendInstance(ctx, "slow-one")
	if err != nil {
		t.Fatalf("SuspendInstance: %v", err)
	}

	// Set the delay ONLY now, so it applies to the relaunch this test
	// drives below and not to the boot above. Referenced by its literal
	// name, not an imported constant: plugins/mock is a separate Go module
	// and its launchDelayEnvVar constant is not importable from
	// kernel/supervisor — see plugins/mock/readiness.go for the
	// authoritative definition of this variable's name and parsing
	// contract.
	t.Setenv("WEBSPACES_MOCK_LAUNCH_DELAY_MS", "4000")

	resumeErrCh := make(chan error, 1)
	resumeReturned := make(chan struct{})
	go func() {
		resumeErrCh <- resume(context.Background())
		close(resumeReturned)
	}()

	// Sleep long enough that the resume is genuinely inside the launch
	// (not yet returned), so the assertions below measure a real window
	// under test rather than an already-finished resume.
	time.Sleep(300 * time.Millisecond)
	select {
	case <-resumeReturned:
		t.Fatal("resume returned before its slow relaunch could plausibly have completed — the test's timing assumption is invalid, so the rest of this test proves nothing")
	default:
	}

	// A 4000ms delay against a 2s threshold, chosen deliberately: with the
	// fix, ProbeSources is a lock-free read (genMu.RLock, never s.mu) plus
	// one gRPC Health call to an already-running subprocess —
	// sub-millisecond in practice, a ~2000x margin under 2s. Without the
	// fix, ProbeSources would resolve Host() through s.mu and wait behind
	// the resume closure's s.mu hold across the whole 4s launch — it could
	// not return before that completes, a 2x margin the other way. The
	// threshold sits between two orders of magnitude, not on a knife edge.
	// Every delay used here stays far below go-plugin's own one-minute
	// client StartTimeout default.
	probeStart := time.Now()
	health := sup.ProbeSources(ctx)
	probeElapsed := time.Since(probeStart)
	if probeElapsed >= 2*time.Second {
		t.Fatalf("phase success criterion 4 / G-08-5: ProbeSources took %v (>= 2s) while an unrelated source's resume closure was relaunching a WhatsApp-shaped slow plugin — a health probe of an unrelated source must never block behind a plugin subprocess relaunch", probeElapsed)
	}
	if len(health) != 1 {
		t.Fatalf("expected exactly one health entry (only \"control\" launched during the relaunch window), got %d: %+v", len(health), health)
	}
	if health[0].Name != "control" {
		t.Fatalf("expected the one health entry to be \"control\", got %q", health[0].Name)
	}
	if !health[0].Reachable {
		t.Fatalf("expected \"control\" to report reachable during the relaunch window, got ProbeError %q", health[0].ProbeError)
	}

	// The Coordinator() half of the reader path — ProbeSources alone does
	// not exercise this.
	refreshStart := time.Now()
	refreshResult, err := sup.Refresh(ctx, "control")
	refreshElapsed := time.Since(refreshStart)
	if err != nil {
		t.Fatalf("Refresh(control) during the relaunch window: %v", err)
	}
	if refreshElapsed >= 2*time.Second {
		t.Fatalf("phase success criterion 4 / G-08-5: Refresh(control) took %v (>= 2s) while an unrelated source's resume closure was relaunching a WhatsApp-shaped slow plugin", refreshElapsed)
	}
	if refreshResult.Status != "ok" {
		t.Fatalf("expected Refresh(control) to report status \"ok\", got %q (error: %q)", refreshResult.Status, refreshResult.Error)
	}

	<-resumeReturned
	if err := <-resumeErrCh; err != nil {
		t.Fatalf("resume: %v", err)
	}

	resumed := sup.Host().Plugins()
	if len(resumed) != 2 {
		t.Fatalf("expected both instances launched again after resume, got %+v", namesOf(resumed))
	}

	// Idempotency edge: calling the same resume closure a second time,
	// after it has already returned, must be a no-op — no relaunch, no
	// error, the same launched set.
	if err := resume(context.Background()); err != nil {
		t.Fatalf("second call to resume must be a no-op, got error: %v", err)
	}
	resumedAgain := sup.Host().Plugins()
	if len(resumedAgain) != 2 {
		t.Fatalf("expected exactly two launched instances after a second resume call, got %+v", namesOf(resumedAgain))
	}
	if got, want := namesOf(resumedAgain), namesOf(resumed); got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("expected the same launched instance names after a second resume call, got %+v (was %+v)", got, want)
	}
}
