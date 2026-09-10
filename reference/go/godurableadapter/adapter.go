// Package godurableadapter resolves neutral selections against authenticated Project-v9 entities.
package godurableadapter

import (
	"fmt"
	"seme.local/reference/durableinstance"
	"seme.local/reference/projectv9instance"
	"seme.local/reference/wire"
)

type Named struct{ Package, Name string }
type Selection struct {
	Identity, StateOwner, PortIdentity, PortOwner         string
	Version1, Version2, Validator1, Validator2, Migration Named
	MaximumPayloadBytes, MaximumKeyBytes                  uint64
}

func Resolve(in projectv9instance.Inputs, s Selection) (durableinstance.Model, error) {
	if err := projectv9instance.Validate(in); err != nil {
		return durableinstance.Model{}, fmt.Errorf("durable_adapter.project:%w", err)
	}
	e, err := wire.Decode(in.Composed)
	if err != nil {
		return durableinstance.Model{}, err
	}
	packages := map[string]wire.ID{}
	for x, q := range e.Entities {
		if q.Schema == id("b010") {
			n := string(q.Fields[id("b100")].Bytes)
			if n == "" || packages[n] != (wire.ID{}) {
				return durableinstance.Model{}, fmt.Errorf("durable_adapter.package")
			}
			packages[n] = x
		}
	}
	stateOwner, portOwner := packages[s.StateOwner], packages[s.PortOwner]
	if stateOwner == (wire.ID{}) || portOwner == (wire.ID{}) {
		return durableinstance.Model{}, fmt.Errorf("durable_adapter.owner")
	}
	owners := declarationOwners(e)
	resolve := func(n Named, schema string) (wire.ID, error) {
		owner := packages[n.Package]
		var found wire.ID
		for x, q := range e.Entities {
			if q.Schema == id(schema) && string(q.Fields[nameField(schema)].Bytes) == n.Name && owners[x] == owner {
				if found != (wire.ID{}) {
					return found, fmt.Errorf("durable_adapter.ambiguous:%s", n.Name)
				}
				found = x
			}
		}
		if found == (wire.ID{}) {
			return found, fmt.Errorf("durable_adapter.missing:%s", n.Name)
		}
		return found, nil
	}
	v1, err := resolve(s.Version1, "9030")
	if err != nil {
		return durableinstance.Model{}, err
	}
	v2, err := resolve(s.Version2, "9030")
	if err != nil {
		return durableinstance.Model{}, err
	}
	f1, err := resolve(s.Validator1, "9011")
	if err != nil {
		return durableinstance.Model{}, err
	}
	f2, err := resolve(s.Validator2, "9011")
	if err != nil {
		return durableinstance.Model{}, err
	}
	mig, err := resolve(s.Migration, "9011")
	if err != nil {
		return durableinstance.Model{}, err
	}
	integer, err := oneType(e, "9010", func(q wire.Entity) bool {
		return q.Fields[id("9100")].Unsigned == 64 && q.Fields[id("9101")].Tag == 2 && q.Fields[id("9101")].Unsigned == 1
	})
	if err != nil {
		return durableinstance.Model{}, err
	}
	str, err := oneType(e, "9040", func(wire.Entity) bool { return true })
	if err != nil {
		return durableinstance.Model{}, err
	}
	return durableinstance.Model{Identity: s.Identity, PortIdentity: s.PortIdentity, StateOwner: stateOwner, PortOwner: portOwner, Version1Type: v1, Version2Type: v2, ErrorType: integer, Validator1: f1, Validator2: f2, Migration: mig, KeyType: str, MaximumPayloadBytes: s.MaximumPayloadBytes, MaximumKeyBytes: s.MaximumKeyBytes}, nil
}
func declarationOwners(e wire.Envelope) map[wire.ID]wire.ID {
	o := map[wire.ID]wire.ID{}
	details := map[wire.ID]wire.ID{}
	for x, q := range e.Entities {
		if q.Schema == id("b021") {
			details[x] = q.Fields[id("b210")].Reference
			for _, v := range q.Fields[id("b211")].List {
				o[e.Entities[v.Reference].Fields[id("b220")].Reference] = q.Fields[id("b210")].Reference
			}
		}
	}
	for _, q := range e.Entities {
		if q.Schema == id("b028") {
			o[q.Fields[id("b280")].Reference] = details[q.Fields[id("b281")].Reference]
		}
	}
	return o
}
func nameField(schema string) wire.ID {
	if schema == "9030" {
		return id("9300")
	}
	return id("9110")
}
func oneType(e wire.Envelope, s string, accept func(wire.Entity) bool) (wire.ID, error) {
	var out wire.ID
	for x, q := range e.Entities {
		if q.Schema == id(s) && accept(q) {
			if out != (wire.ID{}) {
				return out, fmt.Errorf("durable_adapter.type_ambiguous")
			}
			out = x
		}
	}
	if out == (wire.ID{}) {
		return out, fmt.Errorf("durable_adapter.type_missing")
	}
	return out, nil
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
