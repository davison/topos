package httpapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/pluginhost"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// Fetcher is the minimal request-time plugin-call surface item.go depends
// on. *pluginhost.Host satisfies this structurally. Kept as an interface
// (rather than a concrete *pluginhost.Host parameter) so item_test.go can
// exercise every response branch — 404, 502, security headers, 415, id
// encoding — without launching a real plugin subprocess. This is the
// deliberate exception to the "httpapi never reaches a plugin" rule:
// stream.go must never import pluginhost (KERN-02); item.go is exactly
// the request-time, item-open boundary where a live plugin call belongs.
// source is the instance id (item.Item.Source, D-08) — pluginhost.Host's
// own Fetch resolves the launched plugin by instance id, not by plugin
// kind, so every call site below must pass it.Source, never it.SourceType.
type Fetcher interface {
	Fetch(ctx context.Context, source, sourceID string, variant toposv1.ContentVariant) (pluginhost.FetchResult, error)
}

// allowedRenditionTypes is the MIME allowlist enforced on every byte
// served by ItemContentHandler/ItemThumbnailHandler (T-01-10). A
// plugin-supplied MIME type is never echoed into a response header
// without first being checked against this set.
var allowedRenditionTypes = map[string]bool{
	"application/pdf": true,
	"image/png":       true,
	"image/jpeg":      true,
	"image/gif":       true,
	"image/webp":      true,
	// text/html: added for the SilverBullet plugin's rendered-markdown
	// rendition (D-04). Safe to serve from the kernel's own origin under
	// the same hardened header set every rendition already gets below
	// (Content-Security-Policy: ...; sandbox, X-Content-Type-Options:
	// nosniff) — the producing plugin sanitizes with bluemonday before
	// this byte ever reaches the kernel, and the sandboxed iframe boundary
	// is a second, independent layer on top of that (T-02-01).
	"text/html": true,
	// text/plain: added for Phase 12's filesystem source plugin, whose
	// D-04 plain-text preview shape (12-RESEARCH.md Pitfall 1, T-12-07)
	// serves a document's raw text bytes with no sanitize/wrap step —
	// unlike text/html, there is no markup to strip, so this entry is
	// safe by construction: the same hardened header set below (nosniff,
	// sandboxed CSP) still applies, and the browser renders it as inert
	// text under that CSP regardless of the file's own content.
	"text/plain": true,
}

type rendition struct {
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	URL       string `json:"url"`
}

type itemContent struct {
	Available         bool       `json:"available"`
	UnavailableReason string     `json:"unavailable_reason"`
	Text              string     `json:"text"`
	Rendition         *rendition `json:"rendition"`
}

type itemDetailResponse struct {
	SchemaVersion int         `json:"schema_version"`
	Item          streamItem  `json:"item"`
	Content       itemContent `json:"content"`
}

// itemIDParam resolves the {id} path parameter to its decoded form. chi
// routes against r.URL.RawPath when the client sent one (i.e. the request
// path contained percent-escapes), so chi.URLParam returns the raw,
// still-escaped segment for a request like
// "/api/items/paperless%3A42" — url.PathUnescape is required so
// "paperless:42" and "paperless%3A42" resolve to the same item id.
func itemIDParam(r *http.Request) string {
	raw := chi.URLParam(r, "id")
	if decoded, err := url.PathUnescape(raw); err == nil {
		return decoded
	}
	return raw
}

// ItemHandler serves GET /api/items/{id}: an index read to resolve the
// composite id to source (instance id) /source_id and the item's own
// metadata, plus exactly one request-time plugin Fetch call (full variant)
// for the live extracted text and rendition descriptor (KERN-03). cfgStore
// is resolved fresh as the first statement of the returned closure
// (07-02-PLAN.md Task 2) to label source_display_name (D-09) — inert
// configuration data, never a plugin call — so a display-name edit saved
// through PUT /api/config is visible on the very next item request with no
// kernel restart.
func ItemHandler(store *index.Store, cfgStore *config.Store, fetcher Fetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgStore.Expanded()
		id := itemIDParam(r)
		ctx := r.Context()

		it, ok, err := store.GetItem(ctx, id)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if !ok {
			WriteError(w, http.StatusNotFound, "item_not_found", "item \""+id+"\" was not found in the index")
			return
		}

		result, err := fetcher.Fetch(ctx, it.Source, it.SourceID, toposv1.ContentVariant_CONTENT_VARIANT_FULL)
		if err != nil {
			writeFetchError(w, id, err)
			return
		}

		content := itemContent{
			Available:         result.Available,
			UnavailableReason: result.UnavailableReason,
			Text:              result.Text,
		}
		if result.Available && result.MimeType != "" {
			content.Rendition = &rendition{
				MimeType:  result.MimeType,
				SizeBytes: result.SizeBytes,
				URL:       "/api/items/" + id + "/content",
			}
		}

		WriteJSON(w, http.StatusOK, itemDetailResponse{
			SchemaVersion: schemaVersion,
			Item:          toStreamItemFor(it, cfg.DisplayNameFor),
			Content:       content,
		})
	}
}

// ItemContentHandler serves GET /api/items/{id}/content — the preview
// rendition's raw bytes, streamed straight through with io.Copy.
func ItemContentHandler(store *index.Store, fetcher Fetcher) http.HandlerFunc {
	return renditionHandler(store, fetcher, toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW)
}

