// Command go-upb07-project projects an authenticated Project-v10 bundle while
// preserving independently checked ordinary detached project files.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/godurablemanifest"
	"seme.local/reference/goupb07bundle"
	"seme.local/reference/projectbundle"
	"seme.local/reference/projectroundtrip"
	"seme.local/reference/projectsource"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb07-project:", err)
		os.Exit(65)
	}
}

func run(parent context.Context, args []string) error {
	f := flag.NewFlagSet("go-upb07-project", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	paths := map[string]*string{}
	for _, name := range []string{"root", "to", "module", "bundle", "selection", "durable-selection", "foundation", "execution", "package", "dependency", "configuration", "resource", "durable-state", "source-presentation", "project-v8", "project-v9", "project-v10", "k0", "g1-compiler"} {
		paths[name] = new(string)
		f.StringVar(paths[name], name, "", name)
	}
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	for name, value := range paths {
		if *value == "" {
			return fmt.Errorf("flag_missing:%s", name)
		}
	}
	root, err := strictDir(*paths["root"])
	if err != nil {
		return err
	}
	destination := filepath.Clean(*paths["to"])
	if !filepath.IsAbs(destination) || inside(root, destination) {
		return fmt.Errorf("destination")
	}
	if _, err = strictDir(filepath.Dir(destination)); err != nil {
		return fmt.Errorf("destination_parent")
	}
	if _, err = os.Lstat(destination); err == nil {
		return fmt.Errorf("destination_exists")
	}
	read := func(name string) ([]byte, error) { return readStrict(*paths[name]) }
	inputs := map[string][]byte{}
	for _, name := range []string{"selection", "durable-selection", "foundation", "execution", "package", "dependency", "configuration", "resource", "durable-state", "source-presentation", "project-v8", "project-v9", "project-v10", "k0", "g1-compiler"} {
		inputs[name], err = read(name)
		if err != nil {
			return fmt.Errorf("%s:%w", name, err)
		}
	}
	selection, err := goconfigurationmanifest.Parse(inputs["selection"])
	if err != nil {
		return err
	}
	durableSelection, err := godurablemanifest.Parse(inputs["durable-selection"])
	if err != nil {
		return err
	}
	v8, err := contractcatalog.ResolveProjectContractSetV8(inputs["foundation"], inputs["execution"], inputs["package"], inputs["dependency"], inputs["configuration"], inputs["project-v8"])
	if err != nil {
		return err
	}
	v9, err := contractcatalog.ResolveProjectContractSetV9(inputs["foundation"], inputs["execution"], inputs["package"], inputs["dependency"], inputs["configuration"], inputs["resource"], inputs["project-v9"])
	if err != nil {
		return err
	}
	v10, err := contractcatalog.ResolveProjectContractSetV10(inputs["foundation"], inputs["execution"], inputs["package"], inputs["dependency"], inputs["configuration"], inputs["resource"], inputs["durable-state"], inputs["source-presentation"], inputs["project-v9"], inputs["project-v10"])
	if err != nil {
		return err
	}
	a, manifest, blobs, err := goupb07bundle.ReadDirectory(*paths["bundle"])
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	loaded, err := goupb07bundle.Load(ctx, goupb07bundle.Input{Contracts: v10, V9: v9, V8: v8, Artifacts: a, Manifest: manifest, Selection: selection, DurableSelection: durableSelection, Blobs: blobs, Compile: func(ctx context.Context, source []byte) ([]byte, error) {
		return compile(ctx, *paths["k0"], *paths["g1-compiler"], source)
	}})
	if err != nil {
		return err
	}
	// Resource paths belong to the authenticated resource contract, while the
	// bytes remain ordinary source-tree material. Require both views to agree.
	for _, item := range loaded.Base.Resource.Model.Resources {
		path := filepath.Join(root, filepath.FromSlash(item.Path))
		data, e := readStrict(path)
		if e != nil || !bytes.Equal(data, loaded.Blobs[item.SHA256]) {
			return fmt.Errorf("resource_source:%s", item.Identity)
		}
	}
	packages, err := goupb07bundle.Project(loaded)
	if err != nil {
		return err
	}
	policy := projectsource.Policy{TrackedExtensions: []string{".go"}, IgnoredPrefixes: []string{".seme-cache/"}, IgnoredSuffixes: []string{"_test.go"}, VendoredPrefixes: []string{"vendor/"}, GeneratedHeader: []byte("// Code generated "), MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 8 << 20}
	snapshot, err := projectsource.Discover(root, *paths["module"], projectsource.Toolchain{Language: "go", Toolchain: "go1.26", Profile: "upb-07-bounded-v1", SemanticRevision: "go-upb07-durable-v1"}, policy)
	if err != nil {
		return err
	}
	sourceBundle, err := projectbundle.Capture(root, snapshot, policy)
	if err != nil {
		return err
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
	_, err = projectroundtrip.PublishProjected(destination, snapshot, sourceBundle, projected, policy)
	return err
}

func strictDir(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("not_absolute")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != filepath.Clean(path) {
		return "", fmt.Errorf("not_plain")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("not_directory")
	}
	return path, nil
}
func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
func readStrict(path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("not_absolute")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != filepath.Clean(path) {
		return nil, fmt.Errorf("not_plain")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > 64<<20 {
		return nil, fmt.Errorf("not_regular")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("changed")
	}
	data, err := io.ReadAll(io.LimitReader(file, 64<<20+1))
	if err != nil || int64(len(data)) != opened.Size() {
		return nil, fmt.Errorf("changed")
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("changed")
	}
	return data, nil
}
func compile(ctx context.Context, k0, compiler string, source []byte) ([]byte, error) {
	directory, err := os.MkdirTemp("", "seme-upb07-project-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	input, output := filepath.Join(directory, "input.g1"), filepath.Join(directory, "output.seme")
	if err = os.WriteFile(input, source, 0600); err != nil {
		return nil, err
	}
	data, err := exec.CommandContext(ctx, k0, compiler, input, output).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compiler:%w:%s", err, data)
	}
	return readStrict(output)
}
