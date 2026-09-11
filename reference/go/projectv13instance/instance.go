// Package projectv13instance binds an authenticated Project-v12 snapshot to
// one independently valid, executable Target-v1 placement plan.
package projectv13instance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectv12instance"
	"seme.local/reference/targetplaninstance"
	"seme.local/reference/wire"
)

type Inputs struct {
	Contracts  contractcatalog.ProjectContractSetV13
	ProjectV12 projectv12instance.Inputs
	Plan       targetplaninstance.Inputs
	Composed   []byte
}

func Emit(in Inputs) ([]byte, error) { return emit(in) }

func Validate(in Inputs) error {
	want, err := emit(Inputs{Contracts: in.Contracts, ProjectV12: in.ProjectV12, Plan: targetplaninstance.Inputs{
		TargetContract: in.Plan.TargetContract, SemanticContracts: in.Plan.SemanticContracts, Authority: in.Plan.Authority, Model: in.Plan.Model, Artifact: in.Plan.Artifact,
	}})
	if err != nil {
		return err
	}
	if !bytes.Equal(want, in.Composed) {
		return fmt.Errorf("project_v13.artifact")
	}
	return nil
}

func emit(in Inputs) ([]byte, error) {
	if !in.Contracts.Validated() || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: xid("e000"), Revision: xid("e032")}) || in.Contracts.Target().Digest() != in.Plan.TargetContract.Digest() {
		return nil, fmt.Errorf("project_v13.contracts")
	}
	if err := projectv12instance.Validate(in.ProjectV12); err != nil {
		return nil, fmt.Errorf("project_v13.project_v12:%w", err)
	}
	if !bytes.Equal(in.ProjectV12.Composed, in.Plan.Authority) {
		return nil, fmt.Errorf("project_v13.mixed_project")
	}
	if err := targetplaninstance.Validate(in.Plan); err != nil {
		return nil, fmt.Errorf("project_v13.plan:%w", err)
	}
	project, err := wire.Decode(in.ProjectV12.Composed)
	if err != nil {
		return nil, err
	}
	plan, err := wire.Decode(in.Plan.Artifact)
	if err != nil {
		return nil, err
	}
	out := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for _, part := range []wire.Envelope{project, plan} {
		for identity, value := range part.Entities {
			if prior, exists := out.Entities[identity]; exists && !same(prior, value) {
				return nil, fmt.Errorf("project_v13.collision:%s", identity)
			}
			out.Entities[identity] = value
		}
	}
	base, err := exactlyOne(out, xid("e029"))
	if err != nil {
		return nil, err
	}
	executionPlan, err := exactlyOne(out, xid("c014"))
	if err != nil {
		return nil, err
	}
	p := out.Entities[executionPlan]
	root, okay := p.Fields[xid("c140")]
	executable, executableOkay := p.Fields[xid("c144")]
	if !okay || root.Tag != 6 || root.Reference != base || !executableOkay || executable.Tag != 2 {
		return nil, fmt.Errorf("project_v13.non_executable_or_wrong_root")
	}

	snapshot := stable(in, "snapshot")
	out.Entities[snapshot] = makeEntity(snapshot, "e02c", map[string]wire.Value{
		"e2c0": reference(base), "e2c1": reference(executionPlan), "e2c2": bytesValue(make([]byte, 32)),
	})
	value := out.Entities[snapshot]
	value.Fields[xid("e2c2")] = bytesValue(contentRevision(out, snapshot))
	out.Entities[snapshot] = value

	pins := []contractcatalog.Pin{
		{Module: xid("1000"), Revision: xid("1001")}, {Module: xid("10000"), Revision: xid("10001")},
		{Module: xid("13000"), Revision: xid("13001")}, {Module: xid("3000"), Revision: xid("3001")},
		{Module: xid("4000"), Revision: xid("4006")}, {Module: xid("6000"), Revision: xid("6001")},
		{Module: xid("8000"), Revision: xid("8001")}, {Module: xid("9000"), Revision: xid("9024")},
		{Module: xid("b000"), Revision: xid("b004")}, {Module: xid("c000"), Revision: xid("c001")},
		{Module: xid("e000"), Revision: xid("e032")}, {Module: xid("f000"), Revision: xid("f001")},
	}
	imports := make([]wire.Value, 0, len(pins))
	for _, pin := range pins {
		identity := stable(in, "import", pin.Module.String())
		out.Entities[identity] = makeEntity(identity, "13", map[string]wire.Value{"130": reference(pin.Module), "131": bytesValue(pin.Revision[:])})
		imports = append(imports, reference(identity))
	}
	sort.Slice(imports, func(i, j int) bool { return bytes.Compare(imports[i].Reference[:], imports[j].Reference[:]) < 0 })
	out.Module = stable(in, "module")
	out.Entities[out.Module] = makeEntity(out.Module, "12", map[string]wire.Value{
		"120": bytesValue([]byte("placed-project-v1")), "121": {Tag: 7, List: imports}, "122": references(snapshot),
	})
	out.Revision = artifactRevision(out)
	return wire.Encode(out)
}

