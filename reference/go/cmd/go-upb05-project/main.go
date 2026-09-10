// Command go-upb05-project projects an authenticated eleven-artifact UPB05
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
	names := []string{"root", "to", "module", "selection", "manifest", "construction", "project-instance-v1", "inventory-v2", "package-instance-v2", "project-instance-v3", "dependency-instance-v1", "project-instance-v4", "package-instance-v3", "project-instance-v5", "configuration-instance-v2", "project-instance-v7", "foundation-contract", "execution-contract", "package-v1-contract", "package-v2-contract", "package-v3-contract", "dependency-contract", "configuration-v1-contract", "configuration-v2-contract", "project-v1-contract", "project-v2-contract", "project-v3-contract", "project-v4-contract", "project-v5-contract", "project-v6-contract", "project-v7-contract", "k0", "g1-compiler"}
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
	execution, package1, package2, package3, dependency := must("execution-contract"), must("package-v1-contract"), must("package-v2-contract"), must("package-v3-contract"), must("dependency-contract")
	pv1, pv2, pv3, pv4, pv5, pv6, pv7 := must("project-v1-contract"), must("project-v2-contract"), must("project-v3-contract"), must("project-v4-contract"), must("project-v5-contract"), must("project-v6-contract"), must("project-v7-contract")
	foundation, c1, c2 := must("foundation-contract"), must("configuration-v1-contract"), must("configuration-v2-contract")
	v1, e := contractcatalog.ResolveProjectContractSet(execution, package1, pv1)
	if e != nil {
		return e
	}
	v2, e := contractcatalog.ResolveProjectContractSetV2(execution, package1, pv2)
	if e != nil {
		return e
	}
	v3, e := contractcatalog.ResolveProjectContractSetV3(execution, package2, pv3)
	if e != nil {
		return e
	}
	v4, e := contractcatalog.ResolveProjectContractSetV4(execution, package2, dependency, pv4)
	if e != nil {
		return e
	}
	v5, e := contractcatalog.ResolveProjectContractSetV5(execution, package3, dependency, pv5)
	if e != nil {
		return e
	}
	v6, e := contractcatalog.ResolveProjectContractSetV6(foundation, execution, package3, dependency, c1, pv6)
	if e != nil {
		return e
	}
	v7, e := contractcatalog.ResolveProjectContractSetV7(foundation, execution, package3, dependency, c2, pv7)
	if e != nil {
		return e
	}
	a := goupb05bundle.Artifacts{Construction: must("construction"), ProjectV1: must("project-instance-v1"), InventoryV2: must("inventory-v2"), PackageV2: must("package-instance-v2"), ProjectV3: must("project-instance-v3"), DependencyV1: must("dependency-instance-v1"), ProjectV4: must("project-instance-v4"), PackageV3: must("package-instance-v3"), ProjectV5: must("project-instance-v5"), ConfigurationV2: must("configuration-instance-v2"), ProjectV7: must("project-instance-v7")}
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	loaded, e := goupb05bundle.Load(ctx, goupb05bundle.Input{Contracts: goupb05bundle.Contracts{V1: v1, V2: v2, V3: v3, V4: v4, V5: v5, V6: v6, V7: v7, ProjectV2: v2.Project()}, Artifacts: a, Manifest: must("manifest"), Selection: selection, Compile: func(ctx context.Context, in []byte) ([]byte, error) {
		return compile(ctx, *paths["k0"], *paths["g1-compiler"], in)
	}})
	if e != nil {
		return e
	}
	packages, e := goupb05bundle.Project(loaded)
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
	i, e := os.Lstat(p)
	if e != nil || !i.Mode().IsRegular() || i.Size() > 64<<20 {
		return nil, fmt.Errorf("not_regular")
	}
	return os.ReadFile(p)
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
