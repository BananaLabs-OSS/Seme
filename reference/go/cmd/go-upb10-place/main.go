// Command go-upb10-place derives and atomically publishes the UPB-10 target
// plan and Project-v13 binding from an authenticated source-free UPB-09 bundle.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"seme.local/reference/goprojectplacementadapter"
	"seme.local/reference/goupb09cmdload"
	"seme.local/reference/goupb10bundle"
	"seme.local/reference/goupb10cmdload"
	"seme.local/reference/goupb10deployment"
	"seme.local/reference/goupb10report"
	"seme.local/reference/projectv13instance"
	"seme.local/reference/targetplaninstance"
)

type options struct {
	base                  goupb09cmdload.Paths
	target, projectV13    string
	out, policy           string
	targetName            string
	targetRevision        uint64
	canonicalVM, pulpCell string
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb10-place:", err)
		os.Exit(1)
	}
}

func run(parent context.Context, arguments []string, stderr io.Writer) error {
	set := flag.NewFlagSet("go-upb10-place", flag.ContinueOnError)
	set.SetOutput(stderr)
	var options options
	options.base.Bind(set)
	set.StringVar(&options.target, "target", "", "Target Contract v1")
	set.StringVar(&options.projectV13, "project-v13", "", "Project Contract v13")
	set.StringVar(&options.out, "out", "", "new output directory")
	set.StringVar(&options.policy, "policy", "mixed", "mixed or exact-only")
	set.StringVar(&options.targetName, "target-name", "wasm32-pulp-go-host-v1", "target identity")
	set.Uint64Var(&options.targetRevision, "target-revision", 1, "target revision")
	set.StringVar(&options.canonicalVM, "canonical-vm", "", "canonical VM Wasm artifact")
	set.StringVar(&options.pulpCell, "pulp-cell", "", "Pulp cell manifest")
	if err := set.Parse(arguments); err != nil || set.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	if options.out == "" || !filepath.IsAbs(options.out) || filepath.Clean(options.out) != options.out || options.targetName == "" || options.targetRevision == 0 || options.canonicalVM == "" || options.pulpCell == "" {
		return fmt.Errorf("options")
	}
	if _, err := os.Lstat(options.out); !os.IsNotExist(err) {
		return fmt.Errorf("output_exists")
	}
	allowed := []targetplaninstance.Fidelity{targetplaninstance.Exact, targetplaninstance.NativeIsland}
	if options.policy == "exact-only" {
		allowed = []targetplaninstance.Fidelity{targetplaninstance.Exact}
	} else if options.policy != "mixed" {
		return fmt.Errorf("policy")
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	defer cancel()
	loaded, err := goupb09cmdload.Load(ctx, options.base)
	if err != nil {
		return err
	}
	contracts, err := goupb10cmdload.ResolveContracts(goupb10cmdload.Paths{Base: options.base, Target: options.target, ProjectV13: options.projectV13})
	if err != nil {
		return err
	}
	planInput, err := goprojectplacementadapter.Derive(loaded.Bundle.Project, contracts.Target(), goprojectplacementadapter.Policy{Name: options.targetName, Revision: options.targetRevision, AllowedFidelity: allowed})
	if err != nil {
		return err
	}
	plan, err := targetplaninstance.Emit(planInput)
	if err != nil {
		return err
	}
	planInput.Artifact = plan
	if err = targetplaninstance.Validate(planInput); err != nil {
		return err
	}
	if options.policy == "exact-only" {
		return fmt.Errorf("non_executable_policy")
	}
	projectInput := projectv13instance.Inputs{Contracts: contracts, ProjectV12: loaded.Bundle.Project, Plan: planInput}
	project, err := projectv13instance.Emit(projectInput)
	if err != nil {
		return err
	}
	projectInput.Composed = project
	if err = projectv13instance.Validate(projectInput); err != nil {
		return err
	}
	catalog, err := goupb10deployment.EmitCatalog(planInput.Authority, planInput.Model.Target)
	if err != nil {
		return err
	}
	vmBytes, err := strictRead(options.canonicalVM)
	if err != nil {
		return fmt.Errorf("canonical_vm:%w", err)
	}
	cellBytes, err := strictRead(options.pulpCell)
	if err != nil {
		return fmt.Errorf("pulp_cell:%w", err)
	}
	vm, err := goupb10deployment.DigestArtifact("canonical-vm.wasm", "wasm", vmBytes)
	if err != nil {
		return err
	}
	cell, err := goupb10deployment.DigestArtifact("pulp.cell.toml", "pulp-cell-manifest", cellBytes)
	if err != nil {
		return err
	}
	launch, err := goupb10deployment.EmitLaunch(catalog, plan, project, goupb10deployment.PinnedPulpCommit, map[string]goupb10deployment.Artifact{vm.Name: vm, cell.Name: cell}, planInput.Model.Boundaries)
	if err != nil {
		return err
	}
	bundle, err := goupb10bundle.Load(goupb10bundle.Input{
		Contracts: contracts, Base: loaded.Bundle,
		Policy:    goprojectplacementadapter.Policy{Name: options.targetName, Revision: options.targetRevision, AllowedFidelity: allowed},
		Artifacts: goupb10bundle.Artifacts{Base: loaded.Bundle.Artifacts, TargetPlan: plan, ProjectV13: project, ProviderCatalog: catalog, LaunchManifest: launch, CanonicalVM: vmBytes, PulpCell: cellBytes},
	})
	if err != nil {
		return err
	}
	report, err := goupb10report.Inspect(bundle)
	if err != nil {
		return err
	}
	reportBytes, err := goupb10report.Marshal(report)
	if err != nil {
		return err
	}
	baseManifest, err := strictRead(filepath.Join(options.base.Base.Bundle, "COMPLETE.sha256"))
	if err != nil {
		return fmt.Errorf("base_manifest:%w", err)
	}
	return goupb10bundle.WriteDirectory(options.out, goupb10bundle.PlacementFiles{TargetPlan: plan, ProjectV13: project, Report: reportBytes, ProviderCatalog: catalog, LaunchManifest: launch, CanonicalVM: vmBytes, PulpCell: cellBytes}, baseManifest)
}

func strictRead(path string) ([]byte, error) {
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
