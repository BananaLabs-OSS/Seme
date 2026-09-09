// Package sourceinventory emits and validates detached, non-executable source
// inventories bound to an immutable semantic ProjectSnapshot.
package sourceinventory

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectinstance"
	"seme.local/reference/projectsource"
	"seme.local/reference/wire"
)

var (
	moduleSchema         = id("00000000000000000000000000000012")
	importSchema         = id("00000000000000000000000000000013")
	snapshotSchema       = id("0000000000000000000000000000e011")
	toolchainSchema      = id("0000000000000000000000000000e012")
	classificationSchema = id("0000000000000000000000000000e013")
	preservationSchema   = id("0000000000000000000000000000e014")
	unitSchema           = id("0000000000000000000000000000e015")
	inventorySchema      = id("0000000000000000000000000000e016")
	projectV2            = contractcatalog.Pin{Module: id("0000000000000000000000000000e000"), Revision: id("0000000000000000000000000000e002")}
)

func Emit(contract contractcatalog.Contract, project []byte, source projectsource.Snapshot) ([]byte, error) {
	if !contract.Validated() || contract.Pin() != projectV2 {
		return nil, fmt.Errorf("source_inventory.contract")
	}
	if err := projectinstance.Validate(project); err != nil {
		return nil, fmt.Errorf("source_inventory.project:%w", err)
	}
	if err := projectsource.ValidateSnapshot(source); err != nil {
		return nil, err
	}
	semantic, err := wire.Decode(project)
	if err != nil {
		return nil, err
	}
	snapshot, err := one(semantic, snapshotSchema)
	if err != nil {
		return nil, err
	}
	closure, err := reachable(semantic.Entities, snapshot)
	if err != nil {
		return nil, err
	}
	identityRef := closure[snapshot].Fields[id("0000000000000000000000000000e110")].Reference
	identity := closure[identityRef].Fields[id("0000000000000000000000000000e100")]
	if identity.Tag != 5 || string(identity.Bytes) != source.RootIdentity {
		return nil, fmt.Errorf("source_inventory.project_identity")
	}
	semanticRevision := closure[snapshot].Fields[id("0000000000000000000000000000e111")]
	if semanticRevision.Tag != 5 || len(semanticRevision.Bytes) != sha256.Size {
		return nil, fmt.Errorf("source_inventory.semantic_revision")
	}
	entities := closure
	toolchain := stable("toolchain", source.Toolchain.Language, source.Toolchain.Toolchain, source.Toolchain.Profile, source.Toolchain.SemanticRevision)
	entities[toolchain] = wire.Entity{ID: toolchain, Schema: toolchainSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("e120"): blob(source.Toolchain.Language), id("e121"): blob(source.Toolchain.Toolchain), id("e122"): blob(source.Toolchain.Profile), id("e123"): blob(source.Toolchain.SemanticRevision)}}
	classIDs := map[projectsource.Class]wire.ID{}
	preservationIDs := map[projectsource.Preservation]wire.ID{}
	unitRefs := []wire.Value{}
	for _, u := range source.Units {
		classCode, ok := classCode(u.Class)
		if !ok {
			return nil, fmt.Errorf("source_inventory.class")
		}
		cid := stable("classification", fmt.Sprint(classCode))
		classIDs[u.Class] = cid
		entities[cid] = wire.Entity{ID: cid, Schema: classificationSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("e130"): unsigned(classCode)}}
		mode, ok := preservationCode(u.Preservation)
		if !ok {
			return nil, fmt.Errorf("source_inventory.preservation")
		}
		pid := stable("preservation", fmt.Sprint(mode))
		preservationIDs[u.Preservation] = pid
		entities[pid] = wire.Entity{ID: pid, Schema: preservationSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("e140"): unsigned(mode)}}
		uid := stable("unit", source.RootIdentity, u.Path)
		unitRefs = append(unitRefs, ref(uid))
		digest, _ := hex.DecodeString(u.SHA256)
		entities[uid] = wire.Entity{ID: uid, Schema: unitSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("e150"): blob(u.Path), id("e151"): bytesValue(digest), id("e152"): unsigned(uint64(u.Size)), id("e153"): ref(cid), id("e154"): ref(pid), id("e155"): ref(toolchain)}}
	}
	toolRefs := []wire.Value{ref(toolchain)}
	inventory := stable("inventory", source.RootIdentity)
	revisionBytes, _ := hex.DecodeString(source.ContentRevision)
	entities[inventory] = wire.Entity{ID: inventory, Schema: inventorySchema, Version: 1, Fields: map[wire.ID]wire.Value{id("e160"): bytesValue(revisionBytes), id("e161"): ref(snapshot), id("e162"): semanticRevision, id("e163"): list(toolRefs), id("e164"): list(unitRefs)}}
	module := stable("module", source.RootIdentity)
	imp := stable("import", module.String())
	entities[imp] = wire.Entity{ID: imp, Schema: importSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("130"): ref(projectV2.Module), id("131"): bytesValue(projectV2.Revision[:])}}
	exports := []wire.Value{ref(inventory), ref(toolchain)}
	for _, x := range classIDs {
		exports = append(exports, ref(x))
	}
	for _, x := range preservationIDs {
		exports = append(exports, ref(x))
	}
	exports = append(exports, unitRefs...)
	sort.Slice(exports, func(i, j int) bool { return less(exports[i].Reference, exports[j].Reference) })
	entities[module] = wire.Entity{ID: module, Schema: moduleSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("120"): blob("source-inventory-v1"), id("121"): list([]wire.Value{ref(imp)}), id("122"): list(exports)}}
	envelope := wire.Envelope{Module: module, Entities: entities}
	revision, err := artifactRevision(envelope)
	if err != nil {
		return nil, err
	}
	envelope.Revision = revision
	out, err := wire.Encode(envelope)
	if err != nil {
		return nil, err
	}
	if err = Validate(contract, project, out); err != nil {
		return nil, err
	}
	return out, nil
}

