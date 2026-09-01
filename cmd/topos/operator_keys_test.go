package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/pluginhost"
)

// TestInstallOperatorKeys_InstallsTheActiveConfigsKeys (M2-R4): the helper
// both entry points call puts the config's trusted keys into the accepted
// set, with the operator's word, and an empty table clears them — so a
// server with zero configured sources still lists an operator-signed
// binary as operator_trusted from its first request.
func TestInstallOperatorKeys_InstallsTheActiveConfigsKeys(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	t.Cleanup(func() { pluginhost.SetOperatorProvenanceKeys(nil) })
	installOperatorKeys(&config.Config{Plugins: config.PluginsConfig{TrustedKeys: []config.TrustedKey{{ID: "acme-2026a", PublicKey: base64.StdEncoding.EncodeToString(pub)}}}})
	found := false
	for _, k := range pluginhost.AcceptedProvenanceKeys() {
		if k.ID == "acme-2026a" && k.Word == pluginhost.KeyWordOperator && k.PublicKey.Equal(pub) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected acme-2026a in the accepted set with the operator's word: %+v", pluginhost.AcceptedProvenanceKeys())
	}
	installOperatorKeys(&config.Config{})
	for _, k := range pluginhost.AcceptedProvenanceKeys() {
		if k.ID == "acme-2026a" {
			t.Fatal("an empty table must clear the operator's keys")
		}
	}
}
