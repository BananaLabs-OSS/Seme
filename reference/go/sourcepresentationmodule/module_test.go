package sourcepresentationmodule

import (
	"bytes"
	"strings"
	"testing"
)

func TestEmitDeterministicNeutralContract(t *testing.T) {
	var a, b bytes.Buffer
	if err := Emit(&a); err != nil {
		t.Fatal(err)
	}
	if err := Emit(&b); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("nondeterministic")
	}
	for _, want := range []string{ModuleID, RevisionID, "0000000000000000000000000000e00b", "00000000000000000000000000001010", "00000000000000000000000000001116"} {
		if !strings.Contains(a.String(), want) {
			t.Fatalf("missing %s", want)
		}
	}
	for _, bad := range []string{"golang", "javascript", "typescript", "workbench", "workshop"} {
		if strings.Contains(strings.ToLower(a.String()), bad) {
			t.Fatalf("contamination %s", bad)
		}
	}
}
func TestVisibilityClosed(t *testing.T) {
	if ValidateVisibility(0) != nil || ValidateVisibility(1) != nil {
		t.Fatal("valid visibility")
	}
	if ValidateVisibility(2) == nil {
		t.Fatal("open visibility")
	}
}
