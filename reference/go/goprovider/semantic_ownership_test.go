package goprovider

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func uab11Snapshot(t *testing.T, revision uint64, reverse bool) DocumentSnapshot {
	t.Helper()
	root := "../../../fixtures/go-uab-11"
	paths := []string{"application/application.go", "model/model.go", "policy/policy.go"}
	if reverse {
		paths[0], paths[2] = paths[2], paths[0]
	}
	files := map[string]string{}
	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		files[path] = string(content)
	}
	return DocumentSnapshot{Revision: revision, ModulePath: "example.test/go-uab-11", PackagePath: "example.test/go-uab-11/application", Entry: "Apply", Files: files}
}

func TestUAB11SemanticOwnershipComesFromTypedCanonicalWalk(t *testing.T) {
	module := resolutionModule(t)
	a, _ := NewIncrementalSession(module)
	first := a.Apply(uab11Snapshot(t, 1, false))
	if !first.Valid {
		t.Fatalf("diagnostics=%#v", first.Diagnostics)
	}
	b, _ := NewIncrementalSession(module)
	second := b.Apply(uab11Snapshot(t, 99, true))
	if !second.Valid || !reflect.DeepEqual(first.Packages, second.Packages) || !reflect.DeepEqual(first.Resolution, second.Resolution) {
		t.Fatal("semantic ownership depends on map order or client revision")
	}
	canonical := first.CanonicalG1
	kinds := map[SemanticDeclarationKind]int{}
	declarations := map[string]bool{}
	for _, pkg := range first.Packages {
		var resolved []SemanticDeclarationMetadata
		for _, rp := range first.Resolution.Packages {
			if rp.Name == pkg.Name {
				resolved = rp.Supplemental
			}
		}
		if !reflect.DeepEqual(pkg.Supplemental, resolved) {
			t.Fatalf("resolution/session ownership drift for %s", pkg.Name)
		}
		prior := ""
		for _, item := range pkg.Supplemental {
			if item.Package != pkg.Name || item.Declaration <= prior || declarations[item.Declaration] || !strings.Contains(canonical, item.Declaration) || item.Origin.File == "" || item.Origin.Line == 0 || item.Origin.ByteEnd <= item.Origin.ByteStart {
				t.Fatalf("invalid ownership %#v", item)
			}
			prior = item.Declaration
			declarations[item.Declaration] = true
			kinds[item.Kind]++
			if item.Kind == SemanticGenericRealization && item.GenericDefinition != "" {
				t.Fatal("invented a generic definition absent from canonical Execution")
			}
		}
	}
	for _, kind := range []SemanticDeclarationKind{SemanticRecord, SemanticInterface, SemanticMethod, SemanticGenericRealization} {
		if kinds[kind] == 0 {
			t.Fatalf("UAB11 did not exercise %s: %#v", kind, kinds)
		}
	}
	first.Packages[0].Supplemental[0].ReferencedImports = append(first.Packages[0].Supplemental[0].ReferencedImports, "forged")
	first.Resolution.Packages[0].Supplemental[0].Name = "forged"
	bad := uab11Snapshot(t, 2, false)
	bad.Files["application/application.go"] = "package application\nfunc Apply("
	retained := a.Apply(bad)
	if retained.Valid || reflect.DeepEqual(retained.Packages, first.Packages) || reflect.DeepEqual(retained.Resolution, first.Resolution) {
		t.Fatal("supplemental ownership aliases caller mutation")
	}
}

func TestSemanticOwnershipRejectsUnemittedIdentity(t *testing.T) {
	metadata := []PackageMetadata{{Name: "p"}}
	unit := &checkedSessionPackage{path: "p"}
	function := sessionFunction{id: "0123456789abcdef0123456789abcdef", packagePath: "p", method: true}
	diagnostic := attachSemanticOwnership(metadata, []*checkedSessionPackage{unit}, []sessionFunction{function}, nil)
	// An absent entity is omitted rather than guessed; this is the required
	// behavior for unsupported declarations and generic definitions.
	if diagnostic != nil || len(metadata[0].Supplemental) != 0 {
		t.Fatalf("unemitted declaration was published: %#v %#v", diagnostic, metadata)
	}
}

func TestSemanticOwnershipRetainsPrivateRecordAndMethodVisibility(t *testing.T) {
	session, _ := NewIncrementalSession(resolutionModule(t))
	result := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/private", PackagePath: "example.test/private", Entry: "Apply", Files: map[string]string{"dep/dep.go": `package dep
func Inc(v int64) int64 { return v + 1 }
`, "private.go": `package private
import helper "example.test/private/dep"
type counter struct { Value int64 }
func (counter) add(v int64) int64 { return helper.Inc(v) }
func Apply(v int64) int64 { return counter{Value: v}.add(v) }
`}})
	if !result.Valid {
		t.Fatalf("diagnostics=%#v", result.Diagnostics)
	}
	want := map[string]SemanticDeclarationKind{"counter": SemanticRecord, "add": SemanticMethod}
	for _, item := range result.Packages[0].Supplemental {
		if kind, ok := want[item.Name]; ok {
			if item.Kind != kind || item.Exported {
				t.Fatalf("private visibility lost: %#v", item)
			}
			delete(want, item.Name)
			if item.Name == "add" {
				if len(item.ImportReferences) != 1 || item.ImportReferences[0].Alias != "helper" || item.ImportReferences[0].Requested != "example.test/private/dep" || item.ImportReferences[0].Resolved != "example.test/private/dep" || !item.ImportReferences[0].Local || item.ImportReferences[0].Location.File != "private.go" {
					t.Fatalf("exact typed import reference lost: %#v", item.ImportReferences)
				}
			}
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing private declarations: %#v", want)
	}
}
