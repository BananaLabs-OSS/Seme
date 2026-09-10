// Package orderedtransportmodule emits the language-neutral Ordered Transport Contract v1.
package orderedtransportmodule

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
)

const ModuleID = "00000000000000000000000000010000"
const RevisionID = "00000000000000000000000000010001"

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
		{0x10100, "OrderedTransportPlan", []field{f(0x11000, "ordered_transport_plan.streams", 5, 0x10101, 2), f(0x11001, "ordered_transport_plan.command_kinds", 5, 0x10103, 2), f(0x11002, "ordered_transport_plan.event_kinds", 5, 0x10104, 2), f(0x11003, "ordered_transport_plan.ports", 5, 0x1010f, 2), f(0x11004, "ordered_transport_plan.content_revision", 4, 0, 0), f(0x11005, "ordered_transport_plan.dispatch_function", 5, 0x9011, 0), f(0x11006, "ordered_transport_plan.replay_function", 5, 0x9011, 0)}},
		{0x10101, "StreamIdentity", []field{f(0x11010, "stream_identity.value", 4, 0, 0)}},
		{0x10102, "CorrelationIdentity", []field{f(0x11020, "correlation_identity.bytes", 4, 0, 0)}},
		{0x10103, "CommandKind", []field{f(0x11030, "command_kind.identity", 4, 0, 0), f(0x11031, "command_kind.owner", 5, 0xb010, 0), f(0x11032, "command_kind.payload_type", 5, 0, 0)}},
		{0x10104, "EventKind", []field{f(0x11040, "event_kind.identity", 4, 0, 0), f(0x11041, "event_kind.owner", 5, 0xb010, 0), f(0x11042, "event_kind.payload_type", 5, 0, 0)}},
		{0x10105, "CanonicalCommandPayload", []field{f(0x11050, "canonical_command_payload.kind", 5, 0x10103, 0), f(0x11051, "canonical_command_payload.bytes", 4, 0, 0), f(0x11052, "canonical_command_payload.sha256", 4, 0, 0)}},
		{0x10106, "CommandEnvelope", []field{f(0x11060, "command_envelope.stream", 5, 0x10101, 0), f(0x11061, "command_envelope.sequence", 3, 0, 0), f(0x11062, "command_envelope.correlation", 5, 0x10102, 0), f(0x11063, "command_envelope.kind", 5, 0x10103, 0), f(0x11064, "command_envelope.payload", 5, 0x10105, 0)}},
		{0x10107, "EventEnvelope", []field{f(0x11070, "event_envelope.sequence", 3, 0, 0), f(0x11071, "event_envelope.command_sequence", 3, 0, 0), f(0x11072, "event_envelope.ordinal", 3, 0, 0), f(0x11073, "event_envelope.correlation", 5, 0x10102, 0), f(0x11074, "event_envelope.kind", 5, 0x10104, 0), f(0x11075, "event_envelope.payload", 5, 0x10115, 0), f(0x11076, "event_envelope.stream", 5, 0x10101, 0)}},
		{0x10108, "CommandReceipt", []field{f(0x11080, "command_receipt.sequence", 3, 0, 0), f(0x11081, "command_receipt.correlation", 5, 0x10102, 0), f(0x11082, "command_receipt.disposition", 3, 0, 0), f(0x11083, "command_receipt.outcome", 4, 0, 0), f(0x11084, "command_receipt.outcome_digest", 4, 0, 0)}},
		{0x10109, "RetainedCommand", []field{f(0x11090, "retained_command.envelope", 5, 0x10106, 0), f(0x11091, "retained_command.receipt", 5, 0x10108, 0), f(0x11092, "retained_command.events", 5, 0x10107, 2)}},
		{0x1010a, "StreamState", []field{f(0x110a0, "stream_state.stream", 5, 0x10101, 0), f(0x110a1, "stream_state.next_command_sequence", 3, 0, 0), f(0x110a2, "stream_state.next_event_sequence", 3, 0, 0), f(0x110a3, "stream_state.ledger", 5, 0x10109, 2)}},
		{0x1010b, "DispatchOutcome", []field{f(0x110b0, "dispatch_outcome.state", 5, 0x1010a, 0), f(0x110b1, "dispatch_outcome.receipt", 5, 0x10108, 0), f(0x110b2, "dispatch_outcome.events", 5, 0x10107, 2), f(0x110b3, "dispatch_outcome.duplicate", 1, 0, 0)}},
		{0x1010c, "Replay", []field{f(0x110c0, "replay.initial_state", 5, 0x1010a, 0), f(0x110c1, "replay.accepted_commands", 5, 0x10106, 2), f(0x110c2, "replay.transcript_digest", 4, 0, 0)}},
		{0x1010d, "TransportError", []field{f(0x110d0, "transport_error.identity", 4, 0, 0)}},
		{0x1010e, "TransportOperationAuthority", []field{
			f(0x110e0, "seme.transport.receive.v1", 4, 0, 0), f(0x110e1, "seme.transport.send.v1", 4, 0, 0), f(0x110e2, "seme.transport.receive.v1.capability", 4, 0, 0), f(0x110e3, "seme.transport.send.v1.capability", 4, 0, 0),
			f(0x110e4, "seme.transport.sequence.0.receive", 3, 0, 0), f(0x110e5, "seme.transport.sequence.1.send", 3, 0, 0), f(0x110e6, "seme.ordered-transport.binary.v1", 4, 0, 0), f(0x110e7, "sha256", 4, 0, 0),
			f(0x110e8, "ordered_transport.maximum_commands.256", 3, 0, 0), f(0x110e9, "ordered_transport.maximum_events.1024", 3, 0, 0), f(0x110ea, "ordered_transport.maximum_events_per_command.4", 3, 0, 0), f(0x110eb, "ordered_transport.maximum_payload_bytes.3072", 3, 0, 0), f(0x110ec, "ordered_transport.maximum_frame_bytes.4096", 3, 0, 0), f(0x110ed, "ordered_transport.correlation_bytes.16", 3, 0, 0), f(0x110ee, "ordered_transport.maximum_stream_bytes.128", 3, 0, 0),
			f(0x110ef, "ordered_transport.first_command_sequence.1", 3, 0, 0), f(0x110f0, "ordered_transport.first_event_sequence.1", 3, 0, 0), f(0x110f1, "ordered_transport.command_terminal_sentinel.257", 3, 0, 0), f(0x110f2, "ordered_transport.event_terminal_sentinel.1025", 3, 0, 0), f(0x110f3, "ordered_transport.duplicate_policy.exact_cached_response", 4, 0, 0), f(0x110f4, "ordered_transport.gap_policy.reject_without_buffering", 4, 0, 0), f(0x110f5, "ordered_transport.correlation_policy.unique_per_command", 4, 0, 0),
			f(0x110f6, "ordered_transport.codec.u32le_body_length_excluding_prefix", 4, 0, 0), f(0x110f7, "ordered_transport.codec.magic.SEMEOT01", 4, 0, 0), f(0x110f8, "ordered_transport.codec.u8_frame_kind_then_pure_value_v1", 4, 0, 0), f(0x110f9, "ordered_transport.codec.exact_length_no_trailing_bytes", 4, 0, 0),
			f(0x110fa, "transport_operation_authority.receive_request_schema", 5, 0x10, 0), f(0x110fb, "transport_operation_authority.receive_outcome_schema", 5, 0x10, 0), f(0x110fc, "transport_operation_authority.send_request_schema", 5, 0x10, 0), f(0x110fd, "transport_operation_authority.send_outcome_schema", 5, 0x10, 0), f(0x110fe, "transport_operation_authority.port_error_schema", 5, 0x10, 0),
		}},
		{0x1010f, "TransportPort", []field{f(0x11100, "transport_port.identity", 4, 0, 0), f(0x11101, "transport_port.owner", 5, 0xb010, 0), f(0x11102, "transport_port.receive_capability", 5, 0x16, 0), f(0x11103, "transport_port.send_capability", 5, 0x16, 0), f(0x11104, "transport_port.receive_effect", 5, 0x15, 0), f(0x11105, "transport_port.send_effect", 5, 0x15, 0), f(0x11106, "transport_port.authority", 5, 0x1010e, 0)}},
		{0x10110, "ReceiveOutcome", []field{f(0x11110, "receive_outcome.variant", 3, 0, 0), f(0x11111, "receive_outcome.frame", 4, 0, 1), f(0x11112, "receive_outcome.error", 5, 0x10112, 1)}},
		{0x10111, "SendRequest", []field{f(0x11120, "send_request.port", 5, 0x1010f, 0), f(0x11121, "send_request.frame", 4, 0, 0)}},
		{0x10112, "TransportPortError", []field{f(0x11130, "transport_port_error.identity", 4, 0, 0), f(0x11131, "transport_port_error.operation_identity", 4, 0, 0)}},
		{0x10113, "SendOutcome", []field{f(0x11140, "send_outcome.variant", 3, 0, 0), f(0x11141, "send_outcome.accepted_sha256", 4, 0, 1), f(0x11142, "send_outcome.error", 5, 0x10112, 1)}},
		{0x10114, "ReceiveRequest", []field{f(0x11150, "receive_request.port", 5, 0x1010f, 0)}},
		{0x10115, "CanonicalEventPayload", []field{f(0x11160, "canonical_event_payload.kind", 5, 0x10104, 0), f(0x11161, "canonical_event_payload.bytes", 4, 0, 0), f(0x11162, "canonical_event_payload.sha256", 4, 0, 0)}},
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
	fmt.Fprintf(out, "en %s %s 1 3\nfi %s by %s\nfi %s li 3\nrf %s\nrf %s\nrf %s\nfi %s li %d\n", ModuleID, id(0x12), id(0x120), text("ordered-transport-contract-v1"), id(0x121), id(0x10002), id(0x10003), id(0x10004), id(0x122), exports)
	for _, s := range d {
		fmt.Fprintf(out, "rf %s\n", id(s.id))
	}
	for _, x := range fields {
		fmt.Fprintf(out, "rf %s\n", id(x.id))
	}
	for _, p := range []struct{ id, module, rev uint64 }{{0x10002, 0xb000, 0xb004}, {0x10003, 0x9000, 0x9024}, {0x10004, 0x3000, 0x3001}} {
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
