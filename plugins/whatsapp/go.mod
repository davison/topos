module github.com/davison/topos/plugins/whatsapp

go 1.25.0

// go.mod require (Task 1 checkpoint, 08-01-PLAN.md, 2026-08-10): unlike
// every other third-party module this repo pins, go.mau.fi/whatsmeow has
// NEVER published a tagged release — every consumer, including this one,
// pins an exact commit pseudo-version with no changelog to review between
// bumps. 08-RESEARCH.md's Package Legitimacy Audit flagged this module SUS
// on that basis alone (not on any code-quality or maintenance-activity
// signal — the project is actively published and imported by 300+ Go
// modules, including the Mautrix WhatsApp bridge running in production for
// thousands of users) and required this exact checkpoint acknowledgment
// before pinning. Task 1's checkpoint returned "approved" for the pinned
// pseudo-version below, re-verified live against `go list -m -json
// go.mau.fi/whatsmeow@latest` on 2026-08-10 (identical to 08-RESEARCH.md's
// own snapshot) with the pinned commit's own go.mod dependency tree
// confirmed 100% cgo-free (closes 08-RESEARCH.md Assumption A4 for this
// exact commit). No `replace` directive is needed here, unlike
// plugins/signal/go.mod's SQLCipher fork situation — whatsmeow's own
// upstream has no missing-feature gap this plugin needs to route around.
// Any future bump of this line is a DELIBERATE, REVIEWED action — never a
// side effect of `go get -u` or `go mod tidy` — and should be preceded by
// re-running the same audit this comment records.
require go.mau.fi/whatsmeow v0.0.0-20260806224404-e277b766ab33

require (
	github.com/mdp/qrterminal/v3 v3.2.1
	modernc.org/sqlite v1.54.0
)
