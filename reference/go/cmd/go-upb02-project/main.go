package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"seme.local/reference/goprojector"
	"seme.local/reference/projectbundle"
	"seme.local/reference/projectroundtrip"
	"seme.local/reference/projectsource"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb02-project:", err)
		os.Exit(65)
	}
}
func run(args []string) error {
	fs := flag.NewFlagSet("go-upb02-project", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("root", "", "original project root")
	construction := fs.String("construction", "", "canonical construction G1")
	artifact := fs.String("package-v2", "", "validated Package v2 artifact")
	module := fs.String("module", "", "native module identity")
	dest := fs.String("to", "", "new projected project destination")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected_arguments")
	}
	for n, v := range map[string]string{"root": *root, "construction": *construction, "package-v2": *artifact, "module": *module, "to": *dest} {
		if v == "" {
			return fmt.Errorf("flag_missing:%s", n)
		}
	}
	rootPath, err := strictDirectory(*root)
	if err != nil {
		return fmt.Errorf("root:%w", err)
	}
	if !filepath.IsAbs(*dest) {
		return fmt.Errorf("destination_not_absolute")
	}
	parent, err := strictDirectory(filepath.Dir(*dest))
	if err != nil {
		return fmt.Errorf("destination_parent:%w", err)
	}
	if filepath.Dir(filepath.Clean(*dest)) != parent {
		return fmt.Errorf("destination_parent")
	}
	if _, err = os.Lstat(*dest); err == nil {
		return fmt.Errorf("destination_exists")
	}
	if !os.IsNotExist(err) {
		return err
	}
	g1, err := readStrict(*construction)
	if err != nil {
		return err
	}
	v2, err := readStrict(*artifact)
	if err != nil {
		return err
	}
	projectedPackages, err := goprojector.ProjectPackagesV2(g1, v2)
	if err != nil {
		return err
	}
	policy := projectsource.Policy{TrackedExtensions: []string{".go"}, IgnoredPrefixes: []string{".seme-cache/"}, IgnoredSuffixes: []string{"_test.go"}, VendoredPrefixes: []string{"vendor/"}, GeneratedHeader: []byte("// Code generated "), MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 8 << 20}
	toolchain := projectsource.Toolchain{Language: "go", Toolchain: "go1.25.6-native/go1.26.0-provider", Profile: "linux-amd64/upb-02-bounded-v1", SemanticRevision: "go-upb02-package-v2"}
	snapshot, err := projectsource.Discover(rootPath, *module, toolchain, policy)
	if err != nil {
		return err
	}
	bundle, err := projectbundle.Capture(rootPath, snapshot, policy)
	if err != nil {
		return err
	}
	projected := map[string][]byte{}
	for packagePath, data := range projectedPackages {
		relative := ""
		if packagePath != *module {
			if !strings.HasPrefix(packagePath, *module+"/") {
				return fmt.Errorf("package_outside_module:%s", packagePath)
			}
			relative = strings.TrimPrefix(packagePath, *module+"/")
		}
		name := "seme_projected.go"
		if relative != "" {
			name = relative + "/seme_projected.go"
		}
		if _, exists := projected[name]; exists {
			return fmt.Errorf("projection_path_duplicate:%s", name)
		}
		projected[name] = data
	}
	_, err = projectroundtrip.PublishProjected(*dest, snapshot, bundle, projected, policy)
	return err
}
func strictDirectory(p string) (string, error) {
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("not_absolute")
	}
	clean := filepath.Clean(p)
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil || resolved != clean {
		return "", fmt.Errorf("not_plain")
	}
	info, err := os.Stat(clean)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("not_directory")
	}
	return clean, nil
}
func readStrict(p string) ([]byte, error) {
	if !filepath.IsAbs(p) {
		return nil, fmt.Errorf("input_not_absolute")
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil || resolved != filepath.Clean(p) {
		return nil, fmt.Errorf("input_not_plain")
	}
	pathBefore, err := os.Lstat(p)
	if err != nil || !pathBefore.Mode().IsRegular() {
		return nil, fmt.Errorf("input_not_regular")
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("input_not_regular")
	}
	const maximum = int64(64 << 20)
	if before.Size() < 0 || before.Size() > maximum {
		return nil, fmt.Errorf("input_size")
	}
	data, err := io.ReadAll(io.LimitReader(f, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != before.Size() {
		return nil, fmt.Errorf("input_changed")
	}
	after, err := f.Stat()
	pathAfter, pathErr := os.Lstat(p)
	if err != nil || pathErr != nil || !os.SameFile(before, after) || !os.SameFile(before, pathBefore) || !os.SameFile(before, pathAfter) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || pathBefore.Size() != pathAfter.Size() || !pathBefore.ModTime().Equal(pathAfter.ModTime()) {
		return nil, fmt.Errorf("input_changed")
	}
	return data, nil
}
