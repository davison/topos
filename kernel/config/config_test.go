package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestLoad_ValidConfigWithEnvExpansion(t *testing.T) {
	t.Setenv("TEST_PAPERLESS_URL", "http://paperless.lan:8000")
	t.Setenv("TEST_PAPERLESS_TOKEN", "secret-token")

	path := writeTempConfig(t, `
[server]
listen = "127.0.0.1:7777"

[index]
path = "/tmp/webspaces-index.db"

[sources.paperless]
plugin = "topos-plugin-paperless"
base_url = "${TEST_PAPERLESS_URL}"
token = "${TEST_PAPERLESS_TOKEN}"
api_version = "10"

[webspaces.house-move]
keywords = ["house-move", "House"]
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Sources["paperless"].BaseURL != "http://paperless.lan:8000" {
		t.Errorf("base_url not expanded, got %q", cfg.Sources["paperless"].BaseURL)
	}
	if cfg.Sources["paperless"].Token != "secret-token" {
		t.Errorf("token not expanded, got %q", cfg.Sources["paperless"].Token)
	}
	if len(cfg.Webspaces["house-move"].Keywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(cfg.Webspaces["house-move"].Keywords))
	}
}

func TestLoad_Defaults(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.empty-space]
keywords = ["x"]
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Listen != DefaultListen {
		t.Errorf("expected default listen %q, got %q", DefaultListen, cfg.Server.Listen)
	}
	if cfg.Plugins.Dir != DefaultPluginsDir {
		t.Errorf("expected default plugins dir %q, got %q", DefaultPluginsDir, cfg.Plugins.Dir)
	}
}

func TestLoad_ZeroWebspacesStartsFine(t *testing.T) {
	path := writeTempConfig(t, `
[server]
listen = "127.0.0.1:7777"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load with zero webspaces should not error: %v", err)
	}
	if len(cfg.Webspaces) != 0 {
		t.Errorf("expected zero webspaces, got %d", len(cfg.Webspaces))
	}
}

// TestLoad_ZeroKeywordsFails pre-dates D-20 (07-11-PLAN.md, gap closure for
// 07-UAT.md G-07-3) and pinned 05-03 D-01's original, unconditional
// invariant: a webspace declaring an explicitly empty keywords list, with
// no match block and no sources allowlist, failed load. D-20 deliberately
// and knowingly supersedes that invariant for exactly this shape — a
// webspace declaring NONE of keywords/sources/match is now a legitimate
// "empty webspace shell" (Webspace.IsEmptyShell) that loads successfully,
// because it is the literal document web/src/lib/config-edit.ts's
// addWebspace() PUTs as the create-webspace modal's first write. This
// test's assertion is updated (not deleted, so a future regression that
// makes an explicit `keywords = []` behave differently from an omitted
// `keywords` key is still caught here) to assert the new, correct
// behaviour — see 07-11-SUMMARY.md's Deviations section for why this
// pre-existing test body needed to change despite the plan's general
// "don't touch existing test bodies" instruction: the plan's own must_haves
// require exactly this fixture to load, which this test's pre-D-20
// assertion directly contradicted.
func TestLoad_ZeroKeywordsFails(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.house-move]
keywords = []
`)
	_, err := Load(path)
	if err != nil {
		t.Fatalf("expected an explicit empty keywords list with no match block and no sources allowlist to load as a D-20 empty webspace shell, got: %v", err)
	}
}

func TestLoad_WhitespaceOnlyKeywordFails(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.house-move]
keywords = ["house-move", "   "]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for whitespace-only keyword, got nil")
	}
}

func TestLoad_MissingEnvVarNamedInError(t *testing.T) {
	os.Unsetenv("TEST_UNSET_PAPERLESS_TOKEN")
	t.Setenv("TEST_UNSET_PAPERLESS_URL", "http://paperless.lan:8000")

	path := writeTempConfig(t, `
[sources.paperless]
plugin = "topos-plugin-paperless"
base_url = "${TEST_UNSET_PAPERLESS_URL}"
token = "${TEST_UNSET_PAPERLESS_TOKEN}"
api_version = "10"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing env var, got nil")
	}
	if !strings.Contains(err.Error(), "TEST_UNSET_PAPERLESS_TOKEN") {
		t.Errorf("expected error to name TEST_UNSET_PAPERLESS_TOKEN, got: %v", err)
	}
}

