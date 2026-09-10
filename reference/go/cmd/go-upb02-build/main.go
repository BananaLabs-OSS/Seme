// Command go-upb02-build builds the bounded Go UPB-02 five-artifact bundle
// with a digest completion manifest.
package main

import (
	"context"
	"crypto/sha256"
	"errors"
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
	"seme.local/reference/projectsource"
)

const maxFile = 2 << 20
const maxTotal = 32 << 20
const maxFiles = 1024
const maxArtifact = 64 << 20

type options struct {
	project, module, pkg, entry, out, executionG1, executionContract, packageV1, packageV2, projectV1, projectV2, projectV3, k0, compiler string
	revision                                                                                                                              uint64
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(parent context.Context, args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("go-upb02-build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var o options
	fs.StringVar(&o.project, "project", "", "absolute project root")
	fs.StringVar(&o.module, "module", "", "module identity")
	fs.StringVar(&o.pkg, "package", "", "root package identity")
	fs.StringVar(&o.entry, "entry", "", "entry function")
	fs.StringVar(&o.out, "out", "", "new absolute output directory")
	fs.StringVar(&o.executionG1, "execution-g1", "", "Execution v35 G1")
	fs.StringVar(&o.executionContract, "execution-contract", "", "Execution v35 contract")
	fs.StringVar(&o.packageV1, "package-v1", "", "Package v1 contract")
	fs.StringVar(&o.packageV2, "package-v2", "", "Package v2 contract")
	fs.StringVar(&o.projectV1, "project-v1", "", "Project v1 contract")
	fs.StringVar(&o.projectV2, "project-v2", "", "Project v2 contract")
	fs.StringVar(&o.projectV3, "project-v3", "", "Project v3 contract")
	fs.StringVar(&o.k0, "k0", "", "frozen absolute K0 executable")
	fs.StringVar(&o.compiler, "g1-compiler", "", "frozen absolute G1 compiler")
	fs.Uint64Var(&o.revision, "revision", 0, "nonzero source revision")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("go_upb02_build.arguments")
	}
	if err := o.validate(); err != nil {
		return err
	}
	root, err := strictDirectory(o.project)
	if err != nil {
		return fmt.Errorf("go_upb02_build.project:%w", err)
	}
	if inside(root, o.out) {
		return fmt.Errorf("go_upb02_build.output_inside_project")
	}
	if _, err = os.Lstat(o.out); err == nil {
		return fmt.Errorf("go_upb02_build.output_exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	for label, p := range map[string]string{"execution_g1": o.executionG1, "execution_contract": o.executionContract, "package_v1": o.packageV1, "package_v2": o.packageV2, "project_v1": o.projectV1, "project_v2": o.projectV2, "project_v3": o.projectV3, "k0": o.k0, "compiler": o.compiler} {
		if err = strictRegular(p); err != nil {
			return fmt.Errorf("go_upb02_build.%s:%w", label, err)
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
		return fmt.Errorf("go_upb02_build.discover:%w", err)
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
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	result, err := goprojectpipeline.Build(ctx, goprojectpipeline.Input{Documents: goprovider.DocumentSnapshot{Revision: o.revision, ModulePath: o.module, PackagePath: o.pkg, Entry: o.entry, Files: files}, Sources: sources, Contracts: goprojectpipeline.Contracts{V1: v1, V2: v2, V3: v3}, ExecutionG1: g1, Compile: func(ctx context.Context, input []byte) ([]byte, error) { return compile(ctx, o, input) }})
	if err != nil {
		return err
	}
	return goprojectpipeline.Publish(o.out, result)
}

func (o options) validate() error {
	values := map[string]string{"project": o.project, "module": o.module, "package": o.pkg, "entry": o.entry, "out": o.out, "execution-g1": o.executionG1, "execution-contract": o.executionContract, "package-v1": o.packageV1, "package-v2": o.packageV2, "project-v1": o.projectV1, "project-v2": o.projectV2, "project-v3": o.projectV3, "k0": o.k0, "g1-compiler": o.compiler}
	for k, v := range values {
		if v == "" {
			return fmt.Errorf("go_upb02_build.flag_missing:%s", k)
		}
	}
	for _, p := range []string{o.project, o.out, o.executionG1, o.executionContract, o.packageV1, o.packageV2, o.projectV1, o.projectV2, o.projectV3, o.k0, o.compiler} {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p {
			return fmt.Errorf("go_upb02_build.absolute_path:%s", p)
		}
	}
	if o.revision == 0 {
		return fmt.Errorf("go_upb02_build.revision")
	}
	return nil
}

func strictDirectory(p string) (string, error) {
	i, err := os.Lstat(p)
	if err != nil {
		return "", err
	}
	if !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("not_plain_directory")
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil || resolved != p {
		return "", fmt.Errorf("symlink_component")
	}
	return p, nil
}
func strictRegular(p string) error {
	i, err := os.Lstat(p)
	if err != nil {
		return err
	}
	if !i.Mode().IsRegular() || i.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("not_plain_file")
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil || resolved != p {
		return fmt.Errorf("symlink_component")
	}
	return nil
}
func readRegular(p string, limit int64) ([]byte, error) {
	before, err := os.Lstat(p)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("go_upb02_build.read_regular")
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	after, err := f.Stat()
	pathAfter, pathErr := os.Lstat(p)
	if err != nil || pathErr != nil || !os.SameFile(before, after) || !os.SameFile(before, pathAfter) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("go_upb02_build.changed_file")
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("go_upb02_build.file_too_large")
	}
	return data, nil
}
func readProject(root string) (map[string]string, error) {
	out := map[string]string{}
	var total int64
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		i, err := d.Info()
		if err != nil {
			return err
		}
		if i.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("go_upb02_build.symlink:%s", p)
		}
		if d.IsDir() {
			return nil
		}
		if !i.Mode().IsRegular() {
			return fmt.Errorf("go_upb02_build.nonregular:%s", p)
		}
		if filepath.Ext(p) != ".go" || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		if len(out) >= maxFiles || i.Size() > maxFile {
			return fmt.Errorf("go_upb02_build.source_limit")
		}
		data, err := readRegular(p, maxFile)
		if err != nil {
			return err
		}
		total += int64(len(data))
		if total > maxTotal {
			return fmt.Errorf("go_upb02_build.total_limit")
		}
		rel, err := filepath.Rel(root, p)
		if err != nil || rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("go_upb02_build.path")
		}
		key := filepath.ToSlash(rel)
		if _, ok := out[key]; ok {
			return fmt.Errorf("go_upb02_build.path_duplicate")
		}
		out[key] = string(data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("go_upb02_build.no_sources")
	}
	return out, nil
}
func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && (rel == "." || !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}
func compile(ctx context.Context, o options, input []byte) ([]byte, error) {
	dir, err := os.MkdirTemp("", "seme-go-upb02-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	src, out := filepath.Join(dir, "input.g1"), filepath.Join(dir, "output.seme")
	if err = os.WriteFile(src, input, 0600); err != nil {
		return nil, err
	}
	want := sha256.Sum256(input)
	cmd := exec.CommandContext(ctx, o.k0, o.compiler, src, out)
	if raw, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go_upb02_build.compile:%w:%s", err, strings.TrimSpace(string(raw)))
	}
	unchanged, err := readRegular(src, maxArtifact)
	if err != nil || sha256.Sum256(unchanged) != want {
		return nil, fmt.Errorf("go_upb02_build.compiler_mutated_input")
	}
	data, err := readRegular(out, maxArtifact)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("go_upb02_build.compiler_output")
	}
	return data, err
}
