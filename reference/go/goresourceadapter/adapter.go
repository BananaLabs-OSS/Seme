// Package goresourceadapter binds explicit resource selections to one
// authenticated Project-v8 source/package snapshot and captures detached bytes.
package goresourceadapter

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"seme.local/reference/goresourcemanifest"
	"seme.local/reference/projectsource"
	"seme.local/reference/projectv8instance"
	"seme.local/reference/resourceinstance"
	"seme.local/reference/sourceinventory"
	"seme.local/reference/wire"
	"sort"
	"strings"
	"unicode/utf8"
)

type Resource struct {
	Identity          string
	Owner, SourceUnit wire.ID
	Path              string
	Kind              uint64
	MediaType         string
	Size              uint64
	SHA256            [32]byte
}

func InstanceModel(in Model) resourceinstance.Model {
	out := resourceinstance.Model{}
	for _, r := range in.Resources {
		out.Resources = append(out.Resources, resourceinstance.Resource{Identity: r.Identity, Owner: r.Owner, SourceUnit: r.SourceUnit, Path: r.Path, Kind: r.Kind, MediaType: r.MediaType, Size: r.Size, SHA256: r.SHA256})
	}
	for _, p := range in.Placements {
		out.Placements = append(out.Placements, resourceinstance.Placement{Resource: p.Resource, Destination: p.Destination})
	}
	return out
}

type Placement struct{ Resource, Destination string }
type Model struct {
	Resources  []Resource
	Placements []Placement
}
type Blob struct {
	SHA256 [32]byte
	Bytes  []byte
}
type Store struct{ Blobs []Blob }

func Resolve(root string, snapshot projectsource.Snapshot, project projectv8instance.Inputs, ownerPackage string, selections []goresourcemanifest.Selection) (Model, Store, error) {
	if ownerPackage == "" || len(selections) == 0 {
		return Model{}, Store{}, fmt.Errorf("go_resource.selection")
	}
	if err := projectv8instance.Validate(project); err != nil {
		return Model{}, Store{}, fmt.Errorf("go_resource.project:%w", err)
	}
	if err := projectsource.ValidateSnapshot(snapshot); err != nil {
		return Model{}, Store{}, err
	}
	if err := sourceinventory.ValidateV8(project.Contracts.Project(), project.ProjectBase, project.Inventory); err != nil {
		return Model{}, Store{}, fmt.Errorf("go_resource.inventory:%w", err)
	}
	inv, err := wire.Decode(project.Inventory)
	if err != nil {
		return Model{}, Store{}, err
	}
	pkg, err := wire.Decode(project.PackageV4)
	if err != nil {
		return Model{}, Store{}, err
	}
	owners := map[string]wire.ID{}
	for x, q := range pkg.Entities {
		if q.Schema == id("b010") {
			n := q.Fields[id("b100")]
			if n.Tag != 5 || owners[string(n.Bytes)] != (wire.ID{}) {
				return Model{}, Store{}, fmt.Errorf("go_resource.package")
			}
			owners[string(n.Bytes)] = x
		}
	}
	type unit struct {
		id     wire.ID
		size   uint64
		digest [32]byte
	}
	units := map[string]unit{}
	for x, q := range inv.Entities {
		if q.Schema != id("e015") {
			continue
		}
		p, d, z := q.Fields[id("e150")], q.Fields[id("e151")], q.Fields[id("e152")]
		if p.Tag != 5 || d.Tag != 5 || len(d.Bytes) != 32 || z.Tag != 3 {
			return Model{}, Store{}, fmt.Errorf("go_resource.source_unit")
		}
		var sum [32]byte
		copy(sum[:], d.Bytes)
		units[string(p.Bytes)] = unit{x, z.Unsigned, sum}
	}
	byPath := map[string]projectsource.Unit{}
	for _, u := range snapshot.Units {
		byPath[u.Path] = u
	}
	out := Model{}
	blobs := map[[32]byte][]byte{}
	owner := owners[ownerPackage]
	if owner == (wire.ID{}) {
		return Model{}, Store{}, fmt.Errorf("go_resource.owner")
	}
	for _, s := range selections {
		u, ok := units[s.Path]
		su, sok := byPath[s.Path]
		if !ok || !sok || su.Preservation != projectsource.ByteExact || uint64(su.Size) != u.size || su.SHA256 != hex.EncodeToString(u.digest[:]) || s.Size != u.size || s.SHA256 != u.digest {
			return Model{}, Store{}, fmt.Errorf("go_resource.selection:%s", s.Identity)
		}
		data, er := projectsource.ReadVerified(root, su)
		if er != nil {
			return Model{}, Store{}, er
		}
		sum := sha256.Sum256(data)
		if sum != u.digest || uint64(len(data)) != u.size {
			return Model{}, Store{}, fmt.Errorf("go_resource.digest:%s", s.Identity)
		}
		kind := uint64(1)
		if strings.HasPrefix(s.MediaType, "text/") {
			kind = 0
			if !utf8.Valid(data) {
				return Model{}, Store{}, fmt.Errorf("go_resource.text_utf8:%s", s.Identity)
			}
		}
		if prior, exists := blobs[sum]; exists && !bytes.Equal(prior, data) {
			return Model{}, Store{}, fmt.Errorf("go_resource.digest_collision")
		}
		blobs[sum] = append([]byte(nil), data...)
		out.Resources = append(out.Resources, Resource{s.Identity, owner, u.id, s.Path, kind, s.MediaType, u.size, sum})
		out.Placements = append(out.Placements, Placement{s.Identity, s.Destination})
	}
	sort.Slice(out.Resources, func(i, j int) bool { return out.Resources[i].Identity < out.Resources[j].Identity })
	sort.Slice(out.Placements, func(i, j int) bool {
		if out.Placements[i].Resource == out.Placements[j].Resource {
			return out.Placements[i].Destination < out.Placements[j].Destination
		}
		return out.Placements[i].Resource < out.Placements[j].Resource
	})
	keys := make([][32]byte, 0, len(blobs))
	for k := range blobs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i][:], keys[j][:]) < 0 })
	store := Store{}
	for _, k := range keys {
		store.Blobs = append(store.Blobs, Blob{k, blobs[k]})
	}
	return out, store, nil
}
func (s Store) Lookup(sum [32]byte) ([]byte, bool) {
	i := sort.Search(len(s.Blobs), func(i int) bool { return bytes.Compare(s.Blobs[i].SHA256[:], sum[:]) >= 0 })
	if i == len(s.Blobs) || s.Blobs[i].SHA256 != sum {
		return nil, false
	}
	return append([]byte(nil), s.Blobs[i].Bytes...), true
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
