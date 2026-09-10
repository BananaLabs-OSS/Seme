// Package projectinstance validates the outer, composed trust boundary of a
// canonical Project instance. Execution and Package instance semantics are
// intentionally the responsibility of their respective validators.
package projectinstance

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"

	"seme.local/reference/projectsnapshot"
	"seme.local/reference/wire"
)

var (
	moduleSchema    = mustID("00000000000000000000000000000012")
	importSchema    = mustID("00000000000000000000000000000013")
	fModuleName     = mustID("00000000000000000000000000000120")
	fImports        = mustID("00000000000000000000000000000121")
	fExports        = mustID("00000000000000000000000000000122")
	fImportModule   = mustID("00000000000000000000000000000130")
	fImportRevision = mustID("00000000000000000000000000000131")
	snapshotSchema  = mustID("0000000000000000000000000000e011")
	fIdentity       = mustID("0000000000000000000000000000e110")
	fPackages       = mustID("0000000000000000000000000000e112")
	fProgram        = mustID("0000000000000000000000000000e114")
)

type pin struct{ module, revision wire.ID }

var requiredPins = []pin{
	{mustID("00000000000000000000000000009000"), mustID("00000000000000000000000000009023")},
	{mustID("0000000000000000000000000000b000"), mustID("0000000000000000000000000000b001")},
	{mustID("0000000000000000000000000000e000"), mustID("0000000000000000000000000000e001")},
}

// Validate accepts only the canonical encoding of a complete Project-instance
// envelope with the exact contract pins and a content-derived artifact revision.
func Validate(source []byte) error {
	return validate(source, requiredPins)
}

// ValidateV8 authenticates the inherited ProjectSnapshot shape under the
// additive Project-v8, Package-v4, and Execution-v36 authorities.
func ValidateV8(source []byte) error {
	return validate(source, []pin{
		{mustID("00000000000000000000000000009000"), mustID("00000000000000000000000000009024")},
		{mustID("0000000000000000000000000000b000"), mustID("0000000000000000000000000000b004")},
		{mustID("0000000000000000000000000000e000"), mustID("0000000000000000000000000000e00a")},
	})
}

func validate(source []byte, pins []pin) error {
	e, err := wire.Decode(source)
	if err != nil {
		return fmt.Errorf("project_instance.wire:%w", err)
	}
	canonical, err := wire.Encode(e)
	if err != nil || !bytes.Equal(canonical, source) {
		return fmt.Errorf("project_instance.noncanonical")
	}
	if err := validateModule(e, pins); err != nil {
		return err
	}
	expected, err := ArtifactRevision(e)
	if err != nil {
		return err
	}
	if e.Revision != expected {
		return fmt.Errorf("project_instance.artifact_revision")
	}
	var snapshotErr error
	if len(pins) == 3 && pins[2].revision == mustID("0000000000000000000000000000e00a") {
		snapshotErr = projectsnapshot.ValidateV8(source)
	} else {
		snapshotErr = projectsnapshot.Validate(source)
	}
	if snapshotErr != nil {
		return fmt.Errorf("project_instance.snapshot:%w", snapshotErr)
	}
	return nil
}

// ArtifactRevision commits to the entire canonical envelope with only its
// revision slot cleared. It is distinct from the inner ProjectSnapshot digest.
func ArtifactRevision(e wire.Envelope) (wire.ID, error) {
	e.Revision = wire.ID{}
	encoded, err := wire.Encode(e)
	if err != nil {
		return wire.ID{}, fmt.Errorf("project_instance.artifact:%w", err)
	}
	h := sha256.New()
	h.Write([]byte("seme.project.artifact.v1\x00"))
	h.Write(encoded)
	var revision wire.ID
	copy(revision[:], h.Sum(nil)[:len(revision)])
	return revision, nil
}

