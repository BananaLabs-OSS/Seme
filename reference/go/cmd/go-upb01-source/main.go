// Command go-upb01-source is the bounded UPB-01 source-inventory proof adapter.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectbundle"
	"seme.local/reference/projectroundtrip"
	"seme.local/reference/projectsource"
	"seme.local/reference/sourceinventory"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb01-source:", err)
		os.Exit(65)
	}
}
func run() error {
	var root, project, executionContract, packageContract, projectContract, out, copyTo string
	flag.StringVar(&root, "root", "", "project root")
	flag.StringVar(&project, "project", "", "validated semantic project")
	flag.StringVar(&executionContract, "execution-contract", "", "Execution v35 contract")
	flag.StringVar(&packageContract, "package-contract", "", "Package v1 contract")
	flag.StringVar(&projectContract, "project-contract", "", "Project v2 contract")
	flag.StringVar(&out, "out", "", "inventory output")
	flag.StringVar(&copyTo, "copy", "", "optional exact preserved-source destination")
	flag.Parse()
	for _, required := range []struct{ name, value string }{{"root", root}, {"project", project}, {"execution-contract", executionContract}, {"package-contract", packageContract}, {"project-contract", projectContract}, {"out", out}} {
		if required.value == "" {
			return fmt.Errorf("flag_missing:%s", required.name)
		}
	}
	if _, e := strictDirectory(root); e != nil {
		return fmt.Errorf("root:%w", e)
	}
	if copyTo != "" {
		if !filepath.IsAbs(copyTo) {
			return fmt.Errorf("copy_not_absolute")
		}
		if _, e := strictDirectory(filepath.Dir(copyTo)); e != nil {
			return fmt.Errorf("copy_parent:%w", e)
		}
		if _, e := os.Lstat(copyTo); e == nil {
			return fmt.Errorf("copy_exists")
		} else if !os.IsNotExist(e) {
			return e
		}
	}
	if !filepath.IsAbs(out) {
		return fmt.Errorf("output_not_absolute")
	}
	if _, e := strictDirectory(filepath.Dir(out)); e != nil {
		return fmt.Errorf("output_parent:%w", e)
	}
	if _, e := os.Lstat(out); e == nil {
		return fmt.Errorf("output_exists")
	} else if !os.IsNotExist(e) {
		return e
	}
	execution, e := readStrict(executionContract)
	if e != nil {
		return e
	}
	packages, e := readStrict(packageContract)
	if e != nil {
		return e
	}
	projectSchema, e := readStrict(projectContract)
	if e != nil {
		return e
	}
	contracts, e := contractcatalog.ResolveProjectContractSetV2(execution, packages, projectSchema)
	if e != nil {
		return e
	}
	semantic, e := readStrict(project)
	if e != nil {
		return e
	}
	policy := projectsource.Policy{TrackedExtensions: []string{".go"}, IgnoredPrefixes: []string{".seme-cache/"}, IgnoredSuffixes: []string{"_test.go"}, VendoredPrefixes: []string{"vendor/"}, GeneratedHeader: []byte("// Code generated "), MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 8 << 20}
	toolchain := projectsource.Toolchain{Language: "go", Toolchain: "go1.25.6", Profile: "upb-01-bounded-v1", SemanticRevision: "core-execution-v35"}
	snapshot, e := projectsource.Discover(root, "example.test/go-project-build-v1", toolchain, policy)
	if e != nil {
		return e
	}
	classes := map[projectsource.Class]bool{}
	for _, u := range snapshot.Units {
		classes[u.Class] = true
	}
	for _, c := range []projectsource.Class{projectsource.Tracked, projectsource.Ignored, projectsource.Generated, projectsource.Vendored, projectsource.Opaque} {
		if !classes[c] {
			return fmt.Errorf("class_missing:%s", c)
		}
	}
	inventory, e := sourceinventory.Emit(contracts.Project(), semantic, snapshot)
	if e != nil {
		return e
	}
	if e = sourceinventory.Validate(contracts.Project(), semantic, inventory); e != nil {
		return e
	}
	bundle, e := projectbundle.Capture(root, snapshot, policy)
	if e != nil {
		return e
	}
	if e = projectbundle.Validate(snapshot, bundle); e != nil {
		return e
	}
	if copyTo != "" {
		if e = projectroundtrip.CopyVerified(root, copyTo, snapshot, policy); e != nil {
			return e
		}
	}
	return writeAtomic(out, inventory)
}
func writeAtomic(path string, data []byte) (err error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(absolute), ".seme-upb01-inventory-")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if _, err = os.Lstat(absolute); err == nil {
		return fmt.Errorf("output_exists")
	}
	if !os.IsNotExist(err) {
		return err
	}
	if err = os.Link(name, absolute); err != nil {
		return err
	}
	return os.Remove(name)
}

func strictDirectory(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("not_absolute")
	}
	clean := filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return "", err
	}
	if resolved != clean {
		return "", fmt.Errorf("symlink_path")
	}
	info, err := os.Lstat(clean)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("not_plain_directory")
	}
	return clean, nil
}
func readStrict(path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("input_not_absolute")
	}
	clean := filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return nil, err
	}
	if resolved != clean {
		return nil, fmt.Errorf("input_symlink_path")
	}
	before, err := os.Lstat(clean)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("input_not_regular")
	}
	if before.Size() > 64<<20 {
		return nil, fmt.Errorf("input_too_large")
	}
	file, err := os.Open(clean)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, opened) {
		return nil, fmt.Errorf("input_changed")
	}
	data, err := io.ReadAll(io.LimitReader(file, (64<<20)+1))
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	pathAfter, pathErr := os.Lstat(clean)
	resolvedAfter, resolveErr := filepath.EvalSymlinks(clean)
	if err != nil || pathErr != nil || resolveErr != nil || resolvedAfter != clean || pathAfter.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, after) || !os.SameFile(before, pathAfter) || int64(len(data)) != after.Size() || !before.ModTime().Equal(after.ModTime()) || len(data) > 64<<20 {
		return nil, fmt.Errorf("input_changed")
	}
	return data, nil
}
