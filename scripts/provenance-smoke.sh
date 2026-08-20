#!/usr/bin/env bash
# Hermetic round-trip gate for the signed release-manifest trust arm
# (16-01-PLAN.md Task 3, D-01/D-08/D-12): proves the producer
# (topos-provenance sign) and the verifier (topos-provenance verify —
# the SAME kernel/pluginhost.VerifySignedProvenance the launch gate
# calls) agree end to end, INCLUDING the link-time key-injection seam
# (provenanceKeysExtra) a dev/e2e build uses to trust its own locally
# signed fixtures — rebuilding the CLI with -ldflags is itself the
# executable proof that seam works, not an in-process test-only
# override.
#
# House style follows scripts/dev-guard-smoke.sh /
# scripts/install-smoke.sh: required-tool preflight, mktemp -d work dir,
# cleanup trap, loud FAIL: messages naming the specific violation. Needs
# no network and no config; safe to run while a real kernel is up on
# 127.0.0.1:7777 — every path this script touches lives under its own
# temp directory.
#
# Sequence: build fixtures -> keygen -> relink CLI with the throwaway
# key -> sign -> verify (must succeed, naming the manifest) -> tamper
# the binary (must refuse, naming the binary) -> restore, delete the
# signature (must refuse) -> restore, flip a byte in the manifest JSON
# (must refuse — the signature covers raw bytes, not a re-serialized
# form). Prints a single "provenance-smoke: OK" line on stdout on full
# success and nothing else, matching the quiet-on-success convention the
# other smoke scripts use.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

for tool in go sed; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "FAIL: required tool '$tool' not found on PATH" >&2
    exit 1
  fi
done

WORK="$(mktemp -d -t topos-provenance-smoke.XXXXXX)"
cleanup() {
  rm -rf "$WORK"
}
trap cleanup EXIT

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

# ---------------------------------------------------------------------
# Step 1: build topos-provenance and one real plugin binary into WORK.
# ---------------------------------------------------------------------
go build -o "$WORK/topos-provenance" ./cmd/topos-provenance \
  || fail "build topos-provenance"
go build -o "$WORK/topos-plugin-mock" ./plugins/mock \
  || fail "build topos-plugin-mock fixture"

# ---------------------------------------------------------------------
# Step 2: keygen a throwaway key into WORK, capturing the printed spec.
# ---------------------------------------------------------------------
mkdir -p "$WORK/keys"
KEY_SPEC="$("$WORK/topos-provenance" keygen --key-id smoke --out-dir "$WORK/keys")" \
  || fail "keygen failed"
[ -n "$KEY_SPEC" ] || fail "keygen printed an empty key spec"

# ---------------------------------------------------------------------
# Step 3: rebuild topos-provenance with -ldflags injecting that spec
# into provenanceKeysExtra — this step is ALSO the executable proof
# that the link-time key-injection seam (D-12) works, not merely an
# in-process test override.
# ---------------------------------------------------------------------
go build -ldflags "-X github.com/davison/topos/kernel/pluginhost.provenanceKeysExtra=$KEY_SPEC" \
  -o "$WORK/topos-provenance-linked" ./cmd/topos-provenance \
  || fail "rebuild topos-provenance with -ldflags provenanceKeysExtra"

# ---------------------------------------------------------------------
# Step 4: sign the plugin binary into WORK, then place the manifest and
# its signature beside the binary — the on-disk layout VerifySignedProvenance
# scans (D-07).
# ---------------------------------------------------------------------
mkdir -p "$WORK/release"
"$WORK/topos-provenance" sign \
  --key-id smoke \
  --repo davison/topos-plugins \
  --tag v0.0.0-smoke \
  --version 0.0.0 \
  --key-file "$WORK/keys/smoke.key" \
  --out-dir "$WORK/release" \
  "$WORK/topos-plugin-mock" \
  >"$WORK/sign.out" 2>&1 \
  || fail "sign failed: $(cat "$WORK/sign.out")"