func validateModule(e wire.Envelope, required []pin) error {
	module, ok := e.Entities[e.Module]
	if !ok || module.Schema != moduleSchema || module.Version != 1 {
		return fmt.Errorf("project_instance.module_declaration")
	}
	modules := 0
	for _, entity := range e.Entities {
		if entity.Schema == moduleSchema {
			modules++
		}
	}
	if modules != 1 {
		return fmt.Errorf("project_instance.module_count:%d", modules)
	}
	if len(module.Fields) != 3 || module.Fields[fModuleName].Tag != 5 || module.Fields[fExports].Tag != 7 {
		return fmt.Errorf("project_instance.module_shape")
	}
	if err := validateExports(e, module.Fields[fExports]); err != nil {
		return err
	}
	imports, ok := module.Fields[fImports]
	if !ok || imports.Tag != 7 || len(imports.List) != len(required) {
		return fmt.Errorf("project_instance.import_count")
	}
	ids := make([]wire.ID, 0, len(imports.List))
	seenIDs := map[wire.ID]bool{}
	seenPins := map[pin]bool{}
	for _, value := range imports.List {
		if value.Tag != 6 || seenIDs[value.Reference] {
			return fmt.Errorf("project_instance.import_reference")
		}
		seenIDs[value.Reference] = true
		ids = append(ids, value.Reference)
		entity, found := e.Entities[value.Reference]
		if !found || entity.Schema != importSchema || entity.Version != 1 || len(entity.Fields) != 2 {
			return fmt.Errorf("project_instance.import_shape:%s", value.Reference)
		}
		mv, mok := entity.Fields[fImportModule]
		rv, rok := entity.Fields[fImportRevision]
		if !mok || mv.Tag != 6 || !rok || rv.Tag != 5 || len(rv.Bytes) != len(wire.ID{}) {
			return fmt.Errorf("project_instance.import_shape:%s", value.Reference)
		}
		var revision wire.ID
		copy(revision[:], rv.Bytes)
		p := pin{mv.Reference, revision}
		if seenPins[p] {
			return fmt.Errorf("project_instance.import_duplicate")
		}
		seenPins[p] = true
	}
	if !sort.SliceIsSorted(ids, func(i, j int) bool { return bytes.Compare(ids[i][:], ids[j][:]) < 0 }) {
		return fmt.Errorf("project_instance.import_order")
	}
	for _, p := range required {
		if !seenPins[p] {
			return fmt.Errorf("project_instance.import_pin")
		}
	}
	for id, entity := range e.Entities {
		if entity.Schema == importSchema && !seenIDs[id] {
			return fmt.Errorf("project_instance.orphan_import:%s", id)
		}
	}
	return nil
}

func validateExports(e wire.Envelope, value wire.Value) error {
	seen := map[wire.ID]bool{}
	listed := make([]wire.ID, 0, len(value.List))
	for _, item := range value.List {
		if item.Tag != 6 || seen[item.Reference] {
			return fmt.Errorf("project_instance.export_reference")
		}
		entity, ok := e.Entities[item.Reference]
		if !ok || entity.Schema == importSchema {
			return fmt.Errorf("project_instance.export_target:%s", item.Reference)
		}
		seen[item.Reference] = true
		listed = append(listed, item.Reference)
	}
	if !sort.SliceIsSorted(listed, func(i, j int) bool { return bytes.Compare(listed[i][:], listed[j][:]) < 0 }) {
		return fmt.Errorf("project_instance.export_order")
	}
	expected := map[wire.ID]bool{}
	snapshots := 0
	for id, entity := range e.Entities {
		if entity.Schema != snapshotSchema {
			continue
		}
		snapshots++
		expected[id] = true
		identity, iok := entity.Fields[fIdentity]
		packages, pok := entity.Fields[fPackages]
		program, gok := entity.Fields[fProgram]
		if !iok || identity.Tag != 6 || !pok || packages.Tag != 7 || !gok || program.Tag != 6 {
			return fmt.Errorf("project_instance.export_snapshot_shape")
		}
		expected[identity.Reference] = true
		expected[program.Reference] = true
		for _, p := range packages.List {
			if p.Tag != 6 {
				return fmt.Errorf("project_instance.export_snapshot_shape")
			}
			expected[p.Reference] = true
		}
	}
	if snapshots != 1 || len(seen) != len(expected) {
		return fmt.Errorf("project_instance.export_set")
	}
	for id := range expected {
		if !seen[id] {
			return fmt.Errorf("project_instance.export_set")
		}
	}
	return nil
}

func mustID(value string) wire.ID {
	id, err := wire.ParseID(value)
	if err != nil {
		panic(err)
	}
	return id
}
