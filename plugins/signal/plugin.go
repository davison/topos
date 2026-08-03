package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)

const (
	sourceType      = "signal"
	displayName     = "Signal"
	contractVersion = "webspaces.v1"

	// pluginName identifies this plugin in Item.Provenance's "plugin" key
	// and in this process's own log lines.
	pluginName = "webspaces-plugin-signal"
)

// noThumbnailReason is the fixed unavailable_reason for the THUMBNAIL
// content variant — a Signal digest has no image rendition, ever.
const noThumbnailReason = "Signal digests have no thumbnail rendition"

// fetchNotImplementedReason is the fixed unavailable_reason FULL/PREVIEW
// Fetch returns in this build. Completed in plan 04-03 (04-01-PLAN.md
// Task 2's own scope note): the Fetch entry point, its variant dispatch,
// and this stub all already exist here, so this is a functionality gap,
// not an architectural one.
const fetchNotImplementedReason = "Signal thread rendering is not yet implemented in this build (see plan 04-03)"

// SourcePlugin implements sdk.SourcePlugin by reading Signal Desktop's
// local SQLCipher database strictly read-only. plugins/proton/plugin.go's
// SourcePlugin is this file's closest analog (04-PATTERNS.md), but unlike
// Proton this plugin caches nothing across calls: Match re-derives
// everything from the database fresh every time (no long-lived
// mailbox-style cache), because the database itself — not a remote
// server round trip — is the only "connection" this plugin ever holds,
// and holding it open across calls would work against the byte-identical
// / live-writer safety goals this phase's success criteria centre on.
type SourcePlugin struct {
	// configDir is Signal Desktop's own config directory ("~" already
	// expanded by main.go) — the source of both config.json (key
	// resolution) and sql/db.sqlite (the message database itself).
	configDir string

	// logOut is the plugin's log sink — os.Stderr in production, parsed
	// and re-emitted through the kernel's hclog so plugin and kernel logs
	// interleave sanely (plugins/proton/plugin.go's identical field).
	// Overridable in tests.
	logOut io.Writer
}

// NewSourcePlugin builds a SourcePlugin. configDir must be non-empty and
// already have any leading "~" expanded — main.go fails startup loudly
// otherwise.
func NewSourcePlugin(configDir string) *SourcePlugin {
	return &SourcePlugin{configDir: configDir, logOut: os.Stderr}
}

func (p *SourcePlugin) configPath() string { return filepath.Join(p.configDir, "config.json") }
func (p *SourcePlugin) dbPath() string     { return filepath.Join(p.configDir, "sql", "db.sqlite") }

func (p *SourcePlugin) Describe(_ context.Context, _ *webspacesv1.DescribeRequest) (*webspacesv1.DescribeResponse, error) {
	return &webspacesv1.DescribeResponse{
		SourceType:      sourceType,
		DisplayName:     displayName,
		ContractVersion: contractVersion,
	}, nil
}

