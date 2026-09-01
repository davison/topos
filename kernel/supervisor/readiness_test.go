package supervisor

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/pluginhost"
)

// TestBoot_FirstRefreshSurvivesAPluginLaunchReadinessWindow is the first
// automated gate in this repo for 08-UAT.md gap G-08-4's class: a plugin
// subprocess that completes the go-plugin handshake before it can actually
// serve Match. Every Phase 8 hermetic gate before this one was structurally
// blind to this failure class, because plugins/mock's Match was
// unconditionally ready — its own readiness window was exactly zero, so no
// fixture in this repo could ever express "the relaunched plugin is not
// ready yet". This test drives a REAL mock subprocess through a genuine
// 700ms launch-readiness window (plugins/mock/readiness.go) and proves the
// kernel's scheduler (kernel/syncer/scheduler.go's bounded first-refresh
// retry) survives it: the source ends up with an "ok" latest sync run and
// persisted, streamable items, rather than pinned on the launch window's
// errored row for the default sync interval.
//
// Phase 11 Task 2 (D-14): this fixture no longer relies on blanket
// environment inheritance — kernel/pluginhost.launch's exec.Cmd carries
// goplugin.ClientConfig.SkipHostEnv:true, and go-plugin's own Client()
// unconditionally appends the FULL os.Environ() onto cmd.Env unless that
// field is set, so a variable merely set on the kernel test process no
// longer reaches the subprocess at all. It now travels the documented,
// reference-driven path instead: t.Setenv sets the value on the kernel
// process exactly as before, but mock-01's own [sources.mock-01.extras]
// block below declares a "${WEBSPACES_MOCK_READY_AFTER_MS}" reference —
// config.EnvRefNames(rawSrc) picks that reference up, and
// kernel/pluginhost.allowedEnv copies the value behind it into the
// subprocess's environment because THIS instance's own raw config
// actually referenced it, not because the kernel happened to have it set.
func TestBoot_FirstRefreshSurvivesAPluginLaunchReadinessWindow(t *testing.T) {
	// t.Setenv MUST run first, before building anything: allowedEnv reads
	// os.LookupEnv at launch time, so the value must already be set on the
	// kernel test process before NewSupervisor below performs its boot-time
	// launch. t.Setenv also forbids t.Parallel() on this test, which is
	// correct — this test owns process env for its duration.
	t.Setenv("WEBSPACES_MOCK_READY_AFTER_MS", "700")

	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.mock-01]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.mock-01.extras]
ready_after_ms = "${WEBSPACES_MOCK_READY_AFTER_MS}"

[webspaces.demo]
keywords = ["demo"]
`)

	// Booting is what fires the eager first refresh — do not call Refresh
	// manually. The whole point of this test is to exercise the exact path
	// that produced the gap: NewSupervisor -> startScheduler -> Scheduler.Run
	// -> runSource's immediate first refresh, racing the mock subprocess's
	// own 700ms readiness window.
	sup, err := NewSupervisor(ctx, idx, cfgStore, pluginhost.Dirs{Trusted: dir}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	// Poll LatestSyncRunPerSource against a generous deadline: the
	// readiness window (0.7s) is far shorter than
	// kernel/syncer/scheduler.go's defaultFirstRefreshRetryDelays total
	// (its authoritative production backoff schedule, not restated here as
	// a bare number that could silently drift when that constant is
	// tuned), and this deadline is generous relative to both, so the retry
	// should supersede the launch-window error well inside 15s even on a
	// loaded machine.
	deadline := time.Now().Add(15 * time.Second)
	var lastStatus, lastError string
	synced := false
	for time.Now().Before(deadline) {
		runs, err := idx.LatestSyncRunPerSource(ctx)
		if err != nil {
			t.Fatalf("LatestSyncRunPerSource: %v", err)
		}
		if run, ok := runs["mock-01"]; ok {
			lastStatus, lastError = run.Status, run.Error
			if run.Status == "ok" {
				synced = true
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !synced {
		t.Fatalf("G-08-4: mock-01 never reached an \"ok\" latest sync run within the deadline — it stayed pinned on the launch-window error instead of the scheduler's retry superseding it. Last observed status=%q error=%q", lastStatus, lastError)
	}

	// Status alone would pass even if the retry succeeded against an empty
	// result — the gap's user-visible symptom was an empty stream under a
	// failure banner, so assert the items themselves persisted too.
	items, err := idx.StreamItems(ctx, "demo", nil, nil, index.ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems(demo): %v", err)
	}
	if len(items) == 0 {
		t.Fatal("G-08-4: expected the \"demo\" webspace to stream mock-01's items once the retry succeeded, got zero")
	}
}
