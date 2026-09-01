package pluginhost

import (
	"context"
	"testing"
	"time"

	"github.com/davison/topos/kernel/config"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeSearchImpl stands in for a plugin process: Match/Fetch/Health are
// never called by the fan-out; Search answers per instance.
type fakeSearchImpl struct {
	hits      int
	delay     time.Duration
	err       error
	gotFields map[string]*toposv1.StringList
	gotReq    []string
}

func (f *fakeSearchImpl) Describe(context.Context, *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error) {
	return &toposv1.DescribeResponse{}, nil
}
func (f *fakeSearchImpl) Match(context.Context, *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	return &toposv1.MatchResponse{}, nil
}
func (f *fakeSearchImpl) Fetch(context.Context, *toposv1.FetchRequest) (*toposv1.FetchResponse, error) {
	return &toposv1.FetchResponse{}, nil
}
func (f *fakeSearchImpl) Health(context.Context, *toposv1.HealthRequest) (*toposv1.HealthResponse, error) {
	return &toposv1.HealthResponse{}, nil
}
func (f *fakeSearchImpl) Search(ctx context.Context, req *toposv1.SearchRequest) (*toposv1.SearchResponse, error) {
	f.gotFields, f.gotReq = req.GetMatchFields(), req.GetRequiredTerms()
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	resp := &toposv1.SearchResponse{Note: "fake"}
	for i := 0; i < f.hits; i++ {
		resp.Hits = append(resp.Hits, &toposv1.SearchHit{Item: &toposv1.Item{SourceId: "h", Title: "hit"}, MatchedIn: toposv1.MatchedIn_MATCHED_IN_BODY})
	}
	resp.Hits = append(resp.Hits, &toposv1.SearchHit{}) // an item-less hit: dropped by shape validation
	return resp, nil
}

// TestSearchSources_FanOut (M2-R2; PR #60 review round 1 — run under
// -race): searchable and unsupported instances are answered concurrently
// into their own slots; only participating instances with resolved
// membership input are asked; the saved filter rides as required terms;
// Unimplemented, timeout and error are classified; item-less hits drop.
func TestSearchSources_FanOut(t *testing.T) {
	fast := &fakeSearchImpl{hits: 2}
	slow := &fakeSearchImpl{hits: 1, delay: 20 * time.Millisecond}
	unimpl := &fakeSearchImpl{err: status.Error(codes.Unimplemented, "no")}
	broken := &fakeSearchImpl{err: status.Error(codes.Internal, "boom")}
	mk := func(name string, impl *fakeSearchImpl, searches bool) *Plugin {
		return &Plugin{name: name, displayName: name, sourceType: "fake", matchVocabulary: []string{"labels"}, impl: impl, searchesContent: searches}
	}
	h := &Host{plugins: []*Plugin{
		mk("a-fast", fast, true),
		mk("b-slow", slow, true),
		mk("c-declines", &fakeSearchImpl{}, false),
		mk("d-unimpl", unimpl, true),
		mk("e-broken", broken, true),
		mk("f-outside", &fakeSearchImpl{hits: 9}, true),
	}}
	ws := config.Webspace{Keywords: []string{"boiler"}, Sources: []string{"a-fast", "b-slow", "c-declines", "d-unimpl", "e-broken"}}

	out := h.SearchSources(context.Background(), ws, "boiler", []string{"invoice"})
	byName := map[string]SourceSearchOutcome{}
	for _, o := range out {
		byName[o.Instance] = o
	}
	if _, asked := byName["f-outside"]; asked {
		t.Error("an instance outside the webspace must never be asked")
	}
	if got := byName["a-fast"]; got.Status != SearchStatusOK || len(got.Hits) != 2 || got.Note != "fake" {
		t.Errorf("a-fast: %+v", got)
	}
	if got := byName["b-slow"]; got.Status != SearchStatusOK || len(got.Hits) != 1 {
		t.Errorf("b-slow: %+v", got)
	}
	if got := byName["c-declines"]; got.Status != SearchStatusUnsupported {
		t.Errorf("c-declines: %+v", got)
	}
	if got := byName["d-unimpl"]; got.Status != SearchStatusUnsupported {
		t.Errorf("d-unimpl (Unimplemented over the wire): %+v", got)
	}
	if got := byName["e-broken"]; got.Status != SearchStatusError || got.Error == "" {
		t.Errorf("e-broken: %+v", got)
	}
	if fast.gotFields["labels"] == nil || fast.gotFields["labels"].GetValues()[0] != "boiler" || len(fast.gotReq) != 1 || fast.gotReq[0] != "invoice" {
		t.Errorf("the request must carry the resolved membership input and the saved filter: %+v %v", fast.gotFields, fast.gotReq)
	}
	for i := 1; i < len(out); i++ {
		if out[i-1].Instance > out[i].Instance {
			t.Errorf("outcomes must be in instance order: %v", out)
		}
	}
}

// TestSearchSources_NoMembershipInputNoCall: a webspace with neither
// keywords nor a match block asks nobody — fail closed, correlate's rule.
func TestSearchSources_NoMembershipInputNoCall(t *testing.T) {
	impl := &fakeSearchImpl{hits: 1}
	h := &Host{plugins: []*Plugin{{name: "a", matchVocabulary: []string{"labels"}, impl: impl, searchesContent: true}}}
	out := h.SearchSources(context.Background(), config.Webspace{Sources: []string{"a"}}, "boiler", nil)
	if len(out) != 0 || impl.gotFields != nil {
		t.Errorf("expected no call and no outcome, got %+v (called: %v)", out, impl.gotFields != nil)
	}
}

// TestSearchSources_PerInstanceFilterRidesAsRequiredTerms (M2-R3, #55):
// an instance named in the webspace's filter_by_source map receives the
// global filter AND its own terms as required_terms; every other
// instance receives the global filter alone, untouched.
func TestSearchSources_PerInstanceFilterRidesAsRequiredTerms(t *testing.T) {
	narrowed := &fakeSearchImpl{hits: 1}
	untouched := &fakeSearchImpl{hits: 1}
	h := &Host{plugins: []*Plugin{
		{name: "a-narrowed", displayName: "a", sourceType: "fake", matchVocabulary: []string{"labels"}, impl: narrowed, searchesContent: true},
		{name: "b-untouched", displayName: "b", sourceType: "fake", matchVocabulary: []string{"labels"}, impl: untouched, searchesContent: true},
	}}
	ws := config.Webspace{
		Keywords:       []string{"boiler"},
		FilterBySource: map[string][]string{"a-narrowed": {"quote", "2026"}},
	}
	h.SearchSources(context.Background(), ws, "boiler", []string{"invoice"})
	if got := narrowed.gotReq; len(got) != 3 || got[0] != "invoice" || got[1] != "quote" || got[2] != "2026" {
		t.Errorf("narrowed instance must receive global + its own terms: %v", got)
	}
	if got := untouched.gotReq; len(got) != 1 || got[0] != "invoice" {
		t.Errorf("the other instance must receive the global filter alone: %v", got)
	}
}
