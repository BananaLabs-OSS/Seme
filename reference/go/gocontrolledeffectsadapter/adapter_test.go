package gocontrolledeffectsadapter

import (
	"testing"

	"seme.local/reference/wire"
)

func TestResolveExplicitOwnedSelections(t *testing.T) {
	e, s := graph()
	m, err := resolveEnvelope(e, s)
	if err != nil {
		t.Fatal(err)
	}
	if m.NextFunction == (wire.ID{}) || m.DispatchFunction == (wire.ID{}) || m.ReplayFunction == (wire.ID{}) || m.DispatchFunction == m.ReplayFunction {
		t.Fatalf("%#v", m)
	}
	if m.Random.Algorithm != "state*48271+1" || m.Bounds.MaximumSteps != 256 || m.Replay.InitialSeed != 1 {
		t.Fatalf("%#v", m)
	}

	t.Run("missing", func(t *testing.T) {
		bad := s
		bad.Next.Name = "Missing"
		if _, e := resolveEnvelope(e, bad); e == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("unowned", func(t *testing.T) {
		bad := s
		bad.Next.Package = "example/streamservice"
		if _, e := resolveEnvelope(e, bad); e == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("duplicate", func(t *testing.T) {
		copy := cloneEnvelope(e)
		original := find(e, "example/controlled", "Next")
		forged := tid("7777")
		q := e.Entities[original]
		q.ID = forged
		copy.Entities[forged] = q
		addOwner(copy, "example/controlled", forged)
		if _, x := resolveEnvelope(copy, s); x == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("parameter-order", func(t *testing.T) {
		copy := cloneEnvelope(e)
		fn := find(e, "example/streamservice", "DispatchControlled")
		q := copy.Entities[fn]
		ps := append([]wire.Value(nil), q.Fields[id("9111")].List...)
		ps[0], ps[1] = ps[1], ps[0]
		q.Fields = cloneFields(q.Fields)
		q.Fields[id("9111")] = wire.Value{Tag: 7, List: ps}
		copy.Entities[fn] = q
		if _, x := resolveEnvelope(copy, s); x == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("tampered-policy", func(t *testing.T) {
		bad := s
		bad.OverflowPolicy = "mathematical-integer"
		if _, x := resolveEnvelope(e, bad); x == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("clone-ownership", func(t *testing.T) {
		before := m.ClockSample
		q := e.Entities[before]
		q.Fields = cloneFields(q.Fields)
		q.Fields[id("9300")] = wire.Value{Tag: 5, Bytes: []byte("Changed")}
		e.Entities[before] = q
		if m.ClockSample != before || m.Clock.Identity != "clock.injected.unix-milliseconds.v1" {
			t.Fatal("model aliased graph")
		}
	})
}

func graph() (wire.Envelope, Selection) {
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	cp, sp := tid("100"), tid("200")
	addPackage(e, cp, "example/controlled")
	addPackage(e, sp, "example/streamservice")
	clock, random, draw := record(e, "1001", "ClockSample"), record(e, "1002", "RandomState"), record(e, "1003", "Draw")
	command, state, result := record(e, "2001", "ControlledCommand"), record(e, "2002", "ControlledState"), record(e, "2003", "ControlledResult")
	next := function(e, "1010", "Next", []wire.ID{random}, draw)
	dispatch := function(e, "2010", "DispatchControlled", []wire.ID{state, command}, result)
	replay := function(e, "2011", "ReplayControlled", []wire.ID{state, command}, result)
	for _, x := range []wire.ID{clock, random, draw, next} {
		addOwnerID(e, cp, x)
	}
	for _, x := range []wire.ID{command, state, result, dispatch, replay} {
		addOwnerID(e, sp, x)
	}
	s := Selection{ClockIdentity: "clock.injected.unix-milliseconds.v1", ClockOwner: "example/controlled", ClockSample: Named{"example/controlled", "ClockSample"}, MonotonicPolicy: "nondecreasing", InjectionPolicy: "explicit-replayable-input", RandomIdentity: "random.seeded.lcg-48271-plus-1.v1", RandomOwner: "example/controlled", RandomState: Named{"example/controlled", "RandomState"}, Draw: Named{"example/controlled", "Draw"}, Next: Named{"example/controlled", "Next"}, Algorithm: "state*48271+1", OverflowPolicy: "signed-i64-modular", EffectIdentity: "observability.log", EffectOwner: "example/streamservice", Capability: "observability.log", Payload: "bool", DeliveryPolicy: "request-not-delivery", Command: Named{"example/streamservice", "ControlledCommand"}, State: Named{"example/streamservice", "ControlledState"}, Result: Named{"example/streamservice", "ControlledResult"}, Dispatch: Named{"example/streamservice", "DispatchControlled"}, Replay: Named{"example/streamservice", "ReplayControlled"}, DuplicatePolicy: "cached-no-new-effects", RejectionPolicy: "atomic-no-effects", Bounds: Bounds{256, 1, 257, 4102444800000, 1, 9223372036854775807, 256, 256}}
	return e, s
}

func addPackage(e wire.Envelope, x wire.ID, name string) {
	e.Entities[x] = wire.Entity{ID: x, Schema: id("b010"), Version: 1, Fields: map[wire.ID]wire.Value{id("b100"): {Tag: 5, Bytes: []byte(name)}}}
}
func record(e wire.Envelope, key, name string) wire.ID {
	x := tid(key)
	e.Entities[x] = wire.Entity{ID: x, Schema: id("9030"), Version: 1, Fields: map[wire.ID]wire.Value{id("9300"): {Tag: 5, Bytes: []byte(name)}}}
	return x
}
func function(e wire.Envelope, key, name string, types []wire.ID, result wire.ID) wire.ID {
	x := tid(key)
	ps := []wire.Value{}
	for i, t := range types {
		p := tid(key + string(rune('a'+i)))
		e.Entities[p] = wire.Entity{ID: p, Schema: id("9012"), Version: 1, Fields: map[wire.ID]wire.Value{id("9121"): {Tag: 6, Reference: t}}}
		ps = append(ps, wire.Value{Tag: 6, Reference: p})
	}
	e.Entities[x] = wire.Entity{ID: x, Schema: id("9011"), Version: 1, Fields: map[wire.ID]wire.Value{id("9110"): {Tag: 5, Bytes: []byte(name)}, id("9111"): {Tag: 7, List: ps}, id("9112"): {Tag: 6, Reference: result}}}
	return x
}
func addOwnerID(e wire.Envelope, owner, decl wire.ID) {
	detail := tid("d" + decl.String()[28:])
	member := tid("e" + decl.String()[28:])
	e.Entities[member] = wire.Entity{ID: member, Schema: id("b022"), Version: 1, Fields: map[wire.ID]wire.Value{id("b220"): {Tag: 6, Reference: decl}}}
	e.Entities[detail] = wire.Entity{ID: detail, Schema: id("b021"), Version: 1, Fields: map[wire.ID]wire.Value{id("b210"): {Tag: 6, Reference: owner}, id("b211"): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: member}}}}}
}
func addOwner(e wire.Envelope, name string, decl wire.ID) {
	for x, q := range e.Entities {
		if q.Schema == id("b010") && string(q.Fields[id("b100")].Bytes) == name {
			addOwnerID(e, x, decl)
			return
		}
	}
}
func find(e wire.Envelope, pkg, name string) wire.ID {
	owners := declarationOwners(e)
	var owner wire.ID
	for x, q := range e.Entities {
		if q.Schema == id("b010") && string(q.Fields[id("b100")].Bytes) == pkg {
			owner = x
		}
	}
	for x, q := range e.Entities {
		if owners[x] == owner && (declarationName(q, "9011") == name || declarationName(q, "9030") == name) {
			return x
		}
	}
	return wire.ID{}
}
func cloneEnvelope(e wire.Envelope) wire.Envelope {
	o := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for x, q := range e.Entities {
		q.Fields = cloneFields(q.Fields)
		o.Entities[x] = q
	}
	return o
}
func cloneFields(x map[wire.ID]wire.Value) map[wire.ID]wire.Value {
	o := map[wire.ID]wire.Value{}
	for k, v := range x {
		o[k] = v
	}
	return o
}
func tid(s string) wire.ID { return id(s) }
