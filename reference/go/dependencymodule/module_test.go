package dependencymodule

import (
	"bytes"
	"testing"
)

func TestDeterministicAndClosedKind(t *testing.T) {
	var a, b bytes.Buffer
	if Emit(&a) != nil || Emit(&b) != nil || !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("nondeterministic")
	}
	if ValidateKind(0) != nil || ValidateKind(1) != nil || ValidateKind(2) == nil {
		t.Fatal("kind range")
	}
	for _, identity := range []string{"0000000000000000000000000000f010", "0000000000000000000000000000f011", "0000000000000000000000000000f012", "0000000000000000000000000000f016", "0000000000000000000000000000f017"} {
		if !bytes.Contains(a.Bytes(), []byte(identity)) {
			t.Fatalf("missing %s", identity)
		}
	}
}
