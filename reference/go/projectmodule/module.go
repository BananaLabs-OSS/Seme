// Package projectmodule emits versioned language-neutral Project Contracts.
package projectmodule

import (
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

const ModuleID = "0000000000000000000000000000e000"
const RevisionID = "0000000000000000000000000000e001"
const RevisionV2ID = "0000000000000000000000000000e002"
const RevisionV3ID = "0000000000000000000000000000e003"
const RevisionV4ID = "0000000000000000000000000000e004"
const RevisionV5ID = "0000000000000000000000000000e005"
const RevisionV6ID = "0000000000000000000000000000e008"
const RevisionV7ID = "0000000000000000000000000000e009"
const RevisionV8ID = "0000000000000000000000000000e00a"
const RevisionV9ID = "0000000000000000000000000000e00b"
const RevisionV10ID = "0000000000000000000000000000e00e"
const RevisionV11ID = "0000000000000000000000000000e030"

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
	if version < 1 || version > 11 {
		return fmt.Errorf("unsupported Project Contract version %d", version)
	}
	b := func(value string) string { return hex.EncodeToString([]byte(value)) }
	exports := "rf 0000000000000000000000000000e010\nrf 0000000000000000000000000000e011\nrf 0000000000000000000000000000e100\nrf 0000000000000000000000000000e110\nrf 0000000000000000000000000000e111\nrf 0000000000000000000000000000e112\nrf 0000000000000000000000000000e113\nrf 0000000000000000000000000000e114\n"
	revision, parent, count, moduleVersion, exportCount, extraSchemas, extraFields, packageRevision := RevisionID, "pc 0", 11, 1, 8, "", "", "0000000000000000000000000000b001"
	executionRevision := "00000000000000000000000000009023"
	importCount, importRefs, dependencyImport := 2, "rf 0000000000000000000000000000e002\nrf 0000000000000000000000000000e003", ""
	lateImport := ""
	if version == 2 {
		revision, parent, count, moduleVersion, exportCount = RevisionV2ID, "pc 1\n"+RevisionID, 33, 2, 30
		exports = v2Exports()
		extraSchemas, extraFields = v2Entities(b)
	}
	if version == 3 {
		revision, parent, count, moduleVersion, exportCount, packageRevision = RevisionV3ID, "pc 1\n"+RevisionV2ID, 41, 3, 38, "0000000000000000000000000000b002"
		exports = v3Exports()
		extraSchemas, extraFields = v2Entities(b)
		v3Schemas, v3Fields := v3Entities(b)
		extraSchemas += v3Schemas
		extraFields += v3Fields
	}
	if version == 4 {
		revision, parent, count, moduleVersion, exportCount, packageRevision = RevisionV4ID, "pc 1\n"+RevisionV3ID, 46, 4, 42, "0000000000000000000000000000b002"
		exports = v4Exports()
		extraSchemas, extraFields = v2Entities(b)
		s3, f3 := v3Entities(b)
		s4, f4 := v4Entities(b)
		extraSchemas += s3 + s4
		extraFields += f3 + f4
		importCount = 3
		importRefs += "\nrf 0000000000000000000000000000e004"
		dependencyImport = "\n\nen 0000000000000000000000000000e004 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 0000000000000000000000000000f000\nfi 00000000000000000000000000000131 by 0000000000000000000000000000f001"
	}
	if version == 5 {
		revision, parent, count, moduleVersion, exportCount, packageRevision = RevisionV5ID, "pc 1\n"+RevisionV4ID, 50, 5, 46, "0000000000000000000000000000b003"
		exports = v5Exports()
		extraSchemas, extraFields = v2Entities(b)
		s3, f3 := v3Entities(b)
		s4, f4 := v4Entities(b)
		s5, f5 := v5Entities(b)
		extraSchemas += s3 + s4 + s5
		extraFields += f3 + f4 + f5
		importCount = 3
		importRefs += "\nrf 0000000000000000000000000000e006"
		dependencyImport = "\n\nen 0000000000000000000000000000e006 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 0000000000000000000000000000f000\nfi 00000000000000000000000000000131 by 0000000000000000000000000000f001"
	}
	if version == 6 {
		revision, parent, count, moduleVersion, exportCount, packageRevision = RevisionV6ID, "pc 1\n"+RevisionV5ID, 55, 6, 50, "0000000000000000000000000000b003"
		exports = v6Exports()
		extraSchemas, extraFields = v2Entities(b)
		s3, f3 := v3Entities(b)
		s4, f4 := v4Entities(b)
		s5, f5 := v5Entities(b)
		s6, f6 := v6Entities(b)
		extraSchemas += s3 + s4 + s5 + s6
		extraFields += f3 + f4 + f5 + f6
		importCount = 4
		importRefs += "\nrf 0000000000000000000000000000e006\nrf 0000000000000000000000000000e007"
		dependencyImport = "\n\nen 0000000000000000000000000000e006 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 0000000000000000000000000000f000\nfi 00000000000000000000000000000131 by 0000000000000000000000000000f001\n\nen 0000000000000000000000000000e007 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000004000\nfi 00000000000000000000000000000131 by 00000000000000000000000000004001"
	}
	if version == 7 {
		revision, parent, count, moduleVersion, exportCount, packageRevision = RevisionV7ID, "pc 1\n"+RevisionV6ID, 59, 7, 54, "0000000000000000000000000000b003"
		exports = v7Exports()
		extraSchemas, extraFields = v2Entities(b)
		s3, f3 := v3Entities(b)
		s4, f4 := v4Entities(b)
		s5, f5 := v5Entities(b)
		s6, f6 := v6Entities(b)
		s7, f7 := v7Entities(b)
		extraSchemas += s3 + s4 + s5 + s6 + s7
		extraFields += f3 + f4 + f5 + f6 + f7
		importCount = 4
		importRefs += "\nrf 0000000000000000000000000000e006\nrf 0000000000000000000000000000e007"
		dependencyImport = "\n\nen 0000000000000000000000000000e006 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 0000000000000000000000000000f000\nfi 00000000000000000000000000000131 by 0000000000000000000000000000f001\n\nen 0000000000000000000000000000e007 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000004000\nfi 00000000000000000000000000000131 by 00000000000000000000000000004005"
	}
	if version == 8 {
		revision, parent, count, moduleVersion, exportCount = RevisionV8ID, "pc 1\n"+RevisionV7ID, 67, 8, 61
		packageRevision, executionRevision = "0000000000000000000000000000b004", "00000000000000000000000000009024"
		exports = v8Exports()
		extraSchemas, extraFields = v2Entities(b)
		s3, f3 := v3Entities(b)
		s4, f4 := v4Entities(b)
		s5, f5 := v5Entities(b)
		s6, f6 := v6Entities(b)
		s7, f7 := v7Entities(b)
		s8, f8 := v8Entities(b)
		extraSchemas += s3 + s4 + s5 + s6 + s7 + s8
		extraFields += f3 + f4 + f5 + f6 + f7 + f8
		importCount = 5
		importRefs += "\nrf 0000000000000000000000000000e006\nrf 0000000000000000000000000000e007\nrf 0000000000000000000000000000e00b"
		dependencyImport = "\n\nen 0000000000000000000000000000e006 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 0000000000000000000000000000f000\nfi 00000000000000000000000000000131 by 0000000000000000000000000000f001\n\nen 0000000000000000000000000000e007 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000004000\nfi 00000000000000000000000000000131 by 00000000000000000000000000004006\n\nen 0000000000000000000000000000e00b 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000003000\nfi 00000000000000000000000000000131 by 00000000000000000000000000003001"
	}
	if version == 9 {
		revision, parent, count, moduleVersion, exportCount = RevisionV9ID, "pc 1\n"+RevisionV8ID, 72, 9, 65
		packageRevision, executionRevision = "0000000000000000000000000000b004", "00000000000000000000000000009024"
		exports = v9Exports()
		extraSchemas, extraFields = v2Entities(b)
		for _, emit := range []func(func(string) string) (string, string){v3Entities, v4Entities, v5Entities, v6Entities, v7Entities, v8Entities, v9Entities} {
			schemas, fields := emit(b)
			extraSchemas += schemas
			extraFields += fields
		}
		importCount = 6
		importRefs += "\nrf 0000000000000000000000000000e006\nrf 0000000000000000000000000000e007\nrf 0000000000000000000000000000e00c\nrf 0000000000000000000000000000e00d"
		dependencyImport = "\n\nen 0000000000000000000000000000e006 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 0000000000000000000000000000f000\nfi 00000000000000000000000000000131 by 0000000000000000000000000000f001\n\nen 0000000000000000000000000000e007 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000004000\nfi 00000000000000000000000000000131 by 00000000000000000000000000004006\n\nen 0000000000000000000000000000e00c 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000003000\nfi 00000000000000000000000000000131 by 00000000000000000000000000003001\n\nen 0000000000000000000000000000e00d 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000006000\nfi 00000000000000000000000000000131 by 00000000000000000000000000006001"
	}
	if version == 10 {
		revision, parent, count, moduleVersion, exportCount = RevisionV10ID, "pc 1\n"+RevisionV9ID, 79, 10, 70
		packageRevision, executionRevision = "0000000000000000000000000000b004", "00000000000000000000000000009024"
		exports = v10Exports()
		extraSchemas, extraFields = v2Entities(b)
		for _, emit := range []func(func(string) string) (string, string){v3Entities, v4Entities, v5Entities, v6Entities, v7Entities, v8Entities, v9Entities, v10Entities} {
			schemas, fields := emit(b)
			extraSchemas += schemas
			extraFields += fields
		}
		importCount = 8
		importRefs += "\nrf 0000000000000000000000000000e006\nrf 0000000000000000000000000000e007\nrf 0000000000000000000000000000e00c\nrf 0000000000000000000000000000e00d\nrf 0000000000000000000000000000e00f\nrf 0000000000000000000000000000e027"
		dependencyImport = "\n\nen 0000000000000000000000000000e006 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 0000000000000000000000000000f000\nfi 00000000000000000000000000000131 by 0000000000000000000000000000f001\n\nen 0000000000000000000000000000e007 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000004000\nfi 00000000000000000000000000000131 by 00000000000000000000000000004006\n\nen 0000000000000000000000000000e00c 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000003000\nfi 00000000000000000000000000000131 by 00000000000000000000000000003001\n\nen 0000000000000000000000000000e00d 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000006000\nfi 00000000000000000000000000000131 by 00000000000000000000000000006001\n\nen 0000000000000000000000000000e00f 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000008000\nfi 00000000000000000000000000000131 by 00000000000000000000000000008001\n\nen 0000000000000000000000000000e027 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000001000\nfi 00000000000000000000000000000131 by 00000000000000000000000000001001"
		const marker = "\n\nen 0000000000000000000000000000e027 "
		at := strings.LastIndex(dependencyImport, marker)
		lateImport, dependencyImport = dependencyImport[at:], dependencyImport[:at]
		lateImport += "\n\n"
	}
	if version == 11 {
		revision, parent, count, moduleVersion, exportCount = RevisionV11ID, "pc 1\n"+RevisionV10ID, 84, 11, 74
		packageRevision, executionRevision = "0000000000000000000000000000b004", "00000000000000000000000000009024"
		exports = v11Exports()
		extraSchemas, extraFields = v2Entities(b)
		for _, emit := range []func(func(string) string) (string, string){v3Entities, v4Entities, v5Entities, v6Entities, v7Entities, v8Entities, v9Entities, v10Entities, v11Entities} {
			schemas, fields := emit(b)
			extraSchemas += schemas
			extraFields += fields
		}
		importCount = 9
		importRefs += "\nrf 0000000000000000000000000000e006\nrf 0000000000000000000000000000e007\nrf 0000000000000000000000000000e00c\nrf 0000000000000000000000000000e00d\nrf 0000000000000000000000000000e00f\nrf 0000000000000000000000000000e027\nrf 0000000000000000000000000000e028"
		dependencyImport = "\n\nen 0000000000000000000000000000e006 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 0000000000000000000000000000f000\nfi 00000000000000000000000000000131 by 0000000000000000000000000000f001\n\nen 0000000000000000000000000000e007 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000004000\nfi 00000000000000000000000000000131 by 00000000000000000000000000004006\n\nen 0000000000000000000000000000e00c 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000003000\nfi 00000000000000000000000000000131 by 00000000000000000000000000003001\n\nen 0000000000000000000000000000e00d 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000006000\nfi 00000000000000000000000000000131 by 00000000000000000000000000006001\n\nen 0000000000000000000000000000e00f 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000008000\nfi 00000000000000000000000000000131 by 00000000000000000000000000008001"
		lateImport = "\n\nen 0000000000000000000000000000e027 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000001000\nfi 00000000000000000000000000000131 by 00000000000000000000000000001001\n\nen 0000000000000000000000000000e028 00000000000000000000000000000013 1 2\nfi 00000000000000000000000000000130 rf 00000000000000000000000000010000\nfi 00000000000000000000000000000131 by 00000000000000000000000000010001\n\n"
	}
	if version >= 2 {
		// Preserve the established generated layout between export, schema, and
		// field sections. These separators are part of the checked G1 bytes.
		exports += "\n"
		extraSchemas = "\n" + extraSchemas + "\n"
	}
	_, err := fmt.Fprintf(out, `# Generated construction projection for Project Contract v%d.
ve 1
mo %s
rv %s
%s
ec %d

en %s 00000000000000000000000000000012 %d 3
fi 00000000000000000000000000000120 by %s
fi 00000000000000000000000000000121 li %d
%s
fi 00000000000000000000000000000122 li %d
%s
en 0000000000000000000000000000e002 00000000000000000000000000000013 1 2
fi 00000000000000000000000000000130 rf 0000000000000000000000000000b000
fi 00000000000000000000000000000131 by %s

en 0000000000000000000000000000e003 00000000000000000000000000000013 1 2
fi 00000000000000000000000000000130 rf 00000000000000000000000000009000
fi 00000000000000000000000000000131 by %s%s

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
%s%s%s`, version, ModuleID, revision, parent, count, ModuleID, moduleVersion, b(fmt.Sprintf("project-contract-v%d", version)), importCount, importRefs, exportCount, exports, packageRevision, executionRevision, dependencyImport, b("ProjectIdentity"), b("ProjectSnapshot"), extraSchemas, lateImport, fields(b), extraFields)
	return err
}

