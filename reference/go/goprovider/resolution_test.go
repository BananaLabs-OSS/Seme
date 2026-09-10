package goprovider

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func resolutionModule(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../../../modules/execution/v35/module.g1")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func resolutionSnapshot(revision uint64, reverse bool) DocumentSnapshot {
	items := [][2]string{{"model/value.go", `package model
func Apply(v int64) int64 { return v + 1 }
func hidden(v int64) int64 { return v }`}, {"app/main.go", `package app
import (
 "log"
 domain "example.test/resolution/model"
)
func Apply(v int64) int64 { return domain.Apply(v) }
func logValue(v int64) int64 { log.Printf("value=%d", v); return v }`}}
	files := map[string]string{}
	if reverse {
		for i := len(items) - 1; i >= 0; i-- {
			files[items[i][0]] = items[i][1]
		}
	} else {
		for _, x := range items {
			files[x[0]] = x[1]
		}
	}
	return DocumentSnapshot{Revision: revision, ModulePath: "example.test/resolution", PackagePath: "example.test/resolution/app", Entry: "Apply", Files: files}
}
func TestResolutionManifestRecordsTypedOwnershipAndImportsDeterministically(t *testing.T) {
	a, _ := NewIncrementalSession(resolutionModule(t))
	first := a.Apply(resolutionSnapshot(1, false))
	if !first.Valid {
		t.Fatalf("%#v", first.Diagnostics)
	}
	b, _ := NewIncrementalSession(resolutionModule(t))
	second := b.Apply(resolutionSnapshot(99, true))
	if !second.Valid {
		t.Fatalf("%#v", second.Diagnostics)
	}
	if !reflect.DeepEqual(first.Resolution, second.Resolution) {
		t.Fatalf("manifest drift\n%#v\n%#v", first.Resolution, second.Resolution)
	}
	if len(first.Resolution.Packages) != 2 {
		t.Fatalf("%#v", first.Resolution)
	}
	byName := map[string]ResolvedPackage{}
	for _, p := range first.Resolution.Packages {
		byName[p.Name] = p
	}
	model := byName["example.test/resolution/model"]
	app := byName["example.test/resolution/app"]
	if !app.Root || model.Root || !reflect.DeepEqual(model.Files, []string{"model/value.go"}) || len(model.Declarations) != 2 {
		t.Fatalf("ownership=%#v", first.Resolution)
	}
	if len(app.Imports) != 2 {
		t.Fatalf("imports=%#v", app.Imports)
	}
	imports := map[string]ResolvedImport{}
	for _, item := range app.Imports {
		imports[item.Path] = item
	}
	localImport, standardImport := imports["example.test/resolution/model"], imports["log"]
	if localImport.Alias != "domain" || !localImport.Local || localImport.ResolvedPath != "example.test/resolution/model" || localImport.Location.File != "app/main.go" || localImport.Location.Line != 4 || standardImport.Alias != "log" || standardImport.Local || standardImport.ResolvedPath != "log" || standardImport.Location.Line != 3 {
		t.Fatalf("import=%#v", app.Imports)
	}
	if model.Declarations[0].Location.File != "model/value.go" || model.Declarations[0].Location.Line == 0 {
		t.Fatalf("declarations=%#v", model.Declarations)
	}
	first.Resolution.Packages[0].Files[0] = "mutated"
	bad := resolutionSnapshot(2, false)
	bad.Files["app/main.go"] = "package app\nfunc Apply("
	retained := a.Apply(bad)
	if retained.Valid || retained.Resolution.Packages[0].Files[0] == "mutated" {
		t.Fatal("manifest was not retained copy-safely")
	}
}
func TestResolutionRejectsDuplicateSemanticIdentityBeforeComposition(t *testing.T) {
	session, _ := NewIncrementalSession(resolutionModule(t))
	identity := "8123456789abcdef0123456789abcdef"
	result := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/duplicate", PackagePath: "example.test/duplicate/app", Entry: "Apply", Files: map[string]string{"dep/dep.go": "package dep\n// seme:id " + identity + "\nfunc Other(v int64) int64 { return v }", "app/app.go": "package app\nimport \"example.test/duplicate/dep\"\n// seme:id " + identity + "\nfunc Apply(v int64) int64 { return dep.Other(v) }"}})
	if result.Valid {
		t.Fatal("duplicate identity reached composition")
	}
	found := false
	for _, d := range result.Diagnostics {
		if d.Code == "go.duplicate_semantic_identity" && d.File != "" && d.Line > 0 && d.Column > 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("diagnostics=%#v", result.Diagnostics)
	}
}
func TestResolutionRejectsExternalNonStandardImportWithLocation(t *testing.T) {
	session, _ := NewIncrementalSession(resolutionModule(t))
	result := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/external", PackagePath: "example.test/external", Entry: "Apply", Files: map[string]string{"main.go": "package external\nimport foreign \"example.invalid/dependency\"\nfunc Apply(v int64) int64 { return foreign.Apply(v) }"}})
	if result.Valid {
		t.Fatal("external dependency accepted")
	}
	found := false
	for _, d := range result.Diagnostics {
		if d.Code == "go.external_import_unsupported" && d.File == "main.go" && d.Line == 2 && d.Column > 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("diagnostics=%#v", result.Diagnostics)
	}
}

