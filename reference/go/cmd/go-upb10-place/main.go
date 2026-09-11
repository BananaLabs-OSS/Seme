// Command go-upb10-place derives and atomically publishes the UPB-10 target
// plan and Project-v13 binding from an authenticated source-free UPB-09 bundle.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprojectplacementadapter"
	"seme.local/reference/goupb09cmdload"
	"seme.local/reference/projectv13instance"
	"seme.local/reference/targetplaninstance"
)

type options struct {
	base               goupb09cmdload.Paths
	target, projectV13 string
	out, policy        string
	targetName         string
	targetRevision     uint64
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
	if err := set.Parse(arguments); err != nil || set.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	if options.out == "" || !filepath.IsAbs(options.out) || filepath.Clean(options.out) != options.out || options.targetName == "" || options.targetRevision == 0 {
		return fmt.Errorf("options")
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
	contracts, err := resolveV13(options)
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
	baseManifest, err := strictRead(filepath.Join(options.base.Base.Bundle, "COMPLETE.sha256"))
	if err != nil {
		return fmt.Errorf("base_manifest:%w", err)
	}
	return publish(options.out, map[string][]byte{"project-v13.seme": project, "target-plan-v1.seme": plan}, baseManifest)
}

func resolveV13(options options) (contractcatalog.ProjectContractSetV13, error) {
	p := options.base.Base
	paths := []string{p.Foundation, p.Execution, p.Package, p.Dependency, p.Configuration, p.Resource, p.Durable, p.Presentation, p.OrderedTransport, options.base.ControlledEffects, options.target, p.ProjectV9, p.ProjectV10, p.ProjectV11, options.base.ProjectV12, options.projectV13}
	values := make([][]byte, len(paths))
	for index, path := range paths {
		value, err := strictRead(path)
		if err != nil {
			return contractcatalog.ProjectContractSetV13{}, err
		}
		values[index] = value
	}
	return contractcatalog.ResolveProjectContractSetV13(values[0], values[1], values[2], values[3], values[4], values[5], values[6], values[7], values[8], values[9], values[10], values[11], values[12], values[13], values[14], values[15])
}

func publish(destination string, artifacts map[string][]byte, baseManifest []byte) error {
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		return fmt.Errorf("output_exists")
	}
	parent := filepath.Dir(destination)
	real, err := filepath.EvalSymlinks(parent)
	if err != nil || real != parent {
		return fmt.Errorf("output_parent")
	}
	temporary, err := os.MkdirTemp(parent, ".seme-upb10-placement-")
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(temporary)
		}
	}()
	names := make([]string, 0, len(artifacts))
	for name := range artifacts {
		names = append(names, name)
	}
	sort.Strings(names)
	manifest := []byte("seme-go-upb10-placement-v1\n")
	baseDigest := sha256.Sum256(baseManifest)
	manifest = append(manifest, []byte("base-complete "+hex.EncodeToString(baseDigest[:])+"\n")...)
	for _, name := range names {
		value := artifacts[name]
		if len(value) == 0 || strings.Contains(name, "/") {
			return fmt.Errorf("artifact")
		}
		digest := sha256.Sum256(value)
		manifest = append(manifest, []byte(name+" "+hex.EncodeToString(digest[:])+"\n")...)
		if err = os.WriteFile(filepath.Join(temporary, name), value, 0o600); err != nil {
			return err
		}
	}
	if err = os.WriteFile(filepath.Join(temporary, "COMPLETE.sha256"), manifest, 0o600); err != nil {
		return err
	}
	if err = os.Rename(temporary, destination); err != nil {
		return fmt.Errorf("publish_commit:%w", err)
	}
	keep = true
	return nil
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
