// Package controlledeffectsinstance emits and validates a project-neutral
// Controlled Effects v1 plan over an authenticated Project-v11 snapshot.
package controlledeffectsinstance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"unicode/utf8"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectv11instance"
	"seme.local/reference/wire"
)

const MaxIdentityBytes = 128

type Clock struct {
	Identity, MonotonicPolicy, InjectionPolicy string
}

type Random struct {
	Identity, Algorithm, OverflowPolicy string
}

type ExternalBooleanEffect struct {
	Identity, CapabilityIdentity, EffectIdentity, DeliveryPolicy string
}

type ReplayStep struct {
	CommandSequence, UnixMilliseconds, ClockSequence, RandomDraw uint64
	EffectValue                                                  bool
}

type Replay struct {
	InitialSeed                      uint64
	Steps                            []ReplayStep
	DuplicatePolicy, RejectionPolicy string
}

type Bounds struct {
	MaximumSteps, FirstClockSequence, ClockTerminalSentinel uint64
	MaximumUnixMilliseconds, MinimumSeed, MaximumSeed       uint64
	MaximumDraws, MaximumEffects                            uint64
}

// Model is the language-adapter boundary. All selections are explicit; this
// package never discovers ambient time, entropy, or effects.
type Model struct {
	Owner                         wire.ID
	Clock                         Clock
	Random                        Random
	ExternalEffect                ExternalBooleanEffect
	Replay                        Replay
	Bounds                        Bounds
	ApplyFunction, ReplayFunction wire.ID
}

type Inputs struct {
	Contracts  contractcatalog.ProjectContractSetV12
	ProjectV11 projectv11instance.Inputs
	Model      Model
	Artifact   []byte
}

func Emit(in Inputs) ([]byte, error) { return emit(in) }
func Validate(in Inputs) error {
	want, err := emit(Inputs{Contracts: in.Contracts, ProjectV11: in.ProjectV11, Model: in.Model})
	if err != nil {
		return err
	}
	if !bytes.Equal(want, in.Artifact) {
		return fmt.Errorf("controlled_effects.instance_artifact")
	}
	return nil
}

