// Package pluginhost discovers, launches, and supervises source plugin
// subprocesses over the go-plugin gRPC transport. It is the only package
// permitted to import hashicorp/go-plugin's client side; kernel/httpapi
// must never import this package (see kernel/httpapi/stream.go).
package pluginhost

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/sdk"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// ErrItemNotFound is returned by Host.Fetch when the plugin reports the
// requested source id does not exist (gRPC codes.NotFound), or when no
// launched plugin is registered for the item's source type.
var ErrItemNotFound = errors.New("pluginhost: item not found")

// ErrSourceUnavailable is returned by Host.Fetch when the plugin's live
// call to its source system failed transiently (gRPC codes.Unavailable,
// or any other transport-level failure calling the plugin).
var ErrSourceUnavailable = errors.New("pluginhost: source unavailable")

// MaxIconBytes is the size ceiling enforced on DescribeResponse.icon at
// capture time (09-01-PLAN.md Task 2, T-09-03): an icon larger than this
// is dropped — never truncated — and never fails the launch.
const MaxIconBytes = 65536

// allowedIconMIME is the kernel-side mime allowlist enforced on
// DescribeResponse.icon_mime at capture time (T-09-02): a plugin cannot
// choose the Content-Type the kernel later serves outside this set. A mime
// not in this set, or an empty mime paired with non-empty icon bytes, has
// its icon dropped and treated identically to "no icon declared."
var allowedIconMIME = map[string]bool{
	"image/svg+xml": true,
	"image/png":     true,
}

// captureIcon validates a Describe response's icon/icon_mime fields against
// MaxIconBytes and allowedIconMIME, returning ok=false whenever the icon is
// absent or fails validation — every failure is treated as "this plugin
// declared no icon," never as a launch failure (09-01-PLAN.md Task 2).
func captureIcon(desc *toposv1.DescribeResponse) (iconBytes []byte, iconMIME string, ok bool) {
	b := desc.GetIcon()
	m := desc.GetIconMime()

	if len(b) == 0 {
		return nil, "", false
	}
	if len(b) > MaxIconBytes {
		return nil, "", false
	}
	if m == "" {
		return nil, "", false
	}
	if !allowedIconMIME[m] {
		return nil, "", false
	}
	return b, m, true
}

// Plugin is one launched, handshaken source plugin subprocess.
type Plugin struct {
	name            string // config key under [sources.<name>] — THE instance identity (D-08)
	sourceType      string // learned via Describe, not trusted from the filename — plugin kind only, never identity
	pluginName      string // Describe-learned display name (e.g. "paperless-ngx") — the plugin KIND's own label
	displayName     string // resolved instance display name (config display_name, or name if unset) — D-09
	matchVocabulary []string
	// iconBytes/iconMIME are captured from the same Describe call launch()
	// already makes (no new RPC), validated by captureIcon. Both are the
	// zero value when the plugin declared no icon, declared one that
	// failed validation, or was built against the pre-Phase-9 contract
	// (09-01-PLAN.md Task 2).
	iconBytes []byte
	iconMIME  string
	// src is the config.Source this instance was launched with — Reconcile
	// (07-02-PLAN.md Task 1, D-06/D-07) compares a currently-configured
	// instance's config.Source against this value to decide whether the
	// instance is unchanged (subprocess left running untouched) or has a
	// different connection config (subprocess relaunched). Never used for
	// anything beyond that diff.
	src    config.Source
	client *goplugin.Client
	impl   sdk.SourcePlugin
	// tier is the provenance Tier this instance's binary resolved to at
	// launch time, via ResolveBinary (T-11-01) — set exactly once here
	// and never overwritten from anything the plugin process itself
	// later reports (Describe, Health, or any other RPC).
	tier Tier
}

// Name returns the config key this plugin was launched under (under
// [sources.<name>]) — the source instance's identity (D-08).
func (p *Plugin) Name() string { return p.name }

