// Command topos-provenance is the release-side and operator-side CLI
// over the signed release-manifest trust arm
// (kernel/pluginhost/provenance.go, 16-01-PLAN.md Task 2, D-01/D-02/D-09):
// every subcommand wraps the SAME exported producers and verifier the
// kernel launch gate itself uses — this tool never reimplements the
// manifest format or the signature scheme.
//
// Subcommands:
//
//	keygen   generate an ed25519 signing keypair, print the public
//	         half's -ldflags -X spec (never the private key)
//	sign     hash plugin binaries, build and sign a release manifest,
//	         write the <repo>-<tag>.provenance.json/.sig pair
//	verify   verify every (or named) plugin binary in a directory
//	         against THIS BINARY'S OWN compiled-in accepted key set —
//	         a real verification against the real key policy, never a
//	         self-signed check that trusts whatever key the manifest
//	         names. This is the entry point install-time verification
//	         (D-09) calls in 16-05-PLAN.md.
//
// Called with no subcommand, or an unknown one, it refuses to run and
// exits non-zero with a usage line on stderr — mirroring
// cmd/topos-manifest's zero-argument refusal exactly.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/davison/topos/kernel/pluginhost"
	"github.com/davison/topos/sdk"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "topos-provenance: refusing to run with no subcommand — a mis-wired build/release recipe must fail loudly")
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "keygen":
		err = runKeygen(os.Args[2:])
	case "sign":
		err = runSign(os.Args[2:])
	case "verify":
		err = runVerify(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "topos-provenance: unknown subcommand %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "topos-provenance: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: topos-provenance <keygen|sign|verify> [flags]")
}

// runKeygen generates an ed25519 keypair, writes the private key at mode
// 0600 (standard base64 of the seed, single line, trailing newline) and
// the public key alongside it, and prints ONLY the new public key's
// FormatProvenanceKeys spec to stdout — the exact string a build recipe
// passes to -ldflags -X ...provenanceKeysExtra=<spec> (D-12) and the
// value 16-04-PLAN.md embeds. The private key is never printed to stdout
// or stderr and never logged.
func runKeygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	keyID := fs.String("key-id", "", "key id to embed in the generated key spec (required)")
	outDir := fs.String("out-dir", "", "directory to write <key-id>.key and <key-id>.pub into (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*keyID) == "" {
		return errors.New("keygen: --key-id is required")
	}
	if err := pluginhost.ValidateProvenanceKeyID(*keyID); err != nil {
		return fmt.Errorf("keygen: %w", err)
	}
	if strings.TrimSpace(*outDir) == "" {
		return errors.New("keygen: --out-dir is required")
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("keygen: generate key: %w", err)
	}

	privPath := filepath.Join(*outDir, *keyID+".key")
	pubPath := filepath.Join(*outDir, *keyID+".pub")

	privLine := base64.StdEncoding.EncodeToString(priv.Seed()) + "\n"
	if err := os.WriteFile(privPath, []byte(privLine), 0o600); err != nil {
		return fmt.Errorf("keygen: write private key: %w", err)
	}
	pubLine := base64.StdEncoding.EncodeToString(pub) + "\n"
	if err := os.WriteFile(pubPath, []byte(pubLine), 0o644); err != nil {
		return fmt.Errorf("keygen: write public key: %w", err)
	}

	fmt.Println(pluginhost.FormatProvenanceKeys([]pluginhost.ProvenanceKey{{ID: *keyID, PublicKey: pub}}))
	return nil
}

