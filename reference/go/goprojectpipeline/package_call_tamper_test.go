package goprojectpipeline

import (
	"strings"
	"testing"

	"seme.local/reference/packagecallinstance"
	"seme.local/reference/packagedetailinstance"
	"seme.local/reference/packageinstance"
	"seme.local/reference/projectinstance"
	"seme.local/reference/projectsnapshot"
	"seme.local/reference/wire"
)

func TestPackageCallRejectsRetargetToPrivateAfterRevisionRecompute(t *testing.T) {
	r, err := Build(t.Context(), fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if err = packagecallinstance.Validate(r.PackageV2); err != nil {
		t.Fatal(err)
	}
	e, err := wire.Decode(r.PackageV2)
	if err != nil {
		t.Fatal(err)
	}
	var private, rootFunction, call wire.ID
	for x, q := range e.Entities {
		if q.Schema == mustID("9011") {
			name := string(q.Fields[mustID("9110")].Bytes)
			if name == "normalize" {
				private = x
			}
			if name == "Apply" {
				rootFunction = x
			}
		}
	}
	if private == (wire.ID{}) || rootFunction == (wire.ID{}) {
		t.Fatal("fixture functions missing")
	}
	rootBody := e.Entities[rootFunction].Fields[mustID("9113")].Reference
	seen := map[wire.ID]bool{}
	queue := []wire.ID{rootBody}
	for len(queue) > 0 {
		x := queue[0]
		queue = queue[1:]
		if seen[x] {
			continue
		}
		seen[x] = true
		q := e.Entities[x]
		if q.Schema == mustID("9060") {
			call = x
			break
		}
		for _, v := range q.Fields {
			collectTestRefs(v, &queue)
		}
	}
	if call == (wire.ID{}) {
		t.Fatal("root call missing")
	}
	q := e.Entities[call]
	q.Fields[mustID("9600")] = wire.Value{Tag: 6, Reference: private}
	e.Entities[call] = q
	recomputePackageProjectRevisions(t, &e)
	raw, err := wire.Encode(e)
	if err != nil {
		t.Fatal(err)
	}
	// All pre-existing Package v2 structural/revision checks accept this
	// same-signature retarget; only joint call authorization rejects it.
	if err = packagedetailinstance.Validate(raw); err != nil {
		t.Fatalf("structural package rejected before derived check: %v", err)
	}
	if err = packagecallinstance.Validate(raw); err == nil {
		t.Fatal("cross-package private retarget accepted")
	}
}

func TestPackageCallRejectsRecomputedAuthorizationAndShapeForgeries(t *testing.T) {
	tests := map[string]struct {
		want   string
		mutate func(*wire.Envelope)
	}{
		"missing-import": {"package_call.unauthorized", func(e *wire.Envelope) {
			for x, q := range e.Entities {
				if q.Schema == mustID("b021") {
					pkg := e.Entities[q.Fields[mustID("b210")].Reference]
					if string(pkg.Fields[mustID("b100")].Bytes) == "example.test/pipeline/app" {
						for _, v := range q.Fields[mustID("b212")].List {
							delete(e.Entities, v.Reference)
						}
						q.Fields[mustID("b212")] = wire.Value{Tag: 7}
						e.Entities[x] = q
					}
				}
			}
		}},
		"wrong-arity": {"package_call.arity", func(e *wire.Envelope) {
			fn := functionNamed(*e, "Apply")
			call := firstCall(*e, e.Entities[fn].Fields[mustID("9113")].Reference)
			q := e.Entities[call]
			q.Fields[mustID("9601")] = wire.Value{Tag: 7}
			e.Entities[call] = q
		}},
		"shared-body": {"package_call.shared_executable", func(e *wire.Envelope) {
			a, b := functionNamed(*e, "Apply"), functionNamed(*e, "AddOne")
			q := e.Entities[b]
			q.Fields[mustID("9113")] = e.Entities[a].Fields[mustID("9113")]
			e.Entities[b] = q
		}},
		"wrong-type": {"package_call.signature", func(e *wire.Envelope) {
			fn := functionNamed(*e, "normalize")
			q := e.Entities[fn]
			parameter := q.Fields[mustID("9111")].List[0].Reference
			var original wire.ID
			for x, v := range e.Entities {
				if v.Schema == mustID("9010") {
					original = x
					break
				}
			}
			other := mustID("ffffffffffffffffffffffffffff1001")
			typ := e.Entities[original]
			typ.ID = other
			typ.Fields = map[wire.ID]wire.Value{}
			for k, v := range e.Entities[original].Fields {
				typ.Fields[k] = v
			}
			typ.Fields[mustID("9101")] = wire.Value{Tag: 1}
			e.Entities[other] = typ
			p := e.Entities[parameter]
			p.Fields[mustID("9121")] = wire.Value{Tag: 6, Reference: other}
			e.Entities[parameter] = p
		}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r, err := Build(t.Context(), fixture(t))
			if err != nil {
				t.Fatal(err)
			}
			e, err := wire.Decode(r.PackageV2)
			if err != nil {
				t.Fatal(err)
			}
			tc.mutate(&e)
			recomputePackageProjectRevisions(t, &e)
			raw, err := wire.Encode(e)
			if err != nil {
				t.Fatal(err)
			}
			if err = packagedetailinstance.Validate(raw); err != nil {
				t.Fatalf("structural validator rejected first: %v", err)
			}
			if err = packagecallinstance.Validate(raw); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v want %s", err, tc.want)
			}
		})
	}
}

