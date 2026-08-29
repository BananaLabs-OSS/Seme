package goprovider

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strconv"
)

// BuildWasmPulpPlan performs the first exact dependency/effect analysis and
// deterministic Target Contract v1 resolution. The accepted source profile is
// deliberately finite and rejects anything it cannot account for.
func BuildWasmPulpPlan(project string, manifest Manifest, modules [][]byte, policy string) (string, error) {
	files, _, err := nativeFiles(project)
	if err != nil {
		return "", err
	}
	if revisionOf(files) != manifest.Revision {
		return "", fmt.Errorf("target.stale_native_revision")
	}
	declaration, fn, signature, info, err := resolveFunction(project, manifest, "Admit")
	if err != nil {
		return "", err
	}
	if err := validateLoggedAdmit(fn, signature, info); err != nil {
		return "", err
	}
	if policy != "allow-adapted" && policy != "exact-only" {
		return "", fmt.Errorf("target.unsupported_policy:%s", policy)
	}

	integerID := stableID("execution", "type", "i64")
	booleanID := stableID("execution", "type", "bool")
	canonicalFunctionID := stableID("canonical-function", declaration.ID)
	parameterIDs := []string{
		stableID("execution", canonicalFunctionID, "parameter", "0"),
		stableID("execution", canonicalFunctionID, "parameter", "1"),
		stableID("execution", canonicalFunctionID, "parameter", "2"),
	}
	readIDs := []string{
		stableID("execution", canonicalFunctionID, "read", "0"),
		stableID("execution", canonicalFunctionID, "read", "1"),
		stableID("execution", canonicalFunctionID, "read", "2"),
	}
	addID := stableID("execution", canonicalFunctionID, "add")
	comparisonID := stableID("execution", canonicalFunctionID, "less-equal")
	packageID := stableID("package", packagePathOf(declaration.NativeKey))
	interfaceID := stableID("package-interface", declaration.ID)
	dependencyID := stableID("dependency", packageID, "go:log")
	capabilityID := stableID("capability", "observability.log")
	effectID := stableID("effect", "observability.log")
	runtimeID := stableID("runtime-assumption", packageID, "go.log.Printf")
	mappingID := stableID("fidelity", declaration.ID, "wasm32-pulp-v1")
	hostAdapterID := stableID("target-dependency", "pulp.host.log-v1")
	targetID := stableID("target", "wasm32-pulp-v1")
	dependencyRuleID := stableID("target-rule", targetID, dependencyID)
	effectRuleID := stableID("target-rule", targetID, effectID)
	dependencyRequirementID := stableID("requirement", packageID, dependencyID)
	effectRequirementID := stableID("requirement", packageID, effectID)
	dependencyResolutionID := stableID("resolution", targetID, dependencyRequirementID, policy)
	effectResolutionID := stableID("resolution", targetID, effectRequirementID, policy)
	boundaryID := stableID("boundary", packageID, hostAdapterID)
	planID := stableID("execution-plan", packageID, targetID, policy)
	diagnosticRuleID := stableID("diagnostic-rule", "target.adaptation_forbidden")
	dependencyDiagnosticID := stableID("diagnostic", dependencyResolutionID)
	effectDiagnosticID := stableID("diagnostic", effectResolutionID)

	instances := []graphEntity{
		{integerID, entity(integerID, "00000000000000000000000000009010", []graphField{unsignedField(0x9100, 64), {0x9101, "tr"}, unsignedField(0x9102, 0)})},
		{booleanID, entity(booleanID, "00000000000000000000000000009020", nil)},
		{capabilityID, entity(capabilityID, "00000000000000000000000000000016", []graphField{bytesField(0x160, "observability.log")})},
		{effectID, entity(effectID, "00000000000000000000000000000015", []graphField{bytesField(0x150, "observability.log"), refField(0x151, capabilityID)})},
		{runtimeID, entity(runtimeID, "0000000000000000000000000000b013", []graphField{bytesField(0xb130, "go.log.Printf"), bytesField(0xb131, "formatted process-global logging sink")})},
		{hostAdapterID, entity(hostAdapterID, "0000000000000000000000000000b013", []graphField{bytesField(0xb130, "pulp.host.log-v1"), bytesField(0xb131, "Wasm host import guarded by observability.log capability")})},
		{dependencyID, entity(dependencyID, "0000000000000000000000000000b012", []graphField{bytesField(0xb120, "go:log"), bytesField(0xb121, "Go standard library for provider profile"), refField(0xb122, hostAdapterID)})},
		{interfaceID, entity(interfaceID, "0000000000000000000000000000b011", []graphField{bytesField(0xb110, "Admit"), refField(0xb111, canonicalFunctionID), refsField(0xb112, []string{integerID, integerID, integerID}), refField(0xb113, booleanID)})},
		{mappingID, entity(mappingID, "0000000000000000000000000000b014", []graphField{refField(0xb140, declaration.ID), refField(0xb141, canonicalFunctionID), unsignedField(0xb142, 2), bytesField(0xb143, "go/types exact decision lift; Pulp host import adapts log.Printf")})},
		{packageID, entity(packageID, "0000000000000000000000000000b010", []graphField{bytesField(0xb100, packagePathOf(declaration.NativeKey)), bytesField(0xb101, manifest.Revision), refsField(0xb102, []string{interfaceID}), refsField(0xb103, []string{dependencyID}), refsField(0xb104, []string{effectID}), refsField(0xb105, []string{runtimeID}), refsField(0xb106, []string{mappingID})})},
		{dependencyRequirementID, entity(dependencyRequirementID, "0000000000000000000000000000c011", []graphField{refField(0xc110, dependencyID), unsignedField(0xc111, 1), refsField(0xc112, []string{runtimeID})})},
		{effectRequirementID, entity(effectRequirementID, "0000000000000000000000000000c011", []graphField{refField(0xc110, effectID), unsignedField(0xc111, 1), refsField(0xc112, []string{runtimeID})})},
		{dependencyRuleID, entity(dependencyRuleID, "0000000000000000000000000000c012", []graphField{refField(0xc120, dependencyID), unsignedField(0xc121, 1), refsField(0xc122, []string{runtimeID}), unsignedField(0xc123, 2), refField(0xc124, hostAdapterID), refsField(0xc125, []string{mappingID})})},
		{effectRuleID, entity(effectRuleID, "0000000000000000000000000000c012", []graphField{refField(0xc120, effectID), unsignedField(0xc121, 1), refsField(0xc122, []string{runtimeID}), unsignedField(0xc123, 2), refField(0xc124, hostAdapterID), refsField(0xc125, []string{mappingID})})},
		{targetID, entity(targetID, "0000000000000000000000000000c010", []graphField{bytesField(0xc100, "wasm32-pulp-v1"), unsignedField(0xc101, 1), refsField(0xc102, []string{dependencyRuleID, effectRuleID})})},
	}
	instances = append(instances,
		graphEntity{parameterIDs[0], entity(parameterIDs[0], "00000000000000000000000000009012", []graphField{bytesField(0x9120, signature.Params().At(0).Name()), refField(0x9121, integerID), unsignedField(0x9122, 0)})},
		graphEntity{parameterIDs[1], entity(parameterIDs[1], "00000000000000000000000000009012", []graphField{bytesField(0x9120, signature.Params().At(1).Name()), refField(0x9121, integerID), unsignedField(0x9122, 1)})},
		graphEntity{parameterIDs[2], entity(parameterIDs[2], "00000000000000000000000000009012", []graphField{bytesField(0x9120, signature.Params().At(2).Name()), refField(0x9121, integerID), unsignedField(0x9122, 2)})},
		graphEntity{readIDs[0], entity(readIDs[0], "00000000000000000000000000009013", []graphField{refField(0x9130, parameterIDs[0])})},
		graphEntity{readIDs[1], entity(readIDs[1], "00000000000000000000000000009013", []graphField{refField(0x9130, parameterIDs[1])})},
		graphEntity{readIDs[2], entity(readIDs[2], "00000000000000000000000000009013", []graphField{refField(0x9130, parameterIDs[2])})},
		graphEntity{addID, entity(addID, "00000000000000000000000000009014", []graphField{refField(0x9140, readIDs[0]), refField(0x9141, readIDs[1]), refField(0x9142, integerID)})},
		graphEntity{comparisonID, entity(comparisonID, "00000000000000000000000000009021", []graphField{refField(0x9160, addID), refField(0x9161, readIDs[2]), refField(0x9162, integerID)})},
		graphEntity{canonicalFunctionID, entity(canonicalFunctionID, "00000000000000000000000000009011", []graphField{bytesField(0x9110, "Admit"), refsField(0x9111, parameterIDs), refField(0x9112, booleanID), refField(0x9113, comparisonID)})},
	)

	resolutionIDs := []string{dependencyResolutionID, effectResolutionID}
	if policy == "allow-adapted" {
		instances = append(instances,
			graphEntity{dependencyResolutionID, resolution(dependencyResolutionID, dependencyRequirementID, targetID, dependencyRuleID, 2, nil, nil)},
			graphEntity{effectResolutionID, resolution(effectResolutionID, effectRequirementID, targetID, effectRuleID, 2, nil, nil)},
			graphEntity{boundaryID, entity(boundaryID, "0000000000000000000000000000c015", []graphField{refField(0xc150, hostAdapterID), refField(0xc151, packageID), refField(0xc152, interfaceID), refField(0xc153, hostAdapterID)})},
			graphEntity{planID, entity(planID, "0000000000000000000000000000c014", []graphField{refField(0xc140, packageID), refField(0xc141, targetID), refsField(0xc142, resolutionIDs), refsField(0xc143, []string{boundaryID}), {0xc144, "tr"}})},
		)
	} else {
		instances = append(instances,
			graphEntity{diagnosticRuleID, entity(diagnosticRuleID, "00000000000000000000000000000019", []graphField{bytesField(0x190, "target.adaptation_forbidden"), refsField(0x191, nil)})},
			graphEntity{dependencyDiagnosticID, targetDiagnostic(dependencyDiagnosticID, diagnosticRuleID, manifest.Revision, dependencyID)},
			graphEntity{effectDiagnosticID, targetDiagnostic(effectDiagnosticID, diagnosticRuleID, manifest.Revision, effectID)},
			graphEntity{dependencyResolutionID, resolution(dependencyResolutionID, dependencyRequirementID, targetID, "", 6, []string{runtimeID}, []string{dependencyDiagnosticID})},
			graphEntity{effectResolutionID, resolution(effectResolutionID, effectRequirementID, targetID, "", 6, []string{runtimeID}, []string{effectDiagnosticID})},
			graphEntity{planID, entity(planID, "0000000000000000000000000000c014", []graphField{refField(0xc140, packageID), refField(0xc141, targetID), refsField(0xc142, resolutionIDs), refsField(0xc143, nil), {0xc144, "fa"}})},
		)
	}

	var all []graphEntity
	for _, module := range modules {
		all = append(all, graphEntities(module)...)
	}
	all = append(all, instances...)
	return composeGraph(0xc000, manifest.Revision, all)
}

