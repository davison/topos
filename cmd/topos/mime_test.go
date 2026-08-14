package main

import (
	"mime"
	"testing"

	"github.com/hashicorp/go-hclog"
)

// TestRegisterManifestMimeType_WebmanifestResolvesToManifestJSON pins
// 13-RESEARCH.md Pitfall 4: after registerManifestMimeType runs,
// mime.TypeByExtension(".webmanifest") must return a media type whose
// parsed type is exactly "application/manifest+json" — the type browsers
// check when deciding PWA installability — regardless of what the host
// OS's own mime.types database does or doesn't already know about this
// extension.
func TestRegisterManifestMimeType_WebmanifestResolvesToManifestJSON(t *testing.T) {
	registerManifestMimeType(hclog.NewNullLogger())

	got := mime.TypeByExtension(".webmanifest")
	mediaType, _, err := mime.ParseMediaType(got)
	if err != nil {
		t.Fatalf("mime.TypeByExtension(%q) = %q, not a parseable media type: %v", ".webmanifest", got, err)
	}
	if mediaType != "application/manifest+json" {
		t.Fatalf("mime.TypeByExtension(%q) media type = %q, want %q", ".webmanifest", mediaType, "application/manifest+json")
	}
}
