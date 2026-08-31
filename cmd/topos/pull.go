package main

// topos plugin pull <url> — one CLI command from a plugin URL to the
// trust tier its provenance earns (M1-R8, GSD PULL-01/PULL-02,
// davison/topos#19).
//
// The flow is stage -> discover -> verify -> place, and nothing is
// written to either plugin directory until verification has fully
// decided the tier — every verification failure aborts BEFORE placement
// and leaves both directories byte-identical to their pre-attempt state
// (PULL-02, GSD phase-18 criterion 2, pinned by pull_test.go with
// directory-aware digest snapshots). Placement itself carries the
// install scripts' precise two-pass guarantee, stated at pullPlace.
//
// Evidence discovery has exactly one convention (Decision on #19): the
// release's own checksums.txt beside the binary is its asset manifest —
// it records the binary's SHA-256 (verified in passing) and names the
// *.provenance.json/*.provenance.sig pair this command fetches and
// verifies. Verification is pluginhost.VerifySignedProvenance over the
// staging directory with the kernel's embedded key set — the launch
// gate's own code path, never a second implementation — so the tier the
// pull concludes is the tier the kernel concludes at launch. There is
// deliberately NO flag that names a tier, a provenance URL, or a key:
// the earned tier cannot be overridden (criterion 1).
//
// Tier outcomes:
//   - a validly-signed manifest naming the binary's exact digest ->
//     TRUSTED: the binary and its vouching manifest pair are placed in
//     the config's trusted plugins directory, so the launch gate can
//     reach the same verdict;
//   - no provenance evidence -> EXTERNAL: the bare binary is placed in
//     the external directory and the consent-and-pin steps are printed
//     — the unchanged untrusted-add flow, exactly as a hand-copied
//     binary would take. This is the state whether checksums.txt is
//     cleanly absent OR present-and-clean but naming no provenance pair
//     (the legitimate unsigned third-party release shape — an author
//     without a key in the kernel's set publishes integrity, not
//     authenticity; amended Decision on #19). A present checksums.txt
//     still MUST name the binary with matching bytes, or the pull
//     aborts;
//   - evidence that exists but does not verify — a checksums line
//     contradicting the downloaded bytes or omitting the binary, a bad
//     signature, an unknown key id, a digest or platform mismatch, or
//     published provenance that simply never vouches for this binary —
//     is a FAILED verification: loud abort, nothing placed, never a
//     demotion to the external tier (Decision on #19; the same
//     never-demote-and-run rule the launch gate applies).

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/pluginhost"
)

// pullHTTPClient is every download's client. Redirects are followed
// (GitHub release assets redirect cross-origin to
// objects.githubusercontent.com — that is normal and allowed), but an
// https->http downgrade anywhere in the chain is refused by name: the
// operator named a scheme, and a downgrade would strip transport
// integrity from artifacts this command is about to trust-evaluate.
// Policy pinned by TestPullCheckRedirect.
var pullHTTPClient = &http.Client{CheckRedirect: pullCheckRedirect}

// pullCheckRedirect implements the policy above: at most ten hops
// (net/http's own conventional cap, made explicit), and never a hop
// from an https origin to a non-https URL.
func pullCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	if via[0].URL.Scheme == "https" && req.URL.Scheme != "https" {
		return fmt.Errorf("refusing redirect from https to %s (%s) — a downgrade would strip transport integrity from the artifact", req.URL.Scheme, req.URL)
	}
	return nil
}

// pullBinaryNamePattern is the same name shape the fleet installer's
// allowlist and the pluginhost directory scan accept: the
// topos-plugin- prefix followed by lowercase letters, digits and
// hyphens. Anything else — uppercase, dots, path separators — is
// refused before a single byte downloads.
var pullBinaryNamePattern = regexp.MustCompile(`^topos-plugin-[a-z0-9-]+$`)

// pullProvenanceNamePattern validates a provenance basename named by
// checksums.txt before it is used as a local write path: no path
// separators, no leading dot (the manifest's names are untrusted text —
// the same discipline scripts/install.sh applies).
var pullProvenanceNamePattern = regexp.MustCompile(`^[^/\\.][^/\\]*\.provenance\.(json|sig)$`)

// runPluginPull loads the config (ONLY the config — never setup(),
// which would open the kernel's index database) and runs the pull
// against its resolved tier directories.
func runPluginPull(rawURL, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("plugin pull: load config: %w", err)
	}
	return pullPlugin(rawURL, cfg, os.Stdout)
}

