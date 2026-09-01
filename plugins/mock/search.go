package main

import (
	"context"
	"sort"
	"strings"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Search (M2-R2, davison/topos#50) — the mock's sdk.ContentSearcher: a
// reference implementation the kernel's suite and the e2e harness prove
// against. It searches title, preview and the fixture body
// (mockFullText) of every item that is a MEMBER under req.match_fields
// (the same rule Match applies — labels against the keywords), refuses
// an empty map rather than searching everything, ANDs every
// required_terms entry with the query, honours limit and reports
// truncated, and says where each hit matched. Bounded snippet; never a
// body.
func (p *SourcePlugin) Search(ctx context.Context, req *toposv1.SearchRequest) (*toposv1.SearchResponse, error) {
	if len(req.GetMatchFields()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "search: match_fields is empty — a search without membership input would be the whole source, which is refused (kernel/correlate's own rule)")
	}
	if err := p.waitSearchDelay(ctx); err != nil {
		return nil, err
	}
	terms := searchTerms(req.GetQuery())
	if len(terms) == 0 {
		return &toposv1.SearchResponse{Hits: []*toposv1.SearchHit{}}, nil
	}
	required := make([]string, 0, len(req.GetRequiredTerms()))
	for _, t := range req.GetRequiredTerms() {
		if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
			required = append(required, t)
		}
	}
	keywords := req.GetMatchFields()["labels"].GetValues()

	var hits []*toposv1.SearchHit
	for _, it := range append(append([]*toposv1.Item{}, mockItems...), searchOnlyItems...) {
		if !labelsMatchAnyKeyword(it.GetLabels(), keywords) {
			continue // not a member — never returned, whatever the query
		}
		title := strings.ToLower(it.GetTitle())
		preview := strings.ToLower(it.GetPreview())
		body := strings.ToLower(fullTextFor(it.GetSourceId()))
		labels := strings.ToLower(strings.Join(it.GetLabels(), " "))
		all := title + " " + preview + " " + body + " " + labels
		ok := true
		for _, t := range append(append([]string{}, terms...), required...) {
			if !strings.Contains(all, t) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		where := toposv1.MatchedIn_MATCHED_IN_BODY
		switch {
		case containsAll(title, terms):
			where = toposv1.MatchedIn_MATCHED_IN_TITLE
		case containsAll(body, terms):
			where = toposv1.MatchedIn_MATCHED_IN_BODY
		case containsAll(labels, terms):
			where = toposv1.MatchedIn_MATCHED_IN_LABELS
		default:
			where = toposv1.MatchedIn_MATCHED_IN_TITLE // a match spread across title+preview reads as the item's own summary
		}
		hits = append(hits, &toposv1.SearchHit{Item: it, Snippet: snippetAround(fullTextFor(it.GetSourceId()), terms[0]), MatchedIn: where})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].GetItem().GetSourceId() < hits[j].GetItem().GetSourceId() })
	truncated := false
	if limit := int(req.GetLimit()); limit > 0 && len(hits) > limit {
		hits, truncated = hits[:limit], true
	}
	if hits == nil {
		hits = []*toposv1.SearchHit{}
	}
	return &toposv1.SearchResponse{Hits: hits, Truncated: truncated, Note: "the mock searched titles, previews, labels and fixture bodies"}, nil
}

func searchTerms(q string) []string {
	var out []string
	for _, f := range strings.Fields(strings.ToLower(q)) {
		if len(f) >= 2 {
			out = append(out, f)
		}
	}
	return out
}

func containsAll(s string, terms []string) bool {
	for _, t := range terms {
		if !strings.Contains(s, t) {
			return false
		}
	}
	return true
}

// snippetAround returns a bounded window of body around the first
// occurrence of term (case-insensitive), or the body's head — never the
// whole body.
func snippetAround(body, term string) string {
	const window = 60
	if body == "" {
		return ""
	}
	i := strings.Index(strings.ToLower(body), term)
	if i < 0 {
		i = 0
	}
	start := i - window
	if start < 0 {
		start = 0
	}
	end := i + len(term) + window
	if end > len(body) {
		end = len(body)
	}
	snip := body[start:end]
	if start > 0 {
		snip = "…" + snip
	}
	if end < len(body) {
		snip += "…"
	}
	return snip
}

// fullTextFor is the body text Search reads for an item — the Fetch
// fixture's for a listed item, the search-only table's for an orphan.
func fullTextFor(sourceID string) string {
	if body, ok := searchOnlyFullText[sourceID]; ok {
		return body
	}
	return mockFullText[sourceID]
}
