package main

import (
	"fmt"

	"go.mau.fi/whatsmeow/proto/waE2E"
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
		p.setHealthy()
		fmt.Fprintf(p.logOut, "%s: connected\n", pluginName)
	case *events.LoggedOut:
		p.setUnhealthy("whatsapp: session logged out from the phone (WhatsApp > Linked devices > this device > Log out)")
	case *events.StreamReplaced:
		p.setUnhealthy("whatsapp: stream replaced by another session")
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

	senderName := e.Info.PushName
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
