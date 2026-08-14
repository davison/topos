// Command topos-manifest is the build-time generator for the kernel's
// link-time trust manifest (D-12, PD-03): it hashes an EXPLICIT list of
// plugin binary paths supplied as command-line arguments and prints
// FormatManifest's comma-separated "name=hexdigest" spec to stdout and
// NOTHING else — the exact string a build recipe passes to
//
//	go build -ldflags "-X github.com/davison/topos/kernel/pluginhost.buildManifest=<spec>"
//
// It deliberately takes an explicit argument list rather than scanning a
// directory: a leftover binary sitting in bin/plugins/ from an earlier
// `make plugins` run must never silently enter a `make build-portable`
// manifest (RESEARCH Pitfall 6) — the caller's own binary list is the only
// authority over what gets hashed.
//
// Called with zero arguments, it refuses to run and exits non-zero: a
// silent empty manifest from a mis-wired build recipe is exactly the loud
// failure this tool exists to guarantee never happens quietly.
package main

import (
	"fmt"
	"os"

	"github.com/davison/topos/kernel/pluginhost"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "topos-manifest: refusing to run with zero binary arguments — a mis-wired build recipe must fail loudly, not silently produce an empty manifest (RESEARCH Pitfall 6)")
		fmt.Fprintln(os.Stderr, "usage: topos-manifest <plugin-binary-path> [<plugin-binary-path> ...]")
		os.Exit(1)
	}

	entries, err := pluginhost.ManifestEntriesForBinaries(os.Args[1:]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "topos-manifest: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(pluginhost.FormatManifest(entries))
}
