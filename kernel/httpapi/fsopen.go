package httpapi

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
)

// Opener triggers the desktop's own handler for path — a function type so
// fsopen_test.go can inject a stub that records the path it was handed
// instead of a real xdg-open invocation (12-01-PLAN.md Task 2, D-06).
type Opener func(ctx context.Context, path string) error

// newXDGOpener returns the production Opener: starts a fixed "xdg-open"
// binary literal with the resolved path as its only argument (T-12-01 —
// the binary name is never derived from config or the request, and no
// shell is involved), and never blocks the HTTP response on the child's
// own exit (T-12-05: the child is reaped by a background wait). The
// returned closure's context parameter is deliberately the blank
// identifier: this implementation structurally cannot bind the child to
// any caller-supplied context, because an HTTP request's context is
// cancelled by net/http essentially synchronously with the handler
// returning — and FilesystemOpenHandler returns within microseconds of
// starting the child. Binding the child's lifetime to that context
// (exec.CommandContext) meant the desktop handler was routinely SIGKILLed
// moments after launch while the kernel reported opened: true (CR-01,
// 12-06-PLAN.md Task 2). The child must outlive the request that started
// it, so it is built with the plain two-argument exec.Command form instead.
func newXDGOpener(logger hclog.Logger) Opener {
	return func(_ context.Context, path string) error {
		cmd := exec.Command("xdg-open", path)
		if err := cmd.Start(); err != nil {
			return err
		}
		go func() {
			if err := cmd.Wait(); err != nil {
				logger.Warn("filesystem open: xdg-open exited with error", "path", path, "error", err.Error())
			}
		}()
		return nil
	}
}

type fsopenResponse struct {
	SchemaVersion int  `json:"schema_version"`
	Opened        bool `json:"opened"`
}

// resolveRoot mirrors plugins/filesystem/item.go's resolveRoot (that
// package is a separate Go module and cannot be imported here, so the
// helper is a hand-kept twin): resolves root's real, symlink-free path with
// filepath.EvalSymlinks, falling back to the lexically cleaned root when
// resolution fails (a root that does not exist yet, or is unreadable) — the
// containment comparison in FilesystemOpenHandler must still have a value
// to compare against, and the lexical clean is the conservative one. D-01's
// relative-path source_id is what makes the join meaningful in the first
// place — resolveRoot only ever resolves the CONFIGURED root, never the
// joined path.
func resolveRoot(root string) string {
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return filepath.Clean(root)
	}
	return resolved
}