func emit(in Inputs) ([]byte, error) {
	if !in.Contracts.Validated() || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e031")}) || in.Contracts.ControlledEffects().Pin() != (contractcatalog.Pin{Module: id("13000"), Revision: id("13001")}) {
		return nil, fmt.Errorf("controlled_effects.instance_contracts")
	}
	if err := projectv11instance.Validate(in.ProjectV11); err != nil {
		return nil, fmt.Errorf("controlled_effects.project_v11:%w", err)
	}
	if !sameAuthorities(in.Contracts, in.ProjectV11.Contracts) {
		return nil, fmt.Errorf("controlled_effects.project_authorities")
	}
	base, err := wire.Decode(in.ProjectV11.Composed)
	if err != nil {
		return nil, fmt.Errorf("controlled_effects.project:%w", err)
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
	b := in.ProjectV11.Composed
	capability := stable(b, "capability", in.Model.ExternalEffect.CapabilityIdentity)
	effect := stable(b, "effect", in.Model.ExternalEffect.EffectIdentity)
	clock := stable(b, "clock")
	random := stable(b, "random")
	external := stable(b, "external-effect")
	replay := stable(b, "replay")
	bounds := stable(b, "bounds")
	e.Entities[capability] = entity(capability, "16", map[string]wire.Value{"160": blob(in.Model.ExternalEffect.CapabilityIdentity)})
	e.Entities[effect] = entity(effect, "15", map[string]wire.Value{"150": blob(in.Model.ExternalEffect.EffectIdentity), "151": ref(capability)})
	e.Entities[clock] = entity(clock, "13101", map[string]wire.Value{"13210": blob(in.Model.Clock.Identity), "13211": ref(id("13102")), "13212": blob(in.Model.Clock.MonotonicPolicy), "13213": blob(in.Model.Clock.InjectionPolicy)})
	e.Entities[random] = entity(random, "13103", map[string]wire.Value{"13230": blob(in.Model.Random.Identity), "13231": blob(in.Model.Random.Algorithm), "13232": blob(in.Model.Random.OverflowPolicy), "13233": ref(id("13104"))})
	e.Entities[external] = entity(external, "13105", map[string]wire.Value{"13250": blob(in.Model.ExternalEffect.Identity), "13251": ref(capability), "13252": ref(effect), "13253": ref(id("9020")), "13254": blob(in.Model.ExternalEffect.DeliveryPolicy)})
	stepRefs := make([]wire.Value, 0, len(in.Model.Replay.Steps))
	for i, s := range in.Model.Replay.Steps {
		x := stable(b, "replay-step", fmt.Sprint(i))
		clockSample := stable(b, "clock-sample", fmt.Sprint(i))
		e.Entities[clockSample] = entity(clockSample, "13102", map[string]wire.Value{"13220": u(s.UnixMilliseconds), "13221": u(s.ClockSequence)})
		e.Entities[x] = entity(x, "13106", map[string]wire.Value{"13260": u(s.CommandSequence), "13261": ref(clockSample), "13262": u(s.RandomDraw), "13263": boolean(s.EffectValue)})
		stepRefs = append(stepRefs, ref(x))
	}
	e.Entities[replay] = entity(replay, "13107", map[string]wire.Value{"13270": u(in.Model.Replay.InitialSeed), "13271": {Tag: 7, List: stepRefs}, "13272": blobBytes(replayDigest(in.Model.Replay)), "13273": blob(in.Model.Replay.DuplicatePolicy), "13274": blob(in.Model.Replay.RejectionPolicy)})
	z := in.Model.Bounds
	e.Entities[bounds] = entity(bounds, "13108", map[string]wire.Value{"13280": u(z.MaximumSteps), "13281": u(z.FirstClockSequence), "13282": u(z.ClockTerminalSentinel), "13283": u(z.MaximumUnixMilliseconds), "13284": u(z.MinimumSeed), "13285": u(z.MaximumSeed), "13286": u(z.MaximumDraws), "13287": u(z.MaximumEffects)})
	root := stable(b, "plan")
	e.Entities[root] = entity(root, "13100", map[string]wire.Value{"13200": ref(clock), "13201": ref(random), "13202": ref(external), "13203": ref(replay), "13204": ref(bounds), "13205": ref(in.Model.ApplyFunction), "13206": ref(in.Model.ReplayFunction), "13207": blobBytes(make([]byte, 32))})
	q := e.Entities[root]
	q.Fields[id("13207")] = blobBytes(contentRevision(e, root))
	e.Entities[root] = q
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("13000"), Revision: id("13001")}, {Module: id("3000"), Revision: id("3001")}, {Module: id("9000"), Revision: id("9024")}, {Module: id("b000"), Revision: id("b004")}, {Module: id("e000"), Revision: id("e030")}} {
		x := stable(b, "import", pin.Module.String())
		e.Entities[x] = entity(x, "13", map[string]wire.Value{"130": ref(pin.Module), "131": blobBytes(pin.Revision[:])})
		imports = append(imports, ref(x))
	}
	sortRefs(imports)
	e.Module = stable(b, "module")
	e.Entities[e.Module] = entity(e.Module, "12", map[string]wire.Value{"120": blob("controlled-effects-plan-v1"), "121": {Tag: 7, List: imports}, "122": list(root)})
	e.Revision = artifactRevision(e)
	return wire.Encode(e)
}

