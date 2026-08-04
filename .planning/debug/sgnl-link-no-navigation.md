---
status: diagnosed
trigger: "UAT gap G-04-1b (phase 04): after the G-04-1 fix, the Signal contact-form deep link sgnl://signal.me/#p/+<E164> is accepted by Signal Desktop (no error modal), but clicking 'Open in Signal' does NOT navigate to the contact's conversation — Signal stays on whatever conversation was last open."
created: 2026-08-04T00:00:00Z
updated: 2026-08-04T01:35:00Z
---

## Current Focus

hypothesis: CONFIRMED — see Resolution.root_cause
test: complete
expecting: —
next_action: "Return ROOT CAUSE FOUND to orchestrator (goal: find_root_cause_only — no fix applied; plan-phase --gaps owns the outcome)"
bug_class: Bohrbug (deterministic — every click on the same item class reproduces identically; 4 recorded launches, all identical argv)
known_pattern_candidate: none (.planning/debug/knowledge-base.md does not exist; MemPalace unavailable in this environment — Phase 0 recorded as skipped)

reasoning_checkpoint:
  hypothesis: "The four retest clicks landed on GROUP conversation digests (CrapCarbs, webspace 'house-move'), for which plugins/signal/deeplink.go deliberately emits the bare 'sgnl://' fallback. A bare 'sgnl://' matches NO route in Signal Desktop's route table, and Signal's second-instance handler raises the window UNCONDITIONALLY before parsing argv — producing exactly 'window comes up, stays on last conversation, no error modal, no navigation'. The literal-'+' contact form (the thing G-04-1's fix changed) was never exercised by this retest."
  confirming_evidence:
    - "journalctl _CMDLINE for all four retest-window Signal launches (01:06:03, 01:06:52, 01:09:58, 01:10:07 BST): '/usr/lib/signal-desktop/signal-desktop -- sgnl://' — the bare scheme, no host, no fragment"
    - "The earlier pre-fix launch (00:11:15 BST) recorded '-- sgnl://signal.me/#p/%2B447990618267' via the SAME launcher chain (xdg-desktop-portal.service) — the chain demonstrably transmits full URLs; it did not truncate anything"
    - "Live API: webspace 'house-move' has exactly 6 signal digests, ALL titled 'CrapCarbs — N messages' and ALL carrying link.url == 'sgnl://'; webspace 'test' has 105 digests, ALL 'Dad' with 'sgnl://signal.me/#p/+447990618267'. 'sgnl://' can only have come from a CrapCarbs group click."
    - "Machine-verified: portal OpenURI with a literal-'+' URL delivered '-- sgnl://signal.me/#p/+0126' verbatim and Signal logged 'Matched signal route: contactByPhoneNumber' — the literal '+' survives browser->portal->argv->URLPattern intact"
    - "Signal's second-instance handler (app.asar ~23456708): 'Z&&(Z.isMinimized()&&Z.restore(),Sb());let r=Xx(t);return r!=null&&Zx(r)' — raise happens unconditionally BEFORE route parsing; a null parse is silent (no log, no modal)"
  falsification_test: "If any retest-window launch's _CMDLINE had contained the contact form (sgnl://signal.me/#p/+44...), or if the portal experiment had truncated the literal-'+' URL, this hypothesis would be false. Neither holds."
  fix_rationale: "No emitted-link defect exists. The 04-04 fix is correct and simply was not exercised. The actionable residue is a UX/verification gap: raise-only links and navigating links are visually indistinguishable in the UI (same button copy, same 'conversation-only' badge), so the intended group-fallback behavior is indistinguishable from a bug to the person testing it."
  blind_spots: "The real 1:1 link (+447990618267 -> 'Dad') was NOT clicked end-to-end — doing so would open a real conversation and mark it read, a state mutation on the user's Signal outside a debugger's remit. Its two remaining steps are covered statically instead: the delivery+validation half was machine-verified with a dud number (+0126, route matched, validator rejected as designed), and the lookup+navigate half was read out of the shipped bundle (f$n/lookupConversationWithoutServiceId). Every failure branch there is VISIBLE (user-not-found modal or a toast), so a silent no-op cannot originate in it."
  candidate_causes:
    - "data: the clicked digests were group conversations, which have no E.164 and therefore take the documented bare-'sgnl://' fallback (CONFIRMED as the proximate cause)"
    - "code: plugins/signal/deeplink.go emits a wrong/mangled link (ELIMINATED — live API shows the correct literal-'+' contact form on all 105 1:1 digests; the bare form is emitted only where it is designed to be)"
    - "environment: the browser/xdg/portal chain mangles the literal '+' (ELIMINATED by direct experiment — portal delivered '+0126' verbatim and Signal matched the route)"
    - "process/verification design: UAT test 1 was re-run against a webspace containing only group digests, so it could not exercise the fixed path (CONFIRMED as the cause of the false failure report)"
    - "product/UX: the UI presents raise-only and navigating deep links identically (same 'Open in Signal' button, same 'conversation-only' fidelity badge), so intended behavior is unfalsifiable to the user (CONFIRMED as the latent defect)"
  and_gate: "yes — two conditions had to hold simultaneously to produce a recorded UAT FAILURE. (1) the clicked digest was a group, so Signal only raised its window; AND (2) nothing in the UI distinguishes a link that navigates from one that only raises, so the tester had no way to tell intended-fidelity-limit from defect. Condition (1) alone is approved behavior (signed off at 04-01's checkpoint); condition (2) alone is harmless while every link navigates. Together they manufacture a false blocker and hide a real UX gap."

