// Package goprojectplacementadapter derives target requirements from an
// authenticated Go Project-v12 graph. It contains Go-profile knowledge; the
// Target-v1 resolver remains language and product neutral.
package goprojectplacementadapter

import (
	"bytes"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectv12instance"
	"seme.local/reference/targetplaninstance"
	"seme.local/reference/wire"
)

type Policy struct {
	Name     string
	Revision uint64
	// RuleNamespace identifies the language/profile adapter that supplied the
	// realization evidence. Empty preserves the historical Go UPB-10 identity
	// so existing authenticated Go bundles remain reproducible.
	RuleNamespace   string
	AllowedFidelity []targetplaninstance.Fidelity
}

// Derive validates the complete Project-v12 construction before inspecting
// canonical entities. Filenames, source spellings, and fixture identities do
// not participate in requirement discovery.
func Derive(project projectv12instance.Inputs, target contractcatalog.Contract, policy Policy) (targetplaninstance.Inputs, error) {
	if err := projectv12instance.Validate(project); err != nil {
		return targetplaninstance.Inputs{}, fmt.Errorf("go_placement.project_v12:%w", err)
	}
	if !target.Validated() || target.Pin() != (contractcatalog.Pin{Module: identity("c000"), Revision: identity("c001")}) || policy.Name == "" || policy.Revision == 0 {
		return targetplaninstance.Inputs{}, fmt.Errorf("go_placement.authority")
	}
	ruleNamespace := policy.RuleNamespace
	if ruleNamespace == "" {
		ruleNamespace = "go-upb10"
	}
	if !validRuleNamespace(ruleNamespace) {
		return targetplaninstance.Inputs{}, fmt.Errorf("go_placement.rule_namespace")
	}
	graph, err := wire.Decode(project.Composed)
	if err != nil {
		return targetplaninstance.Inputs{}, fmt.Errorf("go_placement.decode:%w", err)
	}
	root, err := one(graph, identity("e029"))
	if err != nil {
		return targetplaninstance.Inputs{}, err
	}
	controlled, err := one(graph, identity("13100"))
	if err != nil {
		return targetplaninstance.Inputs{}, err
	}

	model := targetplaninstance.Model{Root: root, Target: targetplaninstance.Target{Name: policy.Name, Revision: policy.Revision}, Allowed: append([]targetplaninstance.Fidelity(nil), policy.AllowedFidelity...)}
	types := map[wire.ID]targetplaninstance.Fidelity{
		identity("9011"): Exact, identity("b010"): Exact, identity("b012"): Exact,
		identity("4010"): Exact, identity("401f"): Exact, identity("6010"): Exact,
		identity("13103"): Exact, identity("13107"): Exact,
		identity("8015"): NativeIsland, identity("1010f"): NativeIsland,
		identity("13101"): NativeIsland, identity("13105"): NativeIsland,
	}
	// The snapshot itself is a requirement: this prevents an implementation
	// from planning a disconnected subset while calling it the project.
	if err = appendRequirement(&model, graph, root, Exact, nil, ruleNamespace); err != nil {
		return targetplaninstance.Inputs{}, err
	}
	for _, entity := range sortedEntities(graph) {
		fidelity, selected := types[entity.Schema]
		if !selected {
			continue
		}
		var boundary *targetplaninstance.Boundary
		if fidelity == NativeIsland {
			value, boundaryErr := nativeBoundary(graph, entity, controlled)
			if boundaryErr != nil {
				return targetplaninstance.Inputs{}, boundaryErr
			}
			boundary = &value
		}
		if err = appendRequirement(&model, graph, entity.ID, fidelity, boundary, ruleNamespace); err != nil {
			return targetplaninstance.Inputs{}, err
		}
	}
	if len(model.Requirements) < 2 || len(model.Boundaries) < 4 {
		return targetplaninstance.Inputs{}, fmt.Errorf("go_placement.incomplete_profile")
	}
	contracts := []contractcatalog.Contract{
		project.Contracts.Foundation(), project.Contracts.Execution(), project.Contracts.Package(),
		project.Contracts.Dependency(), project.Contracts.Configuration(), project.Contracts.Resource(),
		project.Contracts.DurableState(), project.Contracts.Presentation(), project.Contracts.OrderedTransport(),
		project.Contracts.ControlledEffects(), project.Contracts.Project(), target,
	}
	return targetplaninstance.Inputs{TargetContract: target, SemanticContracts: contracts, Authority: bytes.Clone(project.Composed), Model: model}, nil
}

const (
	Exact        = targetplaninstance.Exact
	NativeIsland = targetplaninstance.NativeIsland
)