func v6Entities(b func(string) string) (string, string) {
	schema := fmt.Sprintf("\nen 0000000000000000000000000000e021 00000000000000000000000000000010 1 2\nfi 00000000000000000000000000000100 by %s\nfi 00000000000000000000000000000101 li 3\nrf 0000000000000000000000000000e210\nrf 0000000000000000000000000000e211\nrf 0000000000000000000000000000e212\n", b("ConfiguredProjectSnapshot"))
	field := func(id, name, target string) string {
		kind, count, constraint := 4, 1, ""
		if target != "" {
			kind, count = 5, 2
			constraint = fmt.Sprintf("fi 00000000000000000000000000002001 rf 0000000000000000000000000000%s\n", target)
		}
		return fmt.Sprintf("\nen 0000000000000000000000000000%s 00000000000000000000000000000011 1 4\nfi 00000000000000000000000000000110 by %s\nfi 00000000000000000000000000000111 rc %d\nfi 00000000000000000000000000002000 uu %d\n%sfi 00000000000000000000000000000112 uu 0\nfi 00000000000000000000000000000113 uu 1\n", id, b(name), count, kind, constraint)
	}
	return schema, field("e210", "configured_project_snapshot.complete_project_graph", "e020") + field("e211", "configured_project_snapshot.configuration_graph", "4010") + field("e212", "configured_project_snapshot.content_revision", "")
}

