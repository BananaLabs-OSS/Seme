// Package dependencyinstance validates canonical Dependency Contract v1 instances.
package dependencyinstance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyresolution"
	"seme.local/reference/wire"
)

var moduleID = id("0000000000000000000000000000f000")
var revisionID = id("0000000000000000000000000000f001")

// Validate authenticates the contract and validates the complete closed instance.
func Validate(contract contractcatalog.Contract, source []byte) (dependencyresolution.Closure, error) {
	var zero dependencyresolution.Closure
	if !contract.Validated() || contract.Pin() != (contractcatalog.Pin{Module: moduleID, Revision: revisionID}) {
		return zero, fmt.Errorf("dependency_instance.contract")
	}
	e, err := wire.Decode(source)
	if err != nil {
		return zero, err
	}
	canonical, err := wire.Encode(e)
	if err != nil || !bytes.Equal(canonical, source) {
		return zero, fmt.Errorf("dependency_instance.noncanonical")
	}
	if len(e.Parents) != 0 {
		return zero, fmt.Errorf("dependency_instance.parents")
	}
	mod, ok := e.Entities[e.Module]
	if e.Module != stable("module") {
		return zero, fmt.Errorf("dependency_instance.module_identity")
	}
	if !ok || mod.Schema != id("00000000000000000000000000000012") || mod.Version != 1 || !fields(mod, "120", "121", "122") {
		return zero, fmt.Errorf("dependency_instance.module")
	}
	if name, err := text(mod, "120"); err != nil || name != "seme.dependency-instance" {
		return zero, fmt.Errorf("dependency_instance.module_name")
	}
	imps, err := refs(mod.Fields[idf("121")], true)
	if err != nil || len(imps) != 1 {
		return zero, fmt.Errorf("dependency_instance.imports")
	}
	imp := e.Entities[imps[0]]
	if imp.ID != stable("import") {
		return zero, fmt.Errorf("dependency_instance.import_identity")
	}
	if imp.Schema != id("00000000000000000000000000000013") || imp.Version != 1 || !fields(imp, "130", "131") || imp.Fields[idf("130")].Tag != 6 || imp.Fields[idf("130")].Reference != moduleID || imp.Fields[idf("131")].Tag != 5 || !bytes.Equal(imp.Fields[idf("131")].Bytes, revisionID[:]) {
		return zero, fmt.Errorf("dependency_instance.pin")
	}
	exps, err := refs(mod.Fields[idf("122")], true)
	if err != nil || len(exps) != 1 {
		return zero, fmt.Errorf("dependency_instance.exports")
	}
	ce := e.Entities[exps[0]]
	if ce.ID != stable("closure") {
		return zero, fmt.Errorf("dependency_instance.closure_identity")
	}
	if ce.Schema != id("0000000000000000000000000000f010") || ce.Version != 1 || !fields(ce, "f100", "f101", "f102") || ce.Fields[id("0000000000000000000000000000f102")].Tag != 5 || len(ce.Fields[id("0000000000000000000000000000f102")].Bytes) != 32 {
		return zero, fmt.Errorf("dependency_instance.closure")
	}
	used := map[wire.ID]bool{e.Module: true, imp.ID: true, ce.ID: true}
	kinds := map[wire.ID]dependencyresolution.Kind{}
	parseKind := func(x wire.ID) (dependencyresolution.Kind, error) {
		if k, ok := kinds[x]; ok {
			return k, nil
		}
		q, ok := e.Entities[x]
		if !ok || q.Schema != id("0000000000000000000000000000f014") || q.Version != 1 || !fields(q, "f140") || q.Fields[id("0000000000000000000000000000f140")].Tag != 3 || q.Fields[id("0000000000000000000000000000f140")].Unsigned > 1 {
			return 0, fmt.Errorf("kind")
		}
		used[x] = true
		k := dependencyresolution.Kind(q.Fields[id("0000000000000000000000000000f140")].Unsigned)
		if x != stable("kind", fmt.Sprint(k)) {
			return 0, fmt.Errorf("kind_identity")
		}
		kinds[x] = k
		return k, nil
	}
	parseMeta := func(role, owner string, v wire.Value) ([]dependencyresolution.Metadata, error) {
		xs, er := refs(v, true)
		if er != nil {
			return nil, er
		}
		out := []dependencyresolution.Metadata{}
		for _, x := range xs {
			q, ok := e.Entities[x]
			if !ok || q.Schema != id("0000000000000000000000000000f015") || q.Version != 1 || !fields(q, "f150", "f151") {
				return nil, fmt.Errorf("metadata")
			}
			k, er := text(q, "f150")
			if er != nil {
				return nil, er
			}
			val, er := text(q, "f151")
			if er != nil {
				return nil, er
			}
			used[x] = true
			if x != stable("metadata", role, owner, k, val) {
				return nil, fmt.Errorf("metadata_identity")
			}
			out = append(out, dependencyresolution.Metadata{Key: k, Value: val})
		}
		return out, nil
	}
	resrefs, err := refs(ce.Fields[id("0000000000000000000000000000f101")], true)
	if err != nil {
		return zero, err
	}
	identities := map[wire.ID]string{}
	for _, x := range resrefs {
		q := e.Entities[x]
		s, er := text(q, "f120")
		if er != nil {
			return zero, er
		}
		identities[x] = s
		if x != stable("resolved", s) {
			return zero, fmt.Errorf("dependency_instance.resolution_identity")
		}
	}
	for _, x := range resrefs {
		q := e.Entities[x]
		if q.Schema != id("0000000000000000000000000000f012") || q.Version != 1 {
			return zero, fmt.Errorf("dependency_instance.resolution")
		}
		allowed := []string{"f120", "f122", "f123", "f124", "f125", "f126", "f127"}
		if _, ok := q.Fields[id("0000000000000000000000000000f121")]; ok {
			allowed = append(allowed, "f121")
		}
		if !fields(q, allowed...) {
			return zero, fmt.Errorf("dependency_instance.resolution_fields")
		}
		used[x] = true
		identity, _ := text(q, "f120")
		version, er := text(q, "f122")
		if er != nil {
			return zero, er
		}
		kindRef, er := oneRef(q, "f125")
		if er != nil {
			return zero, er
		}
		kind, er := parseKind(kindRef)
		if er != nil {
			return zero, er
		}
		ecosystem := ""
		if v, ok := q.Fields[id("0000000000000000000000000000f121")]; ok {
			if v.Tag != 6 {
				return zero, fmt.Errorf("ecosystem_ref")
			}
			ec := e.Entities[v.Reference]
			if ec.Schema != id("0000000000000000000000000000f013") || ec.Version != 1 || !fields(ec, "f130") {
				return zero, fmt.Errorf("ecosystem")
			}
			ecosystem, er = text(ec, "f130")
			if er != nil {
				return zero, er
			}
			used[ec.ID] = true
			if ec.ID != stable("ecosystem", ecosystem) {
				return zero, fmt.Errorf("ecosystem_identity")
			}
		}
		ii, er := oneRef(q, "f123")
		if er != nil {
			return zero, er
		}
		ie := e.Entities[ii]
		if ie.Schema != id("0000000000000000000000000000f016") || ie.Version != 1 || !fields(ie, "f160", "f161") {
			return zero, fmt.Errorf("integrity")
		}
		alg, _ := text(ie, "f160")
		integrity, _ := text(ie, "f161")
		used[ii] = true
		if ii != stable("integrity", identity, alg, integrity) {
			return zero, fmt.Errorf("integrity_identity")
		}
		si, er := oneRef(q, "f124")
		if er != nil {
			return zero, er
		}
		se := e.Entities[si]
		if se.Schema != id("0000000000000000000000000000f017") || se.Version != 1 || !fields(se, "f170", "f171", "f172") {
			return zero, fmt.Errorf("source")
		}
		sk, _ := text(se, "f170")
		src, _ := text(se, "f171")
		sms, er := parseMeta("source", identity, se.Fields[id("0000000000000000000000000000f172")])
		if er != nil || len(sms) != 1 || sms[0].Key != "content.digest" {
			return zero, fmt.Errorf("source_digest")
		}
		used[si] = true
		if si != stable("source", identity, sk, src, sms[0].Value) {
			return zero, fmt.Errorf("source_identity")
		}
		deps0, er := refs(q.Fields[id("0000000000000000000000000000f126")], true)
		if er != nil {
			return zero, er
		}
		deps := []string{}
		for _, d := range deps0 {
			v, ok := identities[d]
			if !ok {
				return zero, fmt.Errorf("dependency_ref")
			}
			deps = append(deps, v)
		}
		ms, er := parseMeta("resolved", identity, q.Fields[id("0000000000000000000000000000f127")])
		if er != nil {
			return zero, er
		}
		zero.Entries = append(zero.Entries, dependencyresolution.Entry{Identity: identity, Ecosystem: ecosystem, Version: version, Integrity: integrity, IntegrityAlgorithm: alg, Source: src, SourceKind: sk, Digest: sms[0].Value, Kind: kind, Dependencies: deps, Metadata: ms})
	}
	reqrefs, err := refs(ce.Fields[id("0000000000000000000000000000f100")], true)
	if err != nil {
		return zero, err
	}
	for _, x := range reqrefs {
		q := e.Entities[x]
		if q.Schema != id("0000000000000000000000000000f011") || q.Version != 1 || !fields(q, "f110", "f111", "f112", "f113") {
			return zero, fmt.Errorf("requirement")
		}
		identity, _ := text(q, "f110")
		requirement, _ := text(q, "f111")
		kr, er := oneRef(q, "f112")
		if er != nil {
			return zero, er
		}
		kind, er := parseKind(kr)
		if er != nil {
			return zero, er
		}
		ms, er := parseMeta("requirement", identity, q.Fields[id("0000000000000000000000000000f113")])
		if er != nil {
			return zero, er
		}
		used[x] = true
		if x != stable("requirement", identity, requirement, fmt.Sprint(kind)) {
			return zero, fmt.Errorf("requirement_identity")
		}
		zero.Requirements = append(zero.Requirements, dependencyresolution.Requirement{Identity: identity, Requirement: requirement, Kind: kind, Metadata: ms})
	}
	if len(used) != len(e.Entities) {
		return zero, fmt.Errorf("dependency_instance.orphan")
	}
	dependencyresolution.Normalize(&zero)
	if err := dependencyresolution.Validate(zero); err != nil {
		return zero, err
	}
	want, err := closureRevision(zero)
	if err != nil {
		return zero, err
	}
	if !bytes.Equal(want, ce.Fields[id("0000000000000000000000000000f102")].Bytes) {
		return zero, fmt.Errorf("dependency_instance.content_revision")
	}
	if e.Revision != artifactRevision(e) {
		return zero, fmt.Errorf("dependency_instance.artifact_revision")
	}
	return zero, nil
}