func Validate(contract contractcatalog.Contract, project, source []byte) error {
	if !contract.Validated() || contract.Pin() != projectV2 {
		return fmt.Errorf("source_inventory.contract")
	}
	if err := projectinstance.Validate(project); err != nil {
		return err
	}
	semantic, _ := wire.Decode(project)
	boundSnapshot, err := one(semantic, snapshotSchema)
	if err != nil {
		return err
	}
	expectedClosure, err := reachable(semantic.Entities, boundSnapshot)
	if err != nil {
		return err
	}
	e, err := wire.Decode(source)
	if err != nil {
		return err
	}
	canonical, _ := wire.Encode(e)
	if !bytes.Equal(canonical, source) {
		return fmt.Errorf("source_inventory.noncanonical")
	}
	expectedRevision, err := artifactRevision(e)
	if err != nil || expectedRevision != e.Revision {
		return fmt.Errorf("source_inventory.artifact_revision")
	}
	m, ok := e.Entities[e.Module]
	if !ok || m.Schema != moduleSchema || m.Version != 1 || len(m.Fields) != 3 || m.Fields[id("120")].Tag != 5 || string(m.Fields[id("120")].Bytes) != "source-inventory-v1" || m.Fields[id("121")].Tag != 7 || m.Fields[id("122")].Tag != 7 {
		return fmt.Errorf("source_inventory.module")
	}
	imports := m.Fields[id("121")]
	if imports.Tag != 7 || len(imports.List) != 1 {
		return fmt.Errorf("source_inventory.import")
	}
	im := e.Entities[imports.List[0].Reference]
	if imports.List[0].Tag != 6 || im.Schema != importSchema || im.Version != 1 || len(im.Fields) != 2 || im.Fields[id("130")].Tag != 6 || im.Fields[id("131")].Tag != 5 || im.Fields[id("130")].Reference != projectV2.Module || !bytes.Equal(im.Fields[id("131")].Bytes, projectV2.Revision[:]) {
		return fmt.Errorf("source_inventory.import_pin")
	}
	snapshot, err := one(e, snapshotSchema)
	if err != nil || snapshot != boundSnapshot {
		return fmt.Errorf("source_inventory.snapshot_binding")
	}
	actualClosure, err := reachable(e.Entities, snapshot)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actualClosure, expectedClosure) {
		return fmt.Errorf("source_inventory.semantic_drift")
	}
	inventory, err := one(e, inventorySchema)
	if err != nil {
		return err
	}
	item := e.Entities[inventory]
	if item.Version != 1 || len(item.Fields) != 5 || item.Fields[id("e160")].Tag != 5 || len(item.Fields[id("e160")].Bytes) != sha256.Size || item.Fields[id("e161")].Tag != 6 || item.Fields[id("e162")].Tag != 5 || len(item.Fields[id("e162")].Bytes) != sha256.Size || item.Fields[id("e163")].Tag != 7 || item.Fields[id("e164")].Tag != 7 || item.Fields[id("e161")].Reference != snapshot {
		return fmt.Errorf("source_inventory.shape")
	}
	semanticRevision := expectedClosure[snapshot].Fields[id("e111")]
	if !reflect.DeepEqual(item.Fields[id("e162")], semanticRevision) {
		return fmt.Errorf("source_inventory.semantic_revision")
	}
	toolValues := item.Fields[id("e163")]
	unitValues := item.Fields[id("e164")]
	if toolValues.Tag != 7 || len(toolValues.List) != 1 || toolValues.List[0].Tag != 6 || unitValues.Tag != 7 || len(unitValues.List) == 0 {
		return fmt.Errorf("source_inventory.members")
	}
	tool := e.Entities[toolValues.List[0].Reference]
	if tool.Schema != toolchainSchema || tool.Version != 1 || len(tool.Fields) != 4 {
		return fmt.Errorf("source_inventory.toolchain")
	}
	for _, field := range []wire.ID{id("e120"), id("e121"), id("e122"), id("e123")} {
		if tool.Fields[field].Tag != 5 || len(tool.Fields[field].Bytes) == 0 {
			return fmt.Errorf("source_inventory.toolchain_field")
		}
	}
	snapshotValue := projectsource.Snapshot{RootIdentity: string(expectedClosure[expectedClosure[snapshot].Fields[id("e110")].Reference].Fields[id("e100")].Bytes), ContentRevision: hex.EncodeToString(item.Fields[id("e160")].Bytes), Toolchain: projectsource.Toolchain{Language: string(tool.Fields[id("e120")].Bytes), Toolchain: string(tool.Fields[id("e121")].Bytes), Profile: string(tool.Fields[id("e122")].Bytes), SemanticRevision: string(tool.Fields[id("e123")].Bytes)}}
	seenRefs := map[wire.ID]bool{}
	seenPaths := map[string]bool{}
	priorPath := ""
	expectedExports := map[wire.ID]bool{inventory: true, toolValues.List[0].Reference: true}
	for _, v := range unitValues.List {
		if v.Tag != 6 || seenRefs[v.Reference] {
			return fmt.Errorf("source_inventory.unit_ref")
		}
		seenRefs[v.Reference] = true
		expectedExports[v.Reference] = true
		u := e.Entities[v.Reference]
		if u.Schema != unitSchema || u.Version != 1 || len(u.Fields) != 6 {
			return fmt.Errorf("source_inventory.unit")
		}
		if u.Fields[id("e150")].Tag != 5 || u.Fields[id("e151")].Tag != 5 || u.Fields[id("e152")].Tag != 3 || u.Fields[id("e153")].Tag != 6 || u.Fields[id("e154")].Tag != 6 || u.Fields[id("e155")].Tag != 6 {
			return fmt.Errorf("source_inventory.unit_fields")
		}
		if u.Fields[id("e155")].Reference != toolValues.List[0].Reference {
			return fmt.Errorf("source_inventory.unit_toolchain")
		}
		classificationID := u.Fields[id("e153")].Reference
		preservationID := u.Fields[id("e154")].Reference
		classification := e.Entities[classificationID]
		preservation := e.Entities[preservationID]
		expectedExports[classificationID] = true
		expectedExports[preservationID] = true
		class, ok := decodeClass(classification)
		if !ok {
			return fmt.Errorf("source_inventory.classification")
		}
		mode, ok := decodePreservation(preservation)
		if !ok {
			return fmt.Errorf("source_inventory.preservation")
		}
		size := u.Fields[id("e152")]
		digest := u.Fields[id("e151")]
		path := u.Fields[id("e150")]
		if path.Tag != 5 || digest.Tag != 5 || len(digest.Bytes) != sha256.Size || size.Tag != 3 {
			return fmt.Errorf("source_inventory.unit_fields")
		}
		snapshotValue.Units = append(snapshotValue.Units, projectsource.Unit{Path: string(path.Bytes), SHA256: hex.EncodeToString(digest.Bytes), Size: int64(size.Unsigned), Class: class, Preservation: mode})
		pathText := string(path.Bytes)
		if seenPaths[pathText] || pathText <= priorPath {
			return fmt.Errorf("source_inventory.unit_order")
		}
		seenPaths[pathText] = true
		priorPath = pathText
		if v.Reference != stable("unit", snapshotValue.RootIdentity, pathText) || classificationID != stable("classification", fmt.Sprint(classification.Fields[id("e130")].Unsigned)) || preservationID != stable("preservation", fmt.Sprint(preservation.Fields[id("e140")].Unsigned)) {
			return fmt.Errorf("source_inventory.identity")
		}
	}
	if toolValues.List[0].Reference != stable("toolchain", snapshotValue.Toolchain.Language, snapshotValue.Toolchain.Toolchain, snapshotValue.Toolchain.Profile, snapshotValue.Toolchain.SemanticRevision) || inventory != stable("inventory", snapshotValue.RootIdentity) || e.Module != stable("module", snapshotValue.RootIdentity) || imports.List[0].Reference != stable("import", e.Module.String()) {
		return fmt.Errorf("source_inventory.identity")
	}
	exports := m.Fields[id("122")]
	if exports.Tag != 7 || len(exports.List) != len(expectedExports) {
		return fmt.Errorf("source_inventory.exports")
	}
	var previous wire.ID
	for i, v := range exports.List {
		if v.Tag != 6 || !expectedExports[v.Reference] || (i > 0 && !less(previous, v.Reference)) {
			return fmt.Errorf("source_inventory.exports")
		}
		previous = v.Reference
	}
	allowed := map[wire.ID]bool{}
	for x := range expectedClosure {
		allowed[x] = true
	}
	allowed[e.Module] = true
	allowed[imports.List[0].Reference] = true
	for x := range expectedExports {
		allowed[x] = true
	}
	for x := range e.Entities {
		if !allowed[x] {
			return fmt.Errorf("source_inventory.orphan:%s", x)
		}
	}
	if err := projectsource.ValidateSnapshot(snapshotValue); err != nil {
		return err
	}
	return nil
}

