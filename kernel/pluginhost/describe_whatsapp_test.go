package pluginhost

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
)

// buildWhatsAppPluginDir builds the repo's real plugins/whatsapp binary
// fresh, once per test binary run (mirrors buildMockPluginDir's own
// sync.Once shape, reconcile_test.go) — needed because this file's
// regression test (below) must exercise the REAL storelock.go/connect.go
// startup sequence, not the mock reference plugin, to prove CR-01
// (08-REVIEW.md) actually stays fixed.
var (
	whatsappPluginDirOnce sync.Once
	whatsappPluginDir     string
	whatsappPluginDirErr  error
)

func buildWhatsAppPluginDir(t *testing.T) string {
	t.Helper()
	whatsappPluginDirOnce.Do(func() {
		out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/davison/topos").Output()
		if err != nil {
			whatsappPluginDirErr = fmt.Errorf("resolve module root: %w", err)
			return
		}
		root := strings.TrimSpace(string(out))

		dir, err := os.MkdirTemp("", "topos-pluginhost-describe-whatsapp-test-*")
		if err != nil {
			whatsappPluginDirErr = err
			return
		}

		bin := filepath.Join(dir, "topos-plugin-whatsapp")
		cmd := exec.Command("go", "build", "-o", bin, "./plugins/whatsapp")
		cmd.Dir = root
		if buildOut, err := cmd.CombinedOutput(); err != nil {
			whatsappPluginDirErr = fmt.Errorf("build whatsapp plugin: %w\n%s", err, buildOut)
			return
		}

		whatsappPluginDir = dir
	})
	if whatsappPluginDirErr != nil {
		t.Fatalf("build whatsapp plugin fixture: %v", whatsappPluginDirErr)
	}
	return whatsappPluginDir
}

// TestDescribePluginType_WhatsApp_SucceedsWhileARealInstanceHoldsTheStoreLock
// is CR-01's (08-REVIEW.md) regression test: it launches a REAL
// topos-plugin-whatsapp instance against a real (empty, never-linked) data
// directory — acquiring and holding storelock.go's exclusive flock for the
// launched subprocess's entire lifetime, exactly like a real running
// kernel's configured WhatsApp source always does — then, WHILE that
// instance is still alive and holding the lock, calls DescribePluginType
// against the SAME plugin binary and the SAME data directory (the shape
// both "Edit match settings…" (+page.svelte) and the "+" picker's
// add-existing-instance flow (AddSourceModal.svelte's selectExisting)
// submit today).
//
// Before CR-01's fix, this always failed: DescribePluginType's trial
// launch reached NewSourcePlugin -> startBackgroundClient ->
// acquireStoreLock unconditionally, lost the non-blocking flock race
// against the already-running instance, and the trial subprocess exited
// (fatal) before goplugin.Serve's handshake ever completed — surfacing as
// "pluginhost: trial-launch for describe: …" naming a handshake failure,
// never reaching Describe at all. After the fix (describeOnlyEnvVar,
// plugins/whatsapp/main.go + describeonly.go), the trial launch never
// touches the store lock, so it succeeds and returns the plugin's real
// Describe-declared vocabulary.
func TestDescribePluginType_WhatsApp_SucceedsWhileARealInstanceHoldsTheStoreLock(t *testing.T) {
	dir := buildWhatsAppPluginDir(t)
	dataDir := t.TempDir()

	src := config.Source{Plugin: "topos-plugin-whatsapp", Path: dataDir}

	// Launch a REAL running instance the way the kernel's own boot-time
	// Discover does — this is what holds the exclusive store lock for as
	// long as this test's *Host is alive, standing in for "a WhatsApp
	// source that is currently linked and running" (08-REVIEW.md CR-01's
	// own framing of the normal, intended state this defect broke).
	h, err := Discover(context.Background(), Dirs{Trusted: dir}, map[string]config.Source{
		"whatsapp-test": src,
	}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover (launch the real running instance): %v", err)
	}
	defer h.Shutdown()

	// While that instance is still alive and holding dataDir's store lock,
	// describe the SAME plugin+source — this is exactly what
	// DescribePluginHandler does for "Edit match settings…" and the "+"
	// picker's add-existing-instance flow.
	info, err := DescribePluginType(context.Background(), Dirs{Trusted: dir}, src, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("DescribePluginType against an already-running instance's data directory: %v", err)
	}
	if info.SourceType != "whatsapp" {
		t.Errorf("expected source_type %q, got %q", "whatsapp", info.SourceType)
	}
	if info.PluginDisplayName != "WhatsApp" {
		t.Errorf("expected plugin_display_name %q, got %q", "WhatsApp", info.PluginDisplayName)
	}
	wantVocab := map[string]bool{"groups": true, "contacts": true}
	if len(info.MatchVocabulary) != len(wantVocab) {
		t.Fatalf("expected match_vocabulary %v, got %v", []string{"groups", "contacts"}, info.MatchVocabulary)
	}
	for _, field := range info.MatchVocabulary {
		if !wantVocab[field] {
			t.Errorf("unexpected match_vocabulary field %q, want one of %v", field, wantVocab)
		}
	}

	// The real running instance must still be alive and unaffected by the
	// describe call — Plugins() still reports exactly the one instance
	// Discover launched, proving DescribePluginType registered nothing on
	// this *Host (D-11's own "writing nothing to disk and registering
	// nothing on the running kernel's plugin host" guarantee).
	if len(h.Plugins()) != 1 || h.Plugins()[0].Name() != "whatsapp-test" {
		t.Fatalf("expected the real running instance to be the sole, untouched entry in Plugins(), got %+v", h.Plugins())
	}
}
