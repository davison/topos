package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)

const (
	sourceType      = "paperless"
	displayName     = "paperless-ngx"
	contractVersion = "webspaces.v1"
	previewRuneCap  = 500
)

// SourcePlugin implements sdk.SourcePlugin against a paperless-ngx
// instance via Client.
type SourcePlugin struct {
	client  *Client
	baseURL string
}

// NewSourcePlugin builds a SourcePlugin. baseURL and token must be
// non-empty — callers (main.go) fail startup loudly if either is empty
// after config expansion.
func NewSourcePlugin(baseURL, token, apiVersion string) *SourcePlugin {
	return &SourcePlugin{
		client:  NewClient(baseURL, token, apiVersion),
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

func (p *SourcePlugin) Match(ctx context.Context, req *webspacesv1.MatchRequest) (*webspacesv1.MatchResponse, error) {
	tagIDs, err := p.client.ResolveTagIDs(ctx, req.GetKeywords())
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "paperless: resolve tag ids: %v", err)
	}
	if len(tagIDs) == 0 {
		return &webspacesv1.MatchResponse{}, nil
	}

	docs, err := p.client.ListDocuments(ctx, tagIDs)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "paperless: list documents: %v", err)
	}

	allTags, err := p.client.AllTags(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "paperless: list tags: %v", err)
	}

	items := make([]*webspacesv1.Item, 0, len(docs))
	for _, d := range docs {
		items = append(items, p.toItem(d, allTags))
	}

	return &webspacesv1.MatchResponse{Items: items}, nil
}

func (p *SourcePlugin) toItem(d Document, allTags map[int]Tag) *webspacesv1.Item {
	labels := make([]string, 0, len(d.TagIDs))
	for _, id := range d.TagIDs {
		if t, ok := allTags[id]; ok {
			labels = append(labels, t.Name)
		}
	}

	sourceID := strconv.Itoa(d.ID)

	return &webspacesv1.Item{
		SourceId:               sourceID,
		SourceType:             sourceType,
		Title:                  d.Title,
		Preview:                truncatePreview(d.Content),
		TimestampUnix:          d.Created.Unix(),
		SecondaryTimestampUnix: d.Added.Unix(),
		Fidelity:               webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT,
		DeepLink:               fmt.Sprintf("%s/documents/%s", p.baseURL, sourceID),
		Labels:                 labels,
		Provenance: map[string]string{
			"source_type":      sourceType,
			"source_system":    p.baseURL,
			"source_id":        sourceID,
			"plugin":           "webspaces-plugin-paperless",
			"contract_version": contractVersion,
		},
		HasThumbnail: true,
	}
}

// truncatePreview collapses whitespace runs and truncates to
// previewRuneCap runes on a rune boundary — the preview is a bounded
// snippet, never the full document content (KERN-03).
func truncatePreview(content string) string {
	collapsed := strings.Join(strings.FieldsFunc(content, unicode.IsSpace), " ")
	runes := []rune(collapsed)
	if len(runes) <= previewRuneCap {
		return collapsed
	}
	return string(runes[:previewRuneCap])
}

// Fetch is defined by the contract but not implemented in this plan — a
// later plan implements live content fetch on item-open. This is a
// functionality gap on a boundary that does not move, not an
// architectural stub.
func (p *SourcePlugin) Fetch(_ context.Context, _ *webspacesv1.FetchRequest) (*webspacesv1.FetchResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Fetch is not implemented in this plan (01-01); see plan 01-02")
}

func (p *SourcePlugin) Health(ctx context.Context, _ *webspacesv1.HealthRequest) (*webspacesv1.HealthResponse, error) {
	_, err := p.client.AllTags(ctx)
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
