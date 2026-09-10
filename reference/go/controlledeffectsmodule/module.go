// Package controlledeffectsmodule emits the language-neutral Controlled Effects Contract v1.
package controlledeffectsmodule

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
)

const ModuleID = "00000000000000000000000000013000"
const RevisionID = "00000000000000000000000000013001"

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
		{0x13100, "ControlledEffectsPlan", []field{f(0x13200, "controlled_effects_plan.clock", 5, 0x13101, 0), f(0x13201, "controlled_effects_plan.random", 5, 0x13103, 0), f(0x13202, "controlled_effects_plan.external_effect", 5, 0x13105, 0), f(0x13203, "controlled_effects_plan.replay", 5, 0x13107, 0), f(0x13204, "controlled_effects_plan.bounds", 5, 0x13108, 0), f(0x13205, "controlled_effects_plan.apply_function", 5, 0x9011, 0), f(0x13206, "controlled_effects_plan.replay_function", 5, 0x9011, 0), f(0x13207, "controlled_effects_plan.content_revision", 4, 0, 0)}},
		{0x13101, "ClockAuthority", []field{f(0x13210, "clock_authority.identity", 4, 0, 0), f(0x13211, "clock_authority.sample_type", 5, 0x13102, 0), f(0x13212, "clock_authority.monotonic_policy", 4, 0, 0), f(0x13213, "clock_authority.injection_policy", 4, 0, 0)}},
		{0x13102, "ClockSample", []field{f(0x13220, "clock_sample.unix_milliseconds", 3, 0, 0), f(0x13221, "clock_sample.sequence", 2, 0, 0)}},
		{0x13103, "SeededRandomAuthority", []field{f(0x13230, "seeded_random_authority.identity", 4, 0, 0), f(0x13231, "seeded_random_authority.algorithm", 4, 0, 0), f(0x13232, "seeded_random_authority.overflow_policy", 4, 0, 0), f(0x13233, "seeded_random_authority.state_type", 5, 0x13104, 0), f(0x13234, "seeded_random_authority.next_function", 5, 0x9011, 0)}},
		{0x13104, "SeededRandomState", []field{f(0x13240, "seeded_random_state.value", 3, 0, 0), f(0x13241, "seeded_random_state.draws", 2, 0, 0)}},
		{0x13105, "ExternalBooleanEffect", []field{f(0x13250, "external_boolean_effect.identity", 4, 0, 0), f(0x13251, "external_boolean_effect.capability", 5, 0x16, 0), f(0x13252, "external_boolean_effect.effect", 5, 0x15, 0), f(0x13253, "external_boolean_effect.payload_type", 5, 0x9020, 0), f(0x13254, "external_boolean_effect.delivery_policy", 4, 0, 0)}},
		{0x13106, "ReplayStep", []field{f(0x13260, "replay_step.command_sequence", 2, 0, 0), f(0x13261, "replay_step.clock", 5, 0x13102, 0), f(0x13262, "replay_step.random_before_value", 3, 0, 0), f(0x13263, "replay_step.random_after_value", 3, 0, 0), f(0x13264, "replay_step.random_draw_value", 3, 0, 0), f(0x13265, "replay_step.random_draw_ordinal", 2, 0, 0), f(0x13266, "replay_step.effect_value", 1, 0, 0), f(0x13267, "replay_step.canonical_command", 4, 0, 0), f(0x13268, "replay_step.response_sha256", 4, 0, 0), f(0x13269, "replay_step.events_sha256", 4, 0, 0), f(0x1326a, "replay_step.state_before_sha256", 4, 0, 0), f(0x1326b, "replay_step.state_after_sha256", 4, 0, 0)}},
		{0x13107, "ControlledReplay", []field{f(0x13270, "controlled_replay.initial_seed", 3, 0, 0), f(0x13271, "controlled_replay.steps", 5, 0x13106, 2), f(0x13272, "controlled_replay.transcript_digest", 4, 0, 0), f(0x13273, "controlled_replay.duplicate_policy", 4, 0, 0), f(0x13274, "controlled_replay.rejection_policy", 4, 0, 0), f(0x13275, "controlled_replay.initial_state", 4, 0, 0), f(0x13276, "controlled_replay.initial_state_sha256", 4, 0, 0)}},
		{0x13108, "ControlledEffectsBounds", []field{f(0x13280, "controlled_effects_bounds.maximum_steps", 2, 0, 0), f(0x13281, "controlled_effects_bounds.first_clock_sequence", 2, 0, 0), f(0x13282, "controlled_effects_bounds.clock_terminal_sentinel", 2, 0, 0), f(0x13283, "controlled_effects_bounds.maximum_unix_milliseconds", 3, 0, 0), f(0x13284, "controlled_effects_bounds.minimum_seed", 3, 0, 0), f(0x13285, "controlled_effects_bounds.maximum_seed", 3, 0, 0), f(0x13286, "controlled_effects_bounds.maximum_draws", 2, 0, 0), f(0x13287, "controlled_effects_bounds.maximum_effects", 2, 0, 0), f(0x13288, "controlled_effects_bounds.maximum_command_bytes", 2, 0, 0), f(0x13289, "controlled_effects_bounds.maximum_initial_state_bytes", 2, 0, 0), f(0x1328a, "controlled_effects_bounds.maximum_transcript_bytes", 2, 0, 0)}},
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
	fmt.Fprintf(out, "# Generated construction projection for Controlled Effects Contract v1.\nve 1\nmo %s\nrv %s\npc 0\nec %d\n\n", ModuleID, RevisionID, 4+exports)
	fmt.Fprintf(out, "en %s %s 1 3\nfi %s by %s\nfi %s li 3\nrf %s\nrf %s\nrf %s\nfi %s li %d\n", ModuleID, id(0x12), id(0x120), text("controlled-effects-contract-v1"), id(0x121), id(0x13002), id(0x13003), id(0x13004), id(0x122), exports)
	for _, s := range d {
		fmt.Fprintf(out, "rf %s\n", id(s.id))
	}
	for _, x := range fields {
		fmt.Fprintf(out, "rf %s\n", id(x.id))
	}
	for _, p := range []struct{ id, module, rev uint64 }{{0x13002, 0xb000, 0xb004}, {0x13003, 0x9000, 0x9024}, {0x13004, 0x3000, 0x3001}} {
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
