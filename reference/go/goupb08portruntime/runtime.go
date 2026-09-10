// Package goupb08portruntime binds an authenticated selected TransportPort to
// the project-neutral Ordered Transport host executor.
package goupb08portruntime

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"

	"seme.local/reference/orderedtransportinstance"
	"seme.local/reference/orderedtransportplacement"
	"seme.local/reference/orderedtransportruntime"
)

type Store interface {
	Put([]byte, []byte) error
	Get([]byte) ([]byte, bool)
}
type HostPort struct {
	Profile  orderedtransportruntime.Profile
	Layouts  orderedtransportruntime.Layouts
	Input    io.Reader
	Output   io.Writer
	Store    Store
	FailSend error
}

func (p *HostPort) Receive(r orderedtransportruntime.ReceiveRequest) orderedtransportruntime.ReceiveOutcome {
	if r.OperationIdentity != p.Profile.Receive.Identity {
		return orderedtransportruntime.ReceiveOutcome{Error: &orderedtransportruntime.PortError{Identity: "seme.transport.receive.operation", OperationIdentity: r.OperationIdentity}}
	}
	f, e := orderedtransportruntime.ReadFrame(p.Profile, p.Layouts, p.Input)
	if e != nil {
		return orderedtransportruntime.ReceiveOutcome{Error: &orderedtransportruntime.PortError{Identity: "seme.transport.receive.frame", OperationIdentity: r.OperationIdentity}}
	}
	b, e := orderedtransportruntime.EncodeFrame(p.Profile, p.Layouts, f)
	if e != nil {
		return orderedtransportruntime.ReceiveOutcome{Error: &orderedtransportruntime.PortError{Identity: "seme.transport.receive.frame", OperationIdentity: r.OperationIdentity}}
	}
	return orderedtransportruntime.ReceiveOutcome{Frame: b}
}
func (p *HostPort) Send(r orderedtransportruntime.SendRequest) orderedtransportruntime.SendOutcome {
	if r.OperationIdentity != p.Profile.Send.Identity {
		return orderedtransportruntime.SendOutcome{Error: &orderedtransportruntime.PortError{Identity: "seme.transport.send.operation", OperationIdentity: r.OperationIdentity}}
	}
	if p.FailSend != nil {
		return orderedtransportruntime.SendOutcome{Error: &orderedtransportruntime.PortError{Identity: "seme.transport.send.failed", OperationIdentity: r.OperationIdentity}}
	}
	if _, e := orderedtransportruntime.DecodeFrame(p.Profile, p.Layouts, r.Frame); e != nil {
		return orderedtransportruntime.SendOutcome{Error: &orderedtransportruntime.PortError{Identity: "seme.transport.send.frame", OperationIdentity: r.OperationIdentity}}
	}
	sum := sha256.Sum256(r.Frame)
	receipt := bytes.Clone(sum[:])
	if p.Store == nil || p.Store.Put(receipt, bytes.Clone(r.Frame)) != nil {
		return orderedtransportruntime.SendOutcome{Error: &orderedtransportruntime.PortError{Identity: "seme.transport.send.store", OperationIdentity: r.OperationIdentity}}
	}
	if p.Output != nil {
		if e := orderedtransportruntime.WriteFrame(p.Profile, p.Layouts, p.Output, mustDecode(p.Profile, p.Layouts, r.Frame)); e != nil {
			return orderedtransportruntime.SendOutcome{Error: &orderedtransportruntime.PortError{Identity: "seme.transport.send.write", OperationIdentity: r.OperationIdentity}}
		}
	}
	return orderedtransportruntime.SendOutcome{Receipt: receipt, AcceptedSHA256: sum}
}
func (p *HostPort) Evidence(receipt []byte) ([]byte, bool) {
	if p.Store == nil {
		return nil, false
	}
	b, ok := p.Store.Get(bytes.Clone(receipt))
	return bytes.Clone(b), ok
}
func mustDecode(p orderedtransportruntime.Profile, l orderedtransportruntime.Layouts, b []byte) orderedtransportruntime.Frame {
	f, e := orderedtransportruntime.DecodeFrame(p, l, b)
	if e != nil {
		panic(e)
	}
	return f
}

func ExecuteSelected(in orderedtransportinstance.Inputs, portIdentity string, layouts orderedtransportruntime.Layouts, port orderedtransportruntime.Port, handler orderedtransportruntime.Handler) (orderedtransportruntime.Result, error) {
	if e := orderedtransportinstance.Validate(in); e != nil {
		return orderedtransportruntime.Result{}, e
	}
	selected := false
	for _, p := range in.Model.Ports {
		if p.Identity == portIdentity {
			selected = true
		}
	}
	if !selected {
		return orderedtransportruntime.Result{}, fmt.Errorf("go_upb08_port_runtime.port_not_selected")
	}
	profile, e := orderedtransportruntime.AuthenticatedProfile(in.Contracts.OrderedTransport())
	if e != nil {
		return orderedtransportruntime.Result{}, e
	}
	grants := orderedtransportruntime.Grants{profile.Receive.Capability: true, profile.Send.Capability: true}
	return orderedtransportruntime.Execute(profile, layouts, grants, port, handler), nil
}

type PlacementReport struct {
	PlannerProvider        string `json:"planner_provider"`
	PlannerCarrier         string `json:"planner_carrier"`
	TransportPortSupported bool   `json:"transport_port_supported"`
	Reason                 string `json:"reason"`
}

func RejectPulpTransportPort(c orderedtransportplacement.Candidate) (PlacementReport, error) {
	e := orderedtransportplacement.VerifyAuthoritativeTransportPort(c)
	if e == nil {
		return PlacementReport{}, fmt.Errorf("go_upb08_port_runtime.unexpected_pulp_port")
	}
	return PlacementReport{PlannerProvider: c.Provider, PlannerCarrier: c.Carrier, TransportPortSupported: false, Reason: e.Error()}, nil
}
