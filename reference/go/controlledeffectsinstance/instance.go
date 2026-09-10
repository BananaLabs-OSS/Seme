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
	"seme.local/reference/wasmtarget"
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
	CommandSequence, ClockSequence         uint64
	UnixMilliseconds                       int64
	RandomBefore, RandomAfter, RandomValue int64
	RandomDrawOrdinal                      uint64
	EffectValue                            bool
	CanonicalCommand                       []byte
	ResponseSHA256, EventsSHA256           [32]byte
	StateBeforeSHA256, StateAfterSHA256    [32]byte
}

type Replay struct {
	InitialSeed                      int64
	InitialState                     []byte
	Steps                            []ReplayStep
	DuplicatePolicy, RejectionPolicy string
}

type Bounds struct {
	MaximumSteps, FirstClockSequence, ClockTerminalSentinel uint64
	MaximumUnixMilliseconds, MinimumSeed, MaximumSeed       int64
	MaximumDraws, MaximumEffects                            uint64
	MaximumCommandBytes, MaximumInitialStateBytes           uint64
	MaximumTranscriptBytes                                  uint64
}

// Model is the language-adapter boundary. All selections are explicit; this
// package never discovers ambient time, entropy, or effects.
type Model struct {
	ClockOwner, ClockSample             wire.ID
	RandomOwner, RandomState, Draw      wire.ID
	EffectOwner, Command, State, Result wire.ID
	NextFunction, DispatchFunction      wire.ID
	ReplayFunction                      wire.ID
	Clock                               Clock
	Random                              Random
	ExternalEffect                      ExternalBooleanEffect
	Replay                              Replay
	Bounds                              Bounds
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
	e.Entities[random] = entity(random, "13103", map[string]wire.Value{"13230": blob(in.Model.Random.Identity), "13231": blob(in.Model.Random.Algorithm), "13232": blob(in.Model.Random.OverflowPolicy), "13233": ref(id("13104")), "13234": ref(in.Model.NextFunction)})
	e.Entities[external] = entity(external, "13105", map[string]wire.Value{"13250": blob(in.Model.ExternalEffect.Identity), "13251": ref(capability), "13252": ref(effect), "13253": ref(id("9020")), "13254": blob(in.Model.ExternalEffect.DeliveryPolicy)})
	stepRefs := make([]wire.Value, 0, len(in.Model.Replay.Steps))
	for i, s := range in.Model.Replay.Steps {
		x := stable(b, "replay-step", fmt.Sprint(i))
		clockSample := stable(b, "clock-sample", fmt.Sprint(i))
		e.Entities[clockSample] = entity(clockSample, "13102", map[string]wire.Value{"13220": signed(s.UnixMilliseconds), "13221": u(s.ClockSequence)})
		e.Entities[x] = entity(x, "13106", map[string]wire.Value{"13260": u(s.CommandSequence), "13261": ref(clockSample), "13262": signed(s.RandomBefore), "13263": signed(s.RandomAfter), "13264": signed(s.RandomValue), "13265": u(s.RandomDrawOrdinal), "13266": boolean(s.EffectValue), "13267": blobBytes(s.CanonicalCommand), "13268": blobBytes(s.ResponseSHA256[:]), "13269": blobBytes(s.EventsSHA256[:]), "1326a": blobBytes(s.StateBeforeSHA256[:]), "1326b": blobBytes(s.StateAfterSHA256[:])})
		stepRefs = append(stepRefs, ref(x))
	}
	initialStateDigest := sha256.Sum256(in.Model.Replay.InitialState)
	e.Entities[replay] = entity(replay, "13107", map[string]wire.Value{"13270": signed(in.Model.Replay.InitialSeed), "13271": {Tag: 7, List: stepRefs}, "13272": blobBytes(replayDigest(in.Model.Replay)), "13273": blob(in.Model.Replay.DuplicatePolicy), "13274": blob(in.Model.Replay.RejectionPolicy), "13275": blobBytes(in.Model.Replay.InitialState), "13276": blobBytes(initialStateDigest[:])})
	z := in.Model.Bounds
	e.Entities[bounds] = entity(bounds, "13108", map[string]wire.Value{"13280": u(z.MaximumSteps), "13281": u(z.FirstClockSequence), "13282": u(z.ClockTerminalSentinel), "13283": signed(z.MaximumUnixMilliseconds), "13284": signed(z.MinimumSeed), "13285": signed(z.MaximumSeed), "13286": u(z.MaximumDraws), "13287": u(z.MaximumEffects), "13288": u(z.MaximumCommandBytes), "13289": u(z.MaximumInitialStateBytes), "1328a": u(z.MaximumTranscriptBytes)})
	root := stable(b, "plan")
	e.Entities[root] = entity(root, "13100", map[string]wire.Value{"13200": ref(clock), "13201": ref(random), "13202": ref(external), "13203": ref(replay), "13204": ref(bounds), "13205": ref(in.Model.DispatchFunction), "13206": ref(in.Model.ReplayFunction), "13207": blobBytes(make([]byte, 32))})
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
	if z.MaximumSteps == 0 || uint64(len(m.Replay.Steps)) > z.MaximumSteps || z.FirstClockSequence >= z.ClockTerminalSentinel || z.MinimumSeed > z.MaximumSeed || m.Replay.InitialSeed < z.MinimumSeed || m.Replay.InitialSeed > z.MaximumSeed || z.MaximumDraws == 0 || z.MaximumEffects == 0 || z.MaximumCommandBytes == 0 || z.MaximumInitialStateBytes == 0 || z.MaximumTranscriptBytes == 0 || uint64(len(m.Replay.InitialState)) > z.MaximumInitialStateBytes || uint64(len(m.Replay.InitialState)) > z.MaximumTranscriptBytes {
		return fmt.Errorf("controlled_effects.bounds")
	}
	layout, err := wasmtarget.CertifyPureValueLayout(e, m.State)
	if err != nil || (len(m.Replay.Steps) != 0 && len(m.Replay.InitialState) == 0) {
		return fmt.Errorf("controlled_effects.initial_state_layout")
	}
	if len(m.Replay.InitialState) != 0 {
		initial, decodeErr := wasmtarget.DecodePureValue(layout, m.Replay.InitialState)
		if decodeErr != nil {
			return fmt.Errorf("controlled_effects.initial_state")
		}
		canonical, encodeErr := wasmtarget.EncodePureValue(layout, initial)
		if encodeErr != nil || !bytes.Equal(canonical, m.Replay.InitialState) {
			return fmt.Errorf("controlled_effects.initial_state_canonical")
		}
	}
	commandLayout, err := wasmtarget.CertifyPureValueLayout(e, m.Command)
	if err != nil {
		return fmt.Errorf("controlled_effects.command_layout")
	}
	transcriptBytes := uint64(len(m.Replay.InitialState)) + uint64(len(m.Replay.DuplicatePolicy)) + uint64(len(m.Replay.RejectionPolicy)) + 32
	initialDigest := sha256.Sum256(m.Replay.InitialState)
	previousState := initialDigest
	for i, s := range m.Replay.Steps {
		if s.ClockSequence < z.FirstClockSequence || s.ClockSequence >= z.ClockTerminalSentinel || s.UnixMilliseconds > z.MaximumUnixMilliseconds || s.RandomDrawOrdinal == 0 || s.RandomDrawOrdinal > z.MaximumDraws || len(s.CanonicalCommand) == 0 || uint64(len(s.CanonicalCommand)) > z.MaximumCommandBytes {
			return fmt.Errorf("controlled_effects.replay_step:%d", i)
		}
		command, decodeErr := wasmtarget.DecodePureValue(commandLayout, s.CanonicalCommand)
		if decodeErr != nil {
			return fmt.Errorf("controlled_effects.command:%d", i)
		}
		canonical, encodeErr := wasmtarget.EncodePureValue(commandLayout, command)
		if encodeErr != nil || !bytes.Equal(canonical, s.CanonicalCommand) {
			return fmt.Errorf("controlled_effects.command_canonical:%d", i)
		}
		ordinal := uint64(i + 1)
		if s.CommandSequence != ordinal || s.ClockSequence != ordinal || s.RandomDrawOrdinal != ordinal || (i == 0 && s.RandomBefore != m.Replay.InitialSeed) || (i > 0 && s.RandomBefore != m.Replay.Steps[i-1].RandomAfter) || s.RandomAfter != s.RandomBefore*48271+1 || s.RandomAfter != s.RandomValue || s.StateBeforeSHA256 != previousState || zeroDigest(s.ResponseSHA256) || zeroDigest(s.EventsSHA256) || zeroDigest(s.StateAfterSHA256) {
			return fmt.Errorf("controlled_effects.replay_chain:%d", i)
		}
		previousState = s.StateAfterSHA256
		stepBytes := uint64(len(s.CanonicalCommand)) + 8*7 + 1 + 32*4 + 8
		if transcriptBytes > z.MaximumTranscriptBytes || stepBytes > z.MaximumTranscriptBytes-transcriptBytes {
			return fmt.Errorf("controlled_effects.transcript_bound")
		}
		transcriptBytes += stepBytes
		if i > 0 && (s.CommandSequence <= m.Replay.Steps[i-1].CommandSequence || s.ClockSequence <= m.Replay.Steps[i-1].ClockSequence || s.UnixMilliseconds < m.Replay.Steps[i-1].UnixMilliseconds) {
			return fmt.Errorf("controlled_effects.replay_order:%d", i)
		}
	}
	owners := declarationOwners(e)
	for x, owner := range map[wire.ID]wire.ID{m.ClockSample: m.ClockOwner, m.RandomState: m.RandomOwner, m.Draw: m.RandomOwner, m.NextFunction: m.RandomOwner, m.Command: m.EffectOwner, m.State: m.EffectOwner, m.Result: m.EffectOwner, m.DispatchFunction: m.EffectOwner, m.ReplayFunction: m.EffectOwner} {
		if x == (wire.ID{}) || owner == (wire.ID{}) || owners[x] != owner {
			return fmt.Errorf("controlled_effects.ownership")
		}
	}
	if e.Entities[m.ClockSample].Schema != id("9030") || e.Entities[m.RandomState].Schema != id("9030") || e.Entities[m.Draw].Schema != id("9030") || e.Entities[m.Command].Schema != id("9030") || e.Entities[m.State].Schema != id("9030") || e.Entities[m.Result].Schema != id("9030") {
		return fmt.Errorf("controlled_effects.type")
	}
	if uint64(len(m.Replay.Steps)) > z.MaximumEffects || m.DispatchFunction == m.ReplayFunction || !functionSignature(e, m.NextFunction, []wire.ID{m.RandomState}, m.Draw) || !functionSignature(e, m.DispatchFunction, []wire.ID{m.State, m.Command}, m.Result) || !functionSignature(e, m.ReplayFunction, []wire.ID{m.State, m.Command}, m.Result) || !ownedPureFunction(e, m.RandomOwner, m.NextFunction) || !ownedPureFunction(e, m.EffectOwner, m.ReplayFunction) {
		return fmt.Errorf("controlled_effects.function_or_effect")
	}
	// Dispatch is intentionally the controlled-effect entry point and may carry
	// the selected effect. Replay and random evolution must remain pure.
	if q, ok := e.Entities[m.DispatchFunction]; !ok || q.Schema != id("9011") {
		return fmt.Errorf("controlled_effects.dispatch_function")
	}
	if !exactDispatchEffect(e, m.DispatchFunction, m.ExternalEffect.EffectIdentity, m.ExternalEffect.CapabilityIdentity) {
		return fmt.Errorf("controlled_effects.dispatch_effect")
	}
	return nil
}

