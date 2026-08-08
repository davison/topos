package supervisor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/correlate"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/item"
	"github.com/davison/topos/kernel/pluginhost"
	"github.com/davison/topos/kernel/syncer"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

func newTestIndex(t *testing.T) *index.Store {
	t.Helper()
	s, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// buildMockPluginDir builds the repo's plugins/mock reference plugin
// fresh, once per test binary run, into a shared temp directory — the
// real-subprocess fixture the "removed instance" test below needs
// (kernel/pluginhost.Plugin.Kill() panics on a hand-built value with no
// real client, exactly as kernel/pluginhost/reconcile_test.go's own
// identically-named helper documents).
var (
	mockPluginDirOnce sync.Once
	mockPluginDir     string
	mockPluginDirErr  error
)

func buildMockPluginDir(t *testing.T) string {
	t.Helper()
	mockPluginDirOnce.Do(func() {
		out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/davison/topos").Output()
		if err != nil {
			mockPluginDirErr = fmt.Errorf("resolve module root: %w", err)
			return
		}
		root := strings.TrimSpace(string(out))

		dir, err := os.MkdirTemp("", "topos-supervisor-test-*")
		if err != nil {
			mockPluginDirErr = err
			return
		}

		bin := filepath.Join(dir, "topos-plugin-mock")
		cmd := exec.Command("go", "build", "-o", bin, "./plugins/mock")
		cmd.Dir = root
		if buildOut, err := cmd.CombinedOutput(); err != nil {
			mockPluginDirErr = fmt.Errorf("build mock plugin: %w\n%s", err, buildOut)
			return
		}

		mockPluginDir = dir
	})
	if mockPluginDirErr != nil {
		t.Fatalf("build mock plugin fixture: %v", mockPluginDirErr)
	}
	return mockPluginDir
}

// newTestConfigStore writes contents to a real temp config.toml and wraps
// it in a *config.Store — Apply reads live state off cfgStore.Expanded(),
// and a Save/Reload-backed *config.Store (rather than
// config.NewStoreForTesting) is what lets a test drive Apply through the
// exact same swap path ConfigSaveHandler/ConfigReloadHandler use in
// production.
func newTestConfigStore(t *testing.T, contents string) *config.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	s, err := config.NewStore(path)
	if err != nil {
		t.Fatalf("config.NewStore: %v", err)
	}
	return s
}

