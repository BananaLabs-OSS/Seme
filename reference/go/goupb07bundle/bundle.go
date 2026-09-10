// Package goupb07bundle authenticates and reconstructs a Project-v10 bundle.
package goupb07bundle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/durableinstance"
	"seme.local/reference/goconfigurationadapter"
	"seme.local/reference/godurableadapter"
	"seme.local/reference/goupb06bundle"
	"seme.local/reference/projectv10instance"
	"seme.local/reference/wire"
)

type Artifacts struct {
	Construction, Execution, ProjectBase, Inventory, PackageDetail, PackageV4, Dependency, ConfigurationV3, ProjectV8, Resource, ProjectV9, Durable, ProjectV10 []byte
}

type Input struct {
	Contracts contractcatalog.ProjectContractSetV10
	V9        contractcatalog.ProjectContractSetV9
	V8        contractcatalog.ProjectContractSetV8
	Artifacts Artifacts
	Manifest  []byte
	Selection goconfigurationadapter.Selection
	// DurableSelection is independent authoring evidence. Load resolves it
	// against the authenticated Project-v9 graph and requires it to describe
	// exactly the same model encoded by the durable artifact.
	DurableSelection godurableadapter.Selection
	Compile          goconfigurationadapter.Compile
	Blobs            map[[32]byte][]byte
}

type Result struct {
	Artifacts Artifacts
	Base      goupb06bundle.Result
	Durable   durableinstance.Inputs
	Project   projectv10instance.Inputs
	Blobs     map[[32]byte][]byte
}

// Load rejects before returning any authority unless the closed bundle,
// detached blobs, complete prior chain, durable plan, and v10 binding all
// reproduce byte-for-byte under their exact contract revisions.
func Load(ctx context.Context, in Input) (Result, error) {
	if !in.Contracts.Validated() || !in.V9.Validated() || !in.V8.Validated() || !validManifest(in.Manifest, in.Artifacts, in.Blobs) {
		return Result{}, fmt.Errorf("go_upb07_bundle.authentication")
	}
	a := in.Artifacts
	v9a := goupb06bundle.Artifacts{Construction: a.Construction, Execution: a.Execution, ProjectBase: a.ProjectBase, Inventory: a.Inventory, PackageDetail: a.PackageDetail, PackageV4: a.PackageV4, Dependency: a.Dependency, ConfigurationV3: a.ConfigurationV3, ProjectV8: a.ProjectV8, Resource: a.Resource, ProjectV9: a.ProjectV9}
	base, err := goupb06bundle.Load(ctx, goupb06bundle.Input{Contracts: in.V9, V8: in.V8, Artifacts: v9a, Manifest: v9Manifest(v9a, in.Blobs), Selection: in.Selection, Compile: in.Compile, Blobs: in.Blobs})
	if err != nil {
		return Result{}, err
	}
	model, err := modelFromArtifact(a.Durable)
	if err != nil {
		return Result{}, err
	}
	selected, err := godurableadapter.Resolve(base.Project, in.DurableSelection)
	if err != nil || selected != model {
		return Result{}, fmt.Errorf("go_upb07_bundle.durable_selection")
	}
	di := durableinstance.Inputs{Contracts: in.Contracts, ProjectV9: base.Project, Artifact: a.Durable, Model: model}
	if err = durableinstance.Validate(di); err != nil {
		return Result{}, err
	}
	pi := projectv10instance.Inputs{Contracts: in.Contracts, ProjectV9: base.Project, Durable: di, Composed: a.ProjectV10}
	if err = projectv10instance.Validate(pi); err != nil {
		return Result{}, err
	}
	return Result{Artifacts: clone(a), Base: base, Durable: di, Project: pi, Blobs: cloneBlobs(in.Blobs)}, nil
}

// Project returns ordinary Go package projections from the authenticated
// package graph. Detached resources remain available in Result.Blobs and are
// deliberately not interpreted as source.
func Project(r Result) (map[string][]byte, error) { return goupb06bundle.Project(r.Base) }

