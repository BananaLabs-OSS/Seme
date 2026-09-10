// Command go-upb04-build is the bounded conformance adapter for producing a
// complete Go UPB-04 bundle. It is deliberately not a general publisher.
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
	"seme.local/reference/goupb04pipeline"
	"seme.local/reference/projectsource"
)

const maxFile, maxTotal, maxArtifact = int64(2 << 20), int64(32 << 20), int64(64 << 20)

type options struct {
	project, proxy, module, pkg, entry, out, dependency, version, localFrom, localTo  string
	executionG1, execution, packageV1, packageV2, packageV3                           string
	projectV1, projectV2, projectV3, projectV4, projectV5, dependencyV1, k0, compiler string
	revision                                                                          uint64
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb04-build:", err)
		os.Exit(1)
	}
}

func run(parent context.Context, args []string, stderr io.Writer) error {
	f := flag.NewFlagSet("go-upb04-build", flag.ContinueOnError)
	f.SetOutput(stderr)
	var o options
	for name, dst := range map[string]*string{"project": &o.project, "proxy": &o.proxy, "module": &o.module, "package": &o.pkg, "entry": &o.entry, "out": &o.out, "dependency": &o.dependency, "version": &o.version, "local-from": &o.localFrom, "local-to": &o.localTo, "execution-g1": &o.executionG1, "execution-contract": &o.execution, "package-v1": &o.packageV1, "package-v2": &o.packageV2, "package-v3": &o.packageV3, "project-v1": &o.projectV1, "project-v2": &o.projectV2, "project-v3": &o.projectV3, "project-v4": &o.projectV4, "project-v5": &o.projectV5, "dependency-v1": &o.dependencyV1, "k0": &o.k0, "g1-compiler": &o.compiler} {
		f.StringVar(dst, name, "", name+" input")
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
	root, err := strictDirectory(o.project)
	if err != nil {
		return fmt.Errorf("project:%w", err)
	}
	if _, err = strictDirectory(o.proxy); err != nil {
		return fmt.Errorf("proxy:%w", err)
	}
	if inside(root, o.out) {
		return fmt.Errorf("output_inside_project")
	}
	if _, err = os.Lstat(o.out); err == nil {
		return fmt.Errorf("output_exists")
	} else if !os.IsNotExist(err) {
		return err
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
	sources, err := projectsource.Discover(root, o.module, projectsource.Toolchain{Language: "go", Toolchain: "go1.26", Profile: "bounded-v1", SemanticRevision: "provider-v1"}, policy)
	if err != nil {
		return err
	}
	read := func(p string) []byte { b, _ := readStrict(p); return b }
	v1, err := contractcatalog.ResolveProjectContractSet(read(o.execution), read(o.packageV1), read(o.projectV1))
	if err != nil {
		return err
	}
	v2, err := contractcatalog.ResolveProjectContractSetV2(read(o.execution), read(o.packageV1), read(o.projectV2))
	if err != nil {
		return err
	}
	v3, err := contractcatalog.ResolveProjectContractSetV3(read(o.execution), read(o.packageV2), read(o.projectV3))
	if err != nil {
		return err
	}
	v4, err := contractcatalog.ResolveProjectContractSetV4(read(o.execution), read(o.packageV2), read(o.dependencyV1), read(o.projectV4))
	if err != nil {
		return err
	}
	v5, err := contractcatalog.ResolveProjectContractSetV5(read(o.execution), read(o.packageV3), read(o.dependencyV1), read(o.projectV5))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	base := goprojectpipeline.Input{Documents: goprovider.DocumentSnapshot{Revision: o.revision, ModulePath: o.module, PackagePath: o.pkg, Entry: o.entry, Files: files}, Sources: sources, Contracts: goprojectpipeline.Contracts{V1: v1, V2: v2, V3: v3}, ExecutionG1: read(o.executionG1), Compile: func(ctx context.Context, in []byte) ([]byte, error) { return compile(ctx, o, in) }}
	result, err := goupb04pipeline.Build(ctx, goupb04pipeline.Input{Base: goupb03pipeline.Input{Base: base, Contracts: v4, Dependency: goupb03pipeline.DependencyInput{ProjectRoot: root, ProxyRoot: o.proxy, Module: o.dependency, Version: o.version, LocalFrom: o.localFrom, LocalTo: o.localTo}}, Contracts: v5})
	if err != nil {
		return err
	}
	return goupb04pipeline.Publish(o.out, result)
}

func (o options) validate() error {
	values := []string{o.project, o.proxy, o.module, o.pkg, o.entry, o.out, o.dependency, o.version, o.localFrom, o.localTo, o.executionG1, o.execution, o.packageV1, o.packageV2, o.packageV3, o.projectV1, o.projectV2, o.projectV3, o.projectV4, o.projectV5, o.dependencyV1, o.k0, o.compiler}
	for _, v := range values {
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
	return []string{o.executionG1, o.execution, o.packageV1, o.packageV2, o.packageV3, o.projectV1, o.projectV2, o.projectV3, o.projectV4, o.projectV5, o.dependencyV1, o.k0, o.compiler}
}
func strictDirectory(p string) (string, error) {
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
	return e == nil && (rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."))
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
	if e != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
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
		info, e := d.Info()
		if e != nil {
			return e
		}
		if d.IsDir() {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("nonregular")
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("nonregular")
		}
		if filepath.Ext(p) != ".go" || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		if len(out) >= 1024 || info.Size() > maxFile {
			return fmt.Errorf("bounds")
		}
		total += info.Size()
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
	dir, e := os.MkdirTemp("", "seme-upb04-compile-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(dir)
	src, out := filepath.Join(dir, "in.g1"), filepath.Join(dir, "out.seme")
	if e = os.WriteFile(src, in, 0600); e != nil {
		return nil, e
	}
	cmd := exec.CommandContext(ctx, o.k0, o.compiler, src, out)
	if b, e := cmd.CombinedOutput(); e != nil {
		return nil, fmt.Errorf("compiler:%w:%s", e, b)
	}
	return readStrict(out)
}
