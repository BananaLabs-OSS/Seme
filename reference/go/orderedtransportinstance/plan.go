package orderedtransportinstance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"unicode/utf8"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectv10instance"
	"seme.local/reference/wire"
)

const MaxSelectedStreams = 64
const MaxSelectedKinds = 256
const MaxSelectedPorts = 64

type Stream struct {
	Identity string
	Owner    wire.ID
}

type Kind struct {
	Identity    string
	Owner       wire.ID
	PayloadType wire.ID
}

type Port struct {
	Identity string
	Owner    wire.ID
}

type Model struct {
	Streams          []Stream
	CommandKinds     []Kind
	EventKinds       []Kind
	Ports            []Port
	DispatchFunction wire.ID
	ReplayFunction   wire.ID
}

type Inputs struct {
	Contracts  contractcatalog.ProjectContractSetV11
	ProjectV10 projectv10instance.Inputs
	Model      Model
	Artifact   []byte
}

func Emit(in Inputs) ([]byte, error) { return emitPlan(in) }

func Validate(in Inputs) error {
	want, err := emitPlan(Inputs{Contracts: in.Contracts, ProjectV10: in.ProjectV10, Model: in.Model})
	if err != nil {
		return err
	}
	if !bytes.Equal(want, in.Artifact) {
		return fmt.Errorf("ordered_transport.plan_artifact")
	}
	return nil
}

func emitPlan(in Inputs) ([]byte, error) {
	if !in.Contracts.Validated() || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e030")}) || in.Contracts.OrderedTransport().Pin() != (contractcatalog.Pin{Module: id("10000"), Revision: id("10001")}) {
		return nil, fmt.Errorf("ordered_transport.plan_contracts")
	}
	if err := projectv10instance.Validate(in.ProjectV10); err != nil {
		return nil, fmt.Errorf("ordered_transport.project_v10:%w", err)
	}
	if !sameProjectAuthorities(in.Contracts, in.ProjectV10.Contracts) {
		return nil, fmt.Errorf("ordered_transport.project_authorities")
	}
	authority, err := Authenticate(in.Contracts.OrderedTransport())
	if err != nil {
		return nil, err
	}
	base, err := wire.Decode(in.ProjectV10.Composed)
	if err != nil {
		return nil, fmt.Errorf("ordered_transport.project:%w", err)
	}
	if err := validateModel(base, in.Model); err != nil {
		return nil, err
	}

	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for x, q := range base.Entities {
		if q.Schema != id("12") && q.Schema != id("13") {
			e.Entities[x] = q
		}
	}
	baseBytes := in.ProjectV10.Composed
	streamRefs := make([]wire.Value, 0, len(in.Model.Streams))
	for _, stream := range sortedStreams(in.Model.Streams) {
		x := planStable(baseBytes, "stream", stream.Owner.String(), stream.Identity)
		e.Entities[x] = planEntity(x, "10101", map[string]wire.Value{"11010": planBlob(stream.Identity), "11011": planRef(stream.Owner)})
		streamRefs = append(streamRefs, planRef(x))
	}
	commandRefs := emitKinds(e.Entities, baseBytes, "command", "10103", in.Model.CommandKinds)
	eventRefs := emitKinds(e.Entities, baseBytes, "event", "10104", in.Model.EventKinds)

	receiveCapability := planStable(baseBytes, "capability", authority.ReceiveCapability)
	sendCapability := planStable(baseBytes, "capability", authority.SendCapability)
	receiveEffect := planStable(baseBytes, "effect", authority.ReceiveIdentity)
	sendEffect := planStable(baseBytes, "effect", authority.SendIdentity)
	e.Entities[receiveCapability] = planEntity(receiveCapability, "16", map[string]wire.Value{"160": planBlob(authority.ReceiveCapability)})
	e.Entities[sendCapability] = planEntity(sendCapability, "16", map[string]wire.Value{"160": planBlob(authority.SendCapability)})
	e.Entities[receiveEffect] = planEntity(receiveEffect, "15", map[string]wire.Value{"150": planBlob(authority.ReceiveIdentity), "151": planRef(receiveCapability)})
	e.Entities[sendEffect] = planEntity(sendEffect, "15", map[string]wire.Value{"150": planBlob(authority.SendIdentity), "151": planRef(sendCapability)})
	auth := planStable(baseBytes, "operation-authority")
	e.Entities[auth] = planEntity(auth, "1010e", authorityFields(authority))
	portRefs := make([]wire.Value, 0, len(in.Model.Ports))
	for _, port := range sortedPorts(in.Model.Ports) {
		x := planStable(baseBytes, "port", port.Owner.String(), port.Identity)
		e.Entities[x] = planEntity(x, "1010f", map[string]wire.Value{"11100": planBlob(port.Identity), "11101": planRef(port.Owner), "11102": planRef(receiveCapability), "11103": planRef(sendCapability), "11104": planRef(receiveEffect), "11105": planRef(sendEffect), "11106": planRef(auth)})
		portRefs = append(portRefs, planRef(x))
	}
	root := planStable(baseBytes, "plan")
	e.Entities[root] = planEntity(root, "10100", map[string]wire.Value{"11000": {Tag: 7, List: streamRefs}, "11001": {Tag: 7, List: commandRefs}, "11002": {Tag: 7, List: eventRefs}, "11003": {Tag: 7, List: portRefs}, "11004": planBlobBytes(make([]byte, 32)), "11005": planRef(in.Model.DispatchFunction), "11006": planRef(in.Model.ReplayFunction)})
	q := e.Entities[root]
	q.Fields[id("11004")] = planBlobBytes(planContentRevision(e, root))
	e.Entities[root] = q

	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("10000"), Revision: id("10001")}, {Module: id("3000"), Revision: id("3001")}, {Module: id("9000"), Revision: id("9024")}, {Module: id("b000"), Revision: id("b004")}, {Module: id("e000"), Revision: id("e00e")}} {
		x := planStable(baseBytes, "import", pin.Module.String())
		e.Entities[x] = planEntity(x, "13", map[string]wire.Value{"130": planRef(pin.Module), "131": planBlobBytes(pin.Revision[:])})
		imports = append(imports, planRef(x))
	}
	planSortRefs(imports)
	e.Module = planStable(baseBytes, "module")
	e.Entities[e.Module] = planEntity(e.Module, "12", map[string]wire.Value{"120": planBlob("ordered-transport-plan-v1"), "121": {Tag: 7, List: imports}, "122": planList(root)})
	e.Revision = planArtifactRevision(e)
	return wire.Encode(e)
}