## Symptoms
<!-- prefilled by orchestrator (symptoms_prefilled: true) — IMMUTABLE -->

expected: With Signal Desktop running, clicking "Open in Signal" on a 1:1 (E.164-bearing) conversation's digest raises Signal Desktop and navigates to that contact's conversation.
actual: "the error dialog no longer appears, but the correct conversation is not opened, it remains on whatever was last opened in the UI" (verbatim user report). Signal stays on the previously open conversation.
errors: None reported — the previous validation-error modal is GONE.
reproduction: Test 1 in 04-UAT.md — open a webspace with a Signal digest in the webspaces UI (server on 127.0.0.1:7777), open detail pane, click "Open in Signal".
started: Discovered during UAT re-run 2026-08-04, immediately after gap-closure plan 04-04 changed the emitted link from percent-encoded to literal-plus.

## Eliminated

- hypothesis: "Signal's post-validation tail (ACI/CDSI lookup, contact-known gating, confirmation sheet) silently no-ops for a valid E.164"
  evidence: "Traced the full tail in the shipped bundle. showConversationViaSignalDotMe -> f$n(lookupConversationWithoutServiceId) -> CDSI lookup -> maybeMergeContacts -> showConversation. EVERY non-success branch is user-visible: no result => showUserNotFoundModal; thrown error => toast FailedToFetchPhoneNumber; invalid E164 or null id => F_r() error modal. The only silent paths are success (navigates) and the not-registered early return. 'Silent + no navigation' cannot originate here. Independently, Signal's logs show this handler never ran during the retest window."
  timestamp: 2026-08-04T00:20:00Z

- hypothesis: "A formatting mismatch between our emitted E.164 and Signal's conversations table makes the lookup miss"
  evidence: "Moot — the handler never executed during the retest (no 'Matched signal route' in main.log, no renderer log line). Additionally f$n's e164 branch does not consult the local conversations table first; it goes to CDSI and merges by returned aci/pni, so a local-column format mismatch is not a failure mode for this route. No Signal DB read was necessary."
  timestamp: 2026-08-04T00:30:00Z

- hypothesis: "The browser / xdg / xdg-desktop-portal chain mangles or drops the literal '+' in transit (environment)"
  evidence: "Direct experiment: portal OpenURI('sgnl://signal.me/#p/+0126') -> journal _CMDLINE '/usr/lib/signal-desktop/signal-desktop -- sgnl://signal.me/#p/+0126' (verbatim, literal '+') -> main.log 'Matched signal route: contactByPhoneNumber'. Also 'gio open' with the same shape matched the route. The chain is clean for literal '+'."
  timestamp: 2026-08-04T01:29:00Z