// pullPlugin is the whole flow, factored on *config.Config and an
// io.Writer so pull_test.go drives it against httptest fixtures and
// temp directories with no process-global state.
func pullPlugin(rawURL string, cfg *config.Config, out io.Writer) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("plugin pull: parse url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("plugin pull: unsupported url scheme %q — only http and https are supported", u.Scheme)
	}
	name := path.Base(u.Path)
	if !pullBinaryNamePattern.MatchString(name) {
		return fmt.Errorf("plugin pull: url basename %q is not a plugin binary name (topos-plugin-<name>, lowercase letters, digits and hyphens)", name)
	}
	// Sibling-asset discovery works on the URL's PATH: the binary
	// itself is fetched with whatever query the operator's URL carried
	// (a signed CDN token, say), while checksums.txt and the provenance
	// pair are addressed as plain path siblings with no query — a
	// per-asset token cannot be assumed to authorize its siblings, and a
	// host that needs one per asset is outside this command's one
	// discovery convention. Pinned by TestPull_QueryBearingURL.
	baseURL := *u
	baseURL.Path = path.Dir(u.Path)
	baseURL.RawQuery = ""

	trustedDir, err := pluginsDir(cfg)
	if err != nil {
		return fmt.Errorf("plugin pull: resolve trusted plugins directory: %w", err)
	}
	externalDir, err := externalPluginsDir(cfg)
	if err != nil {
		return fmt.Errorf("plugin pull: resolve external plugins directory: %w", err)
	}

	// --- stage --------------------------------------------------------
	stage, err := os.MkdirTemp("", "topos-plugin-pull-*")
	if err != nil {
		return fmt.Errorf("plugin pull: create staging directory: %w", err)
	}
	defer os.RemoveAll(stage)

	stagedBinary := filepath.Join(stage, name)
	if err := pullDownload(rawURL, stagedBinary, 0o755); err != nil {
		return fmt.Errorf("plugin pull: download %s: %w — nothing was placed", name, err)
	}

	// --- discover -----------------------------------------------------
	// checksums.txt beside the binary is the release's own asset
	// manifest. Cleanly absent (404/410) means "this release publishes
	// no evidence" — the legitimate external-tier state. Any OTHER
	// failure to fetch it is a failed download and aborts: an evidence
	// file that exists but cannot be read must never silently read as
	// absence.
	checksumsURL := baseURL.JoinPath("checksums.txt").String()
	checksums, checksumsPresent, err := pullFetchOptional(checksumsURL)
	if err != nil {
		return fmt.Errorf("plugin pull: fetch %s: %w — nothing was placed", checksumsURL, err)
	}

	provenanceFiles := []string{}
	if checksumsPresent {
		recorded := ""
		provNames := []string{}
		for _, line := range strings.Split(string(checksums), "\n") {
			line = strings.TrimRight(line, "\r")
			if line == "" {
				continue
			}
			hash, rel, found := strings.Cut(line, "  ")
			if !found || hash == "" || rel == "" {
				return fmt.Errorf("plugin pull: checksums.txt line is not a sha256sum entry: %q — nothing was placed", line)
			}
			if rel == name {
				recorded = hash
			}
			if strings.HasSuffix(rel, pluginhost.ProvenanceManifestSuffix) || strings.HasSuffix(rel, pluginhost.ProvenanceSignatureSuffix) {
				if !pullProvenanceNamePattern.MatchString(rel) {
					return fmt.Errorf("plugin pull: checksums.txt names a disallowed provenance path %q — nothing was placed", rel)
				}
				provNames = append(provNames, rel)
			}
		}
		// provNames may legitimately be empty here: a clean checksums.txt
		// that names the binary but no provenance pair is the unsigned
		// third-party release shape — integrity without authenticity —
		// and falls through to the external tier below (amended Decision
		// on #19). Absence of provenance is never an abort; only
		// CONTRADICTING evidence is.
		if recorded == "" {
			// The release's own asset manifest disowns this binary —
			// the tampered-or-misconfigured state verification exists
			// to catch, never a fall-through to the external tier
			// (Decision on #19).
			return fmt.Errorf("plugin pull: the release's checksums.txt does not name %s — a release manifest that disowns the binary is a failed verification; nothing was placed", name)
		}
		onDisk, err := pluginhost.HashBinary(stagedBinary)
		if err != nil {
			return fmt.Errorf("plugin pull: hash downloaded binary: %w — nothing was placed", err)
		}
		if onDisk != recorded {
			return fmt.Errorf("plugin pull: SHA-256 mismatch for %s: checksums.txt records %s, downloaded bytes are %s — nothing was placed", name, recorded, onDisk)
		}
		for _, rel := range provNames {
			if err := pullDownload(baseURL.JoinPath(rel).String(), filepath.Join(stage, rel), 0o644); err != nil {
				return fmt.Errorf("plugin pull: download %s (named by checksums.txt): %w — nothing was placed", rel, err)
			}
			provenanceFiles = append(provenanceFiles, rel)
		}
	}

	// --- verify -------------------------------------------------------
	// The launch gate's own verifier, over the staging directory. A
	// refusal (tamper) aborts; a MATCH earns trusted; and published
	// evidence that never vouches for this binary is an abort too —
	// only the no-evidence-at-all state earns the external tier.
	_, evidence, diagnostics, err := pluginhost.VerifySignedProvenance(pluginhost.Dirs{Trusted: stage}, name, stagedBinary)
	if err != nil {
		return fmt.Errorf("plugin pull: provenance verification refused %s: %w — nothing was placed%s", name, err, pullFormatDiagnostics(diagnostics))
	}
	if evidence == "" && len(provenanceFiles) > 0 {
		return fmt.Errorf("plugin pull: the release publishes provenance evidence (%s) but none of it vouches for %s — an unknown signing key, a wrong platform, or a manifest that does not name this binary; nothing was placed%s", strings.Join(provenanceFiles, ", "), name, pullFormatDiagnostics(diagnostics))
	}

	// --- place --------------------------------------------------------
	if evidence != "" {
		sig := strings.TrimSuffix(evidence, pluginhost.ProvenanceManifestSuffix) + pluginhost.ProvenanceSignatureSuffix
		placed, err := pullPlace(stage, trustedDir, []pullPlacement{
			{name: name, mode: 0o755},
			{name: evidence, mode: 0o644},
			{name: sig, mode: 0o644},
		})
		if err != nil {
			return fmt.Errorf("plugin pull: place into trusted directory %s: %w", trustedDir, err)
		}
		fmt.Fprintf(out, "plugin pull: %s earned the TRUSTED tier — a validly-signed release manifest (%s) names its exact digest\n", name, evidence)
		for _, p := range placed {
			fmt.Fprintf(out, "plugin pull:   wrote %s\n", p)
		}
		fmt.Fprintf(out, "plugin pull: restart the kernel (or wait for the next config apply) and add the source from the picker\n")
		return nil
	}

	placed, err := pullPlace(stage, externalDir, []pullPlacement{{name: name, mode: 0o755}})
	if err != nil {
		return fmt.Errorf("plugin pull: place into external directory %s: %w", externalDir, err)
	}
	fmt.Fprintf(out, "plugin pull: %s carries no provenance evidence — it earned the EXTERNAL tier (no release manifest to verify; this is the legitimate no-evidence state, not an error)\n", name)
	for _, p := range placed {
		fmt.Fprintf(out, "plugin pull:   wrote %s\n", p)
	}
	fmt.Fprintf(out, "plugin pull: one-time steps: restart (or start) your kernel, then add the source\n")
	fmt.Fprintf(out, "plugin pull: through the app's untrusted-add consent flow — the same explicit\n")
	fmt.Fprintf(out, "plugin pull: consent-and-pin path any external binary goes through. It then runs\n")
	fmt.Fprintf(out, "plugin pull: pinned and badged untrusted; re-pulled new bytes are re-accepted\n")
	fmt.Fprintf(out, "plugin pull: through the chip's re-pin flow.\n")
	return nil
}

