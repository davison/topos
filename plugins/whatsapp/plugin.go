package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

const (
	sourceType      = "whatsapp"
	displayName     = "WhatsApp"
	contractVersion = "topos.v2"

	// pluginName identifies this plugin in Item.Provenance's "plugin" key
	// and in this process's own log lines.
	pluginName = "topos-plugin-whatsapp"
)

// matchVocabulary is the field-name vocabulary this plugin declares and
// reads from MatchRequest.match_fields — only "groups" in this plan
// (D-05's widening to a second "contacts" field for 1:1 chats is Plan
// 08-02's work, deliberately absent here).
var matchVocabulary = []string{"groups"}

// noThumbnailReason is the fixed unavailable_reason for the THUMBNAIL
// content variant — a WhatsApp digest has no image rendition, ever.
const noThumbnailReason = "WhatsApp digests have no thumbnail rendition"

// transcriptMimeType is what Fetch's FULL/PREVIEW branch returns — a
// self-contained, sanitized-at-the-kernel HTML fragment, routed by
// CONTENT_SHAPE_CHAT_TRANSCRIPT into DetailPane's existing chat rendering
// with zero new proto change and zero new frontend branch.
const transcriptMimeType = "text/html"

// SourcePlugin implements sdk.SourcePlugin over a live whatsmeow linked-
// device connection PLUS this plugin's own local message store
// (messagestore.go). Unlike every other plugin in this repo (Signal,
// Proton, paperless, SilverBullet — all open-and-close per RPC call), this
// plugin holds both a persistent whatsmeow connection AND an always-open
// *sql.DB across calls: the connection is the only way to ever capture a
// new message at all (WhatsApp has no "poll for new messages" API this
// plugin's Match could otherwise call), and Match/Fetch read the local
// store fresh every call, never a live whatsmeow call — connect.go and
// eventhandler.go own the write side; plugin.go's Match/Fetch own the read
// side.
type SourcePlugin struct {
	// dir is this plugin's own data directory (the [sources.whatsapp].path
	// config value, "~" already expanded by main.go) — home to BOTH
	// whatsmeow's own session store (whatsmeow.db) and this plugin's own
	// message-content store (messages.db).
	dir string

	// logOut is the plugin's log sink — os.Stderr in production.
	// Overridable in tests.
	logOut io.Writer

	container *sqlstore.Container
	client    *whatsmeow.Client
	store     *messageStore
	lock      *storeLock

	// pushNames is a best-effort JID->push-name cache (pushnames.go),
	// sourced from whatsmeow's own HistorySync.Pushnames list — a
	// fallback for message sender_name when a captured message's own
	// Info.PushName is empty, which the real-device spike (2026-08-10)
	// found true for nearly every history-sync-replayed message.
	// Display-only; never a match candidate.
	pushNames *pushNameCache

	// mu guards state/detail: the background whatsmeow event-handler
	// goroutine (eventhandler.go) writes them, while every gRPC handler
	// goroutine (Match/Health, below) reads them — health.go's named
	// healthState taxonomy replaces this plan's own predecessor's single
	// non-healthy flag.
	mu    sync.RWMutex
	state healthState
	// detail is optional dynamic context appended to state.Message()'s
	// own fixed template (currentMessageLocked) — e.g. a TemporaryBan
	// event's own reported reason code/text, or a raw connect error.
	// Never a substitute for the template: health_test.go's uniqueness
	// assertion exercises state.Message() alone, undiluted by detail.
	detail string
}

// NewSourcePlugin opens this plugin's own message store, then starts the
// background whatsmeow client (connect.go). dir must be non-empty and
// already have any leading "~" expanded — main.go fails startup loudly
// otherwise. A failure to open the LOCAL message store, or to acquire the
// store lock, or to open whatsmeow's own session store, is fatal (returned
// here, main.go exits non-zero) — these are local-filesystem
// preconditions this plugin cannot serve without. A failure to CONNECT
// (a live-network issue) is deliberately NOT fatal — see
// startBackgroundClient's own doc comment.
func NewSourcePlugin(ctx context.Context, dir string) (*SourcePlugin, error) {
	store, err := openMessageStore(dir)
	if err != nil {
		return nil, err
	}

	p := &SourcePlugin{dir: dir, logOut: os.Stderr, store: store, pushNames: newPushNameCache()}
	if err := p.startBackgroundClient(ctx); err != nil {
		store.Close()
		return nil, err
	}
	return p, nil
}

