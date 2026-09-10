// Package projectbuild is the bounded Go-source to canonical Project artifact
// pipeline. Compilation remains an injected authoritative boundary.
package projectbuild

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/executionprofile"
	"seme.local/reference/goprovider"
	"seme.local/reference/projectemitter"
	"seme.local/reference/wire"
)

type Compile func(context.Context, []byte) ([]byte, error)

type Result struct {
	Artifact     []byte
	CanonicalG1  []byte
	SourceDigest string
	Packages     []goprovider.PackageMetadata
	Resolution   goprovider.ResolutionManifest
}

func Build(ctx context.Context, snapshot goprovider.DocumentSnapshot, contracts contractcatalog.ProjectContractSet, executionG1 []byte, compile Compile) (Result, error) {
	return build(ctx, snapshot, contracts.Execution(), executionG1, compile, func(input projectemitter.Input) ([]byte, error) { return projectemitter.Emit(contracts, input) })
}

// BuildV8 performs one fresh lift and emits its semantic base directly under
// the authenticated Execution-v36/Package-v4/Project-v8 authorities.
func BuildV8(ctx context.Context, snapshot goprovider.DocumentSnapshot, contracts contractcatalog.ProjectContractSetV8, executionG1 []byte, compile Compile) (Result, error) {
	if !contracts.Validated() {
		return Result{}, fmt.Errorf("project_build.contracts")
	}
	return build(ctx, snapshot, contracts.Execution(), executionG1, compile, func(input projectemitter.Input) ([]byte, error) { return projectemitter.EmitV8Base(contracts, input) })
}

