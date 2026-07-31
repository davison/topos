module github.com/davison/webspaces/plugins/proton

go 1.25.0

require (
	github.com/emersion/go-imap v1.2.1
	github.com/emersion/go-message v0.18.2
)

// golang.org/x/text is pulled in transitively by github.com/emersion/go-message's
// charset handling. hashicorp/go-plugin and google.golang.org/grpc (both
// imported directly by main.go/plugin.go) and every other indirect
// dependency below are already required by the workspace-local
// github.com/davison/webspaces/sdk module (see go.work) and resolve via
// Go's workspace build list without needing a duplicate require here —
// `go mod tidy` cannot be run cleanly against this module in isolation
// because github.com/davison/webspaces/sdk has no published remote
// (mirrors plugins/silverbullet and plugins/mock's go.mod, which have
// the same limitation).
require golang.org/x/text v0.14.0 // indirect