// SourceType returns the source_type learned from the plugin's own
// Describe RPC response — never trusted from the filename (T-01-07). This
// is the plugin KIND, never an identity key: two launched instances of one
// plugin binary share the same SourceType but always have distinct Name().
func (p *Plugin) SourceType() string { return p.sourceType }

// DisplayName returns this source INSTANCE's resolved display name (D-09):
// its config-authored [sources.<name>] display_name, or the instance id
// itself when that key is omitted. This is what the health/sources API and
// the UI show for this specific instance — distinct from PluginDisplayName,
// which names the plugin KIND regardless of how many instances of it are
// configured.
func (p *Plugin) DisplayName() string { return p.displayName }

// Tier returns this instance's launch-time provenance Tier — TierTrusted
// or TierExternal, set once at launch by ResolveBinary and never
// re-derived from anything the plugin process reports (T-11-01).
func (p *Plugin) Tier() Tier { return p.tier }

// PluginDisplayName returns the Describe-learned display name for the
// plugin KIND this instance runs (e.g. "paperless-ngx", "SilverBullet") —
// the same value every instance of this plugin binary reports, regardless
// of its own instance display name. Provenance/diagnostic use only; never
// a substitute for DisplayName in anything the UI renders as a source
// label (02-01-PLAN.md's sourceDisplayName fix is the local-mapping
// predecessor this makes obsolete).
func (p *Plugin) PluginDisplayName() string { return p.pluginName }

// MatchVocabulary returns the field-name vocabulary this plugin declared in
// its Describe response (DescribeResponse.match_vocabulary) — the set of
// keys correlate.matchFieldsFor may populate in a Match request sent to
// this instance. Satisfies correlate.Source.
func (p *Plugin) MatchVocabulary() []string { return p.matchVocabulary }

// Match calls the plugin's Match RPC, wrapping each declared field's value
// list in a StringList (proto3 map values cannot be repeated fields
// directly). Satisfies correlate.Source.
func (p *Plugin) Match(ctx context.Context, fields map[string][]string) (*toposv1.MatchResponse, error) {
	matchFields := make(map[string]*toposv1.StringList, len(fields))
	for k, v := range fields {
		matchFields[k] = &toposv1.StringList{Values: v}
	}
	return p.impl.Match(ctx, &toposv1.MatchRequest{MatchFields: matchFields})
}

// Fetch calls the plugin's Fetch RPC.
func (p *Plugin) Fetch(ctx context.Context, sourceID string, variant toposv1.ContentVariant) (*toposv1.FetchResponse, error) {
	return p.impl.Fetch(ctx, &toposv1.FetchRequest{SourceId: sourceID, Variant: variant})
}

// Health calls the plugin's Health RPC.
func (p *Plugin) Health(ctx context.Context) (*toposv1.HealthResponse, error) {
	return p.impl.Health(ctx, &toposv1.HealthRequest{})
}

// Kill terminates the plugin subprocess.
func (p *Plugin) Kill() {
	p.client.Kill()
}

// Icon returns this plugin instance's validated icon bytes and mime,
// captured from its own Describe response (09-01-PLAN.md Task 2). ok is
// false when the plugin declared no icon, declared one that failed
// captureIcon's validation, or was built against the pre-Phase-9 contract.
func (p *Plugin) Icon() ([]byte, string, bool) {
	if len(p.iconBytes) == 0 || p.iconMIME == "" {
		return nil, "", false
	}
	return p.iconBytes, p.iconMIME, true
}

// Host owns the lifecycle of every launched plugin subprocess.
type Host struct {
	// dirs is the two configured plugin directories Discover launched
	// every plugin binary from (Phase 11, widening the pre-existing
	// single pluginsDir field) — retained (07-02-PLAN.md Task 1) so
	// Reconcile can launch a replacement/added instance later without a
	// caller having to thread the directories through a second time. Set
	// once at construction and never written again, so it needs no lock.
	dirs Dirs

	// mu guards ONLY the plugins field below (08-13-PLAN.md Task 1(C),
	// closing G-08-5's second, latent defect: kernel/supervisor.Supervisor
	// already called Host.Fetch with no lock held, concurrently with a
	// Reconcile writing this field in place — a genuine data race at HEAD,
	// merely narrow enough that no test provoked it). This lock makes
	// concurrent READERS (Plugins/ProbeSources/Fetch/byInstance, via
	// snapshot below) safe against a single in-flight Reconcile; two
	// concurrent Reconcile calls remain excluded by kernel/supervisor's own
	// mutation lock (Supervisor.mu), which every caller of Reconcile in
	// this repo already holds for the call's whole duration.
	mu      sync.RWMutex
	plugins []*Plugin
}

