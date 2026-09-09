package goprovider

import (
	"os"
	"reflect"
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
