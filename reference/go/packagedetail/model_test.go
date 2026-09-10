package packagedetail

import (
	"crypto/sha256"
	"testing"
)

func fixture() Graph {
	d := sha256.Sum256([]byte("source"))
	o := Origin{"src", "a.go", d, 0, 3, 1, 1, 1, 4}
	return Graph{[]Detail{{Identity: "app", Root: true, Sources: []Source{{"src", "a.go", d, 3}}, Members: []Member{{Identity: "fn", Name: "Run", ExportName: "Run", Visibility: Public, Origin: o, Callable: true, Results: []string{"i64"}}}}}}
}
func TestRevisionCommitsContent(t *testing.T) {
	g := fixture()
	a, e := Revision(g)
	if e != nil {
		t.Fatal(e)
	}
	b, e := Revision(g)
	if e != nil || a != b {
		t.Fatal("unstable")
	}
	g.Packages[0].Members[0].Name = "Other"
	c, e := Revision(g)
	if e != nil || a == c {
		t.Fatal("mutation ignored")
	}
}
func TestValidationRejectsOriginVisibilityAndOrdering(t *testing.T) {
	cases := []Graph{fixture(), fixture(), fixture(), fixture()}
	cases[0].Packages[0].Members[0].Origin.ContentDigest = [32]byte{}
	cases[1].Packages[0].Members[0].Visibility = Package
	cases[2].Packages = append(cases[2].Packages, Detail{Identity: "aaa"})
	cases[3].Packages[0].Members[0].Origin.ByteEnd = 0
	for i, g := range cases {
		if Validate(g) == nil {
			t.Fatalf("case %d", i)
		}
	}
}

func TestOwnershipAndNamespaceAdversaries(t *testing.T) {
	t.Run("source without origin", func(t *testing.T) {
		g := fixture()
		g.Packages[0].Sources = append(g.Packages[0].Sources, Source{Identity: "unused", Path: "unused.go", ContentDigest: sha256.Sum256([]byte("unused")), ByteSize: 6})
		if err := Validate(g); err == nil || err.Error() != "package_detail.source_unowned" {
			t.Fatalf("accepted unowned package source: %v", err)
		}
	})
	t.Run("source identity ownership", func(t *testing.T) {
		g := fixture()
		g.Packages = append(g.Packages, Detail{Identity: "lib", Sources: []Source{{Identity: "src", Path: "lib.go", ContentDigest: sha256.Sum256([]byte("lib")), ByteSize: 1}}})
		if Validate(g) == nil {
			t.Fatal("shared source identity accepted")
		}
	})
	t.Run("source path ownership", func(t *testing.T) {
		g := fixture()
		g.Packages = append(g.Packages, Detail{Identity: "lib", Sources: []Source{{Identity: "other", Path: "a.go", ContentDigest: sha256.Sum256([]byte("lib")), ByteSize: 1}}})
		if Validate(g) == nil {
			t.Fatal("shared source path accepted")
		}
	})
	t.Run("member and export names", func(t *testing.T) {
		g := fixture()
		m := g.Packages[0].Members[0]
		m.Identity = "fn2"
		g.Packages[0].Members = append(g.Packages[0].Members, m)
		if Validate(g) == nil {
			t.Fatal("duplicate member name accepted")
		}
		g = fixture()
		m = g.Packages[0].Members[0]
		m.Identity, m.Name = "fn2", "Other"
		g.Packages[0].Members = append(g.Packages[0].Members, m)
		if Validate(g) == nil {
			t.Fatal("duplicate export name accepted")
		}
	})
	t.Run("visibility", func(t *testing.T) {
		g := fixture()
		g.Packages[0].Members[0].Visibility = Project
		g.Packages[0].Members[0].ExportName = ""
		if Validate(g) == nil {
			t.Fatal("project member without export accepted")
		}
	})
	t.Run("aliases and self import", func(t *testing.T) {
		g := fixture()
		o := g.Packages[0].Members[0].Origin
		g.Packages[0].Imports = []Import{{Alias: "x", Requested: "app", Resolved: "app", Class: Local, Origin: o}}
		if Validate(g) == nil {
			t.Fatal("self import accepted")
		}
		g = fixture()
		g.Packages[0].Imports = []Import{{Alias: "x", Requested: "a", Resolved: "a", Class: External, Origin: o}, {Alias: "x", Requested: "b", Resolved: "b", Class: External, Origin: o}}
		g.Packages[0].Imports[1].Origin.ByteStart = 1
		g.Packages[0].Imports[1].Origin.ByteEnd = 2
		g.Packages[0].Imports[1].Origin.StartColumn = 2
		g.Packages[0].Imports[1].Origin.EndColumn = 3
		if Validate(g) == nil {
			t.Fatal("duplicate import alias accepted")
		}
	})
	t.Run("source byte bound", func(t *testing.T) {
		g := fixture()
		g.Packages[0].Members[0].Origin.ByteEnd = 4
		if Validate(g) == nil {
			t.Fatal("out-of-range origin accepted")
		}
	})
}
