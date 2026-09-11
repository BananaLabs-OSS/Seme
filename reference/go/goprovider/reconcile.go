package goprovider

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const reconciliationDirectory = ".seme-reconciliation-v1"

type ReconciliationBundle struct {
	Report                     ProjectionReport
	Prior, Result              Manifest
	PriorG1, ResultG1          []byte
	NativeValidationTranscript []byte
}

// ReadReconciliationBundle independently authenticates a published native
// reconciliation directory and reproduces its provider graphs from source.
func ReadReconciliationBundle(root, providerModuleG1 string) (ReconciliationBundle, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return ReconciliationBundle{}, errors.New("provider.reconciliation_path")
	}
	real, err := filepath.EvalSymlinks(root)
	if err != nil || real != root {
		return ReconciliationBundle{}, errors.New("provider.reconciliation_root")
	}
	metadata := filepath.Join(root, reconciliationDirectory)
	info, err := os.Lstat(metadata)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return ReconciliationBundle{}, errors.New("provider.reconciliation_metadata")
	}
	names := []string{"native-validation.txt", "prior-provider.g1", "prior-provider.json", "projection-report.json", "result-provider.g1", "result-provider.json"}
	artifacts := map[string][]byte{}
	for _, name := range names {
		path := filepath.Join(metadata, name)
		entry, statErr := os.Lstat(path)
		if statErr != nil || !entry.Mode().IsRegular() || entry.Mode()&os.ModeSymlink != 0 || entry.Size() <= 0 || entry.Size() > 64<<20 {
			return ReconciliationBundle{}, errors.New("provider.reconciliation_artifact")
		}
		value, readErr := os.ReadFile(path)
		if readErr != nil || int64(len(value)) != entry.Size() {
			return ReconciliationBundle{}, errors.New("provider.reconciliation_artifact")
		}
		artifacts[name] = value
	}
	entries, err := os.ReadDir(metadata)
	if err != nil || len(entries) != len(names)+1 {
		return ReconciliationBundle{}, errors.New("provider.reconciliation_file_set")
	}
	completePath := filepath.Join(metadata, "COMPLETE.sha256")
	completeInfo, statErr := os.Lstat(completePath)
	if statErr != nil || !completeInfo.Mode().IsRegular() || completeInfo.Mode()&os.ModeSymlink != 0 || completeInfo.Size() <= 0 || completeInfo.Size() > 1<<20 {
		return ReconciliationBundle{}, errors.New("provider.reconciliation_manifest")
	}
	complete, err := os.ReadFile(completePath)
	if err != nil || int64(len(complete)) != completeInfo.Size() || !bytes.Equal(complete, reconciliationManifest(artifacts)) {
		return ReconciliationBundle{}, errors.New("provider.reconciliation_manifest")
	}
	var prior, result Manifest
	var report ProjectionReport
	if json.Unmarshal(artifacts["prior-provider.json"], &prior) != nil || json.Unmarshal(artifacts["result-provider.json"], &result) != nil || json.Unmarshal(artifacts["projection-report.json"], &report) != nil ||
		prior.Version != ManifestVersion || result.Version != ManifestVersion || prior.Contract != "provider-v1" || result.Contract != "provider-v1" ||
		report.BaseRevision != prior.Revision || report.ResultRevision != result.Revision || len(report.IdentityBindings) != 1 {
		return ReconciliationBundle{}, errors.New("provider.reconciliation_records")
	}
	current, err := NativeRevision(root)
	if err != nil || current != result.Revision {
		return ReconciliationBundle{}, errors.New("provider.reconciliation_native_revision")
	}
	reproduced, resultG1, err := Ingest(IngestOptions{Project: root, ModuleG1: providerModuleG1, Prior: &prior})
	if err != nil {
		return ReconciliationBundle{}, err
	}
	reproducedJSON, _ := json.MarshalIndent(reproduced, "", "  ")
	reproducedJSON = append(reproducedJSON, '\n')
	if !bytes.Equal(reproducedJSON, artifacts["result-provider.json"]) || !bytes.Equal([]byte(resultG1), artifacts["result-provider.g1"]) {
		return ReconciliationBundle{}, errors.New("provider.reconciliation_reingest")
	}
	module, err := os.ReadFile(providerModuleG1)
	if err != nil {
		return ReconciliationBundle{}, err
	}
	priorG1, err := emitG1(module, prior)
	if err != nil || !bytes.Equal([]byte(priorG1), artifacts["prior-provider.g1"]) {
		return ReconciliationBundle{}, errors.New("provider.reconciliation_prior_graph")
	}
	binding := report.IdentityBindings[0]
	before, after := declarationByID(prior, binding.ID), declarationByID(result, binding.ID)
	if before == nil || after == nil || before.Name == after.Name || after.Name != binding.Name || before.Qualified == after.Qualified || !strings.HasSuffix(after.Qualified, "."+after.Name) {
		return ReconciliationBundle{}, errors.New("provider.reconciliation_identity_transition")
	}
	return ReconciliationBundle{Report: report, Prior: prior, Result: result, PriorG1: bytes.Clone(artifacts["prior-provider.g1"]), ResultG1: bytes.Clone(artifacts["result-provider.g1"]), NativeValidationTranscript: bytes.Clone(artifacts["native-validation.txt"])}, nil
}