func fields(e wire.Entity, s ...string) bool {
	if len(e.Fields) != len(s) {
		return false
	}
	for _, x := range s {
		if _, ok := e.Fields[idf(x)]; !ok {
			return false
		}
	}
	return true
}
func idf(s string) wire.ID { return id(strings.Repeat("0", 32-len(s)) + s) }
func text(e wire.Entity, f string) (string, error) {
	v, ok := e.Fields[idf(f)]
	if !ok || v.Tag != 5 || len(v.Bytes) == 0 || !utf8.Valid(v.Bytes) {
		return "", fmt.Errorf("text:%s", f)
	}
	return string(v.Bytes), nil
}
func oneRef(e wire.Entity, f string) (wire.ID, error) {
	v, ok := e.Fields[idf(f)]
	if !ok || v.Tag != 6 {
		return wire.ID{}, fmt.Errorf("ref:%s", f)
	}
	return v.Reference, nil
}
func refs(v wire.Value, sorted bool) ([]wire.ID, error) {
	if v.Tag != 7 {
		return nil, fmt.Errorf("list")
	}
	out := make([]wire.ID, len(v.List))
	for i, x := range v.List {
		if x.Tag != 6 {
			return nil, fmt.Errorf("list_ref")
		}
		out[i] = x.Reference
		if sorted && i > 0 && bytes.Compare(out[i-1][:], out[i][:]) >= 0 {
			return nil, fmt.Errorf("list_order")
		}
	}
	return out, nil
}
func closureRevision(c dependencyresolution.Closure) ([]byte, error) {
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
	_ = binary.Write(h, binary.BigEndian, uint64(len(b)))
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
		_ = binary.Write(h, binary.BigEndian, uint64(len(p)))
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