func TestResolutionClassifiesOnlyClosedConsumedStandardImports(t *testing.T) {
	snapshot := DocumentSnapshot{Revision: 1, ModulePath: "example.test/consumed", PackagePath: "example.test/consumed", Entry: "Apply", Files: map[string]string{"main.go": `package consumed
import ("maps"; "slices"; "log")
func Apply(v int64) int64 { _ = maps.Clone(map[int64]int64{1:v}); _ = slices.Replace(slices.Clone([]int64{v}),0,1,v); log.Print(true); return v }
`}}
	r, diagnostics := resolveSnapshot(snapshot)
	if len(diagnostics) != 0 {
		t.Fatalf("%#v", diagnostics)
	}
	want := map[string]string{"maps": "go-consumed:maps:Clone", "slices": "go-consumed:slices:Clone,Replace", "log": "go-consumed:log:Print"}
	for _, im := range r.Packages[0].Imports {
		if !im.Consumed || im.Realization != want[im.Path] {
			t.Fatalf("not exactly classified: %#v", im)
		}
		delete(want, im.Path)
	}
	if len(want) != 0 {
		t.Fatalf("missing=%#v", want)
	}

	misclassified := snapshot
	misclassified.Revision = 2
	misclassified.Files = map[string]string{"main.go": "package consumed\nimport \"log\"\nfunc Apply(v int64) int64 { log.Printf(\"%d\",v); return v }\n"}
	r, diagnostics = resolveSnapshot(misclassified)
	if len(diagnostics) != 0 || len(r.Packages) != 1 || r.Packages[0].Imports[0].Consumed || r.Packages[0].Imports[0].Realization != "" {
		t.Fatalf("unsupported log operation erased: %#v %#v", r, diagnostics)
	}

	unused := snapshot
	unused.Revision = 3
	unused.Files = map[string]string{"main.go": "package consumed\nimport \"maps\"\nfunc Apply(v int64) int64 { return v }\n"}
	if r, d := resolveSnapshot(unused); len(d) == 0 || len(r.Packages) != 0 {
		t.Fatalf("unused stdlib import accepted: %#v %#v", r, d)
	}
}
func TestResolutionPreservesInaccessibleDeclarationLocation(t *testing.T) {
	session, _ := NewIncrementalSession(resolutionModule(t))
	result := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/access", PackagePath: "example.test/access/app", Entry: "Apply", Files: map[string]string{"model/value.go": "package model\nfunc hidden(v int64) int64 { return v }", "app/main.go": "package app\nimport \"example.test/access/model\"\nfunc Apply(v int64) int64 { return model.hidden(v) }"}})
	if result.Valid {
		t.Fatal("inaccessible declaration accepted")
	}
	found := false
	for _, d := range result.Diagnostics {
		if d.Code == "go.type" && d.File == "app/main.go" && d.Line == 3 && d.Column > 0 {
			found = true
		}
	}
	if !found {
		t.Fatalf("diagnostics=%#v", result.Diagnostics)
	}
}

