// env_test.go pins allowedEnv's contract (T-11-08, D-14): a launched
// plugin subprocess receives ONLY the documented desktop-session allowlist
// plus the values behind the ${VAR} references this instance's own raw
// config actually declares — never the kernel's remaining environment,
// regardless of what go-plugin itself might otherwise append (see
// SkipHostEnv on the goplugin.ClientConfig launch builds, host.go).
package pluginhost

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/davison/topos/kernel/config"
)

// TestAllowedEnv_ReferencedVariableIsPresentUnreferencedIsAbsent is this
// file's headline proof: with TOPOS_TEST_REFERENCED set and referenced by
// this instance's own raw config, and TOPOS_TEST_UNREFERENCED set but
// referenced NOWHERE, allowedEnv's output contains the former and
// contains no entry at all named after the latter.
func TestAllowedEnv_ReferencedVariableIsPresentUnreferencedIsAbsent(t *testing.T) {
	t.Setenv("TOPOS_TEST_UNREFERENCED", "leak")
	t.Setenv("TOPOS_TEST_REFERENCED", "ok")

	rawSrc := config.Source{
		Plugin:  "topos-plugin-example",
		BaseURL: "${TOPOS_TEST_REFERENCED}",
	}

	env := allowedEnv(rawSrc, []byte(`{}`), false)

	if !containsEnvEntry(env, "TOPOS_TEST_REFERENCED", "ok") {
		t.Errorf("expected TOPOS_TEST_REFERENCED=ok in the produced environment, got: %v", env)
	}
	for _, e := range env {
		if hasEnvName(e, "TOPOS_TEST_UNREFERENCED") {
			t.Errorf("expected NO entry named TOPOS_TEST_UNREFERENCED (it is set on the kernel process but referenced nowhere in this instance's raw config), got: %v", env)
		}
	}
}

// TestAllowedEnv_DesktopSessionAllowlistCopiedWhenSet proves PATH and HOME
// (part of the fixed, documented allowlist) are present when set on the
// kernel process, regardless of whether this instance's raw config
// references them at all.
func TestAllowedEnv_DesktopSessionAllowlistCopiedWhenSet(t *testing.T) {
	rawSrc := config.Source{Plugin: "topos-plugin-example"}
	env := allowedEnv(rawSrc, []byte(`{}`), false)

	if path, ok := os.LookupEnv("PATH"); ok {
		if !containsEnvEntry(env, "PATH", path) {
			t.Errorf("expected PATH=%q in the produced environment, got: %v", path, env)
		}
	}
	if home, ok := os.LookupEnv("HOME"); ok {
		if !containsEnvEntry(env, "HOME", home) {
			t.Errorf("expected HOME=%q in the produced environment, got: %v", home, env)
		}
	}
}

// TestAllowedEnv_UnsetAllowlistedVariableContributesNoEntry proves the
// allowlist copies present-only: a documented allowlist name that is UNSET
// on the kernel process must never appear as an empty-string entry.
func TestAllowedEnv_UnsetAllowlistedVariableContributesNoEntry(t *testing.T) {
	unsetName := "TOPOS_TEST_DEFINITELY_UNSET_ALLOWLIST_VAR"
	os.Unsetenv(unsetName)

	// Sanity: this name isn't actually one of the real allowlisted names —
	// this test instead proves the GENERAL "present-only" mechanism by
	// checking that no entry in the output is a bare "NAME=" for any
	// allowlisted name that happens to be unset in this environment.
	rawSrc := config.Source{Plugin: "topos-plugin-example"}
	env := allowedEnv(rawSrc, []byte(`{}`), false)

	for _, name := range allowedDesktopSessionEnvVars {
		if _, ok := os.LookupEnv(name); ok {
			continue // legitimately set — covered by the other test above
		}
		for _, e := range env {
			if hasEnvName(e, name) {
				t.Errorf("expected no entry for unset allowlisted variable %q, got: %v", name, env)
			}
		}
	}
}