func targetDiagnostic(id, rule, revision, target string) string {
	return entity(id, "0000000000000000000000000000001a", []graphField{
		refField(0x1a0, rule), bytesHexField(0x1a1, revision), bytesHexField(0x1a2, target),
		refsField(0x1a3, nil), refsField(0x1a4, nil), unsignedField(0x1a5, 2),
	})
}

func resolution(id, requirement, target, rule string, fidelity uint64, unresolved, diagnostics []string) string {
	fields := []graphField{refField(0xc130, requirement), refField(0xc131, target), unsignedField(0xc133, fidelity), refsField(0xc134, unresolved), refsField(0xc135, diagnostics)}
	if rule != "" {
		fields = append(fields, refField(0xc132, rule))
	}
	return entity(id, "0000000000000000000000000000c013", fields)
}

func composeGraph(module uint64, revision string, entities []graphEntity) (string, error) {
	sort.Slice(entities, func(i, j int) bool { return entities[i].id < entities[j].id })
	for i := 1; i < len(entities); i++ {
		if entities[i-1].id == entities[i].id {
			return "", fmt.Errorf("target.identity_collision:%s", entities[i].id)
		}
	}
	var out string
	out += fmt.Sprintf("# Generated Wasm/Pulp Target Contract v1 plan.\nve 1\nmo %032x\nrv %s\npc 0\nec %d\n", module, revision, len(entities))
	for _, item := range entities {
		out += "\n" + item.text
	}
	return out, nil
}