func v7Entities(b func(string) string) (string, string) {
	schema := fmt.Sprintf("\nen 0000000000000000000000000000e022 00000000000000000000000000000010 1 2\nfi 00000000000000000000000000000100 by %s\nfi 00000000000000000000000000000101 li 3\nrf 0000000000000000000000000000e220\nrf 0000000000000000000000000000e221\nrf 0000000000000000000000000000e222\n", b("BoundConfiguredProjectSnapshot"))
	field := func(id, name, target string) string {
		kind, count, constraint := 4, 1, ""
		if target != "" {
			kind, count = 5, 2
			constraint = fmt.Sprintf("fi 00000000000000000000000000002001 rf 0000000000000000000000000000%s\n", target)
		}
		return fmt.Sprintf("\nen 0000000000000000000000000000%s 00000000000000000000000000000011 1 4\nfi 00000000000000000000000000000110 by %s\nfi 00000000000000000000000000000111 rc %d\nfi 00000000000000000000000000002000 uu %d\n%sfi 00000000000000000000000000000112 uu 0\nfi 00000000000000000000000000000113 uu 1\n", id, b(name), count, kind, constraint)
	}
	return schema, field("e220", "bound_configured_project_snapshot.configured_project_snapshot", "e021") + field("e221", "bound_configured_project_snapshot.bound_configuration_graph", "401f") + field("e222", "bound_configured_project_snapshot.content_revision", "")
}

