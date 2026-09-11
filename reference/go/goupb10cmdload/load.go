// Package goupb10cmdload strictly reopens a source-free UPB-10 placement and
// regenerates its semantic authority from the complete UPB-09 predecessor.
package goupb10cmdload

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprojectplacementadapter"
	"seme.local/reference/goupb09cmdload"
	"seme.local/reference/goupb10bundle"
)

type Paths struct {
	Base               goupb09cmdload.Paths
	Target, ProjectV13 string
	Placement          string
}

type Result struct {
	Bundle    goupb10bundle.Result
	Placement goupb10bundle.PlacementFiles
}

func (paths *Paths) Bind(set *flag.FlagSet) {
	paths.Base.Bind(set)
	set.StringVar(&paths.Target, "target", "", "Target Contract v1")
	set.StringVar(&paths.ProjectV13, "project-v13", "", "Project Contract v13")
	set.StringVar(&paths.Placement, "placement", "", "UPB-10 placement directory")
}

func Load(ctx context.Context, paths Paths, policy goprojectplacementadapter.Policy) (Result, error) {
	baseManifest, err := read(filepath.Join(paths.Base.Base.Bundle, "COMPLETE.sha256"))
	if err != nil {
		return Result{}, fmt.Errorf("base_manifest:%w", err)
	}
	placement, err := goupb10bundle.ReadDirectory(paths.Placement, baseManifest)
	if err != nil {
		return Result{}, err
	}
	base, err := goupb09cmdload.Load(ctx, paths.Base)
	if err != nil {
		return Result{}, err
	}
	contracts, err := ResolveContracts(paths)
	if err != nil {
		return Result{}, err
	}
	bundle, err := goupb10bundle.Load(goupb10bundle.Input{Contracts: contracts, Base: base.Bundle, Policy: policy, Artifacts: goupb10bundle.Artifacts{Base: base.Bundle.Artifacts, TargetPlan: placement.TargetPlan, ProjectV13: placement.ProjectV13, ProviderCatalog: placement.ProviderCatalog, LaunchManifest: placement.LaunchManifest, CanonicalVM: placement.CanonicalVM, PulpCell: placement.PulpCell}})
	if err != nil {
		return Result{}, err
	}
	return Result{Bundle: bundle, Placement: placement}, nil
}

func ResolveContracts(paths Paths) (contractcatalog.ProjectContractSetV13, error) {
	p := paths.Base.Base
	locations := []string{p.Foundation, p.Execution, p.Package, p.Dependency, p.Configuration, p.Resource, p.Durable, p.Presentation, p.OrderedTransport, paths.Base.ControlledEffects, paths.Target, p.ProjectV9, p.ProjectV10, p.ProjectV11, paths.Base.ProjectV12, paths.ProjectV13}
	values := make([][]byte, len(locations))
	for index, location := range locations {
		value, err := read(location)
		if err != nil {
			return contractcatalog.ProjectContractSetV13{}, err
		}
		values[index] = value
	}
	return contractcatalog.ResolveProjectContractSetV13(values[0], values[1], values[2], values[3], values[4], values[5], values[6], values[7], values[8], values[9], values[10], values[11], values[12], values[13], values[14], values[15])
}

func read(path string) ([]byte, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, fmt.Errorf("path")
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil || real != path {
		return nil, fmt.Errorf("symlink")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > 64<<20 {
		return nil, fmt.Errorf("regular")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("changed")
	}
	value, err := io.ReadAll(io.LimitReader(file, 64<<20+1))
	if err != nil || int64(len(value)) != opened.Size() {
		return nil, fmt.Errorf("changed")
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("changed")
	}
	return value, nil
}