type named struct {
	name string
	data []byte
}

func files(a Artifacts) []named {
	return []named{{"construction-v36.g1", a.Construction}, {"execution-v36.seme", a.Execution}, {"project-base-v8.seme", a.ProjectBase}, {"inventory-v8.seme", a.Inventory}, {"package-detail-v4.seme", a.PackageDetail}, {"package-v4.seme", a.PackageV4}, {"dependency-v1.seme", a.Dependency}, {"configuration-v3.seme", a.ConfigurationV3}, {"project-v8.seme", a.ProjectV8}, {"resource-v1.seme", a.Resource}, {"project-v9.seme", a.ProjectV9}, {"durable-state-v1.seme", a.Durable}, {"project-v10.seme", a.ProjectV10}}
}

func validManifest(got []byte, a Artifacts, blobs map[[32]byte][]byte) bool {
	want := []byte("seme-go-upb07-bundle-v1\n")
	for _, f := range files(a) {
		if len(f.data) == 0 {
			return false
		}
		s := sha256.Sum256(f.data)
		want = append(want, []byte(f.name+" "+hex.EncodeToString(s[:])+"\n")...)
	}
	keys := make([][32]byte, 0, len(blobs))
	for k, b := range blobs {
		if len(b) == 0 || sha256.Sum256(b) != k {
			return false
		}
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i][:], keys[j][:]) < 0 })
	for _, k := range keys {
		h := hex.EncodeToString(k[:])
		want = append(want, []byte("blobs/"+h+" "+h+"\n")...)
	}
	return bytes.Equal(want, got)
}

func v9Manifest(a goupb06bundle.Artifacts, blobs map[[32]byte][]byte) []byte {
	x := Artifacts{Construction: a.Construction, Execution: a.Execution, ProjectBase: a.ProjectBase, Inventory: a.Inventory, PackageDetail: a.PackageDetail, PackageV4: a.PackageV4, Dependency: a.Dependency, ConfigurationV3: a.ConfigurationV3, ProjectV8: a.ProjectV8, Resource: a.Resource, ProjectV9: a.ProjectV9}
	r := []byte("seme-go-upb06-bundle-v1\n")
	for _, f := range files(x)[:11] {
		s := sha256.Sum256(f.data)
		r = append(r, []byte(f.name+" "+hex.EncodeToString(s[:])+"\n")...)
	}
	keys := make([][32]byte, 0, len(blobs))
	for k := range blobs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i][:], keys[j][:]) < 0 })
	for _, k := range keys {
		h := hex.EncodeToString(k[:])
		r = append(r, []byte("blobs/"+h+" "+h+"\n")...)
	}
	return r
}

