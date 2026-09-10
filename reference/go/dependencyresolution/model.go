// Package dependencyresolution is a language-neutral immutable dependency closure.
package dependencyresolution

import (
	"fmt"
	"sort"
)

type Kind uint8

const (
	Local Kind = iota
	External
)

type Metadata struct{ Key, Value string }
type Requirement struct {
	Identity, Requirement string
	Kind                  Kind
	Metadata              []Metadata
}
type Entry struct {
	Identity, Ecosystem, Version, Integrity, IntegrityAlgorithm, Source, SourceKind, Digest string
	Kind                                                                                    Kind
	Dependencies                                                                            []string
	Metadata                                                                                []Metadata
}
type Closure struct {
	Requirements []Requirement
	Entries      []Entry
}

func Validate(c Closure) error {
	if len(c.Requirements) == 0 || len(c.Entries) == 0 {
		return fmt.Errorf("dependency_resolution.shape")
	}
	seen := map[string]bool{}
	by := map[string]Entry{}
	prior := ""
	for _, x := range c.Entries {
		if x.Identity == "" || x.Source == "" || x.SourceKind == "" || x.Digest == "" || x.Identity <= prior || seen[x.Identity] || x.Kind > External {
			return fmt.Errorf("dependency_resolution.entry")
		}
		if x.Version == "" || x.Integrity == "" || x.IntegrityAlgorithm == "" {
			return fmt.Errorf("dependency_resolution.integrity")
		}
		if x.Kind == External && x.Ecosystem == "" || x.Kind == Local && x.Ecosystem != "" {
			return fmt.Errorf("dependency_resolution.ecosystem")
		}
		if metadata(x.Metadata) != nil {
			return fmt.Errorf("dependency_resolution.metadata")
		}
		pd := ""
		for _, d := range x.Dependencies {
			if d == "" || d <= pd || d == x.Identity {
				return fmt.Errorf("dependency_resolution.dependencies")
			}
			pd = d
		}
		prior = x.Identity
		seen[x.Identity] = true
		by[x.Identity] = x
	}
	prior = ""
	for _, r := range c.Requirements {
		if r.Identity == "" || r.Requirement == "" || r.Identity <= prior || r.Kind > External || !seen[r.Identity] || metadata(r.Metadata) != nil {
			return fmt.Errorf("dependency_resolution.requirement")
		}
		resolved := by[r.Identity]
		if resolved.Kind != r.Kind || resolved.Version != r.Requirement {
			return fmt.Errorf("dependency_resolution.requirement_match")
		}
		prior = r.Identity
	}
	for _, x := range c.Entries {
		for _, d := range x.Dependencies {
			if !seen[d] {
				return fmt.Errorf("dependency_resolution.dependency_missing")
			}
		}
	}
	done := map[string]bool{}
	var visit func(string, map[string]bool) error
	visit = func(x string, active map[string]bool) error {
		if active[x] {
			return fmt.Errorf("dependency_resolution.cycle")
		}
		if done[x] {
			return nil
		}
		next := map[string]bool{}
		for k := range active {
			next[k] = true
		}
		next[x] = true
		for _, d := range by[x].Dependencies {
			if err := visit(d, next); err != nil {
				return err
			}
		}
		done[x] = true
		return nil
	}
	for _, r := range c.Requirements {
		if err := visit(r.Identity, map[string]bool{}); err != nil {
			return err
		}
	}
	if len(done) != len(c.Entries) {
		return fmt.Errorf("dependency_resolution.unreachable")
	}
	return nil
}
func metadata(x []Metadata) error {
	prior := ""
	for _, m := range x {
		if m.Key == "" || m.Value == "" || m.Key <= prior {
			return fmt.Errorf("metadata")
		}
		prior = m.Key
	}
	return nil
}
func Normalize(c *Closure) {
	sort.Slice(c.Entries, func(i, j int) bool { return c.Entries[i].Identity < c.Entries[j].Identity })
	for i := range c.Entries {
		sort.Strings(c.Entries[i].Dependencies)
		sort.Slice(c.Entries[i].Metadata, func(a, b int) bool { return c.Entries[i].Metadata[a].Key < c.Entries[i].Metadata[b].Key })
	}
	sort.Slice(c.Requirements, func(i, j int) bool { return c.Requirements[i].Identity < c.Requirements[j].Identity })
	for i := range c.Requirements {
		sort.Slice(c.Requirements[i].Metadata, func(a, b int) bool { return c.Requirements[i].Metadata[a].Key < c.Requirements[i].Metadata[b].Key })
	}
}
