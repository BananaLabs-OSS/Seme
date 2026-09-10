// Package goupb08bundle authenticates a closed Project-v11 transport bundle.
package goupb08bundle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationadapter"
	"seme.local/reference/godurableadapter"
	"seme.local/reference/goorderedtransportadapter"
	"seme.local/reference/goupb07bundle"
	"seme.local/reference/orderedtransportinstance"
	"seme.local/reference/projectv11instance"
)

type Artifacts struct {
	Base                  goupb07bundle.Artifacts
	Transport, ProjectV11 []byte
}
type Input struct {
	Contracts          contractcatalog.ProjectContractSetV11
	V10                contractcatalog.ProjectContractSetV10
	V9                 contractcatalog.ProjectContractSetV9
	V8                 contractcatalog.ProjectContractSetV8
	Artifacts          Artifacts
	Manifest           []byte
	Selection          goconfigurationadapter.Selection
	DurableSelection   godurableadapter.Selection
	TransportSelection goorderedtransportadapter.Selection
	Compile            goconfigurationadapter.Compile
	Blobs              map[[32]byte][]byte
}
type Result struct {
	Artifacts Artifacts
	Base      goupb07bundle.Result
	Transport orderedtransportinstance.Inputs
	Project   projectv11instance.Inputs
	Blobs     map[[32]byte][]byte
}

// Project delegates ordinary Go projection to the authenticated cumulative
// Project-v10 graph retained by this Project-v11 bundle.
func Project(r Result) (map[string][]byte, error) { return goupb07bundle.Project(r.Base) }

// Load returns no authority until every artifact, detached blob, prior layer,
// independent transport selection, and Project-v11 binding reproduces exactly.
func Load(ctx context.Context, in Input) (Result, error) {
	if !in.Contracts.Validated() || !in.V10.Validated() || !in.V9.Validated() || !in.V8.Validated() || !validManifest(in.Manifest, in.Artifacts, in.Blobs) {
		return Result{}, fmt.Errorf("go_upb08_bundle.authentication")
	}
	b, err := goupb07bundle.Load(ctx, goupb07bundle.Input{Contracts: in.V10, V9: in.V9, V8: in.V8, Artifacts: in.Artifacts.Base, Manifest: v7Manifest(in.Artifacts.Base, in.Blobs), Selection: in.Selection, DurableSelection: in.DurableSelection, Compile: in.Compile, Blobs: in.Blobs})
	if err != nil {
		return Result{}, err
	}
	model, err := goorderedtransportadapter.Resolve(b.Project, in.TransportSelection)
	if err != nil {
		return Result{}, fmt.Errorf("go_upb08_bundle.transport_selection:%w", err)
	}
	ti := orderedtransportinstance.Inputs{Contracts: in.Contracts, ProjectV10: b.Project, Model: model, Artifact: in.Artifacts.Transport}
	if err = orderedtransportinstance.Validate(ti); err != nil {
		return Result{}, err
	}
	pi := projectv11instance.Inputs{Contracts: in.Contracts, ProjectV10: b.Project, Transport: ti, Composed: in.Artifacts.ProjectV11}
	if err = projectv11instance.Validate(pi); err != nil {
		return Result{}, err
	}
	return Result{Artifacts: clone(in.Artifacts), Base: b, Transport: ti, Project: pi, Blobs: cloneBlobs(in.Blobs)}, nil
}

type named struct {
	name string
	data []byte
}

func files(a Artifacts) []named {
	return []named{{"construction-v36.g1", a.Base.Construction}, {"execution-v36.seme", a.Base.Execution}, {"project-base-v8.seme", a.Base.ProjectBase}, {"inventory-v8.seme", a.Base.Inventory}, {"package-detail-v4.seme", a.Base.PackageDetail}, {"package-v4.seme", a.Base.PackageV4}, {"dependency-v1.seme", a.Base.Dependency}, {"configuration-v3.seme", a.Base.ConfigurationV3}, {"project-v8.seme", a.Base.ProjectV8}, {"resource-v1.seme", a.Base.Resource}, {"project-v9.seme", a.Base.ProjectV9}, {"durable-state-v1.seme", a.Base.Durable}, {"source-presentation-v1.seme", a.Base.Presentation}, {"project-v10.seme", a.Base.ProjectV10}, {"ordered-transport-v1.seme", a.Transport}, {"project-v11.seme", a.ProjectV11}}
}
func manifest(header string, fs []named, blobs map[[32]byte][]byte) ([]byte, bool) {
	r := []byte(header + "\n")
	for _, f := range fs {
		if len(f.data) == 0 {
			return nil, false
		}
		s := sha256.Sum256(f.data)
		r = append(r, []byte(f.name+" "+hex.EncodeToString(s[:])+"\n")...)
	}
	keys := make([][32]byte, 0, len(blobs))
	for k, b := range blobs {
		if len(b) == 0 || sha256.Sum256(b) != k {
			return nil, false
		}
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i][:], keys[j][:]) < 0 })
	for _, k := range keys {
		h := hex.EncodeToString(k[:])
		r = append(r, []byte("blobs/"+h+" "+h+"\n")...)
	}
	return r, true
}
func validManifest(got []byte, a Artifacts, b map[[32]byte][]byte) bool {
	want, ok := manifest("seme-go-upb08-bundle-v1", files(a), b)
	return ok && bytes.Equal(got, want)
}
func v7Manifest(a goupb07bundle.Artifacts, b map[[32]byte][]byte) []byte {
	r, _ := manifest("seme-go-upb07-bundle-v1", files(Artifacts{Base: a})[:14], b)
	return r
}
func clone(a Artifacts) Artifacts {
	p := func(x []byte) []byte { return append([]byte(nil), x...) }
	b := a.Base
	b.Construction = p(b.Construction)
	b.Execution = p(b.Execution)
	b.ProjectBase = p(b.ProjectBase)
	b.Inventory = p(b.Inventory)
	b.PackageDetail = p(b.PackageDetail)
	b.PackageV4 = p(b.PackageV4)
	b.Dependency = p(b.Dependency)
	b.ConfigurationV3 = p(b.ConfigurationV3)
	b.ProjectV8 = p(b.ProjectV8)
	b.Resource = p(b.Resource)
	b.ProjectV9 = p(b.ProjectV9)
	b.Durable = p(b.Durable)
	b.Presentation = p(b.Presentation)
	b.ProjectV10 = p(b.ProjectV10)
	return Artifacts{Base: b, Transport: p(a.Transport), ProjectV11: p(a.ProjectV11)}
}
func cloneBlobs(x map[[32]byte][]byte) map[[32]byte][]byte {
	o := map[[32]byte][]byte{}
	for k, v := range x {
		o[k] = append([]byte(nil), v...)
	}
	return o
}
