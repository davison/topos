# Phase 12: Filesystem Source - Pattern Map

**Mapped:** 2026-08-13
**Files analyzed:** 13 (new) + 2 (modified)
**Analogs found:** 15 / 15

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `plugins/filesystem/go.mod` | config | — | `plugins/signal/go.mod` | exact (local-path plugin, own module) |
| `plugins/filesystem/main.go` | config/bootstrap | request-response (subprocess launch) | `plugins/signal/main.go` | exact (Path-only `sourceConfig`, no base_url/token) |
| `plugins/filesystem/plugin.go` | service (SourcePlugin impl) | CRUD (Match/Fetch/Health) | `plugins/paperless/plugin.go` | role-match (ContentVariant switch shape); `plugins/silverbullet/plugin.go` for Extras-field Describe |
| `plugins/filesystem/scope.go` | utility (classifier) | transform | *(no analog — new)* | none — see "No Analog Found" |
| `plugins/filesystem/walk.go` | service (directory walk) | batch | *(no analog — new)*, informed by `plugins/signal/match.go`'s Match-time full-set-build shape | partial |
| `plugins/filesystem/classify.go` | utility | transform | `plugins/paperless/plugin.go`'s `noRenditionReason`/rendition dispatch idiom | partial |
| `plugins/filesystem/render.go` | utility (markdown render) | transform | `plugins/silverbullet/render.go` | exact |
| `plugins/filesystem/deeplink.go` | utility | transform | `plugins/signal/deeplink.go` | role-match (deep-link builder, different scheme) |
| `plugins/filesystem/readonly_test.go` | test (AST guard) | — | `plugins/signal/readonly_test.go` | exact pattern (adapt selector set to `os` package) |
| `plugins/filesystem/*_test.go` | test | — | `plugins/paperless/fetch_test.go`, `plugins/silverbullet/match_test.go` | role-match |
| `kernel/httpapi/item.go` (modify) | controller/config | request-response | itself — add one map entry | exact (existing file, in-place edit) |
| `kernel/httpapi/fsopen.go` | controller (new route) | request-response (exec side-effect) | `kernel/httpapi/whatsapplink.go` | role-match (server-side path resolution + `exec.CommandContext`, much simpler lifecycle) |
| `kernel/httpapi/stream.go` (modify) | controller (serializer) | transform | itself — add scheme-keyed rewrite before existing `Link{URL: it.DeepLink}` construction | exact (existing file, in-place edit) |
| `kernel/httpapi/routes.go` (modify) | route registration | — | existing route registrations for `whatsapplink.go` handlers | exact |
| `web/e2e/specs/*.spec.ts` (new fs source spec) | test (e2e) | — | Phase 11 e2e specs using `web/e2e/fixtures/plugin-binaries.ts`, `config-builder.ts` | exact |

## Pattern Assignments

### `plugins/filesystem/go.mod` (config)

**Analog:** `plugins/signal/go.mod`

```
module github.com/davison/topos/plugins/signal

go 1.25.0

require github.com/mattn/go-sqlite3 v1.14.49
```

Copy the module-declaration shape (`module github.com/davison/topos/plugins/filesystem`, `go 1.25.0`), but this plugin has **no cgo** and **no replace directive** — closer to `plugins/silverbullet/go.mod`'s plain-require shape for that reason. Add:
```
require github.com/yuin/goldmark v1.8.5   // matches plugins/silverbullet/go.mod:8 exactly — reuse the same pinned version
require github.com/bmatcuk/doublestar/v4 v4.10.0   // gated behind the RESEARCH.md checkpoint:human-verify task
```

---

### `plugins/filesystem/main.go` (bootstrap)

**Analog:** `plugins/signal/main.go` (full file read, 94 lines)

**sourceConfig + WEBSPACES_SOURCE_CONFIG read pattern** (lines 33-54):
```go
type sourceConfig struct {
	Path string `json:"path"`
	// NEW for filesystem: Recursive bool `json:"recursive"` (RESEARCH.md
	// Open Question 2 — typed config.Source field, not extras)
}

func main() {
	raw := os.Getenv("WEBSPACES_SOURCE_CONFIG")
	if raw == "" {
		fatal(fmt.Errorf("WEBSPACES_SOURCE_CONFIG is not set"))
	}
	var cfg sourceConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		fatal(fmt.Errorf("parse WEBSPACES_SOURCE_CONFIG: %w", err))
	}
	if cfg.Path == "" {
		fatal(fmt.Errorf("WEBSPACES_SOURCE_CONFIG: path is empty"))
	}
	configDir, err := expandHome(cfg.Path)
	...
}
```