func zeroDigest(x [32]byte) bool { return x == [32]byte{} }

func exactDispatchEffect(e wire.Envelope, fn wire.ID, effectName, capabilityName string) bool {
	q := e.Entities[fn]
	seen, effects := map[wire.ID]bool{}, map[wire.ID]bool{}
	var walk func(wire.ID)
	walk = func(x wire.ID) {
		if seen[x] {
			return
		}
		seen[x] = true
		z, ok := e.Entities[x]
		if !ok {
			return
		}
		if z.Schema == id("90f1") {
			effects[x] = true
			return
		}
		for _, v := range z.Fields {
			collectEffectRefs(v, walk)
		}
	}
	walk(q.Fields[id("9113")].Reference)
	if len(effects) != 1 {
		return false
	}
	for x := range effects {
		invoke := e.Entities[x]
		effectRef, args := invoke.Fields[id("9f10")], invoke.Fields[id("9f11")]
		if effectRef.Tag != 6 || args.Tag != 7 || len(args.List) != 1 || args.List[0].Tag != 6 {
			return false
		}
		effect, ok := e.Entities[effectRef.Reference]
		if !ok || effect.Schema != id("15") || string(effect.Fields[id("150")].Bytes) != effectName {
			return false
		}
		capRef := effect.Fields[id("151")]
		capability, ok := e.Entities[capRef.Reference]
		if capRef.Tag != 6 || !ok || capability.Schema != id("16") || string(capability.Fields[id("160")].Bytes) != capabilityName {
			return false
		}
	}
	return true
}