// snapshot returns a defensive copy of the currently launched plugin set
// under mu.RLock() — the one place the read lock is taken for the plugin
// set, so every read path below shares it rather than reimplementing its
// own lock/copy. A copy, not the live backing array, so a caller ranging
// over the result after the lock is released can never observe a
// concurrent Reconcile's commit mid-iteration.
func (h *Host) snapshot() []*Plugin {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]*Plugin, len(h.plugins))
	copy(out, h.plugins)
	return out
}

// Discover launches one subprocess per configured source, resolving each
// binary through dirs (Phase 11's two-tier ResolveBinary), and returns the
// resulting Host. Every launched plugin's Describe RPC is called
// immediately to learn its source_type — the kernel never trusts a
// plugin's identity from its filename (T-01-07).
func Discover(ctx context.Context, dirs Dirs, sources map[string]config.Source, logger hclog.Logger) (*Host, error) {
	h := &Host{dirs: dirs}

	for name, src := range sources {
		p, err := launch(ctx, dirs, name, src, logger, false)
		if err != nil {
			h.Shutdown()
			return nil, fmt.Errorf("pluginhost: launch source %q: %w", name, err)
		}
		h.plugins = append(h.plugins, p)
	}

	return h, nil
}

// Reconcile brings the launched plugin set in line with sources, in place
// (07-02-PLAN.md Task 1, D-06/D-07's hot-apply): for every configured
// instance not currently launched, or currently launched with a different
// config.Source than sources now declares, it launches a replacement via
// the same unexported launch() Discover uses and kills the instance it
// replaces; every currently launched instance absent from sources is
// killed and dropped; every other instance's subprocess is left running
// completely untouched, so a save that edits one source's config never
// blips an unrelated source's reachability.
//
// Every new/changed launch is attempted before anything currently running
// is torn down: a launch failure kills only what THIS call itself
// launched, leaves the previously running set fully intact, and returns
// an error naming the offending instance — a partial apply must never
// look successful (T-07-11). Instance names are iterated in sorted order
// so which instance is reported first on a multi-failure config is
// deterministic run to run, matching this package's existing discipline
// (matchconfig.go).
//
// Locking (08-13-PLAN.md Task 1(C), G-08-5): this call takes h.mu.RLock()
// only to build existing from a snapshot of the CURRENT plugin set, then
// releases it and performs the whole toLaunch launch loop with NO LOCK
// HELD, then takes h.mu.Lock() for the kill-and-commit region only (the
// existing kill loop plus the h.plugins = next assignment). Holding the
// write lock across the launches is exactly the defect this fix closes —
// a plugin subprocess slow to complete its handshake used to freeze every
// concurrent reader (Plugins/ProbeSources/Fetch/byInstance) kernel-wide,
// contradicting phase success criterion 4's "every other source is
// unaffected" — so it must never be done here again.
func (h *Host) Reconcile(ctx context.Context, sources map[string]config.Source, logger hclog.Logger) error {
	h.mu.RLock()
	existing := make(map[string]*Plugin, len(h.plugins))
	for _, p := range h.plugins {
		existing[p.name] = p
	}
	h.mu.RUnlock()

	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)

	kept := make(map[string]*Plugin, len(names))
	var toLaunch []string
	for _, name := range names {
		p, ok := existing[name]
		if ok && reflect.DeepEqual(p.src, sources[name]) {
			kept[name] = p
			continue
		}
		toLaunch = append(toLaunch, name)
	}

	launched := make(map[string]*Plugin, len(toLaunch))
	for _, name := range toLaunch {
		p, err := launch(ctx, h.dirs, name, sources[name], logger, false)
		if err != nil {
			for _, lp := range launched {
				lp.Kill()
			}
			return fmt.Errorf("pluginhost: reconcile: launch source %q: %w", name, err)
		}
		launched[name] = p
	}

	// Every launch this call needed has now succeeded — only now take the
	// write lock, kill the instances being replaced or removed, and commit
	// the new plugin set.
	h.mu.Lock()
	defer h.mu.Unlock()

	for name, p := range existing {
		if _, stillKept := kept[name]; stillKept {
			continue
		}
		p.Kill()
	}

	next := make([]*Plugin, 0, len(kept)+len(launched))
	for _, name := range names {
		if p, ok := kept[name]; ok {
			next = append(next, p)
			continue
		}
		next = append(next, launched[name])
	}
	h.plugins = next
	return nil
}

