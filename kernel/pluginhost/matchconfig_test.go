package pluginhost

import (
	"strings"
	"testing"

	"github.com/davison/topos/kernel/config"
)

// newTestPlugin builds a *Plugin directly (same package, so unexported
// fields are reachable) without launching a real plugin subprocess —
// ValidateMatchConfig only ever reads Name/SourceType/PluginDisplayName/
// MatchVocabulary off a *Plugin, none of which require a live subprocess.
func newTestPlugin(name, sourceType, pluginName string, vocab []string) *Plugin {
	return &Plugin{name: name, sourceType: sourceType, pluginName: pluginName, matchVocabulary: vocab}
}

func newTestHost(plugins ...*Plugin) *Host {
	return &Host{plugins: plugins}
}

// TestValidateMatchConfig_UnknownFieldFailsNamingEverything pins the exact
// error text D-05 requires: the webspace, the instance, the offending
// field, the plugin binary's display name, its source_type, and the
// vocabulary it does declare.
func TestValidateMatchConfig_UnknownFieldFailsNamingEverything(t *testing.T) {
	host := newTestHost(newTestPlugin("home-email", "proton", "Proton Mail", []string{"folders"}))
	cfg := &config.Config{
		Sources: map[string]config.Source{"home-email": {Plugin: "topos-plugin-proton"}},
		Webspaces: map[string]config.Webspace{
			"house-move": {
				Match: map[string]config.MatchBlock{
					"home-email": {"labels": {"Home"}},
				},
			},
		},
	}

	err := ValidateMatchConfig(cfg, host)
	if err == nil {
		t.Fatal("expected error for an unknown match field, got nil")
	}
	want := `config: webspace "house-move" match block for source "home-email" declares unknown match field "labels" — plugin "Proton Mail" (source_type "proton") declares: [folders]`
	if err.Error() != want {
		t.Errorf("unexpected error text:\ngot:  %s\nwant: %s", err.Error(), want)
	}
}

// TestValidateMatchConfig_DeterministicOrderingAcrossTwoOffendingFields
// proves the reported error does not vary with Go's randomized map
// iteration order: two webspaces, each with one offending field, must
// always report the same (alphabetically-first-webspace) error across
// repeated calls.
func TestValidateMatchConfig_DeterministicOrderingAcrossTwoOffendingFields(t *testing.T) {
	host := newTestHost(
		newTestPlugin("home-email", "proton", "Proton Mail", []string{"folders"}),
		newTestPlugin("wiki", "silverbullet", "SilverBullet", []string{"tags", "pages"}),
	)
	cfg := &config.Config{
		Sources: map[string]config.Source{
			"home-email": {Plugin: "topos-plugin-proton"},
			"wiki":       {Plugin: "topos-plugin-silverbullet"},
		},
		Webspaces: map[string]config.Webspace{
			"garden": {
				Match: map[string]config.MatchBlock{
					"wiki": {"notes": {"x"}},
				},
			},
			"house-move": {
				Match: map[string]config.MatchBlock{
					"home-email": {"labels": {"Home"}},
				},
			},
		},
	}

	var first string
	for i := 0; i < 20; i++ {
		err := ValidateMatchConfig(cfg, host)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if i == 0 {
			first = err.Error()
			continue
		}
		if err.Error() != first {
			t.Fatalf("non-deterministic error across runs:\nrun 0: %s\nrun %d: %s", first, i, err.Error())
		}
	}
	// "garden" sorts before "house-move", so the deterministically-first
	// reported error must be the garden/wiki/notes offense.
	if !strings.Contains(first, "garden") || !strings.Contains(first, "wiki") || !strings.Contains(first, "notes") {
		t.Errorf("expected the deterministically-first error to name webspace garden, instance wiki, field notes, got: %s", first)
	}
}

// TestValidateMatchConfig_UnlaunchedInstanceFails proves a match block
// naming an instance that is configured but was not launched fails rather
// than passing vacuously (an empty byInstance lookup must never be
// silently skipped).
func TestValidateMatchConfig_UnlaunchedInstanceFails(t *testing.T) {
	host := newTestHost() // nothing launched
	cfg := &config.Config{
		Sources: map[string]config.Source{"home-email": {Plugin: "topos-plugin-proton"}},
		Webspaces: map[string]config.Webspace{
			"house-move": {
				Match: map[string]config.MatchBlock{
					"home-email": {"folders": {"Home"}},
				},
			},
		},
	}

	err := ValidateMatchConfig(cfg, host)
	if err == nil {
		t.Fatal("expected error for a match block naming an unlaunched instance, got nil")
	}
	if !strings.Contains(err.Error(), "house-move") || !strings.Contains(err.Error(), "home-email") {
		t.Errorf("expected error to name the webspace and the unlaunched instance, got: %v", err)
	}
}

// TestValidateMatchConfig_EmptyVocabularyWithFallbackFails proves a
// participating instance relying on the keywords fallback fails when its
// plugin declares an empty vocabulary — there is no field for the fallback
// to fan into.
func TestValidateMatchConfig_EmptyVocabularyWithFallbackFails(t *testing.T) {
	host := newTestHost(newTestPlugin("mock-source", "mock", "Mock", nil))
	cfg := &config.Config{
		Sources: map[string]config.Source{"mock-source": {Plugin: "topos-plugin-mock"}},
		Webspaces: map[string]config.Webspace{
			"house-move": {Keywords: []string{"house"}},
		},
	}

	err := ValidateMatchConfig(cfg, host)
	if err == nil {
		t.Fatal("expected error for a fallback-relying instance whose plugin declares an empty vocabulary, got nil")
	}
	if !strings.Contains(err.Error(), "house-move") || !strings.Contains(err.Error(), "mock-source") {
		t.Errorf("expected error to name the webspace and the instance, got: %v", err)
	}
}