- hypothesis: "Our own code emits a wrong link (deeplink.go regression, index corruption, or a frontend sanitizer truncating the href)"
  evidence: "Live API confirms all 105 1:1 digests carry 'sgnl://signal.me/#p/+447990618267' (literal '+', correct shape) and the 6 group digests carry 'sgnl://' exactly as conversationDeepLink() specifies. OpenInSource.svelte passes link.url into an anchor href verbatim; grep found no URL sanitization/rewriting anywhere in web/src. Nothing in our chain truncates."
  timestamp: 2026-08-04T01:32:00Z

## Evidence

- timestamp: 2026-08-04T00:00:00Z
  checked: "Prior debug session .planning/debug/sgnl-link-didnt-make-sense.md (validation-layer trace)"
  found: "Path traced: emitted URL -> contactByPhoneNumber URLPattern route -> captured group verbatim over IPC 'show-conversation-via-signal.me' -> renderer showConversationViaSignalDotMe -> CPt E.164 regex -> on failure F_r() error modal. Recorded blind spot: the tail after a VALID number was never traced."
  implication: "Delivery to the renderer is proven possible (the old modal was reached). Start by tracing the untraced tail."

- timestamp: 2026-08-04T00:10:00Z
  checked: "Installed /usr/lib/signal-desktop/resources/app.asar (signal-desktop 8.21.0-1), renderer showConversationViaSignalDotMe (~offset 29586282)"
  found: |
    async showConversationViaSignalDotMe(e,n){
      if(!gj()){N9.info('...Not registered, returning early');return}
      let{showUserNotFoundModal:r}=window.reduxActions.globalModals,a;
      try{
        if(e==='phoneNumber') CPt(n,!0)&&(a=await f$n({type:'e164',e164:n,phoneNumber:n,showUserNotFoundModal:r,setIsFetchingUUID:L_r}));
        ...
        if(a!=null){window.reduxActions.conversations.showConversation({conversationId:a});return}
      }catch(e){N9.warn('...got error',...),F_r();return}
      N9.info('showConversationViaSignalDotMe: invalid E164'),F_r()
    }
  implication: "Every post-validation failure branch is visible (modal) or logged. A silent no-op is not reachable here for a registered client."

- timestamp: 2026-08-04T00:15:00Z
  checked: "app.asar f$n = lookupConversationWithoutServiceId (~offset 28829899)"
  found: "For type 'e164': awaits a CDSI directory lookup (iIt) — note it does NOT short-circuit on an existing local conversation the way the username branch does — then ConversationController.maybeMergeContacts({aci,pni,e164}) and returns conversation.id. If no id: showUserNotFoundModal({type:'phoneNumber',phoneNumber}). On throw: toast FailedToFetchPhoneNumber."
  implication: "The 1:1 path requires network (CDSI) even for a contact already in the local DB, and both of its failure modes are visible to the user. Confirms the silent symptom cannot come from here, and flags a real runtime caveat for the 1:1 path (offline => visible toast, unregistered number => visible modal)."

- timestamp: 2026-08-04T00:20:00Z
  checked: "Signal Desktop's own logs (~/.config/Signal/logs/main.log + app.log, read-only) against main's commit timeline (BST = UTC+1)"
  found: "Signal's ENTIRE log history contains exactly two deep-link route events: 2026-08-03T20:24:24Z and 2026-08-03T23:11:15.125Z, each followed 0-1ms later by the renderer's 'invalid E164'. Both predate the fix merge (2756532, 00:47:20 BST = 23:47Z). In the retest window (23:54Z-00:08Z) there is NO route match, NO showConversationViaSignalDotMe line, NO unknownSignalLink — while routine logging (30s websocket keepalives) continued unbroken."
  implication: "The retest link never reached Signal's route handler. The crime scene moves off Signal's post-validation tail and onto delivery."

