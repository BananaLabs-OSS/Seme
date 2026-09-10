package configurationmodule

import (
	"bytes"
	"strings"
	"testing"
)

func TestEmitDeterministicCompleteAndEnumsClosed(t *testing.T) {
	var a, b bytes.Buffer
	if Emit(&a) != nil || Emit(&b) != nil || !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("nondeterministic")
	}
	for _, x := range []string{ModuleID, RevisionID, "00000000000000000000000000004010", "00000000000000000000000000004173", "0000000000000000000000000000b003", "00000000000000000000000000009023", "00000000000000000000000000003001"} {
		if !strings.Contains(a.String(), x) {
			t.Fatalf("missing %s", x)
		}
	}
	if ValidateOrigin(3) == nil || ValidateLifecycle(5) == nil {
		t.Fatal("open enum")
	}
}
