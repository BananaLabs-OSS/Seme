// Package configurationv2report produces deterministic, source-free evidence
// for authenticated Configuration v2 bound initialization plans.
package configurationv2report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"seme.local/reference/configurationinstance"
	"seme.local/reference/wire"
)

type Report struct {
	ContractRevision string         `json:"contract_revision"`
	ArtifactRevision string         `json:"artifact_revision"`
	Graph            Graph          `json:"graph"`
	RuntimeInputs    []RuntimeInput `json:"runtime_inputs"`
	Initializers     []Initializer  `json:"initializers"`
}

type Graph struct {
	ID              string `json:"id"`
	Base            string `json:"base"`
	ContentRevision string `json:"content_revision_sha256"`
}

type RuntimeInput struct {
	ID         string `json:"id"`
	Identity   string `json:"identity"`
	Type       string `json:"type"`
	Capability string `json:"capability,omitempty"`
}

type Initializer struct {
	ID        string     `json:"id"`
	Base      string     `json:"base"`
	Owner     string     `json:"owner"`
	Callable  string     `json:"callable"`
	Order     uint64     `json:"order"`
	Arguments []Argument `json:"arguments"`
}

type Argument struct {
	ID        string `json:"id"`
	Parameter string `json:"parameter"`
	Index     uint64 `json:"index"`
	Source    Source `json:"source"`
}

type Source struct {
	ID             string          `json:"id"`
	Kind           string          `json:"kind"`
	DerivedType    string          `json:"derived_type"`
	Field          string          `json:"field,omitempty"`
	RuntimeInput   string          `json:"runtime_input,omitempty"`
	Predecessor    string          `json:"predecessor,omitempty"`
	StaticValue    string          `json:"static_value,omitempty"`
	RecordAssembly *RecordAssembly `json:"record_assembly,omitempty"`
}

type RecordAssembly struct {
	ID         string         `json:"id"`
	RecordType string         `json:"record_type"`
	Members    []RecordMember `json:"members"`
}

type RecordMember struct {
	ID     string `json:"id"`
	Field  string `json:"field"`
	Index  uint64 `json:"index"`
	Source Source `json:"source"`
}

func Inspect(in configurationinstance.BoundInput) (Report, error) {
	if err := configurationinstance.ValidateBound(in); err != nil {
		return Report{}, fmt.Errorf("configuration_v2_report.input:%w", err)
	}
	e, err := wire.Decode(in.Artifact)
	if err != nil {
		return Report{}, err
	}
	graph, err := exactlyOne(e, id("401f"))
	if err != nil {
		return Report{}, err
	}
	g := e.Entities[graph]
	report := Report{ContractRevision: id("4005").String(), ArtifactRevision: e.Revision.String(), Graph: Graph{ID: graph.String(), Base: g.Fields[id("41f0")].Reference.String(), ContentRevision: fmt.Sprintf("%x", g.Fields[id("41f3")].Bytes)}}
	for _, r := range g.Fields[id("41f1")].List {
		q := e.Entities[r.Reference]
		item := RuntimeInput{ID: r.Reference.String(), Identity: string(q.Fields[id("4180")].Bytes), Type: q.Fields[id("4181")].Reference.String()}
		if c, ok := q.Fields[id("4182")]; ok {
			item.Capability = c.Reference.String()
		}
		report.RuntimeInputs = append(report.RuntimeInputs, item)
	}
	for _, r := range g.Fields[id("41f2")].List {
		q := e.Entities[r.Reference]
		base := e.Entities[q.Fields[id("41b0")].Reference]
		item := Initializer{ID: r.Reference.String(), Base: q.Fields[id("41b0")].Reference.String(), Owner: base.Fields[id("4150")].Reference.String(), Callable: base.Fields[id("4151")].Reference.String(), Order: base.Fields[id("4153")].Unsigned}
		for _, aRef := range q.Fields[id("41b1")].List {
			a := e.Entities[aRef.Reference]
			source, er := inspectSource(e, a.Fields[id("41e2")].Reference, map[wire.ID]bool{})
			if er != nil {
				return Report{}, er
			}
			item.Arguments = append(item.Arguments, Argument{ID: aRef.Reference.String(), Parameter: a.Fields[id("41e0")].Reference.String(), Index: a.Fields[id("41e1")].Unsigned, Source: source})
		}
		sort.Slice(item.Arguments, func(i, j int) bool { return item.Arguments[i].Index < item.Arguments[j].Index })
		report.Initializers = append(report.Initializers, item)
	}
	sort.Slice(report.RuntimeInputs, func(i, j int) bool { return report.RuntimeInputs[i].Identity < report.RuntimeInputs[j].Identity })
	sort.Slice(report.Initializers, func(i, j int) bool {
		if report.Initializers[i].Order != report.Initializers[j].Order {
			return report.Initializers[i].Order < report.Initializers[j].Order
		}
		return report.Initializers[i].ID < report.Initializers[j].ID
	})
	return report, nil
}

