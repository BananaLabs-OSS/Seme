// Package goorderedtransportadapter resolves ordinary Go declaration names
// against an authenticated Project-v10 graph. Names select existing semantic
// identities; this adapter never creates language-specific transport meaning.
package goorderedtransportadapter

import (
	"fmt"

	"seme.local/reference/orderedtransportinstance"
	"seme.local/reference/projectv10instance"
	"seme.local/reference/wire"
)

type Named struct{ Package, Name string }
type OwnedIdentity struct{ Identity, Package string }
type KindSelection struct {
	Identity string
	Owner    string
	Payload  Named
}
type Selection struct {
	Streams      []OwnedIdentity
	CommandKinds []KindSelection
	EventKinds   []KindSelection
	Ports        []OwnedIdentity
	Dispatch     Named
	Replay       Named
}

func Resolve(project projectv10instance.Inputs, selection Selection) (orderedtransportinstance.Model, error) {
	if err := projectv10instance.Validate(project); err != nil {
		return orderedtransportinstance.Model{}, fmt.Errorf("go_ordered_transport.project:%w", err)
	}
	e, err := wire.Decode(project.Composed)
	if err != nil {
		return orderedtransportinstance.Model{}, err
	}
	packages := map[string]wire.ID{}
	for x, q := range e.Entities {
		if q.Schema != id("b010") {
			continue
		}
		name := string(q.Fields[id("b100")].Bytes)
		if name == "" || packages[name] != (wire.ID{}) {
			return orderedtransportinstance.Model{}, fmt.Errorf("go_ordered_transport.package")
		}
		packages[name] = x
	}
	owners := declarationOwners(e)
	resolveOwner := func(name string) (wire.ID, error) {
		x := packages[name]
		if x == (wire.ID{}) {
			return x, fmt.Errorf("go_ordered_transport.owner:%s", name)
		}
		return x, nil
	}
	resolveNamed := func(n Named, schema string) (wire.ID, error) {
		owner, er := resolveOwner(n.Package)
		if er != nil {
			return wire.ID{}, er
		}
		var found wire.ID
		for x, q := range e.Entities {
			if q.Schema == id(schema) && declarationName(q, schema) == n.Name && owners[x] == owner {
				if found != (wire.ID{}) {
					return found, fmt.Errorf("go_ordered_transport.ambiguous:%s:%s", n.Package, n.Name)
				}
				found = x
			}
		}
		if found == (wire.ID{}) {
			return found, fmt.Errorf("go_ordered_transport.missing:%s:%s", n.Package, n.Name)
		}
		return found, nil
	}
	var model orderedtransportinstance.Model
	for _, s := range selection.Streams {
		owner, er := resolveOwner(s.Package)
		if er != nil {
			return model, er
		}
		model.Streams = append(model.Streams, orderedtransportinstance.Stream{Identity: s.Identity, Owner: owner})
	}
	resolveKinds := func(items []KindSelection) ([]orderedtransportinstance.Kind, error) {
		out := make([]orderedtransportinstance.Kind, 0, len(items))
		for _, item := range items {
			owner, er := resolveOwner(item.Owner)
			if er != nil {
				return nil, er
			}
			payload, er := resolveNamed(item.Payload, "9030")
			if er != nil {
				return nil, er
			}
			out = append(out, orderedtransportinstance.Kind{Identity: item.Identity, Owner: owner, PayloadType: payload})
		}
		return out, nil
	}
	model.CommandKinds, err = resolveKinds(selection.CommandKinds)
	if err != nil {
		return model, err
	}
	model.EventKinds, err = resolveKinds(selection.EventKinds)
	if err != nil {
		return model, err
	}
	for _, p := range selection.Ports {
		owner, er := resolveOwner(p.Package)
		if er != nil {
			return model, er
		}
		model.Ports = append(model.Ports, orderedtransportinstance.Port{Identity: p.Identity, Owner: owner})
	}
	model.DispatchFunction, err = resolveNamed(selection.Dispatch, "9011")
	if err != nil {
		return model, err
	}
	model.ReplayFunction, err = resolveNamed(selection.Replay, "9011")
	if err != nil {
		return model, err
	}
	return model, nil
}

func declarationOwners(e wire.Envelope) map[wire.ID]wire.ID {
	out := map[wire.ID]wire.ID{}
	details := map[wire.ID]wire.ID{}
	for x, q := range e.Entities {
		if q.Schema == id("b021") {
			owner := q.Fields[id("b210")].Reference
			details[x] = owner
			for _, v := range q.Fields[id("b211")].List {
				member := e.Entities[v.Reference]
				out[member.Fields[id("b220")].Reference] = owner
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
func declarationName(q wire.Entity, schema string) string {
	if schema == "9030" {
		return string(q.Fields[id("9300")].Bytes)
	}
	return string(q.Fields[id("9110")].Bytes)
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