func validateLoggedAdmit(fn *ast.FuncDecl, signature *types.Signature, info *types.Info) error {
	if signature.Params().Len() != 3 || signature.Results().Len() != 1 || !isInt64(signature.Params().At(0).Type()) || !isInt64(signature.Params().At(1).Type()) || !isInt64(signature.Params().At(2).Type()) || !isBool(signature.Results().At(0).Type()) {
		return fmt.Errorf("target.unsupported_signature:Admit")
	}
	if len(fn.Body.List) != 3 {
		return fmt.Errorf("target.unsupported_body:Admit")
	}
	assignment, ok := fn.Body.List[0].(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return fmt.Errorf("target.unsupported_decision_binding")
	}
	acceptedDefinition, ok := assignment.Lhs[0].(*ast.Ident)
	if !ok || acceptedDefinition.Name != "accepted" {
		return fmt.Errorf("target.unsupported_decision_name")
	}
	acceptedObject, ok := info.Defs[acceptedDefinition].(*types.Var)
	if !ok || !isBool(acceptedObject.Type()) {
		return fmt.Errorf("target.unresolved_decision")
	}
	comparison, ok := assignment.Rhs[0].(*ast.BinaryExpr)
	if !ok || comparison.Op != token.LEQ {
		return fmt.Errorf("target.unsupported_decision")
	}
	addition, ok := comparison.X.(*ast.BinaryExpr)
	if !ok || addition.Op != token.ADD {
		return fmt.Errorf("target.unsupported_addition")
	}
	if _, _, err := resolvedParameterOperands(addition.X, addition.Y, signature, info); err != nil {
		return fmt.Errorf("target.nonparameter_addition")
	}
	limit, ok := comparison.Y.(*ast.Ident)
	if !ok || info.Uses[limit] != signature.Params().At(2) {
		return fmt.Errorf("target.nonparameter_limit")
	}
	expression, ok := fn.Body.List[1].(*ast.ExprStmt)
	if !ok {
		return fmt.Errorf("target.missing_log_effect")
	}
	call, ok := expression.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		return fmt.Errorf("target.unsupported_log_call")
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return fmt.Errorf("target.unsupported_log_target")
	}
	logFunction, ok := info.Uses[selector.Sel].(*types.Func)
	if !ok || logFunction.Pkg() == nil || logFunction.Pkg().Path() != "log" || logFunction.Name() != "Printf" {
		return fmt.Errorf("target.unresolved_log_dependency")
	}
	format, ok := call.Args[0].(*ast.BasicLit)
	if !ok || format.Kind != token.STRING {
		return fmt.Errorf("target.dynamic_log_format")
	}
	formatValue, err := strconv.Unquote(format.Value)
	if err != nil || formatValue != "quota.accepted=%t" {
		return fmt.Errorf("target.unsupported_log_format")
	}
	acceptedUse, ok := call.Args[1].(*ast.Ident)
	if !ok || info.Uses[acceptedUse] != acceptedObject {
		return fmt.Errorf("target.unsupported_log_value")
	}
	returned, ok := fn.Body.List[2].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return fmt.Errorf("target.unsupported_return")
	}
	result, ok := returned.Results[0].(*ast.Ident)
	if !ok || info.Uses[result] != acceptedObject {
		return fmt.Errorf("target.unsupported_return_value")
	}
	return nil
}