// stderrTailCap bounds stderrTail's retained byte count. Chosen in the low
// kilobytes: comfortably larger than a single fatal line or the last few
// frames of a panic stack trace (the shapes this capture exists to
// surface — see plugins/signal/main.go's fatal helper for the worked
// example), while staying irrelevant to kernel memory even if every
// launched plugin instance held its own tail for its whole lifetime.
const stderrTailCap = 4096

// stderrTail is a bounded, mutex-guarded tail of a plugin subprocess's
// stderr output, wired into goplugin.ClientConfig.Stderr in launch below.
//
// Why it exists: go-plugin defaults ClientConfig.Stderr to io.Discard, so a
// plugin that calls fatal (writes one line, exits) before ever reaching
// goplugin.Serve writes its only explanation into nothing — the kernel log
// records only "exit status 1", and DescribePluginType's caller sees only
// go-plugin's own four-item handshake diagnostic list, every item of which
// was false in the case this type was written to fix (07-13-PLAN.md,
// closing 07-UAT.md G-07-5: a blank Signal path field). This is a
// recurrence guard for the whole pre-handshake-fatal class, not a fix
// scoped to one field — it fires for any plugin, any config field, any
// future pre-Serve fatal, on both trial launches and boot-time launches,
// since both go through this same launch function.
//
// It is bounded because a plugin looping on stderr before eventually
// exiting must never grow kernel memory without limit — a memory-safety
// requirement, not a cosmetic one. It is mutex-guarded because go-plugin's
// own logStderr goroutine (client.go) writes to this type concurrently
// with the kernel's own goroutine that will eventually read it; an
// unsynchronized read/write pair here would be a genuine data race, not a
// theoretical one. It discards from the FRONT once the cap is exceeded
// (never truncates the incoming write, never stops accepting bytes)
// because the explanatory line is the LAST thing a dying plugin writes —
// stopping at the cap would keep the least useful bytes and discard the
// one line this whole type exists to preserve.
type stderrTail struct {
	mu  sync.Mutex
	buf []byte
}

// Write satisfies io.Writer for goplugin.ClientConfig.Stderr. It always
// reports the full input length written and a nil error — go-plugin's own
// stderr-draining loop (logStderr) does nothing special with a short-write
// report beyond logging it, but this type must never be the reason that
// loop believes a write failed; the tail's own bounding happens entirely
// inside this call, invisibly to the caller.
func (t *stderrTail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > stderrTailCap {
		t.buf = t.buf[len(t.buf)-stderrTailCap:]
	}
	return len(p), nil
}

// lastLine returns the final non-empty line currently held, with
// surrounding whitespace (including any trailing newline(s)) trimmed, or
// an empty string when nothing has been written. Safe to call concurrently
// with Write, but the caller in launch below only ever calls it AFTER
// client.Kill() has returned — Kill waits on go-plugin's own client wait
// group, which the stderr-draining goroutine belongs to, so calling this
// any earlier would race that goroutine and could miss the very line this
// type exists to surface.
func (t *stderrTail) lastLine() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	trimmed := strings.TrimRight(string(t.buf), "\r\n")
	if trimmed == "" {
		return ""
	}
	if idx := strings.LastIndexByte(trimmed, '\n'); idx >= 0 {
		trimmed = trimmed[idx+1:]
	}
	return strings.TrimSpace(trimmed)
}

