package goprojector

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
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
	in.Families = []OwnedFamily{{Kind: "option", Package: "example.test/go-uab-11/model", Name: "Option"}, {Kind: "result", Package: "example.test/go-uab-11/model", Name: "Result"}, {Kind: "transition", Package: "example.test/go-uab-11/model", Name: "Transition"}}
	NormalizeRichPackageOwnership(&in)
	if err = ValidateRichPackageOwnership([]byte(result.CanonicalG1), in); err != nil {
		t.Fatal(err)
	}
	plans, err := planRichPackages([]byte(result.CanonicalG1), in)
	if err != nil {
		t.Fatal(err)
	}
	application := plans["example.test/go-uab-11/application"]
	policy := plans["example.test/go-uab-11/policy"]
	model := plans["example.test/go-uab-11/model"]
	if !reflect.DeepEqual(application.imports, []string{"example.test/go-uab-11/model", "example.test/go-uab-11/policy"}) || !reflect.DeepEqual(policy.imports, []string{"example.test/go-uab-11/model"}) || len(model.imports) != 0 {
		t.Fatalf("imports app=%v policy=%v model=%v", application.imports, policy.imports, model.imports)
	}
	if application.familyNames["result"] != "model.Result" || application.familyNames["transition"] != "model.Transition" || policy.familyNames["result"] != "model.Result" || model.familyNames["result"] != "Result" {
		t.Fatalf("families app=%v policy=%v model=%v", application.familyNames, policy.familyNames, model.familyNames)
	}
	projected, err := ProjectPackagesRich([]byte(result.CanonicalG1), in)
	if err != nil {
		t.Fatal(err)
	}
	if len(projected) != 3 {
		t.Fatalf("projected packages=%d", len(projected))
	}
	var offsetID string
	for _, d := range in.Declarations {
		if d.Package == "example.test/go-uab-11/policy" && d.Name == "Offset" {
			offsetID = d.ID
		}
	}
	withAlias, err := ProjectPackagesRichWithAliases([]byte(result.CanonicalG1), in, []AliasPresentation{{Package: "example.test/go-uab-11/application", Name: "PolicyOffset", Target: offsetID, Imports: []string{"example.test/go-uab-11/policy"}}})
	if err != nil || !strings.Contains(string(withAlias["example.test/go-uab-11/application"]), "type PolicyOffset = policy.Offset") {
		t.Fatalf("authenticated alias projection missing: %v\n%s", err, withAlias["example.test/go-uab-11/application"])
	}
	if output, err := ProjectPackagesRichWithAliases([]byte(result.CanonicalG1), in, []AliasPresentation{{Package: "example.test/go-uab-11/application", Name: "State", Target: offsetID, Imports: []string{"example.test/go-uab-11/policy"}}}); err == nil || output != nil {
		t.Fatal("alias/declaration collision accepted")
	}
	if output, err := ProjectPackagesRichWithAliases([]byte(result.CanonicalG1), in, []AliasPresentation{{Package: "example.test/go-uab-11/application", Name: "Forged", Target: offsetID, Imports: []string{"example.test/undeclared"}}}); err == nil || output != nil {
		t.Fatal("alias import mismatch accepted")
	}
	appSource, policySource, modelSource := string(projected["example.test/go-uab-11/application"]), string(projected["example.test/go-uab-11/policy"]), string(projected["example.test/go-uab-11/model"])
	if strings.Count(appSource, "type State struct") != 1 || strings.Contains(policySource, "type State struct") || strings.Contains(modelSource, "type State struct") {
		t.Fatal("State ownership flattened or duplicated")
	}
	if strings.Count(modelSource, "type Result[") != 1 || strings.Contains(appSource, "type Result[") || strings.Contains(policySource, "type Result[") {
		t.Fatal("Result family ownership flattened or duplicated")
	}
	if strings.Count(policySource, "func (self Offset) Adjust") != 1 || strings.Contains(appSource, "func (self Offset) Adjust") || strings.Contains(modelSource, "func (self Offset) Adjust") {
		t.Fatal("method ownership flattened or duplicated")
	}
	badFamily := in
	badFamily.Families = append([]OwnedFamily(nil), in.Families...)
	badFamily.Families[0].Name = "ForgedOption"
	if err = ValidateRichPackageOwnership([]byte(result.CanonicalG1), badFamily); err == nil {
		t.Fatal("arbitrary family name accepted")
	}
	badFamily = in
	badFamily.Families = append([]OwnedFamily(nil), in.Families...)
	badFamily.Families[0].Package = "example.test/missing"
	if err = ValidateRichPackageOwnership([]byte(result.CanonicalG1), badFamily); err == nil {
		t.Fatal("arbitrary family package accepted")
	}
	dir := t.TempDir()
	if err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test/go-uab-11\n\ngo 1.26\n"), 0644); err != nil {
		t.Fatal(err)
	}
	relift := map[string]string{}
	for packagePath, source := range projected {
		relative := strings.TrimPrefix(packagePath, "example.test/go-uab-11/")
		name := filepath.Join(relative, "projected.go")
		if err = os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0755); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name), source, 0644); err != nil {
			t.Fatal(err)
		}
		relift[filepath.ToSlash(name)] = string(source)
	}
	testSource, err := os.ReadFile(filepath.Join(root, "application/application_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "application/application_test.go"), testSource, 0644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "-count=1", "./...")
	command.Dir = dir
	command.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "cache"))
	if output, e := command.CombinedOutput(); e != nil {
		t.Fatalf("native UAB11: %v\n%s", e, output)
	}
	second, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	again := second.Apply(goprovider.DocumentSnapshot{Revision: 1, ModulePath: "example.test/go-uab-11", PackagePath: "example.test/go-uab-11/application", Entry: "Apply", Files: relift})
	if !again.Valid {
		t.Fatal(again.Diagnostics)
	}
	if again.CanonicalG1 != result.CanonicalG1 {
		t.Fatal("rich projection relift differs")
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

func TestRichImportAliasesExpandCollisionsDeterministically(t *testing.T) {
	packages := []RichPackage{{Identity: "a", Name: "maps"}, {Identity: "b", Name: "maps"}, {Identity: "c", Name: "policy"}}
	a := richImportAliases(packages)
	b := richImportAliases([]RichPackage{packages[2], packages[1], packages[0]})
	if !reflect.DeepEqual(a, b) || a["a"] != "maps2" || a["b"] != "maps3" || a["c"] != "policy" {
		t.Fatalf("a=%v b=%v", a, b)
	}
}

func TestTypeNamesArePackageRelativeAcrossRichFamilies(t *testing.T) {
	g := map[string]entity{
		"state": {schema: sRecordType, fields: map[string][]string{"00000000000000000000000000009300": {"by 5 5374617465"}}},
		"i64":   {schema: sInteger},
		"transition": {schema: sTransitionType, fields: map[string][]string{
			"000000000000000000000000000a0040": {"rf state"}, "000000000000000000000000000a0041": {"rf i64"},
		}},
		"result": {schema: sResultType, fields: map[string][]string{
			"00000000000000000000000000009400": {"rf transition"}, "00000000000000000000000000009401": {"rf i64"},
		}},
	}
	name, err := typeNameRelative(g, "result", map[string]string{"state": "application.State"}, map[string]string{"result": "model.Result", "transition": "model.Transition"})
	if err != nil || name != "model.Result[model.Transition[application.State, int64], int64]" {
		t.Fatalf("name=%q err=%v", name, err)
	}
}

func TestGeneratedImportMayUseAuthenticatedDependencyClosure(t *testing.T) {
	packages := map[string]RichPackage{
		"application": {Identity: "application"},
		"persistence": {Identity: "persistence", Dependencies: []string{"state"}},
		"state":       {Identity: "state", Dependencies: []string{"application"}},
	}
	if !richDependencyReachable(packages, "persistence", "application") {
		t.Fatal("authenticated transitive dependency rejected")
	}
	if richDependencyReachable(packages, "application", "persistence") || richDependencyReachable(packages, "persistence", "forged") {
		t.Fatal("unreachable dependency accepted")
	}
}

func TestExternalNamedRecordStopsTransitiveImportTraversal(t *testing.T) {
	// persistence owns Persist, which names state.State. State in turn names
	// application.App. Go requires persistence to import state, not every
	// package used inside state.State's definition.
	const (
		appID     = "0000000000000000000000000000aa01"
		stateID   = "0000000000000000000000000000aa02"
		persistID = "0000000000000000000000000000aa03"
	)
	record := func(id, name string, refs ...string) string {
		var b strings.Builder
		fmt.Fprintf(&b, "en %s %s 1 2\nfi 00000000000000000000000000009300 by %x\nfi 00000000000000000000000000009301 li %d\n", id, sRecordType, name, len(refs))
		for _, ref := range refs {
			fmt.Fprintf(&b, "rf %s\n", ref)
		}
		return b.String()
	}
	g1 := []byte(record(appID, "App") + record(stateID, "State", appID) + record(persistID, "Persist", stateID))
	in := RichPackageOwnership{
		Packages: []RichPackage{
			{Identity: "example.test/application", Name: "application"},
			{Identity: "example.test/persistence", Name: "persistence", Dependencies: []string{"example.test/state"}},
			{Identity: "example.test/state", Name: "state", Dependencies: []string{"example.test/application"}},
		},
		Declarations: []OwnedDeclaration{
			{ID: appID, Package: "example.test/application", Name: "App", Kind: RecordDeclaration, Exported: true},
			{ID: stateID, Package: "example.test/state", Name: "State", Kind: RecordDeclaration, Exported: true},
			{ID: persistID, Package: "example.test/persistence", Name: "Persist", Kind: RecordDeclaration, Exported: true},
		},
	}
	plans, err := planRichPackages(g1, in)
	if err != nil {
		t.Fatal(err)
	}
	if got := plans["example.test/persistence"].imports; !reflect.DeepEqual(got, []string{"example.test/state"}) {
		t.Fatalf("persistence imports transitive implementation packages: %v", got)
	}
}
