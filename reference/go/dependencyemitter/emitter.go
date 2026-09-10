// Package dependencyemitter emits canonical Dependency Contract v1 instances.
package dependencyemitter

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyinstance"
	"seme.local/reference/dependencyresolution"
	"seme.local/reference/wire"
)

var (
	moduleSchema      = id("00000000000000000000000000000012")
	importSchema      = id("00000000000000000000000000000013")
	moduleID          = id("0000000000000000000000000000f000")
	revisionID        = id("0000000000000000000000000000f001")
	closureSchema     = id("0000000000000000000000000000f010")
	requirementSchema = id("0000000000000000000000000000f011")
	resolvedSchema    = id("0000000000000000000000000000f012")
	ecosystemSchema   = id("0000000000000000000000000000f013")
	kindSchema        = id("0000000000000000000000000000f014")
	metadataSchema    = id("0000000000000000000000000000f015")
	integritySchema   = id("0000000000000000000000000000f016")
	sourceSchema      = id("0000000000000000000000000000f017")
)

// Emit validates c and returns the unique canonical instance bytes. It never
// returns partial bytes on error.
func Emit(contract contractcatalog.Contract, c dependencyresolution.Closure) ([]byte, error) {
	if !contract.Validated() || contract.Pin() != (contractcatalog.Pin{Module: moduleID, Revision: revisionID}) {
		return nil, fmt.Errorf("dependency_emitter.contract")
	}
	if err := dependencyresolution.Validate(c); err != nil {
		return nil, err
	}
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	mod := stable("module")
	imp := stable("import")
	closure := stable("closure")
	e.Module = mod
	put(&e, wire.Entity{ID: imp, Schema: importSchema, Version: 1, Fields: map[wire.ID]wire.Value{
		id("00000000000000000000000000000130"): ref(moduleID), id("00000000000000000000000000000131"): blob(revisionID[:])}})

	kinds := map[dependencyresolution.Kind]wire.ID{}
	usedKinds := map[dependencyresolution.Kind]bool{}
	for _, r := range c.Requirements {
		usedKinds[r.Kind] = true
	}
	for _, x := range c.Entries {
		usedKinds[x.Kind] = true
	}
	for _, k := range []dependencyresolution.Kind{dependencyresolution.Local, dependencyresolution.External} {
		if !usedKinds[k] {
			continue
		}
		kid := stable("kind", fmt.Sprint(k))
		kinds[k] = kid
		put(&e, wire.Entity{ID: kid, Schema: kindSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("0000000000000000000000000000f140"): {Tag: 3, Unsigned: uint64(k)}}})
	}
	meta := func(role, owner string, ms []dependencyresolution.Metadata) []wire.Value {
		out := make([]wire.Value, 0, len(ms))
		for _, m := range ms {
			x := stable("metadata", role, owner, m.Key, m.Value)
			put(&e, wire.Entity{ID: x, Schema: metadataSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("0000000000000000000000000000f150"): blob([]byte(m.Key)), id("0000000000000000000000000000f151"): blob([]byte(m.Value))}})
			out = append(out, ref(x))
		}
		sortValues(out)
		return out
	}
	resolved := map[string]wire.ID{}
	for _, x := range c.Entries {
		resolved[x.Identity] = stable("resolved", x.Identity)
	}
	reqs := []wire.Value{}
	for _, r := range c.Requirements {
		rid := stable("requirement", r.Identity, r.Requirement, fmt.Sprint(r.Kind))
		reqs = append(reqs, ref(rid))
		put(&e, wire.Entity{ID: rid, Schema: requirementSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("0000000000000000000000000000f110"): blob([]byte(r.Identity)), id("0000000000000000000000000000f111"): blob([]byte(r.Requirement)), id("0000000000000000000000000000f112"): ref(kinds[r.Kind]), id("0000000000000000000000000000f113"): list(meta("requirement", r.Identity, r.Metadata))}})
	}
	sortValues(reqs)
	ress := []wire.Value{}
	for _, x := range c.Entries {
		xid := resolved[x.Identity]
		ress = append(ress, ref(xid))
		deps := []wire.Value{}
		for _, d := range x.Dependencies {
			deps = append(deps, ref(resolved[d]))
		}
		sortValues(deps)
		iid := stable("integrity", x.Identity, x.IntegrityAlgorithm, x.Integrity)
		put(&e, wire.Entity{ID: iid, Schema: integritySchema, Version: 1, Fields: map[wire.ID]wire.Value{id("0000000000000000000000000000f160"): blob([]byte(x.IntegrityAlgorithm)), id("0000000000000000000000000000f161"): blob([]byte(x.Integrity))}})
		sm := []dependencyresolution.Metadata{{Key: "content.digest", Value: x.Digest}}
		sid := stable("source", x.Identity, x.SourceKind, x.Source, x.Digest)
		put(&e, wire.Entity{ID: sid, Schema: sourceSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("0000000000000000000000000000f170"): blob([]byte(x.SourceKind)), id("0000000000000000000000000000f171"): blob([]byte(x.Source)), id("0000000000000000000000000000f172"): list(meta("source", x.Identity, sm))}})
		fields := map[wire.ID]wire.Value{id("0000000000000000000000000000f120"): blob([]byte(x.Identity)), id("0000000000000000000000000000f122"): blob([]byte(x.Version)), id("0000000000000000000000000000f123"): ref(iid), id("0000000000000000000000000000f124"): ref(sid), id("0000000000000000000000000000f125"): ref(kinds[x.Kind]), id("0000000000000000000000000000f126"): list(deps), id("0000000000000000000000000000f127"): list(meta("resolved", x.Identity, x.Metadata))}
		if x.Ecosystem != "" {
			eid := stable("ecosystem", x.Ecosystem)
			if _, ok := e.Entities[eid]; !ok {
				put(&e, wire.Entity{ID: eid, Schema: ecosystemSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("0000000000000000000000000000f130"): blob([]byte(x.Ecosystem))}})
			}
			fields[id("0000000000000000000000000000f121")] = ref(eid)
		}
		put(&e, wire.Entity{ID: xid, Schema: resolvedSchema, Version: 1, Fields: fields})
	}
	sortValues(ress)
	digest, err := Revision(c)
	if err != nil {
		return nil, err
	}
	put(&e, wire.Entity{ID: closure, Schema: closureSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("0000000000000000000000000000f100"): list(reqs), id("0000000000000000000000000000f101"): list(ress), id("0000000000000000000000000000f102"): blob(digest)}})
	put(&e, wire.Entity{ID: mod, Schema: moduleSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("00000000000000000000000000000120"): blob([]byte("seme.dependency-instance")), id("00000000000000000000000000000121"): list([]wire.Value{ref(imp)}), id("00000000000000000000000000000122"): list([]wire.Value{ref(closure)})}})
	e.Revision = artifactRevision(e)
	out, err := wire.Encode(e)
	if err != nil {
		return nil, err
	}
	if _, err = dependencyinstance.Validate(contract, out); err != nil {
		return nil, err
	}
	return out, nil
}

