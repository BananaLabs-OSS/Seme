// Command go-project-v4-projector publishes an ordinary multi-package Go
// project from source-free canonical construction and Package-v4 authority.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprojector"
	"seme.local/reference/upb12authority"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "go-project-v4-projector:", err)
		os.Exit(65)
	}
}

func run(args []string) error {
	f := flag.NewFlagSet("go-project-v4-projector", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	construction := f.String("construction", "", "canonical construction G1")
	packageDetail := f.String("package-detail", "", "Package-v2 detail authority")
	packageV4 := f.String("package-v4", "", "Package-v4 authority")
	authority := f.String("authority", "", "closed source-free UPB12 authority")
	module := f.String("module", "", "Go module identity")
	out := f.String("out", "", "new project directory")
	foundation := f.String("foundation", "", "Foundation-v1 contract")
	execution := f.String("execution", "", "Execution-v36 contract")
	packages := f.String("packages", "", "Package-v4 contract")
	dependency := f.String("dependency", "", "Dependency-v1 contract")
	configuration := f.String("configuration", "", "Configuration-v3 contract")
	project := f.String("project", "", "Project-v8 contract")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	values := []*string{module, out, foundation, execution, packages, dependency, configuration, project}
	for _, value := range values {
		if *value == "" {
			return fmt.Errorf("arguments")
		}
	}
	nativeArtifactsComplete := *construction != "" && *packageDetail != "" && *packageV4 != ""
	if (*authority == "" && !nativeArtifactsComplete) || (*authority != "" && (*construction != "" || *packageDetail != "" || *packageV4 != "")) {
		return fmt.Errorf("authority_or_artifacts")
	}
	if strings.TrimSpace(*module) != *module || strings.HasPrefix(*module, "/") || strings.HasSuffix(*module, "/") || strings.Contains(*module, "..") {
		return fmt.Errorf("module")
	}
	if !filepath.IsAbs(*out) || filepath.Clean(*out) != *out {
		return fmt.Errorf("output")
	}
	parent, err := strictDirectory(filepath.Dir(*out))
	if err != nil {
		return fmt.Errorf("output_parent:%w", err)
	}
	if _, err = os.Lstat(*out); !os.IsNotExist(err) {
		return fmt.Errorf("output_exists")
	}
	read := func(path string) ([]byte, error) {
		value, e := strictFile(path)
		if e != nil {
			return nil, e
		}
		return value, nil
	}
	var g1, p2, p4 []byte
	if *authority != "" {
		if *construction != "" || *packageDetail != "" || *packageV4 != "" {
			return fmt.Errorf("authority_or_artifacts")
		}
		files, e := upb12authority.Load(*authority)
		if e != nil {
			return e
		}
		g1, p2, p4 = files["construction-v36.g1"], files["package-detail-v4.seme"], files["package-v4.seme"]
	} else {
		g1, err = read(*construction)
		if err != nil {
			return err
		}
		p2, err = read(*packageDetail)
		if err != nil {
			return err
		}
		p4, err = read(*packageV4)
		if err != nil {
			return err
		}
	}
	contractsRaw := make([][]byte, 6)
	for index, path := range []string{*foundation, *execution, *packages, *dependency, *configuration, *project} {
		contractsRaw[index], err = read(path)
		if err != nil {
			return err
		}
	}
	contracts, err := contractcatalog.ResolveProjectContractSetV8(contractsRaw[0], contractsRaw[1], contractsRaw[2], contractsRaw[3], contractsRaw[4], contractsRaw[5])
	if err != nil {
		return err
	}
	projected, err := goprojector.ProjectPackagesV4(g1, contracts, p2, p4)
	if err != nil {
		return err
	}
	files := map[string][]byte{"go.mod": []byte("module " + *module + "\n\ngo 1.26\n")}
	for identity, source := range projected {
		relative, ok := packagePath(*module, identity)
		if !ok {
			return fmt.Errorf("package_outside_module:%s", identity)
		}
		name := filepath.Join(relative, "seme_projected.go")
		if _, exists := files[name]; exists {
			return fmt.Errorf("path_collision:%s", name)
		}
		files[name] = source
	}
	stage, err := os.MkdirTemp(parent, ".seme-go-project-")
	if err != nil {
		return err
	}
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(stage)
		}
	}()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		target := filepath.Join(stage, name)
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err = os.WriteFile(target, files[name], 0600); err != nil {
			return err
		}
	}
	if err = os.Rename(stage, *out); err != nil {
		return err
	}
	published = true
	return nil
}

func packagePath(module, identity string) (string, bool) {
	if identity == module {
		return ".", true
	}
	prefix := module + "/"
	if !strings.HasPrefix(identity, prefix) {
		return "", false
	}
	relative := strings.TrimPrefix(identity, prefix)
	if relative == "" || filepath.Clean(relative) != relative || relative == ".." || strings.HasPrefix(relative, "../") || strings.Contains(relative, "/../") || strings.Contains(relative, "\\") {
		return "", false
	}
	return relative, true
}

func strictFile(path string) ([]byte, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, fmt.Errorf("path")
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil || real != path {
		return nil, fmt.Errorf("symlink")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64<<20 {
		return nil, fmt.Errorf("regular")
	}
	value, err := os.ReadFile(path)
	if err != nil || int64(len(value)) != info.Size() {
		return nil, fmt.Errorf("changed")
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, after) || !info.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("changed")
	}
	return bytes.Clone(value), nil
}
func strictDirectory(path string) (string, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return "", fmt.Errorf("path")
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil || real != path {
		return "", fmt.Errorf("symlink")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("directory")
	}
	return real, nil
}
