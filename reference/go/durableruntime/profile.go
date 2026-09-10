package durableruntime

import (
	"fmt"

	"seme.local/reference/durableinstance"
	"seme.local/reference/wire"
)

// AuthenticatedProfile derives the executable boundary profile only after the
// exact Durable-v1 contract, Project-v9 snapshot, model, and artifact agree.
// Callers cannot substitute operation identities or weaken bounds afterward
// without producing a different Profile value that fails their evidence bind.
func AuthenticatedProfile(in durableinstance.Inputs) (Profile, error) {
	if err := durableinstance.Validate(in); err != nil {
		return Profile{}, fmt.Errorf("durable_runtime.instance:%w", err)
	}
	e, err := wire.Decode(in.Artifact)
	if err != nil {
		return Profile{}, fmt.Errorf("durable_runtime.artifact:%w", err)
	}
	plan, err := exactlyOne(e, "8010")
	if err != nil {
		return Profile{}, err
	}
	families := plan.Fields[rid("8100")]
	ports := plan.Fields[rid("8101")]
	if families.Tag != 7 || len(families.List) != 1 || ports.Tag != 7 || len(ports.List) != 1 || families.List[0].Tag != 6 || ports.List[0].Tag != 6 {
		return Profile{}, fmt.Errorf("durable_runtime.plan")
	}
	family, fok := e.Entities[families.List[0].Reference]
	port, pok := e.Entities[ports.List[0].Reference]
	if !fok || family.Schema != rid("8011") || !pok || port.Schema != rid("8015") || port.Fields[rid("8152")].Reference != family.ID {
		return Profile{}, fmt.Errorf("durable_runtime.plan_shape")
	}
	current, ok := e.Entities[family.Fields[rid("8112")].Reference]
	if !ok || current.Schema != rid("8012") {
		return Profile{}, fmt.Errorf("durable_runtime.current")
	}
	loadEffect, loadCapability, err := operation(e, port, "8155", "8153")
	if err != nil {
		return Profile{}, err
	}
	casEffect, casCapability, err := operation(e, port, "8156", "8154")
	if err != nil {
		return Profile{}, err
	}
	policy, err := contractFieldName(in.Contracts.DurableState().Envelope(), "8016", "8167")
	if err != nil || policy != "opaque-thread-only" {
		return Profile{}, fmt.Errorf("durable_runtime.token_policy")
	}
	p := Profile{
		FamilyIdentity: string(family.Fields[rid("8110")].Bytes),
		CurrentVersion: current.Fields[rid("8121")].Unsigned,
		CodecIdentity:  string(port.Fields[rid("8159")].Bytes), TokenPolicy: policy,
		MaximumKeyBytes: port.Fields[rid("815a")].Unsigned, MaximumPayloadBytes: port.Fields[rid("8158")].Unsigned,
		Load:            Operation{Identity: loadEffect, Capability: loadCapability, Sequence: 0},
		CompareExchange: Operation{Identity: casEffect, Capability: casCapability, Sequence: 1},
	}
	p = sealProfile(p)
	if !validProfile(p) {
		return Profile{}, fmt.Errorf("durable_runtime.profile")
	}
	return p, nil
}

func operation(e wire.Envelope, port wire.Entity, effectField, capabilityField string) (string, string, error) {
	effect, eok := e.Entities[port.Fields[rid(effectField)].Reference]
	capability, cok := e.Entities[port.Fields[rid(capabilityField)].Reference]
	if !eok || effect.Schema != rid("15") || !cok || capability.Schema != rid("16") || effect.Fields[rid("151")].Reference != capability.ID {
		return "", "", fmt.Errorf("durable_runtime.operation")
	}
	return string(effect.Fields[rid("150")].Bytes), string(capability.Fields[rid("160")].Bytes), nil
}
func contractFieldName(e wire.Envelope, schema, field string) (string, error) {
	q, ok := e.Entities[rid(schema)]
	if !ok || q.Schema != rid("10") {
		return "", fmt.Errorf("schema")
	}
	for _, v := range q.Fields[rid("101")].List {
		if v.Tag == 6 && v.Reference == rid(field) {
			f := e.Entities[v.Reference]
			if n := f.Fields[rid("110")]; f.Schema == rid("11") && n.Tag == 5 {
				return string(n.Bytes), nil
			}
		}
	}
	return "", fmt.Errorf("field")
}
func exactlyOne(e wire.Envelope, schema string) (wire.Entity, error) {
	var out wire.Entity
	n := 0
	for _, q := range e.Entities {
		if q.Schema == rid(schema) {
			out = q
			n++
		}
	}
	if n != 1 {
		return wire.Entity{}, fmt.Errorf("durable_runtime.count:%s", schema)
	}
	return out, nil
}
func rid(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, err := wire.ParseID(s)
	if err != nil {
		panic(err)
	}
	return x
}
