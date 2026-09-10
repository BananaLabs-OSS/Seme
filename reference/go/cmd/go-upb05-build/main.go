// Command go-upb05-build is a bounded repository conformance adapter, not a
// general filesystem publisher.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/goprovider"
	"seme.local/reference/goupb05pipeline"
	"seme.local/reference/projectsource"
)

const maxFile, maxTotal, maxArtifact = int64(2 << 20), int64(32 << 20), int64(64 << 20)

type options struct {
	project, proxy, module, pkg, entry, out, dependency, version, localFrom, localTo, manifest            string
	executionG1, execution, foundation, packageV4, projectV8, dependencyV1, configurationV3, k0, compiler string
	revision                                                                                              uint64
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb05-build:", err)
		os.Exit(1)
	}
}
func run(parent context.Context, args []string, stderr io.Writer) error {
	f := flag.NewFlagSet("go-upb05-build", flag.ContinueOnError)
	f.SetOutput(stderr)
	var o options
	flags := map[string]*string{"project": &o.project, "proxy": &o.proxy, "module": &o.module, "package": &o.pkg, "entry": &o.entry, "out": &o.out, "dependency": &o.dependency, "version": &o.version, "local-from": &o.localFrom, "local-to": &o.localTo, "selection": &o.manifest, "execution-g1": &o.executionG1, "execution-contract": &o.execution, "foundation-contract": &o.foundation, "package-v4": &o.packageV4, "project-v8": &o.projectV8, "dependency-v1": &o.dependencyV1, "configuration-v3": &o.configurationV3, "k0": &o.k0, "g1-compiler": &o.compiler}
	for n, p := range flags {
		f.StringVar(p, n, "", n+" input")
	}
	f.Uint64Var(&o.revision, "revision", 0, "nonzero client revision")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	if err := o.validate(); err != nil {
		return err
	}
	root, err := strictDir(o.project)
	if err != nil {
		return fmt.Errorf("project:%w", err)
	}
	if _, err = strictDir(o.proxy); err != nil {
		return fmt.Errorf("proxy:%w", err)
	}
	if inside(root, o.out) {
		return fmt.Errorf("output_inside_project")
	}
	if _, err = os.Lstat(o.out); err == nil {
		return fmt.Errorf("output_exists")
	}
	for _, p := range o.inputs() {
		if _, err = readStrict(p); err != nil {
			return fmt.Errorf("input:%w", err)
		}
	}
	files, err := readProject(root)
	if err != nil {
		return err
	}
	policy := projectsource.Policy{TrackedExtensions: []string{".go"}, IgnoredSuffixes: []string{"_test.go"}, GeneratedHeader: []byte("// generated"), MaxFiles: 1024, MaxFileBytes: maxFile, MaxTotalBytes: maxTotal}
	sources, err := projectsource.Discover(root, o.module, projectsource.Toolchain{Language: "go", Toolchain: "go1.26", Profile: "upb-05-bounded-v1", SemanticRevision: "provider-v1"}, policy)
	if err != nil {
		return err
	}
	read := func(p string) []byte { b, _ := readStrict(p); return b }
	selection, err := goconfigurationmanifest.Parse(read(o.manifest))
	if err != nil {
		return err
	}
	v8, err := contractcatalog.ResolveProjectContractSetV8(read(o.foundation), read(o.execution), read(o.packageV4), read(o.dependencyV1), read(o.configurationV3), read(o.projectV8))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	result, err := goupb05pipeline.BuildV8(ctx, goupb05pipeline.V8Input{Documents: goprovider.DocumentSnapshot{Revision: o.revision, ModulePath: o.module, PackagePath: o.pkg, Entry: o.entry, Files: files}, Sources: sources, Contracts: v8, Dependency: goupb05pipeline.V8DependencyInput{ProjectRoot: root, ProxyRoot: o.proxy, Module: o.dependency, Version: o.version, LocalFrom: o.localFrom, LocalTo: o.localTo}, Selection: selection, ExecutionG1: read(o.executionG1), Compile: func(ctx context.Context, in []byte) ([]byte, error) { return compile(ctx, o, in) }})
	if err != nil {
		return err
	}
	return goupb05pipeline.PublishV8(o.out, result)
}
func (o options) validate() error {
	vals := []string{o.project, o.proxy, o.module, o.pkg, o.entry, o.out, o.dependency, o.version, o.localFrom, o.localTo, o.manifest, o.executionG1, o.execution, o.foundation, o.packageV4, o.projectV8, o.dependencyV1, o.configurationV3, o.k0, o.compiler}
	for _, v := range vals {
		if v == "" {
			return fmt.Errorf("flag_missing")
		}
	}
	for _, p := range append([]string{o.project, o.proxy, o.out}, o.inputs()...) {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p {
			return fmt.Errorf("absolute_path")
		}
	}
	if o.revision == 0 {
		return fmt.Errorf("revision")
	}
	return nil
}
func (o options) inputs() []string {
	return []string{o.manifest, o.executionG1, o.execution, o.foundation, o.packageV4, o.projectV8, o.dependencyV1, o.configurationV3, o.k0, o.compiler}
}
func strictDir(p string) (string, error) {
	i, e := os.Lstat(p)
	if e != nil || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("not_plain_directory")
	}
	r, e := filepath.EvalSymlinks(p)
	if e != nil || r != p {
		return "", fmt.Errorf("symlink_component")
	}
	return p, nil
}
func inside(root, p string) bool {
	rel, e := filepath.Rel(root, p)
	return e == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))))
}
func readStrict(p string) ([]byte, error) {
	if !filepath.IsAbs(p) {
		return nil, fmt.Errorf("not_absolute")
	}
	r, e := filepath.EvalSymlinks(p)
	if e != nil || r != p {
		return nil, fmt.Errorf("not_plain")
	}
	before, e := os.Lstat(p)
	if e != nil || !before.Mode().IsRegular() || before.Size() < 0 || before.Size() > maxArtifact {
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
	b, e := io.ReadAll(io.LimitReader(f, maxArtifact+1))
	if e != nil || int64(len(b)) != opened.Size() {
		return nil, fmt.Errorf("changed")
	}
	after, e := os.Lstat(p)
	if e != nil || !os.SameFile(before, after) {
		return nil, fmt.Errorf("changed")
	}
	return b, nil
}
func readProject(root string) (map[string]string, error) {
	out := map[string]string{}
	var total int64
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if p == root {
			return nil
		}
		i, e := d.Info()
		if e != nil {
			return e
		}
		if d.IsDir() {
			if i.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("nonregular")
			}
			return nil
		}
		if !i.Mode().IsRegular() {
			return fmt.Errorf("nonregular")
		}
		if filepath.Ext(p) != ".go" || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		if len(out) >= 1024 || i.Size() > maxFile {
			return fmt.Errorf("bounds")
		}
		total += i.Size()
		if total > maxTotal {
			return fmt.Errorf("bounds")
		}
		b, e := readStrict(p)
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(root, p)
		if e != nil {
			return e
		}
		out[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	return out, err
}
func compile(ctx context.Context, o options, in []byte) ([]byte, error) {
	d, e := os.MkdirTemp("", "seme-upb05-compile-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(d)
	src, out := filepath.Join(d, "in.g1"), filepath.Join(d, "out.seme")
	if e = os.WriteFile(src, in, 0600); e != nil {
		return nil, e
	}
	c := exec.CommandContext(ctx, o.k0, o.compiler, src, out)
	if b, e := c.CombinedOutput(); e != nil {
		return nil, fmt.Errorf("compiler:%w:%s", e, b)
	}
	return readStrict(out)
}
