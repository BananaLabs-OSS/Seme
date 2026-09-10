// Package goprojectdependency binds neutral dependency closure entries to the
// exact Go project entities from which they were resolved. The interpretation
// is deliberately Go-specific; the Project and Dependency contracts remain
// language-neutral.
package goprojectdependency

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"seme.local/reference/dependencyinstance"
	"seme.local/reference/dependencyresolution"
	"seme.local/reference/projectdependencyinstance"
	"seme.local/reference/wire"
)

const (
	modPathKey   = "go.mod.path"
	modSourceKey = "go.mod.source"
	modDigestKey = "go.mod.sha256"
	sourcePrefix = "go.source."
)

// BindMetadata adds independently checkable Project-v3 applicability metadata
// to an already resolved bounded Go closure.
func BindMetadata(c dependencyresolution.Closure, projectV3 []byte) (dependencyresolution.Closure, error) {
	if err := dependencyresolution.Validate(c); err != nil {
		return dependencyresolution.Closure{}, err
	}
	p, err := inspect(projectV3)
	if err != nil {
		return dependencyresolution.Closure{}, err
	}
	out := clone(c)
	local, external, err := bounded(&out)
	if err != nil {
		return dependencyresolution.Closure{}, err
	}
	from, ok := metadataValue(local.requirement.Metadata, "go.import.from")
	if !ok || !p.localEdge(from, local.entry.Identity) {
		return dependencyresolution.Closure{}, fmt.Errorf("go_project_dependency.local_edge")
	}
	sources := p.packageSources[local.entry.Identity]
	if len(sources) == 0 {
		return dependencyresolution.Closure{}, fmt.Errorf("go_project_dependency.local_sources")
	}
	for _, u := range sources {
		m := dependencyresolution.Metadata{Key: sourcePrefix + u.id.String(), Value: u.digest}
		local.requirement.Metadata = append(local.requirement.Metadata, m)
		local.entry.Metadata = append(local.entry.Metadata, m)
	}
	for _, target := range []*[]dependencyresolution.Metadata{&external.requirement.Metadata, &external.entry.Metadata} {
		*target = append(*target, dependencyresolution.Metadata{Key: modDigestKey, Value: p.mod.digest}, dependencyresolution.Metadata{Key: modPathKey, Value: "go.mod"}, dependencyresolution.Metadata{Key: modSourceKey, Value: p.mod.id.String()})
	}
	dependencyresolution.Normalize(&out)
	if err = dependencyresolution.Validate(out); err != nil {
		return dependencyresolution.Closure{}, err
	}
	return out, nil
}

// Validate first validates the complete neutral Project-v4 boundary and then
// requires its Dependency-v1 closure to carry exact Go applicability bindings.
func Validate(in projectdependencyinstance.Inputs) error {
	if err := projectdependencyinstance.Validate(in); err != nil {
		return err
	}
	c, err := dependencyinstance.Validate(in.Contracts.Dependency(), in.Dependency)
	if err != nil {
		return err
	}
	want, err := BindMetadata(stripBindings(c), in.ProjectV3.Composed)
	if err != nil {
		return err
	}
	if !closureEqual(want, c) {
		return fmt.Errorf("go_project_dependency.binding")
	}
	return nil
}

type pair struct {
	requirement *dependencyresolution.Requirement
	entry       *dependencyresolution.Entry
}

func bounded(c *dependencyresolution.Closure) (pair, pair, error) {
	by := map[string]*dependencyresolution.Entry{}
	for i := range c.Entries {
		by[c.Entries[i].Identity] = &c.Entries[i]
	}
	var local, external pair
	for i := range c.Requirements {
		r := &c.Requirements[i]
		e := by[r.Identity]
		if e == nil {
			return pair{}, pair{}, fmt.Errorf("go_project_dependency.missing")
		}
		if r.Kind == dependencyresolution.Local {
			if local.entry != nil {
				return pair{}, pair{}, fmt.Errorf("go_project_dependency.local_count")
			}
			local = pair{r, e}
		} else {
			if external.entry != nil {
				return pair{}, pair{}, fmt.Errorf("go_project_dependency.external_count")
			}
			external = pair{r, e}
		}
	}
	if local.entry == nil || external.entry == nil || len(c.Entries) != 2 {
		return pair{}, pair{}, fmt.Errorf("go_project_dependency.profile")
	}
	return local, external, nil
}

type unit struct {
	id     wire.ID
	digest string
}
type project struct {
	mod            unit
	packageSources map[string][]unit
	edges          map[string]map[string]bool
}

