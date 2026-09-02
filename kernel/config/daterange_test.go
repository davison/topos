package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The date narrowing's config half (M3-R1, #70): valid shapes load and
// resolve to day-inclusive bounds; malformed and inverted shapes fail by
// name at load.
func TestLoad_DateRange(t *testing.T) {
	write := func(t *testing.T, body string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	base := `
[sources.mock-01]
plugin = "topos-plugin-mock"
path = "/tmp"

[webspaces.holiday]
keywords = ["demo"]
`
	t.Run("both sides load and resolve day-inclusively", func(t *testing.T) {
		cfg, err := Load(write(t, base+"date_from = \"2026-03-01\"\ndate_to = \"2026-03-31\"\n"))
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		from, to := cfg.Webspaces["holiday"].DateRange()
		wantFrom := time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local).Unix()
		wantTo := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local).Unix() - 1
		if from != wantFrom || to != wantTo {
			t.Fatalf("DateRange: got (%d,%d), want (%d,%d)", from, to, wantFrom, wantTo)
		}
	})
	t.Run("one-sided ranges leave the other bound zero", func(t *testing.T) {
		cfg, err := Load(write(t, base+"date_from = \"2026-03-01\"\n"))
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		from, to := cfg.Webspaces["holiday"].DateRange()
		if from == 0 || to != 0 {
			t.Fatalf("open-ended: got (%d,%d)", from, to)
		}
	})
	for name, line := range map[string]string{
		"not a date":       "date_from = \"soon\"\n",
		"datetime not day": "date_to = \"2026-03-01T10:00:00Z\"\n",
	} {
		t.Run(name+" fails load", func(t *testing.T) {
			if _, err := Load(write(t, base+line)); err == nil || !strings.Contains(err.Error(), "calendar date") {
				t.Fatalf("want a calendar-date load error, got: %v", err)
			}
		})
	}
	t.Run("inverted range fails load", func(t *testing.T) {
		if _, err := Load(write(t, base+"date_from = \"2026-04-01\"\ndate_to = \"2026-03-01\"\n")); err == nil || !strings.Contains(err.Error(), "inverted") {
			t.Fatalf("want the inverted-range error, got: %v", err)
		}
	})
}
