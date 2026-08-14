package main

import (
	"os"
	"path/filepath"
	"runtime"
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
	// Fail-closed resolution (CR-02, 12-06-PLAN.md Task 1) means resolvePath
	// now reaches filepath.EvalSymlinks, which requires a real fixture file
	// rather than a merely lexical join — fixture correction, not assertion
	// loosening.
	if err := os.WriteFile(filepath.Join(root, "invoice.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	got, _, err := resolvePath(root, "invoice.pdf")
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
	if _, _, err := resolvePath(root, "../../etc/passwd"); err == nil {
		t.Fatal("expected resolvePath to refuse a path escaping the root, got nil error")
	}
}

// TestResolvePath_ReturnsTheLexicalIdentityPathAndTheResolvedRealPath
// proves resolvePath's widened signature (12-07-PLAN.md Task 2, WR-02)
// returns two different strings naming the same file: the first is the
// unchanged LEXICAL identity path (D-01), the second is the symlink-free
// real path every read/exec should target.
func TestResolvePath_ReturnsTheLexicalIdentityPathAndTheResolvedRealPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on windows")
	}
	tmp := t.TempDir()
	real := filepath.Join(tmp, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatalf("mkdir real: %v", err)
	}
	if err := os.WriteFile(filepath.Join(real, "doc.md"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write doc.md: %v", err)
	}
	linkRoot := filepath.Join(tmp, "linkroot")
	if err := os.Symlink(real, linkRoot); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	full, resolved, err := resolvePath(linkRoot, "doc.md")
	if err != nil {
		t.Fatalf("resolvePath: %v", err)
	}
	wantFull := filepath.Join(linkRoot, "doc.md")
	if full != wantFull {
		t.Fatalf("expected the lexical join under linkroot %q, got %q", wantFull, full)
	}
	wantResolved := filepath.Join(real, "doc.md")
	if resolved != wantResolved {
		t.Fatalf("expected the resolved join under the real directory %q, got %q", wantResolved, resolved)
	}
	if full == resolved {
		t.Fatal("expected the lexical and resolved paths to differ for a symlinked root")
	}
}

// TestResolvePath_SymlinkSwapOutsideRootIsRefused proves CR-02 is closed:
// a file indexed as legitimate and then swapped on disk for a symlink
// pointing outside the configured root is refused by resolvePath even
// though the source_id string itself contains no ".." segment and the
// lexical check alone would pass cleanly.
func TestResolvePath_SymlinkSwapOutsideRootIsRefused(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("top secret"), 0o644); err != nil {
		t.Fatalf("write outside fixture: %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "notes.md")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	full, resolved, err := resolvePath(root, "notes.md")
	if err == nil {
		t.Fatal("expected resolvePath to refuse a post-index symlink swap outside the root, got nil error")
	}
	if full != "" {
		t.Errorf("expected an empty lexical path on refusal, got %q", full)
	}
	if resolved != "" {
		t.Errorf("expected an empty resolved path on refusal, got %q", resolved)
	}
}

// TestResolvePath_SymlinkedRootStillResolvesAnInRootFile proves the
// resolved-root comparison does not turn a legitimate symlinked root (the
// common dotfile-manager `~/Documents` -> `~/dotfiles/Documents` pattern,
// WR-01) into a false containment failure.
func TestResolvePath_SymlinkedRootStillResolvesAnInRootFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on windows")
	}
	tmp := t.TempDir()
	real := filepath.Join(tmp, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatalf("mkdir real: %v", err)
	}
	if err := os.WriteFile(filepath.Join(real, "doc.md"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write doc.md: %v", err)
	}
	linkRoot := filepath.Join(tmp, "linkroot")
	if err := os.Symlink(real, linkRoot); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	got, _, err := resolvePath(linkRoot, "doc.md")
	if err != nil {
		t.Fatalf("resolvePath: %v", err)
	}
	want := filepath.Join(linkRoot, "doc.md")
	if got != want {
		t.Fatalf("expected the lexical join under linkroot %q, got %q", want, got)
	}
}

// --- toItem (via Match): fidelity EXACT, empty preview, no file body read ---

func TestMatch_TopLevelPDFYieldsExactFidelityAndEmptyPreview(t *testing.T) {
	root := t.TempDir()
	pdfBody := []byte("%PDF-1.4 fixture content that must never be read at Match time")
	if err := os.WriteFile(filepath.Join(root, "invoice.pdf"), pdfBody, 0o644); err != nil {
		t.Fatalf("write fixture pdf: %v", err)
	}

	p := NewSourcePlugin(root, nil, false)
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

// TestMatch_ExtensionOutsideDefaultAllowlistIsIgnored supersedes the
// 12-01 tracer's PDF-only "TestMatch_NonPDFFilesAreIgnored": 12-02-PLAN.md
// Task 2 widens the default allowlist to markdown/plain-text/office/image
// extensions (D-03), so a plain .txt file is now legitimately included —
// only an extension genuinely outside classify.go's extensionTable stays
// ignored with no extras configured.
func TestMatch_ExtensionOutsideDefaultAllowlistIsIgnored(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "archive.zip"), []byte("not a document"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	p := NewSourcePlugin(root, nil, false)
	resp, err := p.Match(t.Context(), &toposv1.MatchRequest{})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(resp.GetItems()) != 0 {
		t.Fatalf("expected 0 items for an extension outside the default allowlist, got %d", len(resp.GetItems()))
	}
}
