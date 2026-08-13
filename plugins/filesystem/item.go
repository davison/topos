package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// relPathSourceID returns full's path relative to root, using forward
// slashes and never a leading "./" or an absolute path (D-01, 12-CONTEXT.md)
// — the per-source stable item identity. full must be inside root; if
// filepath.Rel fails (a genuinely unrelated path — never expected in normal
// operation since every caller derives full from a root-rooted walk), the
// bare file name is returned as a defensive fallback rather than panicking.
func relPathSourceID(root, full string) string {
	rel, err := filepath.Rel(root, full)
	if err != nil {
		return filepath.Base(full)
	}
	return filepath.ToSlash(rel)
}

// folderLabels returns the D-05 folder-vocabulary labels for full: for a
// top-level file (directly in root), the source root's own base name;
// for a nested file, one label per containing-directory segment's own
// name PLUS the cumulative relative directory path — so a webspace match
// block can name either a subfolder name alone or a full relative path
// (12-03-PLAN.md Task 2). A file at "receipts/2026/inv.pdf" under a root
// named "docs" carries the labels "receipts", "2026" and
// "receipts/2026"; a file directly in the root still carries only
// "docs", byte-identical to before recursion existed.
func folderLabels(root, full string) []string {
	dir := filepath.Dir(full)
	if dir == filepath.Clean(root) {
		return []string{filepath.Base(root)}
	}

	relDir, err := filepath.Rel(root, dir)
	if err != nil {
		return []string{filepath.Base(dir)}
	}
	relDir = filepath.ToSlash(relDir)

	segments := strings.Split(relDir, "/")
	labels := make([]string, 0, len(segments)+1)
	labels = append(labels, segments...)
	if len(segments) > 1 {
		labels = append(labels, relDir)
	}
	return labels
}

// fileDeepLink builds a file:// URI over the real absolute path (root
// joined with sourceID, a D-01 relative path) — the mechanism the Task 1
// checkpoint locked (option-a, 12-RESEARCH.md Architecture Pattern 3): the
// kernel rewrites this to the loopback open route at serve time
// (kernel/httpapi/stream.go's resolveStreamLinkURL), keyed on the URL
// scheme alone, never on source_type.
func fileDeepLink(root, sourceID string) string {
	full := filepath.Join(root, filepath.FromSlash(sourceID))
	return "file://" + filepath.ToSlash(full)
}

// resolvePath joins root with sourceID (a D-01 relative path) and
// re-validates the result is still lexically inside root, refusing an
// escape via ".." segments — defense-in-depth even though sourceID
// originates from this plugin's own trusted directory read, never a
// request. Mirrors kernel/httpapi/fsopen.go's identical guard on the
// kernel side of this same join.
func resolvePath(root, sourceID string) (string, error) {
	cleanRoot := filepath.Clean(root)
	full := filepath.Join(cleanRoot, filepath.FromSlash(sourceID))
	if full != cleanRoot && !strings.HasPrefix(full, cleanRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("resolved path escapes source root")
	}
	return full, nil
}
