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
//
// exited (08-09-PLAN.md Task 2) is closed, at most once (guarded by
// closeExitedOnce), when Match returns — proving a dispatched call
// genuinely observed its context's cancellation and unblocked, rather than
// merely never having been reached at all. Only closed when non-nil, so
// the two prior tests' fixture values (which never set it) keep compiling
// and behaving identically.
type blockingSource struct {
	name             string
	entered          chan struct{}
	closeEnteredOnce sync.Once
	exited           chan struct{}
	closeExitedOnce  sync.Once
}

func (b *blockingSource) Name() string              { return b.name }
func (b *blockingSource) SourceType() string        { return "slow" }
func (b *blockingSource) MatchVocabulary() []string { return []string{"keywords"} }
func (b *blockingSource) Match(ctx context.Context, _ map[string][]string) (*toposv1.MatchResponse, error) {
	b.closeEnteredOnce.Do(func() { close(b.entered) })
	<-ctx.Done()
	if b.exited != nil {
		b.closeExitedOnce.Do(func() { close(b.exited) })
	}
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
//
// THE ASSERTION MUST ADDRESS ONE SPECIFIC RUN, NOT THE SOURCE'S LATEST ONE
// (.planning/debug/resolved/apply-midflight-sync-race.md): Apply's
// pre-Reconcile failure branch — the branch this test deliberately drives —
// correctly restarts the OLD generation via startScheduler(oldCfg) BEFORE
// returning, and syncer.Scheduler.Run fires every configured source's
// immediate first refresh, which makes Coordinator.syncOne insert a SECOND
// "running" sync_runs row for "slow" before Match is even called. The
// shared blocker fixture then parks that second run on the new, uncancelled
// generation's context, so it stays "running" for the rest of the test.
// Asserting through idx.LatestSyncRunPerSource (MAX(id) per source) read
// whichever of those two rows won a scheduling race, and reported the
// perfectly healthy second run as a stranded mid-flight sync roughly a
// quarter of the time under -race. SyncRunsForSourceForTesting is ordered
// by id, so runs[0] is deterministically the run that was in flight when
// Apply was called — the only run this test's must_have is about. The pin
// is not weakened by the change: a genuinely stranded mid-flight sync
// leaves runs[0] at status "running" with no finished time and still
// fails here.
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

	// Stop whatever generation is running when this test ends. Without it the
	// generation Apply restarts below is left with a Match parked forever on
	// an uncancelled context: one leaked goroutine and one open index handle
	// per iteration, which is exactly what made this test's own historic
	// flakiness worsen with -count and under full-package parallelism.
	t.Cleanup(s.Shutdown)

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

	runs, err := idx.SyncRunsForSourceForTesting(context.Background(), "slow")
	if err != nil {
		t.Fatalf("SyncRunsForSourceForTesting: %v", err)
	}
	if len(runs) == 0 {
		t.Fatal("expected the mid-flight sync to have recorded a sync_runs row before Apply was called — this test's own setup is not exercising a mid-flight sync at all")
	}
	// runs[0] is the mid-flight run: it was started by the FIRST generation's
	// immediate refresh, which the blocker.entered wait above already proved
	// had begun before Apply was ever called, and sync_runs ids are monotonic
	// in insertion order. See this test's doc comment for why the source's
	// LATEST run is the wrong row to look at here.
	midFlight := runs[0]
	if midFlight.Status == "running" {
		t.Errorf("expected the mid-flight sync to be finalised (not left at status \"running\") by the time Apply returns, got: %+v", midFlight)
	}
	if midFlight.FinishedUnix == 0 {
		t.Errorf("expected the mid-flight sync run to carry a finished time, got: %+v", midFlight)
	}
}

