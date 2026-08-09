package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/davison/topos/sdk"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

func TestDescribe_ReturnsMockstrictIdentityAndTagsVocabulary(t *testing.T) {
	p := NewSourcePlugin("/anything")
	resp, err := p.Describe(context.Background(), &toposv1.DescribeRequest{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if resp.GetSourceType() != "mockstrict" {
		t.Errorf("expected source_type %q, got %q", "mockstrict", resp.GetSourceType())
	}
	if resp.GetDisplayName() == "" {
		t.Error("expected a non-empty display_name")
	}
	if resp.GetContractVersion() != "topos.v2" {
		t.Errorf("expected contract_version %q, got %q", "topos.v2", resp.GetContractVersion())
	}
	if len(resp.GetMatchVocabulary()) != 1 || resp.GetMatchVocabulary()[0] != "tags" {
		t.Errorf("expected match_vocabulary [\"tags\"], got %v", resp.GetMatchVocabulary())
	}
}

// matchFieldsReq builds a MatchRequest carrying a single "tags" field —
// the shape the kernel sends this plugin at sync time.
func matchFieldsReq(tags []string) *toposv1.MatchRequest {
	return &toposv1.MatchRequest{MatchFields: map[string]*toposv1.StringList{
		"tags": {Values: tags},
	}}
}

// TestMatch_ExactCaseInsensitiveMatch proves "STRICT" (different case)
// matches item(s) labelled "strict" and returns exactly those.
func TestMatch_ExactCaseInsensitiveMatch(t *testing.T) {
	p := NewSourcePlugin("/anything")
	resp, err := p.Match(context.Background(), matchFieldsReq([]string{"STRICT"}))
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	got := map[string]bool{}
	for _, it := range resp.GetItems() {
		got[it.GetSourceId()] = true
	}
	want := map[string]bool{"s1": true, "s3": true}
	if len(got) != len(want) {
		t.Fatalf("expected items %v, got %v", want, got)
	}
	for id := range want {
		if !got[id] {
			t.Errorf("expected item %q in the result set, got %v", id, got)
		}
	}
}

// TestMatch_NoSubstringMatching proves the contract's exact-match rule:
// a keyword that is a substring of a label (but not equal to it) does
// not match — "stric" must not match "strict".
func TestMatch_NoSubstringMatching(t *testing.T) {
	p := NewSourcePlugin("/anything")
	resp, err := p.Match(context.Background(), matchFieldsReq([]string{"stric"}))
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(resp.GetItems()) != 0 {
		t.Errorf("expected zero items for a substring-only keyword, got %d", len(resp.GetItems()))
	}
}

// TestMatch_EmptyValueListMatchesNothing proves an empty or absent "tags"
// value list matches nothing.
func TestMatch_EmptyValueListMatchesNothing(t *testing.T) {
	p := NewSourcePlugin("/anything")
	resp, err := p.Match(context.Background(), matchFieldsReq(nil))
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(resp.GetItems()) != 0 {
		t.Errorf("expected zero items for an empty 'tags' value list, got %d", len(resp.GetItems()))
	}
}

// TestMatch_UndeclaredKeyIsIgnored proves a match_fields key outside the
// plugin's declared vocabulary ("labels", which this plugin never
// declares) is ignored entirely — matches only on the declared "tags"
// field.
func TestMatch_UndeclaredKeyIsIgnored(t *testing.T) {
	p := NewSourcePlugin("/anything")
	req := &toposv1.MatchRequest{MatchFields: map[string]*toposv1.StringList{
		"tags":   {Values: []string{"fixture"}},
		"labels": {Values: []string{"should-be-ignored"}},
	}}
	resp, err := p.Match(context.Background(), req)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	// "fixture" labels s2 and s3.
	if len(resp.GetItems()) != 2 {
		t.Fatalf("expected the undeclared 'labels' key to be ignored and both 'fixture'-labelled items returned, got %d", len(resp.GetItems()))
	}
}

// TestFetch_FullVariantForKnownIDReturnsTextAndNoRendition proves the
// full-variant Fetch reports extracted text, no rendition available, and
// a non-empty reason.
func TestFetch_FullVariantForKnownIDReturnsTextAndNoRendition(t *testing.T) {
	p := NewSourcePlugin("/anything")
	resp, err := p.Fetch(context.Background(), &toposv1.FetchRequest{
		SourceId: "s1", Variant: toposv1.ContentVariant_CONTENT_VARIANT_FULL,
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if resp.GetText() == "" {
		t.Error("expected non-empty extracted text")
	}
	if resp.GetAvailable() {
		t.Error("expected available=false (this plugin never has a rendition)")
	}
	if resp.GetUnavailableReason() == "" {
		t.Error("expected a non-empty unavailable_reason")
	}
}

// TestFetch_UnknownIDReturnsNotFound proves an unknown source id maps to
// a gRPC codes.NotFound error.
func TestFetch_UnknownIDReturnsNotFound(t *testing.T) {
	p := NewSourcePlugin("/anything")
	_, err := p.Fetch(context.Background(), &toposv1.FetchRequest{
		SourceId: "does-not-exist", Variant: toposv1.ContentVariant_CONTENT_VARIANT_FULL,
	})
	if err == nil {
		t.Fatal("expected an error for an unknown source id")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Errorf("expected codes.NotFound, got %v", err)
	}
}

// TestFetch_UnspecifiedVariantReturnsInvalidArgument proves
// CONTENT_VARIANT_UNSPECIFIED maps to codes.InvalidArgument.
func TestFetch_UnspecifiedVariantReturnsInvalidArgument(t *testing.T) {
	p := NewSourcePlugin("/anything")
	_, err := p.Fetch(context.Background(), &toposv1.FetchRequest{
		SourceId: "s1", Variant: toposv1.ContentVariant_CONTENT_VARIANT_UNSPECIFIED,
	})
	if err == nil {
		t.Fatal("expected an error for an unspecified content variant")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected codes.InvalidArgument, got %v", err)
	}
}

// TestHealth_AlwaysReachableWithNoError proves this plugin always reports
// reachable with no error — it has nothing to be unreachable from.
func TestHealth_AlwaysReachableWithNoError(t *testing.T) {
	p := NewSourcePlugin("/anything")
	resp, err := p.Health(context.Background(), &toposv1.HealthRequest{})
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if !resp.GetReachable() {
		t.Error("expected reachable=true")
	}
	if resp.GetLastError() != "" {
		t.Errorf("expected empty last_error, got %q", resp.GetLastError())
	}
}

// TestMain_BlankPathExitsNonZeroBeforeServe and
// TestMain_NonEmptyPathDoesNotExitImmediately build the binary and run it
// as a subprocess, proving main()'s fatal guard actually fires (or does
// not) on a real process boundary — a unit test calling functions inside
// this package cannot observe os.Exit or goplugin.Serve's blocking
// behavior, so this is deliberately an integration-style test.
func TestMain_BlankPathExitsNonZeroBeforeServe(t *testing.T) {
	bin := buildMockstrictBinary(t)

	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), `WEBSPACES_SOURCE_CONFIG={"path":""}`)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected a non-zero exit for a blank path, got success; output: %s", out)
	}
	var exitErr *exec.ExitError
	if !isExitError(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %v (output: %s)", err, out)
	}
	if exitErr.ExitCode() == 0 {
		t.Errorf("expected non-zero exit code, got 0")
	}
	if len(out) == 0 {
		t.Error("expected a stderr message naming the empty field")
	}
}

func TestMain_NonEmptyPathDoesNotExitImmediately(t *testing.T) {
	bin := buildMockstrictBinary(t)

	cmd := exec.Command(bin)
	// go-plugin's Serve refuses to run at all (prints its "not meant to be
	// executed directly" notice and exits 1) unless the magic-cookie env
	// var sdk.Handshake declares is set — normally supplied by the kernel's
	// pluginhost launcher, never by a human or a test harness invoking the
	// binary directly. Without it, this test could not distinguish "the
	// blank-path guard did not fire" from "go-plugin's own unrelated
	// direct-execution refusal fired instead".
	cmd.Env = append(os.Environ(),
		`WEBSPACES_SOURCE_CONFIG={"path":"/anything"}`,
		sdk.Handshake.MagicCookieKey+"="+sdk.Handshake.MagicCookieValue,
	)
	if startErr := cmd.Start(); startErr != nil {
		t.Fatalf("start: %v", startErr)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		t.Fatalf("expected the process to still be running (blocked serving) after a short wait, but it exited: %v", err)
	case <-time.After(300 * time.Millisecond):
		// Still running — the guard did not fire. Clean up.
		_ = cmd.Process.Kill()
		<-done
	}
}

func isExitError(err error, target **exec.ExitError) bool {
	ee, ok := err.(*exec.ExitError)
	if ok {
		*target = ee
	}
	return ok
}

// envWithout returns a copy of env with any entry whose key equals name
// removed — used to override a variable via append without relying on
// duplicate-key resolution order, which is not guaranteed across
// platforms.
func envWithout(env []string, name string) []string {
	prefix := name + "="
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// buildMockstrictBinary compiles this package into a temp binary once per
// test process run, cached via t.TempDir() per-test (acceptable here:
// these two tests are the only callers and the build is fast/cgo-free).
func buildMockstrictBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := dir + "/topos-plugin-mockstrict"
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Env = append(envWithout(os.Environ(), "CGO_ENABLED"), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}