// launch resolves src.Plugin against dirs via ResolveBinary (Phase 11's
// two-tier provenance authority), execs the resolved path, performs the
// go-plugin handshake, and calls Describe to learn the launched instance's
// identity. Every caller —
// Discover, Reconcile, and DescribePluginType's trial launch — goes
// through this one function, so the bounded stderr capture wired in below
// (stderrTail) covers a pre-handshake plugin fatal identically whether the
// launch is a UI-driven trial or a real boot-time/hot-apply launch: the
// class of failure this guards against has nothing to do with which caller
// triggered it.
//
// describeOnly (CR-01, 08-REVIEW.md), when true, sets the
// WEBSPACES_DESCRIBE_ONLY=1 environment variable alongside the usual
// WEBSPACES_SOURCE_CONFIG one — set only by DescribePluginType's trial
// launch, never by Discover/Reconcile's real boot-time/hot-apply launches.
// A launched plugin binary that recognises this variable (currently only
// plugins/whatsapp/main.go) may use it to skip acquiring any exclusive,
// process-lifetime resource (e.g. WhatsApp's storelock.go flock) it would
// otherwise hold for as long as this trial-launched subprocess is alive,
// since Describe's answer never depends on that resource. A plugin binary
// that does not recognise the variable simply ignores it and launches
// exactly as before — this is additive, never a behavior change for any
// plugin type that hasn't opted in.
func launch(ctx context.Context, dirs Dirs, name string, src config.Source, logger hclog.Logger, describeOnly bool) (*Plugin, error) {
	binPath, tier, err := ResolveBinary(dirs, src.Plugin, logger)
	if err != nil {
		return nil, fmt.Errorf("plugin binary %s not found: %w", src.Plugin, err)
	}

	sourceConfig, err := json.Marshal(map[string]string{
		"base_url":         src.BaseURL,
		"token":            src.Token,
		"api_version":      src.APIVersion,
		"ca_cert":          src.CACert,
		"username":         src.Username,
		"webmail_base_url": src.WebmailBaseURL,
		// path: the local filesystem path source config field (Source.Path
		// — kernel/config/types.go), needed by a local-path source like
		// SRC-02's Signal plugin, which has no base_url/token at all.
		"path": src.Path,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal source config: %w", err)
	}

	// The exec.Cmd's own Stderr field is deliberately left unset: go-plugin
	// builds its own stderr pipe from this *exec.Cmd internally, and
	// os/exec rejects a command whose stderr writer is already set when
	// the caller (go-plugin, in this case) tries to set it a second time.
	// The capture goes through ClientConfig.Stderr below instead, which
	// go-plugin's own logStderr goroutine writes to.
	cmd := exec.Command(binPath)
	env := append(os.Environ(), "WEBSPACES_SOURCE_CONFIG="+string(sourceConfig))
	if describeOnly {
		env = append(env, "WEBSPACES_DESCRIBE_ONLY=1")
	}
	cmd.Env = env

	tail := &stderrTail{}

	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig:  sdk.Handshake,
		Plugins:          sdk.PluginMap,
		Cmd:              cmd,
		AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolGRPC},
		Logger:           logger.Named("plugin." + name),
		// Raises the gRPC message-size ceiling to match sdk.GRPCServer on
		// the plugin side, so a unary Fetch response carrying a full
		// rendition doesn't hit the 4 MB default (D-Task1, 01-01).
		GRPCDialOptions: []grpc.DialOption{
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(sdk.MaxMessageSize),
				grpc.MaxCallSendMsgSize(sdk.MaxMessageSize),
			),
		},
		// Captures a bounded tail of the child's stderr (07-13-PLAN.md
		// Task 3, closing 07-UAT.md G-07-5's second cause) — without this,
		// go-plugin defaults to io.Discard and a plugin that fatals before
		// the handshake explains itself to nobody.
		Stderr: tail,
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		// The read must happen strictly after Kill() returns: Kill waits
		// on go-plugin's client wait group, which the stderr-draining
		// goroutine (logStderr) belongs to — reading any earlier races
		// that goroutine and can miss the very line this capture exists
		// to surface. When nothing was captured (a genuinely silent
		// pre-handshake exit, or a failure that isn't a pre-handshake
		// fatal at all), the error text is left byte-identical to before
		// this capture existed — no empty parenthetical, no dangling
		// separator.
		if last := tail.lastLine(); last != "" {
			return nil, fmt.Errorf("connect to plugin subprocess: %w (plugin stderr: %s)", err, last)
		}
		return nil, fmt.Errorf("connect to plugin subprocess: %w", err)
	}

	raw, err := rpcClient.Dispense("source")
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("dispense source plugin: %w", err)
	}

	impl, ok := raw.(sdk.SourcePlugin)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("plugin %s does not implement sdk.SourcePlugin", name)
	}

	desc, err := impl.Describe(ctx, &toposv1.DescribeRequest{})
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("describe plugin %s: %w", name, err)
	}

	// The instance display name resolves from the operator's own config
	// (D-09), falling back to the instance id itself when display_name is
	// omitted — never from anything the plugin process asserts. This
	// mirrors config.Config.DisplayNameFor without needing the whole
	// *config.Config here, since launch already holds this instance's own
	// config.Source.
	instanceDisplayName := src.DisplayName
	if instanceDisplayName == "" {
		instanceDisplayName = name
	}

	iconBytes, iconMIME, _ := captureIcon(desc)

	return &Plugin{
		name:            name,
		sourceType:      desc.GetSourceType(),
		pluginName:      desc.GetDisplayName(),
		displayName:     instanceDisplayName,
		matchVocabulary: desc.GetMatchVocabulary(),
		iconBytes:       iconBytes,
		iconMIME:        iconMIME,
		src:             src,
		client:          client,
		impl:            impl,
		tier:            tier,
	}, nil
}

