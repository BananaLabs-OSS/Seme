// Package targetplaninstance resolves and emits language-neutral Target v1
// execution plans. Language providers derive requirements; this package owns
// only canonical resolution, fidelity, boundary, and artifact invariants.
package targetplaninstance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/wire"
)

type Fidelity uint64

const (
	Exact Fidelity = iota
	Refined
	Adapted
	Emulated
	EmbeddedRuntime
	NativeIsland
	Impossible
)

type Requirement struct {
	Identity        string
	Construct       wire.ID
	MinimumRevision uint64
	Properties      []wire.ID
}

type Rule struct {
	Identity            string
	Construct           wire.ID
	MaximumRevision     uint64
	PreservedProperties []wire.ID
	Fidelity            Fidelity
	Dependency          *wire.ID
	Evidence            []wire.ID
}

type Target struct {
	Name     string
	Revision uint64
	Rules    []Rule
}

type Boundary struct {
	Requirement string
	Provider    wire.ID
	Consumer    wire.ID
	Interface   wire.ID
	Transport   wire.ID
}

type Model struct {
	Root         wire.ID
	Target       Target
	Requirements []Requirement
	Allowed      []Fidelity
	Boundaries   []Boundary
}

type Inputs struct {
	TargetContract contractcatalog.Contract
	// SemanticContracts closes imported identities used by Authority. Each
	// contract must already be independently digest-authenticated.
	SemanticContracts []contractcatalog.Contract
	Authority         []byte
	Model             Model
	Artifact          []byte
}

func Emit(in Inputs) ([]byte, error) { return emit(in) }

func Validate(in Inputs) error {
	want, err := emit(Inputs{TargetContract: in.TargetContract, SemanticContracts: in.SemanticContracts, Authority: in.Authority, Model: in.Model})
	if err != nil {
		return err
	}
	if !bytes.Equal(want, in.Artifact) {
		return fmt.Errorf("target_plan.artifact")
	}
	return nil
}

