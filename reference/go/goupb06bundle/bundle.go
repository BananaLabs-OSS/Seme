// Package goupb06bundle authenticates and reconstructs a Project-v9 bundle.
package goupb06bundle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationadapter"
	"seme.local/reference/goupb05bundle"
	"seme.local/reference/projectv9instance"
	"seme.local/reference/resourceinstance"
)

type Artifacts struct{ Construction, Execution, ProjectBase, Inventory, PackageDetail, PackageV4, Dependency, ConfigurationV3, ProjectV8, Resource, ProjectV9 []byte }
type Input struct {
	Contracts contractcatalog.ProjectContractSetV9
	V8        contractcatalog.ProjectContractSetV8
	Artifacts Artifacts
	Manifest  []byte
	Selection goconfigurationadapter.Selection
	Compile   goconfigurationadapter.Compile
	Blobs     map[[32]byte][]byte
}
type Result struct {
	Artifacts Artifacts
	Base      goupb05bundle.V8Result
	Resource  resourceinstance.Inputs
	Project   projectv9instance.Inputs
	Blobs     map[[32]byte][]byte
}

func Load(ctx context.Context, in Input) (Result, error) {
	if !in.Contracts.Validated() || !in.V8.Validated() || !validManifest(in.Manifest, in.Artifacts, in.Blobs) {
		return Result{}, fmt.Errorf("go_upb06_bundle.authentication")
	}
	a := in.Artifacts
	v8a := goupb05bundle.V8Artifacts{Construction: a.Construction, Execution: a.Execution, ProjectBase: a.ProjectBase, Inventory: a.Inventory, PackageDetail: a.PackageDetail, PackageV4: a.PackageV4, Dependency: a.Dependency, ConfigurationV3: a.ConfigurationV3, ProjectV8: a.ProjectV8}
	base, err := goupb05bundle.LoadV8(ctx, goupb05bundle.V8Input{Contracts: in.V8, Artifacts: v8a, Manifest: v8Manifest(v8a), Selection: in.Selection, Compile: in.Compile})
	if err != nil {
		return Result{}, err
	}
	model, err := resourceinstance.ModelFromArtifact(a.Resource)
	if err != nil {
		return Result{}, err
	}
	ri := resourceinstance.Inputs{Contracts: in.Contracts, ProjectV8: base.ProjectV8Input, Model: model, Artifact: a.Resource}
	if err = resourceinstance.Validate(ri); err != nil {
		return Result{}, err
	}
	if err = resourceinstance.VerifyDetached(ri, in.Blobs); err != nil {
		return Result{}, err
	}
	pi := projectv9instance.Inputs{Contracts: in.Contracts, ProjectV8: base.ProjectV8Input, Resource: ri, Composed: a.ProjectV9}
	if err = projectv9instance.Validate(pi); err != nil {
		return Result{}, err
	}
	return Result{Artifacts: clone(a), Base: base, Resource: ri, Project: pi, Blobs: cloneBlobs(in.Blobs)}, nil
}
func Project(r Result) (map[string][]byte, error) { return goupb05bundle.ProjectV8(r.Base) }
func validManifest(got []byte, a Artifacts, blobs map[[32]byte][]byte) bool {
	want := []byte("seme-go-upb06-bundle-v1\n")
	for _, f := range files(a) {
		s := sha256.Sum256(f.data)
		want = append(want, []byte(f.name+" "+hex.EncodeToString(s[:])+"\n")...)
	}
	keys := make([][32]byte, 0, len(blobs))
	for k, b := range blobs {
		if sha256.Sum256(b) != k {
			return false
		}
		keys = append(keys, k)
	}
	sortSums(keys)
	for _, k := range keys {
		h := hex.EncodeToString(k[:])
		want = append(want, []byte("blobs/"+h+" "+h+"\n")...)
	}
	return bytes.Equal(want, got)
}

type named struct {
	name string
	data []byte
}

func files(a Artifacts) []named {
	return []named{{"construction-v36.g1", a.Construction}, {"execution-v36.seme", a.Execution}, {"project-base-v8.seme", a.ProjectBase}, {"inventory-v8.seme", a.Inventory}, {"package-detail-v4.seme", a.PackageDetail}, {"package-v4.seme", a.PackageV4}, {"dependency-v1.seme", a.Dependency}, {"configuration-v3.seme", a.ConfigurationV3}, {"project-v8.seme", a.ProjectV8}, {"resource-v1.seme", a.Resource}, {"project-v9.seme", a.ProjectV9}}
}
func v8Manifest(a goupb05bundle.V8Artifacts) []byte {
	r := []byte("seme-go-upb05-bundle-v2\n")
	for _, f := range files(Artifacts{Construction: a.Construction, Execution: a.Execution, ProjectBase: a.ProjectBase, Inventory: a.Inventory, PackageDetail: a.PackageDetail, PackageV4: a.PackageV4, Dependency: a.Dependency, ConfigurationV3: a.ConfigurationV3, ProjectV8: a.ProjectV8})[:9] {
		s := sha256.Sum256(f.data)
		r = append(r, []byte(f.name+" "+hex.EncodeToString(s[:])+"\n")...)
	}
	return r
}
func sortSums(x [][32]byte) {
	for i := 1; i < len(x); i++ {
		for j := i; j > 0 && bytes.Compare(x[j][:], x[j-1][:]) < 0; j-- {
			x[j], x[j-1] = x[j-1], x[j]
		}
	}
}
func clone(a Artifacts) Artifacts {
	p := func(x []byte) []byte { return append([]byte(nil), x...) }
	return Artifacts{p(a.Construction), p(a.Execution), p(a.ProjectBase), p(a.Inventory), p(a.PackageDetail), p(a.PackageV4), p(a.Dependency), p(a.ConfigurationV3), p(a.ProjectV8), p(a.Resource), p(a.ProjectV9)}
}
func cloneBlobs(x map[[32]byte][]byte) map[[32]byte][]byte {
	o := map[[32]byte][]byte{}
	for k, v := range x {
		o[k] = append([]byte(nil), v...)
	}
	return o
}
