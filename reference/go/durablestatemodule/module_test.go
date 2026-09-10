package durablestatemodule

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
	for _, s := range []string{ModuleID, RevisionID, "0000000000000000000000000000b004", "00000000000000000000000000009024", "00000000000000000000000000003001", "00000000000000000000000000008010", "00000000000000000000000000008159"} {
		if !strings.Contains(a.String(), s) {
			t.Fatalf("missing %s", s)
		}
	}
	for _, s := range []string{"filesystem", "database", "go", "javascript", "sql"} {
		if strings.Contains(strings.ToLower(a.String()), s) {
			t.Fatalf("contamination %s", s)
		}
	}
}
