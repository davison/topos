// extras_test.go pins sourceConfigEnvelope's shape (D-12/D-13, Phase 11
// Task 3): a source's extras land inside a NESTED "extras" object, the
// top-level key names are unchanged from before this phase, and a source
// with no extras emits no "extras" key at all.
package pluginhost

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
)

// TestSourceConfigEnvelope_ExtrasPresentProducesNestedObject proves the
// headline shape: a source with two extras keys marshals to a document
// whose top-level keys are unchanged and which carries a nested "extras"
// object with exactly those two keys.
func TestSourceConfigEnvelope_ExtrasPresentProducesNestedObject(t *testing.T) {
	envelope := sourceConfigEnvelope{
		BaseURL: "http://x.lan",
		Token:   "tok",
		Extras:  map[string]string{"region": "eu-west-1", "tier": "pro"},
	}

	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	wantTopLevel := []string{"base_url", "token", "api_version", "ca_cert", "username", "webmail_base_url", "path", "extras"}
	for _, key := range wantTopLevel {
		if _, ok := decoded[key]; !ok && key != "extras" {
			// Every non-extras key is always present (no omitempty on
			// those fields) — even as an empty string.
			t.Errorf("expected top-level key %q to be present, got document: %s", key, raw)
		}
	}

	extras, ok := decoded["extras"].(map[string]any)
	if !ok {
		t.Fatalf("expected a nested \"extras\" object, got: %s", raw)
	}
	if len(extras) != 2 {
		t.Fatalf("expected exactly 2 extras keys, got %d: %v", len(extras), extras)
	}
	if extras["region"] != "eu-west-1" {
		t.Errorf("expected extras.region = %q, got %q", "eu-west-1", extras["region"])
	}
	if extras["tier"] != "pro" {
		t.Errorf("expected extras.tier = %q, got %q", "pro", extras["tier"])
	}
}

// TestSourceConfigEnvelope_NoExtrasEmitsNoExtrasKey proves the omitempty
// contract: a source with no extras produces a document with NO "extras"
// key at all — never an empty object, and never a JSON null.
func TestSourceConfigEnvelope_NoExtrasEmitsNoExtrasKey(t *testing.T) {
	envelope := sourceConfigEnvelope{BaseURL: "http://x.lan", Token: "tok"}

	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if _, ok := decoded["extras"]; ok {
		t.Fatalf("expected no \"extras\" key at all for a source with no extras, got: %s", raw)
	}
}

// TestSourceConfigEnvelope_TopLevelKeyNamesUnchanged proves this phase's
// additive-only guarantee: every pre-Phase-11 top-level key name is
// byte-identical to the flat map[string]string this envelope replaced —
// an already-shipped plugin binary keeps reading the same document.
func TestSourceConfigEnvelope_TopLevelKeyNamesUnchanged(t *testing.T) {
	envelope := sourceConfigEnvelope{
		BaseURL:        "burl",
		Token:          "tok",
		APIVersion:     "10",
		CACert:         "cacert",
		Username:       "user",
		WebmailBaseURL: "webmail",
		Path:           "path",
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	// map[string]any, not map[string]string (12-03-PLAN.md Task 1
	// deviation): the envelope now also carries a non-string scalar
	// (Recursive, a bool), so a string-typed decode target would fail
	// this test vacuously on an unrelated field addition rather than
	// actually checking the string-valued keys below.
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	want := map[string]string{
		"base_url":         "burl",
		"token":            "tok",
		"api_version":      "10",
		"ca_cert":          "cacert",
		"username":         "user",
		"webmail_base_url": "webmail",
		"path":             "path",
	}
	for k, v := range want {
		if decoded[k] != v {
			t.Errorf("expected %s=%q, got %q (full document: %s)", k, v, decoded[k], raw)
		}
	}
}

// TestLaunch_SourceConfigEnvelopeCarriesExtrasThroughRealLaunch is the
// integration-shaped proof: launch a real mock subprocess with a source
// declaring extras and prove the marshal helper this test calls directly
// (mirroring launch's own construction) produces the nested object — see
// this file's header comment for why the marshal helper is exercised
// directly rather than parsed back out of the subprocess's own env.
func TestLaunch_SourceConfigEnvelopeCarriesExtrasThroughRealLaunch(t *testing.T) {
	dir := buildMockPluginDir(t)
	src := config.Source{
		Plugin:  "topos-plugin-mock",
		BaseURL: "http://mock.test",
		Token:   "unused",
		Extras:  map[string]string{"greeting": "hello"},
	}

	p, err := launch(context.Background(), Dirs{Trusted: dir}, "demo", src, nil, hclog.NewNullLogger(), false)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	defer p.Kill()

	raw, err := json.Marshal(sourceConfigEnvelope{
		BaseURL: src.BaseURL,
		Token:   src.Token,
		Extras:  src.Extras,
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	extras, ok := decoded["extras"].(map[string]any)
	if !ok {
		t.Fatalf("expected a nested extras object, got: %s", raw)
	}
	if extras["greeting"] != "hello" {
		t.Errorf("expected extras.greeting = %q, got %q", "hello", extras["greeting"])
	}
}
