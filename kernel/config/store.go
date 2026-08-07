package config

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pelletier/go-toml/v2"
)

// fileHash hex-encodes the SHA-256 digest of raw — the same digest LoadRaw
// computes over a freshly-read file, so Store.Save's re-read-and-compare
// (D-03) is always comparing like with like.
func fileHash(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// ErrConfigChangedOnDisk is returned by Store.Save when the config file on
// disk has changed since it was last read (D-03's optimistic content-hash
// lock) — the save is rejected outright rather than merged or overwritten,
// so a concurrent hand-edit is never silently clobbered.
var ErrConfigChangedOnDisk = errors.New("config: file changed on disk since it was last read")

// ErrConfigHasUnknownKeys is returned by Store.Save when the config file on
// disk carries a key the Config struct does not model. A canonical rewrite
// would silently drop that key — this guard refuses the write instead
// (D-01's "flattens comments only", never data).
var ErrConfigHasUnknownKeys = errors.New("config: file contains keys topos does not recognise")

// Store is the swappable holder for the kernel's running configuration
// (assumption-delta decision, 07-01-PLAN.md: the running configuration is
// now the primary noun, not a boot-time *config.Config pointer). It holds
// both the expanded (runtime, secret-bearing) and raw (unexpanded,
// persistable) forms behind atomic pointers, plus the last-known file
// hash used by Save's clobber guard (D-03).
//
// Every handler that used to close over a boot-time *config.Config should
// instead hold a *Store and call Expanded()/Raw() per request (or, for the
// three handlers this tracer plan deliberately leaves as a boot-time
// snapshot — WebspacesHandler, ItemHandler, SourceRefreshHandler — resolve
// cfg := cfgStore.Expanded() once at Router construction; see routes.go's
// comment naming 07-02 Task 2 as the fill for that gap).
type Store struct {
	path string

	expanded atomic.Pointer[Config]
	raw      atomic.Pointer[Config]

	mu          sync.RWMutex
	hash        string
	unknownKeys []string
}

// NewStore loads path via LoadRaw and seeds a new Store with the result.
// Callers hold the returned *Store for the kernel's lifetime; Save/Reload
// mutate it in place rather than replacing it.
func NewStore(path string) (*Store, error) {
	expanded, raw, hash, unknownKeys, err := LoadRaw(path)
	if err != nil {
		return nil, err
	}
	s := &Store{path: path, hash: hash, unknownKeys: unknownKeys}
	s.expanded.Store(expanded)
	s.raw.Store(raw)
	return s, nil
}

// NewStoreForTesting builds a Store directly from an in-memory config, with
// no backing file on disk (Path() returns ""). Deviation beyond this
// plan's declared Artifacts (Rule 3 — blocking compile dependency): dozens
// of existing httpapi tests build a *config.Config by hand
// (contract_test.go's testConfig(), agent_test.go, stream_test.go) rather
// than loading a real TOML file, and Router's signature now requires a
// *Store rather than a *config.Config — this constructor is the minimal
// seam that keeps those fixtures working unchanged in shape. Both the
// expanded and raw pointers are seeded with the SAME cfg, since a
// hand-built test fixture has no secret-shaped fields distinguishing the
// two forms. Save and Reload are NOT valid to call on a Store built this
// way (both re-read the empty path) — every other accessor (Expanded, Raw,
// Hash, Path) behaves normally.
func NewStoreForTesting(cfg *Config) *Store {
	s := &Store{}
	s.expanded.Store(cfg)
	s.raw.Store(cfg)
	return s
}

// Expanded returns the current runtime config: os.Expand'd, defaulted,
// validated. May hold secret VALUES in memory — never marshal this to disk
// or serialize it over HTTP (D-05).
func (s *Store) Expanded() *Config {
	return s.expanded.Load()
}

// Raw returns the current raw (unexpanded) config: ${VAR} references and
// "~"-prefixed paths held verbatim. This is the only form ever handed to
// WriteCanonical or serialized as a GET /api/config response body (D-05).
func (s *Store) Raw() *Config {
	return s.raw.Load()
}

// Path returns the config file path this Store was constructed against.
func (s *Store) Path() string {
	return s.path
}

// Hash returns the last-known SHA-256 hex digest of the on-disk config
// file's raw bytes, as recorded by the most recent successful
// NewStore/Save/Reload — the base a client's next Save call must supply to
// prove it read current state (D-03).
func (s *Store) Hash() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hash
}

