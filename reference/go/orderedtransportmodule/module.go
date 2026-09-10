// Package orderedtransportmodule emits the language-neutral Ordered Transport Contract v1.
package orderedtransportmodule

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
)

const ModuleID = "00000000000000000000000000002000"
const RevisionID = "00000000000000000000000000002001"

type field struct {
	id                 uint64
	name               string
	kind, schema, card uint64
}
type schema struct {
	id     uint64
	name   string
	fields []field
}

func f(id uint64, name string, kind, target, card uint64) field {
	return field{id, name, kind, target, card}
}

func declarations() []schema {
	return []schema{
		{0x2010, "OrderedTransportPlan", []field{f(0x2100, "ordered_transport_plan.streams", 5, 0x2011, 2), f(0x2101, "ordered_transport_plan.command_kinds", 5, 0x2013, 2), f(0x2102, "ordered_transport_plan.event_kinds", 5, 0x2014, 2), f(0x2103, "ordered_transport_plan.ports", 5, 0x201f, 2), f(0x2104, "ordered_transport_plan.content_revision", 4, 0, 0), f(0x2105, "ordered_transport_plan.dispatch_function", 5, 0x9011, 0), f(0x2106, "ordered_transport_plan.replay_function", 5, 0x9011, 0)}},
		{0x2011, "StreamIdentity", []field{f(0x2110, "stream_identity.value", 4, 0, 0)}},
		{0x2012, "CorrelationIdentity", []field{f(0x2120, "correlation_identity.bytes", 4, 0, 0)}},
		{0x2013, "CommandKind", []field{f(0x2130, "command_kind.identity", 4, 0, 0), f(0x2131, "command_kind.owner", 5, 0xb010, 0), f(0x2132, "command_kind.payload_type", 5, 0, 0)}},
		{0x2014, "EventKind", []field{f(0x2140, "event_kind.identity", 4, 0, 0), f(0x2141, "event_kind.owner", 5, 0xb010, 0), f(0x2142, "event_kind.payload_type", 5, 0, 0)}},
		{0x2015, "CanonicalCommandPayload", []field{f(0x2150, "canonical_command_payload.kind", 5, 0x2013, 0), f(0x2151, "canonical_command_payload.bytes", 4, 0, 0), f(0x2152, "canonical_command_payload.sha256", 4, 0, 0)}},
		{0x2016, "CommandEnvelope", []field{f(0x2160, "command_envelope.stream", 5, 0x2011, 0), f(0x2161, "command_envelope.sequence", 3, 0, 0), f(0x2162, "command_envelope.correlation", 5, 0x2012, 0), f(0x2163, "command_envelope.kind", 5, 0x2013, 0), f(0x2164, "command_envelope.payload", 5, 0x2015, 0)}},
		{0x2017, "EventEnvelope", []field{f(0x2170, "event_envelope.sequence", 3, 0, 0), f(0x2171, "event_envelope.command_sequence", 3, 0, 0), f(0x2172, "event_envelope.ordinal", 3, 0, 0), f(0x2173, "event_envelope.correlation", 5, 0x2012, 0), f(0x2174, "event_envelope.kind", 5, 0x2014, 0), f(0x2175, "event_envelope.payload", 5, 0x2025, 0), f(0x2176, "event_envelope.stream", 5, 0x2011, 0)}},
		{0x2018, "CommandReceipt", []field{f(0x2180, "command_receipt.sequence", 3, 0, 0), f(0x2181, "command_receipt.correlation", 5, 0x2012, 0), f(0x2182, "command_receipt.disposition", 3, 0, 0), f(0x2183, "command_receipt.outcome", 4, 0, 0), f(0x2184, "command_receipt.outcome_digest", 4, 0, 0)}},
		{0x2019, "RetainedCommand", []field{f(0x2190, "retained_command.envelope", 5, 0x2016, 0), f(0x2191, "retained_command.receipt", 5, 0x2018, 0), f(0x2192, "retained_command.events", 5, 0x2017, 2)}},
		{0x201a, "StreamState", []field{f(0x21a0, "stream_state.stream", 5, 0x2011, 0), f(0x21a1, "stream_state.next_command_sequence", 3, 0, 0), f(0x21a2, "stream_state.next_event_sequence", 3, 0, 0), f(0x21a3, "stream_state.ledger", 5, 0x2019, 2)}},
		{0x201b, "DispatchOutcome", []field{f(0x21b0, "dispatch_outcome.state", 5, 0x201a, 0), f(0x21b1, "dispatch_outcome.receipt", 5, 0x2018, 0), f(0x21b2, "dispatch_outcome.events", 5, 0x2017, 2), f(0x21b3, "dispatch_outcome.duplicate", 1, 0, 0)}},
		{0x201c, "Replay", []field{f(0x21c0, "replay.initial_state", 5, 0x201a, 0), f(0x21c1, "replay.accepted_commands", 5, 0x2016, 2), f(0x21c2, "replay.transcript_digest", 4, 0, 0)}},
		{0x201d, "TransportError", []field{f(0x21d0, "transport_error.identity", 4, 0, 0)}},
		{0x201e, "TransportOperationAuthority", []field{
			f(0x21e0, "seme.transport.receive.v1", 4, 0, 0), f(0x21e1, "seme.transport.send.v1", 4, 0, 0), f(0x21e2, "seme.transport.receive.v1.capability", 4, 0, 0), f(0x21e3, "seme.transport.send.v1.capability", 4, 0, 0),
			f(0x21e4, "seme.transport.sequence.0.receive", 3, 0, 0), f(0x21e5, "seme.transport.sequence.1.send", 3, 0, 0), f(0x21e6, "seme.ordered-transport.binary.v1", 4, 0, 0), f(0x21e7, "sha256", 4, 0, 0),
			f(0x21e8, "ordered_transport.maximum_commands.256", 3, 0, 0), f(0x21e9, "ordered_transport.maximum_events.1024", 3, 0, 0), f(0x21ea, "ordered_transport.maximum_events_per_command.4", 3, 0, 0), f(0x21eb, "ordered_transport.maximum_payload_bytes.3072", 3, 0, 0), f(0x21ec, "ordered_transport.maximum_frame_bytes.4096", 3, 0, 0), f(0x21ed, "ordered_transport.correlation_bytes.16", 3, 0, 0), f(0x21ee, "ordered_transport.maximum_stream_bytes.128", 3, 0, 0),
			f(0x21ef, "ordered_transport.first_command_sequence.1", 3, 0, 0), f(0x21f0, "ordered_transport.first_event_sequence.1", 3, 0, 0), f(0x21f1, "ordered_transport.command_terminal_sentinel.257", 3, 0, 0), f(0x21f2, "ordered_transport.event_terminal_sentinel.1025", 3, 0, 0), f(0x21f3, "ordered_transport.duplicate_policy.exact_cached_response", 4, 0, 0), f(0x21f4, "ordered_transport.gap_policy.reject_without_buffering", 4, 0, 0), f(0x21f5, "ordered_transport.correlation_policy.unique_per_command", 4, 0, 0),
			f(0x21f6, "ordered_transport.codec.u32le_body_length_excluding_prefix", 4, 0, 0), f(0x21f7, "ordered_transport.codec.magic.SEMEOT01", 4, 0, 0), f(0x21f8, "ordered_transport.codec.u8_frame_kind_then_pure_value_v1", 4, 0, 0), f(0x21f9, "ordered_transport.codec.exact_length_no_trailing_bytes", 4, 0, 0),
			f(0x21fa, "transport_operation_authority.receive_request_schema", 5, 0x10, 0), f(0x21fb, "transport_operation_authority.receive_outcome_schema", 5, 0x10, 0), f(0x21fc, "transport_operation_authority.send_request_schema", 5, 0x10, 0), f(0x21fd, "transport_operation_authority.send_outcome_schema", 5, 0x10, 0), f(0x21fe, "transport_operation_authority.port_error_schema", 5, 0x10, 0),
		}},
		{0x201f, "TransportPort", []field{f(0x2200, "transport_port.identity", 4, 0, 0), f(0x2201, "transport_port.owner", 5, 0xb010, 0), f(0x2202, "transport_port.receive_capability", 5, 0x16, 0), f(0x2203, "transport_port.send_capability", 5, 0x16, 0), f(0x2204, "transport_port.receive_effect", 5, 0x15, 0), f(0x2205, "transport_port.send_effect", 5, 0x15, 0), f(0x2206, "transport_port.authority", 5, 0x201e, 0)}},
		{0x2020, "ReceiveOutcome", []field{f(0x2210, "receive_outcome.variant", 3, 0, 0), f(0x2211, "receive_outcome.frame", 4, 0, 1), f(0x2212, "receive_outcome.error", 5, 0x2022, 1)}},
		{0x2021, "SendRequest", []field{f(0x2220, "send_request.port", 5, 0x201f, 0), f(0x2221, "send_request.frame", 4, 0, 0)}},
		{0x2022, "TransportPortError", []field{f(0x2230, "transport_port_error.identity", 4, 0, 0), f(0x2231, "transport_port_error.operation_identity", 4, 0, 0)}},
		{0x2023, "SendOutcome", []field{f(0x2240, "send_outcome.variant", 3, 0, 0), f(0x2241, "send_outcome.accepted_sha256", 4, 0, 1), f(0x2242, "send_outcome.error", 5, 0x2022, 1)}},
		{0x2024, "ReceiveRequest", []field{f(0x2250, "receive_request.port", 5, 0x201f, 0)}},
		{0x2025, "CanonicalEventPayload", []field{f(0x2260, "canonical_event_payload.kind", 5, 0x2014, 0), f(0x2261, "canonical_event_payload.bytes", 4, 0, 0), f(0x2262, "canonical_event_payload.sha256", 4, 0, 0)}},
	}
}

