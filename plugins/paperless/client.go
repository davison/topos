// Command webspaces-plugin-paperless: this file implements the hand-rolled
// paperless-ngx REST client half of the plugin (see plugin.go for the
// webspacesv1.SourcePlugin adapter and main.go for the subprocess
// entrypoint).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Client is a thin, read-only REST client against a paperless-ngx
// instance. Every request uses the GET method — there is no code path in
// this file that sends any other method (PLUG-02: plugins never mutate
// source data stores).
type Client struct {
	baseURL    string
	token      string
	apiVersion string
	http       *http.Client
}

// NewClient builds a Client bounded to at most 4 concurrent in-flight
// connections to paperless-ngx (SRC-04/concurrency) and a 30-second
// per-request timeout, sharing exactly one http.Client/http.Transport
// across every RPC this plugin process serves.
func NewClient(baseURL, token, apiVersion string) *Client {
	transport := &http.Transport{
		MaxConnsPerHost: 4,
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		apiVersion: apiVersion,
		http: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

// Document is a paperless-ngx document as relevant to this plugin.
type Document struct {
	ID      int
	Title   string
	Content string
	Created time.Time // date-only "created" field, parsed as midnight UTC
	Added   time.Time // full-datetime "added" field
	TagIDs  []int
}

// Tag is a paperless-ngx tag.
type Tag struct {
	ID   int
	Name string
}

type tagResult struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type tagsPage struct {
	Next    *string     `json:"next"`
	Results []tagResult `json:"results"`
}

type documentResult struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Created string `json:"created"`
	Added   string `json:"added"`
	Tags    []int  `json:"tags"`
}

type documentsPage struct {
	Next    *string          `json:"next"`
	Results []documentResult `json:"results"`
}

// ResolveTagIDs resolves each keyword to zero or more tag IDs via an exact,
// case-insensitive tag-name lookup (name__iexact — D-03). It never uses a
// substring-based tag name filter, which would make the keyword "house"
// match the tag "Household".
func (c *Client) ResolveTagIDs(ctx context.Context, keywords []string) ([]int, error) {
	var ids []int
	seen := map[int]bool{}
	for _, kw := range keywords {
		q := url.Values{}
		q.Set("name__iexact", kw)
		q.Set("page_size", "100")

		var page tagsPage
		if err := c.getJSON(ctx, "/api/tags/", q, &page); err != nil {
			return nil, fmt.Errorf("paperless: resolve tag for keyword %q: %w", kw, err)
		}
		for _, t := range page.Results {
			if !seen[t.ID] {
				seen[t.ID] = true
				ids = append(ids, t.ID)
			}
		}
	}
	return ids, nil
}

// ListDocuments fetches every document tagged with any of tagIDs
// (confirmed OR semantics via Django's tags__id__in), following pagination
// to completion. Returns an empty slice, not an error, when tagIDs is
// empty.
func (c *Client) ListDocuments(ctx context.Context, tagIDs []int) ([]Document, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}

	idStrs := make([]string, len(tagIDs))
	for i, id := range tagIDs {
		idStrs[i] = strconv.Itoa(id)
	}

	q := url.Values{}
	q.Set("tags__id__in", strings.Join(idStrs, ","))
	q.Set("page_size", "100")
	q.Set("ordering", "-created")

	var docs []Document
	path := "/api/documents/"
	values := q
	for {
		var page documentsPage
		if err := c.getJSON(ctx, path, values, &page); err != nil {
			return nil, fmt.Errorf("paperless: list documents: %w", err)
		}
		for _, d := range page.Results {
			doc, err := toDocument(d)
			if err != nil {
				return nil, fmt.Errorf("paperless: parse document %d: %w", d.ID, err)
			}
			docs = append(docs, doc)
		}
		if page.Next == nil || *page.Next == "" {
			break
		}
		nextPath, nextValues, err := splitNextURL(*page.Next)
		if err != nil {
			return nil, fmt.Errorf("paperless: parse next page URL: %w", err)
		}
		path, values = nextPath, nextValues
	}
	return docs, nil
}

// AllTags fetches every tag known to paperless-ngx, paginated to
// completion, keyed by tag ID. Used to resolve a document's own tags to
// human-readable names for Item.Labels.
func (c *Client) AllTags(ctx context.Context) (map[int]Tag, error) {
	out := map[int]Tag{}
	path := "/api/tags/"
	values := url.Values{"page_size": {"100"}}
	for {
		var page tagsPage
		if err := c.getJSON(ctx, path, values, &page); err != nil {
			return nil, fmt.Errorf("paperless: list tags: %w", err)
		}
		for _, t := range page.Results {
			out[t.ID] = Tag{ID: t.ID, Name: t.Name}
		}
		if page.Next == nil || *page.Next == "" {
			break
		}
		nextPath, nextValues, err := splitNextURL(*page.Next)
		if err != nil {
			return nil, fmt.Errorf("paperless: parse next tags page URL: %w", err)
		}
		path, values = nextPath, nextValues
	}
	return out, nil
}

func toDocument(d documentResult) (Document, error) {
	// created is a date-only field as of paperless-ngx API v9+ (never the
	// deprecated full-datetime creation field that preceded it) — parsed
	// as midnight UTC.
	created, err := time.Parse("2006-01-02", d.Created)
	if err != nil {
		// Fall back to RFC3339 in case the server still returns a full
		// datetime; take just the date portion either way.
		if t, err2 := time.Parse(time.RFC3339, d.Created); err2 == nil {
			created = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		} else {
			return Document{}, fmt.Errorf("parse created %q: %w", d.Created, err)
		}
	}

	added, err := time.Parse(time.RFC3339, d.Added)
	if err != nil {
		added = time.Unix(0, 0).UTC()
	}

	return Document{
		ID: d.ID, Title: d.Title, Content: d.Content,
		Created: created, Added: added, TagIDs: d.Tags,
	}, nil
}

// getJSON performs a single GET request against path+query and decodes the
// JSON response body into out. Every request in this file uses GET; there
// is no PUT/POST/PATCH/DELETE code path anywhere in the plugin.
func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out interface{}) error {
	full := c.baseURL + path
	if len(query) > 0 {
		full += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("Accept", "application/json; version="+c.apiVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		// Never log the request object or its headers — the Authorization
		// header carries the bearer token (T-01-02).
		return fmt.Errorf("request %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, path)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response from %s: %w", path, err)
	}
	return nil
}

var absoluteURLPrefix = regexp.MustCompile(`^https?://[^/]+`)

// splitNextURL turns a paperless-ngx pagination "next" URL (which may be
// absolute, including scheme and host) back into a path + query pair
// relative to this client's configured base URL, so getJSON can prefix it
// consistently.
func splitNextURL(next string) (string, url.Values, error) {
	trimmed := absoluteURLPrefix.ReplaceAllString(next, "")
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", nil, err
	}
	return u.Path, u.Query(), nil
}