func stable(in Inputs, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.project-v13.identity.v1\x00"))
	for _, source := range [][]byte{in.ProjectV12.Composed, in.Plan.Artifact} {
		digest := sha256.Sum256(source)
		h.Write(digest[:])
	}
	for _, part := range parts {
		_ = binary.Write(h, binary.BigEndian, uint64(len(part)))
		h.Write([]byte(part))
	}
	var result wire.ID
	copy(result[:], h.Sum(nil))
	return result
}

func contentRevision(envelope wire.Envelope, root wire.ID) []byte {
	value := envelope.Entities[root]
	value.Fields = cloneFields(value.Fields)
	value.Fields[xid("e2c2")] = bytesValue(make([]byte, 32))
	envelope.Entities[root] = value
	encoded, _ := wire.Encode(wire.Envelope{Entities: closure(envelope.Entities, root)})
	digest := sha256.Sum256(append([]byte("seme.project-v13.snapshot.v1\x00"), encoded...))
	return digest[:]
}

func artifactRevision(envelope wire.Envelope) wire.ID {
	envelope.Revision = wire.ID{}
	encoded, _ := wire.Encode(envelope)
	digest := sha256.Sum256(append([]byte("seme.project-v13.artifact.v1\x00"), encoded...))
	var revision wire.ID
	copy(revision[:], digest[:16])
	return revision
}

func exactlyOne(envelope wire.Envelope, schema wire.ID) (wire.ID, error) {
	var result wire.ID
	for identity, value := range envelope.Entities {
		if value.Schema == schema {
			if result != (wire.ID{}) {
				return result, fmt.Errorf("project_v13.count:%s", schema)
			}
			result = identity
		}
	}
	if result == (wire.ID{}) {
		return result, fmt.Errorf("project_v13.count:%s", schema)
	}
	return result, nil
}

func closure(entities map[wire.ID]wire.Entity, root wire.ID) map[wire.ID]wire.Entity {
	result := map[wire.ID]wire.Entity{}
	queue := []wire.ID{root}
	for len(queue) > 0 {
		identity := queue[0]
		queue = queue[1:]
		if _, exists := result[identity]; exists {
			continue
		}
		value, exists := entities[identity]
		if !exists {
			continue
		}
		result[identity] = value
		for _, field := range value.Fields {
			collect(field, &queue)
		}
	}
	return result
}

func collect(value wire.Value, out *[]wire.ID) {
	if value.Tag == 6 {
		*out = append(*out, value.Reference)
	}
	for _, item := range value.List {
		collect(item, out)
	}
	for _, item := range value.Record {
		collect(item, out)
	}
}

func same(left, right wire.Entity) bool {
	a, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{left.ID: left}})
	b, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{right.ID: right}})
	return bytes.Equal(a, b)
}

func makeEntity(identity wire.ID, schema string, fields map[string]wire.Value) wire.Entity {
	converted := map[wire.ID]wire.Value{}
	for key, value := range fields {
		converted[xid(key)] = value
	}
	return wire.Entity{ID: identity, Schema: xid(schema), Version: 1, Fields: converted}
}

func reference(identity wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: identity} }
func bytesValue(value []byte) wire.Value    { return wire.Value{Tag: 5, Bytes: bytes.Clone(value)} }
func references(identities ...wire.ID) wire.Value {
	result := wire.Value{Tag: 7}
	for _, identity := range identities {
		result.List = append(result.List, reference(identity))
	}
	return result
}
func cloneFields(fields map[wire.ID]wire.Value) map[wire.ID]wire.Value {
	result := map[wire.ID]wire.Value{}
	for key, value := range fields {
		result[key] = value
	}
	return result
}
func xid(value string) wire.ID {
	for len(value) < 32 {
		value = "0" + value
	}
	identity, err := wire.ParseID(value)
	if err != nil {
		panic(err)
	}
	return identity
}
