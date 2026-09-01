package supervisor

import (
	"github.com/davison/topos/kernel/httpapi"
)

// The serve path hands the Supervisor to httpapi.Router as its Fetcher;
// SearchHandler then type-asserts it to httpapi.Searcher for the
// scope=all fan-out. This compile-time assertion is the regression pin
// for that assertion succeeding — if the Supervisor ever stops
// satisfying Searcher, scope=all silently becomes index-only again
// (M2-R2, #54).
var _ httpapi.Searcher = (*Supervisor)(nil)
