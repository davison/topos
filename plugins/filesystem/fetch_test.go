package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// --- Fetch (D-04, 12-02-PLAN.md Task 3): per-preview-kind dispatch ---
// Written before fetch.go, against a temp corpus directory.

func writeFixture(t *testing.T, root, name string, body []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), body, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
}

func fetchFull(t *testing.T, p *SourcePlugin, sourceID string) *toposv1.FetchResponse {
	t.Helper()
	resp, err := p.Fetch(t.Context(), &toposv1.FetchRequest{
		SourceId: sourceID,
		Variant:  toposv1.ContentVariant_CONTENT_VARIANT_FULL,
	})
	if err != nil {
		t.Fatalf("Fetch(%q): %v", sourceID, err)
	}
	return resp
}

func TestFetch_PDFFetchesAvailableWithBytesAndMime(t *testing.T) {
	root := t.TempDir()
	body := []byte("%PDF-1.4 fixture body")
	writeFixture(t, root, "invoice.pdf", body)
	p := NewSourcePlugin(root, nil)

	resp := fetchFull(t, p, "invoice.pdf")
	if !resp.GetAvailable() {
		t.Fatal("expected available true")
	}
	if resp.GetMimeType() != "application/pdf" {
		t.Errorf("expected mime application/pdf, got %q", resp.GetMimeType())
	}
	if resp.GetSizeBytes() != int64(len(body)) {
		t.Errorf("expected size %d, got %d", len(body), resp.GetSizeBytes())
	}
	if string(resp.GetData()) != string(body) {
		t.Errorf("expected data %q, got %q", body, resp.GetData())
	}
}

func TestFetch_PNGFetchesAvailableWithImageMime(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "photo.png", []byte("fake png bytes"))
	p := NewSourcePlugin(root, nil)

	resp := fetchFull(t, p, "photo.png")
	if !resp.GetAvailable() {
		t.Fatal("expected available true")
	}
	if resp.GetMimeType() != "image/png" {
		t.Errorf("expected mime image/png, got %q", resp.GetMimeType())
	}
}

func TestFetch_MarkdownFetchesAsRenderedHTMLWithMarkdownShape(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "notes.md", []byte("# Title\n\nbody\n"))
	p := NewSourcePlugin(root, nil)

	resp := fetchFull(t, p, "notes.md")
	if !resp.GetAvailable() {
		t.Fatal("expected available true")
	}
	if resp.GetMimeType() != "text/html" {
		t.Errorf("expected mime text/html, got %q", resp.GetMimeType())
	}
	if resp.GetContentShape() != toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML {
		t.Errorf("expected CONTENT_SHAPE_MARKDOWN_HTML, got %v", resp.GetContentShape())
	}
	if !strings.Contains(string(resp.GetData()), "<h1") {
		t.Errorf("expected rendered HTML bytes, got %q", resp.GetData())
	}
}

func TestFetch_PlainTextFetchesWithTextPopulated(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "notes.txt", []byte("hello plain text"))
	p := NewSourcePlugin(root, nil)

	resp := fetchFull(t, p, "notes.txt")
	if !resp.GetAvailable() {
		t.Fatal("expected available true")
	}
	if resp.GetMimeType() != "text/plain" {
		t.Errorf("expected mime text/plain, got %q", resp.GetMimeType())
	}
	if resp.GetText() != "hello plain text" {
		t.Errorf("expected text %q, got %q", "hello plain text", resp.GetText())
	}
}

func TestFetch_PlainTextLongerThanBoundIsHonestlyTruncated(t *testing.T) {
	root := t.TempDir()
	body := strings.Repeat("a", maxPlainTextSize+100)
	writeFixture(t, root, "big.txt", []byte(body))
	p := NewSourcePlugin(root, nil)

	resp := fetchFull(t, p, "big.txt")
	if !resp.GetAvailable() {
		t.Fatal("expected available true")
	}
	if !strings.HasSuffix(resp.GetText(), plainTextTruncationNotice) {
		t.Fatalf("expected the truncation notice as the text's final content, got suffix %q",
			resp.GetText()[max(0, len(resp.GetText())-80):])
	}
}

