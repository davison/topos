// Package audit holds repo-wide static-analysis tests only — it has no
// runtime code of its own. It exists so `go build ./...`/`go test ./...`
// in the root module fail the moment a repo-wide invariant this package
// enforces stops being true, rather than that invariant resting on manual
// inspection (today: the outbound-host allowlist, PROJECT.md Constraints /
// 01-01-PLAN.md's third prohibition, closed by gap G-01-6).
//
// This file's package clause exists so `go build ./...` does not fail on a
// directory that would otherwise contain only *_test.go files — a
// directory with no non-test .go source is not a buildable package.
package audit