func emit(in Inputs) ([]byte, error) {
	if !in.TargetContract.Validated() || in.TargetContract.Pin() != (contractcatalog.Pin{Module: id("c000"), Revision: id("c001")}) {
		return nil, fmt.Errorf("target_plan.contract")
	}
	base, err := wire.Decode(in.Authority)
	if err != nil {
		return nil, fmt.Errorf("target_plan.authority:%w", err)
	}
	canonical, err := wire.Encode(base)
	if err != nil || !bytes.Equal(canonical, in.Authority) {
		return nil, fmt.Errorf("target_plan.authority_noncanonical")
	}
	if _, ok := base.Entities[in.Model.Root]; !ok {
		return nil, fmt.Errorf("target_plan.root")
	}
	if in.Model.Target.Name == "" || in.Model.Target.Revision == 0 || len(in.Model.Requirements) == 0 {
		return nil, fmt.Errorf("target_plan.model")
	}
	out := cloneEnvelope(base)
	contracts, err := normalizedContracts(in.TargetContract, in.SemanticContracts)
	if err != nil {
		return nil, err
	}
	for _, contract := range contracts {
		for key, value := range contract.Envelope().Entities {
			if prior, ok := out.Entities[key]; ok && !sameEntity(prior, value) {
				return nil, fmt.Errorf("target_plan.contract_collision:%s", key)
			}
			out.Entities[key] = value
		}
	}
	allowed, err := allowedSet(in.Model.Allowed)
	if err != nil {
		return nil, err
	}
	requirements, err := normalizeRequirements(out.Entities, in.Model.Requirements)
	if err != nil {
		return nil, err
	}
	rules, err := normalizeRules(out.Entities, in.Model.Target.Rules)
	if err != nil {
		return nil, err
	}
	boundaries, err := normalizeBoundaries(out.Entities, in.Model.Root, in.Model.Boundaries)
	if err != nil {
		return nil, err
	}
	if err = validateCandidateBoundaries(requirements, rules, boundaries); err != nil {
		return nil, err
	}

	seed := modelDigest(in.Authority, contracts, in.Model)
	stable := func(parts ...string) wire.ID { return stableID(seed, parts...) }

	targetID := stable("target")
	ruleIDs := map[string]wire.ID{}
	for _, rule := range rules {
		ruleIDs[rule.Identity] = stable("rule", rule.Identity)
	}
	requirementIDs := map[string]wire.ID{}
	for _, requirement := range requirements {
		requirementIDs[requirement.Identity] = stable("requirement", requirement.Identity)
	}
	boundaryByRequirement := map[string]Boundary{}
	for _, boundary := range boundaries {
		boundaryByRequirement[boundary.Requirement] = boundary
	}

	for _, rule := range rules {
		fields := map[wire.ID]wire.Value{
			id("c120"): ref(rule.Construct), id("c121"): unsigned(rule.MaximumRevision),
			id("c122"): refs(rule.PreservedProperties), id("c123"): unsigned(uint64(rule.Fidelity)),
			id("c125"): refs(rule.Evidence),
		}
		if rule.Dependency != nil {
			fields[id("c124")] = ref(*rule.Dependency)
		}
		rid := ruleIDs[rule.Identity]
		out.Entities[rid] = entity(rid, "c012", fields)
	}
	ruleRefs := make([]wire.ID, 0, len(rules))
	for _, rule := range rules {
		ruleRefs = append(ruleRefs, ruleIDs[rule.Identity])
	}
	out.Entities[targetID] = entity(targetID, "c010", map[wire.ID]wire.Value{id("c100"): blob([]byte(in.Model.Target.Name)), id("c101"): unsigned(in.Model.Target.Revision), id("c102"): refs(ruleRefs)})

	resolutionIDs, boundaryIDs := make([]wire.ID, 0, len(requirements)), []wire.ID{}
	executable := true
	diagnosticRuleID := stable("diagnostic-rule", "target.no-permitted-realization")
	diagnosticRuleEmitted := false
	for _, requirement := range requirements {
		qid := requirementIDs[requirement.Identity]
		out.Entities[qid] = entity(qid, "c011", map[wire.ID]wire.Value{id("c110"): ref(requirement.Construct), id("c111"): unsigned(requirement.MinimumRevision), id("c112"): refs(requirement.Properties)})
		selected, ok := selectRule(requirement, rules, allowed)
		resolutionID := stable("resolution", requirement.Identity)
		fields := map[wire.ID]wire.Value{id("c130"): ref(qid), id("c131"): ref(targetID), id("c134"): refs(nil), id("c135"): refs(nil)}
		if !ok {
			executable = false
			if !diagnosticRuleEmitted {
				out.Entities[diagnosticRuleID] = entity(diagnosticRuleID, "19", map[wire.ID]wire.Value{id("190"): blob([]byte("target.no_permitted_realization")), id("191"): records(nil)})
				diagnosticRuleEmitted = true
			}
			diagnosticID := stable("diagnostic", requirement.Identity)
			out.Entities[diagnosticID] = entity(diagnosticID, "1a", map[wire.ID]wire.Value{
				id("1a0"): ref(diagnosticRuleID), id("1a1"): blob(base.Revision[:]), id("1a2"): blob(requirement.Construct[:]),
				id("1a3"): records(nil), id("1a4"): list(nil), id("1a5"): unsigned(2),
			})
			fields[id("c133")] = unsigned(uint64(Impossible))
			fields[id("c134")] = refs(requirement.Properties)
			fields[id("c135")] = refs([]wire.ID{diagnosticID})
		} else {
			fields[id("c132")] = ref(ruleIDs[selected.Identity])
			fields[id("c133")] = unsigned(uint64(selected.Fidelity))
			if selected.Fidelity != Exact && selected.Fidelity != Refined {
				boundary, exists := boundaryByRequirement[requirement.Identity]
				if !exists {
					return nil, fmt.Errorf("target_plan.boundary_missing:%s", requirement.Identity)
				}
				bid := stable("boundary", requirement.Identity)
				out.Entities[bid] = entity(bid, "c015", map[wire.ID]wire.Value{id("c150"): ref(boundary.Provider), id("c151"): ref(boundary.Consumer), id("c152"): ref(boundary.Interface), id("c153"): ref(boundary.Transport)})
				boundaryIDs = append(boundaryIDs, bid)
			}
		}
		out.Entities[resolutionID] = entity(resolutionID, "c013", fields)
		resolutionIDs = append(resolutionIDs, resolutionID)
	}
	planID := stable("plan")
	out.Entities[planID] = entity(planID, "c014", map[wire.ID]wire.Value{id("c140"): ref(in.Model.Root), id("c141"): ref(targetID), id("c142"): refs(resolutionIDs), id("c143"): refs(boundaryIDs), id("c144"): boolean(executable)})
	moduleID, importAuthorityID := stable("module"), stable("import", "authority")
	out.Entities[importAuthorityID] = entity(importAuthorityID, "13", map[wire.ID]wire.Value{id("130"): ref(base.Module), id("131"): blob(base.Revision[:])})
	importIDs := []wire.ID{importAuthorityID}
	for _, contract := range contracts {
		pin := contract.Pin()
		importID := stable("import", pin.Module.String(), pin.Revision.String())
		out.Entities[importID] = entity(importID, "13", map[wire.ID]wire.Value{id("130"): ref(pin.Module), id("131"): blob(pin.Revision[:])})
		importIDs = append(importIDs, importID)
	}
	out.Module = moduleID
	out.Parents = nil
	out.Entities[moduleID] = entity(moduleID, "12", map[wire.ID]wire.Value{id("120"): blob([]byte("target-execution-plan-v1")), id("121"): refs(sortedIDs(importIDs)), id("122"): refs([]wire.ID{planID})})
	out.Revision = wire.ID{}
	encoded, err := wire.Encode(out)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(append([]byte("seme.target-plan.artifact.v1\x00"), encoded...))
	copy(out.Revision[:], h[:16])
	return wire.Encode(out)
}

