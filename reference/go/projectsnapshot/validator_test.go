package projectsnapshot

import (
	"seme.local/reference/wire"
	"testing"
)

func validEnvelope() wire.Envelope {
	identity, pkg, program, fn, snap := id("11111111111111111111111111111111"), id("22222222222222222222222222222222"), id("44444444444444444444444444444444"), id("77777777777777777777777777777777"), id("55555555555555555555555555555555")
	moduleID, importID := id("99999999999999999999999999999999"), id("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	e := wire.Envelope{Module: moduleID, Entities: map[wire.ID]wire.Entity{}}
	e.Entities[moduleID] = wire.Entity{ID: moduleID, Schema: moduleSchema, Version: 1, Fields: map[wire.ID]wire.Value{fImports: {Tag: 7, List: []wire.Value{{Tag: 6, Reference: importID}}}}}
	e.Entities[importID] = wire.Entity{ID: importID, Schema: importSchema, Version: 1, Fields: map[wire.ID]wire.Value{fImportModule: {Tag: 6, Reference: projectModule}, fImportRevision: {Tag: 5, Bytes: projectRevision[:]}}}
	e.Entities[identity] = wire.Entity{ID: identity, Schema: identitySchema, Version: 1, Fields: map[wire.ID]wire.Value{id("0000000000000000000000000000e100"): {Tag: 5, Bytes: []byte("x")}}}
	e.Entities[pkg] = wire.Entity{ID: pkg, Schema: packageSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("0000000000000000000000000000b100"): {Tag: 5, Bytes: []byte("p")}}}
	e.Entities[fn] = wire.Entity{ID: fn, Schema: id("00000000000000000000000000009011"), Version: 1, Fields: map[wire.ID]wire.Value{id("00000000000000000000000000009110"): {Tag: 5, Bytes: []byte("f")}, id("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"): {Tag: 8, Record: map[wire.ID]wire.Value{id("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"): {Tag: 5, Bytes: []byte("record")}}}, id("cccccccccccccccccccccccccccccccc"): {Tag: 9, Hole: id("dddddddddddddddddddddddddddddddd")}}}
	e.Entities[program] = wire.Entity{ID: program, Schema: programSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("00000000000000000000000000009150"): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: fn}}}, id("00000000000000000000000000009151"): {Tag: 6, Reference: fn}}}
	revision, _ := Revision(e, identity, []wire.ID{pkg}, pkg, program)
	e.Entities[snap] = wire.Entity{ID: snap, Schema: snapshotSchema, Version: 1, Fields: map[wire.ID]wire.Value{fIdentity: {Tag: 6, Reference: identity}, fRevision: {Tag: 5, Bytes: revision}, fPackages: {Tag: 7, List: []wire.Value{{Tag: 6, Reference: pkg}}}, fRoot: {Tag: 6, Reference: pkg}, fProgram: {Tag: 6, Reference: program}}}
	return e
}
func TestValidateEnvelopeAdversaries(t *testing.T) {
	if err := ValidateEnvelope(validEnvelope()); err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*wire.Envelope){
		"kind-swap": func(e *wire.Envelope) {
			for k, x := range e.Entities {
				if x.Schema == snapshotSchema {
					v := x.Fields[fRevision]
					v.Tag = 6
					x.Fields[fRevision] = v
					e.Entities[k] = x
				}
			}
		},
		"package-content": func(e *wire.Envelope) {
			for k, x := range e.Entities {
				if x.Schema == packageSchema {
					for f, v := range x.Fields {
						v.Bytes = []byte("changed")
						x.Fields[f] = v
					}
					e.Entities[k] = x
				}
			}
		},
		"function-content": func(e *wire.Envelope) {
			for k, x := range e.Entities {
				if x.Schema == id("00000000000000000000000000009011") {
					for f, v := range x.Fields {
						v.Bytes = []byte("changed")
						x.Fields[f] = v
					}
					e.Entities[k] = x
				}
			}
		},
		"record-content": func(e *wire.Envelope) {
			for k, x := range e.Entities {
				if x.Schema == id("00000000000000000000000000009011") {
					v := x.Fields[id("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")]
					for f, m := range v.Record {
						m.Bytes = []byte("changed")
						v.Record[f] = m
					}
					x.Fields[id("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")] = v
					e.Entities[k] = x
				}
			}
		},
		"hole-identity": func(e *wire.Envelope) {
			for k, x := range e.Entities {
				if x.Schema == id("00000000000000000000000000009011") {
					v := x.Fields[id("cccccccccccccccccccccccccccccccc")]
					v.Hole = id("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
					x.Fields[id("cccccccccccccccccccccccccccccccc")] = v
					e.Entities[k] = x
				}
			}
		},
		"missing-reference": func(e *wire.Envelope) {
			for k, x := range e.Entities {
				if x.Schema == programSchema {
					x.Fields[id("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")] = wire.Value{Tag: 6, Reference: id("ffffffffffffffffffffffffffffffff")}
					e.Entities[k] = x
				}
			}
		},
		"list-item-kind": func(e *wire.Envelope) {
			for k, x := range e.Entities {
				if x.Schema == snapshotSchema {
					v := x.Fields[fPackages]
					v.List[0].Tag = 5
					x.Fields[fPackages] = v
					e.Entities[k] = x
				}
			}
		},
		"unpinned": func(e *wire.Envelope) {
			for k, x := range e.Entities {
				if x.Schema == importSchema {
					delete(e.Entities, k)
				}
			}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			e := validEnvelope()
			mutate(&e)
			if ValidateEnvelope(e) == nil {
				t.Fatal("accepted malformed snapshot")
			}
		})
	}
}
func TestValidateRejectsMalformedWire(t *testing.T) {
	for _, source := range [][]byte{nil, []byte("SEMEK1\r\n"), []byte("not-wire")} {
		if Validate(source) == nil {
			t.Fatal("accepted malformed header/count")
		}
	}
}