func declarationByID(manifest Manifest, identity string) *Declaration {
	for i := range manifest.Declarations {
		if manifest.Declarations[i].ID == identity {
			return &manifest.Declarations[i]
		}
	}
	return nil
}

func reconciliationManifest(artifacts map[string][]byte) []byte {
	names := make([]string, 0, len(artifacts))
	for name := range artifacts {
		names = append(names, name)
	}
	sort.Strings(names)
	complete := []byte("seme-go-reconciliation-v1\n")
	for _, name := range names {
		digest := sha256.Sum256(artifacts[name])
		complete = append(complete, []byte(name+" "+hex.EncodeToString(digest[:])+"\n")...)
	}
	return complete
}

// ProjectRenameBundle atomically publishes the complete projected native
// project together with its before/after provider graphs and validation
// evidence. Nothing appears at destination unless re-ingestion succeeds.
func ProjectRenameBundle(project, destination, providerModuleG1 string, prior Manifest, target, expected, replacement string) (ReconciliationBundle, error) {
	parent := filepath.Dir(destination)
	stage, err := os.MkdirTemp(parent, ".seme-go-reconcile-")
	if err != nil {
		return ReconciliationBundle{}, err
	}
	defer os.RemoveAll(stage)
	candidate := filepath.Join(stage, "project")
	report, transcript, err := ProjectRename(project, candidate, prior, target, expected, replacement, true)
	if err != nil {
		return ReconciliationBundle{}, err
	}
	result, resultG1, err := Ingest(IngestOptions{Project: candidate, ModuleG1: providerModuleG1, Prior: &prior})
	if err != nil {
		return ReconciliationBundle{}, err
	}
	if result.Revision != report.ResultRevision {
		return ReconciliationBundle{}, errors.New("provider.reconciliation_revision")
	}
	priorModule, err := os.ReadFile(providerModuleG1)
	if err != nil {
		return ReconciliationBundle{}, err
	}
	priorG1, err := emitG1(priorModule, prior)
	if err != nil {
		return ReconciliationBundle{}, err
	}
	metadata := filepath.Join(candidate, reconciliationDirectory)
	if _, err = os.Lstat(metadata); !os.IsNotExist(err) {
		return ReconciliationBundle{}, errors.New("provider.reconciliation_metadata_exists")
	}
	if err = os.Mkdir(metadata, 0o700); err != nil {
		return ReconciliationBundle{}, err
	}
	encode := func(value any) ([]byte, error) {
		data, marshalErr := json.MarshalIndent(value, "", "  ")
		return append(data, '\n'), marshalErr
	}
	priorJSON, err := encode(prior)
	if err != nil {
		return ReconciliationBundle{}, err
	}
	resultJSON, err := encode(result)
	if err != nil {
		return ReconciliationBundle{}, err
	}
	reportJSON, err := encode(report)
	if err != nil {
		return ReconciliationBundle{}, err
	}
	artifacts := map[string][]byte{
		"prior-provider.json": priorJSON, "result-provider.json": resultJSON,
		"prior-provider.g1": []byte(priorG1), "result-provider.g1": []byte(resultG1),
		"projection-report.json": reportJSON, "native-validation.txt": transcript,
	}
	for name, value := range artifacts {
		if len(value) == 0 {
			return ReconciliationBundle{}, errors.New("provider.reconciliation_artifact_empty")
		}
		if err = atomicWrite(filepath.Join(metadata, name), value, 0o600); err != nil {
			return ReconciliationBundle{}, err
		}
	}
	complete := reconciliationManifest(artifacts)
	if err = atomicWrite(filepath.Join(metadata, "COMPLETE.sha256"), complete, 0o600); err != nil {
		return ReconciliationBundle{}, err
	}
	if _, err = os.Lstat(destination); !os.IsNotExist(err) {
		return ReconciliationBundle{}, errors.New("provider.destination_exists")
	}
	if err = os.Rename(candidate, destination); err != nil {
		return ReconciliationBundle{}, err
	}
	return ReconciliationBundle{Report: report, Prior: prior, Result: result, PriorG1: bytes.Clone([]byte(priorG1)), ResultG1: bytes.Clone([]byte(resultG1)), NativeValidationTranscript: bytes.Clone(transcript)}, nil
}
