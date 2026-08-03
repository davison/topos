---
status: diagnosed
trigger: "UAT Phase 04 Test 1: Clicking 'Open in Signal' on a 1:1 (E.164-bearing) conversation digest — Signal Desktop shows 'Something went wrong! Sorry, that sgnl:// link didn't make sense!'"
created: 2026-08-04T00:00:00Z
updated: 2026-08-04T00:30:00Z
---

## Current Focus

hypothesis: CONFIRMED — see Resolution.root_cause
test: complete
expecting: —
next_action: "Return ROOT CAUSE FOUND to orchestrator (goal: find_root_cause_only — no fix applied; plan-phase --gaps owns the fix)"
bug_class: Bohrbug (deterministic — Signal rejects the same link every time)

reasoning_checkpoint:
  hypothesis: "plugins/signal/deeplink.go percent-encodes the E.164's leading '+' as '%2B' (url.QueryEscape), emitting 'sgnl://signal.me/#p/%2B44...'. Signal Desktop never percent-decodes the hash-captured phoneNumber, and its E.164 validator /^\\+[1-9]\\d{1,14}$/ requires a LITERAL '+', so validation fails and Signal shows the 'unknown-sgnl-link' error modal."
  confirming_evidence:
    - "deeplink.go line 49+62: fragment built via url.QueryEscape; deeplink_test.go TestEncodePhoneFragment_PlusSignEscaped asserts '+15551234567' -> '%2B15551234567' (the mangling is deliberate and test-enshrined)"
    - "Installed /usr/lib/signal-desktop/resources/app.asar: route contactByPhoneNumber matches hash 'p/:phoneNumber' with schema .min(1) only; captured group passed verbatim over IPC (URLPattern/WHATWG URL do not percent-decode hash or captured groups)"
    - "app.asar renderer: function CPt(e,t){...(t?/^\\+[1-9]\\d{1,14}$/:...).test(e)...} called as CPt(value,!0); on false -> log 'invalid E164' -> F_r() -> showErrorModal(title 'icu:ErrorModal--title', description 'icu:unknown-sgnl-link') — the exact observed dialog"
  falsification_test: "If Signal Desktop percent-decoded the hash anywhere before CPt, or CPt accepted '%2B', the hypothesis would be false — neither is present in the shipped bundle"
  fix_rationale: "Emit the literal '+': E.164 is '+' followed by digits, and '+' is a legal sub-delim in a URI fragment per RFC 3986 — no encoding is needed or tolerated by Signal's parser. Addresses the root cause (payload shape), not a symptom."
  blind_spots: "Not re-executed live against Signal Desktop from this session (would disturb the user's running desktop); confirmed statically against the installed binary's code instead. Literal-'+' acceptance verified by reading CPt + lookup flow, matching the shkspr.mobi source the code itself cites."
  candidate_causes:
    - "code: url.QueryEscape mangles '+' to '%2B' in the fragment (CONFIRMED)"
    - "environment: gio/xdg handoff mangles or re-encodes the URI in transit (ELIMINATED — handoff machine-verified; observed failure is Signal's own post-routing validation modal, which requires the URI to have arrived and matched the signal.me route)"
    - "data: malformed e164 value stored in Signal's DB (ELIMINATED as primary — regardless of the stored value, QueryEscape unconditionally converts the leading '+' to '%2B', which alone guarantees CPt failure; a malformed/empty e164 would have produced the bare 'sgnl://' fallback instead of this link)"
  and_gate: "no — a single condition (percent-encoded '+' reaching a validator that demands a literal '+') is sufficient to produce the exact observed dialog; no second contributing condition required"

## Symptoms

expected: With Signal Desktop running, clicking "Open in Signal" on a 1:1 (E.164-bearing) conversation's digest raises Signal Desktop and navigates to that contact's conversation. Exit-0 + IPC handoff was already machine-verified; what needed a human eye was the visible window-raise + conversation navigation.
actual: Signal Desktop displayed a modal dialog "Something went wrong! Sorry, that sgnl:// link didn't make sense!" instead of navigating to the contact's conversation.
errors: Signal Desktop dialog text: "Something went wrong! Sorry, that sgnl:// link didn't make sense!"
reproduction: Test 1 in .planning/phases/04-signal-conversations/04-UAT.md — open a webspace whose keywords match a 1:1 Signal conversation, open the digest detail pane, click "Open in Signal".
started: Discovered during UAT (2026-08-03). The bare-scheme group form was visually confirmed OK during 04-01's checkpoint; the failure is specific to the E.164 contact form of the link.

## Eliminated

- hypothesis: "The URL shape itself (sgnl://signal.me/#p/... host/fragment form) is wrong for Signal Desktop"
  evidence: "The installed binary explicitly registers the pattern X('sgnl:','signal.me','{/}?',{hash:'p/:phoneNumber'}) — the sgnl://signal.me/#p/<value> shape is a first-class supported route; only the value's encoding is at fault."
  timestamp: 2026-08-04T00:20:00Z

- hypothesis: "gio/xdg handoff mangles or drops the URI before Signal receives it (environment)"
  evidence: "Handoff was machine-verified during 04-01/04-03 (gio open exit-0, single-instance-lock IPC forward observed). The observed failure is Signal's own renderer-side validation modal, which is only reachable after the URI arrived intact, matched the signal.me route, and got as far as E.164 validation."
  timestamp: 2026-08-04T00:25:00Z