func v8Entities(b func(string) string) (string, string) {
	schema := fmt.Sprintf("\nen 0000000000000000000000000000e023 00000000000000000000000000000010 1 2\nfi 00000000000000000000000000000100 by %s\nfi 00000000000000000000000000000101 li 6\nrf 0000000000000000000000000000e230\nrf 0000000000000000000000000000e231\nrf 0000000000000000000000000000e232\nrf 0000000000000000000000000000e233\nrf 0000000000000000000000000000e234\nrf 0000000000000000000000000000e235\n", b("FullConfiguredProjectSnapshot"))
	field := func(id, name, target string) string {
		kind, count, constraint := 4, 1, ""
		if target != "" {
			kind, count = 5, 2
			constraint = fmt.Sprintf("fi 00000000000000000000000000002001 rf 0000000000000000000000000000%s\n", target)
		}
		return fmt.Sprintf("\nen 0000000000000000000000000000%s 00000000000000000000000000000011 1 4\nfi 00000000000000000000000000000110 by %s\nfi 00000000000000000000000000000111 rc %d\nfi 00000000000000000000000000002000 uu %d\n%sfi 00000000000000000000000000000112 uu 0\nfi 00000000000000000000000000000113 uu 1\n", id, b(name), count, kind, constraint)
	}
	fields := field("e230", "full_configured_project_snapshot.source_inventory", "e016")
	fields += field("e231", "full_configured_project_snapshot.dependency_closure", "f010")
	fields += field("e232", "full_configured_project_snapshot.package_graph", "b029")
	fields += field("e233", "full_configured_project_snapshot.program", "9015")
	fields += field("e234", "full_configured_project_snapshot.bound_configuration_graph", "401f")
	fields += field("e235", "full_configured_project_snapshot.content_revision", "")
	return schema, fields
}

