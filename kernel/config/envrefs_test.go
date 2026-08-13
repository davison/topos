package config

import (
	"reflect"
	"testing"
)

// TestEnvRefNames_ScansNestedStructMapSliceForBothReferenceForms proves
// EnvRefNames' full reach: a nested struct field, a map value (an extras
// entry), and a slice element all contribute referenced names, in both the
// ${NAME} and bare $NAME forms os.Expand itself recognises, sorted and
// de-duplicated.
func TestEnvRefNames_ScansNestedStructMapSliceForBothReferenceForms(t *testing.T) {
	cfg := &Config{
		Sources: map[string]Source{
			"paperless": {
				Plugin:  "topos-plugin-paperless",
				BaseURL: "${PAPERLESS_URL}",
				Token:   "$PAPERLESS_TOKEN", // bare form
				Extras:  map[string]string{"region": "${PAPERLESS_REGION}"},
			},
		},
		Webspaces: map[string]Webspace{
			"house-move": {Keywords: []string{"${NOT_REALLY_A_KEYWORD}"}},
		},
	}

	got := EnvRefNames(cfg)
	want := []string{"NOT_REALLY_A_KEYWORD", "PAPERLESS_REGION", "PAPERLESS_TOKEN", "PAPERLESS_URL"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EnvRefNames = %v, want %v", got, want)
	}
}

// TestEnvRefNames_DeduplicatesRepeatedReferences proves the same variable
// name referenced from two different fields contributes exactly one entry.
func TestEnvRefNames_DeduplicatesRepeatedReferences(t *testing.T) {
	cfg := &Config{
		Sources: map[string]Source{
			"a": {BaseURL: "${SHARED_VAR}", Token: "${SHARED_VAR}"},
		},
	}
	got := EnvRefNames(cfg)
	want := []string{"SHARED_VAR"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EnvRefNames = %v, want %v", got, want)
	}
}

// TestEnvRefNames_NoReferencesReturnsEmptyNeverNil proves a config with no
// ${VAR}/$VAR references anywhere returns an empty (never nil) slice.
func TestEnvRefNames_NoReferencesReturnsEmptyNeverNil(t *testing.T) {
	cfg := &Config{Sources: map[string]Source{"a": {Plugin: "topos-plugin-mock", BaseURL: "http://mock.test"}}}
	got := EnvRefNames(cfg)
	if got == nil {
		t.Fatal("expected a non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected zero references, got %v", got)
	}
}

// TestEnvRefNames_NilPointerDoesNotPanic proves a nil *Config (or a nil
// pointer reached partway through the scan) is handled gracefully, never a
// panic.
func TestEnvRefNames_NilPointerDoesNotPanic(t *testing.T) {
	var cfg *Config
	got := EnvRefNames(cfg)
	if len(got) != 0 {
		t.Fatalf("expected zero references for a nil *Config, got %v", got)
	}
}

// TestEnvRefNames_ExtrasMapIncluded proves a Source.Extras value's ${VAR}
// reference is reachable — the exact shape D-14's env allowlist depends on
// (an operator-authored extras value referencing a secret env var must be
// found by the identical scan GET /api/config's env_vars field uses).
func TestEnvRefNames_ExtrasMapIncluded(t *testing.T) {
	src := Source{Extras: map[string]string{"api_key": "${THIRD_PARTY_KEY}"}}
	got := EnvRefNames(src)
	want := []string{"THIRD_PARTY_KEY"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EnvRefNames(src) = %v, want %v", got, want)
	}
}
