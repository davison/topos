// main_test.go pins 16-REVIEW.md WR-02's fix: runKeygen and runSign must
// reject a --key-id outside pluginhost.ValidateProvenanceKeyID's
// restricted charset at the point the id is chosen, with a clear,
// actionable error — never let it reach FormatProvenanceKeys/
// ParseProvenanceKeys and surface later as an opaque "malformed
// provenance key segment" parse failure. Both checks fire before any
// other flag requirement, so a case can name ONLY --key-id and still
// exercise the check in isolation.
package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/davison/topos/kernel/pluginhost"
)

// TestRunKeygen_RejectsKeyIDOutsideCharset proves runKeygen refuses a
// key id containing a "," or "=" — the two characters that would
// silently break FormatProvenanceKeys/ParseProvenanceKeys's own
// comma-separated "keyid=<base64>" round trip — before ever generating a
// keypair or writing a file.
func TestRunKeygen_RejectsKeyIDOutsideCharset(t *testing.T) {
	cases := []string{"has,comma", "has=equals", "has space"}
	for _, id := range cases {
		err := runKeygen([]string{"--key-id", id, "--out-dir", t.TempDir()})
		if err == nil {
			t.Fatalf("runKeygen(--key-id=%q): expected an error, got nil", id)
		}
		if !strings.Contains(err.Error(), "keygen:") {
			t.Errorf("runKeygen(--key-id=%q): expected the error to be prefixed by its own subcommand, got: %v", id, err)
		}
		var wantErr error
		if valErr := pluginhost.ValidateProvenanceKeyID(id); valErr != nil {
			wantErr = valErr
		}
		if wantErr == nil {
			t.Fatalf("test bug: fixture id %q unexpectedly passes ValidateProvenanceKeyID", id)
		}
		if !errors.Is(err, wantErr) && !strings.Contains(err.Error(), "allowed charset") {
			t.Errorf("runKeygen(--key-id=%q): expected the wrapped ValidateProvenanceKeyID error naming the disallowed charset, got: %v", id, err)
		}
	}
}

// TestRunSign_RejectsKeyIDOutsideCharset mirrors
// TestRunKeygen_RejectsKeyIDOutsideCharset for the sign subcommand,
// which the review flagged as also worth validating ("ideally in
// runSign, which also accepts --key-id").
func TestRunSign_RejectsKeyIDOutsideCharset(t *testing.T) {
	cases := []string{"has,comma", "has=equals"}
	for _, id := range cases {
		// A positional binary-path argument is required to get past
		// runSign's own "zero binary-path arguments" refusal, which is
		// checked before --key-id — it does not need to resolve to a
		// real file, since key-id validation happens before the binary
		// is ever hashed.
		err := runSign([]string{"--key-id", id, "unused-fixture-path"})
		if err == nil {
			t.Fatalf("runSign(--key-id=%q): expected an error, got nil", id)
		}
		if !strings.Contains(err.Error(), "sign:") {
			t.Errorf("runSign(--key-id=%q): expected the error to be prefixed by its own subcommand, got: %v", id, err)
		}
		if !strings.Contains(err.Error(), "allowed charset") {
			t.Errorf("runSign(--key-id=%q): expected the charset-violation error, got: %v", id, err)
		}
	}
}

// TestRunKeygen_ValidKeyIDStillSucceeds is the tracer/negative-control:
// an ordinary identifier-shaped --key-id must still work end to end
// (keypair generated, spec printed), proving the new validation is not
// over-broad.
func TestRunKeygen_ValidKeyIDStillSucceeds(t *testing.T) {
	if err := runKeygen([]string{"--key-id", "release-key.v2_2026a", "--out-dir", t.TempDir()}); err != nil {
		t.Fatalf("runKeygen with a valid --key-id: %v", err)
	}
}