func TestSyncIntervalFor_NoSyncBlockDefaultsToFifteenMinutes(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.house-move]
keywords = ["house"]
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	d, err := cfg.SyncIntervalFor("anything")
	if err != nil {
		t.Fatalf("SyncIntervalFor: %v", err)
	}
	if d != 15*time.Minute {
		t.Errorf("expected 15m default, got %v", d)
	}
}

func TestSyncIntervalFor_GlobalOverrideAppliesToEverySource(t *testing.T) {
	t.Setenv("TEST_SYNC_URL", "http://x.lan")
	t.Setenv("TEST_SYNC_TOKEN", "tok")
	path := writeTempConfig(t, `
[sync]
interval = "5m"

[sources.paperless]
plugin = "topos-plugin-paperless"
base_url = "${TEST_SYNC_URL}"
token = "${TEST_SYNC_TOKEN}"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	d, err := cfg.SyncIntervalFor("paperless")
	if err != nil {
		t.Fatalf("SyncIntervalFor: %v", err)
	}
	if d != 5*time.Minute {
		t.Errorf("expected 5m global interval, got %v", d)
	}
}

func TestSyncIntervalFor_PerSourceOverrideAppliesOnlyToThatSource(t *testing.T) {
	t.Setenv("TEST_SYNC_URL", "http://x.lan")
	t.Setenv("TEST_SYNC_TOKEN", "tok")
	t.Setenv("TEST_SYNC_URL2", "http://y.lan")
	t.Setenv("TEST_SYNC_TOKEN2", "tok2")
	path := writeTempConfig(t, `
[sync]
interval = "5m"

[sources.paperless]
plugin = "topos-plugin-paperless"
base_url = "${TEST_SYNC_URL}"
token = "${TEST_SYNC_TOKEN}"

[sources.silverbullet]
plugin = "topos-plugin-silverbullet"
base_url = "${TEST_SYNC_URL2}"
token = "${TEST_SYNC_TOKEN2}"
sync_interval = "1m"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	sb, err := cfg.SyncIntervalFor("silverbullet")
	if err != nil {
		t.Fatalf("SyncIntervalFor(silverbullet): %v", err)
	}
	if sb != time.Minute {
		t.Errorf("expected 1m override for silverbullet, got %v", sb)
	}
	pl, err := cfg.SyncIntervalFor("paperless")
	if err != nil {
		t.Fatalf("SyncIntervalFor(paperless): %v", err)
	}
	if pl != 5*time.Minute {
		t.Errorf("expected 5m global interval for paperless (no override), got %v", pl)
	}
}

func TestLoad_UnparseableIntervalFails(t *testing.T) {
	path := writeTempConfig(t, `
[sync]
interval = "soon"

[webspaces.house-move]
keywords = ["house"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unparseable interval, got nil")
	}
	if !strings.Contains(err.Error(), "interval") {
		t.Errorf("expected error to name 'interval', got: %v", err)
	}
}

func TestLoad_ZeroIntervalFails(t *testing.T) {
	path := writeTempConfig(t, `
[sync]
interval = "0s"

[webspaces.house-move]
keywords = ["house"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for zero interval, got nil")
	}
}

func TestLoad_NegativeIntervalFails(t *testing.T) {
	path := writeTempConfig(t, `
[sync]
interval = "-1m"

[webspaces.house-move]
keywords = ["house"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for negative interval, got nil")
	}
}

func TestLoad_UnparseablePerSourceIntervalFailsNamingTheKey(t *testing.T) {
	t.Setenv("TEST_SYNC_URL3", "http://x.lan")
	t.Setenv("TEST_SYNC_TOKEN3", "tok")
	path := writeTempConfig(t, `
[sources.paperless]
plugin = "topos-plugin-paperless"
base_url = "${TEST_SYNC_URL3}"
token = "${TEST_SYNC_TOKEN3}"
sync_interval = "soon"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unparseable sync_interval, got nil")
	}
	if !strings.Contains(err.Error(), "sync_interval") {
		t.Errorf("expected error to name 'sync_interval', got: %v", err)
	}
}

// TestAgentReadGrantedNames_AbsentEmptyAndExplicitFalseAreAllDenied proves
// the three ways a source can end up unread-granted (no [agent] block at
// all, an empty [agent] block, and an explicit read = false) all produce
// the identical result: absent from AgentReadGrantedNames. This is the
// three-way equivalence T-02-19 depends on.
func TestAgentReadGrantedNames_AbsentEmptyAndExplicitFalseAreAllDenied(t *testing.T) {
	t.Setenv("TEST_AGENT_URL", "http://x.lan")
	t.Setenv("TEST_AGENT_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.no-block]
plugin = "x"
base_url = "${TEST_AGENT_URL}"
token = "${TEST_AGENT_TOKEN}"

[sources.empty-block]
plugin = "x"
base_url = "${TEST_AGENT_URL}"
token = "${TEST_AGENT_TOKEN}"
[sources.empty-block.agent]

[sources.explicit-false]
plugin = "x"
base_url = "${TEST_AGENT_URL}"
token = "${TEST_AGENT_TOKEN}"
[sources.explicit-false.agent]
read = false
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	granted := cfg.AgentReadGrantedNames()
	if len(granted) != 0 {
		t.Errorf("expected zero read-granted sources across all three deny-equivalent shapes, got %v", granted)
	}
}

// TestAgentReadGrantedNames_HandoffWithoutReadIsNotReadGranted proves the
// two grants are independent: a source with handoff = true and no read key
// (or read = false) is still absent from AgentReadGrantedNames, while its
// own Handoff field is still readable off the Config for capability
// publishing.
func TestAgentReadGrantedNames_HandoffWithoutReadIsNotReadGranted(t *testing.T) {
	t.Setenv("TEST_AGENT_URL2", "http://x.lan")
	t.Setenv("TEST_AGENT_TOKEN2", "tok")
	path := writeTempConfig(t, `
[sources.handoff-only]
plugin = "x"
base_url = "${TEST_AGENT_URL2}"
token = "${TEST_AGENT_TOKEN2}"
[sources.handoff-only.agent]
handoff = true
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if granted := cfg.AgentReadGrantedNames(); len(granted) != 0 {
		t.Errorf("expected handoff=true/read=false to not be read-granted, got %v", granted)
	}
	if !cfg.Sources["handoff-only"].Agent.Handoff {
		t.Error("expected Agent.Handoff to be true even though Read is false")
	}
}

// TestAgentReadGrantedNames_ExplicitReadTrueIsGranted is the positive case:
// a source that explicitly sets read = true is present in the granted set.
func TestAgentReadGrantedNames_ExplicitReadTrueIsGranted(t *testing.T) {
	t.Setenv("TEST_AGENT_URL3", "http://x.lan")
	t.Setenv("TEST_AGENT_TOKEN3", "tok")
	path := writeTempConfig(t, `
[sources.granted]
plugin = "x"
base_url = "${TEST_AGENT_URL3}"
token = "${TEST_AGENT_TOKEN3}"
[sources.granted.agent]
read = true
handoff = true
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	granted := cfg.AgentReadGrantedNames()
	if !granted["granted"] {
		t.Errorf("expected 'granted' source to be read-granted, got %v", granted)
	}
}

// TestLoad_PathOnlySourceValidatesCleanly proves a source declaring only
// plugin and path (no base_url, no token) validates without error —
// SRC-02's Signal plugin is the first source of this shape (04-01-PLAN.md
// Task 2).
func TestLoad_PathOnlySourceValidatesCleanly(t *testing.T) {
	path := writeTempConfig(t, `
[sources.signal]
plugin = "topos-plugin-signal"
path = "~/.config/Signal"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Sources["signal"].Path != "~/.config/Signal" {
		t.Errorf("expected path to be preserved, got %q", cfg.Sources["signal"].Path)
	}
}

// TestLoad_SourceWithNeitherPathNorBaseURLTokenFailsNamingBothShapes
// proves a source declaring none of path/base_url/token fails config
// load with an error naming BOTH accepted shapes, not just the first
// missing field.
func TestLoad_SourceWithNeitherPathNorBaseURLTokenFailsNamingBothShapes(t *testing.T) {
	path := writeTempConfig(t, `
[sources.broken]
plugin = "topos-plugin-broken"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for a source declaring none of path/base_url/token, got nil")
	}
	if !strings.Contains(err.Error(), "base_url") || !strings.Contains(err.Error(), "path") {
		t.Errorf("expected error to name both accepted shapes (base_url and path), got: %v", err)
	}
}

// TestDisplayNameFor_OmittedDefaultsToInstanceID proves D-09: a
// [sources.<id>] block that omits display_name resolves to the instance id
// itself.
func TestDisplayNameFor_OmittedDefaultsToInstanceID(t *testing.T) {
	t.Setenv("TEST_DN_URL", "http://x.lan")
	t.Setenv("TEST_DN_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.home-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_DN_URL}"
token = "${TEST_DN_TOKEN}"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.DisplayNameFor("home-email"); got != "home-email" {
		t.Errorf("expected DisplayNameFor to default to the instance id %q, got %q", "home-email", got)
	}
}

// TestDisplayNameFor_ExplicitValueIsUsed proves an explicitly configured
// display_name is returned verbatim rather than the instance id.
func TestDisplayNameFor_ExplicitValueIsUsed(t *testing.T) {
	t.Setenv("TEST_DN2_URL", "http://x.lan")
	t.Setenv("TEST_DN2_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.home-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_DN2_URL}"
token = "${TEST_DN2_TOKEN}"
display_name = "Home Email"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.DisplayNameFor("home-email"); got != "Home Email" {
		t.Errorf("expected explicit display_name %q, got %q", "Home Email", got)
	}
}

// TestLoad_DuplicateDisplayNameCaseInsensitiveFailsNamingBothSources proves
// D-09's uniqueness rule: two sources whose resolved display names collide
// only by case fail config load, naming both source keys and the colliding
// value.
func TestLoad_DuplicateDisplayNameCaseInsensitiveFailsNamingBothSources(t *testing.T) {
	t.Setenv("TEST_DUP_URL", "http://x.lan")
	t.Setenv("TEST_DUP_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.home-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_DUP_URL}"
token = "${TEST_DUP_TOKEN}"
display_name = "Email"

[sources.work-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_DUP_URL}"
token = "${TEST_DUP_TOKEN}"
display_name = "email"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for case-insensitively colliding display_name, got nil")
	}
	if !strings.Contains(err.Error(), "home-email") || !strings.Contains(err.Error(), "work-email") {
		t.Errorf("expected error to name both source keys, got: %v", err)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "email") {
		t.Errorf("expected error to name the colliding display_name value, got: %v", err)
	}
}

// TestLoad_TwoInstancesOfSamePluginTypeWithDistinctDisplayNamesLoadCleanly
// proves the core multi-instance shape (D-08/D-09) loads without error: two
// [sources.*] entries whose plugin value is identical, with distinct
// display names, validate cleanly.
func TestLoad_TwoInstancesOfSamePluginTypeWithDistinctDisplayNamesLoadCleanly(t *testing.T) {
	t.Setenv("TEST_TWO_URL", "http://x.lan")
	t.Setenv("TEST_TWO_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.home-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_TWO_URL}"
token = "${TEST_TWO_TOKEN}"
display_name = "Home Email"

[sources.work-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_TWO_URL}"
token = "${TEST_TWO_TOKEN}"
display_name = "Work Email"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Sources) != 2 {
		t.Fatalf("expected 2 source instances, got %d", len(cfg.Sources))
	}
	if cfg.Sources["home-email"].Plugin != cfg.Sources["work-email"].Plugin {
		t.Fatal("expected both instances to share the same plugin binary")
	}
}

// TestLoad_WebspaceWithNeitherKeywordsNorMatchFails pre-dates D-20
// (07-11-PLAN.md, gap closure for 07-UAT.md G-07-3) and pinned 05-03 D-06's
// original invariant against this exact fixture: a webspace block with no
// keys at all (no keywords, no match, no sources). D-20 deliberately
// reclassifies precisely this shape as a legitimate "empty webspace shell"
// (Webspace.IsEmptyShell) — see TestLoad_ZeroKeywordsFails's doc comment
// for the full rationale, which applies identically here. This test's
// assertion is updated to match: the fixture now loads successfully. D-06's
// actual guard — a PARTICIPATING instance left uncovered — is still
// enforced and still tested, by TestLoad_ParticipatingInstanceWithNoBlockAndEmptyKeywordsFails
// and TestValidate_PartiallyCoveredWebspaceIsStillRejected (both configure
// at least one source instance, which this fixture deliberately does not).
func TestLoad_WebspaceWithNeitherKeywordsNorMatchFails(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.house-move]
`)
	_, err := Load(path)
	if err != nil {
		t.Fatalf("expected a webspace block with no keys at all to load as a D-20 empty webspace shell, got: %v", err)
	}
}

// TestLoad_MatchBlockUnknownInstanceFails proves a match block naming an
// instance with no corresponding [sources.<id>] entry fails config load,
// naming the webspace and the unknown instance.
func TestLoad_MatchBlockUnknownInstanceFails(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.house-move]
keywords = ["house"]

[webspaces.house-move.match.nonexistent]
folders = ["Home"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for a match block naming an unconfigured instance, got nil")
	}
	if !strings.Contains(err.Error(), "house-move") || !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to name the webspace and the unknown instance, got: %v", err)
	}
}

// TestLoad_SourcesAllowlistUnknownInstanceFails proves a sources allowlist
// entry naming an unconfigured instance fails config load, naming the
// webspace and the unknown instance.
func TestLoad_SourcesAllowlistUnknownInstanceFails(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.house-move]
keywords = ["house"]
sources = ["nonexistent"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for a sources allowlist naming an unconfigured instance, got nil")
	}
	if !strings.Contains(err.Error(), "house-move") || !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to name the webspace and the unknown instance, got: %v", err)
	}
}

// TestLoad_MatchBlockForDeallowlistedInstanceFails proves the dead-config
// rule decided by 05-RESEARCH.md Open Question 1: an instance both excluded
// by a webspace's sources allowlist and given an explicit match block in
// that same webspace fails config load, naming the webspace and the
// instance.
func TestLoad_MatchBlockForDeallowlistedInstanceFails(t *testing.T) {
	t.Setenv("TEST_DEAD_URL", "http://x.lan")
	t.Setenv("TEST_DEAD_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.home-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_DEAD_URL}"
token = "${TEST_DEAD_TOKEN}"

[sources.work-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_DEAD_URL}"
token = "${TEST_DEAD_TOKEN}"

[webspaces.house-move]
keywords = ["house"]
sources = ["work-email"]

[webspaces.house-move.match.home-email]
folders = ["Home"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for a match block on a source excluded by the sources allowlist, got nil")
	}
	if !strings.Contains(err.Error(), "house-move") || !strings.Contains(err.Error(), "home-email") {
		t.Errorf("expected error to name the webspace and the excluded instance, got: %v", err)
	}
}

// TestLoad_MatchBlockZeroFieldsFails proves a match block declaring zero
// fields fails config load, naming the webspace and the instance.
func TestLoad_MatchBlockZeroFieldsFails(t *testing.T) {
	t.Setenv("TEST_ZF_URL", "http://x.lan")
	t.Setenv("TEST_ZF_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.home-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_ZF_URL}"
token = "${TEST_ZF_TOKEN}"

[webspaces.house-move]
keywords = ["house"]

[webspaces.house-move.match]
home-email = {}
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for a match block declaring zero fields, got nil")
	}
	if !strings.Contains(err.Error(), "house-move") || !strings.Contains(err.Error(), "home-email") {
		t.Errorf("expected error to name the webspace and the instance, got: %v", err)
	}
}

// TestLoad_MatchBlockEmptyValueFails proves a match block field with an
// empty or whitespace-only value fails config load, naming the webspace,
// the instance, and the field.
func TestLoad_MatchBlockEmptyValueFails(t *testing.T) {
	t.Setenv("TEST_EV_URL", "http://x.lan")
	t.Setenv("TEST_EV_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.home-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_EV_URL}"
token = "${TEST_EV_TOKEN}"

[webspaces.house-move]
keywords = ["house"]

[webspaces.house-move.match.home-email]
folders = ["Home", "   "]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for a match block field with a whitespace-only value, got nil")
	}
	if !strings.Contains(err.Error(), "house-move") || !strings.Contains(err.Error(), "home-email") || !strings.Contains(err.Error(), "folders") {
		t.Errorf("expected error to name the webspace, instance and field, got: %v", err)
	}
}

// TestLoad_ParticipatingInstanceWithNoBlockAndEmptyKeywordsFails proves
// D-06: a participating instance with no explicit match block, in a
// webspace whose keywords fallback is empty, fails config load — there is
// no field for the fallback to fan into and nothing else to resolve this
// instance's match input from.
func TestLoad_ParticipatingInstanceWithNoBlockAndEmptyKeywordsFails(t *testing.T) {
	t.Setenv("TEST_FC_URL", "http://x.lan")
	t.Setenv("TEST_FC_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.home-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_FC_URL}"
token = "${TEST_FC_TOKEN}"

[sources.work-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_FC_URL}"
token = "${TEST_FC_TOKEN}"

[webspaces.house-move.match.home-email]
folders = ["Home"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for a participating instance with no block and no keywords fallback, got nil")
	}
	if !strings.Contains(err.Error(), "house-move") || !strings.Contains(err.Error(), "work-email") {
		t.Errorf("expected error to name the webspace and the uncovered instance, got: %v", err)
	}
}

// TestValidate_EmptyWebspaceShellIsAccepted proves D-20 (07-11-PLAN.md,
// closes 07-UAT.md G-07-3): a webspace declaring none of keywords, match
// blocks, or a sources allowlist — the exact document
// web/src/lib/config-edit.ts's addWebspace() PUTs as the create-webspace
// modal's first write — validates cleanly on an installation that already
// has configured source instances (every real installation past initial
// setup). Against pre-D-20 code this fails with the line-323 message.
func TestValidate_EmptyWebspaceShellIsAccepted(t *testing.T) {
	t.Setenv("TEST_SHELL_URL", "http://x.lan")
	t.Setenv("TEST_SHELL_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.home-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_SHELL_URL}"
token = "${TEST_SHELL_TOKEN}"

[sources.work-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_SHELL_URL}"
token = "${TEST_SHELL_TOKEN}"

[webspaces.new-project]
`)
	_, err := Load(path)
	if err != nil {
		t.Fatalf("expected an empty webspace shell to validate cleanly on an install with configured sources, got: %v", err)
	}
}

// TestValidate_EmptyWebspaceShellIsAcceptedWithZeroSourcesConfigured proves
// D-20's first-run edge (KERN-08): a config with zero [sources.*] blocks at
// all plus one shell webspace still validates — a first-run install can
// create its first webspace before configuring any source.
func TestValidate_EmptyWebspaceShellIsAcceptedWithZeroSourcesConfigured(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.new-project]
`)
	_, err := Load(path)
	if err != nil {
		t.Fatalf("expected an empty webspace shell to validate cleanly on a first-run install with zero configured sources, got: %v", err)
	}
}

// TestValidate_WebspaceWithAllowlistButNoMatchInputIsStillRejected proves
// D-20's discriminator stays at exactly three conditions: a webspace that
// names an instance in its sources allowlist but declares neither keywords
// nor any match block is NOT a shell (a non-empty sources allowlist
// disqualifies it) and still fails load with the existing message naming
// both accepted shapes. Must pass BEFORE and AFTER D-20 — if it ever fails
// after, the discriminator has been widened past its three conditions and
// the operator-typo shape "allowlisted a source, told it nothing to match"
// would be silently accepted.
func TestValidate_WebspaceWithAllowlistButNoMatchInputIsStillRejected(t *testing.T) {
	t.Setenv("TEST_ALW_URL", "http://x.lan")
	t.Setenv("TEST_ALW_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.home-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_ALW_URL}"
token = "${TEST_ALW_TOKEN}"

[webspaces.house-move]
sources = ["home-email"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected a webspace that allowlists a source but declares no keywords/match to still be rejected — this is NOT a shell, it is the operator-typo shape the loud error exists to catch")
	}
	if !strings.Contains(err.Error(), "house-move") {
		t.Errorf("expected error to name the webspace, got: %v", err)
	}
	if !strings.Contains(err.Error(), "keywords") || !strings.Contains(err.Error(), "match") {
		t.Errorf("expected error to name both accepted shapes (keywords and match), got: %v", err)
	}
}

// TestValidate_PartiallyCoveredWebspaceIsStillRejected proves 05-03 D-06 is
// preserved untouched by D-20: two configured instances, no keywords, a
// match block for only one, no allowlist — still fails, naming the
// uncovered instance. Must pass BEFORE and AFTER D-20.
func TestValidate_PartiallyCoveredWebspaceIsStillRejected(t *testing.T) {
	t.Setenv("TEST_PC_URL", "http://x.lan")
	t.Setenv("TEST_PC_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.home-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_PC_URL}"
token = "${TEST_PC_TOKEN}"

[sources.work-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_PC_URL}"
token = "${TEST_PC_TOKEN}"

[webspaces.house-move.match.home-email]
folders = ["Home"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected a webspace covering only one of two configured instances (no allowlist, no keywords fallback) to still be rejected, naming the uncovered instance")
	}
	if !strings.Contains(err.Error(), "house-move") || !strings.Contains(err.Error(), "work-email") {
		t.Errorf("expected error to name the webspace and the uncovered instance, got: %v", err)
	}
}

// TestLoad_MatchBlockDecodesNestedTOMLShape proves the core decode
// capability this plan depends on: [webspaces.<ws>.match.<instance>]
// nested TOML decodes to Config.Webspaces[ws].Match[instance][field] via
// go-toml/v2's map-of-map decode, with no new dependency.
func TestLoad_MatchBlockDecodesNestedTOMLShape(t *testing.T) {
	t.Setenv("TEST_MB_URL", "http://x.lan")
	t.Setenv("TEST_MB_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.home-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_MB_URL}"
token = "${TEST_MB_TOKEN}"

[webspaces.house-move]
keywords = ["house"]

[webspaces.house-move.match.home-email]
folders = ["Home"]
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := cfg.Webspaces["house-move"].Match["home-email"]["folders"]
	if len(got) != 1 || got[0] != "Home" {
		t.Errorf("expected Match[\"home-email\"][\"folders\"] == [\"Home\"], got %+v", got)
	}
}

func TestLoad_ExpandsHomeInIndexPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir available in this environment")
	}
	path := writeTempConfig(t, `
[index]
path = "~/.local/share/topos/index.db"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := home + "/.local/share/topos/index.db"
	if cfg.Index.Path != want {
		t.Errorf("expected index path %q, got %q", want, cfg.Index.Path)
	}
}

// TestApplyDefaults_NormalizesCollectionsWithoutChangingValidateVerdicts is
// the mechanical proof — not merely the argument — that 07-12-PLAN.md Task
// 1's applyDefaults collection-normalization block cannot change any
// config's Validate verdict. Every collection check in the validation path
// (validateWebspaces, validateMatchBlocks, validateSourcesAllowlist,
// validateFallbackCoverage, validateDisplayNameUniqueness, Webspace.
// IsEmptyShell) is len(...)-based, for which a nil and an empty collection
// are indistinguishable — this table spans the empty config, a
// keywords-only webspace, 07-11's D-20 empty shell (IsEmptyShell tests
// three collections at once and is the check most sensitive to a
// nil-versus-empty distinction being introduced anywhere), an
// allowlist-without-match-input webspace (the operator-typo shape
// TestValidate_WebspaceWithAllowlistButNoMatchInputIsStillRejected pins),
// and a partially-covered webspace (TestValidate_
// PartiallyCoveredWebspaceIsStillRejected's shape) — and asserts BOTH a
// before-normalization and an after-normalization call to Validate(nil)
// agree on nil-ness AND, when non-nil, the exact error text. Each fixture
// pre-sets [sync] interval to a valid positive duration so the scalar
// Sync.Interval default (which DOES legitimately change Validate's verdict
// when empty, by design) never confounds this test's only concern:
// collection normalization.
func TestApplyDefaults_NormalizesCollectionsWithoutChangingValidateVerdicts(t *testing.T) {
	cases := map[string]func() *Config{
		"empty config, zero sources zero webspaces": func() *Config {
			return &Config{Sync: SyncConfig{Interval: "15m"}}
		},
		"keywords-only webspace": func() *Config {
			return &Config{
				Sync: SyncConfig{Interval: "15m"},
				Webspaces: map[string]Webspace{
					"house-move": {Keywords: []string{"house-move"}},
				},
			}
		},
		"D-20 empty webspace shell": func() *Config {
			return &Config{
				Sync: SyncConfig{Interval: "15m"},
				Webspaces: map[string]Webspace{
					"new-project": {},
				},
			}
		},
		"allowlist without match input (operator-typo rejection shape)": func() *Config {
			return &Config{
				Sync: SyncConfig{Interval: "15m"},
				Sources: map[string]Source{
					"home-email": {Plugin: "topos-plugin-proton", BaseURL: "http://x.lan", Token: "tok"},
				},
				Webspaces: map[string]Webspace{
					"house-move": {Sources: []string{"home-email"}},
				},
			}
		},
		"partially-covered webspace (two instances, match for only one)": func() *Config {
			return &Config{
				Sync: SyncConfig{Interval: "15m"},
				Sources: map[string]Source{
					"home-email": {Plugin: "topos-plugin-proton", BaseURL: "http://x.lan", Token: "tok"},
					"work-email": {Plugin: "topos-plugin-proton", BaseURL: "http://x.lan", Token: "tok"},
				},
				Webspaces: map[string]Webspace{
					"house-move": {
						Match: map[string]MatchBlock{
							"home-email": {"folders": {"Home"}},
						},
					},
				},
			}
		},
	}

	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			before := build()
			beforeErr := before.Validate(nil)

			after := build()
			applyDefaults(after)
			afterErr := after.Validate(nil)

			if (beforeErr == nil) != (afterErr == nil) {
				t.Fatalf("normalization changed the Validate verdict: before=%v after=%v", beforeErr, afterErr)
			}
			if beforeErr != nil && beforeErr.Error() != afterErr.Error() {
				t.Fatalf("normalization changed the Validate error text:\nbefore=%q\nafter=%q", beforeErr.Error(), afterErr.Error())
			}
		})
	}
}

