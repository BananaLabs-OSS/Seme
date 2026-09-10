// Package goprojectpipeline assembles the bounded Go UPB-02 evidence chain.
// It returns artifacts only after every independent and composed validator has
// accepted the same current source snapshot.
package goprojectpipeline

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/gopackageadapter"
	"seme.local/reference/goprovider"
	"seme.local/reference/packagedetailemitter"
	"seme.local/reference/projectbuild"
	"seme.local/reference/projectsource"
	"seme.local/reference/projectv3emitter"
	"seme.local/reference/sourceinventory"
	"seme.local/reference/wire"
)

type Contracts struct {
	V1, V2, V3 contractcatalog.ProjectContractSet
}
type Input struct {
	Documents   goprovider.DocumentSnapshot
	Sources     projectsource.Snapshot
	Contracts   Contracts
	ExecutionG1 []byte
	Compile     projectbuild.Compile
}
type Result struct{ ProjectV1, InventoryV2, PackageV2, ProjectV3 []byte }

func Build(ctx context.Context, in Input) (Result, error) {
	if err := validateContracts(in.Contracts); err != nil {
		return Result{}, err
	}
	project, err := projectbuild.Build(ctx, in.Documents, in.Contracts.V1, in.ExecutionG1, in.Compile)
	if err != nil {
		return Result{}, err
	}
	if err = ctx.Err(); err != nil {
		return Result{}, err
	}
	inventory, err := sourceinventory.Emit(in.Contracts.V2.Project(), project.Artifact, in.Sources)
	if err != nil {
		return Result{}, fmt.Errorf("go_project_pipeline.inventory:%w", err)
	}
	if err = ctx.Err(); err != nil {
		return Result{}, err
	}
	evidence, err := gopackageadapter.EvidenceFrom(in.Documents, project.Resolution, in.Contracts.V2.Project(), project.Artifact, inventory)
	if err != nil {
		return Result{}, fmt.Errorf("go_project_pipeline.evidence:%w", err)
	}
	detail, err := gopackageadapter.Convert(project.Resolution, project.Packages, evidence)
	if err != nil {
		return Result{}, fmt.Errorf("go_project_pipeline.package_model:%w", err)
	}
	base, err := mergeInventory(project.Artifact, inventory)
	if err != nil {
		return Result{}, err
	}
	packageArtifact, err := packagedetailemitter.Emit(base, detail)
	if err != nil {
		return Result{}, fmt.Errorf("go_project_pipeline.package:%w", err)
	}
	if err = ctx.Err(); err != nil {
		return Result{}, err
	}
	composed, err := projectv3emitter.Emit(projectv3emitter.Input{Contracts: in.Contracts.V3, ProjectV2: in.Contracts.V2.Project(), Project: project.Artifact, Inventory: inventory, PackageGraph: packageArtifact})
	if err != nil {
		return Result{}, fmt.Errorf("go_project_pipeline.compose:%w", err)
	}
	return Result{clone(project.Artifact), clone(inventory), clone(packageArtifact), clone(composed)}, nil
}

func validateContracts(c Contracts) error {
	if !c.V1.Validated() || !c.V2.Validated() || !c.V3.Validated() {
		return fmt.Errorf("go_project_pipeline.contracts")
	}
	want := []contractcatalog.Pin{{Module: mustID("e000"), Revision: mustID("e001")}, {Module: mustID("e000"), Revision: mustID("e002")}, {Module: mustID("e000"), Revision: mustID("e003")}, {Module: mustID("b000"), Revision: mustID("b002")}}
	got := []contractcatalog.Pin{c.V1.Project().Pin(), c.V2.Project().Pin(), c.V3.Project().Pin(), c.V3.Package().Pin()}
	for i := range want {
		if got[i] != want[i] {
			return fmt.Errorf("go_project_pipeline.contract_pin")
		}
	}
	return nil
}

func mergeInventory(project, inventory []byte) (wire.Envelope, error) {
	p, err := wire.Decode(project)
	if err != nil {
		return wire.Envelope{}, err
	}
	s, err := wire.Decode(inventory)
	if err != nil {
		return wire.Envelope{}, err
	}
	for x, q := range s.Entities {
		if q.Schema == mustID("12") || q.Schema == mustID("13") {
			continue
		}
		if old, ok := p.Entities[x]; ok && !sameEntity(old, q) {
			return wire.Envelope{}, fmt.Errorf("go_project_pipeline.inventory_collision:%s", x)
		}
		p.Entities[x] = q
	}
	return p, nil
}

// Publish reserves a new destination and writes a digest manifest last. A
// reader accepts the directory as published only when COMPLETE.sha256 exists
// and verifies every listed artifact.
func Publish(destination string, r Result) error {
	if !filepath.IsAbs(destination) {
		return fmt.Errorf("go_project_pipeline.destination_absolute")
	}
	if filepath.Clean(destination) != destination || filepath.Base(destination) == "." {
		return fmt.Errorf("go_project_pipeline.destination_normalized")
	}
	parent := filepath.Dir(destination)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("go_project_pipeline.parent")
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil || resolved != parent {
		return fmt.Errorf("go_project_pipeline.parent_symlink")
	}
	files := []struct {
		name string
		data []byte
	}{{"project-v1.seme", r.ProjectV1}, {"inventory-v2.seme", r.InventoryV2}, {"package-v2.seme", r.PackageV2}, {"project-v3.seme", r.ProjectV3}}
	for _, file := range files {
		name, data := file.name, file.data
		if len(data) == 0 {
			return fmt.Errorf("go_project_pipeline.artifact_empty:%s", name)
		}
	}
	// Mkdir is the create-only reservation. Unlike Rename, it cannot replace a
	// destination that appears between a prior existence check and publication.
	if err = os.Mkdir(destination, 0700); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("go_project_pipeline.destination_exists")
		}
		return err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(destination)
		}
	}()
	manifest := []byte("seme-go-project-bundle-v1\n")
	for _, file := range files {
		name, data := file.name, file.data
		if err = os.WriteFile(filepath.Join(destination, name), data, 0600); err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		manifest = append(manifest, []byte(name+" "+hex.EncodeToString(sum[:])+"\n")...)
	}
	if err = os.WriteFile(filepath.Join(destination, "COMPLETE.sha256"), manifest, 0600); err != nil {
		return err
	}
	complete = true
	return nil
}

func sameEntity(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
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
func clone(x []byte) []byte { return append([]byte(nil), x...) }
