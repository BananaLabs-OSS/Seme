// Command target-plan-check independently verifies the scoped Target Contract
// v1 Wasm/Pulp plan. It is a conformance oracle, not the solver authority.
package main

import (
	"fmt"
	"os"

	"seme.local/reference/goprovider"
	"seme.local/reference/wire"
)

func main() {
	if len(os.Args) != 4 || (os.Args[3] != "allow-adapted" && os.Args[3] != "exact-only") {
		fatal("usage: target-plan-check PLAN.seme MANIFEST.json allow-adapted|exact-only")
	}
	graph, err := wire.Read(os.Args[1])
	check(err)
	manifest, err := goprovider.ReadManifest(os.Args[2])
	check(err)
	policy := os.Args[3]

	packages := bySchema(graph, 0xb010)
	dependencies := bySchema(graph, 0xb012)
	runtimes := bySchema(graph, 0xb013)
	mappings := bySchema(graph, 0xb014)
	interfaces := bySchema(graph, 0xb011)
	providerDeclarations := bySchema(graph, 0x7013)
	canonicalFunctions := bySchema(graph, 0x9011)
	recordTypes := bySchema(graph, 0x9030)
	recordFields := bySchema(graph, 0x9031)
	fieldReads := bySchema(graph, 0x9032)
	recordConstructs := bySchema(graph, 0x9033)
	stringTypes := bySchema(graph, 0x9040)
	bytesTypes := bySchema(graph, 0x9041)
	resultTypes := bySchema(graph, 0x9042)
	resultOKs := bySchema(graph, 0x9043)
	resultErrors := bySchema(graph, 0x9044)
	stringLiterals := bySchema(graph, 0x9050)
	stringIsEmpty := bySchema(graph, 0x9051)
	conditionals := bySchema(graph, 0x9052)
	functionCalls := bySchema(graph, 0x9060)
	integerLiterals := bySchema(graph, 0x9070)
	integerTypes := bySchema(graph, 0x9010)
	effects := bySchema(graph, 0x15)
	capabilities := bySchema(graph, 0x16)
	targets := bySchema(graph, 0xc010)
	requirements := bySchema(graph, 0xc011)
	rules := bySchema(graph, 0xc012)
	resolutions := bySchema(graph, 0xc013)
	plans := bySchema(graph, 0xc014)
	boundaries := bySchema(graph, 0xc015)
	must(len(packages) == 1 && len(dependencies) == 1 && len(mappings) == 1 && len(interfaces) == 1, "package requirement cardinality mismatch")
	must(len(providerDeclarations) == 2 && len(canonicalFunctions) == 2 && len(functionCalls) == 1, "source/canonical call graph cardinality mismatch")
	must(len(recordTypes) == 3 && len(recordFields) == 9 && len(fieldReads) == 5 && len(recordConstructs) == 2, "record semantics cardinality mismatch")
	must(len(stringTypes) == 1 && len(bytesTypes) == 1 && len(resultTypes) == 1 && len(resultOKs) == 1 && len(resultErrors) == 1, "variable/result semantics cardinality mismatch")
	must(len(stringLiterals) == 1 && len(stringIsEmpty) == 1 && len(conditionals) == 1, "conditional semantics cardinality mismatch")
	must(len(integerTypes) == 1 && len(integerLiterals) == 1, "integer literal semantics cardinality mismatch")
	must(len(runtimes) == 2 && len(effects) == 1 && len(capabilities) == 1, "effect/runtime cardinality mismatch")
	must(len(targets) == 1 && len(requirements) == 2 && len(rules) == 2 && len(resolutions) == 2 && len(plans) == 1, "target plan cardinality mismatch")

	pkg := packages[0]
	must(string(field(pkg, 0xb100).Bytes) == "example.com/seme-quota-log-proof", "package name mismatch")
	must(string(field(pkg, 0xb101).Bytes) == manifest.Revision, "package revision mismatch")
	must(len(field(pkg, 0xb103).List) == 1, "log dependency missing")
	must(len(field(pkg, 0xb104).List) == 1, "logging effect missing")
	must(string(field(dependencies[0], 0xb120).Bytes) == "go:log", "dependency identity mismatch")
	must(string(field(effects[0], 0x150).Bytes) == "observability.log", "effect identity mismatch")
	must(field(effects[0], 0x151).Reference == capabilities[0].ID, "effect capability mismatch")
	must(string(field(capabilities[0], 0x160).Bytes) == "observability.log", "capability name mismatch")
	must(field(mappings[0], 0xb142).Unsigned == 2, "package mapping must remain adapted")
	mappingSource := graph.Entities[field(mappings[0], 0xb140).Reference]
	mappingTarget := graph.Entities[field(mappings[0], 0xb141).Reference]
	must(mappingSource.Schema == identity(0x7013), "mapping source is not provider Declaration")
	must(mappingTarget.Schema == identity(0x9011), "mapping target is not Core Function")
	must(field(interfaces[0], 0xb111).Reference == mappingTarget.ID, "typed interface bypasses canonical Function")
	callee := graph.Entities[field(functionCalls[0], 0x9600).Reference]
	must(callee.Schema == identity(0x9011) && callee.ID != mappingTarget.ID, "call does not reference helper Function")
	must(len(field(functionCalls[0], 0x9601).List) == 3, "helper call arguments mismatch")
	must(field(integerLiterals[0], 0x9700).Unsigned == 0 && field(integerLiterals[0], 0x9701).Reference == integerTypes[0].ID, "typed integer literal mismatch")
	must(len(field(interfaces[0], 0xb112).List) == 1, "typed interface request record missing")
	requestType := graph.Entities[field(interfaces[0], 0xb112).List[0].Reference]
	resultType := graph.Entities[field(interfaces[0], 0xb113).Reference]
	responseType := graph.Entities[field(resultType, 0x9400).Reference]
	must(requestType.Schema == identity(0x9030) && string(field(requestType, 0x9300).Bytes) == "AdmitRequest", "request record mismatch")
	must(resultType.Schema == identity(0x9042), "result type mismatch")
	errorType := graph.Entities[field(resultType, 0x9401).Reference]
	must(errorType.Schema == identity(0x9030) && string(field(errorType, 0x9300).Bytes) == "AdmitError", "error record mismatch")
	must(responseType.Schema == identity(0x9030) && string(field(responseType, 0x9300).Bytes) == "AdmitResponse", "response record mismatch")
	must(len(field(requestType, 0x9301).List) == 5 && len(field(responseType, 0x9301).List) == 3, "record field shape mismatch")

	target := targets[0]
	must(string(field(target, 0xc100).Bytes) == "wasm32-pulp-v1", "target name mismatch")
	must(field(target, 0xc101).Unsigned == 1 && len(field(target, 0xc102).List) == 2, "target rules mismatch")
	for _, rule := range rules {
		must(field(rule, 0xc123).Unsigned == 2, "support rule relabeled from adapted")
		must(field(rule, 0xc124).Tag == 6, "support rule host dependency missing")
		must(len(field(rule, 0xc125).List) == 1, "support rule evidence missing")
	}

	plan := plans[0]
	must(field(plan, 0xc141).Reference == target.ID && len(field(plan, 0xc142).List) == 2, "plan target/resolutions mismatch")
	if policy == "allow-adapted" {
		must(field(plan, 0xc144).Tag == 2, "adapted plan is not executable")
		must(len(boundaries) == 1 && len(field(plan, 0xc143).List) == 1, "adapted boundary missing")
		for _, resolution := range resolutions {
			must(field(resolution, 0xc132).Tag == 6, "selected adapted rule missing")
			must(field(resolution, 0xc133).Unsigned == 2, "adapted resolution relabeled")
			must(len(field(resolution, 0xc134).List) == 0 && len(field(resolution, 0xc135).List) == 0, "adapted resolution unresolved")
		}
		boundary := boundaries[0]
		must(field(boundary, 0xc150).Reference == field(boundary, 0xc153).Reference, "host transport/provider mismatch")
		must(field(boundary, 0xc151).Reference == pkg.ID, "boundary consumer mismatch")
	} else {
		must(field(plan, 0xc144).Tag == 1, "exact-only plan is executable")
		must(len(boundaries) == 0 && len(field(plan, 0xc143).List) == 0, "exact-only plan leaked a boundary")
		for _, resolution := range resolutions {
			_, selected := resolution.Fields[identity(0xc132)]
			must(!selected, "exact-only resolution selected adapted rule")
			must(field(resolution, 0xc133).Unsigned == 6, "exact-only resolution is not impossible")
			must(len(field(resolution, 0xc134).List) == 1 && len(field(resolution, 0xc135).List) == 1, "impossible evidence missing")
			diagnosticReference := field(resolution, 0xc135).List[0].Reference
			diagnostic, exists := graph.Entities[diagnosticReference]
			must(exists && diagnostic.Schema == identity(0x1a), "resolution diagnostic is not canonical Diagnostic")
			rule, exists := graph.Entities[field(diagnostic, 0x1a0).Reference]
			must(exists && rule.Schema == identity(0x19) && string(field(rule, 0x190).Bytes) == "target.adaptation_forbidden", "diagnostic rule mismatch")
		}
	}
}

func bySchema(graph wire.Envelope, low uint64) []wire.Entity {
	want := identity(low)
	var out []wire.Entity
	for _, entity := range graph.Entities {
		if entity.Schema == want {
			out = append(out, entity)
		}
	}
	return out
}
func field(entity wire.Entity, low uint64) wire.Value {
	value, ok := entity.Fields[identity(low)]
	must(ok, "field %x missing", low)
	return value
}
func identity(low uint64) wire.ID {
	var value wire.ID
	value[14] = byte(low >> 8)
	value[15] = byte(low)
	return value
}
func must(ok bool, format string, values ...any) {
	if !ok {
		fatal(fmt.Sprintf(format, values...))
	}
}
func check(err error) {
	if err != nil {
		fatal(err.Error())
	}
}
func fatal(message string) {
	fmt.Fprintln(os.Stderr, "target-plan-check:", message)
	os.Exit(65)
}
