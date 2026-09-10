package goprojectpipeline

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/gopackagev3adapter"
	"seme.local/reference/goprovider"
	"seme.local/reference/packagev3instance"
	"seme.local/reference/projectsource"
	"seme.local/reference/wire"
)

func TestUAB11BuildsCompletePackageV3Evidence(t *testing.T) {
	in := uab11PipelineInput(t, 1, false)
	first, err := Build(t.Context(), in)
	if err != nil {
		t.Fatal(err)
	}
	declarations, err := gopackagev3adapter.Convert(first.Resolution, first.Packages, first.PackageV2)
	if err != nil {
		t.Fatal(err)
	}
	contracts := v5Contracts(t)
	complete, err := packagev3instance.Emit(packagev3instance.Inputs{Contracts: contracts, PackageV2: first.PackageV2, Declarations: declarations})
	if err != nil {
		t.Fatal(err)
	}
	if err = packagev3instance.Validate(contracts, first.PackageV2, complete); err != nil {
		t.Fatal(err)
	}
	if !packageOwnsPath(t, first.PackageV2, "example.test/go-uab-11/model", "model/model.go") {
		t.Fatal("generic-only model source is not owned by Package v2")
	}
	option := false
	for _, d := range declarations {
		if d.Kind == packagev3instance.GenericRealization && d.Name == d.Identity {
			q, _ := wire.Decode(first.PackageV2)
			id, _ := wire.ParseID(d.Identity)
			if q.Entities[id].Schema == mustID("a050") {
				option = true
			}
		}
	}
	if !option {
		t.Fatal("body-local Option realization is unowned")
	}

	otherIn := uab11PipelineInput(t, 99, true)
	other, err := Build(t.Context(), otherIn)
	if err != nil {
		t.Fatal(err)
	}
	otherDeclarations, err := gopackagev3adapter.Convert(other.Resolution, other.Packages, other.PackageV2)
	if err != nil {
		t.Fatal(err)
	}
	// Canonical artifact revisions intentionally include accepted editor
	// revision, while the typed ownership projection itself must not.
	if !reflect.DeepEqual(declarationShape(declarations), declarationShape(otherDeclarations)) {
		t.Fatal("ownership changed with file insertion/client revision")
	}

	forged := append([]goprovider.PackageMetadata(nil), first.Packages...)
	forged[0].Supplemental = append([]goprovider.SemanticDeclarationMetadata(nil), forged[0].Supplemental...)
	forged[0].Supplemental[0].Origin.ByteStart++
	if got, er := gopackagev3adapter.Convert(first.Resolution, forged, first.PackageV2); er == nil || got != nil {
		t.Fatal("tampered typed origin accepted or partial output returned")
	}
	first.Packages[0].Supplemental[0].Name = "mutated"
	if len(first.Packages[0].Supplemental[0].ImportReferences) > 0 {
		first.Packages[0].Supplemental[0].ImportReferences[0].Requested = "mutated"
	}
	again, er := Build(t.Context(), in)
	if er != nil || again.Packages[0].Supplemental[0].Name == "mutated" {
		t.Fatal("pipeline PackageMetadata aliases a prior result")
	}
}

func uab11PipelineInput(t *testing.T, revision uint64, reverse bool) Input {
	t.Helper()
	in := fixture(t)
	root := t.TempDir()
	paths := []string{"application/application.go", "model/model.go", "policy/policy.go"}
	if reverse {
		paths[0], paths[2] = paths[2], paths[0]
	}
	files := map[string]string{}
	for _, name := range paths {
		raw, err := os.ReadFile(filepath.Join("../../../fixtures/go-uab-11", name))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = string(raw)
		target := filepath.Join(root, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(target, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	sources, err := projectsource.Discover(root, "example.test/go-uab-11", projectsource.Toolchain{Language: "go", Toolchain: "go1.26", Profile: "bounded-v1", SemanticRevision: "provider-v1"}, projectsource.Policy{TrackedExtensions: []string{".go"}, GeneratedHeader: []byte("// generated"), MaxFiles: 8, MaxFileBytes: 16384, MaxTotalBytes: 32768})
	if err != nil {
		t.Fatal(err)
	}
	in.Documents = goprovider.DocumentSnapshot{Revision: revision, ModulePath: "example.test/go-uab-11", PackagePath: "example.test/go-uab-11/application", Entry: "Apply", Files: files}
	in.Sources = sources
	return in
}
func v5Contracts(t *testing.T) contractcatalog.ProjectContractSetV5 {
	t.Helper()
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	c, e := contractcatalog.ResolveProjectContractSetV5(read("../../../modules/execution/v35/module.seme"), read("../../../modules/package/v3/module.seme"), read("../../../modules/dependency/v1/module.seme"), read("../../../modules/project/v5/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func packageOwnsPath(t *testing.T, raw []byte, name, path string) bool {
	t.Helper()
	e, er := wire.Decode(raw)
	if er != nil {
		t.Fatal(er)
	}
	for _, q := range e.Entities {
		if q.Schema != mustID("b021") {
			continue
		}
		p := e.Entities[q.Fields[mustID("b210")].Reference]
		if string(p.Fields[mustID("b100")].Bytes) != name {
			continue
		}
		for _, v := range q.Fields[mustID("b213")].List {
			if bytes.Equal(e.Entities[v.Reference].Fields[mustID("b261")].Bytes, []byte(path)) {
				return true
			}
		}
	}
	return false
}
func declarationShape(in []packagev3instance.Declaration) [][4]string {
	out := make([][4]string, len(in))
	for i, d := range in {
		out[i] = [4]string{d.Identity, d.Name, d.ExportName, d.Origin.Path}
	}
	return out
}