// Plugins returns every launched plugin — a defensive copy (via snapshot)
// taken under h.mu.RLock(), never the live backing array, so a caller
// ranging over the result cannot observe a concurrent Reconcile's commit
// mid-iteration.
func (h *Host) Plugins() []*Plugin {
	return h.snapshot()
}

// PluginIcon resolves a plugin BINARY's icon (src.Plugin, e.g.
// "topos-plugin-mock") — never an instance name — over every currently
// launched instance, returning the first one that has a validated icon
// (09-01-PLAN.md Task 2, 09-UI-SPEC.md Fix 10). Two instances of the same
// binary have byte-identical icons (both came from the same Describe
// implementation), so "first launched instance with an icon" is
// equivalent to "this binary's icon." ok is false when no launched
// instance of binary has one — an undescribed plugin type (never
// launched), or every launch attempt so far predates a successful
// Describe.
func (h *Host) PluginIcon(binary string) (iconBytes []byte, iconMIME string, ok bool) {
	for _, p := range h.snapshot() {
		if p.src.Plugin != binary {
			continue
		}
		if b, m, iok := p.Icon(); iok {
			return b, m, true
		}
	}
	return nil, "", false
}

// DescribeInfo is the three Describe-derived facts the "+" chip picker's
// two-step modal needs to build a match-vocabulary-driven form for a
// plugin type that has no configured instance yet (D-11 step 1 -> step 2,
// 07-02-PLAN.md Task 3).
type DescribeInfo struct {
	SourceType        string
	PluginDisplayName string
	MatchVocabulary   []string
	// Tier is the launch-time provenance Tier the trial-launched binary
	// resolved to (T-11-01) — set from ResolveBinary via the trial
	// launch's own *Plugin, never from anything the plugin's Describe
	// response itself asserts.
	Tier Tier
}

