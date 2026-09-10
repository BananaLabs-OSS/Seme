// Package goupb04pipeline assembles the bounded Go UPB-04 evidence chain.
// It extends the authenticated dependency build with complete declaration
// ownership and a Project v5 snapshot; projection consumes only that evidence.
package goupb04pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/gopackagev3adapter"
	"seme.local/reference/goprojectdependency"
	"seme.local/reference/goprojector"
	"seme.local/reference/goupb03pipeline"
	"seme.local/reference/packagecallinstance"
	"seme.local/reference/packagev3instance"
	"seme.local/reference/projectdependencyinstance"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/projectv5instance"
)

type Input struct {
	Base      goupb03pipeline.Input
	Contracts contractcatalog.ProjectContractSetV5
}

type Result struct {
	Base                 goupb03pipeline.Result
	PackageV3, ProjectV5 []byte
}

// Build returns nothing unless every component and cross-component relation
// validates against the authenticated contracts from the same build.
func Build(ctx context.Context, in Input) (Result, error) {
	if !in.Contracts.Validated() {
		return Result{}, fmt.Errorf("go_upb04_pipeline.contracts")
	}
	base, err := goupb03pipeline.Build(ctx, in.Base)
	if err != nil {
		return Result{}, err
	}
	if err = ctx.Err(); err != nil {
		return Result{}, err
	}
	if err = packagecallinstance.Validate(base.Base.PackageV2); err != nil {
		return Result{}, fmt.Errorf("go_upb04_pipeline.calls:%w", err)
	}
	declarations, err := gopackagev3adapter.Convert(base.Base.Resolution, base.Base.Packages, base.Base.PackageV2)
	if err != nil {
		return Result{}, fmt.Errorf("go_upb04_pipeline.ownership:%w", err)
	}
	p3, err := packagev3instance.Emit(packagev3instance.Inputs{Contracts: in.Contracts, PackageV2: base.Base.PackageV2, Declarations: declarations})
	if err != nil {
		return Result{}, fmt.Errorf("go_upb04_pipeline.package_v3:%w", err)
	}
	v3 := projectgraphinstance.Inputs{Contracts: in.Base.Base.Contracts.V3, ProjectV2: in.Base.Base.Contracts.V2.Project(), Project: base.Base.ProjectV1, Inventory: base.Base.InventoryV2, PackageGraph: base.Base.PackageV2, Composed: base.Base.ProjectV3}
	v4 := projectdependencyinstance.Inputs{Contracts: in.Base.Contracts, ProjectV3: v3, Dependency: base.DependencyV1, Composed: base.ProjectV4}
	if err = goprojectdependency.Validate(v4); err != nil {
		return Result{}, fmt.Errorf("go_upb04_pipeline.applicability:%w", err)
	}
	v5in := projectv5instance.Inputs{Contracts: in.Contracts, ProjectV4: v4, PackageV2: base.Base.PackageV2, PackageV3: p3}
	p5, err := projectv5instance.Emit(v5in)
	if err != nil {
		return Result{}, fmt.Errorf("go_upb04_pipeline.project_v5:%w", err)
	}
	v5in.Composed = p5
	if err = projectv5instance.Validate(v5in); err != nil {
		return Result{}, fmt.Errorf("go_upb04_pipeline.project_v5_validate:%w", err)
	}
	return Result{Base: base, PackageV3: clone(p3), ProjectV5: clone(p5)}, nil
}

// Project derives ordinary package sources from authenticated Package v3
// ownership. The caller cannot supply or override ownership assignments.
func Project(r Result, contracts contractcatalog.ProjectContractSetV5) (map[string][]byte, error) {
	out, err := goprojector.ProjectPackagesV3(r.Base.Base.CanonicalG1, contracts, r.Base.Base.PackageV2, r.PackageV3)
	if err != nil {
		return nil, fmt.Errorf("go_upb04_pipeline.project:%w", err)
	}
	return cloneMap(out), nil
}

// Publish reserves a new directory and makes the bundle complete only by
// writing its digest manifest after all nine immutable artifacts.
func Publish(destination string, r Result) error {
	if !filepath.IsAbs(destination) || filepath.Clean(destination) != destination || filepath.Base(destination) == "." {
		return fmt.Errorf("go_upb04_pipeline.destination")
	}
	parent := filepath.Dir(destination)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("go_upb04_pipeline.parent")
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil || resolved != parent {
		return fmt.Errorf("go_upb04_pipeline.parent_symlink")
	}
	files := []struct {
		name string
		data []byte
	}{
		{"construction.g1", r.Base.Base.CanonicalG1},
		{"project-v1.seme", r.Base.Base.ProjectV1},
		{"inventory-v2.seme", r.Base.Base.InventoryV2},
		{"package-v2.seme", r.Base.Base.PackageV2},
		{"project-v3.seme", r.Base.Base.ProjectV3},
		{"dependency-v1.seme", r.Base.DependencyV1},
		{"project-v4.seme", r.Base.ProjectV4},
		{"package-v3.seme", r.PackageV3},
		{"project-v5.seme", r.ProjectV5},
	}
	for _, f := range files {
		if len(f.data) == 0 {
			return fmt.Errorf("go_upb04_pipeline.artifact_empty:%s", f.name)
		}
	}
	if err = os.Mkdir(destination, 0700); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("go_upb04_pipeline.destination_exists")
		}
		return err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(destination)
		}
	}()
	manifest := []byte("seme-go-upb04-bundle-v1\n")
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
func cloneMap(in map[string][]byte) map[string][]byte {
	out := make(map[string][]byte, len(in))
	for k, v := range in {
		out[k] = clone(v)
	}
	return out
}