func modelFromArtifact(data []byte) (durableinstance.Model, error) {
	e, err := wire.Decode(data)
	if err != nil {
		return durableinstance.Model{}, fmt.Errorf("go_upb07_bundle.durable:%w", err)
	}
	one := func(schema string) (wire.Entity, error) {
		var out wire.Entity
		n := 0
		for _, q := range e.Entities {
			if q.Schema == mustID(schema) {
				out = q
				n++
			}
		}
		if n != 1 {
			return out, fmt.Errorf("go_upb07_bundle.durable_count:%s", schema)
		}
		return out, nil
	}
	fam, err := one("8011")
	if err != nil {
		return durableinstance.Model{}, err
	}
	port, err := one("8015")
	if err != nil {
		return durableinstance.Model{}, err
	}
	ref := func(q wire.Entity, field string) (wire.ID, error) {
		v := q.Fields[mustID(field)]
		if v.Tag != 6 {
			return wire.ID{}, fmt.Errorf("go_upb07_bundle.durable_field:%s", field)
		}
		return v.Reference, nil
	}
	versionType := func(id wire.ID) (wire.ID, error) {
		q, ok := e.Entities[id]
		if !ok || q.Schema != mustID("8012") {
			return wire.ID{}, fmt.Errorf("go_upb07_bundle.version")
		}
		return ref(q, "8122")
	}
	validator := func(id wire.ID) (wire.ID, error) {
		q, ok := e.Entities[id]
		if !ok || q.Schema != mustID("8013") {
			return wire.ID{}, fmt.Errorf("go_upb07_bundle.validator")
		}
		return ref(q, "8131")
	}
	vs := fam.Fields[mustID("8113")]
	validators := fam.Fields[mustID("8114")]
	migrations := fam.Fields[mustID("8115")]
	if vs.Tag != 7 || len(vs.List) != 2 || validators.Tag != 7 || len(validators.List) != 2 || migrations.Tag != 7 || len(migrations.List) != 1 {
		return durableinstance.Model{}, fmt.Errorf("go_upb07_bundle.durable_shape")
	}
	v1, err := versionType(vs.List[0].Reference)
	if err != nil {
		return durableinstance.Model{}, err
	}
	v2, err := versionType(vs.List[1].Reference)
	if err != nil {
		return durableinstance.Model{}, err
	}
	f1, err := validator(validators.List[0].Reference)
	if err != nil {
		return durableinstance.Model{}, err
	}
	f2, err := validator(validators.List[1].Reference)
	if err != nil {
		return durableinstance.Model{}, err
	}
	mq, ok := e.Entities[migrations.List[0].Reference]
	if !ok || mq.Schema != mustID("8014") {
		return durableinstance.Model{}, fmt.Errorf("go_upb07_bundle.migration")
	}
	migration, err := ref(mq, "8142")
	if err != nil {
		return durableinstance.Model{}, err
	}
	stateOwner, err := ref(fam, "8111")
	if err != nil {
		return durableinstance.Model{}, err
	}
	portOwner, err := ref(port, "8151")
	if err != nil {
		return durableinstance.Model{}, err
	}
	key, err := ref(port, "8157")
	if err != nil {
		return durableinstance.Model{}, err
	}
	fn := e.Entities[f1]
	resultID, err := ref(fn, "9112")
	if err != nil {
		return durableinstance.Model{}, err
	}
	result := e.Entities[resultID]
	errorType, err := ref(result, "9401")
	if err != nil {
		return durableinstance.Model{}, err
	}
	identity := fam.Fields[mustID("8110")]
	portIdentity := port.Fields[mustID("8150")]
	maxPayload := port.Fields[mustID("8158")]
	maxKey := port.Fields[mustID("815a")]
	if identity.Tag != 5 || portIdentity.Tag != 5 || maxPayload.Tag != 3 || maxKey.Tag != 3 {
		return durableinstance.Model{}, fmt.Errorf("go_upb07_bundle.metadata")
	}
	return durableinstance.Model{Identity: string(identity.Bytes), PortIdentity: string(portIdentity.Bytes), StateOwner: stateOwner, PortOwner: portOwner, Version1Type: v1, Version2Type: v2, ErrorType: errorType, Validator1: f1, Validator2: f2, Migration: migration, KeyType: key, MaximumPayloadBytes: maxPayload.Unsigned, MaximumKeyBytes: maxKey.Unsigned}, nil
}

func mustID(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, err := wire.ParseID(s)
	if err != nil {
		panic(err)
	}
	return x
}
func clone(a Artifacts) Artifacts {
	p := func(x []byte) []byte { return append([]byte(nil), x...) }
	return Artifacts{p(a.Construction), p(a.Execution), p(a.ProjectBase), p(a.Inventory), p(a.PackageDetail), p(a.PackageV4), p(a.Dependency), p(a.ConfigurationV3), p(a.ProjectV8), p(a.Resource), p(a.ProjectV9), p(a.Durable), p(a.ProjectV10)}
}
func cloneBlobs(x map[[32]byte][]byte) map[[32]byte][]byte {
	o := map[[32]byte][]byte{}
	for k, v := range x {
		o[k] = append([]byte(nil), v...)
	}
	return o
}
