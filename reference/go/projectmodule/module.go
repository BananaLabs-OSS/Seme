// Package projectmodule emits versioned language-neutral Project Contracts.
package projectmodule

import (
	"encoding/hex"
	"fmt"
	"io"
)

const ModuleID = "0000000000000000000000000000e000"
const RevisionID = "0000000000000000000000000000e001"
const RevisionV2ID = "0000000000000000000000000000e002"

type SourceClassification uint64

const (
	ClassificationTracked SourceClassification = iota
	ClassificationIgnored
	ClassificationGenerated
	ClassificationVendored
	ClassificationOpaque
)

func ValidateSourceClassification(value uint64) error {
	if value > uint64(ClassificationOpaque) {
		return fmt.Errorf("project source classification %d outside closed range 0..4", value)
	}
	return nil
}

type PreservationMode uint64

const (
	PreservationByteExact PreservationMode = iota
	PreservationSemanticProjection
	PreservationRegenerable
	PreservationReferenceOnly
)

func ValidatePreservationMode(value uint64) error {
	if value > uint64(PreservationReferenceOnly) {
		return fmt.Errorf("project preservation mode %d outside closed range 0..3", value)
	}
	return nil
}

func Emit(out io.Writer) error {
	return EmitVersion(out, 1)
}

func EmitVersion(out io.Writer, version int) error {
	if version != 1 && version != 2 {
		return fmt.Errorf("unsupported Project Contract version %d", version)
	}
	b := func(value string) string { return hex.EncodeToString([]byte(value)) }
	exports := "rf 0000000000000000000000000000e010\nrf 0000000000000000000000000000e011\nrf 0000000000000000000000000000e100\nrf 0000000000000000000000000000e110\nrf 0000000000000000000000000000e111\nrf 0000000000000000000000000000e112\nrf 0000000000000000000000000000e113\nrf 0000000000000000000000000000e114\n"
	revision, parent, count, moduleVersion, exportCount, extraSchemas, extraFields := RevisionID, "pc 0", 11, 1, 8, "", ""
	if version == 2 {
		revision, parent, count, moduleVersion, exportCount = RevisionV2ID, "pc 1\n"+RevisionID, 33, 2, 30
		exports = v2Exports()
		extraSchemas, extraFields = v2Entities(b)
	}
	_, err := fmt.Fprintf(out, `# Generated construction projection for Project Contract v%d.
ve 1
mo %s
rv %s
%s
ec %d

en %s 00000000000000000000000000000012 %d 3
fi 00000000000000000000000000000120 by %s
fi 00000000000000000000000000000121 li 2
rf 0000000000000000000000000000e002
rf 0000000000000000000000000000e003
fi 00000000000000000000000000000122 li %d
%s

en 0000000000000000000000000000e002 00000000000000000000000000000013 1 2
fi 00000000000000000000000000000130 rf 0000000000000000000000000000b000
fi 00000000000000000000000000000131 by 0000000000000000000000000000b001

en 0000000000000000000000000000e003 00000000000000000000000000000013 1 2
fi 00000000000000000000000000000130 rf 00000000000000000000000000009000
fi 00000000000000000000000000000131 by 00000000000000000000000000009023

en 0000000000000000000000000000e010 00000000000000000000000000000010 1 2
fi 00000000000000000000000000000100 by %s
fi 00000000000000000000000000000101 li 1
rf 0000000000000000000000000000e100

en 0000000000000000000000000000e011 00000000000000000000000000000010 1 2
fi 00000000000000000000000000000100 by %s
fi 00000000000000000000000000000101 li 5
rf 0000000000000000000000000000e110
rf 0000000000000000000000000000e111
rf 0000000000000000000000000000e112
rf 0000000000000000000000000000e113
rf 0000000000000000000000000000e114

%s

%s%s`, version, ModuleID, revision, parent, count, ModuleID, moduleVersion, b(fmt.Sprintf("project-contract-v%d", version)), exportCount, exports, b("ProjectIdentity"), b("ProjectSnapshot"), extraSchemas, fields(b), extraFields)
	return err
}