// FilesystemOpenHandler serves POST /api/items/{id}/open (D-06, T-12-02,
// T-12-03, T-12-06): resolves the target absolute path exclusively from
// the indexed item's own source_id plus its source's configured Path —
// nothing read off the request body or query string ever reaches opener —
// re-validates the joined path is still inside the configured root, and
// execs it via opener. The re-validation resolves symlinks with
// filepath.EvalSymlinks on both the joined path and the configured root,
// compares the RESOLVED pair, and fails closed when resolution is
// impossible (CR-02, 12-06-PLAN.md Task 1) — so a file indexed legitimately
// and later swapped on disk for a symlink pointing outside the root is
// refused rather than followed, before opener is ever called. opener is
// handed the resolved, symlink-free path — the same path the containment
// check approved, not the lexical one (WR-02, 12-07-PLAN.md Task 2) — so a
// user whose configured root is a symlink sees the resolved location in
// their desktop application's own UI, which is deliberate. Registered on
// /api only (routes.go) — never on the /agent/v1 mirror.
func FilesystemOpenHandler(store *index.Store, cfgStore *config.Store, opener Opener, logger hclog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := itemIDParam(r)
		ctx := r.Context()

		it, ok, err := store.GetItem(ctx, id)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if !ok {
			WriteError(w, http.StatusNotFound, "item_not_found", "item \""+id+"\" was not found in the index")
			return
		}
		// The item's DeepLink must carry the file:// scheme before anything
		// else is resolved — this is what keeps this route from being
		// reachable for a non-filesystem item (T-12-03), keyed on the URL
		// scheme alone, never it.SourceType (12-RESEARCH.md's "no built-in
		// table of known plugin types" discipline).
		if !strings.HasPrefix(it.DeepLink, "file://") {
			WriteError(w, http.StatusNotFound, "item_not_found", "item \""+id+"\" has no filesystem-openable deep link")
			return
		}

		src, ok := cfgStore.Expanded().Sources[it.Source]
		if !ok || strings.TrimSpace(src.Path) == "" {
			WriteError(w, http.StatusNotFound, "item_not_found", "source \""+it.Source+"\" has no configured local path")
			return
		}

		root, err := expandHomeAndAbs(src.Path)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		// full is joined from the INDEX's own SourceID and the configured
		// root — never from anything in the request (T-12-02, T-12-06).
		full := filepath.Join(root, filepath.FromSlash(it.SourceID))
		if full != root && !strings.HasPrefix(full, root+string(filepath.Separator)) {
			WriteError(w, http.StatusBadRequest, "invalid_path", "resolved path escapes source root")
			return
		}

		// Second containment check against the RESOLVED path and the
		// RESOLVED root (CR-02): a file indexed legitimately and later
		// swapped on disk for a symlink pointing outside the root is
		// refused here, before opener is ever called. A resolution
		// failure that is itself a vanished file (fs.ErrNotExist) answers
		// item_not_found honestly rather than as a containment violation;
		// every other resolution failure, and a resolved path that falls
		// outside the resolved root, answers invalid_path — no new error
		// code, matching today's message for the lexical case.
		resolved, err := filepath.EvalSymlinks(full)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				WriteError(w, http.StatusNotFound, "item_not_found", "item \""+id+"\" no longer exists at its indexed path")
				return
			}
			WriteError(w, http.StatusBadRequest, "invalid_path", "resolved path escapes source root")
			return
		}
		resolvedRoot := resolveRoot(root)
		if resolved != resolvedRoot && !strings.HasPrefix(resolved, resolvedRoot+string(filepath.Separator)) {
			WriteError(w, http.StatusBadRequest, "invalid_path", "resolved path escapes source root")
			return
		}

		// The opener is handed a context detached from the request's own
		// (CR-01): it still carries the request's values but can never be
		// cancelled by the request completing, since the desktop handler
		// must outlive the HTTP response that launched it. This is
		// belt-and-braces with newXDGOpener's own structural inability to
		// bind to a caller context — it is the load-bearing half for the
		// seam, provable at the stubbed-Opener boundary, and protects any
		// future Opener that DOES honour its context. Every other use of
		// ctx in this handler (store.GetItem) stays as-is: index reads
		// should still be cancelled when the requester goes away.
		//
		// opener is handed resolved, not full (WR-02, 12-07-PLAN.md Task 2):
		// the path validated by the containment check above and the path
		// handed to the desktop handler must be one and the same. A user
		// whose configured root is a symlink will therefore see the
		// resolved location in their desktop application's own UI, which is
		// deliberate.
		if err := opener(context.WithoutCancel(ctx), resolved); err != nil {
			WriteError(w, http.StatusBadGateway, "open_failed", err.Error())
			return
		}

		WriteJSON(w, http.StatusOK, fsopenResponse{SchemaVersion: schemaVersion, Opened: true})
	}
}

// expandHomeAndAbs expands a leading "~" in path (the kernel deliberately
// stores Source.Path unexpanded — see kernel/config/types.go's Path doc
// comment) and returns its absolute, cleaned form.
func expandHomeAndAbs(path string) (string, error) {
	expanded := path
	if strings.HasPrefix(path, "~") {
		u, err := user.Current()
		if err != nil {
			return "", err
		}
		expanded = strings.Replace(path, "~", u.HomeDir, 1)
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}
