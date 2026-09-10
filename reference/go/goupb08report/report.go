// Package goupb08report exposes deterministic source-free transport authority.
// It reports authenticated declarations only; it makes no runtime-parity claim.
package goupb08report

import (
	"encoding/json"
	"fmt"
	"sort"

	"seme.local/reference/goupb08bundle"
	"seme.local/reference/orderedtransportinstance"
	"seme.local/reference/projectv11instance"
	"seme.local/reference/wire"
)

type Owned struct {
	Identity    string `json:"identity"`
	Owner       string `json:"owner"`
	PayloadType string `json:"payload_type,omitempty"`
}
type Report struct {
	ProjectContractRevision     string  `json:"project_contract_revision"`
	TransportContractRevision   string  `json:"transport_contract_revision"`
	ProjectArtifactRevision     string  `json:"project_artifact_revision"`
	TransportArtifactRevision   string  `json:"transport_artifact_revision"`
	Codec                       string  `json:"codec"`
	MaximumRetainedPayloadBytes uint64  `json:"maximum_retained_payload_bytes"`
	Streams                     []Owned `json:"streams"`
	CommandKinds                []Owned `json:"command_kinds"`
	EventKinds                  []Owned `json:"event_kinds"`
	Ports                       []Owned `json:"ports"`
	DispatchFunction            string  `json:"dispatch_function"`
	ReplayFunction              string  `json:"replay_function"`
}

func Inspect(in goupb08bundle.Result) (Report, error) {
	if err := orderedtransportinstance.Validate(in.Transport); err != nil {
		return Report{}, fmt.Errorf("go_upb08_report.transport:%w", err)
	}
	if err := projectv11instance.Validate(in.Project); err != nil {
		return Report{}, fmt.Errorf("go_upb08_report.project:%w", err)
	}
	t, err := wire.Decode(in.Artifacts.Transport)
	if err != nil {
		return Report{}, err
	}
	p, err := wire.Decode(in.Artifacts.ProjectV11)
	if err != nil {
		return Report{}, err
	}
	a, err := orderedtransportinstance.Authenticate(in.Transport.Contracts.OrderedTransport())
	if err != nil {
		return Report{}, err
	}
	m := in.Transport.Model
	r := Report{ProjectContractRevision: in.Project.Contracts.Project().Pin().Revision.String(), TransportContractRevision: in.Transport.Contracts.OrderedTransport().Pin().Revision.String(), ProjectArtifactRevision: p.Revision.String(), TransportArtifactRevision: t.Revision.String(), Codec: a.CodecIdentity, MaximumRetainedPayloadBytes: a.MaximumRetainedPayloadBytes, DispatchFunction: m.DispatchFunction.String(), ReplayFunction: m.ReplayFunction.String()}
	for _, x := range m.Streams {
		r.Streams = append(r.Streams, Owned{Identity: x.Identity, Owner: x.Owner.String()})
	}
	for _, x := range m.CommandKinds {
		r.CommandKinds = append(r.CommandKinds, Owned{Identity: x.Identity, Owner: x.Owner.String(), PayloadType: x.PayloadType.String()})
	}
	for _, x := range m.EventKinds {
		r.EventKinds = append(r.EventKinds, Owned{Identity: x.Identity, Owner: x.Owner.String(), PayloadType: x.PayloadType.String()})
	}
	for _, x := range m.Ports {
		r.Ports = append(r.Ports, Owned{Identity: x.Identity, Owner: x.Owner.String()})
	}
	less := func(x []Owned) {
		sort.Slice(x, func(i, j int) bool {
			if x[i].Owner == x[j].Owner {
				return x[i].Identity < x[j].Identity
			}
			return x[i].Owner < x[j].Owner
		})
	}
	less(r.Streams)
	less(r.CommandKinds)
	less(r.EventKinds)
	less(r.Ports)
	return r, nil
}
func Marshal(r Report) ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