func functionNamed(e wire.Envelope, name string) wire.ID {
	for x, q := range e.Entities {
		if q.Schema == mustID("9011") && string(q.Fields[mustID("9110")].Bytes) == name {
			return x
		}
	}
	return wire.ID{}
}
func firstCall(e wire.Envelope, root wire.ID) wire.ID {
	seen := map[wire.ID]bool{}
	q := []wire.ID{root}
	for len(q) > 0 {
		x := q[0]
		q = q[1:]
		if seen[x] {
			continue
		}
		seen[x] = true
		v := e.Entities[x]
		if v.Schema == mustID("9060") {
			return x
		}
		for _, f := range v.Fields {
			collectTestRefs(f, &q)
		}
	}
	return wire.ID{}
}

func recomputePackageProjectRevisions(t *testing.T, e *wire.Envelope) {
	t.Helper()
	packages := []wire.ID{}
	for x, q := range e.Entities {
		if q.Schema == mustID("b010") {
			packages = append(packages, x)
		}
	}
	done := map[wire.ID]bool{}
	var visit func(wire.ID)
	visit = func(x wire.ID) {
		if done[x] {
			return
		}
		p := e.Entities[x]
		for _, dv := range p.Fields[mustID("b103")].List {
			d := e.Entities[dv.Reference]
			visit(d.Fields[mustID("b122")].Reference)
		}
		r, er := packageinstance.Revision(*e, x)
		if er != nil {
			t.Fatal(er)
		}
		p = e.Entities[x]
		p.Fields[mustID("b101")] = wire.Value{Tag: 5, Bytes: r}
		e.Entities[x] = p
		done[x] = true
	}
	for _, x := range packages {
		visit(x)
	}
	for x, q := range e.Entities {
		if q.Schema == mustID("e011") {
			identity := q.Fields[mustID("e110")].Reference
			ps := []wire.ID{}
			for _, v := range q.Fields[mustID("e112")].List {
				ps = append(ps, v.Reference)
			}
			r, er := projectsnapshot.Revision(*e, identity, ps, q.Fields[mustID("e113")].Reference, q.Fields[mustID("e114")].Reference)
			if er != nil {
				t.Fatal(er)
			}
			q.Fields[mustID("e111")] = wire.Value{Tag: 5, Bytes: r}
			e.Entities[x] = q
		}
	}
	for x, q := range e.Entities {
		if q.Schema == mustID("b021") {
			r, er := packagedetailinstance.DetailRevision(*e, x)
			if er != nil {
				t.Fatal(er)
			}
			q.Fields[mustID("b214")] = wire.Value{Tag: 5, Bytes: r}
			e.Entities[x] = q
		}
	}
	for x, q := range e.Entities {
		if q.Schema == mustID("b020") {
			r, er := packagedetailinstance.ContentRevision(*e, x)
			if er != nil {
				t.Fatal(er)
			}
			q.Fields[mustID("b201")] = wire.Value{Tag: 5, Bytes: r}
			e.Entities[x] = q
		}
	}
	r, er := projectinstance.ArtifactRevision(*e)
	if er != nil {
		t.Fatal(er)
	}
	e.Revision = r
}
func collectTestRefs(v wire.Value, q *[]wire.ID) {
	if v.Tag == 6 {
		*q = append(*q, v.Reference)
	}
	for _, x := range v.List {
		collectTestRefs(x, q)
	}
	for _, x := range v.Record {
		collectTestRefs(x, q)
	}
}