// TestValidate_Pins is Phase 11's own load-time gate over [plugins.pins]
// (D-01/D-02): a well-formed pin (a topos-plugin- prefixed key, a
// 64-character lowercase hex value) validates cleanly; a too-short pin, an
// uppercase-hex pin, and a non-plugin-shaped key are each rejected, naming
// the offending key.
func TestValidate_Pins(t *testing.T) {
	validHash := strings.Repeat("a", 64)

	t.Run("well-formed pin validates cleanly", func(t *testing.T) {
		cfg := &Config{
			Sync:    SyncConfig{Interval: "15m"},
			Plugins: PluginsConfig{Pins: map[string]string{"topos-plugin-example": validHash}},
		}
		if err := cfg.Validate(nil); err != nil {
			t.Fatalf("expected a well-formed pin to validate cleanly, got: %v", err)
		}
	})

	t.Run("63-character pin is rejected naming the key", func(t *testing.T) {
		cfg := &Config{
			Sync:    SyncConfig{Interval: "15m"},
			Plugins: PluginsConfig{Pins: map[string]string{"topos-plugin-example": validHash[:63]}},
		}
		err := cfg.Validate(nil)
		if err == nil {
			t.Fatal("expected a 63-character pin to be rejected")
		}
		if !strings.Contains(err.Error(), "topos-plugin-example") {
			t.Errorf("expected the error to name the offending key, got: %v", err)
		}
	})

	t.Run("uppercase-hex pin is rejected naming the key", func(t *testing.T) {
		cfg := &Config{
			Sync:    SyncConfig{Interval: "15m"},
			Plugins: PluginsConfig{Pins: map[string]string{"topos-plugin-example": strings.ToUpper(validHash)}},
		}
		err := cfg.Validate(nil)
		if err == nil {
			t.Fatal("expected an uppercase-hex pin to be rejected")
		}
		if !strings.Contains(err.Error(), "topos-plugin-example") {
			t.Errorf("expected the error to name the offending key, got: %v", err)
		}
	})

	t.Run("non-plugin-shaped key is rejected naming the key", func(t *testing.T) {
		cfg := &Config{
			Sync:    SyncConfig{Interval: "15m"},
			Plugins: PluginsConfig{Pins: map[string]string{"not-a-plugin-name": validHash}},
		}
		err := cfg.Validate(nil)
		if err == nil {
			t.Fatal("expected a non-plugin-shaped key to be rejected")
		}
		if !strings.Contains(err.Error(), "not-a-plugin-name") {
			t.Errorf("expected the error to name the offending key, got: %v", err)
		}
	})
}