// pullDownload fetches url into dest (mode applied) — 2xx or error,
// never a saved error page.
func pullDownload(url, dest string, mode os.FileMode) error {
	resp, err := pullHTTPClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// pullFetchOptional fetches url, distinguishing clean absence (404/410
// -> present=false, no error) from every other failure.
func pullFetchOptional(url string) (body []byte, present bool, err error) {
	resp, err := pullHTTPClient.Get(url)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, false, fmt.Errorf("HTTP %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

// pullPlacement names one staged file and the mode it is placed with.
type pullPlacement struct {
	name string
	mode os.FileMode
}

// pullPlace moves files from the stage into destDir with the
// two-pass discipline scripts/install.sh established: every
// destination pre-checked (absent or a regular file), every copy
// staged to a temporary name INSIDE destDir first, then one pass of
// same-directory renames — so a copy-pass failure removes the staged
// temporaries and leaves destDir unchanged, and each destination is
// wholly old or wholly new bytes at every instant. Returns the
// destination paths written.
func pullPlace(stage, destDir string, files []pullPlacement) ([]string, error) {
	// Pre-check destinations BEFORE creating anything: a refusal here
	// must leave the filesystem untouched, including the destination
	// directory itself (PR #20 review finding 1 — MkdirAll-first could
	// leave a freshly created directory behind a refusal).
	for _, f := range files {
		if info, err := os.Lstat(filepath.Join(destDir, f.name)); err == nil && !info.Mode().IsRegular() {
			return nil, fmt.Errorf("destination %s exists and is not a regular file — refusing to place over it; nothing was placed", filepath.Join(destDir, f.name))
		}
	}
	// Remember the topmost ancestor MkdirAll is about to create, so a
	// failure later in this function can unwind exactly what it
	// created — non-recursive rmdirs from destDir upward, each stopping
	// naturally on a non-empty directory (the install scripts' own
	// directory discipline).
	createdRoot := firstMissingAncestor(destDir)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create %s (a system prefix may need sudo): %w", destDir, err)
	}
	unwindDirs := func() {
		if createdRoot == "" {
			return
		}
		for p := destDir; ; p = filepath.Dir(p) {
			if os.Remove(p) != nil {
				return
			}
			if p == createdRoot {
				return
			}
		}
	}
	tmps := make([]string, 0, len(files))
	cleanup := func() {
		for _, t := range tmps {
			os.Remove(t)
		}
		unwindDirs()
	}
	for i, f := range files {
		tmp, err := os.CreateTemp(destDir, ".topos-plugin-pull.*")
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("cannot create a temporary file in %s (a system prefix may need sudo): %w", destDir, err)
		}
		tmps = append(tmps, tmp.Name())
		src, err := os.Open(filepath.Join(stage, f.name))
		if err == nil {
			_, err = io.Copy(tmp, src)
			src.Close()
		}
		if cerr := tmp.Close(); err == nil {
			err = cerr
		}
		if err == nil {
			err = os.Chmod(tmp.Name(), f.mode)
		}
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("stage %s into %s: %w — the staged copies were removed, nothing was placed", f.name, destDir, err)
		}
		_ = i
	}
	// The rename pass: one atomic per-file replacement after another,
	// over pre-checked destinations in a directory that now exists — the
	// same residual window scripts/install.sh documents. A failure here
	// (exotic: an I/O error, a race on the directory) is reported for
	// exactly what it is — earlier files already replaced, later ones
	// not — never as "nothing was placed"; a re-pull repairs. Every
	// VERIFICATION failure aborted long before this point.
	placed := make([]string, 0, len(files))
	for i, f := range files {
		dest := filepath.Join(destDir, f.name)
		if err := os.Rename(tmps[i], dest); err != nil {
			for _, t := range tmps[i:] {
				os.Remove(t)
			}
			return placed, fmt.Errorf("rename into %s failed after %d of %d files were already replaced — re-run the pull to finish; earlier verification aborts always leave everything unchanged: %w", dest, len(placed), len(files), err)
		}
		placed = append(placed, dest)
	}
	return placed, nil
}

// firstMissingAncestor walks upward from dir and returns the topmost
// path component that does not yet exist — "" when dir already exists.
// Any stat error other than not-exist stops the walk conservatively
// (treat as existing: better to leave a directory than to remove one we
// cannot account for).
func firstMissingAncestor(dir string) string {
	missing := ""
	for p := dir; ; {
		if _, err := os.Lstat(p); err == nil || !os.IsNotExist(err) {
			break
		}
		missing = p
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
		p = parent
	}
	return missing
}

// pullFormatDiagnostics renders VerifySignedProvenance's candidate-level
// diagnostics for the abort message — every collapsed cause by name,
// indented, or nothing when there are none.
func pullFormatDiagnostics(diags []string) string {
	if len(diags) == 0 {
		return ""
	}
	return "\n  " + strings.Join(diags, "\n  ")
}