**goplugin.Serve wiring** (lines 56-71) — copy verbatim, same `sdk.Handshake` / `sdk.SourcePluginGRPCPlugin` / `sdk.GRPCServer` (raised message-size ceiling) shape:
```go
goplugin.Serve(&goplugin.ServeConfig{
	HandshakeConfig: sdk.Handshake,
	Plugins: map[string]goplugin.Plugin{
		"source": &sdk.SourcePluginGRPCPlugin{Impl: impl},
	},
	GRPCServer: sdk.GRPCServer,
})
```

**Home-expansion helper** (lines 79-88) — copy `expandHome` verbatim; filesystem source paths are just as likely to carry a leading `~` as Signal's config dir.

---

### `plugins/filesystem/plugin.go` (SourcePlugin impl)

**Analogs:** `plugins/paperless/plugin.go` (Fetch/ContentVariant dispatch, full 247 lines read) + `plugins/silverbullet/plugin.go` (Describe/Extras shape, lines 1-60 read)

**Describe + Extras pattern** (paperless lines 67-76, silverbullet lines 1-45 for the `ExtrasField`/icon shape):
```go
var matchVocabulary = []string{"folders"} // D-05

func (p *SourcePlugin) Describe(_ context.Context, _ *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error) {
	return &toposv1.DescribeResponse{
		SourceType:      "filesystem",
		DisplayName:     "Filesystem folder",
		ContractVersion: "topos.v2",
		MatchVocabulary: matchVocabulary,
		Extras: []*toposv1.ExtrasField{
			{Key: "include_glob", Label: "Include glob (comma-separated)", Required: false, Secret: false, Placeholder: "**/*.pdf,**/*.md"},
			{Key: "exclude_glob", Label: "Exclude glob (comma-separated)", Required: false, Secret: false, Placeholder: "**/node_modules/**"},
		},
	}, nil
}
```

**Fetch dispatch pattern** (paperless lines 164-180) — copy the switch-on-`ContentVariant` shape exactly, substituting per-branch bodies for the classify-then-fetch logic:
```go
func (p *SourcePlugin) Fetch(ctx context.Context, req *toposv1.FetchRequest) (*toposv1.FetchResponse, error) {
	switch req.GetVariant() {
	case toposv1.ContentVariant_CONTENT_VARIANT_FULL:
		return p.fetchFull(ctx, req.GetSourceId())
	case toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW:
		return p.fetchPreview(ctx, req.GetSourceId())
	case toposv1.ContentVariant_CONTENT_VARIANT_THUMBNAIL:
		return &toposv1.FetchResponse{Available: false, UnavailableReason: "no thumbnail rendition"}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "filesystem: unspecified content variant")
	}
}
```

**"Unavailable, not error" pattern for unsupported types** (paperless lines 152-156, `noRenditionReason` const): office-format files should mirror this exactly —
```go
const noRenditionReason = "preview not supported for this file type; open in source"
// ... return &toposv1.FetchResponse{Available: false, UnavailableReason: noRenditionReason}, nil
```

**Item construction / DeepLink / Provenance shape** (paperless lines 109-138, `toItem`) — same struct-literal shape; `DeepLink` here is the `file://`-scheme URI per RESEARCH.md's Pattern 3 recommendation (not an `http://...` URL like paperless's), and `Fidelity` is `LINK_FIDELITY_EXACT` or `ANCHORED` per file kind rather than always `EXACT`.

**Error wrapping idiom** (paperless throughout): `status.Errorf(codes.Unavailable, "filesystem: %s: %v", op, err)` / `status.Errorf(codes.NotFound, "filesystem: item %q not found", relPath)` — copy the `codes.Unavailable` for I/O failures, `codes.NotFound` for missing files, `codes.InvalidArgument` for malformed requests convention verbatim.

---

### `plugins/filesystem/render.go` (markdown rendering)

**Analog:** `plugins/silverbullet/render.go` (full file, 37 lines) — **copy near-verbatim**, only the doc comment's plugin name changes:
```go
var mdConverter = goldmark.New() // package-init singleton, safe for concurrent use, defaults-only (no raw-HTML passthrough)

func RenderMarkdown(markdown []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := mdConverter.Convert(markdown, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil // UNSANITIZED — kernel's bluemonday layer (kernel/httpapi/rendition.go) sanitizes on serve
}
```
Note the load-bearing comment this file carries: goldmark stays at defaults, sanitization is the kernel's job, not this plugin's.

---

### `plugins/filesystem/readonly_test.go` (AST write-guard)