func allowedSet(values []Fidelity) (map[Fidelity]bool, error) {
	out := map[Fidelity]bool{}
	for _, value := range values {
		if value >= Impossible || out[value] {
			return nil, fmt.Errorf("target_plan.policy")
		}
		out[value] = true
	}
	return out, nil
}

func normalizedContracts(target contractcatalog.Contract, additional []contractcatalog.Contract) ([]contractcatalog.Contract, error) {
	values := append([]contractcatalog.Contract{target}, additional...)
	seen := map[contractcatalog.Pin][sha256.Size]byte{}
	result := make([]contractcatalog.Contract, 0, len(values))
	for _, value := range values {
		if !value.Validated() {
			return nil, fmt.Errorf("target_plan.semantic_contract")
		}
		pin, digest := value.Pin(), value.Digest()
		if prior, exists := seen[pin]; exists {
			if prior != digest {
				return nil, fmt.Errorf("target_plan.semantic_contract_collision")
			}
			continue
		}
		seen[pin] = digest
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i].Pin(), result[j].Pin()
		if a.Module != b.Module {
			return bytes.Compare(a.Module[:], b.Module[:]) < 0
		}
		return bytes.Compare(a.Revision[:], b.Revision[:]) < 0
	})
	return result, nil
}

func normalizeRequirements(entities map[wire.ID]wire.Entity, values []Requirement) ([]Requirement, error) {
	out := append([]Requirement(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i].Identity < out[j].Identity })
	for i := range out {
		out[i].Properties = sortedIDs(out[i].Properties)
		if out[i].Identity == "" || out[i].MinimumRevision == 0 || !present(entities, out[i].Construct) || duplicateRequirement(out, i) || !allPresentUnique(entities, out[i].Properties) {
			return nil, fmt.Errorf("target_plan.requirement")
		}
	}
	return out, nil
}

