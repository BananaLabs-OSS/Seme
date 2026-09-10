// Command go-upb05-project projects an authenticated one-run v36 UPB05
// bundle while preserving detached ordinary project files.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/goupb05bundle"
	"seme.local/reference/projectbundle"
	"seme.local/reference/projectroundtrip"
	"seme.local/reference/projectsource"
	"strings"
	"time"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb05-project:", err)
		os.Exit(65)
	}
}
func run(parent context.Context, args []string) error {
	f := flag.NewFlagSet("go-upb05-project", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	paths := map[string]*string{}
	names := []string{"root", "to", "module", "selection", "manifest", "construction-v36", "execution-v36", "project-base-v8", "inventory-v8", "package-detail-v4", "package-v4", "dependency-v1", "configuration-v3", "project-v8", "foundation-contract", "execution-contract", "package-v4-contract", "dependency-contract", "configuration-v3-contract", "project-v8-contract", "k0", "g1-compiler"}
	for _, n := range names {
		x := new(string)
		paths[n] = x
		f.StringVar(x, n, "", n)
	}
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	for _, n := range names {
		if *paths[n] == "" {
			return fmt.Errorf("flag_missing:%s", n)
		}
	}
	root, err := strictDir(*paths["root"])
	if err != nil {
		return err
	}
	dest := filepath.Clean(*paths["to"])
	if !filepath.IsAbs(dest) || inside(root, dest) {
		return fmt.Errorf("destination")
	}
	if _, err = strictDir(filepath.Dir(dest)); err != nil {
		return fmt.Errorf("destination_parent")
	}
	if _, err = os.Lstat(dest); err == nil {
		return fmt.Errorf("destination_exists")
	}
	read := func(n string) ([]byte, error) { return readStrict(*paths[n]) }
	must := func(n string) []byte { b, _ := read(n); return b }
	for _, n := range names[3:] {
		if _, err = read(n); err != nil {
			return fmt.Errorf("%s:%w", n, err)
		}
	}
	selection, err := goconfigurationmanifest.Parse(must("selection"))
	if err != nil {
		return err
	}
	v8, e := contractcatalog.ResolveProjectContractSetV8(must("foundation-contract"), must("execution-contract"), must("package-v4-contract"), must("dependency-contract"), must("configuration-v3-contract"), must("project-v8-contract"))
	if e != nil {
		return e
	}
	a := goupb05bundle.V8Artifacts{Construction: must("construction-v36"), Execution: must("execution-v36"), ProjectBase: must("project-base-v8"), Inventory: must("inventory-v8"), PackageDetail: must("package-detail-v4"), PackageV4: must("package-v4"), Dependency: must("dependency-v1"), ConfigurationV3: must("configuration-v3"), ProjectV8: must("project-v8")}
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	loaded, e := goupb05bundle.LoadV8(ctx, goupb05bundle.V8Input{Contracts: v8, Artifacts: a, Manifest: must("manifest"), Selection: selection, Compile: func(ctx context.Context, in []byte) ([]byte, error) {
		return compile(ctx, *paths["k0"], *paths["g1-compiler"], in)
	}})
	if e != nil {
		return e
	}
	packages, e := goupb05bundle.ProjectV8(loaded)
	if e != nil {
		return e
	}
	policy := projectsource.Policy{TrackedExtensions: []string{".go"}, IgnoredPrefixes: []string{".seme-cache/"}, IgnoredSuffixes: []string{"_test.go"}, VendoredPrefixes: []string{"vendor/"}, GeneratedHeader: []byte("// Code generated "), MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 8 << 20}
	tool := projectsource.Toolchain{Language: "go", Toolchain: "go1.26", Profile: "upb-05-bounded-v1", SemanticRevision: "go-upb05-configuration-v2"}
	snapshot, e := projectsource.Discover(root, *paths["module"], tool, policy)
	if e != nil {
		return e
	}
	bundle, e := projectbundle.Capture(root, snapshot, policy)
	if e != nil {
		return e
	}
	projected := map[string][]byte{}
	for pkg, data := range packages {
		rel := ""
		if pkg != *paths["module"] {
			if !strings.HasPrefix(pkg, *paths["module"]+"/") {
				return fmt.Errorf("package_outside_module")
			}
			rel = strings.TrimPrefix(pkg, *paths["module"]+"/")
		}
		name := "seme_projected.go"
		if rel != "" {
			name = rel + "/seme_projected.go"
		}
		projected[name] = data
	}
	_, e = projectroundtrip.PublishProjected(dest, snapshot, bundle, projected, policy)
	return e
}
func strictDir(p string) (string, error) {
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("not_absolute")
	}
	r, e := filepath.EvalSymlinks(p)
	if e != nil || r != filepath.Clean(p) {
		return "", fmt.Errorf("not_plain")
	}
	i, e := os.Lstat(p)
	if e != nil || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("not_directory")
	}
	return p, nil
}
func inside(root, p string) bool {
	r, e := filepath.Rel(root, p)
	return e == nil && (r == "." || (r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))))
}
func readStrict(p string) ([]byte, error) {
	if !filepath.IsAbs(p) {
		return nil, fmt.Errorf("not_absolute")
	}
	r, e := filepath.EvalSymlinks(p)
	if e != nil || r != filepath.Clean(p) {
		return nil, fmt.Errorf("not_plain")
	}
	before, e := os.Lstat(p)
	if e != nil || !before.Mode().IsRegular() || before.Size() < 0 || before.Size() > 64<<20 {
		return nil, fmt.Errorf("not_regular")
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("changed")
	}
	b, e := io.ReadAll(io.LimitReader(f, 64<<20+1))
	if e != nil || int64(len(b)) != opened.Size() {
		return nil, fmt.Errorf("changed")
	}
	after, e := os.Lstat(p)
	if e != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("changed")
	}
	return b, nil
}
func compile(ctx context.Context, k0, compiler string, in []byte) ([]byte, error) {
	d, e := os.MkdirTemp("", "seme-upb05-project-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(d)
	src, out := filepath.Join(d, "in.g1"), filepath.Join(d, "out.seme")
	if e = os.WriteFile(src, in, 0600); e != nil {
		return nil, e
	}
	c := exec.CommandContext(ctx, k0, compiler, src, out)
	if b, e := c.CombinedOutput(); e != nil {
		return nil, fmt.Errorf("compiler:%w:%s", e, b)
	}
	return readStrict(out)
}
