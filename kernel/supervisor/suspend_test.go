package supervisor

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/pluginhost"
	"github.com/davison/topos/kernel/syncer"
)

// TestSuspendInstance_AbsentNameIsNoOp proves the D-02 Add-Source case: a
// name not present in the running Host (the instance being linked has not
// been saved to config yet at all) returns a no-op resume and a nil
// error, so the caller needs no special-casing between "suspend a running
// instance" and "nothing to suspend."
func TestSuspendInstance_AbsentNameIsNoOp(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.keep]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, pluginhost.Dirs{Trusted: dir}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	resume, err := sup.SuspendInstance(ctx, "not-configured-at-all")
	if err != nil {
		t.Fatalf("SuspendInstance for an absent name must not error, got: %v", err)
	}
	if resume == nil {
		t.Fatal("SuspendInstance must return a non-nil resume closure even for a no-op")
	}
	if err := resume(ctx); err != nil {
		t.Fatalf("no-op resume must never itself error, got: %v", err)
	}

	// Untouched: the one real running instance is still launched.
	if got := len(sup.Host().Plugins()); got != 1 {
		t.Fatalf("expected the unrelated running instance to be untouched, got %d launched plugins", got)
	}
}

// TestSuspendInstance_StopsThenResumeRestarts proves the core mechanism: a
// named running instance's subprocess is stopped for the duration between
// SuspendInstance and calling its returned resume closure, and relaunched
// once resume runs — while a second, unrelated instance is never touched.
func TestSuspendInstance_StopsThenResumeRestarts(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.suspend-me]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.leave-alone]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, pluginhost.Dirs{Trusted: dir}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	if got := len(sup.Host().Plugins()); got != 2 {
		t.Fatalf("expected 2 launched plugins at boot, got %d", got)
	}

	resume, err := sup.SuspendInstance(ctx, "suspend-me")
	if err != nil {
		t.Fatalf("SuspendInstance: %v", err)
	}

	plugins := sup.Host().Plugins()
	if len(plugins) != 1 || plugins[0].Name() != "leave-alone" {
		t.Fatalf("expected only 'leave-alone' to remain launched while suspended, got %+v", namesOf(plugins))
	}

	if err := resume(ctx); err != nil {
		t.Fatalf("resume: %v", err)
	}

	resumed := sup.Host().Plugins()
	if len(resumed) != 2 {
		t.Fatalf("expected both instances launched again after resume, got %+v", namesOf(resumed))
	}
}

// TestApply_UnrelatedSaveSucceedsWhileAnInstanceIsSuspended is WR-02's
// (08-REVIEW.md) regression test: while "suspend-me" is suspended (a
// stand-in for an in-flight WhatsApp re-link session), an Apply for an
// otherwise-valid config save that is completely unrelated to "suspend-me"
// — it only changes "control"'s display_name — must still succeed. Before
// this fix, Apply's Reconcile call would see "suspend-me" still present in
// newCfg.Sources (SuspendInstance never touches config-of-record) but
// absent from the launched host, try to relaunch it, and — in a real
// deployment — lose the store-lock race against the live link subprocess;
// even without a real store-lock collision, the pre-fix
// pluginhost.ValidateMatchConfig call would independently reject this save
// because "demo"'s explicit match block still names "suspend-me", which
// genuinely has no launched plugin while suspended. Both failure shapes are
// covered here: an explicit match block for the suspended instance, and an
// unrelated instance's connection-config change forcing Reconcile to do
// real work in the same Apply call.
func TestApply_UnrelatedSaveSucceedsWhileAnInstanceIsSuspended(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.suspend-me]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.control]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.demo]
sources = ["suspend-me", "control"]

[webspaces.demo.match.suspend-me]
labels = ["demo"]