func (p project) localEdge(from, to string) bool { return p.edges[from][to] }
func inspect(raw []byte) (project, error) {
	e, err := wire.Decode(raw)
	if err != nil {
		return project{}, err
	}
	p := project{packageSources: map[string][]unit{}, edges: map[string]map[string]bool{}}
	units := map[wire.ID]unit{}
	modCount := 0
	for x, q := range e.Entities {
		if q.Schema == id("e015") {
			path := q.Fields[id("e150")]
			digest := q.Fields[id("e151")]
			if path.Tag != 5 || digest.Tag != 5 || len(digest.Bytes) != 32 {
				return project{}, fmt.Errorf("go_project_dependency.source")
			}
			u := unit{x, hex.EncodeToString(digest.Bytes)}
			units[x] = u
			if string(path.Bytes) == "go.mod" {
				class := e.Entities[q.Fields[id("e153")].Reference]
				pres := e.Entities[q.Fields[id("e154")].Reference]
				if class.Fields[id("e130")].Unsigned != 4 || pres.Fields[id("e140")].Unsigned != 0 {
					return project{}, fmt.Errorf("go_project_dependency.mod_policy")
				}
				p.mod = u
				modCount++
			}
		}
	}
	if modCount != 1 {
		return project{}, fmt.Errorf("go_project_dependency.mod_count")
	}
	names := map[wire.ID]string{}
	for x, q := range e.Entities {
		if q.Schema == id("b010") {
			names[x] = string(q.Fields[id("b100")].Bytes)
		}
	}
	graphs := withSchema(e, id("b020"))
	if len(graphs) != 1 {
		return project{}, fmt.Errorf("go_project_dependency.graph")
	}
	for _, dv := range e.Entities[graphs[0]].Fields[id("b200")].List {
		d, ok := e.Entities[dv.Reference]
		if !ok || d.Schema != id("b021") {
			return project{}, fmt.Errorf("go_project_dependency.detail")
		}
		owner := names[d.Fields[id("b210")].Reference]
		if owner == "" {
			return project{}, fmt.Errorf("go_project_dependency.owner")
		}
		seen := map[wire.ID]bool{}
		for _, ov := range d.Fields[id("b213")].List {
			o := e.Entities[ov.Reference]
			sid := o.Fields[id("b260")].Reference
			u, yes := units[sid]
			if !yes {
				return project{}, fmt.Errorf("go_project_dependency.origin")
			}
			if !seen[sid] {
				p.packageSources[owner] = append(p.packageSources[owner], u)
				seen[sid] = true
			}
		}
		sort.Slice(p.packageSources[owner], func(i, j int) bool {
			return p.packageSources[owner][i].id.String() < p.packageSources[owner][j].id.String()
		})
		for _, iv := range d.Fields[id("b212")].List {
			im := e.Entities[iv.Reference]
			class := e.Entities[im.Fields[id("b242")].Reference]
			if class.Fields[id("b250")].Unsigned == 0 {
				to := names[im.Fields[id("b243")].Reference]
				if to == "" {
					return project{}, fmt.Errorf("go_project_dependency.import_target")
				}
				if p.edges[owner] == nil {
					p.edges[owner] = map[string]bool{}
				}
				p.edges[owner][to] = true
			}
		}
	}
	return p, nil
}

func stripBindings(c dependencyresolution.Closure) dependencyresolution.Closure {
	out := clone(c)
	for i := range out.Requirements {
		out.Requirements[i].Metadata = without(out.Requirements[i].Metadata)
	}
	for i := range out.Entries {
		out.Entries[i].Metadata = without(out.Entries[i].Metadata)
	}
	return out
}
func without(ms []dependencyresolution.Metadata) []dependencyresolution.Metadata {
	out := []dependencyresolution.Metadata{}
	for _, m := range ms {
		if m.Key == modPathKey || m.Key == modSourceKey || m.Key == modDigestKey || strings.HasPrefix(m.Key, sourcePrefix) {
			continue
		}
		out = append(out, m)
	}
	return out
}
func metadataValue(ms []dependencyresolution.Metadata, key string) (string, bool) {
	for _, m := range ms {
		if m.Key == key {
			return m.Value, true
		}
	}
	return "", false
}
func clone(c dependencyresolution.Closure) dependencyresolution.Closure {
	out := c
	out.Requirements = append([]dependencyresolution.Requirement(nil), c.Requirements...)
	out.Entries = append([]dependencyresolution.Entry(nil), c.Entries...)
	for i := range out.Requirements {
		out.Requirements[i].Metadata = append([]dependencyresolution.Metadata(nil), c.Requirements[i].Metadata...)
	}
	for i := range out.Entries {
		out.Entries[i].Dependencies = append([]string(nil), c.Entries[i].Dependencies...)
		out.Entries[i].Metadata = append([]dependencyresolution.Metadata(nil), c.Entries[i].Metadata...)
	}
	return out
}
func closureEqual(a, b dependencyresolution.Closure) bool {
	dependencyresolution.Normalize(&a)
	dependencyresolution.Normalize(&b)
	return reflect.DeepEqual(a, b)
}
func withSchema(e wire.Envelope, s wire.ID) []wire.ID {
	out := []wire.ID{}
	for x, q := range e.Entities {
		if q.Schema == s {
			out = append(out, x)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}
func id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
