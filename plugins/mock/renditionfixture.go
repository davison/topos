// This file holds a TEST FIXTURE mechanism and is NOT part of the plugin
// contract — no real plugin needs it, and docs/plugin-contract.md is
// deliberately not amended to mention it (mirrors readiness.go's own
// header exactly). It exists because the mock plugin, by design, never
// has a byte rendition to offer for any item (see plugin.go's
// noRenditionReason) — which means DetailPane.svelte's `bodyVariant ===
// 'media'` branch (09-UI-SPEC.md Fix 9) was otherwise unreachable by this
// repo's hermetic browser harness: nothing in the closed mock-shaped
// plugin set could ever route an item through it. This fixture gives one
// designated mock item a fake PNG rendition, gated behind an env var that
// is unset in every normal launch, so Fix 9's media-box geometry can be
// proven end to end in a real browser.
package main

import _ "embed"

// renditionFixtureEnvVar names the env var that opts this plugin into
// emitting a fixture rendition for renditionFixtureItemID. The
// "WEBSPACES_" prefix matches readiness.go's own two env vars and the two
// the kernel always sets on every plugin subprocess
// (WEBSPACES_SOURCE_CONFIG, WEBSPACES_DESCRIBE_ONLY) — inherited by the
// subprocess exactly the same way (kernel/pluginhost.launch builds the
// subprocess's environment as append(os.Environ(), ...), so a value set on
// the KERNEL process, e.g. by an e2e fixture spawning it, reaches this
// subprocess whole).
const renditionFixtureEnvVar = "WEBSPACES_MOCK_RENDITION"

// renditionFixtureItemID names the ONE mock item (see plugin.go's
// mockItems) the fixture attaches its rendition to when enabled — item
// "1", which also carries non-empty extracted text (mockFullText["1"]),
// so a single item can exercise both Fix 9 UI considerations at once: the
// media box floating beside flowing text (E2 populated) with the fixture
// on, and the ordinary no-rendition text-only branch (E2 empty, unrelated
// item) with it off. Every OTHER item is deliberately unaffected by this
// fixture regardless of its setting — the no-rendition path stays
// exercisable in the same run.
const renditionFixtureItemID = "1"

// fixtureRenditionMIME is the declared mime for fixtureRenditionPNG,
// returned verbatim in FetchResponse.mime_type — "image/png" is
// confirmed present on kernel/httpapi/item.go's allowedRenditionTypes
// allowlist (09-04-PLAN.md Task 3 explicitly forbids widening that
// allowlist from inside this fixture; it did not need widening).
const fixtureRenditionMIME = "image/png"

// fixtureRenditionPNG is a tiny (well under 1KB) square PNG — any small
// square image is fine, since Fix 9's aspect-lock is what is under test,
// not the image's own dimensions.
//
//go:embed assets/fixture-rendition.png
var fixtureRenditionPNG []byte

// renditionFixtureEnabled reports whether renditionFixtureEnvVar is set to
// a non-empty, non-"0" value, read via getenv (a parameter, never a direct
// os.Getenv call, so this is unit-testable with no process env mutation) —
// mirroring readiness.go's readinessWindowFromEnv/launchDelayFromEnv
// contract exactly. Absent, empty, or "0" means off — the plugin's Fetch
// behaviour is then byte-identical to a build with no fixture at all.
func renditionFixtureEnabled(getenv func(string) string) bool {
	raw := getenv(renditionFixtureEnvVar)
	return raw != "" && raw != "0"
}