- timestamp: 2026-08-04T00:22:00Z
  checked: "app.asar second-instance handler (~offset 23456708) and handleSignalRoute/Zx (~23490539); route table (~21870744)"
  found: "app.on('second-instance',(e,t)=>{...;Z&&(Z.isMinimized()&&Z.restore(),Sb());let r=Xx(t);return r!=null&&Zx(r)}) — the window raise (Sb) is UNCONDITIONAL and happens BEFORE argv parsing. Xx scans each argv entry through the route parser and returns null with NO logging when nothing parses; Zx logs only on a match. Route parse/schema failures inside a matched pattern DO log ('Failed to parse route ...') — no such line exists."
  implication: "An unparseable URL in argv produces precisely: window raises, stays on last conversation, no navigation, no modal, no log. This is the exact reported symptom and is invisible in Signal's logs — which is why the earlier evidence looked like 'nothing happened'."

- timestamp: 2026-08-04T00:25:00Z
  checked: "Live machine handoff (controlled, dud number): gio open 'sgnl://signal.me/#p/+0123'"
  found: "gio exit 0; main.log 00:25:29.642Z 'Matched signal route: contactByPhoneNumber'; app.log 00:25:29.643Z 'showConversationViaSignalDotMe: invalid E164' (the dud number failing validation exactly as designed — no CDSI lookup, no navigation, no state change)."
  implication: "The literal '+' survives the OS chain (gio -> second-instance argv -> URL parse -> URLPattern -> IPC -> renderer) intact. '+' is not the problem."

- timestamp: 2026-08-04T00:27:00Z
  checked: "journalctl --user -o json _CMDLINE for every signal-desktop launch around the retest (BST timestamps)"
  found: |
    00:11:15  pid 3189680  xdg-desktop-portal.service  "/usr/lib/signal-desktop/signal-desktop -- sgnl://signal.me/#p/%2B447990618267"   <- pre-fix click, route MATCHED
    01:06:03  pid 3227562  xdg-desktop-portal.service  "/usr/lib/signal-desktop/signal-desktop -- sgnl://"
    01:06:52  pid 3228102  xdg-desktop-portal.service  "/usr/lib/signal-desktop/signal-desktop -- sgnl://"
    01:09:58  pid 3229649  xdg-desktop-portal.service  "/usr/lib/signal-desktop/signal-desktop -- sgnl://"
    01:10:07  pid 3229832  xdg-desktop-portal.service  "/usr/lib/signal-desktop/signal-desktop -- sgnl://"
    Retest window bounds: fix merged 00:47:20 BST, UAT reopened 00:54:22 BST, result recorded 01:08:23 BST.
  implication: "DECISIVE. All four retest-window launches carried the BARE 'sgnl://' — not the contact form. Signal did launch and raise its window each time (matching app.log's 'app is focused' at 01:06:07 BST), then found no route and went silent. The same portal chain had transmitted a full URL 55 minutes earlier, so nothing truncated it: 'sgnl://' is what the clicked link literally was."

- timestamp: 2026-08-04T01:29:00Z
  checked: "Differential experiment isolating browser-vs-portal: gdbus org.freedesktop.portal.Desktop OpenURI.OpenURI('', 'sgnl://signal.me/#p/+0126', {})"
  found: "journal _CMDLINE: '/usr/lib/signal-desktop/signal-desktop -- sgnl://signal.me/#p/+0126' (literal '+' verbatim); main.log 00:29:36.418Z 'Matched signal route: contactByPhoneNumber'."
  implication: "The portal — the exact launcher Brave used for all five recorded launches — preserves literal-'+' URLs end to end. The 'sgnl://' in the retest argv therefore originated in the page's href, not in any transport."

- timestamp: 2026-08-04T01:31:00Z
  checked: "Live index via the running server's read-only API (GET /api/webspaces, /api/webspaces/{ws}/stream on 127.0.0.1:7777)"
  found: |
    webspace 'test'       (keywords: Dad)                          -> 105 signal digests, ALL titled 'Dad — N messages',       ALL link.url = sgnl://signal.me/#p/+447990618267  (fidelity conversation-only)
    webspace 'house-move' (keywords: house and home, ..., CrapCarbs) ->   6 signal digests, ALL titled 'CrapCarbs — N messages', ALL link.url = sgnl://
    Only these two webspaces exist. No 1:1 digest anywhere carries the bare form; no group digest carries the contact form.
  implication: "'sgnl://' in the retest argv can only have come from clicking a CrapCarbs GROUP digest in 'house-move'. The retest never clicked a 1:1 digest, so it never exercised the code 04-04 fixed."