func v9Entities(b func(string) string) (string, string) {
	schema := fmt.Sprintf("\nen 0000000000000000000000000000e024 00000000000000000000000000000010 1 2\nfi 00000000000000000000000000000100 by %s\nfi 00000000000000000000000000000101 li 3\nrf 0000000000000000000000000000e240\nrf 0000000000000000000000000000e241\nrf 0000000000000000000000000000e242\n", b("ResourceProjectSnapshot"))
	field := func(id, name, target string) string {
		kind, count, constraint := 4, 1, ""
		if target != "" {
			kind, count = 5, 2
			constraint = fmt.Sprintf("fi 00000000000000000000000000002001 rf 0000000000000000000000000000%s\n", target)
		}
		return fmt.Sprintf("\nen 0000000000000000000000000000%s 00000000000000000000000000000011 1 4\nfi 00000000000000000000000000000110 by %s\nfi 00000000000000000000000000000111 rc %d\nfi 00000000000000000000000000002000 uu %d\n%sfi 00000000000000000000000000000112 uu 0\nfi 00000000000000000000000000000113 uu 1\n", id, b(name), count, kind, constraint)
	}
	return schema, field("e240", "resource_project_snapshot.full_configured_project_snapshot", "e023") + field("e241", "resource_project_snapshot.resource_manifest", "6010") + field("e242", "resource_project_snapshot.content_revision", "")
}

