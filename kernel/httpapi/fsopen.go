package httpapi

import (
	"context"
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
// own exit (T-12-05: the child is reaped by a background wait).
func newXDGOpener(logger hclog.Logger) Opener {
	return func(ctx context.Context, path string) error {
		cmd := exec.CommandContext(ctx, "xdg-open", path)
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

// FilesystemOpenHandler serves POST /api/items/{id}/open (D-06, T-12-02,
// T-12-03, T-12-06): resolves the target absolute path exclusively from
// the indexed item's own source_id plus its source's configured Path —
// nothing read off the request body or query string ever reaches opener —
// re-validates the joined path is still inside the configured root, and
// execs it via opener. Registered on /api only (routes.go) — never on the
// /agent/v1 mirror.
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

		if err := opener(ctx, full); err != nil {
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
