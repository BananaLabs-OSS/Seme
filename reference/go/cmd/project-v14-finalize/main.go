// Command project-v14-finalize binds an authenticated before/after semantic
// project revision, Patch-v1 transaction, and native validation transcript
// into one deterministic Project-v14 authority.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprojectplacementadapter"
	"seme.local/reference/goupb10cmdload"
	"seme.local/reference/patchinstance"
	"seme.local/reference/projectv14instance"
	"seme.local/reference/targetplaninstance"
	"seme.local/reference/wire"
)

type report struct {
	Version                              uint64 `json:"version"`
	Language, Project                    string
	ClientRevision                       uint64 `json:"client_revision"`
	Target, Field, Expected, Replacement string
	Occurrences                          []json.RawMessage
}

func main() {
	set := flag.NewFlagSet("project-v14-finalize", flag.ExitOnError)
	var paths goupb10cmdload.Paths
	paths.Bind(set)
	resultBundle := set.String("result-bundle", "", "result Project-v12 bundle")
	resultPlacement := set.String("result-placement", "", "result Project-v13 placement")
	resultSelection := set.String("result-selection", "", "result configuration selection")
	reconciliation := set.String("reconciliation", "", "reconciled native project")
	language := set.String("language", "javascript", "source provider language")
	projectIdentity := set.String("project-identity", "example.test/javascript-upb05", "project semantic identity")
	targetName := set.String("target-name", "wasm32-pulp-javascript-host-v1", "target policy name")
	ruleNamespace := set.String("rule-namespace", "javascript-upb10", "target realization namespace")
	reconciliationLabel := set.String("reconciliation-label", "seme-javascript-reconciliation-v1", "metadata manifest label")
	identityEvidenceRequired := set.Bool("identity-evidence-required", true, "require provider identity evidence")
	patchContract := set.String("patch", "", "Patch-v1 contract")
	languageService := set.String("language-service", "", "Language Service-v1 contract")
	projectV14 := set.String("project-v14", "", "Project-v14 contract")
	out := set.String("out", "", "new reconciliation authority")
	set.Parse(os.Args[1:])
	if set.NArg() != 0 || *resultBundle == "" || *resultPlacement == "" || *resultSelection == "" || *reconciliation == "" || *patchContract == "" || *languageService == "" || *projectV14 == "" || *out == "" {
		fatal("arguments")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	policy := goprojectplacementadapter.Policy{Name: *targetName, Revision: 1, RuleNamespace: *ruleNamespace, AllowedFidelity: []targetplaninstance.Fidelity{targetplaninstance.Exact, targetplaninstance.NativeIsland}}
	prior, err := goupb10cmdload.Load(ctx, paths, policy)
	if err != nil {
		fatal(fmt.Errorf("prior:%w", err))
	}
	resultPaths := paths
	resultPaths.Base.Base.Bundle = *resultBundle
	resultPaths.Base.Base.ConfigurationSelection = *resultSelection
	resultPaths.Placement = *resultPlacement
	result, err := goupb10cmdload.Load(ctx, resultPaths, policy)
	if err != nil {
		fatal(fmt.Errorf("result:%w", err))
	}
	metadata := filepath.Join(*reconciliation, ".seme-reconciliation-v1")
	reportBytes := mustRead(filepath.Join(metadata, "projection-report.json"))
	priorG1 := mustRead(filepath.Join(metadata, "prior-provider.g1"))
	resultG1 := mustRead(filepath.Join(metadata, "result-provider.g1"))
	transcript := mustRead(filepath.Join(metadata, "native-validation.txt"))
	artifacts := map[string][]byte{
		"native-validation.txt":  transcript,
		"prior-provider.g1":      priorG1,
		"projection-report.json": reportBytes,
		"result-provider.g1":     resultG1,
	}
	if *identityEvidenceRequired {
		artifacts["identity-evidence.json"] = mustRead(filepath.Join(metadata, "identity-evidence.json"))
	}
	if !bytes.Equal(mustRead(filepath.Join(metadata, "COMPLETE.sha256")), reconciliationManifest(*reconciliationLabel, artifacts)) {
		fatal("reconciliation_manifest")
	}
	var edit report
	if json.Unmarshal(reportBytes, &edit) != nil || edit.Version != 1 || edit.Language != *language || edit.Project != *projectIdentity || edit.ClientRevision < 2 || len(edit.Occurrences) < 2 {
		fatal("reconciliation_report")
	}
	if !bytes.Equal(priorG1, prior.Bundle.Artifacts.Base.Base.Base.Construction) || !bytes.Equal(resultG1, result.Bundle.Artifacts.Base.Base.Base.Construction) {
		fatal("construction_fixed_point")
	}
	contractsV13, err := goupb10cmdload.ResolveContracts(paths)
	if err != nil {
		fatal(err)
	}
	contracts, err := contractcatalog.ResolveProjectContractSetV14FromV13(contractsV13, mustRead(*patchContract), mustRead(*languageService), mustRead(*projectV14))
	if err != nil {
		fatal(err)
	}
	priorGraph, err := wire.Decode(prior.Placement.ProjectV13)
	if err != nil {
		fatal(err)
	}
	resultGraph, err := wire.Decode(result.Placement.ProjectV13)
	if err != nil {
		fatal(err)
	}
	target, err := wire.ParseID(edit.Target)
	if err != nil {
		fatal(err)
	}
	field, err := wire.ParseID(edit.Field)
	if err != nil {
		fatal(err)
	}
	before, beforeOK := priorGraph.Entities[target]
	after, afterOK := resultGraph.Entities[target]
	beforeName, beforeField := before.Fields[field]
	afterName, afterField := after.Fields[field]
	if !beforeOK || !afterOK || !beforeField || !afterField || beforeName.Tag != 5 || afterName.Tag != 5 || string(beforeName.Bytes) != edit.Expected || string(afterName.Bytes) != edit.Replacement {
		fatal("identity_transition")
	}
	patch, err := patchinstance.EmitRename(contracts.Patch(), patchinstance.Rename{Base: priorGraph, Target: target, Field: field, Expected: beforeName.Bytes, Replacement: afterName.Bytes})
	if err != nil {
		fatal(err)
	}
	projectInput := projectv14instance.Inputs{Contracts: contracts, Prior: prior.Bundle.Project, Result: result.Bundle.Project, Patch: patch, ClientRevision: edit.ClientRevision, NativeValidationTranscript: transcript}
	project, err := projectv14instance.Emit(projectInput)
	if err != nil {
		fatal(err)
	}
	projectInput.Artifact = project
	if err = projectv14instance.Validate(projectInput); err != nil {
		fatal(err)
	}
	publish(*out, map[string][]byte{"patch-v1.seme": patch, "project-v14.seme": project, "native-validation.txt": transcript, "projection-report.json": reportBytes})
}

func reconciliationManifest(label string, artifacts map[string][]byte) []byte {
	names := make([]string, 0, len(artifacts))
	for name := range artifacts {
		names = append(names, name)
	}
	sort.Strings(names)
	manifest := []byte(label + "\n")
	for _, name := range names {
		digest := sha256.Sum256(artifacts[name])
		manifest = append(manifest, []byte(name+" "+hex.EncodeToString(digest[:])+"\n")...)
	}
	return manifest
}

func mustRead(path string) []byte {
	real, err := filepath.EvalSymlinks(path)
	if err != nil || real != path {
		fatal("symlink")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64<<20 {
		fatal("regular")
	}
	value, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	return value
}
func publish(destination string, artifacts map[string][]byte) {
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		fatal("output_exists")
	}
	parent := filepath.Dir(destination)
	real, err := filepath.EvalSymlinks(parent)
	if err != nil || real != parent {
		fatal("output_parent")
	}
	stage, err := os.MkdirTemp(parent, ".seme-project-v14-")
	if err != nil {
		fatal(err)
	}
	keep := false
	defer func() {
		if !keep {
			os.RemoveAll(stage)
		}
	}()
	names := make([]string, 0, len(artifacts))
	for name, value := range artifacts {
		names = append(names, name)
		if err = os.WriteFile(filepath.Join(stage, name), value, 0600); err != nil {
			fatal(err)
		}
	}
	sort.Strings(names)
	manifest := []byte("seme-project-v14-reconciliation-v1\n")
	for _, name := range names {
		digest := sha256.Sum256(artifacts[name])
		manifest = append(manifest, []byte(name+" "+hex.EncodeToString(digest[:])+"\n")...)
	}
	if err = os.WriteFile(filepath.Join(stage, "COMPLETE.sha256"), manifest, 0600); err != nil {
		fatal(err)
	}
	if err = os.Rename(stage, destination); err != nil {
		fatal(err)
	}
	keep = true
}
func fatal(value any) { fmt.Fprintln(os.Stderr, "project-v14-finalize:", value); os.Exit(1) }
