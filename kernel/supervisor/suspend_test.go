package supervisor

import (
	"context"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/pluginhost"
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

	sup, err := NewSupervisor(ctx, idx, cfgStore, dir, hclog.NewNullLogger())
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

	sup, err := NewSupervisor(ctx, idx, cfgStore, dir, hclog.NewNullLogger())
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

func namesOf(plugins []*pluginhost.Plugin) []string {
	names := make([]string, len(plugins))
	for i, p := range plugins {
		names[i] = p.Name()
	}
	return names
}