func normalizeRules(entities map[wire.ID]wire.Entity, values []Rule) ([]Rule, error) {
	out := append([]Rule(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Fidelity != out[j].Fidelity {
			return out[i].Fidelity < out[j].Fidelity
		}
		a, b := optionalID(out[i].Dependency), optionalID(out[j].Dependency)
		if a != b {
			return a < b
		}
		return out[i].Identity < out[j].Identity
	})
	seen := map[string]bool{}
	for i := range out {
		out[i].PreservedProperties, out[i].Evidence = sortedIDs(out[i].PreservedProperties), sortedIDs(out[i].Evidence)
		if out[i].Identity == "" || seen[out[i].Identity] || out[i].MaximumRevision == 0 || out[i].Fidelity >= Impossible || !present(entities, out[i].Construct) || !allPresentUnique(entities, out[i].PreservedProperties) || !allPresentUnique(entities, out[i].Evidence) || (out[i].Dependency != nil && !present(entities, *out[i].Dependency)) {
			return nil, fmt.Errorf("target_plan.rule")
		}
		seen[out[i].Identity] = true
	}
	return out, nil
}

func normalizeBoundaries(entities map[wire.ID]wire.Entity, _ wire.ID, values []Boundary) ([]Boundary, error) {
	out := append([]Boundary(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i].Requirement < out[j].Requirement })
	for i, value := range out {
		if value.Requirement == "" || duplicateBoundary(out, i) || !present(entities, value.Provider) || !present(entities, value.Consumer) || !present(entities, value.Interface) || !present(entities, value.Transport) {
			return nil, fmt.Errorf("target_plan.boundary")
		}
	}
	return out, nil
}

func validateCandidateBoundaries(requirements []Requirement, rules []Rule, boundaries []Boundary) error {
	requirementByName := map[string]Requirement{}
	for _, requirement := range requirements {
		requirementByName[requirement.Identity] = requirement
	}
	for _, boundary := range boundaries {
		requirement, exists := requirementByName[boundary.Requirement]
		if !exists {
			return fmt.Errorf("target_plan.boundary_requirement")
		}
		candidate := false
		for _, rule := range rules {
			if rule.Construct == requirement.Construct && rule.Fidelity != Exact && rule.Fidelity != Refined {
				candidate = true
				break
			}
		}
		if !candidate {
			return fmt.Errorf("target_plan.boundary_without_adaptation:%s", boundary.Requirement)
		}
	}
	return nil
}

func selectRule(requirement Requirement, rules []Rule, allowed map[Fidelity]bool) (Rule, bool) {
	for _, rule := range rules {
		if rule.Construct == requirement.Construct && rule.MaximumRevision >= requirement.MinimumRevision && containsAll(rule.PreservedProperties, requirement.Properties) && allowed[rule.Fidelity] {
			return rule, true
		}
	}
	return Rule{}, false
}

func modelDigest(authority []byte, contracts []contractcatalog.Contract, model Model) [sha256.Size]byte {
	requirements := append([]Requirement(nil), model.Requirements...)
	sort.Slice(requirements, func(i, j int) bool { return requirements[i].Identity < requirements[j].Identity })
	rules := append([]Rule(nil), model.Target.Rules...)
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Fidelity != rules[j].Fidelity {
			return rules[i].Fidelity < rules[j].Fidelity
		}
		a, b := optionalID(rules[i].Dependency), optionalID(rules[j].Dependency)
		if a != b {
			return a < b
		}
		return rules[i].Identity < rules[j].Identity
	})
	allowed := append([]Fidelity(nil), model.Allowed...)
	sort.Slice(allowed, func(i, j int) bool { return allowed[i] < allowed[j] })
	boundaries := append([]Boundary(nil), model.Boundaries...)
	sort.Slice(boundaries, func(i, j int) bool { return boundaries[i].Requirement < boundaries[j].Requirement })
	h := sha256.New()
	h.Write([]byte("seme.target-plan.model.v1\x00"))
	a := sha256.Sum256(authority)
	h.Write(a[:])
	for _, contract := range contracts {
		pin, digest := contract.Pin(), contract.Digest()
		writeID(h, pin.Module)
		writeID(h, pin.Revision)
		h.Write(digest[:])
	}
	writeID(h, model.Root)
	writeText(h, model.Target.Name)
	_ = binary.Write(h, binary.BigEndian, model.Target.Revision)
	for _, r := range requirements {
		writeText(h, r.Identity)
		writeID(h, r.Construct)
		_ = binary.Write(h, binary.BigEndian, r.MinimumRevision)
		for _, p := range sortedIDs(r.Properties) {
			writeID(h, p)
		}
	}
	for _, r := range rules {
		writeText(h, r.Identity)
		writeID(h, r.Construct)
		_ = binary.Write(h, binary.BigEndian, r.MaximumRevision)
		_ = binary.Write(h, binary.BigEndian, uint64(r.Fidelity))
		for _, p := range sortedIDs(r.PreservedProperties) {
			writeID(h, p)
		}
		if r.Dependency != nil {
			writeID(h, *r.Dependency)
		}
		for _, e := range sortedIDs(r.Evidence) {
			writeID(h, e)
		}
	}
	for _, x := range allowed {
		_ = binary.Write(h, binary.BigEndian, uint64(x))
	}
	for _, b := range boundaries {
		writeText(h, b.Requirement)
		writeID(h, b.Provider)
		writeID(h, b.Consumer)
		writeID(h, b.Interface)
		writeID(h, b.Transport)
	}
	var out [sha256.Size]byte
	copy(out[:], h.Sum(nil))
	return out
}

