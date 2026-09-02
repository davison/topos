package supervisor

import (
	"context"
	"github.com/davison/topos/kernel/pluginhost"
	"github.com/hashicorp/go-hclog"
	"strings"
	"testing"

	"github.com/davison/topos/kernel/config"
)

// detectWebspaceRenames (M3-R2, #77): exactly one vanished key paired
// with exactly one appeared key carrying a byte-identical body; anything
// else refuses — a wrong carry would be worse than a resync.
func TestDetectWebspaceRenames(t *testing.T) {
	ws := func(kw string) config.Webspace { return config.Webspace{Keywords: []string{kw}, Filter: []string{"f"}} }
	old := &config.Config{Webspaces: map[string]config.Webspace{"a": ws("x"), "keep": ws("k")}}
	renamed := &config.Config{Webspaces: map[string]config.Webspace{"b": ws("x"), "keep": ws("k")}}
	if got := detectWebspaceRenames(old, renamed); len(got) != 1 || got[0] != [2]string{"a", "b"} {
		t.Fatalf("clean rename: %v", got)
	}
	// A body edit riding the rename refuses.
	edited := &config.Config{Webspaces: map[string]config.Webspace{"b": ws("CHANGED"), "keep": ws("k")}}
	if got := detectWebspaceRenames(old, edited); got != nil {
		t.Fatalf("rename+edit must refuse: %v", got)
	}
	// Ambiguity (two vanished) refuses.
	old2 := &config.Config{Webspaces: map[string]config.Webspace{"a": ws("x"), "c": ws("x")}}
	new2 := &config.Config{Webspaces: map[string]config.Webspace{"b": ws("x")}}
	if got := detectWebspaceRenames(old2, new2); got != nil {
		t.Fatalf("ambiguous must refuse: %v", got)
	}
	// A plain add or delete pairs nothing.
	if got := detectWebspaceRenames(old, &config.Config{Webspaces: map[string]config.Webspace{"a": ws("x"), "keep": ws("k"), "new": ws("n")}}); got != nil {
		t.Fatalf("plain add must refuse: %v", got)
	}
}

// The migration-failure path (PR #80 review round 2): a failing
// RenameWebspace is an apply failure that STILL travels the
// post-Reconcile repair region — the generation commits (a retry works
// on the new base rather than re-diffing a destroyed old one) and the
// cleanup/purge are attempted, never skipped. The index is sabotaged by
// closing it after boot: RenameWebspace, cleanup and purge all error,
// and the joined apply error must carry the rename error while s.cfg
// has advanced to the renamed generation.
func TestApply_RenameMigrationFailureCommitsGenerationAndRunsRepair(t *testing.T) {
	dir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()
	cfgStore := newTestConfigStore(t, `
[sources.mock-01]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"
[webspaces.old]
keywords = ["demo"]
`)
	sup, err := NewSupervisor(ctx, idx, cfgStore, pluginhost.Dirs{Trusted: dir}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	defer sup.Shutdown()

	next := &config.Config{
		Sources: map[string]config.Source{
			"mock-01": {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test", Token: "unused"},
		},
		Webspaces: map[string]config.Webspace{
			"new": {Keywords: []string{"demo"}},
		},
	}
	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Sabotage: every index write from here fails — the rename migration
	// first among them.
	if err := idx.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	applyErr := sup.Apply(ctx)
	if applyErr == nil {
		t.Fatal("Apply must fail when the rename migration fails")
	}
	if !strings.Contains(applyErr.Error(), "webspace rename old -> new") {
		t.Fatalf("the apply error must lead with the rename: %v", applyErr)
	}
	// The generation committed regardless: s.cfg reflects the renamed
	// webspace, so a retry works on the new base.
	if _, ok := sup.cfg.Webspaces["new"]; !ok {
		t.Fatalf("the new generation must be committed on migration failure; s.cfg webspaces: %v", sup.cfg.Webspaces)
	}
	if _, ok := sup.cfg.Webspaces["old"]; ok {
		t.Fatal("the old generation leaked into s.cfg after commitGeneration")
	}
}
