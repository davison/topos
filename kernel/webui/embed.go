// Package webui embeds the built SvelteKit SPA (web/, adapter-static
// output) into the kernel binary. kernel/httpapi serves it behind a
// catch-all for every non-/api/ path, falling back to 200.html for
// unmatched client-side routes.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:build
var buildFS embed.FS

// FS returns the embedded SPA files rooted at the build directory itself
// (i.e. FS().Open("200.html") reaches kernel/webui/build/200.html) — callers
// don't need to know the embed directive's internal directory name.
//
// build/.gitkeep is committed so `all:build` always matches at least one
// file and this package compiles on a clean checkout before any npm build
// has run; before that build, FS() serves an (almost) empty filesystem.
func FS() fs.FS {
	sub, err := fs.Sub(buildFS, "build")
	if err != nil {
		// build/ is committed (via .gitkeep) so this directory always
		// exists at compile time; fs.Sub over a real embedded directory
		// cannot fail.
		panic(err)
	}
	return sub
}
