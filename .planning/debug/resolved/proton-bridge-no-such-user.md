---
status: resolved
trigger: "Proton source unavailable (red dot); live IMAP LOGIN to Proton Mail Bridge returns 'no such user' even with correct account email and aliases. Previously written off since 03-01 as a credential-entry issue; user has now ruled that out."
created: 2026-07-31T00:00:00Z
updated: 2026-07-31T00:00:00Z
---

## Current Focus
<!-- OVERWRITE on each update - reflects NOW -->

bug_class: Bohrbug (deterministic — every LOGIN attempt fails identically, across four verification rounds)
known_pattern_candidate: none (knowledge-base.md does not exist; one resolved session about UI tooltip — unrelated)
hypothesis: CONFIRMED (H1) — the stored PROTON_BRIDGE_PASS is not a Bridge-generated app password (37 symbol-heavy chars vs Bridge's base64url-only rendering); bridge CheckAuth base64url-decodes the password BEFORE examining the username, so the corrupted password makes every (username, password) pair fail, and gluon reports its single auth-failure string "no such user" regardless of which address is tried.
test: complete — code path audited (verbatim pass-through), .env/config inspected (password shape impossible for a Bridge password), monroe:1143 probed (live genuine Bridge 3.25.0), upstream gluon+bridge source confirmed error semantics and password-first check order.
expecting: n/a — diagnosis complete
next_action: return ROOT CAUSE FOUND to orchestrator (goal: find_root_cause_only — no fix applied)

reasoning_checkpoint:
  hypothesis: "Stored PROTON_BRIDGE_PASS is the wrong class of secret (account/password-manager password, not the Bridge app password); Bridge validates the password before the username, and gluon emits 'no such user' for every auth failure — so username changes can never succeed."
  confirming_evidence:
    - ".env PROTON_BRIDGE_PASS: len=37 with 14 symbol types incl. both quote characters — ≥11 characters outside base64url; proton-bridge pkg/algo/encode.go proves the displayed Bridge password is base64.RawURLEncoding output ([A-Za-z0-9-_] only)"
    - "proton-bridge state.go CheckAuth: B64RawDecode(password) is step 1, BridgePass compare is step 2, email match is step 3 — with this password step 1 always errors, so the username is never examined"
    - "gluon backend.go getUserID: single ErrNoSuchUser for all failed pairs + ErrLoginBlocked after maxLoginAttempts — matches BOTH errors 03-01 observed verbatim"
    - "monroe:1143 greeting: genuine Proton Mail Bridge 03.25.00, STARTTLS — transport/forwarder healthy; 03-01 confirmed pinned-cert TLS completes"
  falsification_test: "Replace PROTON_BRIDGE_PASS with the password Bridge's own UI/CLI displays (pure [A-Za-z0-9-_]); if LOGIN with the Bridge-displayed username still returns 'no such user', this hypothesis is wrong (then check account signed-in state on monroe's Bridge and split-mode address restrictions)"
  fix_rationale: "Environmental correction, not code: store the actual Bridge app password. Code path is verified clean."
  blind_spots: "Cannot verify remotely (without a login attempt, which is out of bounds) that the account is currently signed in on monroe's Bridge — an empty/signed-out user list yields the same error; enumerated as user-side check #1. Cannot see Bridge's combined/split mode setting."
  candidate_causes:
    - "data/config: wrong class of secret stored in .env (CONFIRMED sufficient)"
    - "environment: Bridge account signed-out / keychain-lost on monroe (possible independent second condition — user must confirm; not required to explain symptoms)"
    - "code: credential mangling (ELIMINATED)"
    - "process/docs: live_bridge_test.go:160's hint text + 03-01-SUMMARY steered diagnosis to the USERNAME, misreading Bridge's error semantics (contributing cause of the 4-round persistence)"
  and_gate: "no — the corrupted password alone is sufficient to produce every observed symptom (red dot, test failure, alias-invariant error, rate-limit escalation); Bridge-side sign-in state is an independent possible extra condition, listed as a verification step rather than a confirmed contributing cause"

## Symptoms
<!-- Written during gathering, then IMMUTABLE -->

expected: Proton source healthy (no red dot); Proton emails matching webspace keywords appear in stream and detail pane; TestSeenFlagUnchanged_LiveBridge passes against live Bridge.
actual: |
  UI shows Proton source unavailable (red dot). Live test output verbatim:
    === RUN   TestSeenFlagUnchanged_LiveBridge
        live_bridge_test.go:63: live login: no such user (if this says "no such user", see 03-01-SUMMARY.md's documented Bridge-account credential finding — not a code defect)
    --- FAIL: TestSeenFlagUnchanged_LiveBridge (0.01s)
  User has since tried the correct account name (email address) and several aliases — all return "no such user".
errors: '"live login: no such user" (surfaced at plugins/proton/live_bridge_test.go:63)'
reproduction: |
  Tests 1–2 in .planning/phases/03-email-in-the-webspace/03-UAT.md;
  WEBSPACES_PROTON_LIVE_IT=1 PROTON_BRIDGE_ADDR=<addr> PROTON_BRIDGE_USER=<user> PROTON_BRIDGE_PASS=<pass> go test -run TestSeenFlagUnchanged_LiveBridge -v ./plugins/proton/...
started: Present since phase 03-01 (recorded unchanged across four verification rounds as assumed credential-entry issue); credential-entry error ruled out by user today.

## Eliminated
<!-- APPEND only - prevents re-investigating -->

- hypothesis: "H3 — plugin/test code mangles the username or password before LOGIN"
  evidence: plugins/proton/client.go connect() (lines 196-208) and live_bridge_test.go liveDialLiveIT (line 158) pass env/config values verbatim to go-imap v1's Login; go-imap encodes IMAP astrings correctly (quoted-string escaping / literals). kernel/config expandEnv substitutes the raw env value; main.go json-decodes it untouched. No trim/lowercase/re-encode anywhere in the path.
  timestamp: 2026-07-31

- hypothesis: "H2 — the LAN forward reaches the wrong service, wrong port, or a dead Bridge"
  evidence: Read-only greeting probe of monroe:1143 returned 'Proton Mail Bridge 03.25.00 - gluon session ID 137' with STARTTLS capability; 03-01 additionally confirmed the pinned-cert TLS handshake completes (endpoint holds the exported cert's private key). The forwarder and Bridge process are alive and genuine.
  timestamp: 2026-07-31

- hypothesis: "H4 — combined vs split address mode rejects the tried address forms (cause of the current error)"
  evidence: bridge CheckAuth validates the password (base64url decode + BridgePass compare) BEFORE any address matching; with the stored non-base64url password, address-mode logic is unreachable. Mode can only matter after the password is corrected (noted as a follow-up check, not the cause).
  timestamp: 2026-07-31

- hypothesis: "03-01's standing explanation — the Bridge USERNAME is wrong (credential-entry error)"
  evidence: gluon returns "no such user" for any rejected (username, password) pair, and bridge checks the password first; the user exhausting correct address + aliases with identical results is exactly what a wrong password produces. The username value in .env is a clean, well-formed address with no whitespace/quote artifacts.
  timestamp: 2026-07-31

## Evidence
<!-- APPEND only - facts discovered -->

- timestamp: 2026-07-31 (phase 0)
  checked: .planning/debug/knowledge-base.md and resolved/ sessions
  found: knowledge-base.md does not exist; only resolved session is UI-tooltip-related
  implication: no known-pattern candidate; proceed with fresh hypotheses

- timestamp: 2026-07-31
  checked: 03-01-SUMMARY.md live-environment finding
  found: Real Bridge at monroe:1143, STARTTLS, pinned cert. TLS handshake succeeded; failure precisely at LOGIN ("no such user", then "too many login attempts" after rate limiting). Summary also records the stored Bridge password "contains a literal double-quote character" (it broke double-quoted TOML config).
  implication: Endpoint holds the private key matching the exported pinned cert, so a real Proton Bridge is being reached; failure is at auth, not transport. The double-quote-in-password note is a red flag (see below).

- timestamp: 2026-07-31
  checked: plugins/proton/client.go connect() (lines 196-208) and live_bridge_test.go liveDialLiveIT (line 158)
  found: Both paths pass username/password VERBATIM to go-imap v1's conn.Login — no trimming, lowercasing, or re-encoding. go-imap v1 emits IMAP astrings with correct quoting/literals for special chars. The "no such user" text is the server's own NO response, wrapped with %v at client.go:205 / test line 160. Health, Match/Fetch, and the live test all share the same login semantics (env → verbatim → LOGIN).
  implication: H3 (code-side username mangling) has no mechanism in the plugin code itself; the string originates Bridge-side.

- timestamp: 2026-07-31
  checked: kernel/config/config.go expandEnv + config.example.toml [sources.proton]
  found: Raw-text ${VAR} expansion happens BEFORE TOML parsing; username="${PROTON_BRIDGE_USER}", token="${PROTON_BRIDGE_PASS}". Example config documents username as the value Bridge -> Settings displays.
  implication: kernel path and test path both source the same .env values; a bad .env value poisons both (explains UI red dot AND test failure identically).

- timestamp: 2026-07-31
  checked: /home/darren/projects/davison/webspaces/.env value SHAPES (secrets redacted, never printed)
  found: PROTON_BRIDGE_ADDR=monroe:1143. PROTON_BRIDGE_USER is a plain lowercase email at @davisononline.org, len 24, no whitespace/quotes — clean. PROTON_BRIDGE_PASS is len=37 with special chars ! " # $ % & ' ( + / ; < = > } — fourteen distinct symbol characters including both quote types.
  implication: A genuine Proton Bridge app password is base64 of random bytes (~20-22 chars, [A-Za-z0-9+/] only — never quotes, never '!', ';', '}'). A 37-char symbol-heavy string is the SHAPE of a human/password-manager account password, NOT a Bridge-generated password. Strong support for H1: wrong CLASS of password is stored.

- timestamp: 2026-07-31
  checked: Live read-only probe of monroe:1143 (greeting only — no commands sent, no login attempted)
  found: '* OK [CAPABILITY AUTH=PLAIN ID IDLE IMAP4rev1 MOVE STARTTLS UIDPLUS UNSELECT] Proton Mail Bridge 03.25.00 - gluon session ID 137'
  implication: The LAN forwarder works and the endpoint is a genuine, live Proton Mail Bridge v3.25.0 (gluon IMAP backend) in STARTTLS mode. Eliminates "wrong service / wrong port / dead forwarder" entirely.

- timestamp: 2026-07-31
  checked: Upstream gluon source — github.com/ProtonMail/gluon internal/backend/backend.go getUserID + internal/backend/errors.go
  found: |
    getUserID iterates ALL registered users calling connector.Authorize(ctx, username, password); if none authorize it returns ErrNoSuchUser = errors.New("no such user") — the SINGLE error for every auth failure. After maxLoginAttempts it returns ErrLoginBlocked instead ("too many login attempts" — exactly the second error 03-01 observed).
  implication: "no such user" does NOT mean "username unknown". It means "(username, password) pair rejected by every signed-in account". A correct username with a wrong password produces exactly this error. The 03-01/test-hint framing ("re-verify the Bridge username") was a misreading of the error's semantics.

- timestamp: 2026-07-31
  checked: Upstream proton-bridge source — internal/services/imapservice/connector.go Authorize (line 186) + internal/services/useridentity/state.go CheckAuth (line 230) + pkg/algo/encode.go B64RawEncode/B64RawDecode (lines 31-50)
  found: |
    CheckAuth order of operations: (1) algo.B64RawDecode(password) — base64.RawURLEncoding, alphabet [A-Za-z0-9-_], errors on any other character; (2) constant-time compare against the vault BridgePass bytes; (3) ONLY THEN case-insensitive match of the presented email against the account's enabled addresses. B64RawEncode also proves the Bridge UI's displayed password is pure base64url.
  implication: |
    Two decisive consequences:
    (a) The stored 37-char password containing " ! # $ % & ' ( + / ; < = > } is PROVABLY not a Bridge-generated password — 11+ of its characters are outside the base64url alphabet Bridge uses to render its password.
    (b) With that stored password, step (1) fails with CorruptInputError on every login, so the username is NEVER EVEN EXAMINED. Trying the correct address and every alias necessarily yields the identical "no such user" — precisely the reported symptom.

- timestamp: 2026-07-31
  checked: ~/.config/webspaces/config.toml (token values redacted)
  found: "[sources.proton] token = '${PROTON_BRIDGE_PASS}' (single-quoted literal, per 03-01's TOML workaround), username = \"${PROTON_BRIDGE_USER}\", base_url = \"imap://${PROTON_BRIDGE_ADDR}\""
  implication: The kernel health-check path (UI red dot) expands the SAME corrupted .env password the live test uses — one root cause explains both symptoms. Side note: 03-01's "password contains a literal double-quote broke TOML" finding was itself a symptom of this root cause (a real Bridge password can never contain a quote), not an independent hazard occurrence.

## Resolution
<!-- OVERWRITE as understanding evolves -->

root_cause: |
  The value stored in PROTON_BRIDGE_USER's companion secret PROTON_BRIDGE_PASS (.env line 7, consumed identically by the kernel via ~/.config/webspaces/config.toml's token = '${PROTON_BRIDGE_PASS}' and by the live test) is not the Proton Mail Bridge app password — it is a 37-character symbol-heavy string (contains " ! # $ % & ' ( + / ; < = > }) whose character set is impossible for a Bridge-generated password: Bridge renders its password with base64.RawURLEncoding ([A-Za-z0-9-_] only; proton-bridge pkg/algo/encode.go). On every LOGIN, bridge's CheckAuth (internal/services/useridentity/state.go:230) base64url-decodes the presented password FIRST — which always fails for this value — before ever comparing the email/username; gluon's getUserID (internal/backend/backend.go) then reports its single auth-failure error "no such user" (internal/backend/errors.go). Therefore the correct address and every alias all return the identical "no such user", which was misread since 03-01 as a username problem. Not a code defect: plugins/proton passes credentials verbatim (client.go:196-208), and monroe:1143 is a live genuine Proton Mail Bridge 3.25.0 (greeting probe).
fix: |
  (diagnose-only session — no fix applied) Environmental: replace PROTON_BRIDGE_PASS in /home/darren/projects/davison/webspaces/.env with the password Bridge itself displays (Bridge UI on monroe → account → Mailbox details / IMAP password, or `protonmail-bridge --cli` → `login`/`info`); it will be ~20-22 chars of [A-Za-z0-9-_] only. Username: exactly the address Bridge displays. Wait out Bridge's login jail ("too many login attempts") before retrying. Secondary user checks if it still fails: (1) account signed-in state on monroe's Bridge (signed-out account ⇒ same error), (2) combined vs split address mode (split mode restricts which address authenticates). Suggested hardening for gap closure: correct the misleading hint at plugins/proton/live_bridge_test.go:160 and the 03-01-SUMMARY framing; optionally have Health/plugin warn when the configured token contains characters outside base64url (self-diagnosing misconfig).
verification: n/a (find_root_cause_only)
files_changed: []
