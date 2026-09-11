// Command go-upb11-finalize authenticates both complete UPB10 deployments,
// proves the projected source re-lifts to the updated construction, and
// atomically publishes Patch-v1 plus Project-v14 reconciliation authority.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprojectplacementadapter"
	"seme.local/reference/goprovider"
	"seme.local/reference/goupb10cmdload"
	"seme.local/reference/patchinstance"
	"seme.local/reference/projectv14instance"
	"seme.local/reference/targetplaninstance"
	"seme.local/reference/wire"
)

type options struct {
	common                                             goupb10cmdload.Paths
	priorBundle, priorPlacement                        string
	resultBundle, resultPlacement                      string
	reconciliation, providerG1, executionG1            string
	patchContract, languageService, projectV14, output string
	module, packagePath, entry                         string
	clientRevision                                     uint64
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb11-finalize:", err)
		os.Exit(1)
	}
}

func run(parent context.Context, arguments []string, stderr io.Writer) error {
	set := flag.NewFlagSet("go-upb11-finalize", flag.ContinueOnError)
	set.SetOutput(stderr)
	var o options
	o.common.Bind(set)
	set.StringVar(&o.priorBundle, "prior-bundle", "", "prior UPB09 bundle")
	set.StringVar(&o.priorPlacement, "prior-placement", "", "prior UPB10 placement")
	set.StringVar(&o.resultBundle, "result-bundle", "", "result UPB09 bundle")
	set.StringVar(&o.resultPlacement, "result-placement", "", "result UPB10 placement")
	set.StringVar(&o.reconciliation, "reconciliation", "", "projected native reconciliation root")
	set.StringVar(&o.providerG1, "provider-g1", "", "Provider-v1 G1")
	set.StringVar(&o.executionG1, "execution-g1", "", "Execution-v36 G1")
	set.StringVar(&o.patchContract, "patch", "", "Patch-v1 contract")
	set.StringVar(&o.languageService, "language-service", "", "Language Service-v1 contract")
	set.StringVar(&o.projectV14, "project-v14", "", "Project-v14 contract")
	set.StringVar(&o.output, "out", "", "new reconciliation authority directory")
	set.StringVar(&o.module, "module", "", "native module path")
	set.StringVar(&o.packagePath, "root-package", "", "native root package")
	set.StringVar(&o.entry, "entry", "", "native entry function")
	set.Uint64Var(&o.clientRevision, "client-revision", 0, "accepted project revision")
	if err := set.Parse(arguments); err != nil || set.NArg() != 0 || !valid(o) {
		return fmt.Errorf("arguments")
	}
	if _, err := os.Lstat(o.output); !os.IsNotExist(err) {
		return fmt.Errorf("output_exists")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute)
	defer cancel()
	policy := goprojectplacementadapter.Policy{Name: "wasm32-pulp-go-host-v1", Revision: 1, AllowedFidelity: []targetplaninstance.Fidelity{targetplaninstance.Exact, targetplaninstance.NativeIsland}}
	priorPaths, resultPaths := o.common, o.common
	priorPaths.Base.Base.Bundle, priorPaths.Placement = o.priorBundle, o.priorPlacement
	resultPaths.Base.Base.Bundle, resultPaths.Placement = o.resultBundle, o.resultPlacement
	prior, err := goupb10cmdload.Load(ctx, priorPaths, policy)
	if err != nil {
		return fmt.Errorf("prior:%w", err)
	}
	result, err := goupb10cmdload.Load(ctx, resultPaths, policy)
	if err != nil {
		return fmt.Errorf("result:%w", err)
	}
	reconciliation, err := goprovider.ReadReconciliationBundle(o.reconciliation, o.providerG1)
	if err != nil {
		return err
	}
	snapshot, err := goprovider.ReadDocumentSnapshot(o.reconciliation, o.module, o.packagePath, o.entry, o.clientRevision)
	if err != nil {
		return err
	}
	snapshot.IdentityBindings = append([]goprovider.IdentityBinding(nil), reconciliation.Report.IdentityBindings...)
	executionModule, err := read(o.executionG1)
	if err != nil {
		return err
	}
	session, err := goprovider.NewIncrementalSession(executionModule)
	if err != nil {
		return err
	}
	lifted := session.Apply(snapshot)
	construction := result.Bundle.Artifacts.Base.Base.Base.Construction
	if !lifted.Valid || !bytes.Equal([]byte(lifted.CanonicalG1), construction) {
		return fmt.Errorf("result_construction_fixed_point")
	}
	v13, err := goupb10cmdload.ResolveContracts(o.common)
	if err != nil {
		return err
	}
	patchSource, err := read(o.patchContract)
	if err != nil {
		return err
	}
	languageSource, err := read(o.languageService)
	if err != nil {
		return err
	}
	projectSource, err := read(o.projectV14)
	if err != nil {
		return err
	}
	contracts, err := contractcatalog.ResolveProjectContractSetV14FromV13(v13, patchSource, languageSource, projectSource)
	if err != nil {
		return err
	}
	priorGraph, err := wire.Decode(prior.Placement.ProjectV13)
	if err != nil {
		return err
	}
	resultGraph, err := wire.Decode(result.Placement.ProjectV13)
	if err != nil {
		return err
	}
	binding := reconciliation.Report.IdentityBindings[0]
	target, err := wire.ParseID(binding.ID)
	if err != nil {
		return err
	}
	nameField := xid("9110")
	before, beforeOK := priorGraph.Entities[target]
	after, afterOK := resultGraph.Entities[target]
	beforeName, afterName := before.Fields[nameField], after.Fields[nameField]
	if !beforeOK || !afterOK || beforeName.Tag != 5 || afterName.Tag != 5 || bytes.Equal(beforeName.Bytes, afterName.Bytes) || string(afterName.Bytes) != binding.Name {
		return fmt.Errorf("identity_transition")
	}
	patchArtifact, err := patchinstance.EmitRename(contracts.Patch(), patchinstance.Rename{Base: priorGraph, Target: target, Field: nameField, Expected: beforeName.Bytes, Replacement: afterName.Bytes})
	if err != nil {
		return err
	}
	projectInput := projectv14instance.Inputs{Contracts: contracts, Prior: prior.Bundle.Project, Result: result.Bundle.Project, Patch: patchArtifact, ClientRevision: o.clientRevision, NativeValidationTranscript: reconciliation.NativeValidationTranscript}
	projectArtifact, err := projectv14instance.Emit(projectInput)
	if err != nil {
		return err
	}
	projectInput.Artifact = projectArtifact
	if err = projectv14instance.Validate(projectInput); err != nil {
		return err
	}
	return publish(o.output, map[string][]byte{"patch-v1.seme": patchArtifact, "project-v14.seme": projectArtifact, "native-validation.txt": reconciliation.NativeValidationTranscript})
}

