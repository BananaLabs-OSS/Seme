// Command go-upb07-build is the bounded, authenticated Go producer for the
// UPB-07 durable-state project bundle. It is not a general filesystem publisher.
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
	"seme.local/reference/goupb06pipeline"
	"seme.local/reference/goupb07pipeline"
	"seme.local/reference/projectsource"
)

const maxFile, maxTotal, maxArtifact = int64(2 << 20), int64(32 << 20), int64(64 << 20)

type options struct {
	project, proxy, module, pkg, entry, out, dependency, version, localFrom, localTo            string
	configurationSelection, resourceManifest, resourceOwner, durableSelection                   string
	executionG1, execution, foundation, packageV4, projectV8                                    string
	dependencyV1, configurationV3, resourceV1, projectV9, durableV1, presentationV1, projectV10 string
	k0, compiler                                                                                string
	revision                                                                                    uint64
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb07-build:", err)
		os.Exit(1)
	}
}

func run(parent context.Context, args []string, stderr io.Writer) error {
	f := flag.NewFlagSet("go-upb07-build", flag.ContinueOnError)
	f.SetOutput(stderr)
	var o options
	flags := map[string]*string{
		"project": &o.project, "proxy": &o.proxy, "module": &o.module,
		"package": &o.pkg, "entry": &o.entry, "out": &o.out,
		"dependency": &o.dependency, "version": &o.version,
		"local-from": &o.localFrom, "local-to": &o.localTo,
		"selection": &o.configurationSelection, "resources": &o.resourceManifest,
		"resource-owner": &o.resourceOwner, "durable-selection": &o.durableSelection,
		"execution-g1": &o.executionG1, "execution-contract": &o.execution,
		"foundation-contract": &o.foundation, "package-v4": &o.packageV4,
		"project-v8": &o.projectV8, "dependency-v1": &o.dependencyV1,
		"configuration-v3": &o.configurationV3, "resource-v1": &o.resourceV1,
		"project-v9": &o.projectV9, "durable-state-v1": &o.durableV1,
		"source-presentation-v1": &o.presentationV1,
		"project-v10":            &o.projectV10, "k0": &o.k0, "g1-compiler": &o.compiler,
	}
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
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("output_lstat:%w", err)
	}
	inputs := map[string][]byte{}
	for _, p := range o.inputs() {
		b, e := readStrict(p)
		if e != nil {
			return fmt.Errorf("input:%w", e)
		}
		inputs[p] = b
	}
	files, err := readProject(root)
	if err != nil {
		return err
	}
	policy := projectsource.Policy{TrackedExtensions: []string{".go"}, IgnoredSuffixes: []string{"_test.go"}, GeneratedHeader: []byte("// generated"), MaxFiles: 1024, MaxFileBytes: maxFile, MaxTotalBytes: maxTotal}
	sources, err := projectsource.Discover(root, o.module, projectsource.Toolchain{Language: "go", Toolchain: "go1.26", Profile: "upb-07-bounded-v1", SemanticRevision: "provider-v1"}, policy)
	if err != nil {
		return err
	}
	selection, err := goconfigurationmanifest.Parse(inputs[o.configurationSelection])
	if err != nil {
		return err
	}
	v8, err := contractcatalog.ResolveProjectContractSetV8(inputs[o.foundation], inputs[o.execution], inputs[o.packageV4], inputs[o.dependencyV1], inputs[o.configurationV3], inputs[o.projectV8])
	if err != nil {
		return err
	}
	v9, err := contractcatalog.ResolveProjectContractSetV9(inputs[o.foundation], inputs[o.execution], inputs[o.packageV4], inputs[o.dependencyV1], inputs[o.configurationV3], inputs[o.resourceV1], inputs[o.projectV9])
	if err != nil {
		return err
	}
	v10, err := contractcatalog.ResolveProjectContractSetV10(inputs[o.foundation], inputs[o.execution], inputs[o.packageV4], inputs[o.dependencyV1], inputs[o.configurationV3], inputs[o.resourceV1], inputs[o.durableV1], inputs[o.presentationV1], inputs[o.projectV9], inputs[o.projectV10])
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	base := goupb05pipeline.V8Input{
		Documents: goprovider.DocumentSnapshot{Revision: o.revision, ModulePath: o.module, PackagePath: o.pkg, Entry: o.entry, Files: files},
		Sources:   sources, Contracts: v8,
		Dependency: goupb05pipeline.V8DependencyInput{ProjectRoot: root, ProxyRoot: o.proxy, Module: o.dependency, Version: o.version, LocalFrom: o.localFrom, LocalTo: o.localTo},
		Selection:  selection, ExecutionG1: inputs[o.executionG1],
		Compile: func(ctx context.Context, in []byte) ([]byte, error) { return compile(ctx, o, in) },
	}
	r, err := goupb07pipeline.Build(ctx, goupb07pipeline.Input{
		Base:      goupb06pipeline.Input{Base: base, Contracts: v9, Manifest: inputs[o.resourceManifest], OwnerPackage: o.resourceOwner},
		Contracts: v10, Manifest: inputs[o.durableSelection],
	})
	if err != nil {
		return err
	}
	return goupb07pipeline.Publish(o.out, r)
}

