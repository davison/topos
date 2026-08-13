package sdk

import (
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// protoRelPath is the published contract's source of truth, read relative
// to this test file's package directory (sdk/).
const protoRelPath = "../proto/topos/v1/plugin.proto"

// allowedRPCs is the PLUG-02 allowlist: SourcePlugin may declare exactly
// these four RPCs. This is deliberately an allowlist, not a blacklist of
// mutating verb names — a blacklist misses a creatively-named mutating
// RPC, whereas an allowlist fails the build on *any* addition until
// someone deliberately widens it here. See 01-04-PLAN.md Task 1.
var allowedRPCs = []string{"Describe", "Match", "Fetch", "Health"}

// TestContractRPCAllowlist is the mechanical PLUG-02 gate: it reads
// plugin.proto from disk, strips comments so a commented-out or
// string-literal RPC name can't influence the result, and asserts the
// service's RPC set is exactly allowedRPCs.
func TestContractRPCAllowlist(t *testing.T) {
	stripped := stripComments(readProto(t))

	if !strings.Contains(stripped, "service SourcePlugin") {
		t.Fatalf("PLUG-02 (01-04-PLAN.md): expected %s to declare \"service SourcePlugin\"", protoRelPath)
	}

	rpcRe := regexp.MustCompile(`rpc\s+(\w+)\s*\(`)
	matches := rpcRe.FindAllStringSubmatch(stripped, -1)

	got := make([]string, 0, len(matches))
	for _, m := range matches {
		got = append(got, m[1])
	}
	sort.Strings(got)

	want := append([]string(nil), allowedRPCs...)
	sort.Strings(want)

	if !equalStringSlices(got, want) {
		t.Fatalf(
			"PLUG-02 (01-04-PLAN.md): %s declares RPC set %v, want exactly the allowlisted set %v — "+
				"SourcePlugin is read-only by construction; if a new RPC is a deliberate, non-mutating "+
				"addition, widen allowedRPCs in sdk/contract_test.go as part of that change",
			protoRelPath, got, want,
		)
	}
}

// TestContractEnumsZeroValueUnspecified pins both enums' zero values to
// their _UNSPECIFIED variant, so a caller that forgets to set fidelity or
// variant fails closed (an empty/zero LinkFidelity or ContentVariant is
// never silently treated as a meaningful value).
func TestContractEnumsZeroValueUnspecified(t *testing.T) {
	stripped := stripComments(readProto(t))

	for _, enum := range []struct{ name, zero string }{
		{"LinkFidelity", "LINK_FIDELITY_UNSPECIFIED"},
		{"ContentVariant", "CONTENT_VARIANT_UNSPECIFIED"},
		{"ContentShape", "CONTENT_SHAPE_UNSPECIFIED"},
	} {
		if !strings.Contains(stripped, "enum "+enum.name) {
			t.Errorf("expected %s to declare enum %s", protoRelPath, enum.name)
			continue
		}
		zeroRe := regexp.MustCompile(regexp.QuoteMeta(enum.zero) + `\s*=\s*0\s*;`)
		if !zeroRe.MatchString(stripped) {
			t.Errorf("expected enum %s's zero value to be %s = 0", enum.name, enum.zero)
		}
	}
}

// TestContractDeclaresMatchVocabulary is Phase 5's contract-shape gate
// (option-a, D-04/D-05): the published proto must declare both
// DescribeResponse's match_vocabulary field and MatchRequest's match_fields
// map, and the retired "keywords" field name must appear only inside a
// `reserved` clause — never as a live field number — so a future edit
// cannot silently resurrect it under a new tag.
func TestContractDeclaresMatchVocabulary(t *testing.T) {
	stripped := stripComments(readProto(t))

	if !strings.Contains(stripped, "match_vocabulary") {
		t.Errorf("expected %s to declare match_vocabulary on DescribeResponse", protoRelPath)
	}
	if !strings.Contains(stripped, "match_fields") {
		t.Errorf("expected %s to declare match_fields on MatchRequest", protoRelPath)
	}

	reservedRe := regexp.MustCompile(`reserved\s+"keywords"\s*;`)
	if !reservedRe.MatchString(stripped) {
		t.Errorf("expected %s to retire the old \"keywords\" field name via a reserved clause", protoRelPath)
	}

	// "keywords" must not appear as a live field name (e.g.
	// "repeated string keywords = 1;") outside the reserved clause itself.
	liveFieldRe := regexp.MustCompile(`\bkeywords\s*=\s*\d+\s*;`)
	if liveFieldRe.MatchString(stripped) {
		t.Errorf("expected %s to NOT declare a live \"keywords\" field — it must remain retired via reserved only", protoRelPath)
	}
}

// TestContractDeclaresIconFields is Phase 9's contract-shape gate
// (09-01-PLAN.md Task 1, 09-UI-SPEC.md Fix 10): the published proto must
// declare DescribeResponse's icon (field 5) and icon_mime (field 6) — the
// additive icon-identity change every plugin's icon rides on — using the
// same read-the-.proto-source technique TestContractDeclaresMatchVocabulary
// above already established.
func TestContractDeclaresIconFields(t *testing.T) {
	stripped := stripComments(readProto(t))

	iconRe := regexp.MustCompile(`\bicon\s*=\s*5\s*;`)
	if !iconRe.MatchString(stripped) {
		t.Errorf("expected %s to declare DescribeResponse.icon = 5", protoRelPath)
	}

	iconMimeRe := regexp.MustCompile(`\bicon_mime\s*=\s*6\s*;`)
	if !iconMimeRe.MatchString(stripped) {
		t.Errorf("expected %s to declare DescribeResponse.icon_mime = 6", protoRelPath)
	}
}

// TestContractIconFieldsAreAdditive pins the two guarantees a proto3
// additive field change must uphold (09-01-PLAN.md Task 1): the RPC set is
// unchanged — icon/icon_mime are fields, not a new RPC, already enforced by
// TestContractRPCAllowlist above against the unchanged allowedRPCs — and
// sdk.Handshake.ProtocolVersion stays at 2, so a plugin binary built
// against the pre-Phase-9 contract (which never sets these two fields)
// keeps handshaking successfully exactly as it did before this phase.
func TestContractIconFieldsAreAdditive(t *testing.T) {
	if Handshake.ProtocolVersion != 2 {
		t.Errorf("expected sdk.Handshake.ProtocolVersion to remain 2 (icon fields are an additive proto3 change, not a wire break), got %d", Handshake.ProtocolVersion)
	}
}

// TestDescribeResponseIconFieldsZeroValue proves the pre-Phase-9-plugin
// case against the regenerated stubs directly: a DescribeResponse with
// neither icon field set (exactly what a plugin built against the old
// contract produces) reports empty bytes and an empty mime string, never a
// nil-pointer panic or a non-empty zero value.
func TestDescribeResponseIconFieldsZeroValue(t *testing.T) {
	desc := &toposv1.DescribeResponse{}

	if got := desc.GetIcon(); len(got) != 0 {
		t.Errorf("expected GetIcon() on an icon-less DescribeResponse to return empty bytes, got %d bytes", len(got))
	}
	if got := desc.GetIconMime(); got != "" {
		t.Errorf("expected GetIconMime() on an icon-less DescribeResponse to return \"\", got %q", got)
	}
}

// TestContractDeclaresExtrasDeclaration is Phase 11's contract-shape gate
// (11-02-PLAN.md Task 3, D-15): the published proto must declare
// DescribeResponse.extras at field 7 (the next free number after icon_mime
// at 6) and the ExtrasField message's five fields, mirroring the icon-field
// guards above — read the .proto source, never trust the generated Go
// stubs alone to prove what the PUBLISHED contract commits to.
func TestContractDeclaresExtrasDeclaration(t *testing.T) {
	stripped := stripComments(readProto(t))

	extrasFieldRe := regexp.MustCompile(`\bextras\s*=\s*7\s*;`)
	if !extrasFieldRe.MatchString(stripped) {
		t.Errorf("expected %s to declare DescribeResponse.extras = 7", protoRelPath)
	}

	if !strings.Contains(stripped, "message ExtrasField") {
		t.Fatalf("expected %s to declare message ExtrasField", protoRelPath)
	}

	for _, want := range []struct {
		name string
		num  int
	}{
		{"key", 1}, {"label", 2}, {"required", 3}, {"secret", 4}, {"placeholder", 5},
	} {
		fieldRe := regexp.MustCompile(`\b` + regexp.QuoteMeta(want.name) + `\s*=\s*` + strconv.Itoa(want.num) + `\s*;`)
		if !fieldRe.MatchString(stripped) {
			t.Errorf("expected %s's ExtrasField message to declare %s = %d", protoRelPath, want.name, want.num)
		}
	}

	// Fields 1-6 on DescribeResponse must remain untouched by this
	// additive change — the same five checks TestContractDeclaresIconFields
	// and TestContractDeclaresMatchVocabulary above already pin, repeated
	// here so this guard alone is sufficient evidence the extras addition
	// didn't renumber anything.
	for _, want := range []struct {
		name string
		num  int
	}{
		{"source_type", 1}, {"display_name", 2}, {"contract_version", 3},
		{"match_vocabulary", 4}, {"icon", 5}, {"icon_mime", 6},
	} {
		fieldRe := regexp.MustCompile(`\b` + regexp.QuoteMeta(want.name) + `\s*=\s*` + strconv.Itoa(want.num) + `\s*;`)
		if !fieldRe.MatchString(stripped) {
			t.Errorf("expected DescribeResponse's pre-existing field %s = %d to remain unchanged", want.name, want.num)
		}
	}
}

// TestContractExtrasFieldIsAdditive pins the two guarantees a proto3
// additive field change must uphold, mirroring
// TestContractIconFieldsAreAdditive: the RPC set is unchanged (already
// enforced by TestContractRPCAllowlist against the unchanged allowedRPCs),
// and sdk.Handshake.ProtocolVersion stays at 2, so a plugin binary built
// against the pre-extras contract (which never sets this field) keeps
// handshaking successfully exactly as it did before this phase.
func TestContractExtrasFieldIsAdditive(t *testing.T) {
	if Handshake.ProtocolVersion != 2 {
		t.Errorf("expected sdk.Handshake.ProtocolVersion to remain 2 (extras is an additive proto3 change, not a wire break), got %d", Handshake.ProtocolVersion)
	}
}

// TestDescribeResponseExtrasFieldZeroValue proves the pre-extras-contract
// plugin case against the regenerated stubs directly: a DescribeResponse
// with no extras set (exactly what a plugin built against the old contract
// produces) reports a nil/empty slice, never a nil-pointer panic.
func TestDescribeResponseExtrasFieldZeroValue(t *testing.T) {
	desc := &toposv1.DescribeResponse{}
	if got := desc.GetExtras(); len(got) != 0 {
		t.Errorf("expected GetExtras() on an extras-less DescribeResponse to return an empty slice, got %d entries", len(got))
	}
}

func readProto(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(protoRelPath)
	if err != nil {
		t.Fatalf("read %s: %v", protoRelPath, err)
	}
	return string(b)
}

// stripComments removes // line comments and /* */ block comments from a
// .proto source so a commented-out rpc line, or an RPC/enum name mentioned
// only in prose, cannot influence a regexp-based structural check.
func stripComments(src string) string {
	blockRe := regexp.MustCompile(`(?s)/\*.*?\*/`)
	src = blockRe.ReplaceAllString(src, "")
	lineRe := regexp.MustCompile(`//[^\n]*`)
	return lineRe.ReplaceAllString(src, "")
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