func Marshal(report Report) ([]byte, error) { return json.MarshalIndent(report, "", "  ") }

func inspectSource(e wire.Envelope, x wire.ID, active map[wire.ID]bool) (Source, error) {
	if active[x] {
		return Source{}, fmt.Errorf("configuration_v2_report.source_cycle")
	}
	active[x] = true
	defer delete(active, x)
	q := e.Entities[x]
	kind := e.Entities[q.Fields[id("41a0")].Reference].Fields[id("4190")].Unsigned
	names := []string{"resolved-config-field", "runtime-input", "predecessor-ok-payload", "static-canonical", "record-construction"}
	if kind >= uint64(len(names)) {
		return Source{}, fmt.Errorf("configuration_v2_report.source_kind")
	}
	out := Source{ID: x.String(), Kind: names[kind], DerivedType: q.Fields[id("41a1")].Reference.String()}
	switch kind {
	case 0:
		out.Field = q.Fields[id("41a2")].Reference.String()
	case 1:
		out.RuntimeInput = q.Fields[id("41a3")].Reference.String()
	case 2:
		out.Predecessor = q.Fields[id("41a4")].Reference.String()
	case 3:
		out.StaticValue = q.Fields[id("41a5")].Reference.String()
	case 4:
		aID := q.Fields[id("41a6")].Reference
		a := e.Entities[aID]
		assembly := &RecordAssembly{ID: aID.String(), RecordType: a.Fields[id("41c0")].Reference.String()}
		for _, r := range a.Fields[id("41c1")].List {
			m := e.Entities[r.Reference]
			field := m.Fields[id("41d0")].Reference
			child, err := inspectSource(e, m.Fields[id("41d1")].Reference, active)
			if err != nil {
				return Source{}, err
			}
			assembly.Members = append(assembly.Members, RecordMember{ID: r.Reference.String(), Field: field.String(), Index: e.Entities[field].Fields[id("9312")].Unsigned, Source: child})
		}
		sort.Slice(assembly.Members, func(i, j int) bool {
			if assembly.Members[i].Index != assembly.Members[j].Index {
				return assembly.Members[i].Index < assembly.Members[j].Index
			}
			return bytes.Compare([]byte(assembly.Members[i].Field), []byte(assembly.Members[j].Field)) < 0
		})
		out.RecordAssembly = assembly
	}
	return out, nil
}
func exactlyOne(e wire.Envelope, s wire.ID) (wire.ID, error) {
	var out wire.ID
	for x, q := range e.Entities {
		if q.Schema == s {
			if out != (wire.ID{}) {
				return out, fmt.Errorf("configuration_v2_report.schema_count")
			}
			out = x
		}
	}
	if out == (wire.ID{}) {
		return out, fmt.Errorf("configuration_v2_report.schema_count")
	}
	return out, nil
}
func id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, err := wire.ParseID(s)
	if err != nil {
		panic(err)
	}
	return x
}
