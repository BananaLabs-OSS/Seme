package goorderedtransportadapter_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goorderedtransportadapter"
	"seme.local/reference/internal/upb07testfixture"
	"seme.local/reference/orderedtransportinstance"
	"seme.local/reference/projectv11instance"
	"seme.local/reference/wire"
)

func TestAuthenticatedSelectionAndProjectV11Instance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fx, err := upb07testfixture.Build(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	contracts := resolveV11(t, fx.Root)
	selection := goorderedtransportadapter.Selection{
		Streams:      []goorderedtransportadapter.OwnedIdentity{{Identity: "primary", Package: "example.test/go-uab-11/service"}},
		CommandKinds: []goorderedtransportadapter.KindSelection{{Identity: "configuration.initialize.v1", Owner: "example.test/go-uab-11/configuration", Payload: goorderedtransportadapter.Named{Package: "example.test/go-uab-11/configuration", Name: "Input"}}},
		EventKinds:   []goorderedtransportadapter.KindSelection{{Identity: "configuration.initialized.v1", Owner: "example.test/go-uab-11/configuration", Payload: goorderedtransportadapter.Named{Package: "example.test/go-uab-11/configuration", Name: "Initialized"}}},
		Ports:        []goorderedtransportadapter.OwnedIdentity{{Identity: "host.frames.v1", Package: "example.test/go-uab-11/service"}},
		Dispatch:     goorderedtransportadapter.Named{Package: "example.test/go-uab-11/service", Name: "Assemble"},
		Replay:       goorderedtransportadapter.Named{Package: "example.test/go-uab-11/service", Name: "Initialize"},
	}
	model, err := goorderedtransportadapter.Resolve(fx.Result.Project, selection)
	if err != nil {
		t.Fatal(err)
	}
	in := orderedtransportinstance.Inputs{Contracts: contracts, ProjectV10: fx.Result.Project, Model: model}
	first, err := orderedtransportinstance.Emit(in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := orderedtransportinstance.Emit(in)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("nondeterministic: %v", err)
	}
	in.Artifact = first
	if err = orderedtransportinstance.Validate(in); err != nil {
		t.Fatal(err)
	}
	p := projectv11instance.Inputs{Contracts: contracts, ProjectV10: fx.Result.Project, Transport: in}
	p.Composed, err = projectv11instance.Emit(p)
	if err != nil {
		t.Fatal(err)
	}
	if err = projectv11instance.Validate(p); err != nil {
		t.Fatal(err)
	}
	e, err := wire.Decode(p.Composed)
	if err != nil {
		t.Fatal(err)
	}
	if countSchema(e, testID("e026")) != 1 || countSchema(e, testID("10100")) != 1 || countSchema(e, testID("10101")) != 1 || countSchema(e, testID("1010f")) != 1 {
		t.Fatal("selected transport roots missing")
	}
	if authority, ok := oneSchema(e, testID("1010e")); !ok || authority.Fields[testID("110ff")].Tag != 3 || authority.Fields[testID("110ff")].Unsigned != 4096 {
		t.Fatal("retained payload authority missing")
	}

	reversed := selection
	reversed.Streams = append([]goorderedtransportadapter.OwnedIdentity(nil), selection.Streams...)
	reversed.CommandKinds = append([]goorderedtransportadapter.KindSelection(nil), selection.CommandKinds...)
	reversed.EventKinds = append([]goorderedtransportadapter.KindSelection(nil), selection.EventKinds...)
	reversed.Ports = append([]goorderedtransportadapter.OwnedIdentity(nil), selection.Ports...)
	model2, err := goorderedtransportadapter.Resolve(fx.Result.Project, reversed)
	if err != nil {
		t.Fatal(err)
	}
	in.Model = model2
	in.Artifact = nil
	again, err := orderedtransportinstance.Emit(in)
	if err != nil || !bytes.Equal(first, again) {
		t.Fatalf("selection order changed artifact: %v", err)
	}

	t.Run("unknown-owner", func(t *testing.T) {
		bad := selection
		bad.Streams = []goorderedtransportadapter.OwnedIdentity{{Identity: "primary", Package: "invalid.example/missing"}}
		if _, e := goorderedtransportadapter.Resolve(fx.Result.Project, bad); e == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("wrong-payload-owner", func(t *testing.T) {
		bad := selection
		bad.CommandKinds = []goorderedtransportadapter.KindSelection{{Identity: "x", Owner: "example.test/go-uab-11/service", Payload: selection.CommandKinds[0].Payload}}
		m, e := goorderedtransportadapter.Resolve(fx.Result.Project, bad)
		if e == nil {
			q := in
			q.Model = m
			if _, e = orderedtransportinstance.Emit(q); e == nil {
				t.Fatal("accepted")
			}
		}
	})
	t.Run("same-functions", func(t *testing.T) {
		bad := selection
		bad.Replay = bad.Dispatch
		m, e := goorderedtransportadapter.Resolve(fx.Result.Project, bad)
		if e != nil {
			t.Fatal(e)
		}
		q := in
		q.Model = m
		if _, e = orderedtransportinstance.Emit(q); e == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("artifact-tamper", func(t *testing.T) {
		bad := in
		bad.Artifact = append([]byte(nil), first...)
		bad.Artifact[len(bad.Artifact)-1] ^= 1
		if orderedtransportinstance.Validate(bad) == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("mixed-project", func(t *testing.T) {
		bad := p
		bad.Transport.ProjectV10.Composed = append([]byte(nil), bad.Transport.ProjectV10.Composed...)
		bad.Transport.ProjectV10.Composed[len(bad.Transport.ProjectV10.Composed)-1] ^= 1
		if _, e := projectv11instance.Emit(bad); e == nil {
			t.Fatal("accepted")
		}
	})
}

func resolveV11(t *testing.T, root string) contractcatalog.ProjectContractSetV11 {
	t.Helper()
	read := func(path string) []byte {
		b, e := os.ReadFile(filepath.Join(root, path))
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	s, e := contractcatalog.ResolveProjectContractSetV11(read("modules/foundation/v1/module.seme"), read("modules/execution/v36/module.seme"), read("modules/package/v4/module.seme"), read("modules/dependency/v1/module.seme"), read("modules/configuration/v3/module.seme"), read("modules/resource/v1/module.seme"), read("modules/durable-state/v1/module.seme"), read("modules/source-presentation/v1/module.seme"), read("modules/ordered-transport/v1/module.seme"), read("modules/project/v9/module.seme"), read("modules/project/v10/module.seme"), read("modules/project/v11/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func countSchema(e wire.Envelope, s wire.ID) int {
	n := 0
	for _, q := range e.Entities {
		if q.Schema == s {
			n++
		}
	}
	return n
}
func oneSchema(e wire.Envelope, s wire.ID) (wire.Entity, bool) {
	var found wire.Entity
	count := 0
	for _, q := range e.Entities {
		if q.Schema == s {
			found = q
			count++
		}
	}
	return found, count == 1
}
func testID(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
