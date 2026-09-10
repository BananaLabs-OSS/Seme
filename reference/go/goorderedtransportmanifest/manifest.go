// Package goorderedtransportmanifest parses bounded Go-facing transport selection evidence.
package goorderedtransportmanifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"seme.local/reference/goorderedtransportadapter"
)

const Version = "seme.ordered-transport-selection/v1"
const MaxBytes = 1 << 20

type named struct {
	Package string `json:"package"`
	Name    string `json:"name"`
}
type owned struct {
	Identity string `json:"identity"`
	Package  string `json:"package"`
}
type kind struct {
	Identity string `json:"identity"`
	Owner    string `json:"owner"`
	Payload  named  `json:"payload"`
}
type document struct {
	Version      string  `json:"version"`
	Streams      []owned `json:"streams"`
	CommandKinds []kind  `json:"command_kinds"`
	EventKinds   []kind  `json:"event_kinds"`
	Ports        []owned `json:"ports"`
	Dispatch     named   `json:"dispatch"`
	Replay       named   `json:"replay"`
}

func Parse(data []byte) (goorderedtransportadapter.Selection, error) {
	if len(data) == 0 || len(data) > MaxBytes {
		return goorderedtransportadapter.Selection{}, fmt.Errorf("ordered_transport_manifest.size")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var x document
	if err := d.Decode(&x); err != nil {
		return goorderedtransportadapter.Selection{}, fmt.Errorf("ordered_transport_manifest.json:%w", err)
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return goorderedtransportadapter.Selection{}, fmt.Errorf("ordered_transport_manifest.trailing")
	}
	if x.Version != Version {
		return goorderedtransportadapter.Selection{}, fmt.Errorf("ordered_transport_manifest.version")
	}
	c := func(n named) goorderedtransportadapter.Named {
		return goorderedtransportadapter.Named{Package: n.Package, Name: n.Name}
	}
	r := goorderedtransportadapter.Selection{Dispatch: c(x.Dispatch), Replay: c(x.Replay)}
	for _, v := range x.Streams {
		r.Streams = append(r.Streams, goorderedtransportadapter.OwnedIdentity{Identity: v.Identity, Package: v.Package})
	}
	for _, v := range x.Ports {
		r.Ports = append(r.Ports, goorderedtransportadapter.OwnedIdentity{Identity: v.Identity, Package: v.Package})
	}
	for _, v := range x.CommandKinds {
		r.CommandKinds = append(r.CommandKinds, goorderedtransportadapter.KindSelection{Identity: v.Identity, Owner: v.Owner, Payload: c(v.Payload)})
	}
	for _, v := range x.EventKinds {
		r.EventKinds = append(r.EventKinds, goorderedtransportadapter.KindSelection{Identity: v.Identity, Owner: v.Owner, Payload: c(v.Payload)})
	}
	return r, nil
}
