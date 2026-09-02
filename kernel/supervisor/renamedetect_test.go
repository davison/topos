package supervisor

import (
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