// TestApply_RemovedInstance_PluginGoneAndIndexRowsGone drives Apply
// through a REAL config.Store.Save swap (the production path) over two
// genuinely launched mock-plugin instances, then removes one via a second
// save: Host.Plugins() must no longer contain it and its index rows must
// be gone, while the surviving instance's plugin and rows are untouched.
func TestApply_RemovedInstance_PluginGoneAndIndexRowsGone(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	// base_url/token are required by config.Validate's unconditional
	// presence check even for the mock plugin, which ignores both
	// (STATE.md's documented, deliberately-not-relaxed limitation) — dummy
	// values only, never read by plugins/mock.
	cfgStore := newTestConfigStore(t, `
[sources.keep]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.remove]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.demo]
keywords = ["demo"]
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, dir, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	if len(sup.Host().Plugins()) != 2 {
		t.Fatalf("expected 2 launched plugins at boot, got %d", len(sup.Host().Plugins()))
	}

	if err := idx.ReplaceWebspaceSourceItems(ctx, "demo", "remove", []item.Item{testFixtureItem("remove", "1")}); err != nil {
		t.Fatalf("seed removed-instance items: %v", err)
	}
	if err := idx.ReplaceWebspaceSourceItems(ctx, "demo", "keep", []item.Item{testFixtureItem("keep", "1")}); err != nil {
		t.Fatalf("seed kept-instance items: %v", err)
	}

	next := &config.Config{
		Sources: map[string]config.Source{
			"keep": {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused"},
		},
		Webspaces: map[string]config.Webspace{
			"demo": {Keywords: []string{"demo"}},
		},
	}
	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := sup.Apply(ctx); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	plugins := sup.Host().Plugins()
	if len(plugins) != 1 || plugins[0].Name() != "keep" {
		t.Fatalf("expected only the 'keep' instance to remain launched after Apply, got %+v", plugins)
	}

	if _, ok, err := idx.GetItem(ctx, "remove:1"); err != nil {
		t.Fatalf("GetItem(remove:1): %v", err)
	} else if ok {
		t.Error("expected the removed instance's index rows to be gone after Apply")
	}
	if _, ok, err := idx.GetItem(ctx, "keep:1"); err != nil {
		t.Fatalf("GetItem(keep:1): %v", err)
	} else if !ok {
		t.Error("expected the surviving instance's index rows to be untouched by Apply")
	}
}

// testFixtureItem builds a minimal, valid item.Item — kept local to this
// file rather than importing kernel/index's own test fixtures
// (unexported, different package).
func testFixtureItem(source, sourceID string) item.Item {
	return item.Item{
		ID:            item.ID(source, sourceID),
		Source:        source,
		SourceType:    "mock",
		SourceID:      sourceID,
		Title:         "Doc " + sourceID,
		Preview:       "preview text",
		TimestampUnix: 100,
		Fidelity:      item.FidelityExact,
		DeepLink:      "http://example.test/" + sourceID,
		Provenance:    map[string]string{"source_type": "mock"},
	}
}

// blockingSource is a correlate.Source whose Match blocks until ctx is
// cancelled — used to put a sync genuinely "in flight" at the moment
// Apply is invoked. entered is closed at most once (guarded by
// closeEnteredOnce): the Reconcile-failure branch this fixture's own test
// exercises restarts a scheduler generation against the unchanged config
// (deliberately, per must_haves.prohibitions — that branch's restart is
// never touched by this plan), and that restarted generation's own
// immediate first refresh calls Match on this SAME shared fixture a second
// time — a pre-existing race independent of 07-09's fix (reproduces
// identically against unmodified supervisor.go under -race), never
// previously guarded against. Guarding the close is the minimal fix: it
// changes no assertion and no line inside either of this file's two
// prior-content-pinned tests.
type blockingSource struct {
	name             string
	entered          chan struct{}
	closeEnteredOnce sync.Once
}

func (b *blockingSource) Name() string              { return b.name }
func (b *blockingSource) SourceType() string        { return "slow" }
func (b *blockingSource) MatchVocabulary() []string { return []string{"keywords"} }
func (b *blockingSource) Match(ctx context.Context, _ map[string][]string) (*toposv1.MatchResponse, error) {
	b.closeEnteredOnce.Do(func() { close(b.entered) })
	<-ctx.Done()
	return nil, ctx.Err()
}

// TestApply_MidFlightSyncLeavesNoStrandedRunningRow proves the must_have:
// "An apply that lands while a sync is in flight cancels that sync's
// scheduler context and leaves no sync_runs row stranded at status
// running." Apply's own Reconcile step is deliberately made to fail here
// (the fake source names a plugin binary that does not exist in an empty
// pluginsDir) — the point under test is entirely upstream of Reconcile:
// Apply's stopScheduler call must cancel the OLD scheduler generation's
// context and wait for its Run to fully return BEFORE anything else
// happens, so a mid-flight sync is always finalised (via
// Coordinator.syncOne's own existing detached finalize,
// kernel/syncer/coordinator.go) rather than left stranded, regardless of
// whether Reconcile itself goes on to succeed.
func TestApply_MidFlightSyncLeavesNoStrandedRunningRow(t *testing.T) {
	idx := newTestIndex(t)
	blocker := &blockingSource{name: "slow", entered: make(chan struct{})}

	cfg := &config.Config{
		Sync:      config.SyncConfig{Interval: "1h"},
		Sources:   map[string]config.Source{"slow": {Plugin: "topos-plugin-does-not-exist"}},
		Webspaces: map[string]config.Webspace{"demo": {Keywords: []string{"keywords"}}},
	}

	s := &Supervisor{
		idx:        idx,
		cfgStore:   config.NewStoreForTesting(cfg),
		pluginsDir: t.TempDir(), // empty — Reconcile's launch attempt for "slow" fails deterministically
		logger:     hclog.NewNullLogger(),
		baseCtx:    context.Background(),
		host:       &pluginhost.Host{},
		cfg:        cfg,
	}
	engine := &correlate.Engine{Store: idx, Config: cfg}
	s.coord = syncer.NewCoordinator(idx, engine, []correlate.Source{blocker})
	s.startScheduler(cfg)

	select {
	case <-blocker.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("blocking source's Match was never called — the scheduler's immediate first refresh did not fire")
	}

	applyErr := make(chan error, 1)
	go func() { applyErr <- s.Apply(context.Background()) }()

	select {
	case err := <-applyErr:
		if err == nil {
			t.Fatal("expected Apply to fail (Reconcile's launch of a nonexistent plugin binary) — a nil error would mean this test's own setup is not exercising Reconcile at all")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Apply did not return in time — it must not block forever waiting on a source whose context it already cancelled")
	}

	runs, err := idx.LatestSyncRunPerSource(context.Background())
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource: %v", err)
	}
	run := runs["slow"]
	if run.Status == "running" {
		t.Errorf("expected the mid-flight sync to be finalised (not left at status \"running\") by the time Apply returns, got: %+v", run)
	}
	if run.FinishedUnix == 0 {
		t.Errorf("expected the mid-flight sync run to carry a finished time, got: %+v", run)
	}
}

// seedRemovedInstanceHistory seeds one items row and one finished sync_runs
// row for instanceID within webspaceName, so a test can prove BOTH tables
// were cleaned by the D-07 cleanup, not just items (07-VERIFICATION.md
// gaps[0].missing[2] — no existing test asserts the sync_runs half of
// T-07-13 at all).
func seedRemovedInstanceHistory(t *testing.T, idx *index.Store, ctx context.Context, webspaceName, instanceID string) {
	t.Helper()
	if err := idx.ReplaceWebspaceSourceItems(ctx, webspaceName, instanceID, []item.Item{testFixtureItem(instanceID, "1")}); err != nil {
		t.Fatalf("seed items for %s: %v", instanceID, err)
	}
	runID, err := idx.StartSyncRun(ctx, instanceID)
	if err != nil {
		t.Fatalf("StartSyncRun for %s: %v", instanceID, err)
	}
	if err := idx.FinishSyncRun(ctx, runID, "ok", "", 1); err != nil {
		t.Fatalf("FinishSyncRun for %s: %v", instanceID, err)
	}
}

// pluginByName scans plugins for the one launched under instance id name,
// so the tests below can assert on it without repeating an index loop.
func pluginByName(plugins []*pluginhost.Plugin, name string) (*pluginhost.Plugin, bool) {
	for _, p := range plugins {
		if p.Name() == name {
			return p, true
		}
	}
	return nil, false
}

// TestApply_ValidateMatchConfigFailsAfterReconcile_CoordinatorTracksRelaunchedPlugin
// exercises the exact ordering 07-VERIFICATION.md gaps[0].missing[1] names
// as uncovered: Host.Reconcile succeeds (it commits its result by mutating
// the launched plugin set in place) and pluginhost.ValidateMatchConfig then
// fails on the SAME Apply call. This is the branch
// TestApply_MidFlightSyncLeavesNoStrandedRunningRow deliberately never
// reaches — that test forces Reconcile itself to fail, so the host is never
// mutated at all. Here Reconcile succeeds and the vocabulary check is what
// rejects the save, which is the exact shape gaps[0] describes: a rejected
// save must not leave s.coord (and s.cfg) disagreeing with the host that
// Reconcile already committed.
func TestApply_ValidateMatchConfigFailsAfterReconcile_CoordinatorTracksRelaunchedPlugin(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	// "changing" is the instance whose connection config will change in the
	// rejected save below. "control" is the untouched control instance,
	// named by the second webspace's sources allowlist and match block.
	cfgStore := newTestConfigStore(t, `
[sources.changing]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"
display_name = "before"

[sources.control]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.everything]
keywords = ["labels"]

[webspaces.control-only]
sources = ["control"]

[webspaces.control-only.match.control]
labels = ["demo"]
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, dir, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	beforePlugin, ok := pluginByName(sup.Host().Plugins(), "changing")
	if !ok {
		t.Fatalf("expected instance %q launched at boot", "changing")
	}
	beforeCoord := sup.Coordinator()

	// next makes two changes at once, because it is their combination that
	// reaches the defect: the "changing" instance's config.Source differs
	// from what it was launched with (Reconcile must relaunch it), and the
	// "control-only" webspace's match block for the CONTROL instance now
	// declares a field name outside the mock plugin's declared vocabulary
	// ("labels" is the only field the mock declares).
	next := &config.Config{
		Sources: map[string]config.Source{
			"changing": {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused", DisplayName: "after"},
			"control":  {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused"},
		},
		Webspaces: map[string]config.Webspace{
			"everything": {Keywords: []string{"labels"}},
			"control-only": {
				Sources: []string{"control"},
				Match: map[string]config.MatchBlock{
					"control": {"nonexistent_field": []string{"demo"}},
				},
			},
		},
	}
	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save must succeed — config.Validate is deliberately plugin-independent, so an unknown match FIELD NAME must pass it. If this assertion fires, the test's premise has been invalidated by a change elsewhere and this test is no longer covering the branch it claims to: %v", err)
	}

	err = sup.Apply(ctx)
	if err == nil {
		t.Fatal("expected Apply to return a non-nil error from the vocabulary check")
	}
	if !strings.Contains(err.Error(), "nonexistent_field") || !strings.Contains(err.Error(), "control-only") {
		t.Errorf("expected the error to name the foreign match field and the webspace that declared it — this is what proves the vocabulary check produced the error rather than Reconcile (a Reconcile error would name a plugin binary instead). Got: %v", err)
	}

	plugins := sup.Host().Plugins()
	if len(plugins) != 2 {
		t.Fatalf("expected 2 launched plugins after apply, got %d", len(plugins))
	}
	afterPlugin, ok := pluginByName(plugins, "changing")
	if !ok {
		t.Fatalf("expected instance %q still launched after apply", "changing")
	}
	if afterPlugin == beforePlugin {
		t.Fatal("expected a NEW *Plugin pointer for the changed instance after Reconcile committed — this pins the premise that Reconcile genuinely committed, so the test cannot silently degrade into asserting nothing")
	}
	if got := afterPlugin.DisplayName(); got != "after" {
		t.Errorf("expected the relaunched instance to reflect the new display name, got %q", got)
	}

	if sup.Coordinator() == beforeCoord {
		t.Error("expected the coordinator to have been rebuilt (a different pointer) even though Apply returned an error")
	}

	if sup.cfg != cfgStore.Expanded() {
		t.Error("expected the supervisor's own recorded config generation (s.cfg) to be the same pointer cfgStore.Expanded() now returns")
	}

	// The real point of the test: the coordinator must dispatch against the
	// subprocess that is actually alive, not the one Reconcile already
	// killed.
	result, err := sup.Refresh(ctx, "changing")
	if err != nil {
		t.Fatalf("gaps[0]: expected Refresh against the relaunched instance to succeed after a rejected apply, got error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("gaps[0]: expected Refresh's Status to be \"ok\" after a rejected apply — a non-ok status here means the coordinator is dispatching syncs against the subprocess Host.Reconcile already killed, silently breaking this source's sync until some later apply happens to succeed all the way through. Got: %+v", result)
	}
	// Coalesced is tolerated either way: the scheduler generation the apply
	// just started also fires an immediate refresh for a changed instance,
	// and Refresh is documented as single-flight, so joining an in-flight
	// run is a legitimate outcome here and must not be asserted against.
}

// TestApply_RejectedSaveIsIdempotent_SecondApplyDoesNotRelaunchSubprocesses
// is the UI-12 idempotency edge: applying the same already-rejected config a
// second time — the documented POST /api/config/reload recovery path, which
// re-reads the same still-invalid file — fails again with the same error,
// and launches nothing and kills nothing, because the previous apply
// already adopted the new generation and Host.Reconcile finds every
// instance's source already equal to what the config declares.
func TestApply_RejectedSaveIsIdempotent_SecondApplyDoesNotRelaunchSubprocesses(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.changing]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"
display_name = "before"

[sources.control]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.everything]
keywords = ["labels"]

[webspaces.control-only]
sources = ["control"]

[webspaces.control-only.match.control]
labels = ["demo"]
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, dir, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	next := &config.Config{
		Sources: map[string]config.Source{
			"changing": {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused", DisplayName: "after"},
			"control":  {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused"},
		},
		Webspaces: map[string]config.Webspace{
			"everything": {Keywords: []string{"labels"}},
			"control-only": {
				Sources: []string{"control"},
				Match: map[string]config.MatchBlock{
					"control": {"nonexistent_field": []string{"demo"}},
				},
			},
		},
	}
	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	firstErr := sup.Apply(ctx)
	if firstErr == nil {
		t.Fatal("expected the first Apply to be rejected by the vocabulary check")
	}

	pluginsAfterFirst := sup.Host().Plugins()
	pointers := make(map[string]*pluginhost.Plugin, len(pluginsAfterFirst))
	for _, p := range pluginsAfterFirst {
		pointers[p.Name()] = p
	}

	// This is the documented recovery path: POST /api/config/reload over
	// the same still-invalid file, with no intervening save. It is a cheap
	// no-op reconcile precisely because the previous apply adopted the new
	// generation, so Host.Reconcile finds every instance's config.Source
	// already equal to what the config declares.
	secondErr := sup.Apply(ctx)
	if secondErr == nil {
		t.Fatal("expected the second Apply (retrying the same rejected save) to fail again")
	}
	if secondErr.Error() != firstErr.Error() {
		t.Errorf("expected the second Apply's error to match the first verbatim (same rejected save, same failure), got first=%q second=%q", firstErr, secondErr)
	}

	pluginsAfterSecond := sup.Host().Plugins()
	if len(pluginsAfterSecond) != len(pluginsAfterFirst) {
		t.Fatalf("expected the same number of launched plugins across both applies, got %d then %d", len(pluginsAfterFirst), len(pluginsAfterSecond))
	}
	for _, p := range pluginsAfterSecond {
		prior, ok := pointers[p.Name()]
		if !ok {
			t.Errorf("instance %q present after the second apply but not the first", p.Name())
			continue
		}
		if p != prior {
			t.Errorf("expected instance %q's *Plugin pointer to be identical across the retry (a no-op reconcile launches nothing and kills nothing), got a different pointer", p.Name())
		}
	}
}

// TestApply_RemovedInstanceCleanedUpEvenWhenTheSameSaveIsRejected exercises
// the exact combination 07-VERIFICATION.md gaps[0].missing[2] names as
// uncovered: one Apply call that both removes a source instance AND is
// rejected by pluginhost.ValidateMatchConfig for an unrelated reason. It is
// deliberately the intersection of
// TestApply_RemovedInstance_PluginGoneAndIndexRowsGone (removes an instance,
// but the save always succeeds) and
// TestApply_ValidateMatchConfigFailsAfterReconcile_CoordinatorTracksRelaunchedPlugin
// (the save is rejected, but nothing is ever removed) — neither existing
// test alone reaches the branch this one proves: that the D-07 cleanup
// still runs to completion, for BOTH items and sync_runs, even when the
// same Apply call is about to return a non-nil error.
func TestApply_RemovedInstanceCleanedUpEvenWhenTheSameSaveIsRejected(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.removed]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.survivor]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.everything]
keywords = ["labels"]

