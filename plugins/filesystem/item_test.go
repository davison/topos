package main

import (
	"os"
	"path/filepath"
	"testing"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// --- relPathSourceID (D-01): bare filename at the root, forward-slash relative path in a subdirectory ---

func TestRelPathSourceID_TopLevelFileYieldsBareFilename(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "invoice.pdf")

	got := relPathSourceID(root, full)
	if got != "invoice.pdf" {
		t.Fatalf("expected bare filename %q, got %q", "invoice.pdf", got)
	}
}

func TestRelPathSourceID_SubdirectoryFileYieldsForwardSlashRelativePath(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "receipts", "2026", "invoice.pdf")

	got := relPathSourceID(root, full)
	want := "receipts/2026/invoice.pdf"
	if got != want {
		t.Fatalf("expected forward-slash relative path %q, got %q", want, got)
	}
	if filepath.IsAbs(got) {
		t.Fatalf("expected a relative source_id, got an absolute one: %q", got)
	}
	if len(got) >= 2 && got[:2] == "./" {
		t.Fatalf("expected no leading './', got %q", got)
	}
}

// --- folderLabels (D-05): the root's own base name for a top-level file ---

func TestFolderLabels_TopLevelFileIsRootBaseName(t *testing.T) {
	root := filepath.Join(t.TempDir(), "household-docs")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	full := filepath.Join(root, "invoice.pdf")

	got := folderLabels(root, full)
	want := []string{"household-docs"}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("expected labels %v, got %v", want, got)
	}
}

func TestFolderLabels_SubdirectoryFileIsContainingDirectoryBaseName(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "receipts", "invoice.pdf")

	got := folderLabels(root, full)
	want := []string{"receipts"}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("expected labels %v, got %v", want, got)
	}
}

// --- fileDeepLink: file:// URI over the root+relative-path join ---

func TestFileDeepLink_BuildsFileSchemeURIOverRootJoinedWithSourceID(t *testing.T) {
	root := "/mnt/nas/household-docs"
	sourceID := "receipts/invoice.pdf"

	got := fileDeepLink(root, sourceID)
	want := "file:///mnt/nas/household-docs/receipts/invoice.pdf"
	if got != want {
		t.Fatalf("expected deep link %q, got %q", want, got)
	}
}

// --- resolvePath: the same root-prefix guard the kernel's open route re-checks ---

func TestResolvePath_JoinsRootAndSourceID(t *testing.T) {
	root := t.TempDir()
	got, err := resolvePath(root, "invoice.pdf")
	if err != nil {
		t.Fatalf("resolvePath: %v", err)
	}
	want := filepath.Join(root, "invoice.pdf")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolvePath_RefusesEscapeViaDotDotSegments(t *testing.T) {
	root := t.TempDir()
	if _, err := resolvePath(root, "../../etc/passwd"); err == nil {
		t.Fatal("expected resolvePath to refuse a path escaping the root, got nil error")
	}
}

// --- toItem (via Match): fidelity EXACT, empty preview, no file body read ---

func TestMatch_TopLevelPDFYieldsExactFidelityAndEmptyPreview(t *testing.T) {
	root := t.TempDir()
	pdfBody := []byte("%PDF-1.4 fixture content that must never be read at Match time")
	if err := os.WriteFile(filepath.Join(root, "invoice.pdf"), pdfBody, 0o644); err != nil {
		t.Fatalf("write fixture pdf: %v", err)
	}

	p := NewSourcePlugin(root)
	resp, err := p.Match(t.Context(), &toposv1.MatchRequest{})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(resp.GetItems()) != 1 {
		t.Fatalf("expected exactly 1 item, got %d", len(resp.GetItems()))
	}

	it := resp.GetItems()[0]
	if it.GetFidelity() != toposv1.LinkFidelity_LINK_FIDELITY_EXACT {
		t.Errorf("expected LINK_FIDELITY_EXACT, got %v", it.GetFidelity())
	}
	if it.GetPreview() != "" {
		t.Errorf("expected empty preview at Match time, got %q", it.GetPreview())
	}
	if it.GetSourceId() != "invoice.pdf" {
		t.Errorf("expected source_id %q, got %q", "invoice.pdf", it.GetSourceId())
	}
	wantDeepLink := "file://" + filepath.ToSlash(filepath.Join(root, "invoice.pdf"))
	if it.GetDeepLink() != wantDeepLink {
		t.Errorf("expected deep_link %q, got %q", wantDeepLink, it.GetDeepLink())
	}
}

func TestMatch_NonPDFFilesAreIgnored(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("plain text"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	p := NewSourcePlugin(root)
	resp, err := p.Match(t.Context(), &toposv1.MatchRequest{})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(resp.GetItems()) != 0 {
		t.Fatalf("expected 0 items for a non-PDF file, got %d", len(resp.GetItems()))
	}
}
