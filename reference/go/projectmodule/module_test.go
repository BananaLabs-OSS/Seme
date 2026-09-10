package projectmodule

import (
	"bytes"
	"fmt"
	"os"
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

func TestEmitV2ExtendsV1WithExactAncestryAndNeutralSources(t *testing.T) {
	var first, second bytes.Buffer
	if err := EmitVersion(&first, 2); err != nil {
		t.Fatal(err)
	}
	if err := EmitVersion(&second, 2); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Fatal("nondeterministic v2")
	}
	for _, want := range []string{"rv " + RevisionV2ID, "pc 1\n" + RevisionID, "en " + ModuleID + " 00000000000000000000000000000012 2 3", "fi 00000000000000000000000000000122 li 30", "0000000000000000000000000000e012", "0000000000000000000000000000e164", "toolchain_profile.language", "source_classification.code", "source_unit.normalized_relative_path", "source_unit.preservation_mode", "source_inventory.inventory_revision", "source_inventory.semantic_revision", "source_inventory.units"} {
		value := want
		if strings.Contains(want, ".") {
			value = fmt.Sprintf("%x", want)
		}
		if !strings.Contains(first.String(), value) {
			t.Fatalf("missing %s", want)
		}
	}
	for _, bad := range []string{"go_source", "javascript_source", "workbench", "workshop", "ironclad"} {
		if strings.Contains(strings.ToLower(first.String()), bad) {
			t.Fatalf("language/project-specific contamination %q", bad)
		}
	}
	if err := EmitVersion(&bytes.Buffer{}, 12); err == nil {
		t.Fatal("accepted unknown version")
	}
}

func TestEmitV11BindsExactV10AndTransport(t *testing.T) {
	var first, second bytes.Buffer
	if err := EmitVersion(&first, 11); err != nil {
		t.Fatal(err)
	}
	if err := EmitVersion(&second, 11); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("nondeterministic v11")
	}
	for _, want := range []string{RevisionV11ID, "pc 1\n" + RevisionV10ID,
		"00000000000000000000000000002000", "00000000000000000000000000002001",
		"0000000000000000000000000000e026", "0000000000000000000000000000e260",
		"0000000000000000000000000000e261", "0000000000000000000000000000e262",
	} {
		if !strings.Contains(first.String(), want) {
			t.Fatalf("v11 missing %s", want)
		}
	}
	want, err := os.ReadFile("../../../modules/project/v10/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err = EmitVersion(&got, 10); err != nil || !bytes.Equal(want, got.Bytes()) {
		t.Fatalf("v10 changed: %v", err)
	}
}

func TestEmitV11KeepsEntityIdentitiesSorted(t *testing.T) {
	var out bytes.Buffer
	if err := EmitVersion(&out, 11); err != nil {
		t.Fatal(err)
	}
	last := ""
	for _, line := range strings.Split(out.String(), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 0 || parts[0] != "en" {
			continue
		}
		if last != "" && parts[1] <= last {
			t.Fatalf("entity order %s after %s", parts[1], last)
		}
		last = parts[1]
	}
}

func TestEmitV10BindsExactV9DurablePlanAndPresentation(t *testing.T) {
	var out bytes.Buffer
	if err := EmitVersion(&out, 10); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{RevisionV10ID, "pc 1\n" + RevisionV9ID, "00000000000000000000000000008000", "00000000000000000000000000008001", "00000000000000000000000000001000", "00000000000000000000000000001001", "0000000000000000000000000000e025", "0000000000000000000000000000e250", "0000000000000000000000000000e252", "0000000000000000000000000000e253"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("v10 missing %s", want)
		}
	}
	want, err := os.ReadFile("../../../modules/project/v9/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err = EmitVersion(&got, 9); err != nil || !bytes.Equal(want, got.Bytes()) {
		t.Fatalf("v9 changed: %v", err)
	}
}

func TestEmitV10KeepsEntityIdentitiesSortedWhenAddingLateImport(t *testing.T) {
	var out bytes.Buffer
	if err := EmitVersion(&out, 10); err != nil {
		t.Fatal(err)
	}
	last := ""
	for _, line := range strings.Split(out.String(), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 0 || parts[0] != "en" {
			continue
		}
		if last != "" && parts[1] <= last {
			t.Fatalf("entity order %s after %s", parts[1], last)
		}
		last = parts[1]
	}
}

func TestEmitV9BindsResourcesWithoutChangingV8(t *testing.T) {
	var out bytes.Buffer
	if err := EmitVersion(&out, 9); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{RevisionV9ID, "pc 1\n" + RevisionV8ID, "00000000000000000000000000006001", "0000000000000000000000000000e024", "0000000000000000000000000000e240", "0000000000000000000000000000e242"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("v9 missing %s", want)
		}
	}
	want, err := os.ReadFile("../../../modules/project/v8/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	var old bytes.Buffer
	if err = EmitVersion(&old, 8); err != nil || !bytes.Equal(want, old.Bytes()) {
		t.Fatalf("v8 changed: %v", err)
	}
}

func TestEmitV8BindsOneFullExecutionV36Snapshot(t *testing.T) {
	var out bytes.Buffer
	if err := EmitVersion(&out, 8); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{RevisionV8ID, "pc 1\n" + RevisionV7ID, "0000000000000000000000000000b004", "00000000000000000000000000009024", "00000000000000000000000000004006", "00000000000000000000000000003001", "0000000000000000000000000000e023", "0000000000000000000000000000e230", "0000000000000000000000000000e235"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("v8 missing %s", want)
		}
	}
}