// TestAllowedEnv_ExactlyOneSourceConfigEntry proves the envelope variable
// appears exactly once, with the exact bytes passed in.
func TestAllowedEnv_ExactlyOneSourceConfigEntry(t *testing.T) {
	rawSrc := config.Source{Plugin: "topos-plugin-example"}
	payload := []byte(`{"base_url":"http://x"}`)
	env := allowedEnv(rawSrc, payload, false)

	want := "WEBSPACES_SOURCE_CONFIG=" + string(payload)
	count := 0
	for _, e := range env {
		if e == want {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one %q entry, found %d in: %v", want, count, env)
	}
}

// TestAllowedEnv_DescribeOnlyAddsMarker proves WEBSPACES_DESCRIBE_ONLY=1 is
// present for a trial launch and absent otherwise.
func TestAllowedEnv_DescribeOnlyAddsMarker(t *testing.T) {
	rawSrc := config.Source{Plugin: "topos-plugin-example"}

	describeEnv := allowedEnv(rawSrc, []byte(`{}`), true)
	if !containsEnvEntry(describeEnv, "WEBSPACES_DESCRIBE_ONLY", "1") {
		t.Errorf("expected WEBSPACES_DESCRIBE_ONLY=1 for a describe-only launch, got: %v", describeEnv)
	}

	realEnv := allowedEnv(rawSrc, []byte(`{}`), false)
	for _, e := range realEnv {
		if hasEnvName(e, "WEBSPACES_DESCRIBE_ONLY") {
			t.Errorf("expected no WEBSPACES_DESCRIBE_ONLY entry for a real (non-describe-only) launch, got: %v", realEnv)
		}
	}
}

// TestSourceConfigEnvelope_RecursiveKeyPresentOutsideExtras proves
// 12-03-PLAN.md Task 1's envelope contract: a top-level "recursive"
// boolean reflecting the source's own value is always present, alongside
// the existing typed keys and OUTSIDE the nested "extras" object.
func TestSourceConfigEnvelope_RecursiveKeyPresentOutsideExtras(t *testing.T) {
	envelope := sourceConfigEnvelope{
		Path:      "/mnt/docs",
		Recursive: true,
		Extras:    map[string]string{"include_glob": "**/*.pdf"},
	}

	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	got, ok := decoded["recursive"].(bool)
	if !ok {
		t.Fatalf("expected a top-level \"recursive\" boolean, got: %s", raw)
	}
	if !got {
		t.Errorf("expected recursive == true, got false")
	}

	extras, ok := decoded["extras"].(map[string]any)
	if !ok {
		t.Fatalf("expected a nested \"extras\" object, got: %s", raw)
	}
	if _, ok := extras["recursive"]; ok {
		t.Errorf("expected \"recursive\" to stay OUTSIDE the nested extras object, got: %s", raw)
	}
}

// TestSourceConfigEnvelope_RecursiveFalseStillEmitsKey proves recursive is
// NOT omitempty on the envelope: false is a meaningful, present value
// distinct from "field absent", unlike Extras' legitimate absent state.
func TestSourceConfigEnvelope_RecursiveFalseStillEmitsKey(t *testing.T) {
	envelope := sourceConfigEnvelope{Path: "/mnt/docs"}

	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	got, ok := decoded["recursive"]
	if !ok {
		t.Fatalf("expected a \"recursive\" key even when false, got: %s", raw)
	}
	if got != false {
		t.Errorf("expected recursive == false, got %v", got)
	}
}

func hasEnvName(entry, name string) bool {
	return len(entry) > len(name) && entry[:len(name)] == name && entry[len(name)] == '='
}

func containsEnvEntry(env []string, name, value string) bool {
	want := name + "=" + value
	for _, e := range env {
		if e == want {
			return true
		}
	}
	return false
}