MANIFEST_PATH="$(find "$WORK/release" -maxdepth 1 -name '*.provenance.json')"
SIG_PATH="$(find "$WORK/release" -maxdepth 1 -name '*.provenance.sig')"
[ -n "$MANIFEST_PATH" ] || fail "sign did not produce a .provenance.json file"
[ -n "$SIG_PATH" ] || fail "sign did not produce a .provenance.sig file"

MANIFEST_NAME="$(basename "$MANIFEST_PATH")"
SIG_NAME="$(basename "$SIG_PATH")"
cp "$MANIFEST_PATH" "$WORK/$MANIFEST_NAME"
cp "$SIG_PATH" "$WORK/$SIG_NAME"

# Pristine copies for the restore steps between cases 6 -> 7 -> 8, each
# of which tampers exactly one artifact and must undo it before the
# next case runs. Kept in a SEPARATE directory, never inside $WORK
# itself — a pristine copy named "topos-plugin-mock.pristine" sitting
# directly in $WORK would still carry the topos-plugin- prefix and get
# picked up as a second (unsigned) candidate binary by verify's own
# directory scan.
mkdir -p "$WORK/backup"
cp "$WORK/topos-plugin-mock" "$WORK/backup/topos-plugin-mock"
cp "$WORK/$SIG_NAME" "$WORK/backup/$SIG_NAME"
cp "$WORK/$MANIFEST_NAME" "$WORK/backup/$MANIFEST_NAME"

# ---------------------------------------------------------------------
# Step 5: verify --dir <temp> must exit 0 and name the manifest file.
# ---------------------------------------------------------------------
VERIFY_OUT="$("$WORK/topos-provenance-linked" verify --dir "$WORK")" \
  || fail "verify of an untampered signed binary failed: $VERIFY_OUT"
case "$VERIFY_OUT" in
*"$MANIFEST_NAME"*) ;;
*) fail "verify output did not name the vouching manifest file: $VERIFY_OUT" ;;
esac

# ---------------------------------------------------------------------
# Step 6: append a byte to the plugin binary, then verify again — must
# exit non-zero and name the binary in its output.
# ---------------------------------------------------------------------
printf 'X' >>"$WORK/topos-plugin-mock"
if "$WORK/topos-provenance-linked" verify --dir "$WORK" >"$WORK/tamper-binary.out" 2>&1; then
  fail "verify unexpectedly succeeded after tampering the binary"
fi
grep -q "topos-plugin-mock" "$WORK/tamper-binary.out" \
  || fail "tampered-binary verify output did not name the binary: $(cat "$WORK/tamper-binary.out")"

# Restore the pristine binary before the next case.
cp "$WORK/backup/topos-plugin-mock" "$WORK/topos-plugin-mock"

# ---------------------------------------------------------------------
# Step 7: delete the .provenance.sig file, then verify again — must
# exit non-zero (a manifest with no signature is not evidence).
# ---------------------------------------------------------------------
rm -f "$WORK/$SIG_NAME"
if "$WORK/topos-provenance-linked" verify --dir "$WORK" >"$WORK/no-sig.out" 2>&1; then
  fail "verify unexpectedly succeeded with no signature file present"
fi

# Restore the signature before the next case.
cp "$WORK/backup/$SIG_NAME" "$WORK/$SIG_NAME"

# ---------------------------------------------------------------------
# Step 8: flip one character inside the manifest JSON (a value byte,
# never breaking JSON syntax), then verify again — must exit non-zero.
# The signature covers the raw bytes verbatim, so a still-syntactically-
# valid edit must still be caught.
# ---------------------------------------------------------------------
sed -i 's/v0\.0\.0-smoke/v0.0.0-smokE/' "$WORK/$MANIFEST_NAME"
if "$WORK/topos-provenance-linked" verify --dir "$WORK" >"$WORK/tamper-manifest.out" 2>&1; then
  fail "verify unexpectedly succeeded after tampering the manifest JSON"
fi

# Restore the manifest so a second run of this script (or a stray
# re-invocation of just `verify` against this WORK dir) starts clean.
cp "$WORK/backup/$MANIFEST_NAME" "$WORK/$MANIFEST_NAME"

echo "provenance-smoke: OK"