func stableID(seed [sha256.Size]byte, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.target-plan.identity.v1\x00"))
	h.Write(seed[:])
	for _, p := range parts {
		writeText(h, p)
	}
	var out wire.ID
	copy(out[:], h.Sum(nil))
	return out
}
func writeText(h interface{ Write([]byte) (int, error) }, s string) {
	_ = binary.Write(h, binary.BigEndian, uint64(len(s)))
	_, _ = h.Write([]byte(s))
}
func writeID(h interface{ Write([]byte) (int, error) }, x wire.ID) { _, _ = h.Write(x[:]) }
func sortedIDs(in []wire.ID) []wire.ID {
	out := append([]wire.ID(nil), in...)
	sort.Slice(out, func(i, j int) bool { return bytes.Compare(out[i][:], out[j][:]) < 0 })
	return out
}
func containsAll(have, want []wire.ID) bool {
	m := map[wire.ID]bool{}
	for _, x := range have {
		m[x] = true
	}
	for _, x := range want {
		if !m[x] {
			return false
		}
	}
	return true
}
func allPresentUnique(es map[wire.ID]wire.Entity, xs []wire.ID) bool {
	seen := map[wire.ID]bool{}
	for _, x := range xs {
		if seen[x] || !present(es, x) {
			return false
		}
		seen[x] = true
	}
	return true
}
func present(es map[wire.ID]wire.Entity, x wire.ID) bool { _, ok := es[x]; return ok }
func optionalID(x *wire.ID) string {
	if x == nil {
		return ""
	}
	return x.String()
}
func duplicateRequirement(xs []Requirement, i int) bool {
	return i > 0 && xs[i-1].Identity == xs[i].Identity
}
func duplicateBoundary(xs []Boundary, i int) bool {
	return i > 0 && xs[i-1].Requirement == xs[i].Requirement
}
func cloneEnvelope(in wire.Envelope) wire.Envelope {
	b, _ := wire.Encode(in)
	out, _ := wire.Decode(b)
	return out
}
func sameEntity(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func entity(x wire.ID, s string, fs map[wire.ID]wire.Value) wire.Entity {
	return wire.Entity{ID: x, Schema: id(s), Version: 1, Fields: fs}
}
func ref(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
func refs(xs []wire.ID) wire.Value {
	v := wire.Value{Tag: 7}
	for _, x := range xs {
		v.List = append(v.List, ref(x))
	}
	return v
}
func records(xs []wire.Value) wire.Value { return list(xs) }
func list(xs []wire.Value) wire.Value {
	return wire.Value{Tag: 7, List: append([]wire.Value(nil), xs...)}
}
func blob(x []byte) wire.Value     { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func unsigned(x uint64) wire.Value { return wire.Value{Tag: 3, Unsigned: x} }
func boolean(x bool) wire.Value {
	if x {
		return wire.Value{Tag: 2}
	}
	return wire.Value{Tag: 1}
}
func id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, err := wire.ParseID(s)
	if err != nil {
		panic(err)
	}
	return x
}
