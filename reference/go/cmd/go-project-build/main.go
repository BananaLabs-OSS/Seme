// Command go-project-build is a bounded conformance adapter for the Go project
// bridge. It is intentionally not a general-purpose or production publisher.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
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
	"seme.local/reference/goprovider"
	"seme.local/reference/projectbuild"
)

const (
	maxSourceFiles = 1024
	maxSourceBytes = 2 << 20
	maxTotalBytes  = 32 << 20
	maxWireBytes   = 64 << 20
)

type options struct {
	project, module, pkg, entry, out                string
	executionG1, executionContract, packageContract string
	projectContract, k0, compiler, validator        string
	revision                                        uint64
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(parent context.Context) error {
	var o options
	flag.StringVar(&o.project, "project", "", "controlled project directory")
	flag.StringVar(&o.module, "module", "", "module identity")
	flag.StringVar(&o.pkg, "package", "", "root package identity")
	flag.StringVar(&o.entry, "entry", "", "entry function")
	flag.StringVar(&o.out, "out", "", "new final .seme output")
	flag.StringVar(&o.executionG1, "execution-g1", "", "Execution v35 G1")
	flag.StringVar(&o.executionContract, "execution-contract", "", "Execution v35 .seme")
	flag.StringVar(&o.packageContract, "package-contract", "", "Package v1 .seme")
	flag.StringVar(&o.projectContract, "project-contract", "", "Project v1 .seme")
	flag.StringVar(&o.k0, "k0", "", "absolute frozen K0 executable")
	flag.StringVar(&o.compiler, "g1-compiler", "", "absolute frozen g1-compiler.k0")
	flag.StringVar(&o.validator, "kernel-validator", "", "absolute frozen wire validator.k0")
	flag.Uint64Var(&o.revision, "revision", 0, "nonzero client revision")
	flag.Parse()
	if err := o.validate(); err != nil {
		return err
	}

	root, err := strictDirectory(o.project)
	if err != nil {
		return fmt.Errorf("go_project_build.project:%w", err)
	}
	destination, err := filepath.Abs(o.out)
	if err != nil {
		return err
	}
	parentDir, err := strictDirectory(filepath.Dir(destination))
	if err != nil {
		return fmt.Errorf("go_project_build.output_parent:%w", err)
	}
	destination = filepath.Join(parentDir, filepath.Base(destination))
	if inside(root, destination) {
		return fmt.Errorf("go_project_build.output_inside_project")
	}
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("go_project_build.output_exists")
	} else if !os.IsNotExist(err) {
		return err
	}

	for label, path := range map[string]string{
		"execution_g1": o.executionG1, "execution_contract": o.executionContract,
		"package_contract": o.packageContract, "project_contract": o.projectContract,
		"k0": o.k0, "compiler": o.compiler, "validator": o.validator,
	} {
		if err := strictRegular(path); err != nil {
			return fmt.Errorf("go_project_build.%s:%w", label, err)
		}
	}
	files, err := readProject(root)
	if err != nil {
		return err
	}
	eg1, err := readRegular(o.executionG1, maxWireBytes)
	if err != nil {
		return err
	}
	ec, err := readRegular(o.executionContract, maxWireBytes)
	if err != nil {
		return err
	}
	pc, err := readRegular(o.packageContract, maxWireBytes)
	if err != nil {
		return err
	}
	prc, err := readRegular(o.projectContract, maxWireBytes)
	if err != nil {
		return err
	}
	contracts, err := contractcatalog.ResolveProjectContractSet(ec, pc, prc)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	result, err := projectbuild.Build(ctx, goprovider.DocumentSnapshot{
		Revision: o.revision, ModulePath: o.module, PackagePath: o.pkg,
		Entry: o.entry, Files: files,
	}, contracts, eg1, func(ctx context.Context, g1 []byte) ([]byte, error) {
		return compileG1(ctx, o, g1)
	})
	if err != nil {
		return err
	}
	return publishCreateOnly(parentDir, destination, result.Artifact)
}

func (o options) validate() error {
	values := map[string]string{
		"project": o.project, "module": o.module, "package": o.pkg, "entry": o.entry,
		"out": o.out, "execution-g1": o.executionG1, "execution-contract": o.executionContract,
		"package-contract": o.packageContract, "project-contract": o.projectContract,
		"k0": o.k0, "g1-compiler": o.compiler, "kernel-validator": o.validator,
	}
	for name, value := range values {
		if value == "" {
			return fmt.Errorf("go_project_build.flag_missing:%s", name)
		}
	}
	for _, path := range []string{o.project, o.out, o.executionG1, o.executionContract, o.packageContract, o.projectContract, o.k0, o.compiler, o.validator} {
		if !filepath.IsAbs(path) {
			return fmt.Errorf("go_project_build.absolute_path:%s", path)
		}
	}
	if o.revision == 0 {
		return fmt.Errorf("go_project_build.revision")
	}
	return nil
}

func strictDirectory(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("not_plain_directory")
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	if resolved != absolute {
		return "", fmt.Errorf("symlink_component")
	}
	return absolute, nil
}

func strictRegular(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("not_plain_file")
	}
	return nil
}

func readRegular(path string, limit int64) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil || before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("go_project_build.read_regular:%s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return nil, fmt.Errorf("go_project_build.changed_file:%s", path)
	}
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("go_project_build.file_too_large:%s", path)
	}
	return b, nil
}

func readProject(root string) (map[string]string, error) {
	files := map[string]string{}
	var total int64
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("go_project_build.symlink:%s", path)
		}
		if d.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("go_project_build.nonregular:%s", path)
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if len(files) >= maxSourceFiles || info.Size() > maxSourceBytes {
			return fmt.Errorf("go_project_build.source_limit")
		}
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == "." || filepath.IsAbs(relative) || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("go_project_build.path")
		}
		data, err := readRegular(path, maxSourceBytes)
		if err != nil {
			return err
		}
		total += int64(len(data))
		if total > maxTotalBytes {
			return fmt.Errorf("go_project_build.total_source_limit")
		}
		name := filepath.ToSlash(relative)
		if _, duplicate := files[name]; duplicate {
			return fmt.Errorf("go_project_build.path_duplicate:%s", name)
		}
		files[name] = string(data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("go_project_build.no_sources")
	}
	return files, nil
}

func compileG1(ctx context.Context, o options, g1 []byte) ([]byte, error) {
	dir, err := os.MkdirTemp("", "seme-go-project-build-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	input, output := filepath.Join(dir, "input.g1"), filepath.Join(dir, "output.seme")
	if err := os.WriteFile(input, g1, 0o600); err != nil {
		return nil, err
	}
	wantInput := sha256.Sum256(g1)
	for _, argv := range [][]string{{o.compiler, input, output}, {o.validator, output}} {
		command := exec.CommandContext(ctx, o.k0, argv...)
		command.Stdout, command.Stderr = os.Stderr, os.Stderr
		if err := command.Run(); err != nil {
			return nil, err
		}
	}
	gotInput, err := readRegular(input, maxWireBytes)
	if err != nil || sha256.Sum256(gotInput) != wantInput {
		return nil, fmt.Errorf("go_project_build.compiler_mutated_input")
	}
	return readRegular(output, maxWireBytes)
}

func publishCreateOnly(parent, destination string, artifact []byte) error {
	temporary, err := os.CreateTemp(parent, ".seme-project-*.tmp")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return err
	}
	if _, err := io.Copy(temporary, bytes.NewReader(artifact)); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Link(name, destination); err != nil {
		return err
	}
	if err := os.Remove(name); err != nil {
		return err
	}
	directory, err := os.Open(parent)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func inside(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