func collectEffectRefs(v wire.Value, walk func(wire.ID)) {
	if v.Tag == 6 {
		walk(v.Reference)
	}
	for _, x := range v.List {
		collectEffectRefs(x, walk)
	}
	for _, x := range v.Record {
		collectEffectRefs(x, walk)
	}
}

func declarationOwners(e wire.Envelope) map[wire.ID]wire.ID {
	out, details := map[wire.ID]wire.ID{}, map[wire.ID]wire.ID{}
	for x, q := range e.Entities {
		if q.Schema == id("b021") {
			o := q.Fields[id("b210")].Reference
			details[x] = o
			for _, v := range q.Fields[id("b211")].List {
				if member, ok := e.Entities[v.Reference]; ok {
					out[member.Fields[id("b220")].Reference] = o
				}
			}
		}
	}
	for _, q := range e.Entities {
		if q.Schema == id("b028") {
			out[q.Fields[id("b280")].Reference] = details[q.Fields[id("b281")].Reference]
		}
	}
	return out
}

func functionSignature(e wire.Envelope, fn wire.ID, params []wire.ID, result wire.ID) bool {
	q, ok := e.Entities[fn]
	if !ok || q.Schema != id("9011") || q.Fields[id("9112")].Reference != result {
		return false
	}
	ps := q.Fields[id("9111")].List
	if len(ps) != len(params) {
		return false
	}
	for i, p := range ps {
		parameter, ok := e.Entities[p.Reference]
		if !ok || parameter.Schema != id("9012") || parameter.Fields[id("9121")].Reference != params[i] {
			return false
		}
	}
	return true
}

