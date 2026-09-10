// Package configurationmodule emits versioned Configuration and Initialization Contracts.
package configurationmodule

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
)

const ModuleID = "00000000000000000000000000004000"
const RevisionID = "00000000000000000000000000004001"
const RevisionV2ID = "00000000000000000000000000004005"

type Origin uint64

const (
	Explicit Origin = iota
	Default
	CapabilityInput
)

type Lifecycle uint64

const (
	Declared Lifecycle = iota
	Validated
	Initializing
	Initialized
	Failed
)

func ValidateOrigin(x uint64) error {
	if x > uint64(CapabilityInput) {
		return fmt.Errorf("configuration origin %d outside 0..2", x)
	}
	return nil
}
func ValidateLifecycle(x uint64) error {
	if x > uint64(Failed) {
		return fmt.Errorf("lifecycle state %d outside 0..4", x)
	}
	return nil
}

type field struct {
	id                 uint64
	name               string
	kind, schema, card uint64
}
type schema struct {
	id     uint64
	name   string
	fields []field
}

func f(id uint64, name string, kind, schema, card uint64) field {
	return field{id, name, kind, schema, card}
}
func declarations() []schema {
	return []schema{
		{0x4010, "ConfigurationGraph", []field{f(0x4100, "configuration_graph.fields", 5, 0x4011, 2), f(0x4101, "configuration_graph.values", 5, 0x4012, 2), f(0x4102, "configuration_graph.validations", 5, 0x4014, 2), f(0x4103, "configuration_graph.initializers", 5, 0x4015, 2), f(0x4104, "configuration_graph.transitions", 5, 0x4017, 2), f(0x4105, "configuration_graph.content_revision", 4, 0, 0)}},
		{0x4011, "ConfigurationField", []field{f(0x4110, "configuration_field.owner", 5, 0xb010, 0), f(0x4111, "configuration_field.key", 4, 0, 0), f(0x4112, "configuration_field.type", 5, 0, 0), f(0x4113, "configuration_field.default", 5, 0, 1), f(0x4114, "configuration_field.required", 1, 0, 0), f(0x4115, "configuration_field.origin", 5, 0xb026, 0)}},
		{0x4012, "ResolvedConfigurationValue", []field{f(0x4120, "resolved_configuration_value.field", 5, 0x4011, 0), f(0x4121, "resolved_configuration_value.value", 5, 0, 1), f(0x4122, "resolved_configuration_value.origin", 5, 0x4013, 0), f(0x4123, "resolved_configuration_value.capability", 5, 0x16, 1)}},
		{0x4013, "ResolutionOrigin", []field{f(0x4130, "resolution_origin.code", 2, 0, 0)}},
		{0x4014, "ValidationBinding", []field{f(0x4140, "validation_binding.field", 5, 0x4011, 1), f(0x4141, "validation_binding.validator", 5, 0x9011, 0), f(0x4142, "validation_binding.order", 2, 0, 0)}},
		{0x4015, "InitializerUnit", []field{f(0x4150, "initializer_unit.owner", 5, 0xb010, 0), f(0x4151, "initializer_unit.callable", 5, 0x9011, 0), f(0x4152, "initializer_unit.dependencies", 5, 0x4015, 2), f(0x4153, "initializer_unit.order", 2, 0, 0)}},
		{0x4016, "LifecycleState", []field{f(0x4160, "lifecycle_state.code", 2, 0, 0)}},
		{0x4017, "LifecycleTransition", []field{f(0x4170, "lifecycle_transition.initializer", 5, 0x4015, 0), f(0x4171, "lifecycle_transition.from", 5, 0x4016, 0), f(0x4172, "lifecycle_transition.to", 5, 0x4016, 0), f(0x4173, "lifecycle_transition.order", 2, 0, 0)}},
	}
}

