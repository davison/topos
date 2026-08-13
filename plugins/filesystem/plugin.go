package main

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

const (
	sourceType      = "filesystem"
	displayName     = "Filesystem folder"
	contractVersion = "topos.v2"

	// iconMIME is the declared mime for iconSVG below, returned verbatim
	// from Describe (internal/audit's plugin-icon contract, mirroring every
	// other in-repo plugin).
	iconMIME = "image/svg+xml"
)

// matchVocabulary is the field-name vocabulary this plugin declares in its
// Describe response and reads from MatchRequest.match_fields — folder paths,
// mirroring the Proton plugin's own "folders" vocabulary (D-05,
// 12-CONTEXT.md).
var matchVocabulary = []string{"folders"}

// iconSVG is a Lucide folder-family glyph, stroke baked to the literal
// --muted-foreground hex (never "currentColor" — an img-loaded SVG cannot
// inherit page CSS; internal/audit/plugin_icons_test.go enforces this
// mechanically).
//
// Source-Project: @lucide/svelte (lucide-icons/lucide)
// Source-File:    dist/icons/folder.svelte
// Source-Version: @lucide/svelte v1.27.0
// Source-License: ISC
//
//go:embed assets/icon.svg
var iconSVG []byte

// SourcePlugin implements sdk.SourcePlugin over a configured local/network
// filesystem folder: Match resolves each top-level file's scope and
// preview-kind classification through scope.go/classify.go (D-03, D-04)
// instead of the 12-01 tracer's inline ".pdf" test. Subfolder recursion
// remains a later plan's work — this plan widens document scope and
// preview shapes only, still walking the configured root's top level.
type SourcePlugin struct {
	root   string
	extras map[string]string
}

// NewSourcePlugin builds a SourcePlugin rooted at root — already expanded
// (main.go's expandHome) and otherwise unvalidated; a root that does not
// exist or is not readable surfaces honestly through Health/Match, never
// silently. extras carries this instance's own config.Source.Extras
// verbatim (D-12/D-13) — may be nil, a legitimate "no scope overrides
// configured" state that newScope resolves to the default allowlist
// alone.
func NewSourcePlugin(root string, extras map[string]string) *SourcePlugin {
	return &SourcePlugin{root: root, extras: extras}
}

// includeGlobKey and excludeGlobKey are the two extras keys this plugin
// declares in Describe (D-03) and reads in Match via newScope — the exact
// strings scope.go's newScope indexes into the extras map with.
const (
	includeGlobKey = "include_glob"
	excludeGlobKey = "exclude_glob"
)

func (p *SourcePlugin) Describe(_ context.Context, _ *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error) {
	return &toposv1.DescribeResponse{
		SourceType:      sourceType,
		DisplayName:     displayName,
		ContractVersion: contractVersion,
		MatchVocabulary: matchVocabulary,
		Icon:            iconSVG,
		IconMime:        iconMIME,
		// Extras (D-15, PLUG-09): declaring these two keys here is the only
		// place they need to exist — Phase 11's declared-fields editor
		// renders them generically from this response, no new UI code.
		Extras: []*toposv1.ExtrasField{
			{
				Key:         includeGlobKey,
				Label:       "Include glob (comma-separated)",
				Required:    false,
				Secret:      false,
				Placeholder: "**/*.pdf,**/*.md",
			},
			{
				Key:         excludeGlobKey,
				Label:       "Exclude glob (comma-separated)",
				Required:    false,
				Secret:      false,
				Placeholder: "**/node_modules/**",
			},
		},
	}, nil
}

// Match reads the configured root's TOP LEVEL ONLY (os.ReadDir, never
// filepath.WalkDir — recursion is a later plan's work). Each candidate
// file's D-01 relative source_id is resolved through scope.go's
// (*scope).includes, which decides both inclusion and preview-kind
// classification from this instance's extras-driven include/exclude globs
// plus the default extension allowlist (D-03) — compiled once per Match
// call via newScope, not once per file. No file body is ever read here —
// preview stays empty at Match time (D-04's "no new kernel/UI rendering
// work" applies transitively: Match reads only file metadata to build an
// item; Fetch re-derives the same classification fresh, never caching it
// from here).
//
// Match reads only its own declared "folders" field from match_fields and
// ignores every other key present in the request map (D-05): when the
// field is present, only items whose folder label appears in it are kept
// (case-insensitive exact comparison, never substring/prefix); when
// absent, every item is returned so the kernel's keywords fallback can do
// the matching.
func (p *SourcePlugin) Match(_ context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	entries, err := os.ReadDir(p.root)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "filesystem: read root: %v", err)
	}

	sc := newScope(p.extras)
	folders, hasFolders := req.GetMatchFields()["folders"]

	var items []*toposv1.Item
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		full := filepath.Join(p.root, entry.Name())
		sourceID := relPathSourceID(p.root, full)

		if _, included, err := sc.includes(sourceID); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "filesystem: %v", err)
		} else if !included {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			// A file that vanished or became unreadable between ReadDir and
			// Info is skipped for this sync — the next poll picks it back up
			// if it reappears, matching the stat-diff-via-full-replace design
			// (12-RESEARCH.md).
			continue
		}

		it := p.toItem(entry.Name(), info)
		if hasFolders && !labelMatchesAny(it.GetLabels(), folders.GetValues()) {
			continue
		}
		items = append(items, it)
	}

	return &toposv1.MatchResponse{Items: items}, nil
}