**Analog:** `plugins/signal/readonly_test.go` (full file, 157 lines) — same `filepath.WalkDir(".", ...)` + `ast.Inspect` idiom, same negative-control-fixture discipline (`scanSourceForWriteShapedSQL`-equivalent). **Swap the selector/substring vocabulary** from SQL verbs to `os`-package write selectors, per RESEARCH.md's own worked example:
```go
var disallowedOSSelectors = map[string]bool{
	"WriteFile": true, "Remove": true, "RemoveAll": true, "Create": true,
	"OpenFile": true, "Rename": true, "Mkdir": true, "MkdirAll": true,
	"Chmod": true, "Chown": true, "Truncate": true, "Symlink": true, "Link": true,
}
```
Keep the negative-control fixtures (`TestPluginIssuesNoWrite...`'s two fixture strings proving the scanner isn't vacuous) — signal's `fixtureExec`/`fixtureSQLLiteral` pattern (lines 73-91) is the template; write one fixture calling `os.Remove(...)` and confirm the scan flags it.

---

### `kernel/httpapi/item.go` (modify — one-line addition)

**Analog:** itself, in place. Current state (lines 33-51, full block read):
```go
var allowedRenditionTypes = map[string]bool{
	"application/pdf": true,
	"image/png":       true,
	"image/jpeg":      true,
	"image/gif":       true,
	"image/webp":      true,
	"text/html": true,
}
```
Add `"text/plain": true,` following the same inline-comment convention the `text/html` entry uses (explain *why* the entry was added and *when*, citing this phase and D-04).

---

### `kernel/httpapi/fsopen.go` (new loopback open route)

**Analog:** `kernel/httpapi/whatsapplink.go` (imports + `binPath` resolution discipline read, lines 1-52, 139-170, 690-720)

**Imports pattern** (whatsapplink.go lines 1-20):
```go
import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
)
```

