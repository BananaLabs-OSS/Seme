package goupb05pipeline

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"seme.local/reference/configurationinstance"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyemitter"
	"seme.local/reference/goconfigurationadapter"
	"seme.local/reference/goofflineclosure"
	"seme.local/reference/gopackageadapter"
	"seme.local/reference/gopackagev3adapter"
	"seme.local/reference/goprojectdependency"
	"seme.local/reference/goprovider"
	"seme.local/reference/packagedetailemitter"
	"seme.local/reference/packagev3instance"
	"seme.local/reference/projectbuild"
	"seme.local/reference/projectsource"
	"seme.local/reference/projectv8instance"
	"seme.local/reference/sourceinventory"
	"seme.local/reference/wire"
)

// V8DependencyInput is the bounded offline dependency selection. Resolution is
// performed against the same provider run and source inventory as execution.
type V8DependencyInput struct {
	ProjectRoot, ProxyRoot, Module, Version, LocalFrom, LocalTo string
}

// V8Input contains no v35 contract set or previously built UPB artifact.
type V8Input struct {
	Documents   goprovider.DocumentSnapshot
	Sources     projectsource.Snapshot
	Contracts   contractcatalog.ProjectContractSetV8
	Dependency  V8DependencyInput
	Selection   Selection
	ExecutionG1 []byte
	Compile     projectbuild.Compile
}

// v8Components are all independently reproducible inputs to the final Project
// v8 binding. ProjectV8 is added only by the strict instance layer.
type v8Components struct {
	Construction, Execution, ProjectBase, Inventory, PackageDetail, PackageV4 []byte
	Dependency, ConfigurationBase, ConfigurationV3                            []byte
	Base                                                                      projectbuild.Result
	ConfigurationInput                                                        configurationinstance.V3Input
}

type V8Result struct {
	Construction, Execution, ProjectBase, Inventory, PackageDetail, PackageV4 []byte
	Dependency, ConfigurationV3, ProjectV8                                    []byte
	Base                                                                      projectbuild.Result
	ConfigurationInput                                                        configurationinstance.V3Input
	ProjectInput                                                              projectv8instance.Inputs
}

func BuildV8(ctx context.Context, in V8Input) (V8Result, error) {
	c, err := buildV8Components(ctx, in)
	if err != nil {
		return V8Result{}, err
	}
	p8in := projectv8instance.Inputs{Contracts: in.Contracts, ProjectBase: c.ProjectBase, Inventory: c.Inventory, PackageV2: c.PackageDetail, PackageV4: c.PackageV4, Dependency: c.Dependency, Construction: c.Execution, Configuration: c.ConfigurationInput}
	p8, err := projectv8instance.Emit(p8in)
	if err != nil {
		return V8Result{}, fmt.Errorf("go_upb05_pipeline.project_v8:%w", err)
	}
	p8in.Composed = p8
	if err = projectv8instance.Validate(p8in); err != nil {
		return V8Result{}, fmt.Errorf("go_upb05_pipeline.project_v8_validate:%w", err)
	}
	return V8Result{Construction: clone(c.Construction), Execution: clone(c.Execution), ProjectBase: clone(c.ProjectBase), Inventory: clone(c.Inventory), PackageDetail: clone(c.PackageDetail), PackageV4: clone(c.PackageV4), Dependency: clone(c.Dependency), ConfigurationV3: clone(c.ConfigurationV3), ProjectV8: clone(p8), Base: c.Base, ConfigurationInput: c.ConfigurationInput, ProjectInput: p8in}, nil
}

func PublishV8(destination string, r V8Result) error {
	if !filepath.IsAbs(destination) || filepath.Clean(destination) != destination || filepath.Base(destination) == "." {
		return fmt.Errorf("go_upb05_pipeline.destination")
	}
	parent := filepath.Dir(destination)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("go_upb05_pipeline.parent")
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil || resolved != parent {
		return fmt.Errorf("go_upb05_pipeline.parent_symlink")
	}
	files := v8Files(r)
	for _, f := range files {
		if len(f.data) == 0 {
			return fmt.Errorf("go_upb05_pipeline.artifact_empty:%s", f.name)
		}
	}
	if err = os.Mkdir(destination, 0700); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("go_upb05_pipeline.destination_exists")
		}
		return err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(destination)
		}
	}()
	manifest := []byte("seme-go-upb05-bundle-v2\n")
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

type namedV8Artifact struct {
	name string
	data []byte
}

func v8Files(r V8Result) []namedV8Artifact {
	return []namedV8Artifact{{"construction-v36.g1", r.Construction}, {"execution-v36.seme", r.Execution}, {"project-base-v8.seme", r.ProjectBase}, {"inventory-v8.seme", r.Inventory}, {"package-detail-v4.seme", r.PackageDetail}, {"package-v4.seme", r.PackageV4}, {"dependency-v1.seme", r.Dependency}, {"configuration-v3.seme", r.ConfigurationV3}, {"project-v8.seme", r.ProjectV8}}
}

