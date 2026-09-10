// Package durablestatemodule emits the language-neutral Durable State Contract v1.
package durablestatemodule

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
)

const ModuleID = "00000000000000000000000000008000"
const RevisionID = "00000000000000000000000000008001"

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
		{0x8010, "DurableStatePlan", []field{f(0x8100, "durable_state_plan.families", 5, 0x8011, 2), f(0x8101, "durable_state_plan.storage_ports", 5, 0x8015, 2), f(0x8102, "durable_state_plan.content_revision", 4, 0, 0)}},
		{0x8011, "StateFamily", []field{f(0x8110, "state_family.identity", 4, 0, 0), f(0x8111, "state_family.owner", 5, 0xb010, 0), f(0x8112, "state_family.current_version", 5, 0x8012, 0), f(0x8113, "state_family.versions", 5, 0x8012, 2), f(0x8114, "state_family.validators", 5, 0x8013, 2), f(0x8115, "state_family.migrations", 5, 0x8014, 2)}},
		{0x8012, "StateVersion", []field{f(0x8120, "state_version.family", 5, 0x8011, 0), f(0x8121, "state_version.number", 2, 0, 0), f(0x8122, "state_version.state_type", 5, 0, 0)}},
		{0x8013, "StateValidator", []field{f(0x8130, "state_validator.version", 5, 0x8012, 0), f(0x8131, "state_validator.function", 5, 0x9011, 0)}},
		{0x8014, "StateMigration", []field{f(0x8140, "state_migration.from_version", 5, 0x8012, 0), f(0x8141, "state_migration.to_version", 5, 0x8012, 0), f(0x8142, "state_migration.function", 5, 0x9011, 0)}},
		{0x8015, "StoragePort", []field{f(0x8150, "storage_port.identity", 4, 0, 0), f(0x8151, "storage_port.owner", 5, 0xb010, 0), f(0x8152, "storage_port.family", 5, 0x8011, 0), f(0x8153, "storage_port.load_capability", 5, 0x16, 0), f(0x8154, "storage_port.compare_exchange_capability", 5, 0x16, 0), f(0x8155, "storage_port.load_effect", 5, 0x15, 0), f(0x8156, "storage_port.compare_exchange_effect", 5, 0x15, 0), f(0x8157, "storage_port.key_type", 5, 0, 0), f(0x8158, "storage_port.maximum_payload_bytes", 2, 0, 0), f(0x8159, "storage_port.codec_identity", 4, 0, 0)}},
	}
}

func Emit(out io.Writer) error {
	d := declarations()
	fields := []field{}
	for _, s := range d {
		fields = append(fields, s.fields...)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].id < fields[j].id })
	id := func(x uint64) string { return fmt.Sprintf("%032x", x) }
	text := func(s string) string { return hex.EncodeToString([]byte(s)) }
	exports := len(d) + len(fields)
	fmt.Fprintf(out, "# Generated construction projection for Durable State Contract v1.\nve 1\nmo %s\nrv %s\npc 0\nec %d\n\n", ModuleID, RevisionID, 4+exports)
	fmt.Fprintf(out, "en %s %s 1 3\nfi %s by %s\nfi %s li 3\nrf %s\nrf %s\nrf %s\nfi %s li %d\n", ModuleID, id(0x12), id(0x120), text("durable-state-contract-v1"), id(0x121), id(0x8002), id(0x8003), id(0x8004), id(0x122), exports)
	for _, s := range d {
		fmt.Fprintf(out, "rf %s\n", id(s.id))
	}
	for _, x := range fields {
		fmt.Fprintf(out, "rf %s\n", id(x.id))
	}
	for _, p := range []struct{ id, module, rev uint64 }{{0x8002, 0xb000, 0xb004}, {0x8003, 0x9000, 0x9024}, {0x8004, 0x3000, 0x3001}} {
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
