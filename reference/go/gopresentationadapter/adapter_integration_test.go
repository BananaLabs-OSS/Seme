package gopresentationadapter_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"seme.local/reference/gopresentationadapter"
	"seme.local/reference/goprovider"
	"seme.local/reference/internal/upb07testfixture"
	"seme.local/reference/presentationinstance"
	"seme.local/reference/wire"
)

func TestResolveAuthenticatedPresentation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fx, err := upb07testfixture.Build(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	in := fx.Result.Presentation
	packages, err := metadata(in)
	if err != nil {
		t.Fatal(err)
	}
	model, err := gopresentationadapter.Resolve(in.ProjectV9, packages)
	if err != nil {
		t.Fatal(err)
	}
	q := in
	q.Model = model
	q.Artifact = nil
	got, err := presentationinstance.Emit(q)
	if err != nil || !bytes.Equal(got, in.Artifact) {
		t.Fatalf("adapter did not reproduce authenticated presentation: %v", err)
	}

	reject := func(name string, mutate func([]goprovider.PackageMetadata)) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			p := clonePackages(packages)
			mutate(p)
			if _, resolveErr := gopresentationadapter.Resolve(in.ProjectV9, p); resolveErr == nil {
				t.Fatalf("accepted %s", name)
			}
		})
	}
	reject("duplicate-alias", func(p []goprovider.PackageMetadata) {
		p[0].Aliases = append(p[0].Aliases, p[0].Aliases[0])
	})
	reject("wrong-owner-name", func(p []goprovider.PackageMetadata) { p[0].Aliases[0].Package = "foreign.invalid/package" })
	reject("unknown-package", func(p []goprovider.PackageMetadata) { p[0].Name = "foreign.invalid/package" })
	reject("generic", func(p []goprovider.PackageMetadata) { p[0].Aliases[0].Generic = true })
	reject("unknown-target", func(p []goprovider.PackageMetadata) { p[0].Aliases[0].Target = (wire.ID{}).String() })
	reject("malformed-target", func(p []goprovider.PackageMetadata) { p[0].Aliases[0].Target = "not-an-id" })
	reject("unknown-source", func(p []goprovider.PackageMetadata) { p[0].Aliases[0].Document = "missing.go" })
	reject("missing-import", func(p []goprovider.PackageMetadata) {
		p[0].Aliases[0].ReferencedImports = []string{"foreign.invalid/import"}
	})
	withImport := packageAliasWithImport(t, packages)
	reject("duplicate-import", func(p []goprovider.PackageMetadata) {
		a := &p[withImport[0]].Aliases[withImport[1]]
		a.ReferencedImports = append(a.ReferencedImports, a.ReferencedImports[0])
	})

	// Resolve is deliberately insensitive to producer iteration order; the
	// neutral instance is the canonical ordering authority.
	reversed := clonePackages(packages)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	for i := range reversed {
		for a, b := 0, len(reversed[i].Aliases)-1; a < b; a, b = a+1, b-1 {
			reversed[i].Aliases[a], reversed[i].Aliases[b] = reversed[i].Aliases[b], reversed[i].Aliases[a]
		}
	}
	reordered, err := gopresentationadapter.Resolve(in.ProjectV9, reversed)
	if err != nil {
		t.Fatal(err)
	}
	q.Model = reordered
	got, err = presentationinstance.Emit(q)
	if err != nil || !bytes.Equal(got, in.Artifact) {
		t.Fatalf("producer order changed canonical presentation: %v", err)
	}
}

func packageAliasWithImport(t *testing.T, packages []goprovider.PackageMetadata) [2]int {
	t.Helper()
	for i := range packages {
		for j := range packages[i].Aliases {
			if len(packages[i].Aliases[j].ReferencedImports) != 0 {
				return [2]int{i, j}
			}
		}
	}
	t.Fatal("authenticated fixture contains no imported alias")
	return [2]int{}
}

func metadata(in presentationinstance.Inputs) ([]goprovider.PackageMetadata, error) {
	e, err := wire.Decode(in.ProjectV9.Composed)
	if err != nil {
		return nil, err
	}
	ownerNames := map[wire.ID]string{}
	for x, q := range e.Entities {
		if q.Schema == testID("b010") {
			ownerNames[x] = string(q.Fields[testID("b100")].Bytes)
		}
	}
	byName := map[string]int{}
	var out []goprovider.PackageMetadata
	for _, alias := range in.Model.Aliases {
		owner := ownerNames[alias.Owner]
		index, ok := byName[owner]
		if !ok {
			out = append(out, goprovider.PackageMetadata{Name: owner})
			index = len(out) - 1
			byName[owner] = index
		}
		refs := make([]string, 0, len(alias.ImportBindings))
		for _, bindingID := range alias.ImportBindings {
			binding := e.Entities[bindingID]
			refs = append(refs, string(binding.Fields[testID("b241")].Bytes))
		}
		out[index].Aliases = append(out[index].Aliases, goprovider.SourceAliasMetadata{
			Name: alias.Name, Package: owner, Target: alias.Target.String(), Document: alias.Path,
			Exported: alias.Visibility == 1, Start: int(alias.Start), End: int(alias.End),
			Line: int(alias.StartLine), Column: int(alias.StartColumn), EndLine: int(alias.EndLine), EndColumn: int(alias.EndColumn),
			ReferencedImports: refs,
		})
	}
	return out, nil
}

func clonePackages(in []goprovider.PackageMetadata) []goprovider.PackageMetadata {
	out := append([]goprovider.PackageMetadata(nil), in...)
	for i := range out {
		out[i].Aliases = append([]goprovider.SourceAliasMetadata(nil), out[i].Aliases...)
		for j := range out[i].Aliases {
			out[i].Aliases[j].ReferencedImports = append([]string(nil), out[i].Aliases[j].ReferencedImports...)
		}
	}
	return out
}

func testID(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, err := wire.ParseID(s)
	if err != nil {
		panic(err)
	}
	return x
}
