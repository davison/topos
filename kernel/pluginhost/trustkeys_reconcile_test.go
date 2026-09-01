package pluginhost

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	"github.com/davison/topos/kernel/config"
)

// TestTrustKeysChanged (M2-R4): Reconcile relaunches a running instance
// when the operator's keys would change its tier — a withdrawn key demotes
// at the apply that removed it; a newly trusted offered key promotes at
// the apply that added it; anything else keeps the instance as is.
func TestTrustKeysChanged(t *testing.T) {
	pub := make([]byte, ed25519.PublicKeySize)
	b64 := base64.StdEncoding.EncodeToString(pub)
	withKey := &config.Config{Plugins: config.PluginsConfig{TrustedKeys: []config.TrustedKey{{ID: "acme", PublicKey: b64}}}}
	without := &config.Config{}

	vouched := &Plugin{tier: TierOperatorTrusted, trustedKey: "acme"}
	if trustKeysChanged(vouched, withKey) {
		t.Error("a still-trusted key must not relaunch")
	}
	if !trustKeysChanged(vouched, without) {
		t.Error("a withdrawn key must relaunch (demote)")
	}
	if !trustKeysChanged(vouched, nil) {
		t.Error("no raw config: the key is not there — relaunch")
	}

	offered := &Plugin{tier: TierExternal, offeredKey: &KeyOffer{KeyID: "acme", PublicKey: pub}}
	if !trustKeysChanged(offered, withKey) {
		t.Error("an offered key now trusted must relaunch (promote)")
	}
	if trustKeysChanged(offered, without) {
		t.Error("an offer nobody trusted changes nothing")
	}
	other := &config.Config{Plugins: config.PluginsConfig{TrustedKeys: []config.TrustedKey{{ID: "acme", PublicKey: base64.StdEncoding.EncodeToString(make([]byte, 32, 32)[:31])}}}}
	if trustKeysChanged(offered, other) {
		t.Error("the same id with different bytes is not the offered key")
	}

	plain := &Plugin{tier: TierExternal}
	if trustKeysChanged(plain, withKey) {
		t.Error("an external instance with no offer is untouched by keys")
	}
	trusted := &Plugin{tier: TierTrusted}
	if trustKeysChanged(trusted, without) {
		t.Error("the kernel author's word is untouched by operator keys")
	}
}