func appendRequirement(model *targetplaninstance.Model, graph wire.Envelope, construct wire.ID, fidelity targetplaninstance.Fidelity, boundary *targetplaninstance.Boundary, ruleNamespace string) error {
	entity, exists := graph.Entities[construct]
	if !exists {
		return fmt.Errorf("go_placement.construct:%s", construct)
	}
	properties := directReferences(graph, entity)
	name := entity.Schema.String() + ":" + construct.String()
	model.Requirements = append(model.Requirements, targetplaninstance.Requirement{Identity: name, Construct: construct, MinimumRevision: uint64(entity.Version), Properties: properties})
	rule := targetplaninstance.Rule{Identity: ruleNamespace + ":" + name, Construct: construct, MaximumRevision: uint64(entity.Version), PreservedProperties: properties, Fidelity: fidelity, Evidence: evidence(construct, properties)}
	if fidelity != Exact {
		provider := boundary.Provider
		rule.Dependency = &provider
		boundary.Requirement = name
		model.Boundaries = append(model.Boundaries, *boundary)
	}
	model.Target.Rules = append(model.Target.Rules, rule)
	return nil
}

func validRuleNamespace(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for index, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (index > 0 && (r == '-' || r == '.')) {
			continue
		}
		return false
	}
	return true
}

func nativeBoundary(graph wire.Envelope, entity wire.Entity, controlled wire.ID) (targetplaninstance.Boundary, error) {
	field := func(key string) (wire.ID, bool) {
		value, exists := entity.Fields[identity(key)]
		if !exists || value.Tag != 6 {
			return wire.ID{}, false
		}
		_, present := graph.Entities[value.Reference]
		return value.Reference, present
	}
	boundary := targetplaninstance.Boundary{Provider: entity.ID}
	switch entity.Schema {
	case identity("8015"):
		boundary.Consumer, _ = field("8151")
		boundary.Interface, _ = field("8152")
		boundary.Transport = entity.ID
	case identity("1010f"):
		boundary.Consumer, _ = field("11101")
		boundary.Interface, _ = field("11106")
		boundary.Transport = boundary.Interface
	case identity("13101"):
		boundary.Consumer = controlled
		boundary.Interface = entity.ID
		boundary.Transport = entity.ID
	case identity("13105"):
		boundary.Consumer = controlled
		boundary.Interface, _ = field("13252")
		boundary.Transport = entity.ID
	default:
		return boundary, fmt.Errorf("go_placement.native_schema:%s", entity.Schema)
	}
	for _, value := range []wire.ID{boundary.Provider, boundary.Consumer, boundary.Interface, boundary.Transport} {
		if _, exists := graph.Entities[value]; !exists {
			return boundary, fmt.Errorf("go_placement.boundary:%s:%s:%s", entity.Schema, entity.ID, value)
		}
	}
	return boundary, nil
}

func directReferences(graph wire.Envelope, entity wire.Entity) []wire.ID {
	seen := map[wire.ID]bool{}
	var collect func(wire.Value)
	collect = func(value wire.Value) {
		if value.Tag == 6 {
			if _, exists := graph.Entities[value.Reference]; exists {
				seen[value.Reference] = true
			}
		}
		for _, item := range value.List {
			collect(item)
		}
		for _, item := range value.Record {
			collect(item)
		}
	}
	for _, value := range entity.Fields {
		collect(value)
	}
	result := make([]wire.ID, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return bytes.Compare(result[i][:], result[j][:]) < 0 })
	return result
}

func evidence(construct wire.ID, properties []wire.ID) []wire.ID {
	result := append([]wire.ID{construct}, properties...)
	sort.Slice(result, func(i, j int) bool { return bytes.Compare(result[i][:], result[j][:]) < 0 })
	return result
}

func sortedEntities(graph wire.Envelope) []wire.Entity {
	result := make([]wire.Entity, 0, len(graph.Entities))
	for _, entity := range graph.Entities {
		result = append(result, entity)
	}
	sort.Slice(result, func(i, j int) bool { return bytes.Compare(result[i].ID[:], result[j].ID[:]) < 0 })
	return result
}

func one(graph wire.Envelope, schema wire.ID) (wire.ID, error) {
	var result wire.ID
	for identity, entity := range graph.Entities {
		if entity.Schema != schema {
			continue
		}
		if result != (wire.ID{}) {
			return wire.ID{}, fmt.Errorf("go_placement.cardinality:%s", schema)
		}
		result = identity
	}
	if result == (wire.ID{}) {
		return wire.ID{}, fmt.Errorf("go_placement.cardinality:%s", schema)
	}
	return result, nil
}

func identity(value string) wire.ID {
	for len(value) < 32 {
		value = "0" + value
	}
	result, err := wire.ParseID(value)
	if err != nil {
		panic(err)
	}
	return result
}