func declarationsV2() []schema {
	return append(declarations(),
		schema{0x4018, "RuntimeInput", []field{f(0x4180, "runtime_input.identity", 4, 0, 0), f(0x4181, "runtime_input.type", 5, 0, 0), f(0x4182, "runtime_input.capability", 5, 0x16, 1)}},
		schema{0x4019, "ArgumentSourceKind", []field{f(0x4190, "argument_source_kind.code", 2, 0, 0)}},
		schema{0x401a, "ArgumentSource", []field{f(0x41a0, "argument_source.kind", 5, 0x4019, 0), f(0x41a1, "argument_source.derived_type", 5, 0, 0), f(0x41a2, "argument_source.field", 5, 0x4011, 1), f(0x41a3, "argument_source.runtime_input", 5, 0x4018, 1), f(0x41a4, "argument_source.predecessor", 5, 0x401b, 1), f(0x41a5, "argument_source.static_value", 5, 0, 1), f(0x41a6, "argument_source.record_assembly", 5, 0x401c, 1)}},
		schema{0x401b, "BoundInitializerUnit", []field{f(0x41b0, "bound_initializer_unit.base", 5, 0x4015, 0), f(0x41b1, "bound_initializer_unit.arguments", 5, 0x401e, 2)}},
		schema{0x401c, "RecordAssembly", []field{f(0x41c0, "record_assembly.record_type", 5, 0x9030, 0), f(0x41c1, "record_assembly.members", 5, 0x401d, 2)}},
		schema{0x401d, "RecordMemberBinding", []field{f(0x41d0, "record_member_binding.field", 5, 0x9031, 0), f(0x41d1, "record_member_binding.source", 5, 0x401a, 0)}},
		schema{0x401e, "InitializerArgument", []field{f(0x41e0, "initializer_argument.parameter", 5, 0x9012, 0), f(0x41e1, "initializer_argument.index", 2, 0, 0), f(0x41e2, "initializer_argument.source", 5, 0x401a, 0)}},
		schema{0x401f, "BoundConfigurationGraph", []field{f(0x41f0, "bound_configuration_graph.base", 5, 0x4010, 0), f(0x41f1, "bound_configuration_graph.runtime_inputs", 5, 0x4018, 2), f(0x41f2, "bound_configuration_graph.initializers", 5, 0x401b, 2), f(0x41f3, "bound_configuration_graph.content_revision", 4, 0, 0)}},
	)
}

func Emit(out io.Writer) error {
	return EmitVersion(out, 1)
}

func EmitVersion(out io.Writer, version int) error {
	if version < 1 || version > 2 {
		return fmt.Errorf("unsupported Configuration Contract version %d", version)
	}
	d, revision, parent := declarations(), RevisionID, "pc 0"
	if version == 2 {
		d, revision, parent = declarationsV2(), RevisionV2ID, "pc 1\n"+RevisionID
	}
	fields := []field{}
	for _, s := range d {
		fields = append(fields, s.fields...)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].id < fields[j].id })
	id := func(x uint64) string { return fmt.Sprintf("%032x", x) }
	text := func(x string) string { return hex.EncodeToString([]byte(x)) }
	fmt.Fprintf(out, "# Generated construction projection for Configuration and Initialization Contract v%d.\nve 1\nmo %s\nrv %s\n%s\nec %d\n\n", version, ModuleID, revision, parent, 4+len(d)+len(fields))
	fmt.Fprintf(out, "en %s %s %d 3\nfi %s by %s\nfi %s li 3\nrf %s\nrf %s\nrf %s\nfi %s li %d\n", ModuleID, id(0x12), version, id(0x120), text(fmt.Sprintf("configuration-contract-v%d", version)), id(0x121), id(0x4002), id(0x4003), id(0x4004), id(0x122), len(d)+len(fields))
	for _, s := range d {
		fmt.Fprintf(out, "rf %s\n", id(s.id))
	}
	for _, x := range fields {
		fmt.Fprintf(out, "rf %s\n", id(x.id))
	}
	fmt.Fprintf(out, "\nen %s %s 1 2\nfi %s rf %s\nfi %s by %s\n", id(0x4002), id(0x13), id(0x130), id(0xb000), id(0x131), id(0xb003))
	fmt.Fprintf(out, "\nen %s %s 1 2\nfi %s rf %s\nfi %s by %s\n", id(0x4003), id(0x13), id(0x130), id(0x9000), id(0x131), id(0x9023))
	fmt.Fprintf(out, "\nen %s %s 1 2\nfi %s rf %s\nfi %s by %s\n", id(0x4004), id(0x13), id(0x130), id(0x3000), id(0x131), id(0x3001))
	for _, s := range d {
		fmt.Fprintf(out, "\nen %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(s.id), id(0x10), id(0x100), text(s.name), id(0x101), len(s.fields))
		for _, x := range s.fields {
			fmt.Fprintf(out, "rf %s\n", id(x.id))
		}
	}
	for _, x := range fields {
		count, constraint := 1, ""
		if x.schema != 0 {
			count = 2
			constraint = fmt.Sprintf("fi %s rf %s\n", id(0x2001), id(x.schema))
		}
		fmt.Fprintf(out, "\nen %s %s 1 4\nfi %s by %s\nfi %s rc %d\nfi %s uu %d\n%sfi %s uu %d\nfi %s uu 1\n", id(x.id), id(0x11), id(0x110), text(x.name), id(0x111), count, id(0x2000), x.kind, constraint, id(0x112), x.card, id(0x113))
	}
	return nil
}
