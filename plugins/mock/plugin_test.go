package main

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

func TestDescribe_ReturnsMockIdentity(t *testing.T) {
	p := NewSourcePlugin()
	resp, err := p.Describe(context.Background(), &toposv1.DescribeRequest{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if resp.GetSourceType() != "mock" {
		t.Errorf("expected source_type %q, got %q", "mock", resp.GetSourceType())
	}
	if resp.GetDisplayName() == "" {
		t.Error("expected a non-empty display_name")
	}
	if resp.GetContractVersion() != "topos.v2" {
		t.Errorf("expected contract_version %q, got %q", "topos.v2", resp.GetContractVersion())
	}
	if len(resp.GetMatchVocabulary()) != 1 || resp.GetMatchVocabulary()[0] != "labels" {
		t.Errorf("expected match_vocabulary [\"labels\"], got %v", resp.GetMatchVocabulary())
	}
}

// matchFieldsReq builds a MatchRequest carrying a single "labels" field —
// the shape the kernel sends this plugin at sync time.
func matchFieldsReq(labels []string) *toposv1.MatchRequest {
	return &toposv1.MatchRequest{MatchFields: map[string]*toposv1.StringList{
		"labels": {Values: labels},
	}}
}

// TestMatch_KeywordMatchingOneItemsLabelReturnsExactlyThatItem proves
// exact, case-insensitive matching: "MEETING" (different case) matches
// items labelled "meeting", and only those.
func TestMatch_KeywordMatchingOneItemsLabelReturnsExactlyThatItem(t *testing.T) {
	p := NewSourcePlugin()
	resp, err := p.Match(context.Background(), matchFieldsReq([]string{"MEETING"}))
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(resp.GetItems()) == 0 {
		t.Fatal("expected at least one matching item")
	}
	for _, it := range resp.GetItems() {
		found := false
		for _, l := range it.GetLabels() {
			if l == "meeting" {
				found = true
			}
		}
		if !found {
			t.Errorf("item %q returned for keyword MEETING but has no 'meeting' label: %v", it.GetSourceId(), it.GetLabels())
		}
	}
}

// TestMatch_NoSubstringMatching proves the contract's exact-match rule:
// a keyword that is a substring of a label (but not equal to it) does not
// match — mirrors the contract's "house" vs "Household" example.
func TestMatch_NoSubstringMatching(t *testing.T) {
	p := NewSourcePlugin()
	resp, err := p.Match(context.Background(), matchFieldsReq([]string{"dem"}))
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(resp.GetItems()) != 0 {
		t.Errorf("expected zero items for a substring-only keyword (no exact label match), got %d", len(resp.GetItems()))
	}
}

// TestMatch_NonMatchingKeywordReturnsZeroItemsAndNilError proves a
// keyword matching nothing returns an empty item list, not an error.
func TestMatch_NonMatchingKeywordReturnsZeroItemsAndNilError(t *testing.T) {
	p := NewSourcePlugin()
	resp, err := p.Match(context.Background(), matchFieldsReq([]string{"no-such-keyword"}))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(resp.GetItems()) != 0 {
		t.Errorf("expected zero items, got %d", len(resp.GetItems()))
	}
}

// TestMatch_UndeclaredKeyIsIgnored proves a match_fields key outside the
// plugin's declared vocabulary ("tags", which the mock never declares) is
// ignored entirely — the plugin matches only on its own declared "labels"
// field, per D-05's "unknown key is absent, not an error" rule.
func TestMatch_UndeclaredKeyIsIgnored(t *testing.T) {
	p := NewSourcePlugin()
	req := &toposv1.MatchRequest{MatchFields: map[string]*toposv1.StringList{
		"labels": {Values: []string{"demo"}},
		"tags":   {Values: []string{"should-be-ignored"}},
	}}
	resp, err := p.Match(context.Background(), req)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(resp.GetItems()) != len(mockItems) {
		t.Fatalf("expected the undeclared 'tags' key to be ignored and all %d 'demo'-labelled items returned, got %d", len(mockItems), len(resp.GetItems()))
	}
}

// TestMatch_EmptyValueListMatchesNothing proves a declared field with an
// empty values list matches nothing for that field, rather than matching
// everything.
func TestMatch_EmptyValueListMatchesNothing(t *testing.T) {
	p := NewSourcePlugin()
	resp, err := p.Match(context.Background(), matchFieldsReq(nil))
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(resp.GetItems()) != 0 {
		t.Errorf("expected zero items for an empty 'labels' value list, got %d", len(resp.GetItems()))
	}
}

// TestMatch_EveryItemHasValidFidelityAndDeepLink mirrors the kernel's own
// correlation-boundary validation (sync-time rejection of an item with
// unspecified fidelity or an empty deep_link) — a future contract change
// that would break this must fail here first, not silently in a real
// plugin.
func TestMatch_EveryItemHasValidFidelityAndDeepLink(t *testing.T) {
	p := NewSourcePlugin()
	// Every fixed item carries the "demo" label, so this returns all of
	// them.
	resp, err := p.Match(context.Background(), matchFieldsReq([]string{"demo"}))
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(resp.GetItems()) != len(mockItems) {
		t.Fatalf("expected all %d fixed items to carry the 'demo' label, got %d", len(mockItems), len(resp.GetItems()))
	}
	for _, it := range resp.GetItems() {
		if it.GetFidelity() == toposv1.LinkFidelity_LINK_FIDELITY_UNSPECIFIED {
			t.Errorf("item %q has unspecified fidelity", it.GetSourceId())
		}
		if it.GetDeepLink() == "" {
			t.Errorf("item %q has an empty deep_link", it.GetSourceId())
		}
	}
}

// TestMatch_AtLeastOneItemHasGroupAndOneDoesNot proves the fixed set
// exercises both the "standalone document" and "thread/conversation"
// shapes the contract's group_id/group_label fields document.
func TestMatch_AtLeastOneItemHasGroupAndOneDoesNot(t *testing.T) {
	var withGroup, withoutGroup bool
	for _, it := range mockItems {
		if it.GetGroupId() != "" {
			withGroup = true
		} else {
			withoutGroup = true
		}
	}
	if !withGroup {
		t.Error("expected at least one fixed item with a non-empty group_id")
	}
	if !withoutGroup {
		t.Error("expected at least one fixed item with an empty group_id")
	}
}

// TestFetch_FullVariantForKnownIDReturnsTextAndNoRendition proves the
// full-variant Fetch reports extracted text, no rendition, and
// available=true — mirroring every other plugin's own convention for "no
// rendition, but usable text" (plugins/proton, plugins/silverbullet):
// Available answers "did Fetch return something to show", not "is a byte
// rendition specifically present". kernel/httpapi/item.go's
// Content.Available and web/src/lib/format.ts's detailPaneState both key
// their branch choice directly off this field — a FULL response that
// carries text but claims Available: false routes the detail pane to its
// "no longer available" state and never surfaces that text at all, which
// is what an earlier version of this test (and this plugin) got wrong.
func TestFetch_FullVariantForKnownIDReturnsTextAndNoRendition(t *testing.T) {
	p := NewSourcePlugin()
	resp, err := p.Fetch(context.Background(), &toposv1.FetchRequest{
		SourceId: "1", Variant: toposv1.ContentVariant_CONTENT_VARIANT_FULL,
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if resp.GetText() == "" {
		t.Error("expected non-empty extracted text")
	}
	if !resp.GetAvailable() {
		t.Error("expected available=true (the mock has usable text, even with no rendition)")
	}
	if resp.GetMimeType() != "" {
		t.Errorf("expected no rendition mime_type, got %q", resp.GetMimeType())
	}
}

// TestFetch_UnknownIDReturnsNotFound proves an unknown source id maps to
// a gRPC codes.NotFound error, per the contract's Fetch section.
func TestFetch_UnknownIDReturnsNotFound(t *testing.T) {
	p := NewSourcePlugin()
	_, err := p.Fetch(context.Background(), &toposv1.FetchRequest{
		SourceId: "does-not-exist", Variant: toposv1.ContentVariant_CONTENT_VARIANT_FULL,
	})
	if err == nil {
		t.Fatal("expected an error for an unknown source id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Errorf("expected codes.NotFound, got %v", err)
	}
}

// TestHealth_AlwaysReachableWithNoError proves the mock always reports
// reachable with no error — it has nothing to be unreachable from.
func TestHealth_AlwaysReachableWithNoError(t *testing.T) {
	p := NewSourcePlugin()
	resp, err := p.Health(context.Background(), &toposv1.HealthRequest{})
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if !resp.GetReachable() {
		t.Error("expected reachable=true")
	}
	if resp.GetLastError() != "" {
		t.Errorf("expected empty last_error, got %q", resp.GetLastError())
	}
}

// TestReadinessWindowFromEnv is a table test over the fixture's env-var
// parsing (readiness.go): absent, empty, and "0" all mean "no window";
// a positive value builds a window; a negative or non-numeric value is a
// loud parse error.
func TestReadinessWindowFromEnv(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		raw       string
		present   bool
		wantNil   bool
		wantError bool
	}{
		{name: "absent", present: false, wantNil: true},
		{name: "empty", raw: "", present: true, wantNil: true},
		{name: "zero", raw: "0", present: true, wantNil: true},
		{name: "positive", raw: "700", present: true, wantNil: false},
		{name: "negative", raw: "-1", present: true, wantError: true},
		{name: "non-numeric", raw: "soon", present: true, wantError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(name string) string {
				if name != readyAfterEnvVar {
					t.Fatalf("unexpected getenv call for %q", name)
				}
				if !tc.present {
					return ""
				}
				return tc.raw
			}

			w, err := readinessWindowFromEnv(now, getenv)

			if tc.wantError {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("readinessWindowFromEnv: %v", err)
			}
			if tc.wantNil && w != nil {
				t.Errorf("expected a nil window, got %+v", w)
			}
			if !tc.wantNil && w == nil {
				t.Error("expected a non-nil window")
			}
		})
	}
}

// TestMatch_ReadinessWindowInFutureRefusesMatchButDescribeStillSucceeds
// proves the guard shape the whole fixture exists for: a plugin whose
// window has not yet elapsed refuses Match with codes.Unavailable and
// notReadyMessage, while Describe — deliberately never gated — still
// succeeds (mirroring pluginhost.launch's post-handshake Describe call,
// which must never observe the window).
func TestMatch_ReadinessWindowInFutureRefusesMatchButDescribeStillSucceeds(t *testing.T) {
	now := time.Now()
	p := NewSourcePlugin().withReadinessWindow(&readinessWindow{readyAt: now.Add(time.Hour)})

	_, err := p.Match(context.Background(), matchFieldsReq([]string{"demo"}))
	if err == nil {
		t.Fatal("expected an error for a Match call inside the readiness window")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unavailable {
		t.Errorf("expected codes.Unavailable, got %v", err)
	}
	if st.Message() != notReadyMessage {
		t.Errorf("expected message %q, got %q", notReadyMessage, st.Message())
	}

	if _, err := p.Describe(context.Background(), &toposv1.DescribeRequest{}); err != nil {
		t.Errorf("expected Describe to succeed regardless of the readiness window, got %v", err)
	}
}

// TestMatch_ReadinessWindowElapsedReturnsNormalItemSet proves a window
// constructed from a past readyAt (never a sleep) behaves exactly like an
// unconfigured plugin.
func TestMatch_ReadinessWindowElapsedReturnsNormalItemSet(t *testing.T) {
	now := time.Now()
	p := NewSourcePlugin().withReadinessWindow(&readinessWindow{readyAt: now.Add(-time.Hour)})

	resp, err := p.Match(context.Background(), matchFieldsReq([]string{"demo"}))
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(resp.GetItems()) != len(mockItems) {
		t.Errorf("expected all %d fixed 'demo'-labelled items once the window has elapsed, got %d", len(mockItems), len(resp.GetItems()))
	}
}
