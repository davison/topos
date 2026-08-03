# API Coverage — freedesktop Secret Service (`org.freedesktop.secrets`)

> Full coverage by default. Opt-outs are explicit, reasoned decisions.

The `api-coverage` detector fired on Phase 4 because the Signal plugin integrates one
external service API: the **freedesktop Secret Service D-Bus API**, used to unwrap the
Electron `safeStorage` master password when Signal Desktop's `config.json` carries the
`encryptedKey`/`safeStorageBackend` shape (04-RESEARCH.md Pattern 1, Standard Stack).

Two other integrations in this phase are deliberately **not** API surfaces and carry no
matrix: SQLCipher (a linked C library, not a service) and Signal Desktop's local
`db.sqlite` (a data store read through that library, not an API). Signal Desktop's
`sgnl://` scheme handler is an OS-level registration consumed by the desktop, not an API
this plugin calls — its single capability is covered by the deep-link row below.

The capability surface enumerated below is the `org.freedesktop.Secret.Service`,
`org.freedesktop.Secret.Collection`, `org.freedesktop.Secret.Item` and
`org.freedesktop.Secret.Prompt` interfaces.

| capability | decision | reason |
|---|---|---|
| `Service.OpenSession` (`dh-ietf1024-sha256-aes128-cbc-pkcs7`) | INTEGRATE | |
| `Service.OpenSession` (`plain`) | OPT-OUT | the plain, unencrypted session mode transmits the secret over the session bus in the clear; only the encrypted DH mode is used |
| `Service.SearchItems` | INTEGRATE | |
| `Service.Unlock` | INTEGRATE | |
| `Service.ReadAlias` (resolve the `default` collection) | INTEGRATE | |
| `Item.GetSecret` | INTEGRATE | |
| `Prompt.Prompt` / `Prompt.Completed` (handle a locked-collection prompt) | INTEGRATE | |
| `Service.Lock` | OPT-OUT | the plugin never changes the user's keyring lock state; locking a collection the user left unlocked is a side effect on their session, and this project is read-only against every external system |
| `Service.GetSecrets` (batch) | OPT-OUT | exactly one secret is ever retrieved (the Electron `safeStorage` master password); the batch form has no caller |
| `Service.CreateCollection` | OPT-OUT | write path — the plugin never creates keyring state (read-only project constraint) |
| `Service.SetAlias` | OPT-OUT | write path — the plugin never rebinds the user's collection aliases |
| `Collection.CreateItem` | OPT-OUT | write path — the plugin never stores anything in the user's keyring |
| `Collection.SearchItems` (per-collection) | OPT-OUT | `Service.SearchItems` already searches every unlocked collection; the per-collection form adds no reachable secret |
| `Collection.Delete` | OPT-OUT | write path — destructive, never invoked |
| `Item.SetSecret` | OPT-OUT | write path — the plugin never rewrites the stored master password |
| `Item.Delete` | OPT-OUT | write path — destructive, never invoked |
| `Item` property writes (`Label`, `Attributes`) | OPT-OUT | write path — item metadata is read for matching only, never modified |
| Signals (`CollectionCreated`/`Changed`/`Deleted`, `ItemCreated`/`Changed`/`Deleted`) | OPT-OUT | the plugin holds no long-lived keyring subscription; the key is resolved on demand per sync, so there is no cached state a change signal would invalidate |
| Native KWallet D-Bus API (`org.kde.kwalletd5`/`kwalletd6`) | OPT-OUT | out of scope per the locked stack (`.claude/CLAUDE.md`): KWallet has exposed the Secret Service API since KDE Frameworks 5.97, so a modern KDE session is served by the rows above. Sessions on older Frameworks are a documented, accepted gap (04-RESEARCH.md Open Question 2) |
| Signal Desktop `sgnl://` scheme handler (deep link) | INTEGRATE | |