// TestValidateMatchConfig_PassingTwoFieldVocabulary is the positive case: a
// match block whose fields are a subset of its instance's declared
// two-field vocabulary passes cleanly.
func TestValidateMatchConfig_PassingTwoFieldVocabulary(t *testing.T) {
	host := newTestHost(newTestPlugin("wiki", "silverbullet", "SilverBullet", []string{"tags", "pages"}))
	cfg := &config.Config{
		Sources: map[string]config.Source{"wiki": {Plugin: "topos-plugin-silverbullet"}},
		Webspaces: map[string]config.Webspace{
			"house-move": {
				Match: map[string]config.MatchBlock{
					"wiki": {"tags": {"house"}, "pages": {"Move"}},
				},
			},
		},
	}

	if err := ValidateMatchConfig(cfg, host); err != nil {
		t.Fatalf("expected no error for a fully-declared two-field vocabulary, got: %v", err)
	}
}

// newTestHostWithLaunchFailure builds a *Host carrying zero launched
// plugins and exactly one recorded soft launch failure for `instance` — the
// T-11-33 exemption's own fixture shape, same-package direct construction
// (Host.launchFailures is unexported) mirroring newTestHost's own pattern.
func newTestHostWithLaunchFailure(instance string) *Host {
	return &Host{
		launchFailures: map[string]LaunchFailure{
			instance: {Instance: instance, Reason: LaunchFailurePinMismatch},
		},
	}
}

// TestValidateMatchConfig_PinMismatchedInstanceExcusedFromExplicitMatchBlock
// proves T-11-33's own threat-register mitigation at the pluginhost layer
// (11-06-PLAN.md Task 3's browser proof surfaced this gap live —
// 11-02-SUMMARY.md's "Issues Encountered" had flagged it, unresolved, as
// "a later plan... if it becomes user-visible"): an explicit match block
// naming a CURRENTLY pin-mismatched instance (LaunchFailures(), not
// byInstance) must not fail this check — its own launch refusal is already
// reported separately (GET /api/sources), not a config defect to reject an
// unrelated save over.
func TestValidateMatchConfig_PinMismatchedInstanceExcusedFromExplicitMatchBlock(t *testing.T) {
	host := newTestHostWithLaunchFailure("bad-pin")
	cfg := &config.Config{
		Sources: map[string]config.Source{"bad-pin": {Plugin: "topos-plugin-external"}},
		Webspaces: map[string]config.Webspace{
			"demo": {
				Match: map[string]config.MatchBlock{
					"bad-pin": {"labels": {"anything"}},
				},
			},
		},
	}

	if err := ValidateMatchConfig(cfg, host); err != nil {
		t.Fatalf("expected a pin-mismatched instance's explicit match block to be excused, got: %v", err)
	}
}

// TestValidateMatchConfig_PinMismatchedInstanceExcusedFromKeywordsFallback
// is the fallback-vocabulary sibling of the test above — a pin-mismatched
// instance participating via ws.Keywords (no explicit match block) must be
// excused the same way, not just the explicit-match-block path.
func TestValidateMatchConfig_PinMismatchedInstanceExcusedFromKeywordsFallback(t *testing.T) {
	host := newTestHostWithLaunchFailure("bad-pin")
	cfg := &config.Config{
		Sources: map[string]config.Source{"bad-pin": {Plugin: "topos-plugin-external"}},
		Webspaces: map[string]config.Webspace{
			"demo": {Keywords: []string{"demo"}},
		},
	}

	if err := ValidateMatchConfig(cfg, host); err != nil {
		t.Fatalf("expected a pin-mismatched instance's keywords-fallback participation to be excused, got: %v", err)
	}
}

// TestValidateMatchConfig_UnlaunchedInstanceStillFailsWhenNotExcused proves
// the exemption above is scoped EXACTLY to instances LaunchFailures()
// names, not a blanket relaxation — an instance absent from byInstance AND
// absent from the excused set must still fail exactly as
// TestValidateMatchConfig_UnlaunchedInstanceFails already proves for a
// plain *Host with no recorded failures at all.
func TestValidateMatchConfig_UnlaunchedInstanceStillFailsWhenNotExcused(t *testing.T) {
	// A launch failure recorded for a DIFFERENT instance must not excuse
	// this one.
	host := newTestHostWithLaunchFailure("some-other-instance")
	cfg := &config.Config{
		Sources: map[string]config.Source{"home-email": {Plugin: "topos-plugin-proton"}},
		Webspaces: map[string]config.Webspace{
			"house-move": {
				Match: map[string]config.MatchBlock{
					"home-email": {"folders": {"Home"}},
				},
			},
		},
	}

	err := ValidateMatchConfig(cfg, host)
	if err == nil {
		t.Fatal("expected error for a match block naming an unlaunched, non-excused instance, got nil")
	}
	if !strings.Contains(err.Error(), "house-move") || !strings.Contains(err.Error(), "home-email") {
		t.Errorf("expected error to name the webspace and the unlaunched instance, got: %v", err)
	}
}