func reachable(all map[wire.ID]wire.Entity, root wire.ID) (map[wire.ID]wire.Entity, error) {
	out := map[wire.ID]wire.Entity{}
	q := []wire.ID{root}
	for len(q) > 0 {
		x := q[0]
		q = q[1:]
		if _, ok := out[x]; ok {
			continue
		}
		entity, ok := all[x]
		if !ok {
			return nil, fmt.Errorf("source_inventory.reference_missing:%s", x)
		}
		out[x] = entity
		for _, v := range entity.Fields {
			collect(v, &q)
		}
	}
	return out, nil
}
func collect(v wire.Value, q *[]wire.ID) {
	if v.Tag == 6 {
		*q = append(*q, v.Reference)
	}
	for _, x := range v.List {
		collect(x, q)
	}
	for _, x := range v.Record {
		collect(x, q)
	}
}
func artifactRevision(e wire.Envelope) (wire.ID, error) {
	e.Revision = wire.ID{}
	b, err := wire.Encode(e)
	if err != nil {
		return wire.ID{}, err
	}
	sum := sha256.Sum256(append([]byte("seme.source-inventory.artifact.v1\x00"), b...))
	var out wire.ID
	copy(out[:], sum[:16])
	return out, nil
}
func one(e wire.Envelope, schema wire.ID) (wire.ID, error) {
	var out wire.ID
	for x, v := range e.Entities {
		if v.Schema == schema {
			if out != (wire.ID{}) {
				return out, fmt.Errorf("source_inventory.count")
			}
			out = x
		}
	}
	if out == (wire.ID{}) {
		return out, fmt.Errorf("source_inventory.count")
	}
	return out, nil
}
func classCode(c projectsource.Class) (uint64, bool) {
	switch c {
	case projectsource.Tracked:
		return 0, true
	case projectsource.Ignored:
		return 1, true
	case projectsource.Generated:
		return 2, true
	case projectsource.Vendored:
		return 3, true
	case projectsource.Opaque:
		return 4, true
	}
	return 0, false
}
func preservationCode(p projectsource.Preservation) (uint64, bool) {
	switch p {
	case projectsource.ByteExact:
		return 0, true
	case projectsource.SemanticProjection:
		return 1, true
	}
	return 0, false
}
func decodeClass(e wire.Entity) (projectsource.Class, bool) {
	if e.Schema != classificationSchema || e.Version != 1 || len(e.Fields) != 1 {
		return "", false
	}
	v := e.Fields[id("e130")]
	if v.Tag != 3 {
		return "", false
	}
	values := []projectsource.Class{projectsource.Tracked, projectsource.Ignored, projectsource.Generated, projectsource.Vendored, projectsource.Opaque}
	if v.Unsigned >= uint64(len(values)) {
		return "", false
	}
	return values[v.Unsigned], true
}
func decodePreservation(e wire.Entity) (projectsource.Preservation, bool) {
	if e.Schema != preservationSchema || e.Version != 1 || len(e.Fields) != 1 {
		return "", false
	}
	v := e.Fields[id("e140")]
	if v.Tag != 3 {
		return "", false
	}
	values := []projectsource.Preservation{projectsource.ByteExact, projectsource.SemanticProjection}
	if v.Unsigned >= uint64(len(values)) {
		return "", false
	}
	return values[v.Unsigned], true
}
func stable(parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.source-inventory.identity.v1\x00"))
	for _, p := range parts {
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(p)))
		h.Write(n[:])
		h.Write([]byte(p))
	}
	var out wire.ID
	out[0] = 0x80
	copy(out[1:], h.Sum(nil)[:15])
	return out
}
func id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
func ref(x wire.ID) wire.Value       { return wire.Value{Tag: 6, Reference: x} }
func blob(x string) wire.Value       { return bytesValue([]byte(x)) }
func bytesValue(x []byte) wire.Value { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func unsigned(x uint64) wire.Value   { return wire.Value{Tag: 3, Unsigned: x} }
func list(x []wire.Value) wire.Value { return wire.Value{Tag: 7, List: x} }
func less(a, b wire.ID) bool         { return bytes.Compare(a[:], b[:]) < 0 }
