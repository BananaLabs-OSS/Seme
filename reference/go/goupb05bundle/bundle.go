// Package goupb05bundle authenticates and reconstructs the public eleven-file
// UPB05 bundle, including its intentionally omitted structural intermediates.
package goupb05bundle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"seme.local/reference/configurationinstance"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationadapter"
	"seme.local/reference/goprojector"
	"seme.local/reference/projectdependencyinstance"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/projectv5instance"
	"seme.local/reference/projectv6instance"
	"seme.local/reference/projectv7instance"
)

type Contracts struct {
	V1, V2, V3 contractcatalog.ProjectContractSet
	V4         contractcatalog.ProjectContractSetV4
	V5         contractcatalog.ProjectContractSetV5
	V6         contractcatalog.ProjectContractSetV6
	V7         contractcatalog.ProjectContractSetV7
	ProjectV2  contractcatalog.Contract
}
type Artifacts struct{ Construction, ProjectV1, InventoryV2, PackageV2, ProjectV3, DependencyV1, ProjectV4, PackageV3, ProjectV5, ConfigurationV2, ProjectV7 []byte }
type Input struct {
	Contracts Contracts
	Artifacts Artifacts
	Manifest  []byte
	Selection goconfigurationadapter.Selection
	Compile   goconfigurationadapter.Compile
}
type Result struct {
	Artifacts                  Artifacts
	ConfigurationV1, ProjectV6 []byte
	ProjectV5Input             projectv5instance.Inputs
	BoundInput                 configurationinstance.BoundInput
	ProjectV7Input             projectv7instance.Inputs
}

func Load(ctx context.Context, in Input) (Result, error) {
	if !in.Contracts.V1.Validated() || !in.Contracts.V2.Validated() || !in.Contracts.V3.Validated() || !in.Contracts.V4.Validated() || !in.Contracts.V5.Validated() || !in.Contracts.V6.Validated() || !in.Contracts.V7.Validated() || in.Compile == nil {
		return Result{}, fmt.Errorf("go_upb05_bundle.contracts")
	}
	a := in.Artifacts
	if !validManifest(in.Manifest, a) {
		return Result{}, fmt.Errorf("go_upb05_bundle.manifest")
	}
	for name, b := range map[string][]byte{"construction": a.Construction, "project-v1": a.ProjectV1, "inventory-v2": a.InventoryV2, "package-v2": a.PackageV2, "project-v3": a.ProjectV3, "dependency-v1": a.DependencyV1, "project-v4": a.ProjectV4, "package-v3": a.PackageV3, "project-v5": a.ProjectV5, "configuration-v2": a.ConfigurationV2, "project-v7": a.ProjectV7} {
		if len(b) == 0 {
			return Result{}, fmt.Errorf("go_upb05_bundle.missing:%s", name)
		}
	}
	v3 := projectgraphinstance.Inputs{Contracts: in.Contracts.V3, ProjectV2: in.Contracts.ProjectV2, Project: a.ProjectV1, Inventory: a.InventoryV2, PackageGraph: a.PackageV2, Composed: a.ProjectV3}
	if err := projectgraphinstance.Validate(v3); err != nil {
		return Result{}, fmt.Errorf("go_upb05_bundle.v3:%w", err)
	}
	v4 := projectdependencyinstance.Inputs{Contracts: in.Contracts.V4, ProjectV3: v3, Dependency: a.DependencyV1, Composed: a.ProjectV4}
	if err := projectdependencyinstance.Validate(v4); err != nil {
		return Result{}, fmt.Errorf("go_upb05_bundle.v4:%w", err)
	}
	v5 := projectv5instance.Inputs{Contracts: in.Contracts.V5, ProjectV4: v4, PackageV2: a.PackageV2, PackageV3: a.PackageV3, Composed: a.ProjectV5}
	if err := projectv5instance.Validate(v5); err != nil {
		return Result{}, fmt.Errorf("go_upb05_bundle.v5:%w", err)
	}
	plan, err := goconfigurationadapter.Resolve(ctx, goconfigurationadapter.Input{CanonicalG1: a.Construction, Compile: in.Compile, Contracts: in.Contracts.V5, PackageV2: a.PackageV2, PackageV3: a.PackageV3, Fields: in.Selection.Fields, Runtime: in.Selection.Runtime, Units: in.Selection.Units})
	if err != nil {
		return Result{}, fmt.Errorf("go_upb05_bundle.selection:%w", err)
	}
	baseModel, boundModel, err := goconfigurationadapter.Models(plan)
	if err != nil {
		return Result{}, err
	}
	c1in := configurationinstance.Input{Contracts: in.Contracts.V6, ProjectV5: v5, Model: baseModel}
	c1, err := configurationinstance.EmitBindableBase(c1in)
	if err != nil {
		return Result{}, err
	}
	c1in.Artifact = c1
	p6in := projectv6instance.Inputs{Contracts: in.Contracts.V6, ProjectV5: v5, Configuration: c1in}
	p6, err := projectv6instance.EmitBindable(p6in)
	if err != nil {
		return Result{}, err
	}
	p6in.Composed = p6
	bound := configurationinstance.BoundInput{Contracts: in.Contracts.V7, Base: c1in, Model: boundModel}
	wantC2, err := configurationinstance.EmitBound(bound)
	if err != nil || !bytes.Equal(wantC2, a.ConfigurationV2) {
		return Result{}, fmt.Errorf("go_upb05_bundle.configuration_v2")
	}
	bound.Artifact = a.ConfigurationV2
	p7in := projectv7instance.Inputs{Contracts: in.Contracts.V7, ProjectV6: p6in, BoundConfiguration: bound, Composed: a.ProjectV7}
	if err = projectv7instance.Validate(p7in); err != nil {
		return Result{}, fmt.Errorf("go_upb05_bundle.v7:%w", err)
	}
	wantP7, err := projectv7instance.Emit(projectv7instance.Inputs{Contracts: in.Contracts.V7, ProjectV6: p6in, BoundConfiguration: bound})
	if err != nil || !bytes.Equal(wantP7, a.ProjectV7) {
		return Result{}, fmt.Errorf("go_upb05_bundle.project_v7_reproduction")
	}
	return Result{Artifacts: cloneArtifacts(a), ConfigurationV1: append([]byte(nil), c1...), ProjectV6: append([]byte(nil), p6...), ProjectV5Input: v5, BoundInput: bound, ProjectV7Input: p7in}, nil
}