// labelMatchesAny reports whether any of labels exactly, case-
// insensitively equals any of values — no substring/prefix matching,
// mirroring every other plugin's own match-field comparison discipline
// (D-04, docs/plugin-contract.md).
func labelMatchesAny(labels, values []string) bool {
	for _, l := range labels {
		for _, v := range values {
			if strings.EqualFold(l, v) {
				return true
			}
		}
	}
	return false
}

// toItem builds the Item for the top-level file name (info already stat'd
// by Match). source_id is the D-01 relative path; labels is the D-05
// folder-vocabulary value; deep_link is the file:// URI the kernel rewrites
// at serve time (Task 1 checkpoint, option-a); fidelity is always EXACT for
// this tracer's PDF-only scope; preview stays empty.
func (p *SourcePlugin) toItem(name string, info os.FileInfo) *toposv1.Item {
	full := filepath.Join(p.root, name)
	sourceID := relPathSourceID(p.root, full)
	modUnix := info.ModTime().Unix()

	return &toposv1.Item{
		SourceId:               sourceID,
		SourceType:             sourceType,
		Title:                  name,
		TimestampUnix:          modUnix,
		SecondaryTimestampUnix: modUnix,
		Fidelity:               toposv1.LinkFidelity_LINK_FIDELITY_EXACT,
		DeepLink:               fileDeepLink(p.root, sourceID),
		Labels:                 folderLabels(p.root, full),
		Provenance: map[string]string{
			"source_type":      sourceType,
			"source_id":        sourceID,
			"plugin":           "topos-plugin-filesystem",
			"contract_version": contractVersion,
		},
	}
}

// noRenditionReason is the fixed unavailable_reason used for a content
// variant this tracer's PDF-only scope never has a rendition for
// (THUMBNAIL) — a normal outcome, not an error, mirroring the paperless
// plugin's identical convention.
const noRenditionReason = "no thumbnail rendition"

// Fetch dispatches on ContentVariant exactly like plugins/paperless/
// plugin.go does. FULL and PREVIEW both re-read the file fresh from disk
// (never cached from Match, matching every other plugin's "Fetch re-fetches
// fresh from source" rule) and return the PDF's raw bytes; THUMBNAIL
// returns unavailable with a fixed reason — this tracer never generates a
// thumbnail rendition.
func (p *SourcePlugin) Fetch(_ context.Context, req *toposv1.FetchRequest) (*toposv1.FetchResponse, error) {
	switch req.GetVariant() {
	case toposv1.ContentVariant_CONTENT_VARIANT_FULL, toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW:
		return p.fetchBytes(req.GetSourceId())
	case toposv1.ContentVariant_CONTENT_VARIANT_THUMBNAIL:
		return &toposv1.FetchResponse{Available: false, UnavailableReason: noRenditionReason}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "filesystem: unspecified content variant")
	}
}

// fetchBytes re-validates sourceID resolves inside the configured root
// (defense-in-depth, mirroring kernel/httpapi/fsopen.go's identical guard)
// before reading — a missing file is codes.NotFound, any other I/O failure
// is codes.Unavailable, matching the paperless/silverbullet error-code
// convention.
func (p *SourcePlugin) fetchBytes(sourceID string) (*toposv1.FetchResponse, error) {
	full, err := resolvePath(p.root, sourceID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "filesystem: %v", err)
	}

	data, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Errorf(codes.NotFound, "filesystem: item %q not found", sourceID)
		}
		return nil, status.Errorf(codes.Unavailable, "filesystem: read %q: %v", sourceID, err)
	}

	return &toposv1.FetchResponse{
		Available: true,
		MimeType:  "application/pdf",
		SizeBytes: int64(len(data)),
		Data:      data,
	}, nil
}

// Health stats the configured root: a readable directory is reachable;
// anything else (missing, unreadable, or a non-directory) is unreachable
// with the OS error (or a named reason) as last_error — never
// reachable-with-zero-items for an unreadable mount (12-CONTEXT.md
// Claude's Discretion; mirrors the WhatsApp/Signal "degrade honestly"
// precedent).
func (p *SourcePlugin) Health(_ context.Context, _ *toposv1.HealthRequest) (*toposv1.HealthResponse, error) {
	info, err := os.Stat(p.root)
	if err != nil {
		return &toposv1.HealthResponse{Reachable: false, LastError: err.Error()}, nil
	}
	if !info.IsDir() {
		return &toposv1.HealthResponse{Reachable: false, LastError: "filesystem: configured path is not a directory"}, nil
	}
	return &toposv1.HealthResponse{Reachable: true, LastSyncUnix: time.Now().Unix()}, nil
}
