// Package projectbuild is the bounded Go-source to canonical Project artifact
// pipeline. Compilation remains an injected authoritative boundary.
package projectbuild

import (
	"bytes"
	"context"
	"fmt"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprovider"
	"seme.local/reference/projectemitter"
	"seme.local/reference/wire"
)

type Compile func(context.Context, []byte) ([]byte, error)

type Result struct {
	Artifact     []byte
	SourceDigest string
	Packages     []goprovider.PackageMetadata
	Resolution   goprovider.ResolutionManifest
}

func Build(ctx context.Context, snapshot goprovider.DocumentSnapshot, contracts contractcatalog.ProjectContractSet, executionG1 []byte, compile Compile) (Result, error) {
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

	input := projectemitter.Input{
		Identity: snapshot.ModulePath, RootPackage: snapshot.PackagePath,
		Execution: execution,
	}
	for _, p := range lifted.Packages {
		out := projectemitter.Package{Name: p.Name}
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
	artifact, err := projectemitter.Emit(contracts, input)
	if err != nil {
		return Result{}, fmt.Errorf("project_build.emit:%w", err)
	}
	return Result{
		Artifact: append([]byte(nil), artifact...), SourceDigest: lifted.ContentDigest,
		Packages: clonePackages(lifted.Packages), Resolution: cloneResolution(lifted.Resolution),
	}, nil
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