func validateModel(e wire.Envelope, m Model) error {
	for _, s := range []string{m.Clock.Identity, m.Clock.MonotonicPolicy, m.Clock.InjectionPolicy, m.Random.Identity, m.Random.Algorithm, m.Random.OverflowPolicy, m.ExternalEffect.Identity, m.ExternalEffect.CapabilityIdentity, m.ExternalEffect.EffectIdentity, m.ExternalEffect.DeliveryPolicy, m.Replay.DuplicatePolicy, m.Replay.RejectionPolicy} {
		if s == "" || len(s) > MaxIdentityBytes || !utf8.ValidString(s) {
			return fmt.Errorf("controlled_effects.metadata")
		}
	}
	z := m.Bounds
	if z.MaximumSteps == 0 || uint64(len(m.Replay.Steps)) > z.MaximumSteps || z.FirstClockSequence >= z.ClockTerminalSentinel || z.MinimumSeed > z.MaximumSeed || m.Replay.InitialSeed < z.MinimumSeed || m.Replay.InitialSeed > z.MaximumSeed || z.MaximumDraws == 0 || z.MaximumEffects == 0 {
		return fmt.Errorf("controlled_effects.bounds")
	}
	var effects uint64
	for i, s := range m.Replay.Steps {
		if s.ClockSequence < z.FirstClockSequence || s.ClockSequence >= z.ClockTerminalSentinel || s.UnixMilliseconds > z.MaximumUnixMilliseconds || s.RandomDraw >= z.MaximumDraws {
			return fmt.Errorf("controlled_effects.replay_step:%d", i)
		}
		if i > 0 && (s.CommandSequence <= m.Replay.Steps[i-1].CommandSequence || s.ClockSequence <= m.Replay.Steps[i-1].ClockSequence || s.UnixMilliseconds < m.Replay.Steps[i-1].UnixMilliseconds) {
			return fmt.Errorf("controlled_effects.replay_order:%d", i)
		}
		if s.EffectValue {
			effects++
		}
	}
	if effects > z.MaximumEffects || m.ApplyFunction == m.ReplayFunction || !ownedPureFunction(e, m.Owner, m.ApplyFunction) || !ownedPureFunction(e, m.Owner, m.ReplayFunction) {
		return fmt.Errorf("controlled_effects.function_or_effect")
	}
	return nil
}

func ownedPureFunction(e wire.Envelope, owner, fn wire.ID) bool {
	owned := false
	for _, q := range e.Entities {
		if q.Schema == id("b011") && q.Fields[id("b111")].Reference == fn {
			for _, p := range e.Entities {
				if p.Schema == id("b010") {
					for _, v := range p.Fields[id("b102")].List {
						if v.Reference == q.ID && p.ID == owner {
							owned = true
						}
					}
				}
			}
		}
	}
	q, ok := e.Entities[fn]
	return owned && ok && q.Schema == id("9011") && !reachesEffect(e, q.Fields[id("9113")].Reference, map[wire.ID]bool{})
}

func reachesEffect(e wire.Envelope, x wire.ID, seen map[wire.ID]bool) bool {
	if seen[x] {
		return false
	}
	seen[x] = true
	q, ok := e.Entities[x]
	if !ok {
		return true
	}
	if q.Schema == id("90f1") {
		return true
	}
	for _, v := range q.Fields {
		if valueReachesEffect(e, v, seen) {
			return true
		}
	}
	return false
}
func valueReachesEffect(e wire.Envelope, v wire.Value, seen map[wire.ID]bool) bool {
	if v.Tag == 6 {
		return reachesEffect(e, v.Reference, seen)
	}
	for _, x := range v.List {
		if valueReachesEffect(e, x, seen) {
			return true
		}
	}
	for _, x := range v.Record {
		if valueReachesEffect(e, x, seen) {
			return true
		}
	}
	return false
}

func replayDigest(r Replay) []byte {
	h := sha256.New()
	h.Write([]byte("seme.controlled-effects.replay.v1\x00"))
	_ = binary.Write(h, binary.BigEndian, r.InitialSeed)
	for _, s := range r.Steps {
		_ = binary.Write(h, binary.BigEndian, s.CommandSequence)
		_ = binary.Write(h, binary.BigEndian, s.UnixMilliseconds)
		_ = binary.Write(h, binary.BigEndian, s.ClockSequence)
		_ = binary.Write(h, binary.BigEndian, s.RandomDraw)
		if s.EffectValue {
			h.Write([]byte{1})
		} else {
			h.Write([]byte{0})
		}
	}
	writeString(h, r.DuplicatePolicy)
	writeString(h, r.RejectionPolicy)
	return h.Sum(nil)
}