**"Resolve identity server-side, never trust the request" discipline** (whatsapplink.go lines 701-705, `binPath` comment + resolution): the same discipline — resolve the real path from **trusted server-side state** (the index row's `SourceID`, joined against the configured source `Path`), never from `chi.URLParam`/request body directly, then defense-in-depth re-validate:
```go
func FilesystemOpenHandler(store *index.Store, cfgStore *config.Store, logger hclog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		it, ok, err := store.GetItem(r.Context(), id)
		if err != nil || !ok {
			WriteError(w, http.StatusNotFound, "item_not_found", "item not found")
			return
		}
		src, ok := cfgStore.Expanded().Sources[it.Source]
		if !ok || src.Path == "" {
			WriteError(w, http.StatusNotFound, "item_not_found", "source has no local path configured")
			return
		}
		root, err := filepath.Abs(src.Path)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		full := filepath.Join(root, it.SourceID)
		if !strings.HasPrefix(full, root+string(filepath.Separator)) {
			WriteError(w, http.StatusBadRequest, "invalid_path", "resolved path escapes source root")
			return
		}
		cmd := exec.Command("xdg-open", full) // fixed binary name literal, mirrors whatsapplink.go's binPath discipline
		if err := cmd.Start(); err != nil {
			WriteError(w, http.StatusBadGateway, "open_failed", err.Error())
			return
		}
		go func() { _ = cmd.Wait() }() // fire-and-forget, never block the response — mirrors whatsapplink.go's async subprocess handling
		WriteJSON(w, http.StatusOK, map[string]bool{"opened": true})
	}
}
```
Unlike `whatsapplink.go`'s `linkSpawner` (session lifecycle: stdout streaming, kill func, `Suspender` coordination), this route needs **none** of that ceremony — no session, no poll, no kill. Do not copy the `linkSpawnResult`/channel machinery; it is overkill for a single fire-and-forget exec.

---

### `kernel/httpapi/stream.go` (modify — scheme-keyed DeepLink rewrite)

**Analog:** itself, in place (lines 180-206, `toStreamItem` read in full).

Current: `Link: link{URL: it.DeepLink, Fidelity: string(it.Fidelity)},` — echoes `DeepLink` verbatim.

Add a small helper, called before this line, that rewrites `file://`-scheme deep links to the new loopback route — **keyed off URL scheme, never `it.SourceType`** (RESEARCH.md's explicit anti-pattern: no built-in table of plugin types):
```go
func resolveStreamLinkURL(it item.Item) string {
	if strings.HasPrefix(it.DeepLink, "file://") {
		return "/api/items/" + it.ID + "/open"
	}
	return it.DeepLink
}
// ...
Link: link{URL: resolveStreamLinkURL(it), Fidelity: string(it.Fidelity)},
```

---

## Shared Patterns

### Local-path (`Path`-only) source config — no `base_url`/`token`
**Source:** `plugins/signal/main.go` lines 23-54, `kernel/config/config.go:339` (`Validate`'s local-path branch — verified by RESEARCH.md, not independently re-read this session)
**Apply to:** `plugins/filesystem/main.go` — copy the `sourceConfig{ Path string }` shape and the `WEBSPACES_SOURCE_CONFIG` unmarshal/validate sequence directly; add a `Recursive bool` field per RESEARCH.md Open Question 2's recommendation.

### ContentVariant switch dispatch in Fetch
**Source:** `plugins/paperless/plugin.go` lines 164-180
**Apply to:** `plugins/filesystem/plugin.go`'s `Fetch` — identical three-branch switch shape (FULL/PREVIEW/THUMBNAIL), classify-then-branch bodies.

### AST-walk write-guard test (PLUG-02 mechanical enforcement)
**Source:** `plugins/signal/readonly_test.go` (full file) — `filepath.WalkDir` + `ast.Inspect` + negative-control fixtures
**Apply to:** `plugins/filesystem/readonly_test.go` — swap SQL-verb vocabulary for `os`-package write-selector vocabulary; keep the negative-control-fixture discipline so the scanner's own correctness is tested, not assumed.

### "Unavailable, not error" for unsupported content
**Source:** `plugins/paperless/plugin.go` lines 152-156, 224-226 (`noRenditionReason`)
**Apply to:** `plugins/filesystem/plugin.go`'s office-format branch of `fetchPreview` — return `FetchResponse{Available: false, UnavailableReason: "..."}`, never a gRPC error, for a normal "this file type has no inline preview" outcome.

### "Resolve identity from trusted server-side state, never the request" for exec surfaces
**Source:** `kernel/httpapi/whatsapplink.go` lines 690-705 (`binPath` resolution discipline)
**Apply to:** `kernel/httpapi/fsopen.go` — the item id comes from the URL, but the **path** that reaches `exec.Command` is resolved exclusively from the index row + configured source root, with a lexical-prefix re-check as defense-in-depth. Never accept a path in the request body/query.

### Goldmark markdown rendering, defaults-only, kernel sanitizes
**Source:** `plugins/silverbullet/render.go` (full file)
**Apply to:** `plugins/filesystem/render.go` — copy near-verbatim; do not enable any "unsafe HTML" goldmark extension; sanitization stays the kernel's job (`kernel/httpapi/rendition.go`).

## No Analog Found

Files with no close match in the codebase — genuinely new machinery this phase introduces (planner should follow RESEARCH.md's Architecture Patterns / Code Examples sections, which already give worked, line-cited designs for each):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `plugins/filesystem/scope.go` | utility (extras-driven glob scope resolution) | transform | No prior plugin does include/exclude glob-based scope narrowing — Phase 11's extras machinery exists, but no consumer has used it for file-scope filtering yet. Follow RESEARCH.md's Pattern 1 TOML example + `doublestar/v4` usage (gate behind the checkpoint:human-verify task). |
| `plugins/filesystem/walk.go` | service (stat-diff directory walk) | batch | No prior plugin walks an arbitrary directory tree; closest conceptual precedent (`plugins/signal/match.go`'s "build the complete current item set every Match call") is same *shape* but entirely different mechanism (SQL query vs. filesystem walk). Follow RESEARCH.md's Pitfall 3 guidance (`filepath.WalkDir`, don't descend into symlinked dirs, `SkipDir` on permission errors). |
| `plugins/filesystem/classify.go` | utility (extension → preview-kind map) | transform | No prior plugin has a fixed extension-to-MIME classifier; explicitly recommended as a hand-rolled `map[string]string`, not `mime.TypeByExtension` (RESEARCH.md Anti-Patterns). |
| `kernel/httpapi/fsopen.go` | controller (new exec route) | request-response + exec side-effect | Closest analog (`whatsapplink.go`) has a full session/poll/kill state machine this route deliberately does not need — see Pattern Assignment above for the trimmed-down design actually to copy. |

## Metadata

**Analog search scope:** `plugins/signal/`, `plugins/paperless/`, `plugins/silverbullet/`, `kernel/httpapi/` (item.go, whatsapplink.go, stream.go, routes.go), `kernel/correlate/correlate.go`, `kernel/config/config.go` (RESEARCH.md line citations reused directly, not re-read where already verified this session)
**Files scanned:** ~20 (full or targeted reads)
**Pattern extraction date:** 2026-08-13
