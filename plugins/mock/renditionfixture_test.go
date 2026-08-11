package main

import (
	"context"
	"testing"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// TestFetch_RenditionFixtureOff_ByteIdenticalToNoFixture proves the
// fixture-off default: every branch returns exactly what it returned
// before this fixture existed. This is asserted as a test, not trusted,
// per 09-04-PLAN.md Task 3's own instruction — plugin.go is also carrying
// 09-01's icon wiring, and a regression here would otherwise be
// misattributed to that unrelated change.
func TestFetch_RenditionFixtureOff_ByteIdenticalToNoFixture(t *testing.T) {
	p := NewSourcePlugin() // renditionFixture defaults to false (zero value)

	full, err := p.Fetch(context.Background(), &toposv1.FetchRequest{
		SourceId: renditionFixtureItemID, Variant: toposv1.ContentVariant_CONTENT_VARIANT_FULL,
	})
	if err != nil {
		t.Fatalf("Fetch FULL: %v", err)
	}
	if !full.GetAvailable() {
		t.Error("expected available=true (extracted text is present) for the FULL variant")
	}
	if full.GetText() == "" {
		t.Error("expected non-empty extracted text")
	}
	if full.GetMimeType() != "" {
		t.Errorf("expected no rendition mime_type with the fixture off, got %q", full.GetMimeType())
	}
	if full.GetSizeBytes() != 0 {
		t.Errorf("expected size_bytes 0 with the fixture off, got %d", full.GetSizeBytes())
	}
	if len(full.GetData()) != 0 {
		t.Errorf("expected no rendition data with the fixture off, got %d bytes", len(full.GetData()))
	}

	for _, variant := range []toposv1.ContentVariant{
		toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW,
		toposv1.ContentVariant_CONTENT_VARIANT_THUMBNAIL,
	} {
		resp, err := p.Fetch(context.Background(), &toposv1.FetchRequest{
			SourceId: renditionFixtureItemID, Variant: variant,
		})
		if err != nil {
			t.Fatalf("Fetch %v: %v", variant, err)
		}
		if resp.GetAvailable() {
			t.Errorf("Fetch %v: expected available=false with the fixture off, got true", variant)
		}
		if resp.GetUnavailableReason() != noRenditionReason {
			t.Errorf("Fetch %v: expected unavailable_reason %q, got %q", variant, noRenditionReason, resp.GetUnavailableReason())
		}
		if len(resp.GetData()) != 0 {
			t.Errorf("Fetch %v: expected no rendition data with the fixture off, got %d bytes", variant, len(resp.GetData()))
		}
	}
}

// TestFetch_RenditionFixtureOn_FullVariantCarriesRenditionDescriptor
// proves the FULL response for the designated fixture item additionally
// carries a rendition descriptor (mime image/png) once the fixture is on
// — the trigger for DetailPane.svelte's detailBodyVariant to route to the
// media branch (web/src/lib/format.ts).
func TestFetch_RenditionFixtureOn_FullVariantCarriesRenditionDescriptor(t *testing.T) {
	p := NewSourcePlugin().withRenditionFixture(true)

	resp, err := p.Fetch(context.Background(), &toposv1.FetchRequest{
		SourceId: renditionFixtureItemID, Variant: toposv1.ContentVariant_CONTENT_VARIANT_FULL,
	})
	if err != nil {
		t.Fatalf("Fetch FULL: %v", err)
	}
	if !resp.GetAvailable() {
		t.Error("expected available=true")
	}
	if resp.GetText() == "" {
		t.Error("expected the extracted text to still be present alongside the rendition descriptor")
	}
	if resp.GetMimeType() != fixtureRenditionMIME {
		t.Errorf("expected mime_type %q, got %q", fixtureRenditionMIME, resp.GetMimeType())
	}
	if resp.GetSizeBytes() != int64(len(fixtureRenditionPNG)) {
		t.Errorf("expected size_bytes %d, got %d", len(fixtureRenditionPNG), resp.GetSizeBytes())
	}
}

// TestFetch_RenditionFixtureOn_PreviewVariantReturnsEmbeddedBytes proves
// the PREVIEW response for the fixture item returns the embedded PNG
// bytes with mime image/png instead of available=false, once the fixture
// is on — this is the byte stream kernel/httpapi's renditionHandler
// serves through GET /api/items/{id}/content.
func TestFetch_RenditionFixtureOn_PreviewVariantReturnsEmbeddedBytes(t *testing.T) {
	p := NewSourcePlugin().withRenditionFixture(true)

	resp, err := p.Fetch(context.Background(), &toposv1.FetchRequest{
		SourceId: renditionFixtureItemID, Variant: toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW,
	})
	if err != nil {
		t.Fatalf("Fetch PREVIEW: %v", err)
	}
	if !resp.GetAvailable() {
		t.Fatal("expected available=true with the fixture on")
	}
	if resp.GetMimeType() != fixtureRenditionMIME {
		t.Errorf("expected mime_type %q, got %q", fixtureRenditionMIME, resp.GetMimeType())
	}
	if string(resp.GetData()) != string(fixtureRenditionPNG) {
		t.Error("expected the PREVIEW response's data to be the exact embedded fixture PNG bytes")
	}
	if resp.GetSizeBytes() != int64(len(fixtureRenditionPNG)) {
		t.Errorf("expected size_bytes %d, got %d", len(fixtureRenditionPNG), resp.GetSizeBytes())
	}
}

// TestFetch_RenditionFixtureOn_EveryOtherItemUnaffected proves the
// fixture is scoped to renditionFixtureItemID alone — every other mock
// item stays on the no-rendition path even with the fixture on, so the
// fixture-off behaviour stays exercisable in the same run.
func TestFetch_RenditionFixtureOn_EveryOtherItemUnaffected(t *testing.T) {
	p := NewSourcePlugin().withRenditionFixture(true)

	for _, it := range mockItems {
		if it.GetSourceId() == renditionFixtureItemID {
			continue
		}
		resp, err := p.Fetch(context.Background(), &toposv1.FetchRequest{
			SourceId: it.GetSourceId(), Variant: toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW,
		})
		if err != nil {
			t.Fatalf("Fetch PREVIEW for item %q: %v", it.GetSourceId(), err)
		}
		if resp.GetAvailable() {
			t.Errorf("item %q: expected available=false (only %q carries the fixture rendition), got true", it.GetSourceId(), renditionFixtureItemID)
		}
		if resp.GetUnavailableReason() != noRenditionReason {
			t.Errorf("item %q: expected unavailable_reason %q, got %q", it.GetSourceId(), noRenditionReason, resp.GetUnavailableReason())
		}
	}
}

// TestRenditionFixtureEnabled is a table test over the fixture's env-var
// parsing, mirroring readiness.go's own table-test style: absent, empty,
// and "0" all mean off; any other non-empty value means on (a simple
// boolean gate, unlike the two sibling fixtures in readiness.go — no
// numeric parsing, so no malformed-value error path to cover here).
func TestRenditionFixtureEnabled(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		present bool
		want    bool
	}{
		{name: "absent", present: false, want: false},
		{name: "empty", raw: "", present: true, want: false},
		{name: "zero", raw: "0", present: true, want: false},
		{name: "one", raw: "1", present: true, want: true},
		{name: "true", raw: "true", present: true, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(name string) string {
				if name != renditionFixtureEnvVar {
					t.Fatalf("unexpected getenv call for %q", name)
				}
				if !tc.present {
					return ""
				}
				return tc.raw
			}

			got := renditionFixtureEnabled(getenv)
			if got != tc.want {
				t.Errorf("renditionFixtureEnabled(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}