[webspaces.demo.match.control]
labels = ["demo"]
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, pluginhost.Dirs{Trusted: dir}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	if len(sup.Host().Plugins()) != 2 {
		t.Fatalf("expected 2 launched plugins at boot, got %d", len(sup.Host().Plugins()))
	}

	resume, err := sup.SuspendInstance(ctx, "suspend-me")
	if err != nil {
		t.Fatalf("SuspendInstance: %v", err)
	}

	plugins := sup.Host().Plugins()
	if len(plugins) != 1 || plugins[0].Name() != "control" {
		t.Fatalf("expected only 'control' to remain launched while suspended, got %+v", namesOf(plugins))
	}

	// An unrelated save: "suspend-me" and the "demo" webspace's match
	// block naming it are both left byte-identical; only "control"'s
	// display_name changes, forcing Reconcile to do real relaunch work in
	// the same Apply call this test drives.
	next := &config.Config{
		Sources: map[string]config.Source{
			"suspend-me": {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused"},
			"control":    {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused", DisplayName: "renamed"},
		},
		Webspaces: map[string]config.Webspace{
			"demo": {
				Sources: []string{"suspend-me", "control"},
				Match: map[string]config.MatchBlock{
					"suspend-me": {"labels": []string{"demo"}},
					"control":    {"labels": []string{"demo"}},
				},
			},
		},
	}
	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := sup.Apply(ctx); err != nil {
		t.Fatalf("expected Apply to succeed for a save unrelated to the suspended instance, got: %v", err)
	}

	afterApply := sup.Host().Plugins()
	if len(afterApply) != 1 || afterApply[0].Name() != "control" {
		t.Fatalf("expected 'suspend-me' to remain un-relaunched (still suspended) and only 'control' launched after Apply, got %+v", namesOf(afterApply))
	}
	if got := afterApply[0].DisplayName(); got != "renamed" {
		t.Errorf("expected the unrelated instance's config change to have actually applied, got display_name %q", got)
	}

	if err := resume(ctx); err != nil {
		t.Fatalf("resume after an interleaved Apply: %v", err)
	}

	resumed := sup.Host().Plugins()
	if len(resumed) != 2 {
		t.Fatalf("expected both instances launched again after resume, got %+v", namesOf(resumed))
	}
}

func namesOf(plugins []*pluginhost.Plugin) []string {
	names := make([]string, len(plugins))
	for i, p := range plugins {
		names[i] = p.Name()
	}
	return names
}

// TestSuspendInstance_ResumedInstanceStillSyncs is the regression test
// 08-UAT.md G-08-3 proved missing: before this fix, a source instance
// suspended for a WhatsApp link session and then resumed kept failing every
// subsequent sync with grpc-go's ErrClientConnClosing ("rpc error: code =
// Canceled desc = grpc: the client connection is closing") — the
// syncer.Coordinator captured its *pluginhost.Plugin handles once at
// construction, and neither SuspendInstance nor its resume closure ever
// rebuilt one, so every sync path went on calling Match through the
// go-plugin client Host.Reconcile had already Kill()ed.
//
// The fixture's "demo" webspace, carrying an explicit match block naming
// EACH instance, is the point of the test: correlate.Engine.SyncSource only
// calls Match for a (webspace, source) pair that participates, so a fixture
// with no participating webspace would never touch the plugin handle at
// all — the test would pass against the defective code while proving
// nothing.
func TestSuspendInstance_ResumedInstanceStillSyncs(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sync]
interval = "1h"

[sources.suspend-me]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.leave-alone]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.demo]
sources = ["suspend-me", "leave-alone"]

[webspaces.demo.match.suspend-me]
labels = ["demo"]

[webspaces.demo.match.leave-alone]
labels = ["demo"]
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, pluginhost.Dirs{Trusted: dir}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	if got := len(sup.Host().Plugins()); got != 2 {
		t.Fatalf("expected 2 launched plugins at boot, got %d", got)
	}

	resume, err := sup.SuspendInstance(ctx, "suspend-me")
	if err != nil {
		t.Fatalf("SuspendInstance: %v", err)
	}

	if err := resume(ctx); err != nil {
		t.Fatalf("resume: %v", err)
	}

	result, err := sup.Refresh(ctx, "suspend-me")
	if err != nil {
		t.Fatalf("Refresh(suspend-me) after resume: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected Status \"ok\" for a resumed instance's next refresh, got %q (error: %q) — a non-ok status here means the coordinator is still dispatching against the subprocess SuspendInstance already killed", result.Status, result.Error)
	}
	if result.Error != "" {
		t.Fatalf("expected an empty Error for a resumed instance's next refresh, got %q", result.Error)
	}

	leaveAloneResult, err := sup.Refresh(ctx, "leave-alone")
	if err != nil {
		t.Fatalf("Refresh(leave-alone): %v", err)
	}
	if leaveAloneResult.Status != "ok" {
		t.Fatalf("expected the untouched instance \"leave-alone\" to still sync successfully, got Status %q (error: %q)", leaveAloneResult.Status, leaveAloneResult.Error)
	}

	resumed := sup.Host().Plugins()
	if len(resumed) != 2 {
		t.Fatalf("expected both instances launched again after resume, got %+v", namesOf(resumed))
	}
}

