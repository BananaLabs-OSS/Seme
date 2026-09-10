// Package goupb03pipeline extends the validated Go UPB-02 build with one
// bounded offline dependency closure and its Project v4 binding.
package goupb03pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyemitter"
	"seme.local/reference/goofflineclosure"
	"seme.local/reference/goprojectdependency"
	"seme.local/reference/goprojectpipeline"
	"seme.local/reference/goprovider"
	"seme.local/reference/projectdependencyinstance"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/projectv4emitter"
)

type DependencyInput struct{ ProjectRoot, ProxyRoot, Module, Version, LocalFrom, LocalTo string }
type Input struct {
	Base       goprojectpipeline.Input
	Contracts  contractcatalog.ProjectContractSetV4
	Dependency DependencyInput
}
type Result struct {
	Base                    goprojectpipeline.Result
	DependencyV1, ProjectV4 []byte
}

func Build(ctx context.Context, in Input) (Result, error) {
	if err := validateContracts(in.Base.Contracts, in.Contracts); err != nil {
		return Result{}, err
	}
	base, err := goprojectpipeline.Build(ctx, in.Base)
	if err != nil {
		return Result{}, err
	}
	if err = ctx.Err(); err != nil {
		return Result{}, err
	}
	closure, err := goofflineclosure.Load(goofflineclosure.Input{ProjectRoot: in.Dependency.ProjectRoot, ProxyRoot: in.Dependency.ProxyRoot, Module: in.Dependency.Module, Version: in.Dependency.Version, LocalFrom: in.Dependency.LocalFrom, LocalTo: in.Dependency.LocalTo, Resolution: base.Resolution})
	if err != nil {
		return Result{}, fmt.Errorf("go_upb03_pipeline.dependency:%w", err)
	}
	closure, err = goprojectdependency.BindMetadata(closure, base.ProjectV3)
	if err != nil {
		return Result{}, fmt.Errorf("go_upb03_pipeline.applicability:%w", err)
	}
	dependency, err := dependencyemitter.Emit(in.Contracts.Dependency(), closure)
	if err != nil {
		return Result{}, fmt.Errorf("go_upb03_pipeline.dependency_emit:%w", err)
	}
	if err = ctx.Err(); err != nil {
		return Result{}, err
	}
	v3 := projectgraphinstance.Inputs{Contracts: in.Base.Contracts.V3, ProjectV2: in.Base.Contracts.V2.Project(), Project: base.ProjectV1, Inventory: base.InventoryV2, PackageGraph: base.PackageV2, Composed: base.ProjectV3}
	v4, err := projectv4emitter.Emit(projectv4emitter.Input{Contracts: in.Contracts, ProjectV3: v3, Dependency: dependency})
	if err != nil {
		return Result{}, fmt.Errorf("go_upb03_pipeline.project_v4:%w", err)
	}
	if err = goprojectdependency.Validate(projectdependencyinstance.Inputs{Contracts: in.Contracts, ProjectV3: v3, Dependency: dependency, Composed: v4}); err != nil {
		return Result{}, fmt.Errorf("go_upb03_pipeline.applicability_validate:%w", err)
	}
	return Result{Base: cloneBase(base), DependencyV1: clone(dependency), ProjectV4: clone(v4)}, nil
}

func validateContracts(base goprojectpipeline.Contracts, v4 contractcatalog.ProjectContractSetV4) error {
	if !base.V1.Validated() || !base.V2.Validated() || !base.V3.Validated() || !v4.Validated() {
		return fmt.Errorf("go_upb03_pipeline.contracts")
	}
	want := []contractcatalog.Pin{
		{Module: mustID("9000"), Revision: mustID("9023")},
		{Module: mustID("b000"), Revision: mustID("b002")},
		{Module: mustID("e000"), Revision: mustID("e004")},
		{Module: mustID("f000"), Revision: mustID("f001")},
	}
	got := []contractcatalog.Pin{v4.Execution().Pin(), v4.Package().Pin(), v4.Project().Pin(), v4.Dependency().Pin()}
	for i := range want {
		if got[i] != want[i] {
			return fmt.Errorf("go_upb03_pipeline.contract_pin")
		}
	}
	for _, c := range []contractcatalog.Contract{base.V1.Execution(), base.V2.Execution(), base.V3.Execution()} {
		if c.Pin() != got[0] {
			return fmt.Errorf("go_upb03_pipeline.execution_contract_mismatch")
		}
	}
	if base.V3.Package().Pin() != got[1] {
		return fmt.Errorf("go_upb03_pipeline.package_contract_mismatch")
	}
	return nil
}

// Publish reserves a new bundle directory and writes the completion manifest
// only after all seven artifacts have been written.
func Publish(destination string, r Result) error {
	if !filepath.IsAbs(destination) || filepath.Clean(destination) != destination || filepath.Base(destination) == "." {
		return fmt.Errorf("go_upb03_pipeline.destination")
	}
	parent := filepath.Dir(destination)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("go_upb03_pipeline.parent")
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil || resolved != parent {
		return fmt.Errorf("go_upb03_pipeline.parent_symlink")
	}
	files := []struct {
		name string
		data []byte
	}{{"construction.g1", r.Base.CanonicalG1}, {"project-v1.seme", r.Base.ProjectV1}, {"inventory-v2.seme", r.Base.InventoryV2}, {"package-v2.seme", r.Base.PackageV2}, {"project-v3.seme", r.Base.ProjectV3}, {"dependency-v1.seme", r.DependencyV1}, {"project-v4.seme", r.ProjectV4}}
	for _, f := range files {
		if len(f.data) == 0 {
			return fmt.Errorf("go_upb03_pipeline.artifact_empty:%s", f.name)
		}
	}
	if err = os.Mkdir(destination, 0700); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("go_upb03_pipeline.destination_exists")
		}
		return err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(destination)
		}
	}()
	manifest := []byte("seme-go-upb03-bundle-v1\n")
	for _, f := range files {
		if err = os.WriteFile(filepath.Join(destination, f.name), f.data, 0600); err != nil {
			return err
		}
		sum := sha256.Sum256(f.data)
		manifest = append(manifest, []byte(f.name+" "+hex.EncodeToString(sum[:])+"\n")...)
	}
	if err = os.WriteFile(filepath.Join(destination, "COMPLETE.sha256"), manifest, 0600); err != nil {
		return err
	}
	complete = true
	return nil
}

func cloneBase(x goprojectpipeline.Result) goprojectpipeline.Result {
	x.CanonicalG1 = clone(x.CanonicalG1)
	x.ProjectV1 = clone(x.ProjectV1)
	x.InventoryV2 = clone(x.InventoryV2)
	x.PackageV2 = clone(x.PackageV2)
	x.ProjectV3 = clone(x.ProjectV3)
	for i := range x.Resolution.Packages {
		p := &x.Resolution.Packages[i]
		p.Files = append([]string(nil), p.Files...)
		p.Declarations = append([]goprovider.ResolvedDeclaration(nil), p.Declarations...)
		p.Imports = append([]goprovider.ResolvedImport(nil), p.Imports...)
	}
	return x
}
func clone(x []byte) []byte { return append([]byte(nil), x...) }

func mustID(short string) (id [16]byte) {
	b, err := hex.DecodeString(fmt.Sprintf("%032s", short))
	if err != nil {
		panic(err)
	}
	copy(id[:], b)
	return id
}