// UnknownKeys returns the last-known set of TOML key paths the on-disk
// config file carries that the Config struct does not model, as of the
// most recent successful NewStore/Save/Reload — GET /api/config surfaces
// this so a hand-edited file with a stray key is visible to the UI before
// the next save would refuse to persist over it.
func (s *Store) UnknownKeys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.unknownKeys
}

// swap atomically replaces both config pointers and the recorded hash/
// unknown-keys pair — the single mutation point every successful Save/
// Reload funnels through, so a reader can never observe values from two
// different file generations.
func (s *Store) swap(expanded, raw *Config, hash string, unknownKeys []string) {
	s.expanded.Store(expanded)
	s.raw.Store(raw)
	s.mu.Lock()
	s.hash = hash
	s.unknownKeys = unknownKeys
	s.mu.Unlock()
}

// Save validates, writes and hot-applies next as the Store's new raw
// config (D-06: save = apply immediately, no separate apply step). The
// full sequence, in order (07-01-PLAN.md Task 1):
//
//  1. Re-read the file from disk and hash it. If that hash differs from
//     baseHash, the caller read a now-stale version — return
//     ErrConfigChangedOnDisk without writing anything (D-03).
//  2. Strict-decode those same re-read bytes via UnknownKeys. A non-empty
//     result means the on-disk file carries a key this Config struct
//     doesn't model — a canonical rewrite of next would silently drop it,
//     so refuse and name the keys (D-01's lossless-rewrite prohibition)
//     rather than writing anything.
//  3. Dry-run next through the identical expand -> unmarshal -> defaults ->
//     home-dir-expansion -> Validate path LoadRaw uses, WITHOUT writing
//     anything yet. A validation failure is returned unchanged so its
//     message reaches the caller verbatim (D-09) — no second rule set,
//     the kernel's own load-time validator is the only judge.
//  4. Only once all of the above succeed: WriteCanonical(next) — the
//     canonical rewrite, backup, and atomic rename (D-01/D-02/D-04).
//  5. Re-read the just-written file to recompute its hash, and swap both
//     pointers plus the hash under one lock, so a concurrent reader never
//     observes a partially-applied save.
func (s *Store) Save(next *Config, baseHash string) error {
	currentBytes, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("config: re-read %s before save: %w", s.path, err)
	}
	if fileHash(currentBytes) != baseHash {
		return ErrConfigChangedOnDisk
	}

	if unknown := UnknownKeys(currentBytes); len(unknown) > 0 {
		return fmt.Errorf("%w: %s", ErrConfigHasUnknownKeys, strings.Join(unknown, ", "))
	}

	dryRunExpanded, err := dryRunExpand(next)
	if err != nil {
		return err
	}

	if err := WriteCanonical(s.path, next); err != nil {
		return err
	}

	newBytes, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("config: re-read %s after save: %w", s.path, err)
	}

	s.swap(dryRunExpanded, next, fileHash(newBytes), UnknownKeys(newBytes))
	return nil
}

// dryRunExpand marshals next to TOML bytes in memory and runs the SAME
// expand -> unmarshal -> defaults -> home-dir-expansion -> Validate path
// LoadRaw uses over a real file — this is D-09's dry run: the kernel's one
// validator, reused rather than duplicated, over content that has not
// touched disk yet. On success it returns the resulting expanded config
// (the value Save swaps in as the new Store.Expanded()) so a successful
// save never re-parses the file a third time.
func dryRunExpand(next *Config) (*Config, error) {
	body, err := toml.Marshal(next)
	if err != nil {
		return nil, fmt.Errorf("config: marshal dry-run config: %w", err)
	}

	expandedStr, missing := expandEnv(string(body))
	var dryRun Config
	if err := toml.Unmarshal([]byte(expandedStr), &dryRun); err != nil {
		return nil, fmt.Errorf("config: parse dry-run config: %w", err)
	}
	applyDefaults(&dryRun)
	if err := dryRun.expandIndexPathHome(); err != nil {
		return nil, err
	}
	if err := dryRun.expandSourceCACertPathsHome(); err != nil {
		return nil, err
	}
	if err := dryRun.Validate(missing); err != nil {
		return nil, err
	}
	return &dryRun, nil
}

// Reload re-reads the config file from disk through the same LoadRaw path
// NewStore uses and swaps it in on success (D-08's explicit Reload
// affordance — no file watcher). On failure it returns the error and
// leaves the Store's previous pointers and hash untouched: an invalid file
// on reload keeps the last-good config running rather than tearing it down.
func (s *Store) Reload() error {
	expanded, raw, hash, unknownKeys, err := LoadRaw(s.path)
	if err != nil {
		return err
	}
	s.swap(expanded, raw, hash, unknownKeys)
	return nil
}
