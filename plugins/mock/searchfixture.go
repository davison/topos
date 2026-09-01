package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// searchDelayEnvVar is the fixture-only knob behind the e2e suite's
// "a slow source never blocks the fast ones" proof (M2-R2, davison/topos
// #54): a non-negative integer number of milliseconds the mock sleeps
// inside Search before answering. Off by default (absent, empty or "0"),
// so a real installation's mock is unaffected; a malformed value fails
// startup loudly, exactly as launchDelayFromEnv (readiness.go) does. It
// reaches one mock INSTANCE the same way every WEBSPACES_MOCK_* fixture
// var does: the spec's source config references
// ${WEBSPACES_MOCK_SEARCH_DELAY_MS} from its extras block, which puts the
// variable in that subprocess's environment and no other's.
const searchDelayEnvVar = "WEBSPACES_MOCK_SEARCH_DELAY_MS"

func searchDelayFromEnv(getenv func(string) string) (time.Duration, error) {
	raw := getenv(searchDelayEnvVar)
	if raw == "" || raw == "0" {
		return 0, nil
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms < 0 {
		return 0, fmt.Errorf("%s: invalid value %q: must be a non-negative integer number of milliseconds", searchDelayEnvVar, raw)
	}
	return time.Duration(ms) * time.Millisecond, nil
}

// withSearchDelay sets the fixture-only Search delay and returns p for
// chaining from main.go. Not part of the plugin contract.
func (p *SourcePlugin) withSearchDelay(d time.Duration) *SourcePlugin {
	p.searchDelay = d
	return p
}

// waitSearchDelay sleeps for the fixture delay, or until the kernel gives
// up on this source (its SearchBudget context) — whichever comes first —
// so a delayed mock never lingers past the fan-out that asked.
func (p *SourcePlugin) waitSearchDelay(ctx context.Context) error {
	if p.searchDelay <= 0 {
		return nil
	}
	select {
	case <-time.After(p.searchDelay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// searchOnlyItems are items the source can FIND but never lists through
// Match — the shape of a body hit whose item the kernel has not (yet)
// synced, which the fan-out returns with indexed:false and the app
// renders from the plugin's own Item fields, marked as such (davison/
// topos#50). Members of the demo label like the rest, so a webspace
// keyed on it reaches them.
var searchOnlyItems = []*toposv1.Item{
	{
		SourceId:      "5",
		SourceType:    sourceType,
		Title:         "A note beyond the index",
		Preview:       "A note the mock source can find but never lists — it exists to be a hit the index has not seen.",
		TimestampUnix: 1704412800, // 2024-01-05T00:00:00Z
		Fidelity:      toposv1.LinkFidelity_LINK_FIDELITY_EXACT,
		DeepLink:      "http://localhost/mock/notes/5",
		Labels:        []string{"demo"},
	},
}

// searchOnlyFullText is the search-only items' body text — the terms
// that reach them; deliberately shares no word with mockFullText.
var searchOnlyFullText = map[string]string{
	"5": "Only the source itself can find this orphaned note; nothing about it was ever synced.",
}
