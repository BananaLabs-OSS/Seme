package controlledeffectsmodule

import (
	"bytes"
	"strings"
	"testing"
)

func TestEmitDeterministicNeutralAndBounded(t *testing.T) {
	var first, second bytes.Buffer
	if Emit(&first) != nil || Emit(&second) != nil || first.String() != second.String() {
		t.Fatal("nondeterministic")
	}
	for _, value := range []string{ModuleID, RevisionID, "00000000000000000000000000013100", "00000000000000000000000000013108", "00000000000000000000000000013283", "00000000000000000000000000013287"} {
		if !strings.Contains(first.String(), value) {
			t.Fatalf("missing %s", value)
		}
	}
	lower := strings.ToLower(first.String())
	for _, forbidden := range []string{"time.now", "math/rand", "crypto/rand", "filesystem", "database", "javascript", "golang", "workbench"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("mechanism contamination %s", forbidden)
		}
	}
}