// setHealthState records state (and optional dynamic detail) as this
// plugin's current health — the ONLY place that mutates p.state/p.detail.
// Called from the background whatsmeow event-handler goroutine
// (eventhandler.go) and from connect.go's own boot-time branches; never
// from a gRPC handler goroutine (Match/Health only ever READ via
// healthState()/currentMessage()).
func (p *SourcePlugin) setHealthState(state healthState, detail string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
	p.detail = detail
	if !state.Healthy() {
		fmt.Fprintf(p.logOut, "%s: %s\n", pluginName, p.currentMessageLocked())
	}
}

// healthState returns this plugin's current named health state.
func (p *SourcePlugin) healthState() healthState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// currentMessage returns the CURRENT state's own fixed template (health.go)
// with any dynamic detail appended in parentheses — the actual text
// Match/Health emit. Never a substitute for the template: it always
// contains state.Message() verbatim as a prefix.
func (p *SourcePlugin) currentMessage() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentMessageLocked()
}

// currentMessageLocked is currentMessage's body, callable while p.mu is
// already held (setHealthState's own log line needs this without
// recursively re-acquiring the lock).
func (p *SourcePlugin) currentMessageLocked() string {
	msg := p.state.Message()
	if p.detail != "" {
		msg += " (" + p.detail + ")"
	}
	return msg
}

func (p *SourcePlugin) Describe(_ context.Context, _ *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error) {
	return &toposv1.DescribeResponse{
		SourceType:      sourceType,
		DisplayName:     displayName,
		ContractVersion: contractVersion,
		MatchVocabulary: matchVocabulary,
	}, nil
}

// Match reads healthState() FIRST — any state whose Healthy() is false
// returns a gRPC error, NEVER an empty success, even when the request
// carries zero keywords (the zero-keywords early return sits BELOW this
// guard, deliberately — a de-linked plugin asked with no keywords must
// still surface the error rather than a silent success). Only once the
// plugin is confirmed healthy does Match resolve keywords against this
// plugin's own local chats table (D-05/D-06-equivalent for this plan's
// groups-only scope, match.go), then group the matched chats' FULL
// message history into chat-day digests (digest.go). A HEALTHY plugin
// whose store genuinely holds no matching chat still returns a successful
// EMPTY response — the two outcomes kernel/correlate/correlate.go treats
// oppositely (wipes previously-synced rows on empty success, preserves
// them on error — 08-PATTERNS.md's single most load-bearing pattern,
// T-08-05's mitigation) must stay distinguishable at every call site.
func (p *SourcePlugin) Match(_ context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	state := p.healthState()
	if !state.Healthy() {
		return nil, status.Errorf(codes.Unavailable, "whatsapp: %s", p.currentMessage())
	}

	keywords := req.GetMatchFields()["groups"].GetValues()
	if len(keywords) == 0 {
		return &toposv1.MatchResponse{}, nil
	}

	chats, err := p.store.Chats()
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "whatsapp: %v", err)
	}

	matched := eligibleChats(chats, keywords)
	if len(matched) == 0 {
		return &toposv1.MatchResponse{}, nil
	}

	chatJIDs := make([]string, 0, len(matched))
	names := make(map[string]string, len(matched))
	for _, c := range matched {
		chatJIDs = append(chatJIDs, c.ChatJID)
		names[c.ChatJID] = c.Name
	}

	msgs, err := p.store.MessagesForChats(chatJIDs)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "whatsapp: %v", err)
	}

	digests := buildDigests(msgs, names)

	items := make([]*toposv1.Item, 0, len(digests))
	for _, d := range digests {
		items = append(items, p.toItem(d))
	}

	// Count-only: never a chat name, sender name or message body. This
	// log is forwarded verbatim into the kernel's log stream.
	fmt.Fprintf(p.logOut, "%s: match: %d matched chat(s), %d digest(s)\n", pluginName, len(matched), len(items))

	return &toposv1.MatchResponse{Items: items}, nil
}

