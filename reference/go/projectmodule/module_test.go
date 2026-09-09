package projectmodule

import (
	"bytes"
	"strings"
	"testing"
)

func TestEmitUsesFreshIdentityAndExactImports(t *testing.T) {
	var first, second bytes.Buffer
	if err := Emit(&first); err != nil {
		t.Fatal(err)
	}
	if err := Emit(&second); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Fatal("nondeterministic module")
	}
	for _, value := range []string{ModuleID, "0000000000000000000000000000b001", "00000000000000000000000000009023", "fi 00000000000000000000000000000121 li 2"} {
		if !strings.Contains(first.String(), value) {
			t.Fatalf("missing %s", value)
		}
	}
	if strings.Contains(first.String(), "0000000000000000000000000000c000") {
		t.Fatal("collides with Target Contract v1")
	}
}
