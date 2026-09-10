// Package godurablemanifest parses the bounded project-facing durable selection.
package godurablemanifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"seme.local/reference/godurableadapter"
)

const Version = "seme.durable-state-selection/v1"
const MaxBytes = 1 << 20

type named struct {
	Package string `json:"package"`
	Name    string `json:"name"`
}
type document struct {
	Version             string `json:"version"`
	Identity            string `json:"identity"`
	StateOwner          string `json:"state_owner"`
	PortIdentity        string `json:"port_identity"`
	PortOwner           string `json:"port_owner"`
	Version1            named  `json:"version_1_type"`
	Version2            named  `json:"version_2_type"`
	Validator1          named  `json:"validator_1"`
	Validator2          named  `json:"validator_2"`
	Migration           named  `json:"migration"`
	ErrorType           string `json:"error_type"`
	KeyType             string `json:"key_type"`
	Codec               string `json:"codec"`
	MaximumPayloadBytes uint64 `json:"maximum_payload_bytes"`
	MaximumKeyBytes     uint64 `json:"maximum_key_bytes"`
}

func Parse(data []byte) (godurableadapter.Selection, error) {
	if len(data) == 0 || len(data) > MaxBytes {
		return godurableadapter.Selection{}, fmt.Errorf("durable_manifest.size")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var x document
	if err := d.Decode(&x); err != nil {
		return godurableadapter.Selection{}, fmt.Errorf("durable_manifest.json:%w", err)
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return godurableadapter.Selection{}, fmt.Errorf("durable_manifest.trailing")
	}
	if x.Version != Version || x.ErrorType != "int64" || x.KeyType != "string" || x.Codec != "seme.durable-state.canonical.v1" {
		return godurableadapter.Selection{}, fmt.Errorf("durable_manifest.profile")
	}
	c := func(n named) godurableadapter.Named { return godurableadapter.Named{Package: n.Package, Name: n.Name} }
	return godurableadapter.Selection{Identity: x.Identity, StateOwner: x.StateOwner, PortIdentity: x.PortIdentity, PortOwner: x.PortOwner, Version1: c(x.Version1), Version2: c(x.Version2), Validator1: c(x.Validator1), Validator2: c(x.Validator2), Migration: c(x.Migration), MaximumPayloadBytes: x.MaximumPayloadBytes, MaximumKeyBytes: x.MaximumKeyBytes}, nil
}