// toItem builds a toposv1.Item from one digest. Only a MATCHED chat's
// digests ever reach here — a captured message from a chat matching
// nothing never becomes an Item (this plugin's own store necessarily
// captures every inbound message the linked device receives, but capture
// must never become exposure — this plan's threat_model prohibition).
func (p *SourcePlugin) toItem(d digest) *toposv1.Item {
	sourceID := sourceIDForDigest(d.ChatJID, d.Day)
	return &toposv1.Item{
		SourceId:      sourceID,
		SourceType:    sourceType,
		Title:         digestTitle(d.ChatName, d.MessageCount),
		Preview:       d.Preview,
		TimestampUnix: d.LastMessageUnix,
		GroupId:       d.ChatJID,
		GroupLabel:    "", // the title already carries the identifying context
		Fidelity:      toposv1.LinkFidelity_LINK_FIDELITY_CONVERSATION_ONLY,
		// isGroup is always true here — eligibleChats (match.go) filters
		// to groups only in this plan's scope; Plan 08-02 must thread
		// the real per-chat IsGroup through once 1:1 matching exists.
		DeepLink:     conversationDeepLink(true, d.ChatJID),
		Labels:       []string{d.ChatName},
		HasThumbnail: false,
		Provenance: map[string]string{
			"source_type":      sourceType,
			"source_system":    p.dir,
			"source_id":        sourceID,
			"plugin":           pluginName,
			"contract_version": contractVersion,
		},
	}
}

// Fetch implements live content fetch on item-open — never called from
// Match/sync. THUMBNAIL is always unavailable. FULL and PREVIEW share one
// path (fetchTranscript): a WhatsApp digest has nothing FULL offers beyond
// what PREVIEW already renders.
func (p *SourcePlugin) Fetch(_ context.Context, req *toposv1.FetchRequest) (*toposv1.FetchResponse, error) {
	switch req.GetVariant() {
	case toposv1.ContentVariant_CONTENT_VARIANT_THUMBNAIL:
		return &toposv1.FetchResponse{Available: false, UnavailableReason: noThumbnailReason}, nil
	case toposv1.ContentVariant_CONTENT_VARIANT_FULL, toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW:
		return p.fetchTranscript(req.GetSourceId())
	default:
		return nil, status.Error(codes.InvalidArgument, "whatsapp: unspecified content variant")
	}
}

// fetchTranscript re-derives sourceID's (chatJID, day) pair, re-reads that
// chat's messages from this plugin's own local store (never a live
// whatsmeow call — content is fetched live on open, from the store this
// plugin already owns), filters to the requested day, renders them into a
// sanitized-at-the-kernel HTML transcript fragment (render.go), and
// returns it as MimeType transcriptMimeType.
func (p *SourcePlugin) fetchTranscript(sourceID string) (*toposv1.FetchResponse, error) {
	chatJID, day, err := decodeSourceID(sourceID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "whatsapp: source_id %q is not a recognised digest id: %v", sourceID, err)
	}

	msgs, err := p.store.MessagesForChats([]string{chatJID})
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "whatsapp: %v", err)
	}

	var dayMsgs []messageRecord
	for _, m := range msgs {
		if localDayKey(m.SentAtUnixMs) == day {
			dayMsgs = append(dayMsgs, m)
		}
	}
	if len(dayMsgs) == 0 {
		return nil, status.Errorf(codes.NotFound, "whatsapp: no messages found for chat %q on day %q (source_id %q)", chatJID, day, sourceID)
	}
	sort.Slice(dayMsgs, func(i, j int) bool {
		if dayMsgs[i].SentAtUnixMs != dayMsgs[j].SentAtUnixMs {
			return dayMsgs[i].SentAtUnixMs < dayMsgs[j].SentAtUnixMs
		}
		return dayMsgs[i].ID < dayMsgs[j].ID
	})

	// renderTranscript returns an UNSANITIZED, UNWRAPPED fragment —
	// sanitization, wrapping and theming happen at the kernel's rendition
	// boundary (kernel/httpapi/rendition.go). ContentShape declares the
	// chat profile so the kernel applies the right policy.
	fragment := renderTranscript(dayMsgs)

	return &toposv1.FetchResponse{
		Available:    true,
		MimeType:     transcriptMimeType,
		ContentShape: toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT,
		SizeBytes:    int64(len(fragment)),
		Data:         fragment,
		Provenance: map[string]string{
			"source_type": sourceType,
			"source_id":   sourceID,
		},
	}, nil
}

// Health reports reachable when linked and connected, and reachable false
// with a specific, actionable, per-cause last_error otherwise (health.go's
// named healthState taxonomy — five distinct non-healthy causes, five
// distinct messages, never one generic "unavailable"). Never includes any
// message content or session key material.
func (p *SourcePlugin) Health(_ context.Context, _ *toposv1.HealthRequest) (*toposv1.HealthResponse, error) {
	state := p.healthState()
	if !state.Healthy() {
		return &toposv1.HealthResponse{Reachable: false, LastError: p.currentMessage()}, nil
	}
	return &toposv1.HealthResponse{
		Reachable:    true,
		LastSyncUnix: time.Now().Unix(),
	}, nil
}
