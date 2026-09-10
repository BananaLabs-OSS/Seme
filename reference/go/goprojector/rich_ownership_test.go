package goprojector

import (
	"os"
	"path/filepath"
	"testing"

	"seme.local/reference/goprovider"
)

func TestRichOwnershipValidatesUAB11WithoutGuessingIDs(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v35/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	root := "../../../fixtures/go-uab-11"
	for _, name := range []string{"model/model.go", "policy/policy.go", "application/application.go"} {
		b, e := os.ReadFile(filepath.Join(root, name))
		if e != nil {
			t.Fatal(e)
		}
		files[name] = string(b)
	}
	s, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := s.Apply(goprovider.DocumentSnapshot{Revision: 1, ModulePath: "example.test/go-uab-11", PackagePath: "example.test/go-uab-11/application", Entry: "Apply", Files: files})
	if !result.Valid {
		t.Fatal(result.Diagnostics)
	}
	in := RichPackageOwnership{}
	for _, p := range result.Packages {
		in.Packages = append(in.Packages, RichPackage{Identity: p.Name, Name: filepath.Base(p.Name), Dependencies: append([]string(nil), p.Dependencies...)})
		for _, f := range p.Members {
			in.Declarations = append(in.Declarations, OwnedDeclaration{ID: f.ID, Package: p.Name, Name: f.Name, Kind: FunctionDeclaration, Exported: f.Exported})
		}
	}
	g, err := parse([]byte(result.CanonicalG1))
	if err != nil {
		t.Fatal(err)
	}
	for id, e := range g {
		kind := DeclarationKind("")
		name := ""
		owner := "example.test/go-uab-11/policy"
		exported := true
		switch e.schema {
		case sRecordType:
			kind = RecordDeclaration
			name, _ = text(e, "00000000000000000000000000009300")
			if name == "State" || name == "Command" {
				owner = "example.test/go-uab-11/application"
			}
		case sInterfaceType:
			kind = InterfaceDeclaration
			name, _ = text(e, "000000000000000000000000000a0100")
		case sMethod:
			kind = MethodDeclaration
			name, _ = text(e, "000000000000000000000000000a0020")
		default:
			continue
		}
		in.Declarations = append(in.Declarations, OwnedDeclaration{ID: id, Package: owner, Name: name, Kind: kind, Exported: exported})
	}
	in.Families = []OwnedFamily{{Kind: "result", Package: "example.test/go-uab-11/model", Name: "Result"}, {Kind: "transition", Package: "example.test/go-uab-11/model", Name: "Transition"}}
	NormalizeRichPackageOwnership(&in)
	if err = ValidateRichPackageOwnership([]byte(result.CanonicalG1), in); err != nil {
		t.Fatal(err)
	}
	missing := in
	missing.Declarations = append([]OwnedDeclaration(nil), in.Declarations[:len(in.Declarations)-1]...)
	if err = ValidateRichPackageOwnership([]byte(result.CanonicalG1), missing); err == nil {
		t.Fatal("missing ownership accepted")
	}
	wrong := in
	wrong.Declarations = append([]OwnedDeclaration(nil), in.Declarations...)
	for i := range wrong.Declarations {
		if wrong.Declarations[i].Kind == MethodDeclaration {
			wrong.Declarations[i].Package = "example.test/go-uab-11/application"
			break
		}
	}
	if err = ValidateRichPackageOwnership([]byte(result.CanonicalG1), wrong); err == nil {
		t.Fatal("method/receiver owner mismatch accepted")
	}
}
