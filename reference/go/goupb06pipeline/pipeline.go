// Package goupb06pipeline builds the bounded detached-resource project layer.
package goupb06pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/goresourceadapter"
	"seme.local/reference/goresourcemanifest"
	"seme.local/reference/goupb05pipeline"
	"seme.local/reference/projectv9instance"
	"seme.local/reference/resourceinstance"
)

type Input struct {
	Base      goupb05pipeline.V8Input
	Contracts contractcatalog.ProjectContractSetV9
	Manifest  []byte
	// OwnerPackage explicitly binds detached resources to one authenticated
	// Package-v4 owner; it is policy, not inferred from filenames.
	OwnerPackage string
}

// Publish writes a create-only bundle. COMPLETE is written last; any failure
// removes the new directory so consumers never observe a claimed partial set.
func Publish(destination string, r Result) error {
	if !filepath.IsAbs(destination) || filepath.Clean(destination) != destination || filepath.Base(destination) == "." {
		return fmt.Errorf("go_upb06.destination")
	}
	parent := filepath.Dir(destination)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("go_upb06.parent")
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil || resolved != parent {
		return fmt.Errorf("go_upb06.parent_symlink")
	}
	files := artifacts(r)
	for _, f := range files {
		if len(f.data) == 0 {
			return fmt.Errorf("go_upb06.artifact_empty:%s", f.name)
		}
	}
	if err = os.Mkdir(destination, 0700); err != nil {
		return fmt.Errorf("go_upb06.destination_exists")
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
	manifest := []byte("seme-go-upb06-bundle-v1\n")
	for _, f := range files {
		if err = os.WriteFile(filepath.Join(destination, f.name), f.data, 0600); err != nil {
			return err
		}
		sum := sha256.Sum256(f.data)
		manifest = append(manifest, []byte(f.name+" "+hex.EncodeToString(sum[:])+"\n")...)
	}
	for _, b := range r.Store.Blobs {
		name := "blobs/" + hex.EncodeToString(b.SHA256[:])
		if len(b.Bytes) == 0 || sha256.Sum256(b.Bytes) != b.SHA256 {
			return fmt.Errorf("go_upb06.blob")
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

type artifact struct {
	name string
	data []byte
}

func artifacts(r Result) []artifact {
	return []artifact{{"construction-v36.g1", r.Base.Construction}, {"execution-v36.seme", r.Base.Execution}, {"project-base-v8.seme", r.Base.ProjectBase}, {"inventory-v8.seme", r.Base.Inventory}, {"package-detail-v4.seme", r.Base.PackageDetail}, {"package-v4.seme", r.Base.PackageV4}, {"dependency-v1.seme", r.Base.Dependency}, {"configuration-v3.seme", r.Base.ConfigurationV3}, {"project-v8.seme", r.Base.ProjectV8}, {"resource-v1.seme", r.Resource}, {"project-v9.seme", r.ProjectV9}}
}

type Result struct {
	Base                goupb05pipeline.V8Result
	Resource, ProjectV9 []byte
	Store               goresourceadapter.Store
	ResourceInput       resourceinstance.Inputs
	ProjectInput        projectv9instance.Inputs
}

func Build(ctx context.Context, in Input) (Result, error) {
	if !in.Contracts.Validated() {
		return Result{}, fmt.Errorf("go_upb06.contracts")
	}
	base, err := goupb05pipeline.BuildV8(ctx, in.Base)
	if err != nil {
		return Result{}, err
	}
	selections, err := goresourcemanifest.Parse(in.Manifest)
	if err != nil {
		return Result{}, err
	}
	model, store, err := goresourceadapter.Resolve(in.Base.Dependency.ProjectRoot, in.Base.Sources, base.ProjectInput, in.OwnerPackage, selections)
	if err != nil {
		return Result{}, err
	}
	ri := resourceinstance.Inputs{Contracts: in.Contracts, ProjectV8: base.ProjectInput, Model: goresourceadapter.InstanceModel(model)}
	resource, err := resourceinstance.Emit(ri)
	if err != nil {
		return Result{}, err
	}
	ri.Artifact = resource
	blobs := map[[32]byte][]byte{}
	for _, b := range store.Blobs {
		blobs[b.SHA256] = b.Bytes
	}
	if err = resourceinstance.VerifyDetached(ri, blobs); err != nil {
		return Result{}, err
	}
	pi := projectv9instance.Inputs{Contracts: in.Contracts, ProjectV8: base.ProjectInput, Resource: ri}
	project, err := projectv9instance.Emit(pi)
	if err != nil {
		return Result{}, err
	}
	pi.Composed = project
	if err = projectv9instance.Validate(pi); err != nil {
		return Result{}, err
	}
	return Result{Base: base, Resource: append([]byte(nil), resource...), ProjectV9: append([]byte(nil), project...), Store: store, ResourceInput: ri, ProjectInput: pi}, nil
}