func v2Entities(b func(string) string) (string, string) {
	type field struct {
		id, name, schema string
		kind, card       int
	}
	schemas := []struct {
		id, name string
		fields   []field
	}{
		{"e012", "ToolchainProfile", []field{{"e120", "toolchain_profile.language", "", 4, 0}, {"e121", "toolchain_profile.toolchain", "", 4, 0}, {"e122", "toolchain_profile.profile", "", 4, 0}, {"e123", "toolchain_profile.semantic_revision", "", 4, 0}}},
		{"e013", "SourceClassification", []field{{"e130", "source_classification.code", "", 2, 0}}},
		{"e014", "PreservationMode", []field{{"e140", "preservation_mode.code", "", 2, 0}}},
		{"e015", "SourceUnit", []field{{"e150", "source_unit.normalized_relative_path", "", 4, 0}, {"e151", "source_unit.content_digest", "", 4, 0}, {"e152", "source_unit.byte_size", "", 2, 0}, {"e153", "source_unit.classification", "e013", 5, 0}, {"e154", "source_unit.preservation_mode", "e014", 5, 0}, {"e155", "source_unit.toolchain", "e012", 5, 0}}},
		{"e016", "SourceInventory", []field{{"e160", "source_inventory.inventory_revision", "", 4, 0}, {"e161", "source_inventory.snapshot", "e011", 5, 0}, {"e162", "source_inventory.semantic_revision", "", 4, 0}, {"e163", "source_inventory.toolchains", "e012", 5, 2}, {"e164", "source_inventory.units", "e015", 5, 2}}},
	}
	var schemaOut, fieldOut string
	for _, s := range schemas {
		schemaOut += fmt.Sprintf("\nen 0000000000000000000000000000%s 00000000000000000000000000000010 1 2\nfi 00000000000000000000000000000100 by %s\nfi 00000000000000000000000000000101 li %d\n", s.id, b(s.name), len(s.fields))
		for _, f := range s.fields {
			schemaOut += fmt.Sprintf("rf 0000000000000000000000000000%s\n", f.id)
		}
	}
	for _, s := range schemas {
		for _, f := range s.fields {
			rc, schema := 1, ""
			if f.schema != "" {
				rc = 2
				schema = fmt.Sprintf("fi 00000000000000000000000000002001 rf 0000000000000000000000000000%s\n", f.schema)
			}
			fieldOut += fmt.Sprintf("\nen 0000000000000000000000000000%s 00000000000000000000000000000011 1 4\nfi 00000000000000000000000000000110 by %s\nfi 00000000000000000000000000000111 rc %d\nfi 00000000000000000000000000002000 uu %d\n%sfi 00000000000000000000000000000112 uu %d\nfi 00000000000000000000000000000113 uu 1\n", f.id, b(f.name), rc, f.kind, schema, f.card)
		}
	}
	return schemaOut, fieldOut
}

func v2Exports() string {
	ids := []string{"e010", "e011", "e012", "e013", "e014", "e015", "e016", "e100", "e110", "e111", "e112", "e113", "e114", "e120", "e121", "e122", "e123", "e130", "e140", "e150", "e151", "e152", "e153", "e154", "e155", "e160", "e161", "e162", "e163", "e164"}
	var out string
	for _, x := range ids {
		out += fmt.Sprintf("rf 0000000000000000000000000000%s\n", x)
	}
	return out
}

func fields(b func(string) string) string {
	definitions := []struct {
		id, name, schema string
		card             int
	}{
		{"e100", "project_identity.name", "", 0},
		{"e110", "project_snapshot.identity", "e010", 0},
		{"e111", "project_snapshot.content_revision", "", 0},
		{"e112", "project_snapshot.packages", "b010", 2},
		{"e113", "project_snapshot.root", "b010", 0},
	}
	var result string
	for _, f := range definitions {
		kind := 5
		if f.schema == "" {
			kind = 4
		}
		count, schema := 1, ""
		if f.schema != "" {
			count, schema = 2, fmt.Sprintf("fi 00000000000000000000000000002001 rf 0000000000000000000000000000%s\n", f.schema)
		}
		result += fmt.Sprintf("en 0000000000000000000000000000%s 00000000000000000000000000000011 1 4\nfi 00000000000000000000000000000110 by %s\nfi 00000000000000000000000000000111 rc %d\nfi 00000000000000000000000000002000 uu %d\n%sfi 00000000000000000000000000000112 uu %d\nfi 00000000000000000000000000000113 uu 1\n\n", f.id, b(f.name), count, kind, schema, f.card)
	}
	// Program is an imported reference, unlike content_revision bytes.
	result += fmt.Sprintf("en 0000000000000000000000000000e114 00000000000000000000000000000011 1 4\nfi 00000000000000000000000000000110 by %s\nfi 00000000000000000000000000000111 rc 2\nfi 00000000000000000000000000002000 uu 5\nfi 00000000000000000000000000002001 rf 00000000000000000000000000009015\nfi 00000000000000000000000000000112 uu 0\nfi 00000000000000000000000000000113 uu 1\n", b("project_snapshot.program"))
	return result
}
