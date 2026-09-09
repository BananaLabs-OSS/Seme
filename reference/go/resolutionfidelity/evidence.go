// Package resolutionfidelity emits checked UPB-01 evidence. It describes
// source resolution only; executable Program target plans remain independent.
package resolutionfidelity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"seme.local/reference/projectsource"
)

type Evidence struct {
	Version           uint64                   `json:"version"`
	InventoryRevision string                   `json:"inventory_revision"`
	UABProfile        string                   `json:"uab_profile"`
	Toolchain         ToolchainEvidence        `json:"toolchain"`
	Classifications   []ClassificationEvidence `json:"classifications"`
	Inventory         BoundaryEvidence         `json:"source_inventory"`
	BehaviorProgram   ProgramEvidence          `json:"behavior_program"`
}
type ToolchainEvidence struct {
	Language         string `json:"language"`
	Toolchain        string `json:"toolchain"`
	ProviderProfile  string `json:"provider_profile"`
	SemanticRevision string `json:"semantic_revision"`
}
type ClassificationEvidence struct {
	Class     string `json:"class"`
	Fidelity  string `json:"fidelity"`
	Scope     string `json:"scope"`
	Placement string `json:"placement"`
	Wasm      bool   `json:"wasm"`
}
type BoundaryEvidence struct {
	Placement string `json:"placement"`
	Wasm      bool   `json:"wasm"`
	Role      string `json:"role"`
}
type ProgramEvidence struct {
	SeparateTargetPlan bool   `json:"separate_target_plan"`
	Claim              string `json:"claim"`
}

func Build(snapshot projectsource.Snapshot, uabProfile string) ([]byte, error) {
	if err := projectsource.ValidateSnapshot(snapshot); err != nil {
		return nil, err
	}
	if snapshot.Toolchain.Language != "go" || uabProfile == "" {
		return nil, fmt.Errorf("resolution_fidelity.profile")
	}
	e := expected(snapshot, uabProfile)
	out, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func Validate(snapshot projectsource.Snapshot, uabProfile string, source []byte) error {
	if err := projectsource.ValidateSnapshot(snapshot); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.DisallowUnknownFields()
	var got Evidence
	if err := decoder.Decode(&got); err != nil {
		return fmt.Errorf("resolution_fidelity.json:%w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("resolution_fidelity.trailing")
	}
	want, err := Build(snapshot, uabProfile)
	if err != nil {
		return err
	}
	canonical, err := json.Marshal(got)
	if err != nil {
		return err
	}
	canonical = append(canonical, '\n')
	if !bytes.Equal(canonical, source) {
		return fmt.Errorf("resolution_fidelity.noncanonical")
	}
	if !bytes.Equal(source, want) {
		return fmt.Errorf("resolution_fidelity.claim_mismatch")
	}
	return nil
}

func expected(snapshot projectsource.Snapshot, uab string) Evidence {
	return Evidence{Version: 1, InventoryRevision: snapshot.ContentRevision, UABProfile: uab, Toolchain: ToolchainEvidence{Language: snapshot.Toolchain.Language, Toolchain: snapshot.Toolchain.Toolchain, ProviderProfile: snapshot.Toolchain.Profile, SemanticRevision: snapshot.Toolchain.SemanticRevision}, Classifications: []ClassificationEvidence{
		{Class: "tracked", Fidelity: "exact", Scope: "declared-uab-provider-profile-only", Placement: "semantic-provider", Wasm: false},
		{Class: "ignored", Fidelity: "host-byte-preserved", Scope: "bytes-only", Placement: "host-boundary", Wasm: false},
		{Class: "generated", Fidelity: "host-byte-preserved", Scope: "bytes-only", Placement: "host-boundary", Wasm: false},
		{Class: "vendored", Fidelity: "native-island-opaque", Scope: "no-semantic-claim", Placement: "native-island", Wasm: false},
		{Class: "opaque", Fidelity: "host-byte-preserved", Scope: "bytes-only", Placement: "host-boundary", Wasm: false},
	}, Inventory: BoundaryEvidence{Placement: "host-boundary", Wasm: false, Role: "resolution-metadata-only"}, BehaviorProgram: ProgramEvidence{SeparateTargetPlan: true, Claim: "none-from-source-inventory"}}
}
