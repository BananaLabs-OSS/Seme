// Command go-upb04-project projects an authenticated Package v3 graph while
// preserving every detached non-semantic source unit from the original tree.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprojector"
	"seme.local/reference/projectbundle"
	"seme.local/reference/projectroundtrip"
	"seme.local/reference/projectsource"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb04-project:", err)
		os.Exit(65)
	}
}

func run(args []string) error {
	f := flag.NewFlagSet("go-upb04-project", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	root := f.String("root", "", "absolute original project root")
	construction := f.String("construction", "", "canonical construction G1")
	packageV2 := f.String("package-v2", "", "Package v2 instance")
	packageV3 := f.String("package-v3", "", "Package v3 instance")
	module := f.String("module", "", "module identity")
	dest := f.String("to", "", "new projection directory")
	execution := f.String("execution-contract", "", "Execution v35 contract")
	packageContract := f.String("package-v3-contract", "", "Package v3 contract")
	dependency := f.String("dependency-contract", "", "Dependency v1 contract")
	project := f.String("project-v5-contract", "", "Project v5 contract")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected_arguments")
	}
	values := map[string]string{"root": *root, "construction": *construction, "package-v2": *packageV2, "package-v3": *packageV3, "module": *module, "to": *dest, "execution-contract": *execution, "package-v3-contract": *packageContract, "dependency-contract": *dependency, "project-v5-contract": *project}
	for name, value := range values {
		if value == "" {
			return fmt.Errorf("flag_missing:%s", name)
		}
	}
	rootPath, err := strictDirectory(*root)
	if err != nil {
		return fmt.Errorf("root:%w", err)
	}
	if !filepath.IsAbs(*dest) {
		return fmt.Errorf("destination_not_absolute")
	}
	if inside(rootPath, filepath.Clean(*dest)) {
		return fmt.Errorf("destination_inside_root")
	}
	parent, err := strictDirectory(filepath.Dir(*dest))
	if err != nil || filepath.Dir(filepath.Clean(*dest)) != parent {
		return fmt.Errorf("destination_parent")
	}
	if _, err = os.Lstat(*dest); err == nil {
		return fmt.Errorf("destination_exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	read := func(p string) ([]byte, error) { return readStrict(p) }
	g1, err := read(*construction)
	if err != nil {
		return err
	}
	p2, err := read(*packageV2)
	if err != nil {
		return err
	}
	p3, err := read(*packageV3)
	if err != nil {
		return err
	}
	ec, err := read(*execution)
	if err != nil {
		return err
	}
	pc, err := read(*packageContract)
	if err != nil {
		return err
	}
	dc, err := read(*dependency)
	if err != nil {
		return err
	}
	vc, err := read(*project)
	if err != nil {
		return err
	}
	contracts, err := contractcatalog.ResolveProjectContractSetV5(ec, pc, dc, vc)
	if err != nil {
		return err
	}
	packages, err := goprojector.ProjectPackagesV3(g1, contracts, p2, p3)
	if err != nil {
		return err
	}
	policy := projectsource.Policy{TrackedExtensions: []string{".go"}, IgnoredPrefixes: []string{".seme-cache/"}, IgnoredSuffixes: []string{"_test.go"}, VendoredPrefixes: []string{"vendor/"}, GeneratedHeader: []byte("// Code generated "), MaxFiles: 128, MaxFileBytes: 1 << 20, MaxTotalBytes: 8 << 20}
	tool := projectsource.Toolchain{Language: "go", Toolchain: "go1.25.6-native/go1.26.0-provider", Profile: "linux-amd64/upb-04-bounded-v1", SemanticRevision: "go-upb04-package-v3"}
	snapshot, err := projectsource.Discover(rootPath, *module, tool, policy)
	if err != nil {
		return err
	}
	bundle, err := projectbundle.Capture(rootPath, snapshot, policy)
	if err != nil {
		return err
	}
	projected := map[string][]byte{}
	for packagePath, data := range packages {
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

func inside(root, p string) bool {
	relative, err := filepath.Rel(root, p)
	return err == nil && (relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))))
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
	info, err := os.Lstat(clean)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
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
	before, err := os.Lstat(p)
	if err != nil || !before.Mode().IsRegular() || before.Size() < 0 || before.Size() > 64<<20 {
		return nil, fmt.Errorf("input_not_regular")
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("input_changed")
	}
	data, err := io.ReadAll(io.LimitReader(f, 64<<20+1))
	if err != nil || int64(len(data)) != opened.Size() {
		return nil, fmt.Errorf("input_changed")
	}
	after, err := os.Lstat(p)
	if err != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("input_changed")
	}
	return data, nil
}