func validateModel(e wire.Envelope, m Model) error {
	if len(m.Streams) == 0 || len(m.Streams) > MaxSelectedStreams || len(m.CommandKinds) == 0 || len(m.CommandKinds) > MaxSelectedKinds || len(m.EventKinds) == 0 || len(m.EventKinds) > MaxSelectedKinds || len(m.Ports) == 0 || len(m.Ports) > MaxSelectedPorts {
		return fmt.Errorf("ordered_transport.plan_count")
	}
	owners, declarations := planOwnership(e)
	seen := map[string]bool{}
	for _, stream := range m.Streams {
		if !planName(stream.Identity, MaximumStreamBytes) || !owners[stream.Owner] || seen["s\x00"+stream.Identity] {
			return fmt.Errorf("ordered_transport.plan_stream")
		}
		seen["s\x00"+stream.Identity] = true
	}
	checkKinds := func(prefix string, kinds []Kind) error {
		for _, kind := range kinds {
			if !planName(kind.Identity, 128) || !owners[kind.Owner] || declarations[kind.PayloadType] != kind.Owner || !planType(e.Entities[kind.PayloadType]) || seen[prefix+"\x00"+kind.Identity] {
				return fmt.Errorf("ordered_transport.plan_kind:%s", prefix)
			}
			seen[prefix+"\x00"+kind.Identity] = true
		}
		return nil
	}
	if err := checkKinds("c", m.CommandKinds); err != nil {
		return err
	}
	if err := checkKinds("e", m.EventKinds); err != nil {
		return err
	}
	for _, port := range m.Ports {
		if !planName(port.Identity, 128) || !owners[port.Owner] || seen["p\x00"+port.Identity] {
			return fmt.Errorf("ordered_transport.plan_port")
		}
		seen["p\x00"+port.Identity] = true
	}
	if m.DispatchFunction == m.ReplayFunction || !planPureFunction(e, m.DispatchFunction, declarations[m.DispatchFunction]) || !planPureFunction(e, m.ReplayFunction, declarations[m.ReplayFunction]) {
		return fmt.Errorf("ordered_transport.plan_function")
	}
	return nil
}

