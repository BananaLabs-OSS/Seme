// Package goupb05pipeline adds authenticated configuration and initialization
// bindings to the cumulative Go project pipeline.
package goupb05pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"seme.local/reference/configurationinstance"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationadapter"
	"seme.local/reference/goupb04pipeline"
	"seme.local/reference/projectdependencyinstance"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/projectv5instance"
	"seme.local/reference/projectv6instance"
	"seme.local/reference/projectv7instance"
)

type Selection struct {
	Fields  []goconfigurationadapter.FieldSelection
	Runtime []goconfigurationadapter.RuntimeInputSelection
	Units   []goconfigurationadapter.UnitSelection
}
type Input struct {
	Base      goupb04pipeline.Input
	V6        contractcatalog.ProjectContractSetV6
	V7        contractcatalog.ProjectContractSetV7
	Selection Selection
}
type Result struct {
	Base                                                   goupb04pipeline.Result
	ConfigurationV1, ProjectV6, ConfigurationV2, ProjectV7 []byte
}

func Build(ctx context.Context, in Input) (Result, error) {
	if !in.V6.Validated() || !in.V7.Validated() {
		return Result{}, fmt.Errorf("go_upb05_pipeline.contracts")
	}
	base, err := goupb04pipeline.Build(ctx, in.Base)
	if err != nil {
		return Result{}, err
	}
	projectV5 := v5Inputs(in.Base, base)
	plan, err := goconfigurationadapter.Resolve(ctx, goconfigurationadapter.Input{CanonicalG1: base.Base.Base.CanonicalG1, Packages: base.Base.Base.Packages, Compile: goconfigurationadapter.Compile(in.Base.Base.Base.Compile), Contracts: in.Base.Contracts, PackageV2: base.Base.Base.PackageV2, PackageV3: base.PackageV3, Fields: in.Selection.Fields, Runtime: in.Selection.Runtime, Units: in.Selection.Units})
	if err != nil {
		return Result{}, fmt.Errorf("go_upb05_pipeline.adapter:%w", err)
	}
	model, bound, err := goconfigurationadapter.Models(plan)
	if err != nil {
		return Result{}, err
	}
	c1in := configurationinstance.Input{Contracts: in.V6, ProjectV5: projectV5, Model: model}
	c1, err := configurationinstance.EmitBindableBase(c1in)
	if err != nil {
		return Result{}, fmt.Errorf("go_upb05_pipeline.configuration_v1:%w", err)
	}
	c1in.Artifact = c1
	p6in := projectv6instance.Inputs{Contracts: in.V6, ProjectV5: projectV5, Configuration: c1in}
	p6, err := projectv6instance.EmitBindable(p6in)
	if err != nil {
		return Result{}, fmt.Errorf("go_upb05_pipeline.project_v6:%w", err)
	}
	p6in.Composed = p6
	bIn := configurationinstance.BoundInput{Contracts: in.V7, Base: c1in, Model: bound}
	c2, err := configurationinstance.EmitBound(bIn)
	if err != nil {
		return Result{}, fmt.Errorf("go_upb05_pipeline.configuration_v2:%w", err)
	}
	bIn.Artifact = c2
	p7in := projectv7instance.Inputs{Contracts: in.V7, ProjectV6: p6in, BoundConfiguration: bIn}
	p7, err := projectv7instance.Emit(p7in)
	if err != nil {
		return Result{}, fmt.Errorf("go_upb05_pipeline.project_v7:%w", err)
	}
	p7in.Composed = p7
	if err = configurationinstance.ValidateBindableBase(c1in); err != nil {
		return Result{}, err
	}
	if err = projectv6instance.ValidateBindable(p6in); err != nil {
		return Result{}, err
	}
	if err = configurationinstance.ValidateBound(bIn); err != nil {
		return Result{}, err
	}
	if err = projectv7instance.Validate(p7in); err != nil {
		return Result{}, err
	}
	return Result{Base: base, ConfigurationV1: clone(c1), ProjectV6: clone(p6), ConfigurationV2: clone(c2), ProjectV7: clone(p7)}, nil
}

func v5Inputs(in goupb04pipeline.Input, r goupb04pipeline.Result) projectv5instance.Inputs {
	b := r.Base.Base
	v3 := projectgraphinstance.Inputs{Contracts: in.Base.Base.Contracts.V3, ProjectV2: in.Base.Base.Contracts.V2.Project(), Project: b.ProjectV1, Inventory: b.InventoryV2, PackageGraph: b.PackageV2, Composed: b.ProjectV3}
	v4 := projectdependencyinstance.Inputs{Contracts: in.Base.Contracts, ProjectV3: v3, Dependency: r.Base.DependencyV1, Composed: r.Base.ProjectV4}
	return projectv5instance.Inputs{Contracts: in.Contracts, ProjectV4: v4, PackageV2: b.PackageV2, PackageV3: r.PackageV3, Composed: r.ProjectV5}
}

// Publish creates the eleven-artifact public bundle. The bindable v1/v6
// structural intermediates are validated above and embedded transitively in
// Configuration v2/Project v7; they are not separate public bundle members.
func Publish(destination string, r Result) error {
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
	b := r.Base.Base.Base
	files := []struct {
		name string
		data []byte
	}{{"construction.g1", b.CanonicalG1}, {"project-v1.seme", b.ProjectV1}, {"inventory-v2.seme", b.InventoryV2}, {"package-v2.seme", b.PackageV2}, {"project-v3.seme", b.ProjectV3}, {"dependency-v1.seme", r.Base.Base.DependencyV1}, {"project-v4.seme", r.Base.Base.ProjectV4}, {"package-v3.seme", r.Base.PackageV3}, {"project-v5.seme", r.Base.ProjectV5}, {"configuration-v2.seme", r.ConfigurationV2}, {"project-v7.seme", r.ProjectV7}}
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
	manifest := []byte("seme-go-upb05-bundle-v1\n")
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
func clone(x []byte) []byte { return append([]byte(nil), x...) }