- hypothesis: "The e164 value stored in Signal's DB is malformed (data)"
  evidence: "Irrelevant to the observed failure: url.QueryEscape unconditionally rewrites the leading '+' to '%2B', which alone guarantees CPt rejection even for a perfectly valid E.164. A missing/empty e164 would have produced the bare 'sgnl://' fallback link, not this signal.me form."
  timestamp: 2026-08-04T00:25:00Z

## Evidence

- timestamp: 2026-08-04T00:05:00Z
  checked: "grep for sgnl/signal.me across repo source"
  found: "Deep link built in exactly one place: plugins/signal/deeplink.go conversationDeepLink(). For private+E.164: 'sgnl://signal.me/#p/' + encodePhoneFragment(e164). encodePhoneFragment = url.QueryEscape, which encodes '+' as '%2B'. deeplink_test.go line 74-79 (TestEncodePhoneFragment_PlusSignEscaped) explicitly asserts encodePhoneFragment('+15551234567') == '%2B15551234567' — the '+'-mangling is deliberate and test-enshrined."
  implication: "The link Signal Desktop received was 'sgnl://signal.me/#p/%2B44...' (percent-encoded plus), not 'sgnl://signal.me/#p/+44...' (literal plus) as documented in the sources the code itself cites (shkspr.mobi blog shows literal '+')."

- timestamp: 2026-08-04T00:15:00Z
  checked: "Installed Signal Desktop binary: /usr/lib/signal-desktop/resources/app.asar (pacman package signal-desktop) — signal.me route definition"
  found: "Route 'contactByPhoneNumber' defined with patterns X('https:','signal.me','{/}?',{hash:'p/:phoneNumber'}) and X('sgnl:','signal.me','{/}?',{hash:'p/:phoneNumber'}); schema is only .min(1); parse() returns e.hash.groups.phoneNumber verbatim. Matched route sends IPC 'show-conversation-via-signal.me' {kind:'phoneNumber', value:<captured group>}. URLPattern/WHATWG URL do not percent-decode the hash or captured groups, so value arrives as '%2B44...'."
  implication: "No percent-decoding happens in the main process; the '%2B'-prefixed string flows to the renderer as-is."

- timestamp: 2026-08-04T00:20:00Z
  checked: "app.asar renderer: showConversationViaSignalDotMe flow and E.164 validator"
  found: "showConversationViaSignalDotMe: for kind 'phoneNumber' calls CPt(value,!0) before any conversation lookup; on false, logs 'showConversationViaSignalDotMe: invalid E164' and calls F_r(). Validator: function CPt(e,t){return typeof e==='string'?(t?/^\\+[1-9]\\d{1,14}$/:/^\\+?[1-9]\\d{1,14}$/).test(e):!1} — requires a LITERAL '+' first character. '%2B447...' starts with '%', fails."
  implication: "The percent-encoded '+' is precisely what makes the shipped validator reject the value."

- timestamp: 2026-08-04T00:25:00Z
  checked: "app.asar: what F_r() displays"
  found: "function F_r(){window.reduxActions.globalModals.showErrorModal({title:P9('icu:ErrorModal--title'),description:P9('icu:unknown-sgnl-link')})} — i.e. title 'Something went wrong!' + description 'Sorry, that sgnl:// link didn't make sense!' (string 'didn't make sense!' confirmed present in the same bundle)."
  implication: "This is byte-for-byte the dialog the user photographed — the failure path is fully traced from our emitted URL to the observed modal."

## Resolution

root_cause: "plugins/signal/deeplink.go's encodePhoneFragment() runs the E.164 through url.QueryEscape, which percent-encodes the mandatory leading '+' as '%2B', emitting 'sgnl://signal.me/#p/%2B44...'. Signal Desktop's link pipeline never percent-decodes the hash-captured phoneNumber (URLPattern group -> .min(1) schema -> IPC -> renderer), and its E.164 validator (/^\\+[1-9]\\d{1,14}$/, mustStartWithPlus=true) requires a LITERAL '+' as the first character. '%2B...' fails the regex, so Signal logs 'invalid E164' and shows the ErrorModal with 'icu:unknown-sgnl-link' — exactly the observed 'Something went wrong! Sorry, that sgnl:// link didn't make sense!' dialog. The encoding choice is deliberate and test-enshrined (TestEncodePhoneFragment_PlusSignEscaped asserts '+15551234567' -> '%2B15551234567'), so the unit tests pass while the real consumer rejects the payload. url.QueryEscape is also the wrong encoder for a URI fragment in general — it targets application/x-www-form-urlencoded query components; '+' is a legal sub-delim in a fragment per RFC 3986 and needs no encoding."
fix: "(not applied — goal: find_root_cause_only) Direction: emit the literal '+': validate the source value against ^\\+[1-9][0-9]{1,14}$ and emit it verbatim when valid, falling back to the bare 'sgnl://' form when not (any character that would 'need escaping' makes the value not-an-E.164, which Signal would reject anyway — validate-and-fallback, not escape). Invert/replace TestEncodePhoneFragment_PlusSignEscaped and rework TestDeepLink_UnsafeCharactersAreEscapedNotEmittedRaw to assert the fallback instead of escaping."
verification: "(diagnosis verification) Full failure path traced statically through the exact installed binary (/usr/lib/signal-desktop/resources/app.asar): emitted URL -> contactByPhoneNumber URLPattern route -> verbatim group over IPC -> CPt regex rejection -> F_r() -> the photographed modal. Literal-'+' acceptance confirmed by the same code path (CPt passes '+44...' and proceeds to e164 conversation lookup)."
files_changed: []