// TestApply_EagerResyncDoesNotOutliveItsGeneration proves the
// generation-scoping fix behind Apply's eager-resync dispatch
// (08-09-PLAN.md Task 2, closing 08-UAT.md G-08-3's untracked-goroutine
// sibling): a background sync dispatched into a scheduler generation is
// bounded by THAT generation's own cancellable context and tracked by its
// wait group, so stopScheduler — now reachable from the WhatsApp
// link-start HTTP request path via SuspendInstance (08-09-PLAN.md Task 1)
// — can never block forever on a sync it has no way to cancel.
//
// The dispatch is driven against s.coord/s.genCtx/s.genWG — the exact
// triple Apply's own post-commitGeneration dispatch loop reads under s.mu
// — rather than through a literal Apply(ctx) call reaching that loop for
// blockingSource specifically: commitGeneration always rebuilds the
// coordinator from s.host.Plugins() (kernel/pluginhost.Host has no seam
// for an in-memory fake source), so a genuinely successful Apply can only
// ever route its own dispatch through a REAL launched plugin subprocess —
// and this plan's file scope has no seam to make a real subprocess block
// on demand (inventing one would require changing a plugins/ file, outside
// files_modified). Driving the same fields with the same s.mu discipline
// Apply's own dispatch loop uses is the faithful proxy: it proves
// genCtx/genWG bound a dispatched sync identically to how Apply's own loop
// would.
func TestApply_EagerResyncDoesNotOutliveItsGeneration(t *testing.T) {
	idx := newTestIndex(t)
	blocker := &blockingSource{name: "slow", entered: make(chan struct{}), exited: make(chan struct{})}
	// A participating webspace is required: correlate.Engine.SyncSource only
	// calls Match for a (webspace, source) pair that participates (see
	// suspend_test.go's identical note) — with no webspace at all, Match is
	// never called and blocker.entered would never close.
	engineCfg := &config.Config{Webspaces: map[string]config.Webspace{"demo": {Keywords: []string{"keywords"}}}}
	engine := &correlate.Engine{Store: idx, Config: engineCfg}

	// No configured sources at all: Scheduler.Run's own goroutine set is
	// empty, so it cannot race the dispatch under test for blocker — the
	// ONLY caller of blocker.Match in this test is the manually-dispatched
	// goroutine below.
	cfg := &config.Config{}
	s := &Supervisor{
		idx:      idx,
		cfgStore: config.NewStoreForTesting(cfg),
		logger:   hclog.NewNullLogger(),
		baseCtx:  context.Background(),
		host:     &pluginhost.Host{},
		cfg:      cfg,
	}
	s.coord = syncer.NewCoordinator(idx, engine, []correlate.Source{blocker})
	s.startScheduler(cfg)

	// Mirror Apply's own eager-resync dispatch exactly
	// (kernel/supervisor/supervisor.go): read coord/genCtx/genWG under
	// s.mu, register the goroutine on genWG BEFORE it starts, and call
	// coord.Refresh with genCtx — never a detached context.Background().
	s.mu.Lock()
	coord := s.coord
	genCtx := s.genCtx
	genWG := s.genWG
	s.mu.Unlock()

	genWG.Add(1)
	go func() {
		defer genWG.Done()
		coord.Refresh(genCtx, "slow")
	}()

	select {
	case <-blocker.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("the dispatched resync's Match was never called")
	}

	shutdownDone := make(chan struct{})
	go func() {
		s.Shutdown()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not return within its deadline — an eager-resync goroutine dispatched with an uncancellable context blocks stopScheduler forever, which would now hang the WhatsApp link-start HTTP request path (stopScheduler is reachable from SuspendInstance)")
	}

	select {
	case <-blocker.exited:
	default:
		t.Error("expected the dispatched resync's Match call to have returned (its context cancelled) by the time Shutdown completed")
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

// TestApply_MultipleRemovedInstances_OneCleanupFailureDoesNotAbandonTheRest
// covers 07-VERIFICATION.md gaps[0].missing[1] — the second, related failure
// mode in the same loop: an early instance's delete error must not abandon
// cleanup for every later-sorted instance in the same batch, which strands
// them permanently for the identical reason gaps[0]'s primary failure mode
// does (s.cfg advances past the removal on this same Apply call regardless
// of the cleanup's own outcome).
//
// The failure lever is the index store itself: idx.Close() is called after
// the save and before Apply, so every DeleteSourceItems/DeleteSyncRuns call
// returns an error. This lever is used because Supervisor.idx is a
// concrete *index.Store, not an interface — forcing a per-instance SQL
// failure any other way would require introducing an interface seam that is
// out of scope for a gap repair. newTestIndex also closes the store in its
// own t.Cleanup, which is safe because database/sql's Close is idempotent.
func TestApply_MultipleRemovedInstances_OneCleanupFailureDoesNotAbandonTheRest(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	// "alpha" and "zulu" are both removed — their sorted order is
	// unambiguous (alpha < zulu) and both ids are distinctive enough to
	// assert on by substring. "keep" is the survivor.
	cfgStore := newTestConfigStore(t, `
[sources.alpha]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.zulu]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.keep]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.everything]
keywords = ["labels"]
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, dir, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	if len(sup.Host().Plugins()) != 3 {
		t.Fatalf("expected 3 launched plugins at boot, got %d", len(sup.Host().Plugins()))
	}

	seedRemovedInstanceHistory(t, idx, ctx, "everything", "alpha")
	seedRemovedInstanceHistory(t, idx, ctx, "everything", "zulu")
	seedRemovedInstanceHistory(t, idx, ctx, "everything", "keep")

	// next keeps only the survivor. No dangling match-block or allowlist
	// references to either removed instance, and every match block uses
	// only the mock plugin's declared vocabulary field ("labels") — this
	// test must be rejected by the CLEANUP, not by the vocabulary check, so
	// the config is otherwise entirely valid.
	next := &config.Config{
		Sources: map[string]config.Source{
			"keep": {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused"},
		},
		Webspaces: map[string]config.Webspace{
			"everything": {Keywords: []string{"labels"}},
		},
	}
	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// The failure lever: close the index store so every
	// DeleteSourceItems/DeleteSyncRuns call the cleanup makes returns an
	// error.
	if err := idx.Close(); err != nil {
		t.Fatalf("idx.Close (failure lever): %v", err)
	}

	applyErr := sup.Apply(ctx)
	if applyErr == nil {
		t.Fatal("expected Apply to return a non-nil error — every removed instance's cleanup delete fails against a closed index store")
	}

	msg := applyErr.Error()
	if !strings.Contains(msg, "alpha") || !strings.Contains(msg, "zulu") {
		t.Fatalf("expected the error to name BOTH removed instances — a message naming only the first sorted instance means the loop still returns on its first failure and every later instance in the batch is being abandoned with no retry path, since s.cfg advances on this same call. Got: %v", applyErr)
	}
	if !strings.Contains(msg, "delete items for removed source") {
		t.Errorf("expected the error to carry the \"delete items for removed source\" phrasing, confirming the failures came from the cleanup rather than from anywhere else. Got: %v", applyErr)
	}

	plugins := sup.Host().Plugins()
	if len(plugins) != 1 || plugins[0].Name() != "keep" {
		t.Fatalf("expected only the survivor \"keep\" to remain launched after Apply (pinning that Reconcile committed and the cleanup was genuinely reached, so this test cannot pass vacuously on an error raised before the cleanup) — got %+v", plugins)
	}
}

// The five tests below drive 07-16-PLAN.md Task 2's synchronous purge.
// Every seed uses testFixtureItem directly against idx.ReplaceWebspaceSourceItems
// (the same helper/pattern TestApply_RemovedInstance_PluginGoneAndIndexRowsGone
// above uses) rather than depending on a real plugin Match RPC's timing:
// NewSupervisor's own boot-time scheduler starts an immediate, genuinely
// asynchronous refresh for every configured source (syncer/scheduler.go's
// runSource), and calling Apply immediately afterward — as every test below
// does, deliberately, with no sleep — cancels that in-flight boot
// generation's context via stopScheduler before its own Match RPC
// necessarily has time to complete or persist anything. A test that relied
// on that racing boot sync to establish its "before" state would be
// flaky by construction. Seeding directly sidesteps this entirely: it is a
// pure local index write with no dependency on any goroutine's timing, and
// it uses source id "1" with plugins/mock's own real fixture id scheme, so
// if the launched mock plugin's OWN background refresh (boot's or a later
// generation's) ever does win a race and rewrites the same (webspace,
// source) pair, it produces the byte-identical item id — never a
// disagreement the assertions below would need to arbitrate.
//
// Assertions check for presence/absence of specific item ids rather than
// exact row counts, for the same reason: a still-participating pair may
// legitimately end up with MORE than the one seeded row if a real
// background sync happens to land during the test (plugins/mock's fixture
// set has four items, all labelled "demo") — that is harmless noise this
// test does not care about, whereas the seeded id's presence or absence is
// exactly what the purge does or does not control.

// TestApply_PurgesDeparticipatedWebspaceRows_NarrowingClearsOnlyTheFlippedPair
// is the core case (07-16-PLAN.md Task 2): narrowing ONE webspace to
// exclude ONE still-configured instance clears exactly that pair's rows,
// synchronously, by the time Apply returns — leaving the still-
// participating instance's rows in the SAME webspace untouched, the
// excluded instance's rows in every OTHER webspace it still participates
// in untouched, and the excluded instance's own items rows (the hybrid
// data model's own record, never the join) untouched.
func TestApply_PurgesDeparticipatedWebspaceRows_NarrowingClearsOnlyTheFlippedPair(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.alpha]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.beta]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.ws1]
keywords = ["demo"]

[webspaces.ws2]
keywords = ["demo"]
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, dir, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	// Seed both instances' contribution to both webspaces directly — see
	// this block's own doc comment for why this sidesteps the boot
	// scheduler's own racing initial refresh entirely.
	if err := idx.ReplaceWebspaceSourceItems(ctx, "ws1", "alpha", []item.Item{testFixtureItem("alpha", "1")}); err != nil {
		t.Fatalf("seed ws1/alpha: %v", err)
	}
	if err := idx.ReplaceWebspaceSourceItems(ctx, "ws1", "beta", []item.Item{testFixtureItem("beta", "1")}); err != nil {
		t.Fatalf("seed ws1/beta: %v", err)
	}
	if err := idx.ReplaceWebspaceSourceItems(ctx, "ws2", "alpha", []item.Item{testFixtureItem("alpha", "1")}); err != nil {
		t.Fatalf("seed ws2/alpha: %v", err)
	}
	if err := idx.ReplaceWebspaceSourceItems(ctx, "ws2", "beta", []item.Item{testFixtureItem("beta", "1")}); err != nil {
		t.Fatalf("seed ws2/beta: %v", err)
	}

	// Narrow ws1 to exclude beta; ws2 is untouched. Both sources' own
	// connection config is byte-identical to what they were launched with,
	// so Reconcile relaunches nothing — the only thing that changed is
	// participation.
	next := &config.Config{
		Sources: map[string]config.Source{
			"alpha": {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused"},
			"beta":  {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused"},
		},
		Webspaces: map[string]config.Webspace{
			"ws1": {Keywords: []string{"demo"}, Sources: []string{"alpha"}},
			"ws2": {Keywords: []string{"demo"}},
		},
	}
	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := sup.Apply(ctx); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// The narrowed webspace: alpha's row survives, beta's is gone —
	// asserted on the statement immediately after Apply returns, no sleep,
	// no eventually-loop.
	ws1Items, err := idx.StreamItems(ctx, "ws1", nil)
	if err != nil {
		t.Fatalf("StreamItems(ws1): %v", err)
	}
	ws1IDs := idsOfSupervisorTest(ws1Items)
	if !ws1IDs[item.ID("alpha", "1")] {
		t.Errorf("expected ws1 to still contain alpha's item (still-participating instance keeps every row in the narrowed webspace), got: %v", ws1Items)
	}
	if ws1IDs[item.ID("beta", "1")] {
		t.Errorf("expected ws1 to no longer contain beta's item — beta was excluded by the narrowed allowlist and its rows must be purged by the time Apply returns, got: %v", ws1Items)
	}

	// The OTHER webspace: both alpha's and beta's rows are untouched — a
	// webspace narrowed for one instance must never widen into a deletion
	// for that instance's rows anywhere else.
	ws2Items, err := idx.StreamItems(ctx, "ws2", nil)
	if err != nil {
		t.Fatalf("StreamItems(ws2): %v", err)
	}
	ws2IDs := idsOfSupervisorTest(ws2Items)
	if !ws2IDs[item.ID("alpha", "1")] {
		t.Errorf("expected ws2 (untouched by the ws1 narrowing) to still contain alpha's item, got: %v", ws2Items)
	}
	if !ws2IDs[item.ID("beta", "1")] {
		t.Errorf("expected ws2 (untouched by the ws1 narrowing) to still contain beta's item, got: %v", ws2Items)
	}

	// beta's own items row (the hybrid data model's actual record) survives
	// — only the ws1/beta JOIN row was removed, never the item itself, so
	// re-adding beta to ws1 restores its items without a refetch from the
	// source system.
	if _, ok, err := idx.GetItem(ctx, item.ID("beta", "1")); err != nil {
		t.Fatalf("GetItem(beta:1): %v", err)
	} else if !ok {
		t.Error("expected beta's own item row to survive the purge — only the (ws1, beta) join row should have been removed")
	}
}

// TestApply_PurgesDeparticipatedWebspaceRows_LastSourceRemovedLeavesEmptyShellStreamingNothing
// covers the edge case named in 07-16-PLAN.md's UI-12/empty audit row:
// narrowing a webspace's LAST participating source turns it into a D-20
// empty shell, and every remaining pair (here, the only pair) is cleared —
// the webspace streams nothing, while the instance's own items row
// survives.
func TestApply_PurgesDeparticipatedWebspaceRows_LastSourceRemovedLeavesEmptyShellStreamingNothing(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.solo]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.shell-target]
keywords = ["demo"]
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, dir, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	if err := idx.ReplaceWebspaceSourceItems(ctx, "shell-target", "solo", []item.Item{testFixtureItem("solo", "1")}); err != nil {
		t.Fatalf("seed shell-target/solo: %v", err)
	}

	// Narrow shell-target to a D-20 empty shell (07-11-PLAN.md): declares
	// none of keywords/sources/match, a legitimate loadable config state.
	next := &config.Config{
		Sources: map[string]config.Source{
			"solo": {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused"},
		},
		Webspaces: map[string]config.Webspace{
			"shell-target": {},
		},
	}
	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := sup.Apply(ctx); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	items, err := idx.StreamItems(ctx, "shell-target", nil)
	if err != nil {
		t.Fatalf("StreamItems(shell-target): %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected shell-target to stream nothing once its last participating source was removed, got: %+v", items)
	}

	if _, ok, err := idx.GetItem(ctx, item.ID("solo", "1")); err != nil {
		t.Fatalf("GetItem(solo:1): %v", err)
	} else if !ok {
		t.Error("expected solo's own item row to survive turning shell-target into an empty shell")
	}
}

// TestApply_PurgesDeparticipatedWebspaceRows_NoOpConfigPerformsNoClear proves
// the UI-12 idempotency edge: an Apply where no pair's participation
// flipped must not clear anything. The assertion runs immediately after
// Apply returns — a buggy purge that incorrectly clears an unchanged pair
// would show as a MISSING row right here, synchronously (the purge itself
// runs inside Apply; a background resync, if any ever raced in afterward,
// could only ADD the same deterministic id back, never explain away an
// incorrect clear that already happened inside this very call).
func TestApply_PurgesDeparticipatedWebspaceRows_NoOpConfigPerformsNoClear(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.alpha]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.ws1]
keywords = ["demo"]
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, dir, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	if err := idx.ReplaceWebspaceSourceItems(ctx, "ws1", "alpha", []item.Item{testFixtureItem("alpha", "1")}); err != nil {
		t.Fatalf("seed ws1/alpha: %v", err)
	}

	// Save the exact same (expanded) config back — a genuine no-op from
	// the purge's own diff perspective: no webspace or instance name
	// changed, and no participation could have flipped.
	if err := cfgStore.Save(cfgStore.Expanded(), cfgStore.Hash()); err != nil {
		t.Fatalf("Save (no-op): %v", err)
	}

	if err := sup.Apply(ctx); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	items, err := idx.StreamItems(ctx, "ws1", nil)
	if err != nil {
		t.Fatalf("StreamItems(ws1): %v", err)
	}
	if !idsOfSupervisorTest(items)[item.ID("alpha", "1")] {
		t.Errorf("expected ws1 to still contain alpha's item after a no-op apply (no participation flipped, so nothing should have been cleared), got: %v", items)
	}
}

// TestApply_PurgesDeparticipatedWebspaceRows_DeletedWebspaceRowsUntouched
// proves the prohibition guarding against the worse-than-the-defect
// failure mode: a webspace removed from the config entirely is OUT OF the
// purge's diff scope (the intersection of both configs' webspace names),
// so its rows are left exactly as they were — never touched, never
// resurrected through a clear-then-upsert. Removing "doomed" from newCfg
// gives ParticipatesIn(newCfg.Webspaces["doomed"], "alpha") the same false
// answer a real deletion produces (an absent key reads as the Webspace
// zero value); if the purge's diff were scoped to old webspace names
// alone rather than the intersection, this would look exactly like a
// true-to-false flip and wipe "doomed"'s rows — which is the case this
// test exists to catch.
func TestApply_PurgesDeparticipatedWebspaceRows_DeletedWebspaceRowsUntouched(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.alpha]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.ws1]
keywords = ["demo"]

[webspaces.doomed]
keywords = ["demo"]
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, dir, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	if err := idx.ReplaceWebspaceSourceItems(ctx, "doomed", "alpha", []item.Item{testFixtureItem("alpha", "1")}); err != nil {
		t.Fatalf("seed doomed/alpha: %v", err)
	}

	next := &config.Config{
		Sources: map[string]config.Source{
			"alpha": {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused"},
		},
		Webspaces: map[string]config.Webspace{
			"ws1": {Keywords: []string{"demo"}},
		},
	}
	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := sup.Apply(ctx); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	items, err := idx.StreamItems(ctx, "doomed", nil)
	if err != nil {
		t.Fatalf("StreamItems(doomed): %v", err)
	}
	if !idsOfSupervisorTest(items)[item.ID("alpha", "1")] {
		t.Errorf("expected doomed's rows to survive being deleted from the config entirely (out of the purge's diff scope), got: %v", items)
	}
}

// TestApply_PurgesDeparticipatedWebspaceRows_FailureIsJoinedIntoApplyError
// proves the purge's failure handling mirrors cleanupRemovedInstances
// exactly: a per-pair clear failure is collected and named (webspace and
// instance), never returned early, so both of two simultaneously-flipped
// pairs in the same webspace are reported rather than the loop abandoning
// the second after the first fails.
func TestApply_PurgesDeparticipatedWebspaceRows_FailureIsJoinedIntoApplyError(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.alpha]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.beta]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.everything]
keywords = ["demo"]
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, dir, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	// Turn "everything" into a D-20 empty shell: BOTH alpha and beta flip
	// from participating to not, in the same webspace, in the same Apply
	// call.
	next := &config.Config{
		Sources: map[string]config.Source{
			"alpha": {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused"},
			"beta":  {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused"},
		},
		Webspaces: map[string]config.Webspace{
			"everything": {},
		},
	}
	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// The failure lever: close the index store so every
	// ReplaceWebspaceSourceItems clear call the purge makes returns an
	// error — same lever TestApply_MultipleRemovedInstances_... uses for
	// the cleanup's own batching test.
	if err := idx.Close(); err != nil {
		t.Fatalf("idx.Close (failure lever): %v", err)
	}

	applyErr := sup.Apply(ctx)
	if applyErr == nil {
		t.Fatal("expected Apply to return a non-nil error — both clear calls fail against a closed index store")
	}

	msg := applyErr.Error()
	if !strings.Contains(msg, "everything") {
		t.Errorf("expected the error to name the webspace \"everything\", got: %v", applyErr)
	}
	if !strings.Contains(msg, "alpha") || !strings.Contains(msg, "beta") {
		t.Fatalf("expected the error to name BOTH flipped instances — naming only one means the loop returns early on its first failure and abandons the rest of the batch. Got: %v", applyErr)
	}
	if !strings.Contains(msg, "clear webspace") {
		t.Errorf("expected the error to carry the \"clear webspace\" phrasing, confirming the failures came from the purge rather than anywhere else. Got: %v", applyErr)
	}
}

// idsOfSupervisorTest mirrors kernel/correlate's own idsOf helper — kept
// local since these are different packages and idsOf is unexported.
func idsOfSupervisorTest(items []item.Item) map[string]bool {
	ids := make(map[string]bool, len(items))
	for _, it := range items {
		ids[it.ID] = true
	}
	return ids
}