// runSign hashes every positional binary-path argument through
// pluginhost.ManifestEntriesForBinaries (reusing the one-hashing path,
// never a second implementation), builds the release manifest via
// BuildProvenanceManifest, signs those exact bytes via
// SignProvenanceManifest, and writes
// <out-dir>/<repo-basename>-<tag>.provenance.json and its
// .provenance.sig sibling. Zero binary paths is a loud, non-zero refusal
// — mirroring cmd/topos-manifest's own RESEARCH Pitfall 6 discipline.
func runSign(args []string) error {
	fs := flag.NewFlagSet("sign", flag.ContinueOnError)
	keyID := fs.String("key-id", "", "key id that signs this release (required)")
	repo := fs.String("repo", "", "release repo, e.g. davison/topos-plugins (required)")
	tag := fs.String("tag", "", "release tag (required)")
	version := fs.String("version", "", "plugin version recorded in every entry (required)")
	contract := fs.String("contract", sdk.ContractVersion, "gRPC contract generation recorded in every entry (default: the generation this kernel build speaks; recorded metadata — the launch gate reads the plugin's own Describe declaration, docs/plugin-trust.md)")
	goos := fs.String("os", runtime.GOOS, "release platform OS")
	arch := fs.String("arch", runtime.GOARCH, "release platform arch")
	outDir := fs.String("out-dir", "", "directory to write the manifest and signature into (required)")
	keyFile := fs.String("key-file", "", "path to a base64-encoded ed25519 private key seed (mutually exclusive with --key-env; --key-file wins if both are set)")
	keyEnv := fs.String("key-env", "TOPOS_PROVENANCE_SIGNING_KEY", "environment variable holding the base64-encoded ed25519 private key seed — so CI passes the secret through the environment and never through argv, where it would appear in a process listing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	binaries := fs.Args()
	if len(binaries) == 0 {
		return errors.New("sign: refusing to run with zero binary-path arguments — a mis-wired release recipe must fail loudly, not silently produce an empty manifest")
	}
	if strings.TrimSpace(*keyID) == "" {
		return errors.New("sign: --key-id is required")
	}
	if err := pluginhost.ValidateProvenanceKeyID(*keyID); err != nil {
		return fmt.Errorf("sign: %w", err)
	}
	if strings.TrimSpace(*repo) == "" {
		return errors.New("sign: --repo is required")
	}
	if strings.TrimSpace(*tag) == "" {
		return errors.New("sign: --tag is required")
	}
	if strings.TrimSpace(*version) == "" {
		return errors.New("sign: --version is required")
	}
	if strings.TrimSpace(*outDir) == "" {
		return errors.New("sign: --out-dir is required")
	}

	priv, err := loadSigningKey(*keyFile, *keyEnv)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}

	hashes, err := pluginhost.ManifestEntriesForBinaries(binaries...)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}

	entries := make([]pluginhost.ProvenanceEntry, 0, len(hashes))
	for name, hash := range hashes {
		entries = append(entries, pluginhost.ProvenanceEntry{
			Name:     name,
			SHA256:   hash,
			Version:  *version,
			Contract: *contract,
		})
	}

	release := pluginhost.ProvenanceRelease{Repo: *repo, Tag: *tag, OS: *goos, Arch: *arch}
	manifestBytes, err := pluginhost.BuildProvenanceManifest(release, entries)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}
	sigBytes, err := pluginhost.SignProvenanceManifest(manifestBytes, *keyID, priv)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}

	basename := filepath.Base(*repo) + "-" + *tag
	manifestPath := filepath.Join(*outDir, basename+pluginhost.ProvenanceManifestSuffix)
	sigPath := filepath.Join(*outDir, basename+pluginhost.ProvenanceSignatureSuffix)
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		return fmt.Errorf("sign: write manifest: %w", err)
	}
	if err := os.WriteFile(sigPath, sigBytes, 0o644); err != nil {
		return fmt.Errorf("sign: write signature: %w", err)
	}

	fmt.Println(manifestPath)
	fmt.Println(sigPath)
	return nil
}

// loadSigningKey reads a base64-encoded ed25519 private key SEED (the
// same shape keygen writes to <key-id>.key) from keyFile when set,
// otherwise from the keyEnv environment variable — never from a
// command-line argument, where it would appear in a process listing.
func loadSigningKey(keyFile, keyEnv string) (ed25519.PrivateKey, error) {
	var encoded string
	if keyFile != "" {
		raw, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("read --key-file %s: %w", keyFile, err)
		}
		encoded = strings.TrimSpace(string(raw))
	} else {
		encoded = strings.TrimSpace(os.Getenv(keyEnv))
		if encoded == "" {
			return nil, fmt.Errorf("no signing key: --key-file was not given and environment variable %s is empty", keyEnv)
		}
	}

	seed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode signing key: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("signing key is %d bytes, want %d (an ed25519 seed, as keygen writes)", len(seed), ed25519.SeedSize)
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

// nameList is a repeatable flag.Value for --name.
type nameList []string

func (n *nameList) String() string { return strings.Join(*n, ",") }
func (n *nameList) Set(v string) error {
	*n = append(*n, v)
	return nil
}

// runVerify verifies every (or every explicitly --name'd) plugin binary
// in --dir against pluginhost.VerifySignedProvenance, using THIS
// BINARY'S OWN compiled-in accepted key set — a real verification
// against the real key policy, never a self-signed check that trusts
// whatever key a manifest happens to name. It prints one line per binary
// naming the binary and either the manifest file that vouched for it or
// the specific failure cause, and exits non-zero if ANY named binary
// failed to verify, or if --dir contains no plugin binaries at all (a
// verify pass over nothing must never report success).
func runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	dir := fs.String("dir", "", "plugin directory to verify (required)")
	var names nameList
	fs.Var(&names, "name", "plugin binary name to verify (repeatable); defaults to every topos-plugin-* binary in --dir")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*dir) == "" {
		return errors.New("verify: --dir is required")
	}

	targets := []string(names)
	if len(targets) == 0 {
		entries, err := os.ReadDir(*dir)
		if err != nil {
			return fmt.Errorf("verify: read --dir %s: %w", *dir, err)
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasPrefix(e.Name(), pluginhost.PluginBinaryPrefix) {
				targets = append(targets, e.Name())
			}
		}
	}
	if len(targets) == 0 {
		return fmt.Errorf("verify: %s contains no plugin binaries — a verify pass over nothing is never a success", *dir)
	}
	sort.Strings(targets)

	dirs := pluginhost.Dirs{Trusted: *dir}
	anyFailed := false
	for _, name := range targets {
		path := filepath.Join(*dir, name)
		_, evidence, _, err := pluginhost.VerifySignedProvenance(dirs, name, path)
		switch {
		case err != nil:
			anyFailed = true
			fmt.Printf("%s: FAIL: %v\n", name, err)
		case evidence == "":
			anyFailed = true
			fmt.Printf("%s: FAIL: not named by any validly-signed release manifest in %s\n", name, *dir)
		default:
			fmt.Printf("%s: OK (%s)\n", name, evidence)
		}
	}

	if anyFailed {
		return errors.New("one or more binaries failed provenance verification")
	}
	return nil
}