func TestFetch_DocxFetchesUnavailableWithNoMimeOrBytes(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "report.docx", []byte("not really an office doc"))
	p := NewSourcePlugin(root, nil)

	resp := fetchFull(t, p, "report.docx")
	if resp.GetAvailable() {
		t.Fatal("expected available false for a .docx file")
	}
	if resp.GetUnavailableReason() == "" {
		t.Error("expected a named unavailable reason")
	}
	if resp.GetMimeType() != "" {
		t.Errorf("expected no mime type, got %q", resp.GetMimeType())
	}
	if len(resp.GetData()) != 0 {
		t.Errorf("expected no bytes, got %d", len(resp.GetData()))
	}
}

func TestFetch_SVGFetchesUnavailableWithNamedReason(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "diagram.svg", []byte("<svg></svg>"))
	p := NewSourcePlugin(root, nil)

	resp := fetchFull(t, p, "diagram.svg")
	if resp.GetAvailable() {
		t.Fatal("expected available false for a .svg file")
	}
	if resp.GetUnavailableReason() == "" {
		t.Error("expected a named unavailable reason")
	}
}

func TestFetch_ThumbnailAlwaysUnavailableForEveryKind(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "invoice.pdf", []byte("%PDF-1.4"))
	writeFixture(t, root, "notes.md", []byte("# hi"))
	writeFixture(t, root, "notes.txt", []byte("hi"))
	writeFixture(t, root, "report.docx", []byte("doc"))
	p := NewSourcePlugin(root, nil)

	for _, sourceID := range []string{"invoice.pdf", "notes.md", "notes.txt", "report.docx"} {
		resp, err := p.Fetch(t.Context(), &toposv1.FetchRequest{
			SourceId: sourceID,
			Variant:  toposv1.ContentVariant_CONTENT_VARIANT_THUMBNAIL,
		})
		if err != nil {
			t.Fatalf("Fetch THUMBNAIL(%q): %v", sourceID, err)
		}
		if resp.GetAvailable() {
			t.Errorf("%s: expected THUMBNAIL to be unavailable", sourceID)
		}
	}
}

func TestFetch_MissingFileIsNotFoundGRPCError(t *testing.T) {
	root := t.TempDir()
	p := NewSourcePlugin(root, nil)

	_, err := p.Fetch(t.Context(), &toposv1.FetchRequest{
		SourceId: "does-not-exist.pdf",
		Variant:  toposv1.ContentVariant_CONTENT_VARIANT_FULL,
	})
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Fatalf("expected codes.NotFound, got %v", err)
	}
}

func TestFetch_OversizeFileIsUnavailableWithSizeReasonAndBytesNeverRead(t *testing.T) {
	root := t.TempDir()
	// A sparse file reporting a size over the cap without actually
	// allocating maxByteRenditionSize+1 bytes on disk — proves the cap
	// check happens before any read, not after a full read.
	f, err := os.Create(filepath.Join(root, "huge.pdf"))
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	if err := f.Truncate(maxByteRenditionSize + 1); err != nil {
		t.Fatalf("truncate fixture: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close fixture: %v", err)
	}
	p := NewSourcePlugin(root, nil)

	resp := fetchFull(t, p, "huge.pdf")
	if resp.GetAvailable() {
		t.Fatal("expected available false for an oversize file")
	}
	if resp.GetUnavailableReason() == "" {
		t.Error("expected a named reason citing the size limit")
	}
	if len(resp.GetData()) != 0 {
		t.Error("expected no bytes to have been read for an oversize file")
	}
}

func TestFetch_SourceIDEscapingTheRootIsRefusedBeforeAnyFileIsOpened(t *testing.T) {
	root := t.TempDir()
	p := NewSourcePlugin(root, nil)

	_, err := p.Fetch(t.Context(), &toposv1.FetchRequest{
		SourceId: "../../etc/passwd",
		Variant:  toposv1.ContentVariant_CONTENT_VARIANT_FULL,
	})
	if err == nil {
		t.Fatal("expected an error for a source_id escaping the configured root")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("expected codes.InvalidArgument, got %v", err)
	}
}