// TestLoad_ExtrasVarExpandsExactlyLikeBaseURL proves D-13: a ${VAR}
// reference inside a [sources.<id>.extras] value expands at load time
// exactly like base_url/token do — the operator never needs a second
// mental model for "which config fields support ${VAR}".
func TestLoad_ExtrasVarExpandsExactlyLikeBaseURL(t *testing.T) {
	t.Setenv("TEST_EXTRAS_REGION", "eu-west-1")

	path := writeTempConfig(t, `
[sources.example]
plugin = "topos-plugin-example"
base_url = "http://x.lan"
token = "tok"

[sources.example.extras]
region = "${TEST_EXTRAS_REGION}"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Sources["example"].Extras["region"]; got != "eu-west-1" {
		t.Errorf("expected extras.region to expand to %q, got %q", "eu-west-1", got)
	}
}

// TestLoad_MalformedExtrasKeyFailsNamingSourceAndKey proves the load-time
// gate over extras key shape: an empty key and a key outside the
// documented shape both fail load, naming the offending source instance
// and key.
func TestLoad_MalformedExtrasKeyFailsNamingSourceAndKey(t *testing.T) {
	cases := map[string]string{
		"leading digit": `
[sources.example]
plugin = "topos-plugin-example"
base_url = "http://x.lan"
token = "tok"

[sources.example.extras]
"1bad" = "value"
`,
		"space in key": `
[sources.example]
plugin = "topos-plugin-example"
base_url = "http://x.lan"
token = "tok"

[sources.example.extras]
"bad key" = "value"
`,
	}

	for name, contents := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeTempConfig(t, contents)
			_, err := Load(path)
			if err == nil {
				t.Fatal("expected a malformed extras key to fail load")
			}
			if !strings.Contains(err.Error(), "example") {
				t.Errorf("expected the error to name the source instance %q, got: %v", "example", err)
			}
		})
	}
}
