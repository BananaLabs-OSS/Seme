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
)

const reconciliationDirectory = ".seme-reconciliation-v1"

type ReconciliationBundle struct {
	Report                     ProjectionReport
	Prior, Result              Manifest
	PriorG1, ResultG1          []byte
	NativeValidationTranscript []byte
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
