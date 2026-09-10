package configurationmodule

import (
	"bytes"
	"os"
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

func TestEmitV2AddsTypedInitializerBindingsWithoutChangingV1(t *testing.T) {
	want, err := os.ReadFile("../../../modules/configuration/v1/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	var v1 bytes.Buffer
	if err := EmitVersion(&v1, 1); err != nil || !bytes.Equal(want, v1.Bytes()) {
		t.Fatalf("v1 changed: %v", err)
	}
	var a, b bytes.Buffer
	if EmitVersion(&a, 2) != nil || EmitVersion(&b, 2) != nil || !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("nondeterministic v2")
	}
	for _, id := range []string{RevisionV2ID, "00000000000000000000000000004018", "0000000000000000000000000000401f", "00000000000000000000000000004180", "000000000000000000000000000041f3"} {
		if !strings.Contains(a.String(), id) {
			t.Fatalf("missing %s", id)
		}
	}
	var v3 bytes.Buffer
	if err := EmitVersion(&v3, 3); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{RevisionV3ID, "pc 1\n" + RevisionV2ID, "0000000000000000000000000000b004", "00000000000000000000000000009024"} {
		if !strings.Contains(v3.String(), want) {
			t.Fatalf("v3 missing %s", want)
		}
	}
	if err := EmitVersion(&bytes.Buffer{}, 4); err == nil {
		t.Fatal("accepted unknown version")
	}
}
