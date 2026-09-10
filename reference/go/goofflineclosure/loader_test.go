package goofflineclosure

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"seme.local/reference/goprovider"
)

func TestLoadsCommittedUPB03OfflineClosure(t *testing.T) {
	in := fixture(t)
	a, err := Load(in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Load(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Requirements) != 2 || len(a.Entries) != 2 || a.Entries[1].Identity != "example.test/seme/checksum" || a.Entries[0].Source != "workspace:example.test/go-upb03-dependency-v1/model" || a.Entries[1].Source != "go-proxy:example.test/seme/checksum@v1.2.3" {
		t.Fatalf("closure=%#v", a)
	}
	if a.Entries[0].Digest != b.Entries[0].Digest || a.Entries[1].Integrity != b.Entries[1].Integrity {
		t.Fatal("nondeterministic")
	}
}

func TestRejectsSymlinkTraversalTamperAndUndeclared(t *testing.T) {
	base := fixture(t)
	t.Run("project-symlink", func(t *testing.T) {
		parent := t.TempDir()
		link := filepath.Join(parent, "project")
		if err := os.Symlink(base.ProjectRoot, link); err != nil {
			t.Fatal(err)
		}
		x := base
		x.ProjectRoot = link
		if _, err := Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("proxy-symlink", func(t *testing.T) {
		parent := t.TempDir()
		link := filepath.Join(parent, "proxy")
		if err := os.Symlink(base.ProxyRoot, link); err != nil {
			t.Fatal(err)
		}
		x := base
		x.ProxyRoot = link
		if _, err := Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("local-traversal", func(t *testing.T) {
		x := base
		x.Resolution.Packages[1].Files = []string{"../go.mod"}
		if _, err := Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("missing-edge", func(t *testing.T) {
		x := base
		x.Resolution.Packages[0].Imports = nil
		if _, err := Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("undeclared-coordinate", func(t *testing.T) {
		x := base
		x.Module = "example.test/seme/other"
		if _, err := Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("undeclared-mod", func(t *testing.T) {
		x := copyFixture(t, base)
		p := filepath.Join(x.ProjectRoot, "go.mod")
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		b = []byte(string(b) + "\nrequire example.test/seme/other v1.0.0\n")
		if err = os.WriteFile(p, b, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err = Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("sum-tamper", func(t *testing.T) {
		x := copyFixture(t, base)
		p := filepath.Join(x.ProjectRoot, "go.sum")
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		for i := range b {
			if b[i] == 'i' {
				b[i] = 'A'
				break
			}
		}
		if err = os.WriteFile(p, b, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err = Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("duplicate-file-owner", func(t *testing.T) {
		x := base
		x.Resolution.Packages[2].Files = []string{"model/value.go"}
		if _, err := Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("unlisted-local-file", func(t *testing.T) {
		x := copyFixture(t, base)
		if err := os.WriteFile(filepath.Join(x.ProjectRoot, "model/extra.go"), []byte("package model\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("zip-tamper", func(t *testing.T) {
		x := copyFixture(t, base)
		p := filepath.Join(x.ProxyRoot, "example.test/seme/checksum/@v/v1.2.3.zip")
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		b[len(b)/2] ^= 1
		if err = os.WriteFile(p, b, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err = Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("zip-traversal", func(t *testing.T) {
		x := copyFixture(t, base)
		p := filepath.Join(x.ProxyRoot, "example.test/seme/checksum/@v/v1.2.3.zip")
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		z := zip.NewWriter(f)
		w, err := z.Create("example.test/seme/checksum@v1.2.3/../evil")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
		if err = z.Close(); err != nil {
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err = Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("zip-symlink", func(t *testing.T) {
		x := copyFixture(t, base)
		p := filepath.Join(x.ProxyRoot, "example.test/seme/checksum/@v/v1.2.3.zip")
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(base.ProxyRoot, "example.test/seme/checksum/@v/v1.2.3.zip")
		if err := os.Symlink(target, p); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("local-symlink", func(t *testing.T) {
		x := copyFixture(t, base)
		p := filepath.Join(x.ProjectRoot, "model/value.go")
		os.Remove(p)
		if err := os.Symlink(filepath.Join(x.ProjectRoot, "go.mod"), p); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("local-directory-symlink", func(t *testing.T) {
		x := copyFixture(t, base)
		outside := t.TempDir()
		if err := os.WriteFile(filepath.Join(outside, "value.go"), []byte("package model\n"), 0600); err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(x.ProjectRoot, "model/value.go")
		if err := os.Remove(file); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Dir(file)
		if err := os.Remove(dir); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, dir); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(x); err == nil {
			t.Fatal("accepted")
		}
	})
}

func fixture(t *testing.T) Input {
	t.Helper()
	project, err := filepath.Abs("../../../fixtures/go-upb03-dependency-v1")
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := filepath.Abs("../../../fixtures/go-upb03-offline-proxy")
	if err != nil {
		t.Fatal(err)
	}
	app := "example.test/go-upb03-dependency-v1/application"
	model := "example.test/go-upb03-dependency-v1/model"
	policy := "example.test/go-upb03-dependency-v1/policy"
	manifest := goprovider.ResolutionManifest{Packages: []goprovider.ResolvedPackage{
		{Name: app, Root: true, Files: []string{"application/application.go"}, Imports: []goprovider.ResolvedImport{{Path: model, ResolvedPath: model, Local: true}, {Path: policy, ResolvedPath: policy, Local: true}}},
		{Name: model, Files: []string{"model/value.go"}},
		{Name: policy, Files: []string{"policy/score.go"}},
	}}
	return Input{ProjectRoot: project, ProxyRoot: proxy, Module: "example.test/seme/checksum", Version: "v1.2.3", LocalFrom: app, LocalTo: model, Resolution: manifest}
}
func copyFixture(t *testing.T, base Input) Input {
	t.Helper()
	root := t.TempDir()
	project := filepath.Join(root, "project")
	proxy := filepath.Join(root, "proxy")
	copyTree(t, base.ProjectRoot, project)
	copyTree(t, base.ProxyRoot, proxy)
	base.ProjectRoot, base.ProxyRoot = project, proxy
	return base
}
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		to := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(to, 0700)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(to, b, 0600)
	}); err != nil {
		t.Fatal(err)
	}
}
