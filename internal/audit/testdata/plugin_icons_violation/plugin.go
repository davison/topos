// Package main is a negative-control fixture for
// TestPluginIconsScanner_FixtureReportsAllOffenseKinds
// (internal/audit/plugin_icons_test.go) — never built as part of any real
// Go package (this directory is under testdata/, which every standard Go
// tool ignores for package discovery). It deliberately violates three of
// pluginIconOffenses's checks at once:
//
//  1. This go:embed directive has no preceding four-key provenance
//     comment block.
//  2. assets/icon.svg (sibling file) uses stroke="currentColor" instead
//     of a baked hex.
//  3. Describe's DescribeResponse literal sets Icon but never IconMime.
package main

import (
	_ "embed"
)

//go:embed assets/icon.svg
var iconSVG []byte

const iconMIME = "image/svg+xml"

// toposv1 stands in for the real sdk/gen/topos/v1 package this fixture
// deliberately does not import — parser.ParseFile (with
// SkipObjectResolution, exactly as pluginIconOffenses uses it) parses an
// unresolved qualified identifier like toposv1.DescribeResponse{} fine
// without needing the import to actually exist or resolve.
func Describe() *toposv1.DescribeResponse {
	return &toposv1.DescribeResponse{
		SourceType: "fixture",
		Icon:       iconSVG,
		// IconMime deliberately omitted.
	}
}