func valid(o options) bool {
	values := []string{o.priorBundle, o.priorPlacement, o.resultBundle, o.resultPlacement, o.reconciliation, o.providerG1, o.executionG1, o.patchContract, o.languageService, o.projectV14, o.output, o.module, o.packagePath, o.entry}
	for _, value := range values {
		if value == "" {
			return false
		}
	}
	return o.clientRevision != 0 && filepath.IsAbs(o.output) && filepath.Clean(o.output) == o.output
}

func publish(destination string, artifacts map[string][]byte) error {
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		return fmt.Errorf("output_exists")
	}
	parent := filepath.Dir(destination)
	real, err := filepath.EvalSymlinks(parent)
	if err != nil || real != parent {
		return fmt.Errorf("output_parent")
	}
	stage, err := os.MkdirTemp(parent, ".seme-go-upb11-finalize-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	names := make([]string, 0, len(artifacts))
	for name, value := range artifacts {
		if len(value) == 0 {
			return fmt.Errorf("empty_artifact")
		}
		names = append(names, name)
		if err = os.WriteFile(filepath.Join(stage, name), value, 0o600); err != nil {
			return err
		}
	}
	sort.Strings(names)
	manifest := []byte("seme-go-upb11-authority-v1\n")
	for _, name := range names {
		digest := sha256.Sum256(artifacts[name])
		manifest = append(manifest, []byte(name+" "+hex.EncodeToString(digest[:])+"\n")...)
	}
	if err = os.WriteFile(filepath.Join(stage, "COMPLETE.sha256"), manifest, 0o600); err != nil {
		return err
	}
	return os.Rename(stage, destination)
}

func read(path string) ([]byte, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, fmt.Errorf("path")
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil || real != path {
		return nil, fmt.Errorf("symlink")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64<<20 {
		return nil, fmt.Errorf("regular")
	}
	return os.ReadFile(path)
}

func xid(value string) wire.ID {
	for len(value) < 32 {
		value = "0" + value
	}
	result, err := wire.ParseID(value)
	if err != nil {
		panic(err)
	}
	return result
}