func build(ctx context.Context, snapshot goprovider.DocumentSnapshot, executionContract contractcatalog.Contract, executionG1 []byte, compile Compile, emit func(projectemitter.Input) ([]byte, error)) (Result, error) {
	if compile == nil {
		return Result{}, fmt.Errorf("project_build.compiler_missing")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	session, err := goprovider.NewIncrementalSession(executionG1)
	if err != nil {
		return Result{}, err
	}
	lifted := session.Apply(snapshot)
	if !lifted.Accepted || !lifted.Valid || lifted.Revision != snapshot.Revision || lifted.LastValidRevision != snapshot.Revision || lifted.Disposition != "accepted-valid" {
		return Result{}, fmt.Errorf("project_build.current_snapshot_invalid")
	}
	compiled, err := compile(ctx, []byte(lifted.CanonicalG1))
	if err != nil {
		return Result{}, fmt.Errorf("project_build.compile:%w", err)
	}
	execution, err := wire.Decode(compiled)
	if err != nil {
		return Result{}, fmt.Errorf("project_build.execution_wire:%w", err)
	}
	canonical, err := wire.Encode(execution)
	if err != nil || !bytes.Equal(canonical, compiled) {
		return Result{}, fmt.Errorf("project_build.execution_noncanonical")
	}
	if err = executionprofile.ValidateConstruction(executionContract, execution); err != nil {
		return Result{}, fmt.Errorf("project_build.execution_profile:%w", err)
	}

	input := projectemitter.Input{
		Identity: snapshot.ModulePath, RootPackage: snapshot.PackagePath,
		Execution: execution,
	}
	effects, err := deriveEffects(execution, lifted.Packages)
	if err != nil {
		return Result{}, err
	}
	for _, p := range lifted.Packages {
		out := projectemitter.Package{Name: p.Name}
		out.Effects = append([]wire.ID(nil), effects[p.Name]...)
		for _, dependency := range p.Dependencies {
			out.Dependencies = append(out.Dependencies, projectemitter.Dependency{Name: dependency, Package: dependency})
		}
		for _, function := range p.Functions {
			item, err := interfaceFromMetadata(function)
			if err != nil {
				return Result{}, err
			}
			out.Interfaces = append(out.Interfaces, item)
		}
		input.Packages = append(input.Packages, out)
	}
	artifact, err := emit(input)
	if err != nil {
		return Result{}, fmt.Errorf("project_build.emit:%w", err)
	}
	return Result{
		Artifact: append([]byte(nil), artifact...), CanonicalG1: []byte(lifted.CanonicalG1), SourceDigest: lifted.ContentDigest,
		Packages: clonePackages(lifted.Packages), Resolution: cloneResolution(lifted.Resolution),
	}, nil
}

func deriveEffects(execution wire.Envelope, packages []goprovider.PackageMetadata) (map[string][]wire.ID, error) {
	functionSchema := mustID("00000000000000000000000000009011")
	methodSchema := mustID("0000000000000000000000000000a002")
	effectSchema := mustID("00000000000000000000000000000015")
	invokeSchema := mustID("000000000000000000000000000090f1")
	invokeEffect := mustID("00000000000000000000000000009f10")
	reachable, err := executionProgramClosure(execution)
	if err != nil {
		return nil, err
	}
	type callableOwner struct {
		packageName string
		schema      wire.ID
	}
	functionOwners := map[wire.ID]callableOwner{}
	for _, p := range packages {
		for _, member := range p.Members {
			function, err := wire.ParseID(member.ID)
			if err != nil {
				return nil, fmt.Errorf("project_build.member_identity:%w", err)
			}
			if prior, exists := functionOwners[function]; exists && prior.packageName != p.Name {
				return nil, fmt.Errorf("project_build.function_multiple_owners:%s", function)
			}
			functionOwners[function] = callableOwner{packageName: p.Name, schema: functionSchema}
		}
		for _, declaration := range p.Supplemental {
			if declaration.Kind != goprovider.SemanticMethod {
				continue
			}
			function, err := wire.ParseID(declaration.Declaration)
			if err != nil {
				return nil, fmt.Errorf("project_build.method_identity:%w", err)
			}
			if prior, exists := functionOwners[function]; exists && prior.packageName != p.Name {
				return nil, fmt.Errorf("project_build.function_multiple_owners:%s", function)
			}
			functionOwners[function] = callableOwner{packageName: p.Name, schema: methodSchema}
		}
	}
	result := map[string][]wire.ID{}
	effectClaims := map[wire.ID]map[string]bool{}
	callables := make([]wire.ID, 0, len(functionOwners))
	for function := range functionOwners {
		callables = append(callables, function)
	}
	sort.Slice(callables, func(i, j int) bool { return bytes.Compare(callables[i][:], callables[j][:]) < 0 })
	for _, function := range callables {
		ownership := functionOwners[function]
		if !reachable[function] {
			continue
		}
		if q, ok := execution.Entities[function]; !ok || q.Schema != ownership.schema {
			return nil, fmt.Errorf("project_build.member_missing:%s", function)
		}
		owner := ownership.packageName
		seen := map[wire.ID]bool{}
		queue := []wire.ID{function}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			if seen[current] {
				continue
			}
			seen[current] = true
			entity, ok := execution.Entities[current]
			if !ok {
				return nil, fmt.Errorf("project_build.effect_reference_missing:%s", current)
			}
			if current != function && (entity.Schema == functionSchema || entity.Schema == methodSchema) {
				continue // A call belongs to the callee's package, not the caller.
			}
			if entity.Schema == invokeSchema {
				value, ok := entity.Fields[invokeEffect]
				if !ok || value.Tag != 6 {
					return nil, fmt.Errorf("project_build.effect_invoke_shape:%s", current)
				}
				claims := effectClaims[value.Reference]
				if claims == nil {
					claims = map[string]bool{}
					effectClaims[value.Reference] = claims
				}
				if !claims[owner] {
					claims[owner] = true
					result[owner] = append(result[owner], value.Reference)
				}
			}
			queue = append(queue, entityReferences(entity)...)
		}
	}
	effects := make([]wire.ID, 0)
	for identity, entity := range execution.Entities {
		if entity.Schema == effectSchema && reachable[identity] {
			effects = append(effects, identity)
		}
	}
	sort.Slice(effects, func(i, j int) bool { return bytes.Compare(effects[i][:], effects[j][:]) < 0 })
	for _, identity := range effects {
		if len(effectClaims[identity]) == 0 {
			return nil, fmt.Errorf("project_build.effect_unowned:%s", identity)
		}
	}
	for name := range result {
		sort.Slice(result[name], func(i, j int) bool { return bytes.Compare(result[name][i][:], result[name][j][:]) < 0 })
	}
	return result, nil
}