// DescribePluginType trial-launches src (a config.Source naming the
// plugin binary to run, submitted but not yet persisted anywhere) under a
// fixed, non-persisted instance name, calls its Describe RPC, and kills
// the subprocess before returning — writing nothing to disk and
// registering nothing on any *Host (D-11 step 1 -> step 2). It reuses the
// existing unexported launch() verbatim (the same binary-stat, handshake,
// dispense and Describe-call sequence Discover already performs per
// configured instance) rather than duplicating any of it, and calls no
// RPC beyond the one Describe call launch() already makes — the trial-
// launch path must never become a general plugin-invocation surface for
// request-supplied input (PLUG-02, T-07-10); see
// kernel/httpapi/config_test.go's AST guard pinning exactly this over
// this function's own body.
//
// Every trial launch here passes describeOnly=true to launch() (CR-01,
// 08-REVIEW.md): this call site is ALWAYS a describe-only launch, never a
// real boot-time/hot-apply one, regardless of whether src names a plugin
// type with no configured instance yet, or an already-configured,
// already-running instance's own stored connection fields (the "+"
// picker's one-step existing-instance add flow, and every source chip's
// "Edit match settings…" entry, both reuse this same call — see
// kernel/httpapi/config.go's DescribePluginHandler doc comment). For most
// plugin types this changes nothing observable (they open-and-close per
// RPC call and hold no cross-instance exclusive resource); for WhatsApp —
// the one plugin in this repo that holds a persistent connection and an
// exclusive store lock for its entire process lifetime — this is what
// stops the trial-launch from always losing that lock race against a real
// running instance and failing before Describe is ever reached.
func DescribePluginType(ctx context.Context, dirs Dirs, src config.Source, logger hclog.Logger) (DescribeInfo, error) {
	p, err := launch(ctx, dirs, "__trial__", src, logger, true)
	if err != nil {
		return DescribeInfo{}, fmt.Errorf("pluginhost: trial-launch for describe: %w", err)
	}
	defer p.Kill()

	return DescribeInfo{
		SourceType:        p.SourceType(),
		PluginDisplayName: p.PluginDisplayName(),
		MatchVocabulary:   p.MatchVocabulary(),
		Tier:              p.Tier(),
	}, nil
}

// SourceHealth is one plugin's live reachability probe result, keyed to
// its config name and Describe-learned identity. 02-02-PLAN.md's D-08:
// this is a liveness signal only — last-sync time and last-error are
// deliberately NOT part of this type, because those come from the
// kernel's own sync_runs history (kernel/index.Store), never from a
// plugin's self-reported HealthResponse fields (A-PLUG-04). A plugin that
// reports a rosier history than the kernel actually recorded must not be
// able to turn its own health chip green.
type SourceHealth struct {
	Name        string // config key under [sources.<name>]
	SourceType  string // Describe-learned
	DisplayName string // Describe-learned
	// Plugin is the launched instance's plugin BINARY name (src.Plugin,
	// e.g. "topos-plugin-mock") — never the instance id or SourceType.
	// Threaded through to GET /api/sources (09-01-PLAN.md Task 3) so the
	// SPA can address GET /api/plugins/{plugin}/icon per row without a
	// second lookup.
	Plugin string
	// Tier is the launched instance's launch-time provenance Tier
	// (T-11-01) — set from the same *Plugin.Tier() every other field on
	// this line reads, never re-derived from a live RPC.
	Tier       Tier
	Reachable  bool
	ProbeError string
}