func ownedPureFunction(e wire.Envelope, owner, fn wire.ID) bool {
	owned := declarationOwners(e)[fn] == owner
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
	writeBytes(h, r.InitialState)
	for _, s := range r.Steps {
		_ = binary.Write(h, binary.BigEndian, s.CommandSequence)
		_ = binary.Write(h, binary.BigEndian, s.UnixMilliseconds)
		_ = binary.Write(h, binary.BigEndian, s.ClockSequence)
		_ = binary.Write(h, binary.BigEndian, s.RandomBefore)
		_ = binary.Write(h, binary.BigEndian, s.RandomAfter)
		_ = binary.Write(h, binary.BigEndian, s.RandomValue)
		_ = binary.Write(h, binary.BigEndian, s.RandomDrawOrdinal)
		if s.EffectValue {
			h.Write([]byte{1})
		} else {
			h.Write([]byte{0})
		}
		writeBytes(h, s.CanonicalCommand)
		h.Write(s.ResponseSHA256[:])
		h.Write(s.EventsSHA256[:])
		h.Write(s.StateBeforeSHA256[:])
		h.Write(s.StateAfterSHA256[:])
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
func writeBytes(w byteWriter, b []byte) {
	_ = binary.Write(w, binary.BigEndian, uint64(len(b)))
	_, _ = w.Write(b)
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
func signed(x int64) wire.Value     { return wire.Value{Tag: 4, Unsigned: uint64(x)} }
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
