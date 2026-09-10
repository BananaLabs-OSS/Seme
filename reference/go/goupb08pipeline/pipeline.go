// Package goupb08pipeline builds and publishes the bounded ordered-transport layer above UPB07.
package goupb08pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goorderedtransportadapter"
	"seme.local/reference/goorderedtransportmanifest"
	"seme.local/reference/goupb07pipeline"
	"seme.local/reference/orderedtransportinstance"
	"seme.local/reference/projectv11instance"
)

type Input struct {
	Base      goupb07pipeline.Input
	Contracts contractcatalog.ProjectContractSetV11
	Manifest  []byte
}
type Result struct {
	Base                  goupb07pipeline.Result
	Transport, ProjectV11 []byte
	Selection             goorderedtransportadapter.Selection
	TransportInput        orderedtransportinstance.Inputs
	ProjectInput          projectv11instance.Inputs
}

func Build(ctx context.Context, in Input) (Result, error) {
	if !in.Contracts.Validated() {
		return Result{}, fmt.Errorf("go_upb08.contracts")
	}
	base, err := goupb07pipeline.Build(ctx, in.Base)
	if err != nil {
		return Result{}, err
	}
	selection, err := goorderedtransportmanifest.Parse(in.Manifest)
	if err != nil {
		return Result{}, err
	}
	model, err := goorderedtransportadapter.Resolve(base.ProjectInput, selection)
	if err != nil {
		return Result{}, err
	}
	ti := orderedtransportinstance.Inputs{Contracts: in.Contracts, ProjectV10: base.ProjectInput, Model: model}
	transport, err := orderedtransportinstance.Emit(ti)
	if err != nil {
		return Result{}, err
	}
	ti.Artifact = transport
	if err = orderedtransportinstance.Validate(ti); err != nil {
		return Result{}, err
	}
	pi := projectv11instance.Inputs{Contracts: in.Contracts, ProjectV10: base.ProjectInput, Transport: ti}
	project, err := projectv11instance.Emit(pi)
	if err != nil {
		return Result{}, err
	}
	pi.Composed = project
	if err = projectv11instance.Validate(pi); err != nil {
		return Result{}, err
	}
	return Result{Base: base, Transport: append([]byte(nil), transport...), ProjectV11: append([]byte(nil), project...), Selection: selection, TransportInput: ti, ProjectInput: pi}, nil
}

type artifact struct {
	name string
	data []byte
}

func Artifacts(r Result) []artifact {
	a := []artifact{}
	for _, f := range baseArtifacts(r.Base) {
		a = append(a, f)
	}
	return append(a, artifact{"ordered-transport-v1.seme", r.Transport}, artifact{"project-v11.seme", r.ProjectV11})
}
func baseArtifacts(r goupb07pipeline.Result) []artifact {
	return []artifact{{"construction-v36.g1", r.Base.Base.Construction}, {"execution-v36.seme", r.Base.Base.Execution}, {"project-base-v8.seme", r.Base.Base.ProjectBase}, {"inventory-v8.seme", r.Base.Base.Inventory}, {"package-detail-v4.seme", r.Base.Base.PackageDetail}, {"package-v4.seme", r.Base.Base.PackageV4}, {"dependency-v1.seme", r.Base.Base.Dependency}, {"configuration-v3.seme", r.Base.Base.ConfigurationV3}, {"project-v8.seme", r.Base.Base.ProjectV8}, {"resource-v1.seme", r.Base.Resource}, {"project-v9.seme", r.Base.ProjectV9}, {"durable-state-v1.seme", r.Durable}, {"source-presentation-v1.seme", r.Presentation}, {"project-v10.seme", r.ProjectV10}}
}
func Publish(destination string, r Result) error {
	if !filepath.IsAbs(destination) || filepath.Clean(destination) != destination || filepath.Base(destination) == "." {
		return fmt.Errorf("go_upb08.destination")
	}
	parent := filepath.Dir(destination)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("go_upb08.parent")
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil || resolved != parent {
		return fmt.Errorf("go_upb08.parent_symlink")
	}
	for _, f := range Artifacts(r) {
		if len(f.data) == 0 {
			return fmt.Errorf("go_upb08.artifact_empty:%s", f.name)
		}
	}
	if err = os.Mkdir(destination, 0700); err != nil {
		return fmt.Errorf("go_upb08.destination_exists")
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(destination)
		}
	}()
	if err = os.Mkdir(filepath.Join(destination, "blobs"), 0700); err != nil {
		return err
	}
	manifest := []byte("seme-go-upb08-bundle-v1\n")
	for _, f := range Artifacts(r) {
		if err = os.WriteFile(filepath.Join(destination, f.name), f.data, 0600); err != nil {
			return err
		}
		s := sha256.Sum256(f.data)
		manifest = append(manifest, []byte(f.name+" "+hex.EncodeToString(s[:])+"\n")...)
	}
	for _, b := range r.Base.Base.Store.Blobs {
		if len(b.Bytes) == 0 || sha256.Sum256(b.Bytes) != b.SHA256 {
			return fmt.Errorf("go_upb08.blob")
		}
		n := "blobs/" + hex.EncodeToString(b.SHA256[:])
		if err = os.WriteFile(filepath.Join(destination, n), b.Bytes, 0600); err != nil {
			return err
		}
		manifest = append(manifest, []byte(n+" "+hex.EncodeToString(b.SHA256[:])+"\n")...)
	}
	if err = os.WriteFile(filepath.Join(destination, "COMPLETE.sha256"), manifest, 0600); err != nil {
		return err
	}
	complete = true
	return nil
}
