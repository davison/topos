package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/davison/topos/kernel/config"
)

// effectiveDateRange resolves the range a read honours (M3-R1, #70): the
// webspace's SAVED range, intersected with an optional live ?from=/?to=
// preview (calendar dates, from's day-start through to's day-end) — the
// params can only narrow further, never widen past what config saved,
// so the saved range stays the truth for every consumer while the UI
// previews an unsaved one. An unparseable param is a 400 by the caller.
func effectiveDateRange(r *http.Request, ws config.Webspace) (from, to int64, err error) {
	from, to = ws.DateRange()
	if raw := r.URL.Query().Get("from"); raw != "" {
		t, perr := time.ParseInLocation("2006-01-02", raw, time.Local)
		if perr != nil {
			return 0, 0, fmt.Errorf("from %q is not a calendar date (want YYYY-MM-DD)", raw)
		}
		if u := t.Unix(); u > from {
			from = u
		}
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		t, perr := time.ParseInLocation("2006-01-02", raw, time.Local)
		if perr != nil {
			return 0, 0, fmt.Errorf("to %q is not a calendar date (want YYYY-MM-DD)", raw)
		}
		if u := t.AddDate(0, 0, 1).Unix() - 1; to == 0 || u < to {
			to = u
		}
	}
	return from, to, nil
}
