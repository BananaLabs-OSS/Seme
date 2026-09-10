// Command go-upb03-build builds a bounded Go project and its pinned offline
// dependency closure into a complete Project-v4 bundle.
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
	"seme.local/reference/goprojectpipeline"
	"seme.local/reference/goprovider"
	"seme.local/reference/goupb03pipeline"
	"seme.local/reference/projectsource"
)

const maxFile, maxTotal, maxArtifact = int64(2 << 20), int64(32 << 20), int64(64 << 20)
const maxFiles = 1024

type options struct {
	project, proxy, module, pkg, entry, out, dependency, version, localFrom, localTo string
	executionG1, executionContract, packageV1, packageV2                             string
	projectV1, projectV2, projectV3, projectV4, dependencyV1, k0, compiler           string
	revision                                                                         uint64
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(parent context.Context, args []string, stderr io.Writer) error {
	f := flag.NewFlagSet("go-upb03-build", flag.ContinueOnError)
	f.SetOutput(stderr)
	var o options
	f.StringVar(&o.project, "project", "", "absolute project root")
	f.StringVar(&o.proxy, "proxy", "", "absolute offline GOPROXY root")
	f.StringVar(&o.module, "module", "", "module identity")
	f.StringVar(&o.pkg, "package", "", "root package identity")
	f.StringVar(&o.entry, "entry", "", "entry function")
	f.StringVar(&o.dependency, "dependency", "", "external module identity")
	f.StringVar(&o.version, "version", "", "exact dependency version")
	f.StringVar(&o.localFrom, "local-from", "", "local dependency-edge source package")
	f.StringVar(&o.localTo, "local-to", "", "local dependency-edge target package")
	f.StringVar(&o.out, "out", "", "new absolute output directory")
	f.Uint64Var(&o.revision, "revision", 0, "nonzero source revision")
	for name, dst := range map[string]*string{"execution-g1": &o.executionG1, "execution-contract": &o.executionContract, "package-v1": &o.packageV1, "package-v2": &o.packageV2, "project-v1": &o.projectV1, "project-v2": &o.projectV2, "project-v3": &o.projectV3, "project-v4": &o.projectV4, "dependency-v1": &o.dependencyV1, "k0": &o.k0, "g1-compiler": &o.compiler} {
		f.StringVar(dst, name, "", name+" input")
	}
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("go_upb03_build.arguments")
	}
	if err := o.validate(); err != nil {
		return err
	}
	root, err := strictDirectory(o.project)
	if err != nil {
		return fmt.Errorf("go_upb03_build.project:%w", err)
	}
	if _, err = strictDirectory(o.proxy); err != nil {
		return fmt.Errorf("go_upb03_build.proxy:%w", err)
	}
	if inside(root, o.out) {
		return fmt.Errorf("go_upb03_build.output_inside_project")
	}
	if _, err = os.Lstat(o.out); err == nil {
		return fmt.Errorf("go_upb03_build.output_exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	paths := []string{o.executionG1, o.executionContract, o.packageV1, o.packageV2, o.projectV1, o.projectV2, o.projectV3, o.projectV4, o.dependencyV1, o.k0, o.compiler}
	for _, p := range paths {
		if err = strictRegular(p); err != nil {
			return fmt.Errorf("go_upb03_build.input:%w", err)
		}
	}
	files, err := readProject(root)
	if err != nil {
		return err
	}
	tool := projectsource.Toolchain{Language: "go", Toolchain: "go1.26", Profile: "bounded-v1", SemanticRevision: "provider-v1"}
	policy := projectsource.Policy{TrackedExtensions: []string{".go"}, IgnoredSuffixes: []string{"_test.go"}, GeneratedHeader: []byte("// generated"), MaxFiles: maxFiles, MaxFileBytes: maxFile, MaxTotalBytes: maxTotal}
	sources, err := projectsource.Discover(root, o.module, tool, policy)
	if err != nil {
		return err
	}
	read := func(p string) ([]byte, error) { return readRegular(p, maxArtifact) }
	ec, err := read(o.executionContract)
	if err != nil {
		return err
	}
	p1, err := read(o.packageV1)
	if err != nil {
		return err
	}
	p2, err := read(o.packageV2)
	if err != nil {
		return err
	}
	r1, err := read(o.projectV1)
	if err != nil {
		return err
	}
	r2, err := read(o.projectV2)
	if err != nil {
		return err
	}
	r3, err := read(o.projectV3)
	if err != nil {
		return err
	}
	r4, err := read(o.projectV4)
	if err != nil {
		return err
	}
	d1, err := read(o.dependencyV1)
	if err != nil {
		return err
	}
	g1, err := read(o.executionG1)
	if err != nil {
		return err
	}
	v1, err := contractcatalog.ResolveProjectContractSet(ec, p1, r1)
	if err != nil {
		return err
	}
	v2, err := contractcatalog.ResolveProjectContractSetV2(ec, p1, r2)
	if err != nil {
		return err
	}
	v3, err := contractcatalog.ResolveProjectContractSetV3(ec, p2, r3)
	if err != nil {
		return err
	}
	v4, err := contractcatalog.ResolveProjectContractSetV4(ec, p2, d1, r4)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	result, err := goupb03pipeline.Build(ctx, goupb03pipeline.Input{Base: goprojectpipeline.Input{Documents: goprovider.DocumentSnapshot{Revision: o.revision, ModulePath: o.module, PackagePath: o.pkg, Entry: o.entry, Files: files}, Sources: sources, Contracts: goprojectpipeline.Contracts{V1: v1, V2: v2, V3: v3}, ExecutionG1: g1, Compile: func(ctx context.Context, input []byte) ([]byte, error) { return compile(ctx, o, input) }}, Contracts: v4, Dependency: goupb03pipeline.DependencyInput{ProjectRoot: root, ProxyRoot: o.proxy, Module: o.dependency, Version: o.version, LocalFrom: o.localFrom, LocalTo: o.localTo}})
	if err != nil {
		return err
	}
	return goupb03pipeline.Publish(o.out, result)
}

func (o options) validate() error {
	values := map[string]string{"project": o.project, "proxy": o.proxy, "module": o.module, "package": o.pkg, "entry": o.entry, "out": o.out, "dependency": o.dependency, "version": o.version, "local-from": o.localFrom, "local-to": o.localTo, "execution-g1": o.executionG1, "execution-contract": o.executionContract, "package-v1": o.packageV1, "package-v2": o.packageV2, "project-v1": o.projectV1, "project-v2": o.projectV2, "project-v3": o.projectV3, "project-v4": o.projectV4, "dependency-v1": o.dependencyV1, "k0": o.k0, "g1-compiler": o.compiler}
	for k, v := range values {
		if v == "" {
			return fmt.Errorf("go_upb03_build.flag_missing:%s", k)
		}
	}
	for _, p := range []string{o.project, o.proxy, o.out, o.executionG1, o.executionContract, o.packageV1, o.packageV2, o.projectV1, o.projectV2, o.projectV3, o.projectV4, o.dependencyV1, o.k0, o.compiler} {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p {
			return fmt.Errorf("go_upb03_build.absolute_path")
		}
	}
	if o.revision == 0 {
		return fmt.Errorf("go_upb03_build.revision")
	}
	return nil
}
func strictDirectory(p string) (string, error) {
	i, e := os.Lstat(p)
	if e != nil {
		return "", e
	}
	if !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("not_plain_directory")
	}
	r, e := filepath.EvalSymlinks(p)
	if e != nil || r != p {
		return "", fmt.Errorf("symlink_component")
	}
	return p, nil
}
func strictRegular(p string) error {
	i, e := os.Lstat(p)
	if e != nil {
		return e
	}
	if !i.Mode().IsRegular() || i.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("not_plain_file")
	}
	r, e := filepath.EvalSymlinks(p)
	if e != nil || r != p {
		return fmt.Errorf("symlink_component")
	}
	return nil
}
func readRegular(p string, limit int64) ([]byte, error) {
	before, e := os.Lstat(p)
	if e != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("go_upb03_build.read_regular")
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, limit+1))
	if e != nil {
		return nil, e
	}
	opened, e := f.Stat()
	after, pe := os.Lstat(p)
	if e != nil || pe != nil || !os.SameFile(before, opened) || !os.SameFile(before, after) || before.Size() != opened.Size() || before.Size() != after.Size() || int64(len(data)) != before.Size() || !before.ModTime().Equal(opened.ModTime()) || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("go_upb03_build.changed_file")
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("go_upb03_build.file_too_large")
	}
	return data, nil
}
func readProject(root string) (map[string]string, error) {
	out := map[string]string{}
	var total int64
	e := filepath.WalkDir(root, func(p string, d fs.DirEntry, we error) error {
		if we != nil {
			return we
		}
		i, e := d.Info()
		if e != nil {
			return e
		}
		if i.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("go_upb03_build.symlink")
		}
		if d.IsDir() {
			return nil
		}
		if !i.Mode().IsRegular() {
			return fmt.Errorf("go_upb03_build.nonregular")
		}
		if filepath.Ext(p) != ".go" || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		if len(out) >= maxFiles || i.Size() > maxFile {
			return fmt.Errorf("go_upb03_build.source_limit")
		}
		b, e := readRegular(p, maxFile)
		if e != nil {
			return e
		}
		total += int64(len(b))
		if total > maxTotal {
			return fmt.Errorf("go_upb03_build.total_limit")
		}
		rel, e := filepath.Rel(root, p)
		if e != nil || rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("go_upb03_build.path")
		}
		out[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if e != nil {
		return nil, e
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("go_upb03_build.no_sources")
	}
	return out, nil
}
func inside(root, path string) bool {
	rel, e := filepath.Rel(root, path)
	return e == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
func compile(ctx context.Context, o options, input []byte) ([]byte, error) {
	dir, e := os.MkdirTemp("", "seme-go-upb03-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(dir)
	src, out := filepath.Join(dir, "input.g1"), filepath.Join(dir, "output.seme")
	if e = os.WriteFile(src, input, 0600); e != nil {
		return nil, e
	}
	cmd := exec.CommandContext(ctx, o.k0, o.compiler, src, out)
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	if b, e := cmd.CombinedOutput(); e != nil {
		return nil, fmt.Errorf("go_upb03_build.compile:%s", string(b))
	}
	return readRegular(out, maxArtifact)
}