// ProbeSources calls every launched plugin's Health RPC concurrently — a
// live reachability probe, not a cached value — and returns one
// SourceHealth per plugin, in launch order (so the response order is
// stable run to run, matching the deterministic ordering
// kernel/httpapi.SourcesHandler sorts on top of). A transport error or a
// Reachable:false response both map to Reachable:false with the error
// text in ProbeError; ProbeSources itself never returns a Go error — one
// plugin being unreachable is data, not a failure of the whole probe.
func (h *Host) ProbeSources(ctx context.Context) []SourceHealth {
	plugins := h.snapshot()
	out := make([]SourceHealth, len(plugins))
	var wg sync.WaitGroup
	for i, p := range plugins {
		wg.Add(1)
		go func(i int, p *Plugin) {
			defer wg.Done()
			health := SourceHealth{Name: p.Name(), SourceType: p.SourceType(), DisplayName: p.DisplayName(), Plugin: p.src.Plugin, Tier: p.tier}
			resp, err := p.Health(ctx)
			switch {
			case err != nil:
				health.Reachable = false
				health.ProbeError = err.Error()
			case !resp.GetReachable():
				health.Reachable = false
				health.ProbeError = resp.GetLastError()
			default:
				health.Reachable = true
			}
			out[i] = health
		}(i, p)
	}
	wg.Wait()
	return out
}

// FetchResult is the kernel-domain translation of a plugin's unary Fetch
// response. Body is nil when there is no rendition (available=false, or a
// full-variant fetch that carried only extracted text).
type FetchResult struct {
	Available         bool
	UnavailableReason string
	MimeType          string
	SizeBytes         int64
	Text              string
	Provenance        map[string]string
	Body              io.ReadCloser
	// ContentShape carries FetchResponse.content_shape (D-11) into the
	// kernel domain — meaningful only when MimeType is "text/html";
	// kernel/httpapi.sanitizeAndWrapRendition uses it to select the
	// sanitize/wrap/theme policy for this rendition. Zero value
	// (CONTENT_SHAPE_UNSPECIFIED) for every other MimeType.
	ContentShape toposv1.ContentShape
}

// Fetch calls the Fetch RPC on the plugin registered for source (the
// source INSTANCE id supplied by an index row — item.Item.Source, D-08),
// request-time only (never sync-time), and translates the result into
// kernel-domain types and errors. gRPC codes.NotFound maps to
// ErrItemNotFound, codes.Unavailable and any other transport failure maps
// to ErrSourceUnavailable.
func (h *Host) Fetch(ctx context.Context, source, sourceID string, variant toposv1.ContentVariant) (FetchResult, error) {
	p := h.byInstance(source)
	if p == nil {
		return FetchResult{}, fmt.Errorf("%w: no plugin registered for source %q", ErrItemNotFound, source)
	}

	resp, err := p.impl.Fetch(ctx, &toposv1.FetchRequest{SourceId: sourceID, Variant: variant})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return FetchResult{}, fmt.Errorf("%w: %s", ErrItemNotFound, st.Message())
			case codes.Unavailable:
				return FetchResult{}, fmt.Errorf("%w: %s", ErrSourceUnavailable, st.Message())
			}
		}
		return FetchResult{}, fmt.Errorf("%w: %v", ErrSourceUnavailable, err)
	}

	var body io.ReadCloser
	if len(resp.GetData()) > 0 {
		body = io.NopCloser(bytes.NewReader(resp.GetData()))
	}

	return FetchResult{
		Available:         resp.GetAvailable(),
		UnavailableReason: resp.GetUnavailableReason(),
		MimeType:          resp.GetMimeType(),
		SizeBytes:         resp.GetSizeBytes(),
		Text:              resp.GetText(),
		Provenance:        resp.GetProvenance(),
		Body:              body,
		ContentShape:      resp.GetContentShape(),
	}, nil
}

// byInstance returns the launched plugin whose config-key instance id
// (Name()) matches, or nil if none is registered. Keyed on instance id, not
// on the Describe-learned plugin kind, so two launched instances of one
// plugin binary resolve to two distinct *Plugin values (D-08). Takes
// h.mu.RLock() itself (via snapshot) — Fetch, its only caller, makes no
// other access to h.plugins, so there is no nested-lock risk here.
func (h *Host) byInstance(source string) *Plugin {
	for _, p := range h.snapshot() {
		if p.name == source {
			return p
		}
	}
	return nil
}

// Shutdown kills every launched plugin subprocess.
func (h *Host) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, p := range h.plugins {
		p.Kill()
	}
}