type byteWriter interface{ Write([]byte) (int, error) }

func writeString(w byteWriter, s string) {
	_ = binary.Write(w, binary.BigEndian, uint64(len(s)))
	_, _ = w.Write([]byte(s))
}
func stable(base []byte, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.controlled-effects-plan.identity.v1\x00"))
	x := sha256.Sum256(base)
	h.Write(x[:])
	for _, p := range parts {
		writeString(h, p)
	}
	var out wire.ID
	copy(out[:], h.Sum(nil))
	return out
}
func id(s string) wire.ID {
	var x wire.ID
	for _, c := range []byte(s) {
		var n byte
		if c >= '0' && c <= '9' {
			n = c - '0'
		} else {
			n = c - 'a' + 10
		}
		carry := n
		for i := 15; i >= 0; i-- {
			v := uint16(x[i])*16 + uint16(carry)
			x[i] = byte(v)
			carry = byte(v >> 8)
		}
	}
	return x
}
func entity(x wire.ID, schema string, fs map[string]wire.Value) wire.Entity {
	f := map[wire.ID]wire.Value{}
	for k, v := range fs {
		f[id(k)] = v
	}
	return wire.Entity{ID: x, Schema: id(schema), Version: 1, Fields: f}
}
func ref(x wire.ID) wire.Value      { return wire.Value{Tag: 6, Reference: x} }
func blob(s string) wire.Value      { return blobBytes([]byte(s)) }
func blobBytes(b []byte) wire.Value { return wire.Value{Tag: 5, Bytes: append([]byte(nil), b...)} }
func u(x uint64) wire.Value         { return wire.Value{Tag: 3, Unsigned: x} }
func boolean(x bool) wire.Value {
	if x {
		return wire.Value{Tag: 2}
	}
	return wire.Value{Tag: 1}
}
func list(xs ...wire.ID) wire.Value {
	v := wire.Value{Tag: 7}
	for _, x := range xs {
		v.List = append(v.List, ref(x))
	}
	return v
}
func sortRefs(v []wire.Value) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && bytes.Compare(v[j].Reference[:], v[j-1].Reference[:]) < 0; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}
func contentRevision(e wire.Envelope, root wire.ID) []byte {
	q := e.Entities[root]
	f := map[wire.ID]wire.Value{}
	for k, v := range q.Fields {
		f[k] = v
	}
	q.Fields = f
	q.Fields[id("13207")] = blobBytes(make([]byte, 32))
	e.Entities[root] = q
	b, _ := wire.Encode(wire.Envelope{Entities: closure(e.Entities, root)})
	h := sha256.Sum256(append([]byte("seme.controlled-effects-plan.content.v1\x00"), b...))
	return h[:]
}
func closure(es map[wire.ID]wire.Entity, root wire.ID) map[wire.ID]wire.Entity {
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
			collect(v, &todo)
		}
	}
	return out
}
func collect(v wire.Value, out *[]wire.ID) {
	if v.Tag == 6 {
		*out = append(*out, v.Reference)
	}
	for _, x := range v.List {
		collect(x, out)
	}
	for _, x := range v.Record {
		collect(x, out)
	}
}
func artifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.controlled-effects-plan.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func sameAuthorities(a contractcatalog.ProjectContractSetV12, b contractcatalog.ProjectContractSetV11) bool {
	return a.Foundation().Pin() == b.Foundation().Pin() && a.Execution().Pin() == b.Execution().Pin() && a.Package().Pin() == b.Package().Pin() && a.Dependency().Pin() == b.Dependency().Pin() && a.Configuration().Pin() == b.Configuration().Pin() && a.Resource().Pin() == b.Resource().Pin() && a.DurableState().Pin() == b.DurableState().Pin() && a.Presentation().Pin() == b.Presentation().Pin() && a.OrderedTransport().Pin() == b.OrderedTransport().Pin() && b.Project().Pin() == (contractcatalog.Pin{Module: id("e000"), Revision: id("e030")})
}