func v10Entities(b func(string) string) (string, string) {
	schema := fmt.Sprintf("\nen 0000000000000000000000000000e025 00000000000000000000000000000010 1 2\nfi 00000000000000000000000000000100 by %s\nfi 00000000000000000000000000000101 li 4\nrf 0000000000000000000000000000e250\nrf 0000000000000000000000000000e251\nrf 0000000000000000000000000000e252\nrf 0000000000000000000000000000e253\n", b("DurableProjectSnapshot"))
	field := func(id, name, target string) string {
		kind, count, constraint := 4, 1, ""
		if target != "" {
			kind, count = 5, 2
			constraint = fmt.Sprintf("fi 00000000000000000000000000002001 rf 0000000000000000000000000000%s\n", target)
		}
		return fmt.Sprintf("\nen 0000000000000000000000000000%s 00000000000000000000000000000011 1 4\nfi 00000000000000000000000000000110 by %s\nfi 00000000000000000000000000000111 rc %d\nfi 00000000000000000000000000002000 uu %d\n%sfi 00000000000000000000000000000112 uu 0\nfi 00000000000000000000000000000113 uu 1\n", id, b(name), count, kind, constraint)
	}
	return schema, field("e250", "durable_project_snapshot.resource_project_snapshot", "e024") + field("e251", "durable_project_snapshot.durable_state_plan", "8010") + field("e252", "durable_project_snapshot.content_revision", "") + field("e253", "durable_project_snapshot.presentation_manifest", "1010")
}

func v11Entities(b func(string) string) (string, string) {
	schema := fmt.Sprintf("\nen 0000000000000000000000000000e026 00000000000000000000000000000010 1 2\nfi 00000000000000000000000000000100 by %s\nfi 00000000000000000000000000000101 li 3\nrf 0000000000000000000000000000e260\nrf 0000000000000000000000000000e261\nrf 0000000000000000000000000000e262\n", b("TransportProjectSnapshot"))
	field := func(id, name, target string) string {
		kind, count, constraint := 4, 1, ""
		if target != "" {
			kind, count = 5, 2
			constraint = fmt.Sprintf("fi 00000000000000000000000000002001 rf %s%s\n", strings.Repeat("0", 32-len(target)), target)
		}
		return fmt.Sprintf("\nen 0000000000000000000000000000%s 00000000000000000000000000000011 1 4\nfi 00000000000000000000000000000110 by %s\nfi 00000000000000000000000000000111 rc %d\nfi 00000000000000000000000000002000 uu %d\n%sfi 00000000000000000000000000000112 uu 0\nfi 00000000000000000000000000000113 uu 1\n", id, b(name), count, kind, constraint)
	}
	return schema, field("e260", "transport_project_snapshot.durable_project_snapshot", "e025") + field("e261", "transport_project_snapshot.transport_manifest", "10100") + field("e262", "transport_project_snapshot.content_revision", "")
}