// TestSuspendInstance_SuspendedWindowRecordsNoErroredRun is Task 3's Guard
// 1 (08-09-PLAN.md): a refresh issued while an instance is suspended must
// never write an errored sync_runs row for it — a kernel lifecycle
// artifact must never be pinned to a source's health surface as a failed
// sync. Reuses the participating-webspace fixture from
// TestSuspendInstance_ResumedInstanceStillSyncs, above.
func TestSuspendInstance_SuspendedWindowRecordsNoErroredRun(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sync]
interval = "1h"

[sources.suspend-me]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.leave-alone]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.demo]
sources = ["suspend-me", "leave-alone"]

[webspaces.demo.match.suspend-me]
labels = ["demo"]

[webspaces.demo.match.leave-alone]
labels = ["demo"]
`)

	sup, err := NewSupervisor(ctx, idx, cfgStore, pluginhost.Dirs{Trusted: dir}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	// Establish a real "ok" row as the instance's latest recorded outcome
	// before suspending it — this is what a row written during the
	// suspension window would need to overwrite, and what the final
	// assertion proves it never did.
	preResult, err := sup.Refresh(ctx, "suspend-me")
	if err != nil {
		t.Fatalf("Refresh(suspend-me) before suspend: %v", err)
	}
	if preResult.Status != "ok" {
		t.Fatalf("expected the pre-suspend refresh to succeed, got Status %q (error: %q)", preResult.Status, preResult.Error)
	}

	resume, err := sup.SuspendInstance(ctx, "suspend-me")
	if err != nil {
		t.Fatalf("SuspendInstance: %v", err)
	}

	// The coordinator has no entry for a suspended instance — Refresh must
	// answer ErrUnknownSource BEFORE a sync_runs row is ever started
	// (syncer.Coordinator.Refresh's own doc comment), never attempt (and
	// fail) a real sync against it.
	_, refreshErr := sup.Refresh(ctx, "suspend-me")
	if !errors.Is(refreshErr, syncer.ErrUnknownSource) {
		t.Fatalf("expected Refresh during the suspension window to return syncer.ErrUnknownSource, got: %v", refreshErr)
	}

	runs, err := idx.LatestSyncRunPerSource(ctx)
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource (during suspension): %v", err)
	}
	run, ok := runs["suspend-me"]
	if !ok {
		t.Fatal("expected a latest sync_runs row for \"suspend-me\" to still exist (the pre-suspend one)")
	}
	if run.Status != "ok" {
		t.Errorf("expected \"suspend-me\"'s latest sync_runs row to still report status \"ok\" during the suspension window — a row written here is what pinned the WhatsApp source's chip to a closing-connection error in 08-UAT.md test 3, and it survives restarts because nothing ever overwrites a latest-run row except a later run. Got status %q, error %q", run.Status, run.Error)
	}
	if run.Error != "" {
		t.Errorf("expected \"suspend-me\"'s latest sync_runs row to carry an empty error during the suspension window, got %q", run.Error)
	}

	if err := resume(ctx); err != nil {
		t.Fatalf("resume: %v", err)
	}

	postResult, err := sup.Refresh(ctx, "suspend-me")
	if err != nil {
		t.Fatalf("Refresh(suspend-me) after resume: %v", err)
	}
	if postResult.Status != "ok" {
		t.Fatalf("expected a further refresh after resume to record \"ok\" again, got Status %q (error: %q)", postResult.Status, postResult.Error)
	}
}