// openGuarded resolves the key, opens db.sqlite read-only, and guards the
// schema version ceiling — the three preconditions every RPC that touches
// the database needs, in the order 04-RESEARCH.md Pattern 2 requires (the
// schema guard runs BEFORE any messages/conversations query). Callers
// must Close() the returned *sql.DB.
func (p *SourcePlugin) openGuarded() (*sql.DB, error) {
	cfg, err := readSignalConfig(p.configPath())
	if err != nil {
		return nil, err
	}
	rawHexKey, err := resolveKey(cfg)
	if err != nil {
		return nil, err
	}
	db, err := openReadOnly(p.dbPath(), rawHexKey)
	if err != nil {
		return nil, err
	}
	if err := guardSchemaVersion(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// Match resolves keywords against Signal's own conversations (D-05/D-06,
// match.go), then groups the matched conversations' FULL message history
// — no time window (D-08) — into conversation-day digests (D-01/D-02/
// D-04, digest.go). A zero-length keyword list, and zero matched
// conversations, both return a successful EMPTY response — never an
// error (plugins/proton's identical precedent: a webspace with no
// matching content is empty, not failed).
func (p *SourcePlugin) Match(_ context.Context, req *webspacesv1.MatchRequest) (*webspacesv1.MatchResponse, error) {
	keywords := req.GetKeywords()
	if len(keywords) == 0 {
		return &webspacesv1.MatchResponse{}, nil
	}

	db, err := p.openGuarded()
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "signal: %v", err)
	}
	defer db.Close()

	ownAci, err := readOwnAci(db)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "signal: %v", err)
	}

	convs, err := readConversations(db, ownAci)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "signal: %v", err)
	}

	matched := eligibleConversations(convs, keywords)
	if len(matched) == 0 {
		return &webspacesv1.MatchResponse{}, nil
	}

	matchedByID := make(map[string]conversation, len(matched))
	convIDs := make([]string, 0, len(matched))
	names := make(map[string]string, len(matched))
	for _, c := range matched {
		matchedByID[c.ID] = c
		convIDs = append(convIDs, c.ID)
		names[c.ID] = conversationDisplayName(c)
	}

	// senderNames resolves any sender (not just the matched conversations'
	// own contacts — a group can carry messages from any member Signal
	// Desktop knows) by service id, built from the FULL conversation set
	// this Match already read, never a second query.
	senderNames := buildSenderNames(convs, ownAci)

	msgs, err := readMessages(db, convIDs, senderNames)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "signal: %v", err)
	}

	digests := buildDigests(msgs, names)

	items := make([]*webspacesv1.Item, 0, len(digests))
	for _, d := range digests {
		items = append(items, p.toItem(d, matchedByID[d.ConversationID]))
	}

	// Count-only: never a conversation name, sender name or message
	// body. This log is forwarded verbatim into the kernel's log stream.
	fmt.Fprintf(p.logOut, "%s: match: %d matched conversation(s), %d digest(s)\n", pluginName, len(matched), len(items))

	return &webspacesv1.MatchResponse{Items: items}, nil
}

// toItem builds a webspacesv1.Item from one digest and the (already
// matched) conversation it belongs to.
func (p *SourcePlugin) toItem(d digest, conv conversation) *webspacesv1.Item {
	sourceID := sourceIDForDigest(d.ConversationID, d.Day)
	return &webspacesv1.Item{
		SourceId:      sourceID,
		SourceType:    sourceType,
		Title:         digestTitle(d.ConversationName, d.MessageCount),
		Preview:       d.Preview,
		TimestampUnix: d.LastMessageUnix,
		GroupId:       d.ConversationID,
		GroupLabel:    "", // 04-UI-SPEC.md: left empty — the title already carries the identifying context
		Fidelity:      webspacesv1.LinkFidelity_LINK_FIDELITY_CONVERSATION_ONLY,
		DeepLink:      conversationDeepLink(conv.Type, conv.E164),
		Labels:        []string{d.ConversationName},
		HasThumbnail:  false,
		Provenance: map[string]string{
			"source_type":      sourceType,
			"source_system":    p.configDir,
			"source_id":        sourceID,
			"plugin":           pluginName,
			"contract_version": contractVersion,
		},
	}
}

// Fetch implements live content fetch on item-open — never called from
// Match/sync. THUMBNAIL is always unavailable (a Signal digest has no
// image rendition). FULL/PREVIEW are stubbed unavailable in this build —
// completed in plan 04-03 (see fetchNotImplementedReason's doc comment).
func (p *SourcePlugin) Fetch(_ context.Context, req *webspacesv1.FetchRequest) (*webspacesv1.FetchResponse, error) {
	switch req.GetVariant() {
	case webspacesv1.ContentVariant_CONTENT_VARIANT_THUMBNAIL:
		return &webspacesv1.FetchResponse{Available: false, UnavailableReason: noThumbnailReason}, nil
	case webspacesv1.ContentVariant_CONTENT_VARIANT_FULL, webspacesv1.ContentVariant_CONTENT_VARIANT_PREVIEW:
		return &webspacesv1.FetchResponse{Available: false, UnavailableReason: fetchNotImplementedReason}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "signal: unspecified content variant")
	}
}