// ItemThumbnailHandler serves GET /api/items/{id}/thumbnail — the
// thumbnail rendition's raw bytes, streamed straight through with
// io.Copy.
func ItemThumbnailHandler(store *index.Store, fetcher Fetcher) http.HandlerFunc {
	return renditionHandler(store, fetcher, toposv1.ContentVariant_CONTENT_VARIANT_THUMBNAIL)
}

// renditionHandler is shared by ItemContentHandler and
// ItemThumbnailHandler. This is the sharpest security surface in the
// phase (T-01-10): it serves source-controlled bytes from the kernel's
// own origin, so every accepted MIME type is checked against an
// allowlist and every response carries a hardened header set before any
// body is written.
func renditionHandler(store *index.Store, fetcher Fetcher, variant toposv1.ContentVariant) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := itemIDParam(r)
		ctx := r.Context()

		it, ok, err := store.GetItem(ctx, id)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if !ok {
			WriteError(w, http.StatusNotFound, "item_not_found", "item \""+id+"\" was not found in the index")
			return
		}

		result, err := fetcher.Fetch(ctx, it.Source, it.SourceID, variant)
		if err != nil {
			writeFetchError(w, id, err)
			return
		}

		if !result.Available || result.Body == nil {
			WriteError(w, http.StatusNotFound, "content_unavailable", "no rendition is available for item \""+id+"\"")
			return
		}
		defer result.Body.Close()

		// Never echo a plugin-supplied MIME string into the response
		// header without matching it against the allowlist first.
		if !allowedRenditionTypes[result.MimeType] {
			WriteError(w, http.StatusUnsupportedMediaType, "unsupported_rendition_type",
				"rendition MIME type \""+result.MimeType+"\" is not on the allowlist")
			return
		}

		// D-11: a text/html rendition arrives from the plugin as an
		// unwrapped, unthemed, unsanitized fragment — the kernel is the
		// one place that sanitizes, wraps and themes it, via the
		// content-shape-keyed policy table in rendition.go. A shape the
		// kernel does not recognise (including CONTENT_SHAPE_UNSPECIFIED)
		// is refused outright: the kernel fails closed rather than ever
		// serving an unsanitized document from its own origin.
		//
		// UI-09: the optional ?hl= query parameter carries the user's raw
		// in-webspace search query. highlightTerms (rendition.go) derives
		// the bounded literal-term set sanitizeAndWrapRendition highlights
		// inside the sanitized fragment; an absent or empty hl parameter
		// yields a nil terms slice, which sanitizeAndWrapRendition treats
		// as "skip highlighting entirely" — the no-search path is
		// byte-identical to this route's pre-UI-09 output.
		//
		// docs/api.md documents ?hl= as content-route-only (WR-01):
		// terms are only ever derived for the PREVIEW variant
		// (ItemContentHandler), never for THUMBNAIL
		// (ItemThumbnailHandler) -- even though nothing in the MIME
		// allowlist prevents a plugin from returning a text/html
		// thumbnail rendition, that path must never highlight.
		var body []byte
		if result.MimeType == "text/html" {
			fragment, err := io.ReadAll(result.Body)
			if err != nil {
				WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			var terms []string
			if variant == toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW {
				terms = highlightTerms(r.URL.Query().Get("hl"))
			}
			wrapped, err := sanitizeAndWrapRendition(result.ContentShape, fragment, terms)
			if err != nil {
				WriteError(w, http.StatusBadGateway, "unsupported_content_shape",
					"item \""+id+"\": "+err.Error())
				return
			}
			body = wrapped
		}

		h := w.Header()
		h.Set("Content-Type", result.MimeType)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Content-Disposition", "inline")
		// style-src 'unsafe-inline' (added live via UAT — see
		// 02-01-SUMMARY.md): without it, default-src 'none' with no
		// explicit style-src blocks the browser from applying ANY inline
		// <style> element, including the kernel's own composed rendition
		// stylesheet (rendition.go), served but silently never applied.
		// Scripts remain fully blocked regardless (default-src 'none'
		// plus the sandbox directive plus the embedding <iframe>'s own
		// sandbox attribute in DetailPane.svelte, none of which this
		// change touches). style-src 'unsafe-inline' is safe here
		// specifically because the only inline style any rendition
		// document can ever carry (D-11) is the kernel's own composed
		// stylesheet, injected by sanitizeAndWrapRendition strictly AFTER
		// its bluemonday policy runs over the plugin-supplied fragment —
		// that policy strips any <style> element or style attribute that
		// originated from page content, so this directive cannot be
		// exploited by a hostile or malformed source document. This CSP
		// is shared by every rendition type (PDF, images, text/html);
		// widening style-src does not change how a PDF or image renders
		// (neither has any inline stylesheet to apply) so this is a
		// monotonic widening, not a behavior change, for those types.
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; object-src 'none'; sandbox")
		h.Set("Cache-Control", "private, no-store")
		w.WriteHeader(http.StatusOK)
		if result.MimeType == "text/html" {
			_, _ = w.Write(body)
			return
		}
		_, _ = io.Copy(w, result.Body)
	}
}

// writeFetchError maps a pluginhost.Fetch error to the shared HTTP error
// envelope: ErrItemNotFound -> 404, ErrSourceUnavailable (and anything
// else) -> 502. A source-unavailable failure must never fall through to a
// 200 with a silently empty content object.
func writeFetchError(w http.ResponseWriter, id string, err error) {
	if errors.Is(err, pluginhost.ErrItemNotFound) {
		WriteError(w, http.StatusNotFound, "item_not_found", err.Error())
		return
	}
	WriteError(w, http.StatusBadGateway, "source_unavailable", err.Error())
}