func Revision(c dependencyresolution.Closure) ([]byte, error) {
	dependencyresolution.Normalize(&c)
	if len(c.Requirements) == 0 {
		c.Requirements = nil
	}
	if len(c.Entries) == 0 {
		c.Entries = nil
	}
	for i := range c.Requirements {
		if len(c.Requirements[i].Metadata) == 0 {
			c.Requirements[i].Metadata = nil
		}
	}
	for i := range c.Entries {
		if len(c.Entries[i].Dependencies) == 0 {
			c.Entries[i].Dependencies = nil
		}
		if len(c.Entries[i].Metadata) == 0 {
			c.Entries[i].Metadata = nil
		}
	}
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Write([]byte("seme.dependency-closure.v1\x00"))
	binary.Write(h, binary.BigEndian, uint64(len(b)))
	h.Write(b)
	return h.Sum(nil), nil
}
func artifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.dependency-artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func stable(parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.dependency-instance.identity.v1\x00"))
	for _, p := range parts {
		binary.Write(h, binary.BigEndian, uint64(len(p)))
		h.Write([]byte(p))
	}
	var x wire.ID
	copy(x[:], h.Sum(nil)[:16])
	return x
}
func id(s string) wire.ID {
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
func put(e *wire.Envelope, x wire.Entity) {
	if _, ok := e.Entities[x.ID]; ok {
		return
	}
	e.Entities[x.ID] = x
}
func ref(x wire.ID) wire.Value       { return wire.Value{Tag: 6, Reference: x} }
func blob(x []byte) wire.Value       { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func list(x []wire.Value) wire.Value { return wire.Value{Tag: 7, List: x} }
func sortValues(x []wire.Value) {
	sort.Slice(x, func(i, j int) bool { return bytes.Compare(x[i].Reference[:], x[j].Reference[:]) < 0 })
}
