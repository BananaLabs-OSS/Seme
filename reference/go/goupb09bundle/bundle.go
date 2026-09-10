// Package goupb09bundle authenticates a closed, source-free Project-v12
// controlled-effects bundle and every cumulative predecessor.
package goupb09bundle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/controlledeffectsinstance"
	"seme.local/reference/goconfigurationadapter"
	"seme.local/reference/gocontrolledeffectsadapter"
	"seme.local/reference/godurableadapter"
	"seme.local/reference/goorderedtransportadapter"
	"seme.local/reference/goupb08bundle"
	"seme.local/reference/projectv12instance"
)

type Artifacts struct {
	Base                          goupb08bundle.Artifacts
	ControlledEffects, ProjectV12 []byte
	ReplayAuthority               []byte
}
type Input struct {
	Contracts          contractcatalog.ProjectContractSetV12
	V11                contractcatalog.ProjectContractSetV11
	V10                contractcatalog.ProjectContractSetV10
	V9                 contractcatalog.ProjectContractSetV9
	V8                 contractcatalog.ProjectContractSetV8
	Artifacts          Artifacts
	Manifest           []byte
	Selection          goconfigurationadapter.Selection
	DurableSelection   godurableadapter.Selection
	TransportSelection goorderedtransportadapter.Selection
	EffectsSelection   gocontrolledeffectsadapter.Selection
	Compile            goconfigurationadapter.Compile
	Blobs              map[[32]byte][]byte
}
type Result struct {
	Artifacts Artifacts
	Base      goupb08bundle.Result
	Effects   controlledeffectsinstance.Inputs
	Project   projectv12instance.Inputs
	Blobs     map[[32]byte][]byte
}

func Project(r Result) (map[string][]byte, error) { return goupb08bundle.Project(r.Base) }

func Load(ctx context.Context, in Input) (Result, error) {
	if !in.Contracts.Validated() || !in.V11.Validated() || !in.V10.Validated() || !in.V9.Validated() || !in.V8.Validated() || !validManifest(in.Manifest, in.Artifacts, in.Blobs) {
		return Result{}, fmt.Errorf("go_upb09_bundle.authentication")
	}
	b, err := goupb08bundle.Load(ctx, goupb08bundle.Input{Contracts: in.V11, V10: in.V10, V9: in.V9, V8: in.V8, Artifacts: in.Artifacts.Base, Manifest: v8Manifest(in.Artifacts.Base, in.Blobs), Selection: in.Selection, DurableSelection: in.DurableSelection, TransportSelection: in.TransportSelection, Compile: in.Compile, Blobs: in.Blobs})
	if err != nil {
		return Result{}, err
	}
	model, err := gocontrolledeffectsadapter.Resolve(b.Project, in.EffectsSelection)
	if err != nil {
		return Result{}, fmt.Errorf("go_upb09_bundle.effects_selection:%w", err)
	}
	replay, err := parseReplay(in.Artifacts.ReplayAuthority, model.Bounds)
	if err != nil {
		return Result{}, err
	}
	model.Replay = replay
	ei := controlledeffectsinstance.Inputs{Contracts: in.Contracts, ProjectV11: b.Project, Model: model, Artifact: in.Artifacts.ControlledEffects}
	if err = controlledeffectsinstance.Validate(ei); err != nil {
		return Result{}, err
	}
	pi := projectv12instance.Inputs{Contracts: in.Contracts, ProjectV11: b.Project, Effects: ei, Composed: in.Artifacts.ProjectV12}
	if err = projectv12instance.Validate(pi); err != nil {
		return Result{}, err
	}
	return Result{Artifacts: clone(in.Artifacts), Base: b, Effects: ei, Project: pi, Blobs: cloneBlobs(in.Blobs)}, nil
}

type named struct {
	name string
	data []byte
}

func files(a Artifacts) []named {
	b := a.Base.Base
	return []named{{"construction-v36.g1", b.Construction}, {"execution-v36.seme", b.Execution}, {"project-base-v8.seme", b.ProjectBase}, {"inventory-v8.seme", b.Inventory}, {"package-detail-v4.seme", b.PackageDetail}, {"package-v4.seme", b.PackageV4}, {"dependency-v1.seme", b.Dependency}, {"configuration-v3.seme", b.ConfigurationV3}, {"project-v8.seme", b.ProjectV8}, {"resource-v1.seme", b.Resource}, {"project-v9.seme", b.ProjectV9}, {"durable-state-v1.seme", b.Durable}, {"source-presentation-v1.seme", b.Presentation}, {"project-v10.seme", b.ProjectV10}, {"ordered-transport-v1.seme", a.Base.Transport}, {"project-v11.seme", a.Base.ProjectV11}, {"controlled-effects-v1.seme", a.ControlledEffects}, {"project-v12.seme", a.ProjectV12}, {"controlled-replay-v1.json", a.ReplayAuthority}}
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
	want, ok := manifest("seme-go-upb09-bundle-v1", files(a), b)
	return ok && bytes.Equal(got, want)
}
func v8Manifest(a goupb08bundle.Artifacts, b map[[32]byte][]byte) []byte {
	x := Artifacts{Base: a}
	r, _ := manifest("seme-go-upb08-bundle-v1", files(x)[:16], b)
	return r
}
func clone(a Artifacts) Artifacts {
	p := func(x []byte) []byte { return bytes.Clone(x) }
	b := a.Base
	c := b.Base
	c.Construction = p(c.Construction)
	c.Execution = p(c.Execution)
	c.ProjectBase = p(c.ProjectBase)
	c.Inventory = p(c.Inventory)
	c.PackageDetail = p(c.PackageDetail)
	c.PackageV4 = p(c.PackageV4)
	c.Dependency = p(c.Dependency)
	c.ConfigurationV3 = p(c.ConfigurationV3)
	c.ProjectV8 = p(c.ProjectV8)
	c.Resource = p(c.Resource)
	c.ProjectV9 = p(c.ProjectV9)
	c.Durable = p(c.Durable)
	c.Presentation = p(c.Presentation)
	c.ProjectV10 = p(c.ProjectV10)
	b.Base = c
	b.Transport = p(b.Transport)
	b.ProjectV11 = p(b.ProjectV11)
	return Artifacts{Base: b, ControlledEffects: p(a.ControlledEffects), ProjectV12: p(a.ProjectV12), ReplayAuthority: p(a.ReplayAuthority)}
}
func cloneBlobs(x map[[32]byte][]byte) map[[32]byte][]byte {
	o := map[[32]byte][]byte{}
	for k, v := range x {
		o[k] = bytes.Clone(v)
	}
	return o
}
