package main

import (
	"fmt"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// attachmentPlaceholder is the fixed body text a message with no plaintext
// but a known non-text content type (image, video, audio, document,
// sticker, contact card, location) is captured with — never the file's
// own bytes, per the hybrid data model's "content stays in the source"
// rule.
const attachmentPlaceholder = "📎 Attachment"

// handleEvent is registered via Client.AddEventHandler (connect.go) and
// runs continuously, fully decoupled from when the kernel next calls
// Match — this is the background writer half of this plugin's
// architecture (T-08-02's mitigation: no send-capable Client method is
// ever called from here or anywhere else in this plugin).
func (p *SourcePlugin) handleEvent(evt any) {
	switch e := evt.(type) {
	case *events.Connected:
		p.setHealthState(healthStateLinked, "")
		fmt.Fprintf(p.logOut, "%s: connected\n", pluginName)
		// BLOCKING FIX (2026-08-10 real-device spike): history sync
		// alone never populates a group's own subject — see
		// groupsync.go's own doc comment. Runs in its own goroutine so
		// this IQ round trip never blocks whatsmeow's own event-dispatch
		// loop.
		go p.syncJoinedGroups()
	case *events.LoggedOut:
		// health.go's healthStateFromLogoutReason translates e.Reason
		// into the correct named cause (de-link / ban / session-expiry —
		// Task 1's own taxonomy) — never a single generic "logged out"
		// message the way 08-01's own predecessor code read.
		p.setHealthState(healthStateFromLogoutReason(e.Reason), "")
	case *events.TemporaryBan:
		// A dedicated event type (reason 402), distinct from the
		// LoggedOut/ConnectFailure family above — whatsmeow's own
		// TempBanReason.String() already composes a code+description
		// ("101: you sent too many messages..."), captured here as this
		// state's dynamic detail per this task's own action text.
		p.setHealthState(healthStateBanned, e.Code.String())
	case *events.ConnectFailure:
		// The truly-unrecognised-reason fallback whatsmeow's own
		// connectionevents.go dispatches when a connect failure is
		// neither a recognised logout, a temp ban, nor one of the
		// auto-retried transient codes — mapped to de-linked, never
		// silently to healthy (this task's own explicit requirement).
		p.setHealthState(healthStateDelinked, fmt.Sprintf("connect failure %d: %s", int(e.Reason), e.Message))
	case *events.StreamReplaced:
		p.setHealthState(healthStateStreamReplaced, "")
	case *events.Message:
		p.handleMessageEvent(e)
	case *events.HistorySync:
		p.handleHistorySync(e)
	case *events.GroupInfo:
		p.handleGroupInfoEvent(e)
	}
}

// handleMessageEvent appends a live or history-sync-replayed message to
// this plugin's own message store (messagestore.go) — the ONLY place this
// plugin ever writes a message row. A captured message from a chat
// matching no webspace's match configuration is still captured here (the
// plugin's own store necessarily captures every inbound message the
// linked device receives), but Match (plugin.go) only ever converts a
// MATCHED chat's rows into Items — capture must never become exposure.
func (p *SourcePlugin) handleMessageEvent(e *events.Message) {
	if e.Message.GetReactionMessage() != nil {
		return // a reaction is not a message in its own right in this plan's schema
	}

	chatJID := e.Info.Chat.String()
	isGroup := e.Info.IsGroup

	if err := p.store.EnsureChat(chatJID, isGroup); err != nil {
		fmt.Fprintf(p.logOut, "%s: ensure chat: %v\n", pluginName, err)
		return
	}

	if pm := e.Message.GetProtocolMessage(); pm != nil {
		switch pm.GetType() {
		case waE2E.ProtocolMessage_REVOKE:
			targetID := pm.GetKey().GetID()
			if err := p.store.MarkDeleted(chatJID, targetID); err != nil {
				fmt.Fprintf(p.logOut, "%s: mark deleted: %v\n", pluginName, err)
			}
		case waE2E.ProtocolMessage_MESSAGE_EDIT:
			targetID := pm.GetKey().GetID()
			newBody := extractMessageText(pm.GetEditedMessage())
			if err := p.store.MarkEdited(chatJID, targetID, newBody); err != nil {
				fmt.Fprintf(p.logOut, "%s: mark edited: %v\n", pluginName, err)
			}
		}
		return // every ProtocolMessage variant (recognised or not) carries no chat content of its own
	}

	body := extractMessageText(e.Message)
	if body == "" {
		return // no plaintext or known-media content this plugin captures (e.g. a receipt/control payload)
	}

	// Real-device spike (2026-08-10): a history-sync-replayed message's
	// own Info.PushName is empty for nearly every message ("the only
	// non-empty messages.sender_name is 'You'"). Fall back to the
	// best-effort pushNames cache (populated from HistorySync's own
	// top-level Pushnames list, handleHistorySync below) before falling
	// back further to the bare sender JID — never an empty string.
	senderName := e.Info.PushName
	if senderName == "" {
		senderName = p.pushNames.lookup(e.Info.Sender.ToNonAD().String())
	}
	if e.Info.IsFromMe {
		senderName = ownSenderLabel
	}

	rec := messageRecord{
		ID:           e.Info.ID,
		ChatJID:      chatJID,
		SenderJID:    e.Info.Sender.String(),
		SenderName:   senderName,
		IsFromMe:     e.Info.IsFromMe,
		SentAtUnixMs: e.Info.Timestamp.UnixMilli(),
		Body:         body,
	}
	if err := p.store.Append(rec); err != nil {
		fmt.Fprintf(p.logOut, "%s: append message: %v\n", pluginName, err)
	}
}

// handleHistorySync replays a whatsmeow first-link backfill payload
// through the identical handleMessageEvent path a live message uses (via
// Client.ParseWebMessage, whatsmeow's own documented conversion from a
// WebMessageInfo to an *events.Message) — so first-link backfill lands in
// the store identically to live messages, per this plan's own action
// text.
func (p *SourcePlugin) handleHistorySync(e *events.HistorySync) {
	if e.Data == nil {
		return
	}

	// Merge this payload's own top-level Pushnames list BEFORE
	// processing any message below, so the very first replayed message
	// from a newly-seen sender can already benefit from it.
	p.pushNames.merge(pushnamesFromProto(e.Data.GetPushnames()))

	count := 0
	for _, conv := range e.Data.GetConversations() {
		chatJID, err := types.ParseJID(conv.GetID())
		if err != nil {
			continue
		}
		for _, historyMsg := range conv.GetMessages() {
			webMsg := historyMsg.GetMessage()
			if webMsg == nil {
				continue
			}
			msgEvt, err := p.client.ParseWebMessage(chatJID, webMsg)
			if err != nil {
				continue
			}
			p.handleMessageEvent(msgEvt)
			count++
		}
	}
	fmt.Fprintf(p.logOut, "%s: history sync: processed %d message(s) across %d chat(s)\n", pluginName, count, len(e.Data.GetConversations()))
}

// handleGroupInfoEvent refreshes a group chat's cached subject — the ONLY
// path that ever writes a chat's name (T-08-01's mitigation: never a
// message sender's self-asserted push name).
func (p *SourcePlugin) handleGroupInfoEvent(e *events.GroupInfo) {
	if e.Name == nil {
		return
	}
	chatJID := e.JID.String()
	if err := p.store.UpsertChatName(chatJID, true, e.Name.Name, e.Timestamp.UnixMilli()); err != nil {
		fmt.Fprintf(p.logOut, "%s: upsert chat name: %v\n", pluginName, err)
	}
}

// pushnamesFromProto converts one HistorySync payload's own top-level
// Pushnames list (a JID->pushname map WhatsApp delivers once per history
// sync, distinct from any individual message's own PushName field — see
// pushNameCache's own doc comment) into a plain map for pushNameCache.merge.
func pushnamesFromProto(list []*waHistorySync.Pushname) map[string]string {
	out := make(map[string]string, len(list))
	for _, pn := range list {
		id := pn.GetID()
		if id == "" {
			continue
		}
		out[id] = pn.GetPushname()
	}
	return out
}

// extractMessageText returns msg's plaintext body (plain text or extended
// text), or attachmentPlaceholder for a known non-text message type
// (image/video/audio/document/sticker/contact/location), or "" for
// anything else this plugin does not capture at all (e.g. a poll,
// button-reply, or unrecognised payload shape).
func extractMessageText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if c := msg.GetConversation(); c != "" {
		return c
	}
	if et := msg.GetExtendedTextMessage(); et != nil && et.GetText() != "" {
		return et.GetText()
	}
	switch {
	case msg.GetImageMessage() != nil,
		msg.GetVideoMessage() != nil,
		msg.GetAudioMessage() != nil,
		msg.GetDocumentMessage() != nil,
		msg.GetStickerMessage() != nil,
		msg.GetContactMessage() != nil,
		msg.GetLocationMessage() != nil:
		return attachmentPlaceholder
	default:
		return ""
	}
}
