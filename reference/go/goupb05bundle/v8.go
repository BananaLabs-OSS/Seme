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
	"seme.local/reference/projectv8instance"
)

// V8Artifacts is the complete one-run Execution-v36 bundle. Construction is
// reproducible G1; Execution is its separately authenticated canonical wire.
type V8Artifacts struct {
	Construction, Execution, ProjectBase, Inventory, PackageDetail, PackageV4 []byte
	Dependency, ConfigurationV3, ProjectV8                                    []byte
}

type V8Input struct {
	Contracts contractcatalog.ProjectContractSetV8
	Artifacts V8Artifacts
	Manifest  []byte
	Selection goconfigurationadapter.Selection
	Compile   goconfigurationadapter.Compile
}

type V8Result struct {
	Artifacts      V8Artifacts
	Configuration  configurationinstance.V3Input
	ProjectV8Input projectv8instance.Inputs
}

func LoadV8(ctx context.Context, in V8Input) (V8Result, error) {
	if !in.Contracts.Validated() || in.Compile == nil || !validV8Manifest(in.Manifest, in.Artifacts) {
		return V8Result{}, fmt.Errorf("go_upb05_bundle.v8_authentication")
	}
	a := in.Artifacts
	for name, value := range map[string][]byte{"construction": a.Construction, "execution": a.Execution, "project": a.ProjectBase, "inventory": a.Inventory, "package-detail": a.PackageDetail, "package": a.PackageV4, "dependency": a.Dependency, "configuration": a.ConfigurationV3, "complete": a.ProjectV8} {
		if len(value) == 0 {
			return V8Result{}, fmt.Errorf("go_upb05_bundle.v8_missing:%s", name)
		}
	}
	plan, err := goconfigurationadapter.ResolveV8(ctx, goconfigurationadapter.InputV8{CanonicalG1: a.Construction, Compile: in.Compile, Contracts: in.Contracts, PackageV2: a.PackageDetail, PackageV4: a.PackageV4, Fields: in.Selection.Fields, Runtime: in.Selection.Runtime, Units: in.Selection.Units})
	if err != nil {
		return V8Result{}, fmt.Errorf("go_upb05_bundle.v8_selection:%w", err)
	}
	base, bound, err := goconfigurationadapter.Models(plan)
	if err != nil {
		return V8Result{}, err
	}
	baseInput := configurationinstance.V3BaseInput{Contracts: in.Contracts, PackageV2: a.PackageDetail, PackageV4: a.PackageV4, Model: base}
	baseBytes, err := configurationinstance.EmitV3Base(baseInput)
	if err != nil {
		return V8Result{}, err
	}
	baseInput.Artifact = baseBytes
	configuration := configurationinstance.V3Input{Contracts: in.Contracts, Base: baseInput, Model: bound}
	wantConfiguration, err := configurationinstance.EmitV3(configuration)
	if err != nil || !bytes.Equal(wantConfiguration, a.ConfigurationV3) {
		return V8Result{}, fmt.Errorf("go_upb05_bundle.v8_configuration")
	}
	configuration.Artifact = a.ConfigurationV3
	p8 := projectv8instance.Inputs{Contracts: in.Contracts, ProjectBase: a.ProjectBase, Inventory: a.Inventory, PackageV2: a.PackageDetail, PackageV4: a.PackageV4, Dependency: a.Dependency, Construction: a.Execution, Configuration: configuration, Composed: a.ProjectV8}
	if err = projectv8instance.Validate(p8); err != nil {
		return V8Result{}, fmt.Errorf("go_upb05_bundle.v8_project:%w", err)
	}
	wantP8, err := projectv8instance.Emit(projectv8instance.Inputs{Contracts: in.Contracts, ProjectBase: a.ProjectBase, Inventory: a.Inventory, PackageV2: a.PackageDetail, PackageV4: a.PackageV4, Dependency: a.Dependency, Construction: a.Execution, Configuration: configuration})
	if err != nil || !bytes.Equal(wantP8, a.ProjectV8) {
		return V8Result{}, fmt.Errorf("go_upb05_bundle.v8_reproduction")
	}
	return V8Result{Artifacts: cloneV8Artifacts(a), Configuration: configuration, ProjectV8Input: p8}, nil
}

func ProjectV8(r V8Result) (map[string][]byte, error) {
	if err := projectv8instance.Validate(r.ProjectV8Input); err != nil {
		return nil, err
	}
	return goprojector.ProjectPackagesV4(r.Artifacts.Construction, r.ProjectV8Input.Contracts, r.Artifacts.PackageDetail, r.Artifacts.PackageV4)
}

func validV8Manifest(got []byte, a V8Artifacts) bool {
	files := []struct {
		name string
		data []byte
	}{{"construction-v36.g1", a.Construction}, {"execution-v36.seme", a.Execution}, {"project-base-v8.seme", a.ProjectBase}, {"inventory-v8.seme", a.Inventory}, {"package-detail-v4.seme", a.PackageDetail}, {"package-v4.seme", a.PackageV4}, {"dependency-v1.seme", a.Dependency}, {"configuration-v3.seme", a.ConfigurationV3}, {"project-v8.seme", a.ProjectV8}}
	want := []byte("seme-go-upb05-bundle-v2\n")
	for _, file := range files {
		sum := sha256.Sum256(file.data)
		want = append(want, []byte(file.name+" "+hex.EncodeToString(sum[:])+"\n")...)
	}
	return bytes.Equal(want, got)
}

func cloneV8Artifacts(a V8Artifacts) V8Artifacts {
	return V8Artifacts{append([]byte(nil), a.Construction...), append([]byte(nil), a.Execution...), append([]byte(nil), a.ProjectBase...), append([]byte(nil), a.Inventory...), append([]byte(nil), a.PackageDetail...), append([]byte(nil), a.PackageV4...), append([]byte(nil), a.Dependency...), append([]byte(nil), a.ConfigurationV3...), append([]byte(nil), a.ProjectV8...)}
}