func TestResolutionStructuralImportFailuresCarryLocations(t *testing.T) {
	tests := []struct {
		name, code, file string
		line             int
		files            map[string]string
	}{
		{"missing-local-import", "go.local_import_missing", "app/main.go", 2, map[string]string{"app/main.go": "package app\nimport \"example.test/located/missing\"\nfunc Apply(v int64) int64 { return missing.Apply(v) }\n"}},
		{"package-clause-mismatch", "go.type", "app/b.go", 1, map[string]string{"app/a.go": "package app\nfunc Apply(v int64) int64 { return v }\n", "app/b.go": "package other\nfunc Other(v int64) int64 { return v }\n"}},
		{"import-cycle", "go.import_cycle", "dep/dep.go", 2, map[string]string{"app/app.go": "package app\nimport \"example.test/located/dep\"\nfunc Apply(v int64) int64 { return dep.Apply(v) }\n", "dep/dep.go": "package dep\nimport \"example.test/located/app\"\nfunc Apply(v int64) int64 { return app.Apply(v) }\n"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session, _ := NewIncrementalSession(resolutionModule(t))
			result := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/located", PackagePath: "example.test/located/app", Entry: "Apply", Files: test.files})
			if result.Valid || result.CanonicalG1 != "" || len(result.Resolution.Packages) != 0 {
				t.Fatalf("invalid structure published state: %#v", result)
			}
			found := false
			for _, d := range result.Diagnostics {
				if d.Code == test.code && d.File == test.file && d.Line == test.line && d.Column > 0 {
					found = true
				}
			}
			if !found {
				t.Fatalf("diagnostics=%#v", result.Diagnostics)
			}
		})
	}
}

func TestResolutionDiagnosticsAreDeterministicAndOrdered(t *testing.T) {
	makeSnapshot := func(revision uint64, reverse bool) DocumentSnapshot {
		files := map[string]string{}
		items := [][2]string{{"app/z.go", "package app\nfunc Other(v int64) int64 { return missingZ + v }\n"}, {"app/a.go", "package app\nfunc Apply(v int64) int64 { return missingA + v }\n"}}
		if reverse {
			items[0], items[1] = items[1], items[0]
		}
		for _, item := range items {
			files[item[0]] = item[1]
		}
		return DocumentSnapshot{Revision: revision, ModulePath: "example.test/diagnostics", PackagePath: "example.test/diagnostics/app", Entry: "Apply", Files: files}
	}
	a, _ := NewIncrementalSession(resolutionModule(t))
	b, _ := NewIncrementalSession(resolutionModule(t))
	left, right := a.Apply(makeSnapshot(1, false)), b.Apply(makeSnapshot(99, true))
	if left.Valid || right.Valid || len(left.Diagnostics) < 2 || !reflect.DeepEqual(left.Diagnostics, right.Diagnostics) {
		t.Fatalf("diagnostics differ:\n%#v\n%#v", left.Diagnostics, right.Diagnostics)
	}
	for i := 1; i < len(left.Diagnostics); i++ {
		prior, next := left.Diagnostics[i-1], left.Diagnostics[i]
		if prior.File > next.File || (prior.File == next.File && prior.Line > next.Line) {
			t.Fatalf("diagnostics not ordered: %#v", left.Diagnostics)
		}
	}
}

func TestUnsupportedExternalImportRetainsLastValidAtomically(t *testing.T) {
	session, _ := NewIncrementalSession(resolutionModule(t))
	valid := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/atomic", PackagePath: "example.test/atomic", Entry: "Apply", Files: map[string]string{"main.go": "package atomic\nfunc Apply(v int64) int64 { return v }\n"}})
	if !valid.Valid {
		t.Fatal(valid.Diagnostics)
	}
	invalid := session.Apply(DocumentSnapshot{Revision: 2, ModulePath: "example.test/atomic", PackagePath: "example.test/atomic", Entry: "Apply", Files: map[string]string{"main.go": "package atomic\nimport foreign \"example.invalid/dependency\"\nfunc Apply(v int64) int64 { return foreign.Apply(v) }\n"}})
	if !invalid.Accepted || invalid.Valid || invalid.LastValidRevision != 1 || invalid.CanonicalG1 != valid.CanonicalG1 || !reflect.DeepEqual(invalid.Packages, valid.Packages) || !reflect.DeepEqual(invalid.Resolution, valid.Resolution) {
		t.Fatalf("external failure leaked partial state: %#v", invalid)
	}
	if len(invalid.Diagnostics) == 0 || invalid.Diagnostics[0].Code != "go.external_import_unsupported" || invalid.Diagnostics[0].File != "main.go" || invalid.Diagnostics[0].Line != 2 || !strings.Contains(invalid.Diagnostics[0].Message, "example.invalid/dependency") {
		t.Fatalf("external diagnostic=%#v", invalid.Diagnostics)
	}
}
