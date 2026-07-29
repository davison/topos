# API Coverage — IMAP (Proton Mail Bridge), via `github.com/emersion/go-imap` v1

> Full coverage by default. Opt-outs are explicit, reasoned decisions.

The automated detector reported `detected: false` for this phase's scope. This
matrix is produced anyway because the phase genuinely integrates an external
protocol surface (IMAP4rev1 against Proton Mail Bridge), and the decisive
subtraction — every mutating IMAP capability — is the phase's central guarantee
rather than an oversight. Recording it here makes that decision auditable instead
of implicit.

The capability surface below is the IMAP4rev1 command set as exposed by
`github.com/emersion/go-imap` v1's client, grouped by command.

| capability | decision | reason |
|---|---|---|
| CAPABILITY | INTEGRATE | |
| LOGIN / AUTHENTICATE | INTEGRATE | |
| STARTTLS | INTEGRATE | |
| LOGOUT | INTEGRATE | |
| NOOP | INTEGRATE | |
| LIST | INTEGRATE | |
| EXAMINE (read-only mailbox open) | INTEGRATE | |
| STATUS | INTEGRATE | |
| FETCH ENVELOPE | INTEGRATE | |
| FETCH INTERNALDATE | INTEGRATE | |
| FETCH UID | INTEGRATE | |
| FETCH FLAGS | INTEGRATE | |
| FETCH BODY.PEEK | INTEGRATE | |
| UID FETCH | INTEGRATE | |
| UID SEARCH | INTEGRATE | |
| SELECT (read-write mailbox open) | OPT-OUT | Read-only by construction (PLUG-02) — it lets the server accept mutating commands on the session. EXAMINE is used instead, enforced by an AST scan and a wire-transcript test. |
| FETCH BODY (non-peek) | OPT-OUT | A non-peek body fetch makes the server implicitly set the `\Seen` flag — the exact behaviour SRC-01's second success criterion forbids. BODY.PEEK is used unconditionally. |
| STORE / UID STORE | OPT-OUT | Mutates message flags in the user's mailbox — forbidden by PLUG-02 (plugins never mutate source data stores) and by SRC-01 criterion 2. |
| EXPUNGE | OPT-OUT | Destroys messages in the user's mailbox — forbidden by PLUG-02. |
| COPY / UID COPY | OPT-OUT | Mutates the user's mailbox — forbidden by PLUG-02. |
| MOVE / UID MOVE | OPT-OUT | Mutates the user's mailbox — forbidden by PLUG-02. |
| APPEND | OPT-OUT | Writes a new message into the user's mailbox — forbidden by PLUG-02. |
| CREATE / DELETE / RENAME mailbox | OPT-OUT | Mutates the user's mailbox structure — forbidden by PLUG-02. |
| SUBSCRIBE / UNSUBSCRIBE / LSUB | OPT-OUT | Mutates (or reads) server-side subscription state that webspaces has no reason to touch; keyword matching operates over the full LIST result, so subscription state is irrelevant. |
| SEARCH (non-UID, sequence-number form) | OPT-OUT | Not needed — every lookup that matters is by Message-ID, and the UID form is the stable one. Sequence numbers shift under concurrent mailbox activity. |
| IDLE (push notification of new mail) | OPT-OUT | Not needed: the kernel's scheduler and single-flight coordinator (KERN-04) already drive every source; a source-specific wake-up path would bypass that coordinator. |
| SORT / THREAD extensions | OPT-OUT | Not needed — ordering is the kernel's own total, stable chronological order over the local index (documented in `docs/api.md` "Ordering guarantee"), never the server's. |
| QUOTA | OPT-OUT | Not needed — webspaces never writes to the mailbox, so remaining quota carries no information it can act on. |
| COMPRESS | OPT-OUT | Not needed — the connection is LAN-local to the user's own home server. |
| NAMESPACE | OPT-OUT | Not needed — Proton Mail Bridge exposes a single personal namespace, and the LIST result is enumerated in full rather than navigated by namespace. |
