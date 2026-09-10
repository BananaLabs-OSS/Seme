// Package goupb07pipeline builds the bounded durable-state project layer above UPB06.
package goupb07pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/durableinstance"
	"seme.local/reference/godurableadapter"
	"seme.local/reference/godurablemanifest"
	"seme.local/reference/goupb06pipeline"
	"seme.local/reference/projectv10instance"
)

type Input struct {
	Base      goupb06pipeline.Input
	Contracts contractcatalog.ProjectContractSetV10
	Manifest  []byte
}
type Result struct {
	Base                goupb06pipeline.Result
	Durable, ProjectV10 []byte
	DurableInput        durableinstance.Inputs
	ProjectInput        projectv10instance.Inputs
}

func Build(ctx context.Context, in Input) (Result, error) {
	if !in.Contracts.Validated() {
		return Result{}, fmt.Errorf("go_upb07.contracts")
	}
	base, err := goupb06pipeline.Build(ctx, in.Base)
	if err != nil {
		return Result{}, err
	}
	selection, err := godurablemanifest.Parse(in.Manifest)
	if err != nil {
		return Result{}, err
	}
	model, err := godurableadapter.Resolve(base.ProjectInput, selection)
	if err != nil {
		return Result{}, err
	}
	di := durableinstance.Inputs{Contracts: in.Contracts, ProjectV9: base.ProjectInput, Model: model}
	durable, err := durableinstance.Emit(di)
	if err != nil {
		return Result{}, err
	}
	di.Artifact = durable
	if err = durableinstance.Validate(di); err != nil {
		return Result{}, err
	}
	pi := projectv10instance.Inputs{Contracts: in.Contracts, ProjectV9: base.ProjectInput, Durable: di}
	project, err := projectv10instance.Emit(pi)
	if err != nil {
		return Result{}, err
	}
	pi.Composed = project
	if err = projectv10instance.Validate(pi); err != nil {
		return Result{}, err
	}
	return Result{Base: base, Durable: append([]byte(nil), durable...), ProjectV10: append([]byte(nil), project...), DurableInput: di, ProjectInput: pi}, nil
}

type artifact struct {
	name string
	data []byte
}

func artifacts(r Result) []artifact {
	return []artifact{{"construction-v36.g1", r.Base.Base.Construction}, {"execution-v36.seme", r.Base.Base.Execution}, {"project-base-v8.seme", r.Base.Base.ProjectBase}, {"inventory-v8.seme", r.Base.Base.Inventory}, {"package-detail-v4.seme", r.Base.Base.PackageDetail}, {"package-v4.seme", r.Base.Base.PackageV4}, {"dependency-v1.seme", r.Base.Base.Dependency}, {"configuration-v3.seme", r.Base.Base.ConfigurationV3}, {"project-v8.seme", r.Base.Base.ProjectV8}, {"resource-v1.seme", r.Base.Resource}, {"project-v9.seme", r.Base.ProjectV9}, {"durable-state-v1.seme", r.Durable}, {"project-v10.seme", r.ProjectV10}}
}
func Publish(destination string, r Result) error {
	if !filepath.IsAbs(destination) || filepath.Clean(destination) != destination || filepath.Base(destination) == "." {
		return fmt.Errorf("go_upb07.destination")
	}
	parent := filepath.Dir(destination)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("go_upb07.parent")
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil || resolved != parent {
		return fmt.Errorf("go_upb07.parent_symlink")
	}
	for _, f := range artifacts(r) {
		if len(f.data) == 0 {
			return fmt.Errorf("go_upb07.artifact_empty:%s", f.name)
		}
	}
	if err = os.Mkdir(destination, 0700); err != nil {
		return fmt.Errorf("go_upb07.destination_exists")
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
	manifest := []byte("seme-go-upb07-bundle-v1\n")
	for _, f := range artifacts(r) {
		if err = os.WriteFile(filepath.Join(destination, f.name), f.data, 0600); err != nil {
			return err
		}
		sum := sha256.Sum256(f.data)
		manifest = append(manifest, []byte(f.name+" "+hex.EncodeToString(sum[:])+"\n")...)
	}
	for _, b := range r.Base.Store.Blobs {
		name := "blobs/" + hex.EncodeToString(b.SHA256[:])
		if len(b.Bytes) == 0 || sha256.Sum256(b.Bytes) != b.SHA256 {
			return fmt.Errorf("go_upb07.blob")
		}
		if err = os.WriteFile(filepath.Join(destination, name), b.Bytes, 0600); err != nil {
			return err
		}
		manifest = append(manifest, []byte(name+" "+hex.EncodeToString(b.SHA256[:])+"\n")...)
	}
	if err = os.WriteFile(filepath.Join(destination, "COMPLETE.sha256"), manifest, 0600); err != nil {
		return err
	}
	complete = true
	return nil
}