- timestamp: 2026-08-04T01:32:00Z
  checked: "web/src/lib/components/OpenInSource.svelte, DetailPane.svelte, api.ts — the click path from API JSON to the anchor"
  found: "OpenInSource renders <Button href={link.url} target=_blank rel='noopener noreferrer'> with the API value verbatim; grep for sgnl/protocol/sanitiz/safeUrl/new URL/startsWith across web/src found no URL rewriting or validation of link.url anywhere. The same component renders BOTH link classes identically — same 'Open in {source}' copy, same 'conversation-only' badge — whether the link navigates to a conversation or merely raises the app."
  implication: "Confirms our frontend did not truncate anything. Also surfaces the latent product defect: nothing in the UI lets a user (or a UAT tester) distinguish a link that will navigate from one that will only raise Signal — which is why intended behavior was reported as a blocker."

- timestamp: 2026-08-04T01:34:00Z
  checked: "Complete sgnl: route table extracted from app.asar (~21870744) + URLPattern verification of candidate link shapes (node v26.5.1, same URLPattern spec Chromium implements)"
  found: |
    Routes registered for the sgnl: scheme in 8.21.0: contactByPhoneNumber (signal.me #p/), contactByEncryptedUsername (signal.me #eu/),
    groupInvites (signal.group/# , sgnl://joingroup/#), linkDevice, captcha (signalcaptcha:), linkCall, artAddStickers,
    showConversation (sgnl://show-conversation?token=), startCallLobby, showWindow (sgnl://show-window), cancelPresenting, donation{ValidationComplete,PaypalApproved,PaypalCanceled}.
    Verified matches:  "sgnl://"                           -> NO ROUTE (silent no-op)
                       "sgnl://show-window"                -> showWindow
                       "sgnl://signal.me/#p/+447990618267" -> contactByPhoneNumber
    showConversation takes an ephemeral token resolved by Signal's own fM.resolveToken (minted internally for notifications) — not constructible by an external app.
    groupInvites is join-by-invite-code semantics (prompts to JOIN), requires the group's invite link, and does not open an existing conversation.
  implication: "There is NO route in Signal Desktop 8.21.0 that opens an EXISTING group conversation, and none that opens a conversation by id. Conversation-only fidelity for groups is a hard upstream ceiling, exactly as the phase's locked decision assumed. 'sgnl://show-window' is, however, a strictly more honest fallback than the bare 'sgnl://': it hits a route Signal explicitly handles instead of relying on the unmatched-URL side effect of the unconditional second-instance raise."

## Resolution

root_cause: |
  NOT a defect in the emitted deep link. Two conditions combined (AND-gate) to produce the report:

  (1) PROXIMATE — the retest clicked GROUP digests, not 1:1 digests. All four Signal launches in the retest window
      received argv "signal-desktop -- sgnl://" (journalctl _CMDLINE, pids 3227562/3228102/3229649/3229832 at
      01:06:03/01:06:52/01:09:58/01:10:07 BST). The bare "sgnl://" is exactly what plugins/signal/deeplink.go's
      conversationDeepLink() emits by design for a group (or a 1:1 with no valid E.164), and the live index shows the
      only Signal items carrying it are the 6 'CrapCarbs' group digests in webspace 'house-move'. The 105 1:1 'Dad'
      digests in webspace 'test' all carry the corrected literal-'+' contact form. Signal Desktop's second-instance
      handler raises its window UNCONDITIONALLY before parsing argv, then finds that "sgnl://" matches no route and
      returns null WITHOUT logging or showing anything — producing precisely "window comes up, no error dialog, stays
      on whatever was last open". That is the documented, already-approved conversation-only fidelity limit for groups
      (signed off at 04-01's checkpoint), not a regression. The literal-'+' fix from 04-04 was therefore never exercised
      by this retest: the contact-form link never entered Signal's argv even once after the fix merged.

  (2) LATENT (the real, actionable defect) — the UI makes intended behavior indistinguishable from a bug. OpenInSource.svelte
      renders raise-only links and navigating links identically: same "Open in Signal" button, same "conversation-only"
      fidelity badge. A user (and the UAT tester) has no way to know that a group digest's button can only raise the app,
      so correct behavior reads as a failure. Without condition (2), condition (1) would have been recognised on sight.

  Everything else on the path is verified healthy: the literal '+' survives browser -> xdg-desktop-portal -> argv ->
  URLPattern -> IPC intact (portal OpenURI with '+0126' delivered verbatim and matched contactByPhoneNumber), the index
  and API emit the correct shape on all 105 1:1 rows, and the frontend passes link.url through unmodified.

fix: |
  (not applied — goal: find_root_cause_only). Directions for the gap-closure planner, in priority order:

  A. RETEST CORRECTLY FIRST (no code change): re-run UAT test 1 against webspace 'test' (keyword 'Dad') — every one of its
     105 digests carries sgnl://signal.me/#p/+447990618267. Verify mechanically rather than by eye, using the technique
     that cracked this session: after the click, run
       journalctl --user --since '-2min' -o json | grep -o '"_CMDLINE":"[^"]*signal-desktop[^"]*"'
     and confirm the argv contains the full contact-form URL, plus `grep 'Matched signal route' ~/.config/Signal/logs/main.log`.
     That converts "did the deep link work?" from a judgment call into a machine-checkable assertion.
     Caveat to expect on the 1:1 path (traced, not a defect): f$n performs a CDSI network lookup even for a contact
     already in the local DB, so this path needs connectivity; an unregistered number yields a visible "user not found"
     modal and a network failure yields a visible toast.

  B. CLOSE THE REAL GAP (UX honesty for raise-only links): differentiate the two link classes in the UI so a raise-only
     link never masquerades as a navigating one — e.g. a distinct fidelity value (or a boolean on Link) for
     "app-raise-only", different button copy ("Open Signal" vs "Open conversation in Signal"), and a one-line explanation
     that Signal Desktop exposes no deep link for group conversations. This is what actually failed the user's
     expectation, and it generalises to any future source with a partial deep-link story.

  C. OPTIONAL HARDENING (small, evidence-backed): emit "sgnl://show-window" instead of the bare "sgnl://" for the fallback.
     Verified: "sgnl://show-window" matches Signal's registered showWindow route (explicit, supported behavior), whereas
     the bare "sgnl://" matches NO route and gets its window-raise only as a side effect of the unconditional
     second-instance raise. Same user-visible result today, but resting on a supported route instead of an accident.

  D. UPSTREAM LIMITATION — record as closed, do not chase: Signal Desktop 8.21.0's complete sgnl: route table
     (contactByPhoneNumber, contactByEncryptedUsername, groupInvites, linkDevice, captcha, linkCall, artAddStickers,
     showConversation, startCallLobby, showWindow, cancelPresenting, donation*) contains NO route that opens an existing
     group conversation and none that opens a conversation by id. showConversation needs an ephemeral token minted
     internally by Signal; groupInvites is join-by-invite-code semantics. No emittable link can navigate to a group.

verification: |
  (diagnosis verification — no fix applied) Root cause established by differential experiment plus recorded ground truth,
  not inference:
  - Ground truth on what Signal actually received: journalctl _CMDLINE for all five recorded launches (one pre-fix with the
    full %2B URL, four in the retest window with bare "sgnl://").
  - The transport was exonerated by running the exact same launcher (xdg-desktop-portal OpenURI) with a literal-'+' URL and
    a dud number: argv arrived verbatim and Signal logged 'Matched signal route: contactByPhoneNumber'. A second,
    independent handoff (gio open) reproduced the same result.
  - The data side was read live from the running server's read-only API: bare links exist on exactly the 6 group digests
    and nowhere else.
  - Signal-side silence was explained mechanically from the shipped bundle (unconditional raise before argv parse; null
    parse returns without logging or modal), and every alternative silent-failure branch was ruled out by reading the
    post-validation tail (all visible: modal or toast).
  Constraints honored: Signal's database and config were never opened or written; the live index was not modified; the
  developer's `webspaces serve` on 127.0.0.1:7777 was only queried with GETs and never restarted. Test links used the
  non-routable dud numbers +0123/+0126, which fail Signal's own validator before any lookup, so no conversation was
  opened and no read-state was changed.

files_changed: []