func buildV8Components(ctx context.Context, in V8Input) (v8Components, error) {
	if !in.Contracts.Validated() || in.Compile == nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.v8_contracts")
	}
	base, err := projectbuild.BuildV8(ctx, in.Documents, in.Contracts, in.ExecutionG1, in.Compile)
	if err != nil {
		return v8Components{}, err
	}
	inventory, err := sourceinventory.EmitV8(in.Contracts.Project(), base.Artifact, in.Sources)
	if err != nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.inventory:%w", err)
	}
	evidence, err := gopackageadapter.EvidenceFromV8(in.Documents, base.Resolution, in.Contracts.Project(), base.Artifact, inventory)
	if err != nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.evidence:%w", err)
	}
	detail, err := gopackageadapter.Convert(base.Resolution, base.Packages, evidence)
	if err != nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.package_model:%w", err)
	}
	merged, err := mergeV8(base.Artifact, inventory)
	if err != nil {
		return v8Components{}, err
	}
	packageDetail, err := packagedetailemitter.EmitV4(merged, detail)
	if err != nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.package_detail:%w", err)
	}
	declarations, err := gopackagev3adapter.ConvertV4(base.Resolution, base.Packages, packageDetail)
	if err != nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.ownership:%w", err)
	}
	packageV4, err := packagev3instance.EmitV4(packagev3instance.InputsV4{Contracts: in.Contracts, PackageV2: packageDetail, Declarations: declarations})
	if err != nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.package_v4:%w", err)
	}
	closure, err := goofflineclosure.Load(goofflineclosure.Input{ProjectRoot: in.Dependency.ProjectRoot, ProxyRoot: in.Dependency.ProxyRoot, Module: in.Dependency.Module, Version: in.Dependency.Version, LocalFrom: in.Dependency.LocalFrom, LocalTo: in.Dependency.LocalTo, Resolution: base.Resolution})
	if err != nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.dependency:%w", err)
	}
	closure, err = goprojectdependency.BindMetadataV8(closure, packageV4)
	if err != nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.dependency_binding:%w", err)
	}
	dependency, err := dependencyemitter.Emit(in.Contracts.Dependency(), closure)
	if err != nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.dependency_emit:%w", err)
	}
	plan, err := goconfigurationadapter.ResolveNeutralV8(ctx, goconfigurationadapter.NeutralInputV8{CanonicalG1: base.CanonicalG1, Compile: goconfigurationadapter.Compile(in.Compile), Contracts: in.Contracts, PackageV2: packageDetail, PackageV4: packageV4, Fields: in.Selection.Fields, Runtime: in.Selection.Runtime, Units: in.Selection.Units})
	if err != nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.configuration_adapter:%w", err)
	}
	model, bound, err := goconfigurationadapter.Models(plan)
	if err != nil {
		return v8Components{}, err
	}
	baseInput := configurationinstance.V3BaseInput{Contracts: in.Contracts, PackageV2: packageDetail, PackageV4: packageV4, Model: model}
	configurationBase, err := configurationinstance.EmitV3Base(baseInput)
	if err != nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.configuration_base:%w", err)
	}
	baseInput.Artifact = configurationBase
	configurationInput := configurationinstance.V3Input{Contracts: in.Contracts, Base: baseInput, Model: bound}
	configurationV3, err := configurationinstance.EmitV3(configurationInput)
	if err != nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.configuration_v3:%w", err)
	}
	configurationInput.Artifact = configurationV3
	if err = configurationinstance.ValidateV3(configurationInput); err != nil {
		return v8Components{}, fmt.Errorf("go_upb05_pipeline.configuration_validate:%w", err)
	}
	return v8Components{Construction: clone(base.CanonicalG1), Execution: clone(base.Execution), ProjectBase: clone(base.Artifact), Inventory: clone(inventory), PackageDetail: clone(packageDetail), PackageV4: clone(packageV4), Dependency: clone(dependency), ConfigurationBase: clone(configurationBase), ConfigurationV3: clone(configurationV3), Base: base, ConfigurationInput: configurationInput}, nil
}

func mergeV8(project, inventory []byte) (wire.Envelope, error) {
	p, err := wire.Decode(project)
	if err != nil {
		return wire.Envelope{}, err
	}
	s, err := wire.Decode(inventory)
	if err != nil {
		return wire.Envelope{}, err
	}
	for id, entity := range s.Entities {
		if entity.Schema == mustWireID("12") || entity.Schema == mustWireID("13") {
			continue
		}
		if old, exists := p.Entities[id]; exists {
			a, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{id: old}})
			b, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{id: entity}})
			if !bytes.Equal(a, b) {
				return wire.Envelope{}, fmt.Errorf("go_upb05_pipeline.inventory_collision:%s", id)
			}
		}
		p.Entities[id] = entity
	}
	return p, nil
}

func mustWireID(short string) wire.ID {
	for len(short) < 32 {
		short = "0" + short
	}
	id, err := wire.ParseID(short)
	if err != nil {
		panic(err)
	}
	return id
}