func planOwnership(e wire.Envelope) (map[wire.ID]bool, map[wire.ID]wire.ID) {
	owners, declarations := map[wire.ID]bool{}, map[wire.ID]wire.ID{}
	for x, q := range e.Entities {
		if q.Schema != id("b010") {
			continue
		}
		owners[x] = true
		for _, v := range q.Fields[id("b102")].List {
			member := e.Entities[v.Reference]
			if member.Schema == id("b011") {
				declarations[member.Fields[id("b111")].Reference] = x
			}
		}
	}
	for _, q := range e.Entities {
		if q.Schema == id("b028") {
			detail := e.Entities[q.Fields[id("b281")].Reference]
			declarations[q.Fields[id("b280")].Reference] = detail.Fields[id("b210")].Reference
		}
	}
	return owners, declarations
}

func planPureFunction(e wire.Envelope, fn, owner wire.ID) bool {
	q, ok := e.Entities[fn]
	if !ok || owner == (wire.ID{}) || q.Schema != id("9011") {
		return false
	}
	seen := map[wire.ID]bool{}
	todo := []wire.ID{q.Fields[id("9113")].Reference}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if seen[x] {
			continue
		}
		seen[x] = true
		z, ok := e.Entities[x]
		if !ok {
			return false
		}
		if z.Schema == id("90f1") {
			return false
		}
		for _, v := range z.Fields {
			planCollect(v, &todo)
		}
	}
	return true
}

func planType(q wire.Entity) bool {
	switch q.Schema.String() {
	case id("9010").String(), id("9020").String(), id("9030").String(), id("9040").String(), id("9041").String(), id("9042").String(), id("90f2").String(), id("90f8").String(), id("a010").String(), id("a020").String(), id("a040").String(), id("a050").String():
		return true
	}
	return false
}

func authorityFields(a Authority) map[string]wire.Value {
	f := map[string]wire.Value{
		"110e0": planBlob(a.ReceiveIdentity), "110e1": planBlob(a.SendIdentity), "110e2": planBlob(a.ReceiveCapability), "110e3": planBlob(a.SendCapability),
		"110e4": planU(0), "110e5": planU(1), "110e6": planBlob(a.CodecIdentity), "110e7": planBlob(a.DigestIdentity),
		"110e8": planU(a.MaximumCommands), "110e9": planU(a.MaximumEvents), "110ea": planU(a.MaximumEventsPerCommand), "110eb": planU(a.MaximumPayloadBytes), "110ec": planU(a.MaximumFrameBytes), "110ed": planU(a.CorrelationBytes), "110ee": planU(a.MaximumStreamBytes),
		"110ef": planU(a.FirstCommandSequence), "110f0": planU(a.FirstEventSequence), "110f1": planU(a.CommandTerminalSentinel), "110f2": planU(a.EventTerminalSentinel), "110f3": planBlob(a.DuplicatePolicy), "110f4": planBlob(a.GapPolicy), "110f5": planBlob(a.CorrelationPolicy),
		"110f6": planBlob(a.CodecLayout[0]), "110f7": planBlob(a.CodecLayout[1]), "110f8": planBlob(a.CodecLayout[2]), "110f9": planBlob(a.CodecLayout[3]),
		"110fa": planRef(id("10114")), "110fb": planRef(id("10110")), "110fc": planRef(id("10111")), "110fd": planRef(id("10113")), "110fe": planRef(id("10112")),
		"110ff": planU(a.MaximumRetainedPayloadBytes),
	}
	return f
}

func emitKinds(es map[wire.ID]wire.Entity, base []byte, prefix, schema string, kinds []Kind) []wire.Value {
	kinds = sortedKinds(kinds)
	refs := make([]wire.Value, 0, len(kinds))
	for _, kind := range kinds {
		x := planStable(base, prefix, kind.Owner.String(), kind.Identity)
		identityField := "11040"
		if schema == "10103" {
			identityField = "11030"
		}
		es[x] = planEntity(x, schema, map[string]wire.Value{identityField: planBlob(kind.Identity)})
		q := es[x]
		if schema == "10103" {
			q.Fields[id("11031")] = planRef(kind.Owner)
			q.Fields[id("11032")] = planRef(kind.PayloadType)
		} else {
			q.Fields[id("11041")] = planRef(kind.Owner)
			q.Fields[id("11042")] = planRef(kind.PayloadType)
		}
		es[x] = q
		refs = append(refs, planRef(x))
	}
	return refs
}