// Health attempts key resolution, a read-only open, and the schema guard
// — this plugin's equivalent of "reachability" is "can we open the
// database at all", not a network dial. Any failure returns
// Reachable:false with a specific, actionable LastError naming the
// failing step — never a gRPC error, matching every other plugin. Never
// includes the key or any config.json field value.
func (p *SourcePlugin) Health(_ context.Context, _ *webspacesv1.HealthRequest) (*webspacesv1.HealthResponse, error) {
	db, err := p.openGuarded()
	if err != nil {
		return &webspacesv1.HealthResponse{Reachable: false, LastError: err.Error()}, nil
	}
	defer db.Close()

	return &webspacesv1.HealthResponse{
		Reachable:    true,
		LastSyncUnix: time.Now().Unix(),
	}, nil
}

// --- Database row reading (this plugin's own SQL layer; no separate
// client file exists yet — plugins/proton/plugin.go's listMailboxes is
// the closest analog for "row-reading helpers living alongside Match in
// plugin.go rather than a dedicated client file"). ---

// conversationFields is the subset of a conversations.json blob this
// plugin needs beyond what conversations' own SQL columns already carry
// (id, type, name, profileName, profileFamilyName, e164, serviceId are
// all real columns — see readConversations' SELECT — so only the four
// system/nickname fields below, which have no SQL column of their own,
// need a JSON unmarshal).
type conversationFields struct {
	SystemGivenName    string `json:"systemGivenName"`
	SystemFamilyName   string `json:"systemFamilyName"`
	NicknameGivenName  string `json:"nicknameGivenName"`
	NicknameFamilyName string `json:"nicknameFamilyName"`
}

// readConversations reads every group and 1:1 conversation row, resolving
// each row's system/nickname name fields from its JSON blob (see
// conversationFields) and marking IsNoteToSelf by comparing serviceId
// against ownAci (empty ownAci — an unlinked install — marks nothing).
func readConversations(db *sql.DB, ownAci string) ([]conversation, error) {
	rows, err := db.Query(`
		SELECT id, type, name, profileName, profileFamilyName, e164, serviceId, json
		FROM conversations
		WHERE type IN ('private', 'group')
	`)
	if err != nil {
		return nil, fmt.Errorf("query conversations: %w", err)
	}
	defer rows.Close()

	var out []conversation
	for rows.Next() {
		var id, typ string
		var name, profileName, profileFamilyName, e164, serviceID sql.NullString
		var rawJSON string
		if err := rows.Scan(&id, &typ, &name, &profileName, &profileFamilyName, &e164, &serviceID, &rawJSON); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}

		var fields conversationFields
		if err := json.Unmarshal([]byte(rawJSON), &fields); err != nil {
			return nil, fmt.Errorf("parse conversation record: %w", err)
		}

		out = append(out, conversation{
			ID:                 id,
			Type:               typ,
			Name:               name.String,
			SystemGivenName:    fields.SystemGivenName,
			SystemFamilyName:   fields.SystemFamilyName,
			NicknameGivenName:  fields.NicknameGivenName,
			NicknameFamilyName: fields.NicknameFamilyName,
			ProfileName:        profileName.String,
			ProfileFamilyName:  profileFamilyName.String,
			E164:               e164.String,
			ServiceID:          serviceID.String,
			IsNoteToSelf:       ownAci != "" && serviceID.String == ownAci,
		})
	}
	return out, rows.Err()
}

// accountIdentityItem is the shape of the items table's "value" column
// (itself a JSON blob) for the "uuid_id" row: Signal Desktop's own
// persisted "<aci>.<deviceId>" account identity string.
type accountIdentityItem struct {
	Value string `json:"value"`
}