func v5Entities(b func(string) string) (string, string) {
	schema := fmt.Sprintf("\nen 0000000000000000000000000000e020 00000000000000000000000000000010 1 2\nfi 00000000000000000000000000000100 by %s\nfi 00000000000000000000000000000101 li 3\nrf 0000000000000000000000000000e200\nrf 0000000000000000000000000000e201\nrf 0000000000000000000000000000e202\n", b("CompleteProjectGraphSnapshot"))
	field := func(id, name, target string) string {
		kind, count, constraint := 4, 1, ""
		if target != "" {
			kind, count = 5, 2
			constraint = fmt.Sprintf("fi 00000000000000000000000000002001 rf 0000000000000000000000000000%s\n", target)
		}
		return fmt.Sprintf("\nen 0000000000000000000000000000%s 00000000000000000000000000000011 1 4\nfi 00000000000000000000000000000110 by %s\nfi 00000000000000000000000000000111 rc %d\nfi 00000000000000000000000000002000 uu %d\n%sfi 00000000000000000000000000000112 uu 0\nfi 00000000000000000000000000000113 uu 1\n", id, b(name), count, kind, constraint)
	}
	return schema, field("e200", "complete_project_graph_snapshot.dependency_graph_snapshot", "e019") + field("e201", "complete_project_graph_snapshot.complete_package_graph", "b029") + field("e202", "complete_project_graph_snapshot.content_revision", "")
}

func v4Entities(b func(string) string) (string, string) {
	schema := fmt.Sprintf("\nen 0000000000000000000000000000e019 00000000000000000000000000000010 1 2\nfi 00000000000000000000000000000100 by %s\nfi 00000000000000000000000000000101 li 3\nrf 0000000000000000000000000000e190\nrf 0000000000000000000000000000e191\nrf 0000000000000000000000000000e192\n", b("DependencyGraphSnapshot"))
	field := func(id, name, target string) string {
		constraint := ""
		kind := 4
		count := 1
		if target != "" {
			kind = 5
			count = 2
			constraint = fmt.Sprintf("fi 00000000000000000000000000002001 rf 0000000000000000000000000000%s\n", target)
		}
		return fmt.Sprintf("\nen 0000000000000000000000000000%s 00000000000000000000000000000011 1 4\nfi 00000000000000000000000000000110 by %s\nfi 00000000000000000000000000000111 rc %d\nfi 00000000000000000000000000002000 uu %d\n%sfi 00000000000000000000000000000112 uu 0\nfi 00000000000000000000000000000113 uu 1\n", id, b(name), count, kind, constraint)
	}
	return schema, field("e190", "dependency_graph_snapshot.project_graph_snapshot", "e018") + field("e191", "dependency_graph_snapshot.dependency_closure", "f010") + field("e192", "dependency_graph_snapshot.content_revision", "")
}

func v3Entities(b func(string) string) (string, string) {
	type field struct {
		id, name, schema string
		kind, card       int
	}
	schemas := []struct {
		id, name string
		fields   []field
	}{
		{"e017", "PackageGraphBinding", []field{{"e170", "package_graph_binding.package_graph", "b020", 5, 0}, {"e171", "package_graph_binding.source_inventory", "e016", 5, 0}, {"e172", "package_graph_binding.content_revision", "", 4, 0}}},
		{"e018", "ProjectGraphSnapshot", []field{{"e180", "project_graph_snapshot.snapshot", "e011", 5, 0}, {"e181", "project_graph_snapshot.package_graph_binding", "e017", 5, 0}, {"e182", "project_graph_snapshot.content_revision", "", 4, 0}}},
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
			rc, constraint := 1, ""
			if f.schema != "" {
				rc = 2
				constraint = fmt.Sprintf("fi 00000000000000000000000000002001 rf 0000000000000000000000000000%s\n", f.schema)
			}
			fieldOut += fmt.Sprintf("\nen 0000000000000000000000000000%s 00000000000000000000000000000011 1 4\nfi 00000000000000000000000000000110 by %s\nfi 00000000000000000000000000000111 rc %d\nfi 00000000000000000000000000002000 uu %d\n%sfi 00000000000000000000000000000112 uu %d\nfi 00000000000000000000000000000113 uu 1\n", f.id, b(f.name), rc, f.kind, constraint, f.card)
		}
	}
	return schemaOut, fieldOut
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

