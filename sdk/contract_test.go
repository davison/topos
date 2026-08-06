package sdk

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
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