func TestEmitV7BindsConfigurationV2WithoutChangingEarlierRevisions(t *testing.T) {
	var out bytes.Buffer
	if err := EmitVersion(&out, 7); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"rv " + RevisionV7ID, "pc 1\n" + RevisionV6ID, "00000000000000000000000000004005", "0000000000000000000000000000e022", "0000000000000000000000000000e220", "0000000000000000000000000000e221", "0000000000000000000000000000e222"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %s", want)
		}
	}
	for version := 1; version <= 6; version++ {
		want, err := os.ReadFile(fmt.Sprintf("../../../modules/project/v%d/module.g1", version))
		if err != nil {
			t.Fatal(err)
		}
		var got bytes.Buffer
		if err = EmitVersion(&got, version); err != nil || !bytes.Equal(want, got.Bytes()) {
			t.Fatalf("v%d changed: %v", version, err)
		}
	}
}

func TestEmitV5BindsCompletePackageGraph(t *testing.T) {
	var out bytes.Buffer
	if err := EmitVersion(&out, 5); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"rv " + RevisionV5ID, "pc 1\n" + RevisionV4ID, "0000000000000000000000000000b003", "0000000000000000000000000000e020", "0000000000000000000000000000e200", "0000000000000000000000000000e201", "0000000000000000000000000000e202"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %s", want)
		}
	}
	for version := 1; version <= 4; version++ {
		want, err := os.ReadFile(fmt.Sprintf("../../../modules/project/v%d/module.g1", version))
		if err != nil {
			t.Fatal(err)
		}
		var got bytes.Buffer
		if err = EmitVersion(&got, version); err != nil || !bytes.Equal(want, got.Bytes()) {
			t.Fatalf("v%d changed: %v", version, err)
		}
	}
}

func TestEmitV6BindsConfigurationWithoutChangingEarlierRevisions(t *testing.T) {
	var out bytes.Buffer
	if err := EmitVersion(&out, 6); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"rv " + RevisionV6ID, "pc 1\n" + RevisionV5ID, "00000000000000000000000000004000", "00000000000000000000000000004001", "0000000000000000000000000000e021", "0000000000000000000000000000e210", "0000000000000000000000000000e211", "0000000000000000000000000000e212"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %s", want)
		}
	}
	for version := 1; version <= 5; version++ {
		want, err := os.ReadFile(fmt.Sprintf("../../../modules/project/v%d/module.g1", version))
		if err != nil {
			t.Fatal(err)
		}
		var got bytes.Buffer
		if err = EmitVersion(&got, version); err != nil || !bytes.Equal(want, got.Bytes()) {
			t.Fatalf("v%d changed: %v", version, err)
		}
	}
}

func TestEmitV4BindsDependencyClosure(t *testing.T) {
	var out bytes.Buffer
	if err := EmitVersion(&out, 4); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"rv " + RevisionV4ID, "pc 1\n" + RevisionV3ID, "fi 00000000000000000000000000000121 li 3", "0000000000000000000000000000f000", "0000000000000000000000000000f001", "0000000000000000000000000000e019", "0000000000000000000000000000e190", "0000000000000000000000000000e191", "0000000000000000000000000000e192"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %s", want)
		}
	}
	for version, path := range map[int]string{1: "../../../modules/project/v1/module.g1", 2: "../../../modules/project/v2/module.g1", 3: "../../../modules/project/v3/module.g1"} {
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var got bytes.Buffer
		if err = EmitVersion(&got, version); err != nil || !bytes.Equal(want, got.Bytes()) {
			t.Fatalf("v%d changed: %v", version, err)
		}
	}
}

func TestEmitV3BindsProjectSourceAndPackageGraphs(t *testing.T) {
	var out bytes.Buffer
	if err := EmitVersion(&out, 3); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"rv " + RevisionV3ID, "pc 1\n" + RevisionV2ID, "en " + ModuleID + " 00000000000000000000000000000012 3 3", "fi 00000000000000000000000000000122 li 38", "0000000000000000000000000000b002", "00000000000000000000000000009023", "PackageGraphBinding", "ProjectGraphSnapshot", "package_graph_binding.package_graph", "package_graph_binding.source_inventory", "project_graph_snapshot.snapshot", "project_graph_snapshot.package_graph_binding"} {
		value := want
		if strings.Contains(want, ".") || want == "PackageGraphBinding" || want == "ProjectGraphSnapshot" {
			value = fmt.Sprintf("%x", want)
		}
		if !strings.Contains(out.String(), value) {
			t.Fatalf("missing %s", want)
		}
	}
	if strings.Contains(out.String(), "0000000000000000000000000000b001") {
		t.Fatal("v3 retained the Package v1 pin")
	}
	for version, path := range map[int]string{1: "../../../modules/project/v1/module.g1", 2: "../../../modules/project/v2/module.g1"} {
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var generated bytes.Buffer
		if err := EmitVersion(&generated, version); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(generated.Bytes(), want) {
			t.Fatalf("v%d bytes changed", version)
		}
	}
}

func TestV2ClosedEnumsRejectUnknownValues(t *testing.T) {
	for value := uint64(0); value <= 4; value++ {
		if err := ValidateSourceClassification(value); err != nil {
			t.Fatalf("classification %d: %v", value, err)
		}
	}
	if ValidateSourceClassification(5) == nil {
		t.Fatal("accepted unknown classification")
	}
	for value := uint64(0); value <= 3; value++ {
		if err := ValidatePreservationMode(value); err != nil {
			t.Fatalf("preservation %d: %v", value, err)
		}
	}
	if ValidatePreservationMode(4) == nil {
		t.Fatal("accepted unknown preservation mode")
	}
}
