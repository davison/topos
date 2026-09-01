package main

import (
	"context"
	"testing"
	"time"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func searchReq(q string, keywords []string, required ...string) *toposv1.SearchRequest {
	req := &toposv1.SearchRequest{Query: q, RequiredTerms: required}
	if keywords != nil {
		req.MatchFields = map[string]*toposv1.StringList{"labels": {Values: keywords}}
	}
	return req
}

// TestSearch_RefusesEmptyMembership (M2-R2): no match_fields, no search —
// never "the whole source".
func TestSearch_RefusesEmptyMembership(t *testing.T) {
	p := &SourcePlugin{}
	_, err := p.Search(context.Background(), searchReq("standup", nil))
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
	_, err = p.Search(context.Background(), &toposv1.SearchRequest{Query: "standup", MatchFields: map[string]*toposv1.StringList{}})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("an empty map is refused too, got %v", err)
	}
}

// TestSearch_SearchesWithinMembership: only members (labels matching the
// keywords) are ever returned; the query reaches the fixture bodies.
func TestSearch_SearchesWithinMembership(t *testing.T) {
	p := &SourcePlugin{}
	// "mocked" appears only in item 4's fixture BODY ("a shopping list,
	// mocked."), never in its title or preview; its label is "demo", not
	// "meeting".
	resp, err := p.Search(context.Background(), searchReq("mocked", []string{"meeting"}))
	if err != nil || len(resp.GetHits()) != 0 {
		t.Fatalf("a non-member must not be returned however well it matches: %v %+v", err, resp)
	}
	resp, err = p.Search(context.Background(), searchReq("mocked", []string{"demo"}))
	if err != nil || len(resp.GetHits()) != 1 || resp.GetHits()[0].GetItem().GetSourceId() != "4" {
		t.Fatalf("expected item 4 by its body: %v %+v", err, resp)
	}
	if resp.GetHits()[0].GetMatchedIn() != toposv1.MatchedIn_MATCHED_IN_BODY {
		t.Errorf("matched in the body, got %v", resp.GetHits()[0].GetMatchedIn())
	}
	if len(resp.GetHits()[0].GetSnippet()) == 0 || len(resp.GetHits()[0].GetSnippet()) > 200 {
		t.Errorf("a bounded snippet, never the body: %q", resp.GetHits()[0].GetSnippet())
	}
}

// TestSearch_RequiredTermsLimitAndTitle: required terms AND with the
// query; a title match says so; limit truncates and says so.
func TestSearch_RequiredTermsLimitAndTitle(t *testing.T) {
	p := &SourcePlugin{}
	resp, err := p.Search(context.Background(), searchReq("standup", []string{"meeting"}))
	if err != nil || len(resp.GetHits()) != 2 {
		t.Fatalf("both standups: %v %+v", err, resp)
	}
	if resp.GetHits()[0].GetMatchedIn() != toposv1.MatchedIn_MATCHED_IN_TITLE {
		t.Errorf("'standup' is in the title, got %v", resp.GetHits()[0].GetMatchedIn())
	}
	resp, _ = p.Search(context.Background(), searchReq("standup", []string{"meeting"}, "wednesday"))
	if len(resp.GetHits()) != 1 || resp.GetHits()[0].GetItem().GetSourceId() != "3" {
		t.Errorf("a required term narrows to item 3: %+v", resp)
	}
	req := searchReq("standup", []string{"meeting"})
	req.Limit = 1
	resp, _ = p.Search(context.Background(), req)
	if len(resp.GetHits()) != 1 || !resp.GetTruncated() {
		t.Errorf("limit 1 truncates and says so: %+v", resp)
	}
}

// A search-only item is found by Search yet never listed by Match — the
// unsynced-hit shape the fan-out marks indexed:false (davison/topos#54).
func TestSearch_OrphanIsFoundButNeverMatched(t *testing.T) {
	p := NewSourcePlugin()
	req := &toposv1.SearchRequest{Query: "orphaned", MatchFields: map[string]*toposv1.StringList{"labels": {Values: []string{"demo"}}}}
	resp, err := p.Search(context.Background(), req)
	if err != nil || len(resp.GetHits()) != 1 || resp.GetHits()[0].GetItem().GetSourceId() != "5" {
		t.Fatalf("orphan search: hits=%v err=%v", resp.GetHits(), err)
	}
	m, err := p.Match(context.Background(), &toposv1.MatchRequest{MatchFields: req.MatchFields})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	for _, it := range m.GetItems() {
		if it.GetSourceId() == "5" {
			t.Fatalf("Match listed the search-only item")
		}
	}
}

func TestSearchDelayFromEnv(t *testing.T) {
	for raw, want := range map[string]time.Duration{"": 0, "0": 0, "250": 250 * time.Millisecond} {
		got, err := searchDelayFromEnv(func(string) string { return raw })
		if err != nil || got != want {
			t.Fatalf("%q: got %v, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"-1", "soon", "1.5"} {
		if _, err := searchDelayFromEnv(func(string) string { return raw }); err == nil {
			t.Fatalf("%q: want an error", raw)
		}
	}
	// The delay yields to the kernel's budget rather than outliving it.
	p := NewSourcePlugin().withSearchDelay(time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := p.Search(ctx, &toposv1.SearchRequest{Query: "x", MatchFields: map[string]*toposv1.StringList{"labels": {Values: []string{"demo"}}}}); err == nil {
		t.Fatal("delayed Search ignored the cancelled context")
	}
}
