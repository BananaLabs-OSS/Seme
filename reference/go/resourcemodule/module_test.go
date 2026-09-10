package resourcemodule

import (
	"bytes"
	"strings"
	"testing"
)

func TestEmitDeterministicNeutralAndPinned(t *testing.T) {
	var a, b bytes.Buffer
	if Emit(&a) != nil || Emit(&b) != nil || a.String() != b.String() {
		t.Fatal("nondeterministic")
	}
	for _, x := range []string{ModuleID, RevisionID, "0000000000000000000000000000b004", "00000000000000000000000000006010", "00000000000000000000000000006131"} {
		if !strings.Contains(a.String(), x) {
			t.Fatalf("missing %s", x)
		}
	}
	for _, x := range []string{"go", "javascript", "filesystem", "decoder", "raw_source"} {
		if strings.Contains(strings.ToLower(a.String()), x) {
			t.Fatalf("contamination %s", x)
		}
	}
}
func TestClosedKind(t *testing.T) {
	if ValidateRepresentationKind(0) != nil || ValidateRepresentationKind(1) != nil || ValidateRepresentationKind(2) == nil {
		t.Fatal("bad closed enum")
	}
}
