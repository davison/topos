// Command webspaces-plugin-proton: MIME part extraction from a peeked
// RFC822 message. This plan extracts only the text/plain part — the
// sanitized text/html rendition path is plan 03-02 (see PLAN.md
// objective).
package main

import (
	"bytes"
	"io"
	"unicode/utf8"

	message "github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
)

const (
	// maxPartBytes bounds a single MIME part read, well under
	// sdk.MaxMessageSize (64 MiB), so a crafted message with one enormous
	// part cannot exhaust memory (T-03-07).
	maxPartBytes = 8 * 1024 * 1024
	// maxParts bounds the number of MIME parts read from a single
	// message, so a crafted message with thousands of parts cannot
	// exhaust memory or spin forever (T-03-07).
	maxParts = 256
	// previewRuneCap mirrors plugins/silverbullet's Snippet rune cap —
	// truncation is by rune count, never byte count, so a multi-byte
	// preview is never cut mid-codepoint.
	previewRuneCap = 500
)

// PlainTextPart extracts the first text/plain inline part from a peeked
// RFC822 message via mail.CreateReader/NextPart. Every part read goes
// through io.LimitReader bounded by maxPartBytes, and the loop stops
// after maxParts parts, so a maliciously crafted message cannot exhaust
// memory (T-03-07). A message with no text/plain part returns empty text
// rather than an error.
//
// go-message's CreateReader/NextPart can both return a non-fatal
// "unknown charset/transfer-encoding" error alongside a still-usable
// reader/part (message.IsUnknownCharset / message.IsUnknownEncoding) —
// this function treats that class of error as recoverable (keep reading
// with whatever best-effort decode go-message produced) rather than
// failing extraction outright, since one malformed part must not make
// the whole message's body permanently unavailable.
func PlainTextPart(raw []byte) (string, error) {
	mr, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil && !isBenignParseError(err) {
		return "", err
	}
	if mr == nil {
		return "", nil
	}
	defer mr.Close()

	for i := 0; i < maxParts; i++ {
		p, perr := mr.NextPart()
		if perr == io.EOF {
			break
		}
		if perr != nil && !isBenignParseError(perr) {
			return "", perr
		}
		if p == nil {
			continue
		}

		h, ok := p.Header.(*mail.InlineHeader)
		if !ok {
			continue
		}
		ct, _, _ := h.ContentType()
		if ct != "text/plain" {
			continue
		}

		b, _ := io.ReadAll(io.LimitReader(p.Body, maxPartBytes))
		return string(b), nil
	}

	return "", nil
}

// isBenignParseError reports whether err is the "unknown charset" or
// "unknown transfer encoding" class of error go-message returns
// alongside a still-usable reader/part — see the package doc comments on
// mail.CreateReader and mail.Reader.NextPart.
func isBenignParseError(err error) bool {
	return message.IsUnknownCharset(err) || message.IsUnknownEncoding(err)
}

// Snippet truncates s to at most previewRuneCap runes, by rune count
// never byte count, so a multi-byte body preview is never cut
// mid-codepoint. Mirrors plugins/silverbullet's Snippet helper.
func Snippet(s string) string {
	if utf8.RuneCountInString(s) <= previewRuneCap {
		return s
	}
	runes := []rune(s)
	return string(runes[:previewRuneCap])
}