func Emit(out io.Writer) error {
	d := declarations()
	var fields []field
	for _, s := range d {
		fields = append(fields, s.fields...)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].id < fields[j].id })
	id := func(x uint64) string { return fmt.Sprintf("%032x", x) }
	text := func(s string) string { return hex.EncodeToString([]byte(s)) }
	exports := len(d) + len(fields)
	fmt.Fprintf(out, "# Generated construction projection for Ordered Transport Contract v1.\nve 1\nmo %s\nrv %s\npc 0\nec %d\n\n", ModuleID, RevisionID, 4+exports)
	fmt.Fprintf(out, "en %s %s 1 3\nfi %s by %s\nfi %s li 3\nrf %s\nrf %s\nrf %s\nfi %s li %d\n", ModuleID, id(0x12), id(0x120), text("ordered-transport-contract-v1"), id(0x121), id(0x2002), id(0x2003), id(0x2004), id(0x122), exports)
	for _, s := range d {
		fmt.Fprintf(out, "rf %s\n", id(s.id))
	}
	for _, x := range fields {
		fmt.Fprintf(out, "rf %s\n", id(x.id))
	}
	for _, p := range []struct{ id, module, rev uint64 }{{0x2002, 0xb000, 0xb004}, {0x2003, 0x9000, 0x9024}, {0x2004, 0x3000, 0x3001}} {
		fmt.Fprintf(out, "\nen %s %s 1 2\nfi %s rf %s\nfi %s by %s\n", id(p.id), id(0x13), id(0x130), id(p.module), id(0x131), id(p.rev))
	}
	for _, s := range d {
		fmt.Fprintf(out, "\nen %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(s.id), id(0x10), id(0x100), text(s.name), id(0x101), len(s.fields))
		for _, x := range s.fields {
			fmt.Fprintf(out, "rf %s\n", id(x.id))
		}
	}
	for _, x := range fields {
		count, constraint := 1, ""
		if x.schema != 0 {
			count = 2
			constraint = fmt.Sprintf("fi %s rf %s\n", id(0x2001), id(x.schema))
		}
		fmt.Fprintf(out, "\nen %s %s 1 4\nfi %s by %s\nfi %s rc %d\nfi %s uu %d\n%sfi %s uu %d\nfi %s uu 1\n", id(x.id), id(0x11), id(0x110), text(x.name), id(0x111), count, id(0x2000), x.kind, constraint, id(0x112), x.card, id(0x113))
	}
	return nil
}
