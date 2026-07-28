package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)

const (
	sourceType      = "silverbullet"
	displayName     = "SilverBullet"
	contractVersion = "webspaces.v1"
	previewRuneCap  = 500
	// matchConcurrency bounds how many page bodies Match reads at once
	// (T-02-05): SilverBullet has no server-side tag filter (RESEARCH.md
	// Pitfall 2), so every sync reads every candidate page's body — this
	// cap, together with the shared transport's MaxConnsPerHost, keeps a
	// large space from opening hundreds of simultaneous connections
	// against the user's own home server.
	matchConcurrency = 4
)

// SourcePlugin implements sdk.SourcePlugin against a SilverBullet instance
// via Client.
type SourcePlugin struct {
	client  *Client
	baseURL string
}

// NewSourcePlugin builds a SourcePlugin. baseURL and token must be
// non-empty — callers (main.go) fail startup loudly if either is empty
// after config expansion. caCertPath is optional (see NewClient's doc
// comment for why this plugin needs it, beyond the plan's originally
// sketched two-argument constructor).
func NewSourcePlugin(baseURL, token, caCertPath string) *SourcePlugin {
	return &SourcePlugin{
		client:  NewClient(baseURL, token, caCertPath),
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (p *SourcePlugin) Describe(_ context.Context, _ *webspacesv1.DescribeRequest) (*webspacesv1.DescribeResponse, error) {
	return &webspacesv1.DescribeResponse{
		SourceType:      sourceType,
		DisplayName:     displayName,
		ContractVersion: contractVersion,
	}, nil
}

// pageMatch holds one page's Match-time state: its listing metadata, the
// tags resolved by ExtractTagsAndBody, and its frontmatter-stripped body
// (kept only long enough to build the item's Snippet — never persisted
// beyond that).
type pageMatch struct {
	file FileMeta
	tags []string
	body []byte
}

// Match lists the space once, filters to markdown pages (isPage), then
// reads each candidate page's body through a bounded worker pool
// (matchConcurrency) to extract its tags and test D-03's keyword match —
// SilverBullet has no server-side tag/name filter (RESEARCH.md Pitfall 2),
// so this cost is unavoidable and scales with total space size, not
// matched-item count (accepted for this phase's MVP scope, A-SRC-05).
func (p *SourcePlugin) Match(ctx context.Context, req *webspacesv1.MatchRequest) (*webspacesv1.MatchResponse, error) {
	keywords := req.GetKeywords()
	if len(keywords) == 0 {
		return &webspacesv1.MatchResponse{}, nil
	}

	files, err := p.client.ListFiles(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "silverbullet: list files: %v", err)
	}

	var candidates []FileMeta
	for _, f := range files {
		if isPage(f) {
			candidates = append(candidates, f)
		}
	}

	matches := make([]*pageMatch, len(candidates))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(matchConcurrency)
	for i, f := range candidates {
		i, f := i, f
		g.Go(func() error {
			raw, err := p.client.ReadFile(gctx, f.Name)
			if err != nil {
				// A single unreadable page (e.g. deleted between listing
				// and read, or a transient error) must not fail the whole
				// sync — it simply never matches, same as a page with no
				// matching tag/name would.
				return nil
			}
			body, tags := ExtractTagsAndBody(raw)
			pagePath := strings.TrimSuffix(f.Name, ".md")
			for _, kw := range keywords {
				if MatchesKeyword(pagePath, tags, kw) {
					matches[i] = &pageMatch{file: f, tags: tags, body: body}
					return nil
				}
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, status.Errorf(codes.Unavailable, "silverbullet: match: %v", err)
	}

	items := make([]*webspacesv1.Item, 0, len(candidates))
	for _, m := range matches {
		if m == nil {
			continue
		}
		items = append(items, p.toItem(m.file, m.tags, m.body))
	}
	return &webspacesv1.MatchResponse{Items: items}, nil
}

// toItem builds a webspacesv1.Item from one matched page (D-01, D-04):
// source_id is the page path with ".md" stripped, title is that path's
// final segment, preview is the frontmatter-stripped body's snippet, and
// the deep link is an exact-fidelity link to {base_url}/{page-path}.
func (p *SourcePlugin) toItem(f FileMeta, tags []string, body []byte) *webspacesv1.Item {
	sourceID := strings.TrimSuffix(f.Name, ".md")
	title := sourceID
	if idx := strings.LastIndex(sourceID, "/"); idx >= 0 {
		title = sourceID[idx+1:]
	}

	return &webspacesv1.Item{
		SourceId:               sourceID,
		SourceType:             sourceType,
		Title:                  title,
		Preview:                Snippet(body),
		TimestampUnix:          f.LastModified / 1000,
		SecondaryTimestampUnix: f.Created / 1000,
		Fidelity:               webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT,
		DeepLink:               fmt.Sprintf("%s/%s", p.baseURL, sourceID),
		Labels:                 tags,
		Provenance: map[string]string{
			"source_type":      sourceType,
			"source_system":    p.baseURL,
			"source_id":        sourceID,
			"plugin":           "webspaces-plugin-silverbullet",
			"contract_version": contractVersion,
		},
		HasThumbnail: false,
	}
}

// Fetch is stubbed in Task 1 — Task 2 fills it in with the real
// goldmark+bluemonday rendered-markdown variant switch. This is the one
// place a later task fills without an architectural change (per the
// plan's own note).
func (p *SourcePlugin) Fetch(_ context.Context, _ *webspacesv1.FetchRequest) (*webspacesv1.FetchResponse, error) {
	return nil, status.Error(codes.Unimplemented, "silverbullet: fetch not yet implemented")
}

func (p *SourcePlugin) Health(ctx context.Context, _ *webspacesv1.HealthRequest) (*webspacesv1.HealthResponse, error) {
	_, err := p.client.ListFiles(ctx)
	if err != nil {
		return &webspacesv1.HealthResponse{
			Reachable: false,
			LastError: err.Error(),
		}, nil
	}
	return &webspacesv1.HealthResponse{
		Reachable:    true,
		LastSyncUnix: time.Now().Unix(),
	}, nil
}