func v3Exports() string {
	ids := []string{"e010", "e011", "e012", "e013", "e014", "e015", "e016", "e017", "e018", "e100", "e110", "e111", "e112", "e113", "e114", "e120", "e121", "e122", "e123", "e130", "e140", "e150", "e151", "e152", "e153", "e154", "e155", "e160", "e161", "e162", "e163", "e164", "e170", "e171", "e172", "e180", "e181", "e182"}
	var out string
	for _, x := range ids {
		out += fmt.Sprintf("rf 0000000000000000000000000000%s\n", x)
	}
	return out
}
func v4Exports() string {
	out := ""
	for _, x := range []string{"e010", "e011", "e012", "e013", "e014", "e015", "e016", "e017", "e018", "e019", "e100", "e110", "e111", "e112", "e113", "e114", "e120", "e121", "e122", "e123", "e130", "e140", "e150", "e151", "e152", "e153", "e154", "e155", "e160", "e161", "e162", "e163", "e164", "e170", "e171", "e172", "e180", "e181", "e182", "e190", "e191", "e192"} {
		out += fmt.Sprintf("rf 0000000000000000000000000000%s\n", x)
	}
	return out
}
func v5Exports() string {
	out := ""
	for _, x := range []string{"e010", "e011", "e012", "e013", "e014", "e015", "e016", "e017", "e018", "e019", "e020", "e100", "e110", "e111", "e112", "e113", "e114", "e120", "e121", "e122", "e123", "e130", "e140", "e150", "e151", "e152", "e153", "e154", "e155", "e160", "e161", "e162", "e163", "e164", "e170", "e171", "e172", "e180", "e181", "e182", "e190", "e191", "e192", "e200", "e201", "e202"} {
		out += fmt.Sprintf("rf 0000000000000000000000000000%s\n", x)
	}
	return out
}
func v6Exports() string {
	out := v5Exports()
	for _, x := range []string{"e021", "e210", "e211", "e212"} {
		out += fmt.Sprintf("rf 0000000000000000000000000000%s\n", x)
	}
	return out
}

func v7Exports() string {
	out := v6Exports()
	for _, x := range []string{"e022", "e220", "e221", "e222"} {
		out += fmt.Sprintf("rf 0000000000000000000000000000%s\n", x)
	}
	return out
}

func v8Exports() string {
	out := v7Exports()
	for _, x := range []string{"e023", "e230", "e231", "e232", "e233", "e234", "e235"} {
		out += fmt.Sprintf("rf 0000000000000000000000000000%s\n", x)
	}
	return out
}

func v9Exports() string {
	out := v8Exports()
	for _, x := range []string{"e024", "e240", "e241", "e242"} {
		out += fmt.Sprintf("rf 0000000000000000000000000000%s\n", x)
	}
	return out
}

func v10Exports() string {
	out := v9Exports()
	for _, x := range []string{"e025", "e250", "e251", "e252", "e253"} {
		out += fmt.Sprintf("rf 0000000000000000000000000000%s\n", x)
	}
	return out
}

func v11Exports() string {
	out := v10Exports()
	for _, x := range []string{"e026", "e260", "e261", "e262"} {
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
