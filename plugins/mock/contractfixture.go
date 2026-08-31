package main

import "fmt"

// contractVersionEnvVar names the env var that overrides the contract
// generation this plugin's Describe declares. Fixture-only, exactly like
// WEBSPACES_MOCK_READY_AFTER_MS (readiness.go): off by default, so a
// real installation's mock is unaffected, and it exists so the kernel's
// contract-generation launch gate (M1-R6/DIST-03, kernel/pluginhost's
// contract check) can be driven end to end by the hermetic browser
// harness and the Go suite — a plugin that DECLARES an unsupported
// generation, launched for real, refused for real. It reaches this
// subprocess the same way every WEBSPACES_MOCK_* fixture var does: the
// spec's source config references ${WEBSPACES_MOCK_CONTRACT_VERSION},
// which puts the name in config.EnvRefNames' per-instance allowlist
// (kernel/pluginhost/host.go's allowedEnv) — the kernel never passes its
// environment through wholesale.
const contractVersionEnvVar = "WEBSPACES_MOCK_CONTRACT_VERSION"

// contractVersionFromEnv resolves the contract generation Describe
// declares, read via getenv (a parameter, not a direct os.Getenv call,
// so this is unit-testable with no process env mutation). Absent or
// empty returns the sdk's own generation — byte-identical to the
// plugin's pre-fixture behaviour. Any non-empty value is returned
// VERBATIM, including values the kernel will refuse: producing exactly
// that refusal is this fixture's whole purpose. Never an error — there
// is no malformed value, only a declared one.
func contractVersionFromEnv(getenv func(string) string) string {
	if v := getenv(contractVersionEnvVar); v != "" {
		return v
	}
	return contractVersion
}

// String-typed sanity hook kept alongside the fixture: fmt is imported
// for the error paths of sibling fixtures; keep the linter honest here.
var _ = fmt.Sprintf
