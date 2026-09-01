package pluginhost

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/correlate"
	"github.com/davison/topos/kernel/item"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SearchBudget bounds one source's answer to a fan-out (M2-R2,
// davison/topos#50): a slow source times out by name; it never delays the
// index's own answer, which the HTTP layer serves first (scope=index).
const SearchBudget = 5 * time.Second

// SearchLimit is the per-source hit cap the kernel asks for.
const SearchLimit = 50

// Source search outcome vocabulary — CLOSED, mirrored in docs/api.md.
const (
	SearchStatusOK          = "ok"
	SearchStatusUnsupported = "unsupported"
	SearchStatusTimeout     = "timeout"
	SearchStatusError       = "error"
)

// SearchHit is one source-reported hit, converted to the index's own item
// shape so the HTTP layer renders and merges it like any other row.
type SearchHit struct {
	Item      item.Item
	Snippet   string
	MatchedIn string // "title" | "body" | "labels" | "attachment" | ""
}

// SourceSearchOutcome is one participating instance's answer.
type SourceSearchOutcome struct {
	Instance    string
	DisplayName string
	Status      string
	Hits        []SearchHit
	Truncated   bool
	Note        string
	Error       string
	ElapsedMS   int64
}

// SearchSources fans a query out to every running instance that
// participates in ws WITH resolved membership input — the same rule
// correlate applies to sync: no match input, no call (fail closed) — and
// declares the Search capability; each under SearchBudget, in parallel.
// The instance's resolved match_fields ride in the request, as do the
// webspace's saved filter terms (required), so the source ANDs both.
// Membership of the returned hits is the source's promise, trusted
// exactly as sync trusts its Match result set (docs/plugin-contract.md,
// "Search"); the kernel enforces its own side: participation, request
// construction, shape validation, attribution.
func (h *Host) SearchSources(ctx context.Context, ws config.Webspace, query string, required []string) []SourceSearchOutcome {
	plugins := h.snapshot()
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].name < plugins[j].name })
	// Every outcome is written to its OWN preallocated slot — one per
	// participating instance, decided before any goroutine starts — so
	// the synchronous "unsupported" outcomes and the concurrent RPC
	// outcomes never touch shared state (PR #60 review round 1).
	type slot struct {
		p      *Plugin
		fields map[string][]string
	}
	var slots []slot
	for _, p := range plugins {
		fields, participates, _ := correlate.MatchFieldsFor(ws, p)
		if !participates || len(fields) == 0 {
			continue
		}
		slots = append(slots, slot{p: p, fields: fields})
	}
	out := make([]SourceSearchOutcome, len(slots))
	var wg sync.WaitGroup
	for i, sl := range slots {
		if !sl.p.searchesContent {
			out[i] = SourceSearchOutcome{Instance: sl.p.name, DisplayName: sl.p.displayName, Status: SearchStatusUnsupported}
			continue
		}
		// This instance's own filter_by_source terms ride WITH the
		// webspace's global filter as required_terms (M2-R3, #55) — the
		// source ANDs both, exactly as the index query does.
		instanceRequired := required
		if extra := ws.FilterBySource[sl.p.name]; len(extra) > 0 {
			instanceRequired = append(append([]string{}, required...), extra...)
		}
		wg.Add(1)
		go func(i int, p *Plugin, fields map[string][]string, required []string) {
			defer wg.Done()
			out[i] = h.searchOne(ctx, p, query, required, fields)
		}(i, sl.p, sl.fields, instanceRequired)
	}
	wg.Wait()
	return out
}

func (h *Host) searchOne(ctx context.Context, p *Plugin, query string, required []string, fields map[string][]string) SourceSearchOutcome {
	res := SourceSearchOutcome{Instance: p.name, DisplayName: p.displayName}
	req := &toposv1.SearchRequest{Query: query, Limit: SearchLimit, MatchFields: make(map[string]*toposv1.StringList, len(fields)), RequiredTerms: required}
	for k, v := range fields {
		req.MatchFields[k] = &toposv1.StringList{Values: v}
	}
	ctx, cancel := context.WithTimeout(ctx, SearchBudget)
	defer cancel()
	start := time.Now()
	resp, err := p.Search(ctx, req)
	res.ElapsedMS = time.Since(start).Milliseconds()
	if err != nil {
		switch {
		case status.Code(err) == codes.Unimplemented:
			res.Status = SearchStatusUnsupported
		case errors.Is(ctx.Err(), context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded:
			res.Status = SearchStatusTimeout
		default:
			res.Status = SearchStatusError
			res.Error = err.Error()
		}
		return res
	}
	res.Status = SearchStatusOK
	res.Truncated = resp.GetTruncated()
	res.Note = resp.GetNote()
	for _, hit := range resp.GetHits() {
		if hit.GetItem() == nil || hit.GetItem().GetSourceId() == "" {
			continue // shape validation: a hit without an item is dropped, not rendered
		}
		res.Hits = append(res.Hits, SearchHit{
			Item:      item.FromProto(p.name, p.sourceType, hit.GetItem()),
			Snippet:   hit.GetSnippet(),
			MatchedIn: matchedInString(hit.GetMatchedIn()),
		})
	}
	return res
}

func matchedInString(m toposv1.MatchedIn) string {
	switch m {
	case toposv1.MatchedIn_MATCHED_IN_TITLE:
		return "title"
	case toposv1.MatchedIn_MATCHED_IN_BODY:
		return "body"
	case toposv1.MatchedIn_MATCHED_IN_LABELS:
		return "labels"
	case toposv1.MatchedIn_MATCHED_IN_ATTACHMENT:
		return "attachment"
	}
	return ""
}