// readOwnAci reads the account's own ACI (Signal's per-account stable
// identifier) from the items table's "uuid_id" row, stripping the
// trailing ".<deviceId>" suffix Signal Desktop stores it with — so it
// compares equal to a conversation's own bare serviceId column. Returns
// ("", nil) for a fresh, never-linked install (no items row yet): Note to
// Self detection then simply excludes nothing, which is safe because an
// unlinked install also has zero real conversations.
func readOwnAci(db *sql.DB) (string, error) {
	var rawJSON string
	err := db.QueryRow(`SELECT json FROM items WHERE id = 'uuid_id'`).Scan(&rawJSON)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read account identity: %w", err)
	}

	var item accountIdentityItem
	if err := json.Unmarshal([]byte(rawJSON), &item); err != nil {
		return "", fmt.Errorf("parse account identity: %w", err)
	}
	aci, _, _ := strings.Cut(item.Value, ".")
	return aci, nil
}

// buildSenderNames maps every conversation's own service id to the best
// available DISPLAY name for messages it sends — display purposes only,
// never matching (D-06 restricts matching alone; see
// conversationDisplayName's own doc comment). The account's own service
// id maps to the fixed "You" label rather than its own conversation
// display name, matching this transcript's own outgoing-message
// convention.
func buildSenderNames(convs []conversation, ownAci string) map[string]string {
	out := make(map[string]string, len(convs)+1)
	for _, c := range convs {
		if c.Type != "private" || c.ServiceID == "" {
			continue
		}
		out[c.ServiceID] = conversationDisplayName(c)
	}
	if ownAci != "" {
		out[ownAci] = "You"
	}
	return out
}

// unknownSenderName is the fallback digest.go's tail snippet uses when a
// message's sourceServiceId has no entry in senderNames (a sender Signal
// Desktop has no conversation record for at all — rare, but not
// impossible for an old/orphaned message).
const unknownSenderName = "Unknown"

// readMessages reads every real chat message (type IN
// ('incoming','outgoing') — excluding system/notification event rows
// such as 'profile-change', 'group-v2-change', 'call-history', which are
// not messages a day's "N messages" count should include) for every
// conversation in conversationIDs, with NO time window (D-08: full
// history backfill), resolving each row's sender display name via
// senderNames (buildSenderNames' output).
//
// sourceServiceId is empty for an OUTGOING message in a 1:1 (private)
// conversation — Signal Desktop leaves the sender implicit there (the
// conversationId itself already identifies the pairing), unlike a GROUP
// conversation's own outgoing rows, which DO carry the account's own
// service id (confirmed against this task's real, live db.sqlite for
// both conversation shapes). readMessages therefore falls back to the
// fixed "You" label for any outgoing row with no sourceServiceId, rather
// than misreporting it as unknownSenderName.
func readMessages(db *sql.DB, conversationIDs []string, senderNames map[string]string) ([]message, error) {
	if len(conversationIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(conversationIDs))
	args := make([]any, 0, len(conversationIDs))
	for i, id := range conversationIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		SELECT conversationId, sent_at, type, sourceServiceId, body
		FROM messages
		WHERE conversationId IN (%s)
		  AND type IN ('incoming', 'outgoing')
		  AND sent_at IS NOT NULL
		ORDER BY sent_at ASC
	`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var out []message
	for rows.Next() {
		var conversationID, msgType string
		var sentAt int64
		var sourceServiceID, body sql.NullString
		if err := rows.Scan(&conversationID, &sentAt, &msgType, &sourceServiceID, &body); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}

		senderName := senderNames[sourceServiceID.String]
		switch {
		case senderName != "":
			// resolved via senderNames.
		case sourceServiceID.String == "" && msgType == "outgoing":
			senderName = "You"
		default:
			senderName = unknownSenderName
		}

		out = append(out, message{
			ConversationID: conversationID,
			SentAtUnixMs:   sentAt,
			SenderName:     senderName,
			Body:           body.String,
		})
	}
	return out, rows.Err()
}