[webspaces.control-only]
sources = ["survivor"]

[webspaces.control-only.match.survivor]
labels = ["demo"]
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, dir, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	if len(sup.Host().Plugins()) != 2 {
		t.Fatalf("expected 2 launched plugins at boot, got %d", len(sup.Host().Plugins()))
	}

	seedRemovedInstanceHistory(t, idx, ctx, "everything", "removed")
	seedRemovedInstanceHistory(t, idx, ctx, "everything", "survivor")

	// Pre-assertion: both instances have a seeded run BEFORE apply, so the
	// post-apply assertion below cannot pass vacuously against rows that
	// were never there in the first place.
	preRuns, err := idx.LatestSyncRunPerSource(ctx)
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource (pre-apply): %v", err)
	}
	if _, ok := preRuns["removed"]; !ok {
		t.Fatal("expected a seeded sync run for \"removed\" before Apply")
	}
	if _, ok := preRuns["survivor"]; !ok {
		t.Fatal("expected a seeded sync run for \"survivor\" before Apply")
	}

	// next makes two changes at once, because it is their combination that
	// reaches the defect:
	//   - "removed" is absent from next.Sources entirely, and absent from
	//     every webspace's sources allowlist and match block;
	//   - "control-only"'s match block for the SURVIVOR now declares a field
	//     name outside the mock plugin's declared vocabulary ("labels" is
	//     the only field the mock plugin declares).
	next := &config.Config{
		Sources: map[string]config.Source{
			"survivor": {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused"},
		},
		Webspaces: map[string]config.Webspace{
			"everything": {Keywords: []string{"labels"}},
			"control-only": {
				Sources: []string{"survivor"},
				Match: map[string]config.MatchBlock{
					"survivor": {"nonexistent_field": []string{"demo"}},
				},
			},
		},
	}
	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save must succeed — config.Validate is deliberately plugin-independent, so an unknown match FIELD NAME must pass it. If this assertion fires, the test's premise has been invalidated by a change elsewhere and this test is no longer covering the branch it claims to: %v", err)
	}

	applyErr := sup.Apply(ctx)
	if applyErr == nil {
		t.Fatal("expected Apply to return a non-nil error from the vocabulary check")
	}
	if !strings.Contains(applyErr.Error(), "nonexistent_field") || !strings.Contains(applyErr.Error(), "control-only") {
		t.Errorf("expected the error to name the foreign match field and the webspace that declared it — this proves the vocabulary check produced it, rather than Reconcile (a Reconcile error would name a plugin binary instead). Got: %v", applyErr)
	}

	plugins := sup.Host().Plugins()
	if len(plugins) != 1 || plugins[0].Name() != "survivor" {
		t.Fatalf("expected only the \"survivor\" instance to remain launched after Apply (pinning that Reconcile genuinely committed the removal) — got %+v", plugins)
	}

	if _, ok, err := idx.GetItem(ctx, "removed:1"); err != nil {
		t.Fatalf("GetItem(removed:1): %v", err)
	} else if ok {
		t.Error("gaps[0]/T-07-13: expected the removed instance's items row to be gone after Apply — a present row here means the cleanup was skipped by the vocabulary rejection, and it can never run again because s.cfg has already advanced past the removal on this same Apply call")
	}

	postRuns, err := idx.LatestSyncRunPerSource(ctx)
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource (post-apply): %v", err)
	}
	if _, ok := postRuns["removed"]; ok {
		t.Error("gaps[0]/T-07-13: expected the removed instance's sync_runs history to be gone after Apply — this is the sync-history half of T-07-13 that no existing test asserts")
	}

	if _, ok, err := idx.GetItem(ctx, "survivor:1"); err != nil {
		t.Fatalf("GetItem(survivor:1): %v", err)
	} else if !ok {
		t.Error("expected the surviving instance's items row to be untouched by Apply — the cleanup must delete exactly the removed instance's data and nothing else")
	}
	if _, ok := postRuns["survivor"]; !ok {
		t.Error("expected the surviving instance's sync_runs entry to be untouched by Apply")
	}

	if sup.cfg != cfgStore.Expanded() {
		t.Error("expected the supervisor's own recorded config generation (s.cfg) to be the same pointer cfgStore.Expanded() now returns — a rejected save is state repair, not success, but the new generation must still be adopted (07-09's invariant, unweakened)")
	}
}
