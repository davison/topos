// This file holds a TEST FIXTURE mechanism and is NOT part of the plugin
// contract — no real plugin needs it, and docs/plugin-contract.md is
// deliberately not amended to mention it. It exists because 08-UAT.md gap
// G-08-4 was a kernel launch-readiness defect (no gate between launching a
// plugin subprocess and issuing its first Match) that no fixture in this
// repo could express: this plugin's Match was, until now, unconditionally
// ready, so its own readiness window was exactly zero — and every hermetic
// gate in Phase 8 shared that blind spot, since plugins/mock is the only
// plugin any hermetic test drives against a real subprocess. It is kept in
// its own file, separate from plugin.go, so plugin.go stays readable as the
// contract documentation PLUG-05 built it to be.
package main

import (
	"fmt"
	"strconv"
	"time"
)

// readyAfterEnvVar names the env var that opts this plugin into the
// launch-readiness window. The "WEBSPACES_" prefix matches the two env
// vars the kernel already sets on every plugin subprocess
// (WEBSPACES_SOURCE_CONFIG, WEBSPACES_DESCRIBE_ONLY — see
// kernel/pluginhost/host.go launch()), which is also what makes an env var
// set on the KERNEL process reach this subprocess: pluginhost.launch
// builds the subprocess's environment as append(os.Environ(), ...), so the
// kernel's own environment (including this variable, when a test sets it
// via t.Setenv before booting a supervisor) is inherited whole.
const readyAfterEnvVar = "WEBSPACES_MOCK_READY_AFTER_MS"

// notReadyMessage is a fixed, distinctive string naming the fixture and
// the window it models, so a test asserting on it cannot be satisfied by
// some unrelated failure.
const notReadyMessage = "plugins/mock: WEBSPACES_MOCK_READY_AFTER_MS launch-readiness window not yet elapsed"

// readinessWindow models an opt-in delay between process start and the
// plugin actually being willing to serve Match/Health. A nil
// *readinessWindow reports ready at every instant — the zero-cost default
// for the normal, unconfigured case, so every call site can treat "no
// window configured" and "an already-elapsed window" identically.
type readinessWindow struct {
	readyAt time.Time
}

// ready reports whether the window has elapsed as of now. A nil receiver
// is always ready — this is what makes "no window configured" the
// zero-cost default at both of Match's and Health's call sites.
func (w *readinessWindow) ready(now time.Time) bool {
	if w == nil {
		return true
	}
	return !now.Before(w.readyAt)
}

// readinessWindowFromEnv parses readyAfterEnvVar, read via getenv (a
// parameter, not a direct os.Getenv call, so this is unit-testable with no
// process env mutation). Absent, empty, or "0" returns (nil, nil) — no
// window, byte-identical to the plugin's pre-fixture behaviour. A positive
// integer number of milliseconds returns a window ending that many
// milliseconds after now. Any other value (non-integer, negative) is a
// loud startup failure — never a silently ignored setting — reported as an
// error naming both the variable and the bad value.
func readinessWindowFromEnv(now time.Time, getenv func(string) string) (*readinessWindow, error) {
	raw := getenv(readyAfterEnvVar)
	if raw == "" || raw == "0" {
		return nil, nil
	}

	ms, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid value %q: must be a non-negative integer number of milliseconds: %w", readyAfterEnvVar, raw, err)
	}
	if ms < 0 {
		return nil, fmt.Errorf("%s: invalid value %q: must be a non-negative integer number of milliseconds", readyAfterEnvVar, raw)
	}

	return &readinessWindow{readyAt: now.Add(time.Duration(ms) * time.Millisecond)}, nil
}

// launchDelayEnvVar names the env var that opts this plugin into an
// artificial delay BEFORE goplugin.Serve is reached — distinct from
// readyAfterEnvVar above, which delays Match/Health readiness AFTER the
// go-plugin handshake has already completed. This one delays the
// handshake itself: it models a plugin that is slow to come up at all
// (the shape the whatsapp plugin's serve-mode login wait presented, and the
// shape any plugin can present, since go-plugin's own client
// StartTimeout default is a full minute). It exists so
// kernel/supervisor's cross-source isolation gate
// (kernel/supervisor/launchlatency_test.go,
// TestResume_SlowRelaunchDoesNotFreezeOtherSources) has a controllable
// slow launch to drive against a real subprocess. Kept in this file so
// the package's "TEST FIXTURE, not contract" header governs it too.
const launchDelayEnvVar = "WEBSPACES_MOCK_LAUNCH_DELAY_MS"

// launchDelayFromEnv parses launchDelayEnvVar, read via getenv (a
// parameter, never a direct os.Getenv call, so this is unit-testable with
// no process env mutation) — mirroring readinessWindowFromEnv's contract
// exactly. Absent, empty, or "0" returns a zero duration and a nil error
// (no delay, byte-identical to the plugin's pre-fixture behaviour). A
// positive base-10 integer returns that many milliseconds. Any other
// value (non-integer, negative) is a loud startup failure — never a
// silently ignored setting — reported as an error naming both the
// variable and the bad value.
func launchDelayFromEnv(getenv func(string) string) (time.Duration, error) {
	raw := getenv(launchDelayEnvVar)
	if raw == "" || raw == "0" {
		return 0, nil
	}

	ms, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid value %q: must be a non-negative integer number of milliseconds: %w", launchDelayEnvVar, raw, err)
	}
	if ms < 0 {
		return 0, fmt.Errorf("%s: invalid value %q: must be a non-negative integer number of milliseconds", launchDelayEnvVar, raw)
	}

	return time.Duration(ms) * time.Millisecond, nil
}
