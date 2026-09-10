// Package packagedetail is a language-neutral resolved package graph. It does
// not prescribe a source language, wire format, or execution target.
package packagedetail

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"path"
	"strings"
)

type Visibility uint8

const (
	Package Visibility = iota
	Project
	Public
)

type ImportClass uint8

const (
	Local ImportClass = iota
	External
)

type Origin struct {
	SourceIdentity, Path                       string
	ContentDigest                              [32]byte
	ByteStart, ByteEnd                         uint64
	StartLine, StartColumn, EndLine, EndColumn uint32
}
type Graph struct{ Packages []Detail }
type Detail struct {
	Identity string
	Root     bool
	Sources  []Source
	Members  []Member
	Imports  []Import
}
type Source struct {
	Identity, Path string
	ContentDigest  [32]byte
	ByteSize       uint64
}
type Member struct {
	Identity, Name, ExportName string
	Visibility                 Visibility
	Origin                     Origin
	Callable                   bool
	Parameters, Results        []string
}
type Import struct {
	Alias, Requested, Resolved string
	Class                      ImportClass
	Origin                     Origin
}

func Validate(g Graph) error {
	if len(g.Packages) == 0 {
		return fmt.Errorf("package_detail.empty")
	}
	roots := 0
	packages := map[string]bool{}
	members := map[string]bool{}
	sourceIdentities := map[string]string{}
	sourcePaths := map[string]string{}
	for i, p := range g.Packages {
		if p.Identity == "" || (i > 0 && g.Packages[i-1].Identity >= p.Identity) {
			return fmt.Errorf("package_detail.package_order")
		}
		packages[p.Identity] = true
		if p.Root {
			roots++
		}
		sources := map[string]Source{}
		for j, s := range p.Sources {
			if s.Identity == "" || !normalized(s.Path) || (j > 0 && p.Sources[j-1].Identity >= s.Identity) {
				return fmt.Errorf("package_detail.source_order")
			}
			sources[s.Identity] = s
			if _, exists := sourceIdentities[s.Identity]; exists {
				return fmt.Errorf("package_detail.source_identity_owner")
			}
			if _, exists := sourcePaths[s.Path]; exists {
				return fmt.Errorf("package_detail.source_path_owner")
			}
			sourceIdentities[s.Identity], sourcePaths[s.Path] = p.Identity, p.Identity
		}
		memberNames, exportNames := map[string]bool{}, map[string]bool{}
		for j, m := range p.Members {
			if m.Identity == "" || m.Name == "" || (j > 0 && p.Members[j-1].Identity >= m.Identity) {
				return fmt.Errorf("package_detail.member_order")
			}
			if members[m.Identity] {
				return fmt.Errorf("package_detail.member_duplicate")
			}
			members[m.Identity] = true
			if memberNames[m.Name] {
				return fmt.Errorf("package_detail.member_name_duplicate")
			}
			memberNames[m.Name] = true
			if m.Visibility > Public {
				return fmt.Errorf("package_detail.visibility")
			}
			if m.Visibility != Package && m.ExportName == "" {
				return fmt.Errorf("package_detail.visible_export")
			}
			if m.Visibility == Package && m.ExportName != "" {
				return fmt.Errorf("package_detail.package_export")
			}
			if m.ExportName != "" {
				if exportNames[m.ExportName] {
					return fmt.Errorf("package_detail.export_name_duplicate")
				}
				exportNames[m.ExportName] = true
			}
			if err := origin(m.Origin, sources); err != nil {
				return err
			}
			if !m.Callable && (len(m.Parameters) > 0 || len(m.Results) > 0) {
				return fmt.Errorf("package_detail.signature")
			}
			if empty(m.Parameters) || empty(m.Results) {
				return fmt.Errorf("package_detail.type_identity")
			}
		}
		aliases := map[string]bool{}
		for j, im := range p.Imports {
			if im.Requested == "" || im.Resolved == "" || im.Class > External {
				return fmt.Errorf("package_detail.import")
			}
			if j > 0 && !importLess(p.Imports[j-1], im) {
				return fmt.Errorf("package_detail.import_order")
			}
			if im.Alias != "" {
				if aliases[im.Alias] {
					return fmt.Errorf("package_detail.import_alias_duplicate")
				}
				aliases[im.Alias] = true
			}
			if im.Class == Local && im.Resolved == p.Identity {
				return fmt.Errorf("package_detail.local_self_import")
			}
			if err := origin(im.Origin, sources); err != nil {
				return err
			}
		}
	}
	if roots != 1 {
		return fmt.Errorf("package_detail.root_count")
	}
	for _, p := range g.Packages {
		for _, im := range p.Imports {
			if im.Class == Local && !packages[im.Resolved] {
				return fmt.Errorf("package_detail.local_import")
			}
		}
	}
	return nil
}
func normalized(p string) bool {
	return p != "" && !strings.HasPrefix(p, "/") && !strings.Contains(p, "\\") && path.Clean(p) == p && p != "." && !strings.HasPrefix(p, "../")
}
func Revision(g Graph) ([32]byte, error) {
	if err := Validate(g); err != nil {
		return [32]byte{}, err
	}
	h := sha256.New()
	s(h.Write, "seme.package-detail.v2")
	u(h.Write, uint64(len(g.Packages)))
	for _, p := range g.Packages {
		s(h.Write, p.Identity)
		b(h.Write, p.Root)
		u(h.Write, uint64(len(p.Sources)))
		for _, x := range p.Sources {
			s(h.Write, x.Identity)
			s(h.Write, x.Path)
			h.Write(x.ContentDigest[:])
			u(h.Write, x.ByteSize)
		}
		u(h.Write, uint64(len(p.Members)))
		for _, m := range p.Members {
			s(h.Write, m.Identity)
			s(h.Write, m.Name)
			s(h.Write, m.ExportName)
			h.Write([]byte{byte(m.Visibility)})
			o(h.Write, m.Origin)
			b(h.Write, m.Callable)
			ss(h.Write, m.Parameters)
			ss(h.Write, m.Results)
		}
		u(h.Write, uint64(len(p.Imports)))
		for _, x := range p.Imports {
			s(h.Write, x.Alias)
			s(h.Write, x.Requested)
			s(h.Write, x.Resolved)
			h.Write([]byte{byte(x.Class)})
			o(h.Write, x.Origin)
		}
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out, nil
}
func origin(o Origin, sources map[string]Source) error {
	x, ok := sources[o.SourceIdentity]
	if !ok || x.Path != o.Path || x.ContentDigest != o.ContentDigest {
		return fmt.Errorf("package_detail.origin_source")
	}
	if o.ByteStart >= o.ByteEnd || o.ByteEnd > x.ByteSize || o.StartLine == 0 || o.StartColumn == 0 || o.EndLine == 0 || o.EndColumn == 0 || o.EndLine < o.StartLine || (o.EndLine == o.StartLine && o.EndColumn <= o.StartColumn) {
		return fmt.Errorf("package_detail.origin_range")
	}
	return nil
}
func importLess(a, b Import) bool {
	if a.Origin.SourceIdentity != b.Origin.SourceIdentity {
		return a.Origin.SourceIdentity < b.Origin.SourceIdentity
	}
	if a.Origin.ByteStart != b.Origin.ByteStart {
		return a.Origin.ByteStart < b.Origin.ByteStart
	}
	if a.Requested != b.Requested {
		return a.Requested < b.Requested
	}
	return a.Alias < b.Alias
}
func empty(x []string) bool {
	for _, v := range x {
		if v == "" {
			return true
		}
	}
	return false
}

type writer func([]byte) (int, error)

func u(w writer, n uint64) { var x [8]byte; binary.BigEndian.PutUint64(x[:], n); w(x[:]) }
func s(w writer, x string) { u(w, uint64(len(x))); w([]byte(x)) }
func b(w writer, x bool) {
	if x {
		w([]byte{1})
	} else {
		w([]byte{0})
	}
}
func ss(w writer, x []string) {
	u(w, uint64(len(x)))
	for _, v := range x {
		s(w, v)
	}
}
func o(w writer, x Origin) {
	s(w, x.SourceIdentity)
	s(w, x.Path)
	w(x.ContentDigest[:])
	u(w, x.ByteStart)
	u(w, x.ByteEnd)
	u(w, uint64(x.StartLine))
	u(w, uint64(x.StartColumn))
	u(w, uint64(x.EndLine))
	u(w, uint64(x.EndColumn))
}