func (o options) validate() error {
	values := []string{o.project, o.proxy, o.module, o.pkg, o.entry, o.out, o.dependency, o.version, o.localFrom, o.localTo, o.configurationSelection, o.resourceManifest, o.resourceOwner, o.durableSelection, o.executionG1, o.execution, o.foundation, o.packageV4, o.projectV8, o.dependencyV1, o.configurationV3, o.resourceV1, o.projectV9, o.durableV1, o.presentationV1, o.projectV10, o.k0, o.compiler}
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
	return []string{o.configurationSelection, o.resourceManifest, o.durableSelection, o.executionG1, o.execution, o.foundation, o.packageV4, o.projectV8, o.dependencyV1, o.configurationV3, o.resourceV1, o.projectV9, o.durableV1, o.presentationV1, o.projectV10, o.k0, o.compiler}
}

func strictDir(p string) (string, error) {
	i, err := os.Lstat(p)
	if err != nil || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("not_plain_directory")
	}
	r, err := filepath.EvalSymlinks(p)
	if err != nil || r != p {
		return "", fmt.Errorf("symlink_component")
	}
	return p, nil
}

func inside(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))))
}

func readStrict(p string) ([]byte, error) {
	if !filepath.IsAbs(p) {
		return nil, fmt.Errorf("not_absolute")
	}
	r, err := filepath.EvalSymlinks(p)
	if err != nil || r != p {
		return nil, fmt.Errorf("not_plain")
	}
	before, err := os.Lstat(p)
	if err != nil || !before.Mode().IsRegular() || before.Size() < 0 || before.Size() > maxArtifact {
		return nil, fmt.Errorf("not_regular")
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("changed")
	}
	b, err := io.ReadAll(io.LimitReader(f, maxArtifact+1))
	if err != nil || int64(len(b)) != opened.Size() {
		return nil, fmt.Errorf("changed")
	}
	after, err := os.Lstat(p)
	if err != nil || !os.SameFile(before, after) {
		return nil, fmt.Errorf("changed")
	}
	return b, nil
}

func readProject(root string) (map[string]string, error) {
	out := map[string]string{}
	var total int64
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if p == root {
			return nil
		}
		i, err := d.Info()
		if err != nil {
			return err
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
		b, err := readStrict(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	return out, err
}

func compile(ctx context.Context, o options, in []byte) ([]byte, error) {
	d, err := os.MkdirTemp("", "seme-upb07-compile-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(d)
	src, out := filepath.Join(d, "in.g1"), filepath.Join(d, "out.seme")
	if err = os.WriteFile(src, in, 0600); err != nil {
		return nil, err
	}
	c := exec.CommandContext(ctx, o.k0, o.compiler, src, out)
	if b, runErr := c.CombinedOutput(); runErr != nil {
		return nil, fmt.Errorf("compiler:%w:%s", runErr, b)
	}
	return readStrict(out)
}