func executionProgramClosure(execution wire.Envelope) (map[wire.ID]bool, error) {
	programSchema := mustID("00000000000000000000000000009015")
	var program wire.ID
	for identity, entity := range execution.Entities {
		if entity.Schema != programSchema {
			continue
		}
		if program != (wire.ID{}) {
			return nil, fmt.Errorf("project_build.program_count")
		}
		program = identity
	}
	if program == (wire.ID{}) {
		return nil, fmt.Errorf("project_build.program_count")
	}
	reachable := map[wire.ID]bool{}
	queue := []wire.ID{program}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if reachable[current] {
			continue
		}
		entity, ok := execution.Entities[current]
		if !ok {
			return nil, fmt.Errorf("project_build.program_reference_missing:%s", current)
		}
		reachable[current] = true
		queue = append(queue, entityReferences(entity)...)
	}
	return reachable, nil
}

func entityReferences(entity wire.Entity) []wire.ID {
	out := []wire.ID{}
	for _, value := range entity.Fields {
		collectReferences(value, &out)
	}
	sort.Slice(out, func(i, j int) bool { return bytes.Compare(out[i][:], out[j][:]) < 0 })
	return out
}

func collectReferences(value wire.Value, out *[]wire.ID) {
	if value.Tag == 6 {
		*out = append(*out, value.Reference)
	}
	for _, item := range value.List {
		collectReferences(item, out)
	}
	for _, item := range value.Record {
		collectReferences(item, out)
	}
}

func mustID(value string) wire.ID {
	id, err := wire.ParseID(value)
	if err != nil {
		panic(err)
	}
	return id
}

func interfaceFromMetadata(function goprovider.PackageFunctionMetadata) (projectemitter.Interface, error) {
	functionID, err := wire.ParseID(function.ID)
	if err != nil {
		return projectemitter.Interface{}, fmt.Errorf("project_build.function_identity:%w", err)
	}
	item := projectemitter.Interface{Name: function.Name, Function: functionID}
	for _, raw := range function.Parameters {
		typeID, err := wire.ParseID(raw)
		if err != nil {
			return projectemitter.Interface{}, fmt.Errorf("project_build.parameter_identity:%w", err)
		}
		item.Parameters = append(item.Parameters, typeID)
	}
	item.Result, err = wire.ParseID(function.Result)
	if err != nil {
		return projectemitter.Interface{}, fmt.Errorf("project_build.result_identity:%w", err)
	}
	return item, nil
}

func clonePackages(in []goprovider.PackageMetadata) []goprovider.PackageMetadata {
	out := make([]goprovider.PackageMetadata, len(in))
	for i, p := range in {
		out[i] = p
		out[i].Dependencies = append([]string(nil), p.Dependencies...)
		out[i].Members = cloneFunctions(p.Members)
		out[i].Functions = cloneFunctions(p.Functions)
		out[i].Supplemental = make([]goprovider.SemanticDeclarationMetadata, len(p.Supplemental))
		for j, declaration := range p.Supplemental {
			out[i].Supplemental[j] = declaration
			out[i].Supplemental[j].ReferencedImports = append([]string(nil), declaration.ReferencedImports...)
			out[i].Supplemental[j].ImportReferences = append([]goprovider.SemanticImportReference(nil), declaration.ImportReferences...)
		}
	}
	return out
}

func cloneFunctions(in []goprovider.PackageFunctionMetadata) []goprovider.PackageFunctionMetadata {
	out := make([]goprovider.PackageFunctionMetadata, len(in))
	for i, function := range in {
		out[i] = function
		out[i].Parameters = append([]string(nil), function.Parameters...)
	}
	return out
}

func cloneResolution(in goprovider.ResolutionManifest) goprovider.ResolutionManifest {
	out := goprovider.ResolutionManifest{Packages: make([]goprovider.ResolvedPackage, len(in.Packages))}
	for i, p := range in.Packages {
		out.Packages[i] = p
		out.Packages[i].Files = append([]string(nil), p.Files...)
		out.Packages[i].Declarations = append([]goprovider.ResolvedDeclaration(nil), p.Declarations...)
		out.Packages[i].Imports = append([]goprovider.ResolvedImport(nil), p.Imports...)
	}
	return out
}
