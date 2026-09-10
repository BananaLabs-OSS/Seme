package godependencyresolver

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"seme.local/reference/dependencyresolution"
)

func fixture() Input {
	lf := map[string][]byte{"local.go": []byte("package local")}
	ef := map[string][]byte{"go.mod": []byte("module example.test/ext")}
	ld, _, _ := tree(lf)
	ed, h, _ := tree(ef)
	return Input{GoMod: []byte("module example.test/app\nrequire example.test/ext v1.2.3\n"), GoSum: []byte("example.test/ext v1.2.3 " + h + "\n"), Local: Local{From: "example.test/app/root", To: "example.test/app/local", Source: "workspace/local", Digest: ld, Files: lf}, External: External{Module: "example.test/ext", Version: "v1.2.3", Source: "cache/ext@v1.2.3", Digest: ed, Files: ef}, PackageImports: map[string][]string{"example.test/app/root": {"example.test/app/local"}}}
}
func TestResolveOfflineDeterministic(t *testing.T) {
	a, e := Resolve(fixture())
	if e != nil {
		t.Fatal(e)
	}
	b, e := Resolve(fixture())
	if e != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("nondeterministic")
	}
	if !strings.HasPrefix(a.Entries[0].Version, "source:") {
		t.Fatal("local revision missing")
	}
}

func TestNeutralMetadataRejectsDuplicateAndUnsorted(t *testing.T) {
	closure, e := Resolve(fixture())
	if e != nil {
		t.Fatal(e)
	}
	for _, metadata := range [][]dependencyresolution.Metadata{{{Key: "z", Value: "1"}, {Key: "a", Value: "2"}}, {{Key: "a", Value: "1"}, {Key: "a", Value: "2"}}} {
		x := closure
		x.Entries = append([]dependencyresolution.Entry(nil), closure.Entries...)
		x.Entries[0].Metadata = metadata
		if dependencyresolution.Validate(x) == nil {
			t.Fatalf("accepted metadata %#v", metadata)
		}
	}
}

func TestNeutralRequirementsRejectMismatch(t *testing.T) {
	base, e := Resolve(fixture())
	if e != nil {
		t.Fatal(e)
	}
	cases := []func(*dependencyresolution.Closure){func(x *dependencyresolution.Closure) { x.Requirements = x.Requirements[:1] }, func(x *dependencyresolution.Closure) { x.Requirements[1].Identity = x.Requirements[0].Identity }, func(x *dependencyresolution.Closure) {
		x.Requirements[0], x.Requirements[1] = x.Requirements[1], x.Requirements[0]
	}, func(x *dependencyresolution.Closure) { x.Requirements[0].Kind = dependencyresolution.External }, func(x *dependencyresolution.Closure) { x.Requirements[0].Identity = "dangling" }}
	for i, f := range cases {
		x := base
		x.Requirements = append([]dependencyresolution.Requirement(nil), base.Requirements...)
		f(&x)
		if dependencyresolution.Validate(x) == nil {
			t.Fatalf("accepted case %d: %#v", i, x.Requirements)
		}
	}
}
func TestRejectsForgery(t *testing.T) {
	cases := map[string]func(*Input){"replace": func(x *Input) { x.GoMod = append(x.GoMod, []byte("replace example.test/ext => ../x\n")...) }, "floating": func(x *Input) {
		x.GoMod = []byte("module example.test/app\nrequire example.test/ext latest\n")
		x.External.Version = "latest"
	}, "undeclared": func(x *Input) { x.External.Module = "other" }, "sum": func(x *Input) { x.GoSum = []byte("example.test/ext v1.2.3 h1:bad\n") }, "external-content": func(x *Input) { x.External.Files["go.mod"] = []byte("changed") }, "local-import": func(x *Input) { x.PackageImports = nil }, "local-content": func(x *Input) { x.Local.Files["local.go"] = []byte("changed") }}
	for n, f := range cases {
		t.Run(n, func(t *testing.T) {
			x := fixture()
			f(&x)
			out, e := Resolve(x)
			if e == nil || len(out.Entries) != 0 {
				t.Fatalf("out=%#v err=%v", out, e)
			}
		})
	}
}

func TestPinnedStableSemverAndTreePaths(t *testing.T) {
	for _, v := range []string{"v0.0.0", "v1.2.3", "v10.20.30"} {
		if !pinned(v) {
			t.Fatalf("rejected %s", v)
		}
	}
	for _, v := range []string{"v1.2.x", "latest", "master", ">=v1.2.3", "v01.2.3", "v1.02.3", "v1.2.03", "v1.2", "v1.2.3-rc.1"} {
		if pinned(v) {
			t.Fatalf("accepted %s", v)
		}
	}
	for _, p := range []string{".", "..", "a/..", "a//b", "a/./b", "/a", "a/", "a\\b", "a\x00b", "a\nb"} {
		if validTreePath(p) {
			t.Fatalf("accepted path %q", p)
		}
	}
}

func TestOfflineProxyZipMatchesCommittedGoSum(t *testing.T) {
	zipPath := filepath.Join("../../../fixtures/go-upb03-offline-proxy/example.test/seme/checksum/@v/v1.2.3.zip")
	raw, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	prefix := "example.test/seme/checksum@v1.2.3/"
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !strings.HasPrefix(f.Name, prefix) {
			t.Fatalf("zip path %q", f.Name)
		}
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		data, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		files[f.Name] = data
	}
	_, got, err := tree(files)
	if err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile("../../../fixtures/go-upb03-dependency-v1/go.sum")
	if err != nil {
		t.Fatal(err)
	}
	want, err := findSum(sum, "example.test/seme/checksum", "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("h1 got %s want %s", got, want)
	}
}
