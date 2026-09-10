package gopackageadapter

import (
	"crypto/sha256"
	"seme.local/reference/goprovider"
	"seme.local/reference/packagedetail"
	"testing"
)

func fixture() (goprovider.ResolutionManifest, []goprovider.PackageMetadata, Evidence) {
	l := goprovider.ProjectLocation{File: "a.go", Line: 1, Column: 1, ByteStart: 0, ByteEnd: 1, EndLine: 1, EndColumn: 2}
	r := goprovider.ResolutionManifest{Packages: []goprovider.ResolvedPackage{{Name: "app", Root: true, Files: []string{"a.go"}, Declarations: []goprovider.ResolvedDeclaration{{ID: "fn", Name: "Run", Exported: true, Location: l}}}}}
	m := []goprovider.PackageMetadata{{Name: "app", Root: true, Members: []goprovider.PackageFunctionMetadata{{ID: "fn", Name: "Run", Result: "i64", Exported: true, Document: "a.go", Line: 1, Column: 1}}}}
	d := sha256.Sum256([]byte("x"))
	s := packagedetail.Source{Identity: "source", Path: "a.go", ContentDigest: d, ByteSize: 1}
	p := Evidence{Sources: map[string]packagedetail.Source{"a.go": s}, Origins: map[Key]packagedetail.Origin{{"a.go", 1, 1}: {SourceIdentity: "source", Path: "a.go", ContentDigest: d, ByteStart: 0, ByteEnd: 1, StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 2}}}
	return r, m, p
}
func TestConvertExactCrossCheck(t *testing.T) {
	r, m, p := fixture()
	g, e := Convert(r, m, p)
	if e != nil {
		t.Fatal(e)
	}
	if g.Packages[0].Members[0].Visibility != packagedetail.Public {
		t.Fatal("visibility")
	}
	m[0].Members[0].Name = "bad"
	if _, e = Convert(r, m, p); e == nil {
		t.Fatal("mismatch accepted")
	}
}
func TestConvertRequiresFullProvenanceAndUniqueIdentities(t *testing.T) {
	r, m, p := fixture()
	if _, e := Convert(r, m, Evidence{}); e == nil {
		t.Fatal("missing provenance accepted")
	}
	r, m, p = fixture()
	p.Sources["extra.go"] = packagedetail.Source{Identity: "extra", Path: "extra.go"}
	if _, e := Convert(r, m, p); e == nil {
		t.Fatal("extra source accepted")
	}
	r, m, p = fixture()
	o := p.Origins[Key{"a.go", 1, 1}]
	o.ContentDigest = [32]byte{}
	p.Origins[Key{"a.go", 1, 1}] = o
	if _, e := Convert(r, m, p); e == nil {
		t.Fatal("origin digest mismatch accepted")
	}
	r.Packages = append(r.Packages, r.Packages[0])
	if _, e := Convert(r, m, p); e == nil {
		t.Fatal("duplicate package accepted")
	}
	r, m, p = fixture()
	r.Packages[0].Declarations = append(r.Packages[0].Declarations, r.Packages[0].Declarations[0])
	if _, e := Convert(r, m, p); e == nil {
		t.Fatal("duplicate declaration accepted")
	}
}