func validManifest(got []byte, a Artifacts) bool {
	files := []struct {
		name string
		data []byte
	}{{"construction.g1", a.Construction}, {"project-v1.seme", a.ProjectV1}, {"inventory-v2.seme", a.InventoryV2}, {"package-v2.seme", a.PackageV2}, {"project-v3.seme", a.ProjectV3}, {"dependency-v1.seme", a.DependencyV1}, {"project-v4.seme", a.ProjectV4}, {"package-v3.seme", a.PackageV3}, {"project-v5.seme", a.ProjectV5}, {"configuration-v2.seme", a.ConfigurationV2}, {"project-v7.seme", a.ProjectV7}}
	want := []byte("seme-go-upb05-bundle-v1\n")
	for _, f := range files {
		s := sha256.Sum256(f.data)
		want = append(want, []byte(f.name+" "+hex.EncodeToString(s[:])+"\n")...)
	}
	return bytes.Equal(want, got)
}
func Project(r Result) (map[string][]byte, error) {
	if err := projectv7instance.Validate(r.ProjectV7Input); err != nil {
		return nil, err
	}
	return goprojector.ProjectPackagesV3(r.Artifacts.Construction, r.ProjectV5Input.Contracts, r.Artifacts.PackageV2, r.Artifacts.PackageV3)
}
func cloneArtifacts(a Artifacts) Artifacts {
	return Artifacts{append([]byte(nil), a.Construction...), append([]byte(nil), a.ProjectV1...), append([]byte(nil), a.InventoryV2...), append([]byte(nil), a.PackageV2...), append([]byte(nil), a.ProjectV3...), append([]byte(nil), a.DependencyV1...), append([]byte(nil), a.ProjectV4...), append([]byte(nil), a.PackageV3...), append([]byte(nil), a.ProjectV5...), append([]byte(nil), a.ConfigurationV2...), append([]byte(nil), a.ProjectV7...)}
}
