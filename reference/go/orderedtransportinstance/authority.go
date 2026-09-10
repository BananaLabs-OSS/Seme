// Package orderedtransportinstance authenticates the immutable authority
// encoded by the Ordered Transport Contract v1. It performs no network I/O.
package orderedtransportinstance

import (
	"fmt"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/wire"
)

const (
	CodecIdentity                  = "seme.ordered-transport.binary.v1"
	DigestIdentity                 = "sha256"
	MaximumCommands         uint64 = 256
	MaximumEvents           uint64 = 1024
	MaximumEventsPerCommand uint64 = 4
	MaximumPayloadBytes     uint64 = 3072
	MaximumFrameBytes       uint64 = 4096
	CorrelationBytes        uint64 = 16
	MaximumStreamBytes      uint64 = 128
)

type Authority struct {
	ReceiveIdentity, SendIdentity                                                string
	ReceiveCapability, SendCapability                                            string
	CodecIdentity, DigestIdentity                                                string
	MaximumCommands, MaximumEvents, MaximumEventsPerCommand                      uint64
	MaximumPayloadBytes, MaximumFrameBytes, CorrelationBytes, MaximumStreamBytes uint64
	FirstCommandSequence, FirstEventSequence                                     uint64
	CommandTerminalSentinel, EventTerminalSentinel                               uint64
	DuplicatePolicy, GapPolicy, CorrelationPolicy                                string
	CodecLayout                                                                  [4]string
}

func Authenticate(c contractcatalog.Contract) (Authority, error) {
	if !c.Validated() || c.Pin() != (contractcatalog.Pin{Module: id("2000"), Revision: id("2001")}) {
		return Authority{}, fmt.Errorf("ordered_transport.authority_contract")
	}
	e := c.Envelope()
	schema, ok := e.Entities[id("201e")]
	if !ok || schema.Schema != id("10") || schema.Version != 1 {
		return Authority{}, fmt.Errorf("ordered_transport.authority_schema")
	}
	listed := schema.Fields[id("101")]
	if listed.Tag != 7 || len(listed.List) != 31 {
		return Authority{}, fmt.Errorf("ordered_transport.authority_fields")
	}
	names := map[wire.ID]string{}
	for _, item := range listed.List {
		f, exists := e.Entities[item.Reference]
		n := f.Fields[id("110")]
		if item.Tag != 6 || !exists || f.Schema != id("11") || n.Tag != 5 {
			return Authority{}, fmt.Errorf("ordered_transport.authority_field")
		}
		names[item.Reference] = string(n.Bytes)
	}
	wantNames := map[string]string{
		"21e0": "seme.transport.receive.v1", "21e1": "seme.transport.send.v1", "21e2": "seme.transport.receive.v1.capability", "21e3": "seme.transport.send.v1.capability",
		"21e4": "seme.transport.sequence.0.receive", "21e5": "seme.transport.sequence.1.send", "21e6": CodecIdentity, "21e7": DigestIdentity,
		"21e8": "ordered_transport.maximum_commands.256", "21e9": "ordered_transport.maximum_events.1024", "21ea": "ordered_transport.maximum_events_per_command.4", "21eb": "ordered_transport.maximum_payload_bytes.3072", "21ec": "ordered_transport.maximum_frame_bytes.4096", "21ed": "ordered_transport.correlation_bytes.16", "21ee": "ordered_transport.maximum_stream_bytes.128",
		"21ef": "ordered_transport.first_command_sequence.1", "21f0": "ordered_transport.first_event_sequence.1", "21f1": "ordered_transport.command_terminal_sentinel.257", "21f2": "ordered_transport.event_terminal_sentinel.1025", "21f3": "ordered_transport.duplicate_policy.exact_cached_response", "21f4": "ordered_transport.gap_policy.reject_without_buffering", "21f5": "ordered_transport.correlation_policy.unique_per_command",
		"21f6": "ordered_transport.codec.u32le_body_length_excluding_prefix", "21f7": "ordered_transport.codec.magic.SEMEOT01", "21f8": "ordered_transport.codec.u8_frame_kind_then_pure_value_v1", "21f9": "ordered_transport.codec.exact_length_no_trailing_bytes",
	}
	for field, want := range wantNames {
		if names[id(field)] != want {
			return Authority{}, fmt.Errorf("ordered_transport.authority_value:%s", field)
		}
	}
	for field, target := range map[string]string{"21fa": "2024", "21fb": "2020", "21fc": "2021", "21fd": "2023", "21fe": "2022"} {
		f := e.Entities[id(field)]
		constraint := f.Fields[id("111")]
		if names[id(field)] == "" || constraint.Tag != 8 || constraint.Record[id("2001")].Tag != 6 || constraint.Record[id("2001")].Reference != id("10") || e.Entities[id(target)].Schema != id("10") {
			return Authority{}, fmt.Errorf("ordered_transport.authority_reference:%s", field)
		}
	}
	return Authority{ReceiveIdentity: names[id("21e0")], SendIdentity: names[id("21e1")], ReceiveCapability: names[id("21e2")], SendCapability: names[id("21e3")], CodecIdentity: names[id("21e6")], DigestIdentity: names[id("21e7")], MaximumCommands: MaximumCommands, MaximumEvents: MaximumEvents, MaximumEventsPerCommand: MaximumEventsPerCommand, MaximumPayloadBytes: MaximumPayloadBytes, MaximumFrameBytes: MaximumFrameBytes, CorrelationBytes: CorrelationBytes, MaximumStreamBytes: MaximumStreamBytes, FirstCommandSequence: 1, FirstEventSequence: 1, CommandTerminalSentinel: 257, EventTerminalSentinel: 1025, DuplicatePolicy: names[id("21f3")], GapPolicy: names[id("21f4")], CorrelationPolicy: names[id("21f5")], CodecLayout: [4]string{names[id("21f6")], names[id("21f7")], names[id("21f8")], names[id("21f9")]}}, nil
}

func id(s string) wire.ID {
	var out wire.ID
	for len(s) < 32 {
		s = "0" + s
	}
	for i := 0; i < 16; i++ {
		fmt.Sscanf(s[i*2:i*2+2], "%02x", &out[i])
	}
	return out
}
