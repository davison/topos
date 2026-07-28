package main

import "testing"

// TestMatchesKeyword_TagCaseInsensitive proves D-03's tag match: exact,
// case-insensitive against a tag value.
func TestMatchesKeyword_TagCaseInsensitive(t *testing.T) {
	if !MatchesKeyword("projects/House move", []string{"house and home"}, "House And Home") {
		t.Fatal("expected a case-insensitive exact tag match")
	}
}

// TestMatchesKeyword_FinalPathSegment proves D-03's name match against the
// page path's final segment, case-insensitive.
func TestMatchesKeyword_FinalPathSegment(t *testing.T) {
	if !MatchesKeyword("projects/House move", nil, "house move") {
		t.Fatal("expected a case-insensitive match against the final path segment")
	}
}

// TestMatchesKeyword_NoPrefixMatch proves D-03's exclusion: no
// prefix/substring matching against the page name.
func TestMatchesKeyword_NoPrefixMatch(t *testing.T) {
	if MatchesKeyword("projects/House move", nil, "house") {
		t.Fatal("expected no match: \"house\" is a prefix of \"House move\", not an exact match")
	}
}

// TestMatchesKeyword_FullPath proves D-03's name match against the full
// space-relative path (extension already stripped by the caller).
func TestMatchesKeyword_FullPath(t *testing.T) {
	if !MatchesKeyword("projects/House move", nil, "projects/house move") {
		t.Fatal("expected a case-insensitive match against the full path")
	}
}

// TestIsPage_ExcludesUnderscorePaths proves isPage filters SilverBullet's
// own leading-underscore library/plug paths out of webspace matching.
func TestIsPage_ExcludesUnderscorePaths(t *testing.T) {
	if isPage(FileMeta{Name: "_plug/foo.md"}) {
		t.Error("expected _plug/foo.md to be excluded")
	}
	if !isPage(FileMeta{Name: "notes/a.md"}) {
		t.Error("expected notes/a.md to be included")
	}
	if isPage(FileMeta{Name: "img/a.png"}) {
		t.Error("expected img/a.png (non-markdown) to be excluded")
	}
}

// TestExtractTagsAndBody_FrontmatterAndInlineUnion proves ExtractTagsAndBody
// strips YAML frontmatter and unions its tags: values with inline #tags
// found in the remaining body.
func TestExtractTagsAndBody_FrontmatterAndInlineUnion(t *testing.T) {
	raw := []byte("---\ntags: [house and home, admin]\n---\nbody #urgent")
	body, tags := ExtractTagsAndBody(raw)

	if string(body) != "body #urgent" {
		t.Errorf("expected the frontmatter-stripped body to be %q, got %q", "body #urgent", string(body))
	}

	want := map[string]bool{"house and home": true, "admin": true, "urgent": true}
	if len(tags) != len(want) {
		t.Fatalf("expected %d tags, got %d: %v", len(want), len(tags), tags)
	}
	for _, tag := range tags {
		if !want[tag] {
			t.Errorf("unexpected tag %q", tag)
		}
	}
}