func sortedStreams(in []Stream) []Stream {
	out := append([]Stream(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Owner == out[j].Owner {
			return out[i].Identity < out[j].Identity
		}
		return bytes.Compare(out[i].Owner[:], out[j].Owner[:]) < 0
	})
	return out
}
func sortedKinds(in []Kind) []Kind {
	out := append([]Kind(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Owner == out[j].Owner {
			return out[i].Identity < out[j].Identity
		}
		return bytes.Compare(out[i].Owner[:], out[j].Owner[:]) < 0
	})
	return out
}
func sortedPorts(in []Port) []Port {
	out := append([]Port(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Owner == out[j].Owner {
			return out[i].Identity < out[j].Identity
		}
		return bytes.Compare(out[i].Owner[:], out[j].Owner[:]) < 0
	})
	return out
}
func planName(s string, max uint64) bool {
	return s != "" && uint64(len(s)) <= max && utf8.ValidString(s)
}
func planCollect(v wire.Value, o *[]wire.ID) {
	if v.Tag == 6 {
		*o = append(*o, v.Reference)
	}
	if v.Tag == 7 {
		for _, x := range v.List {
			planCollect(x, o)
		}
	}
	if v.Tag == 8 {
		for _, x := range v.Record {
			planCollect(x, o)
		}
	}
}
func planStable(base []byte, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.ordered-transport-plan.identity.v1\x00"))
	x := sha256.Sum256(base)
	h.Write(x[:])
	for _, p := range parts {
		_ = binary.Write(h, binary.BigEndian, uint64(len(p)))
		h.Write([]byte(p))
	}
	var out wire.ID
	copy(out[:], h.Sum(nil))
	return out
}
func planEntity(x wire.ID, s string, fs map[string]wire.Value) wire.Entity {
	f := map[wire.ID]wire.Value{}
	for k, v := range fs {
		f[id(k)] = v
	}
	return wire.Entity{ID: x, Schema: id(s), Version: 1, Fields: f}
}
func planRef(x wire.ID) wire.Value      { return wire.Value{Tag: 6, Reference: x} }
func planBlob(s string) wire.Value      { return planBlobBytes([]byte(s)) }
func planBlobBytes(b []byte) wire.Value { return wire.Value{Tag: 5, Bytes: append([]byte(nil), b...)} }
func planU(x uint64) wire.Value         { return wire.Value{Tag: 3, Unsigned: x} }
func planList(xs ...wire.ID) wire.Value {
	v := wire.Value{Tag: 7}
	for _, x := range xs {
		v.List = append(v.List, planRef(x))
	}
	return v
}
func planSortRefs(v []wire.Value) {
	sort.Slice(v, func(i, j int) bool { return bytes.Compare(v[i].Reference[:], v[j].Reference[:]) < 0 })
}
func planContentRevision(e wire.Envelope, root wire.ID) []byte {
	q := e.Entities[root]
	q.Fields = planClone(q.Fields)
	q.Fields[id("11004")] = planBlobBytes(make([]byte, 32))
	e.Entities[root] = q
	b, _ := wire.Encode(wire.Envelope{Entities: planClosure(e.Entities, root)})
	h := sha256.Sum256(append([]byte("seme.ordered-transport-plan.content.v1\x00"), b...))
	return h[:]
}
func planArtifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.ordered-transport-plan.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func planClone(in map[wire.ID]wire.Value) map[wire.ID]wire.Value {
	out := map[wire.ID]wire.Value{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
func planClosure(es map[wire.ID]wire.Entity, root wire.ID) map[wire.ID]wire.Entity {
	out := map[wire.ID]wire.Entity{}
	todo := []wire.ID{root}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if _, ok := out[x]; ok {
			continue
		}
		q, ok := es[x]
		if !ok {
			continue
		}
		out[x] = q
		for _, v := range q.Fields {
			planCollect(v, &todo)
		}
	}
	return out
}
func sameProjectAuthorities(a contractcatalog.ProjectContractSetV11, b contractcatalog.ProjectContractSetV10) bool {
	return a.Foundation().Pin() == b.Foundation().Pin() && a.Execution().Pin() == b.Execution().Pin() && a.Package().Pin() == b.Package().Pin() && a.Dependency().Pin() == b.Dependency().Pin() && a.Configuration().Pin() == b.Configuration().Pin() && a.Resource().Pin() == b.Resource().Pin() && a.DurableState().Pin() == b.DurableState().Pin() && a.Presentation().Pin() == b.Presentation().Pin() && b.Project().Pin() == (contractcatalog.Pin{Module: id("e000"), Revision: id("e00e")})
}
