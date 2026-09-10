// Package goofflineclosure loads one bounded local edge and one pinned module
// from disk into the language-neutral Go dependency resolver without invoking
// Go or accessing a network.
package goofflineclosure

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"seme.local/reference/dependencyresolution"
	"seme.local/reference/godependencyresolver"
	"seme.local/reference/goprovider"
)

const maxFile int64 = 2 << 20
const maxTotal int64 = 32 << 20
const maxFiles = 1024
const maxZip int64 = 16 << 20

var modulePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~-]*(/[A-Za-z0-9][A-Za-z0-9._~-]*)+$`)
var versionPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

type Input struct {
	ProjectRoot, ProxyRoot, Module, Version, LocalFrom, LocalTo string
	Resolution                                                  goprovider.ResolutionManifest
}

func Load(in Input) (dependencyresolution.Closure, error) {
	root, err := directory(in.ProjectRoot)
	if err != nil {
		return dependencyresolution.Closure{}, fmt.Errorf("go_offline.project:%w", err)
	}
	proxy, err := directory(in.ProxyRoot)
	if err != nil {
		return dependencyresolution.Closure{}, fmt.Errorf("go_offline.proxy:%w", err)
	}
	if !modulePattern.MatchString(in.Module) || !versionPattern.MatchString(in.Version) {
		return dependencyresolution.Closure{}, fmt.Errorf("go_offline.coordinate")
	}
	goMod, err := read(filepath.Join(root, "go.mod"), maxFile)
	if err != nil {
		return dependencyresolution.Closure{}, fmt.Errorf("go_offline.mod:%w", err)
	}
	goSum, err := read(filepath.Join(root, "go.sum"), maxFile)
	if err != nil {
		return dependencyresolution.Closure{}, fmt.Errorf("go_offline.sum:%w", err)
	}
	packages := map[string]goprovider.ResolvedPackage{}
	imports := map[string][]string{}
	ownedFiles := map[string]string{}
	for _, p := range in.Resolution.Packages {
		if _, ok := packages[p.Name]; ok {
			return dependencyresolution.Closure{}, fmt.Errorf("go_offline.package_duplicate")
		}
		packages[p.Name] = p
		for _, name := range p.Files {
			if owner, exists := ownedFiles[name]; exists {
				return dependencyresolution.Closure{}, fmt.Errorf("go_offline.file_owned_twice:%s:%s", owner, name)
			}
			ownedFiles[name] = p.Name
		}
		for _, im := range p.Imports {
			if im.Local {
				imports[p.Name] = append(imports[p.Name], im.ResolvedPath)
			}
		}
	}
	from, fok := packages[in.LocalFrom]
	to, tok := packages[in.LocalTo]
	if !fok || !tok || in.LocalFrom == in.LocalTo {
		return dependencyresolution.Closure{}, fmt.Errorf("go_offline.local_packages")
	}
	edge := false
	for _, im := range from.Imports {
		if im.Local && im.ResolvedPath == in.LocalTo {
			edge = true
		}
	}
	if !edge {
		return dependencyresolution.Closure{}, fmt.Errorf("go_offline.local_edge")
	}
	localFiles := map[string][]byte{}
	if err := verifyPackageFiles(root, to.Files); err != nil {
		return dependencyresolution.Closure{}, err
	}
	for _, name := range to.Files {
		if !relative(name) {
			return dependencyresolution.Closure{}, fmt.Errorf("go_offline.local_path")
		}
		data, er := readContained(root, name, maxFile)
		if er != nil {
			return dependencyresolution.Closure{}, er
		}
		localFiles[name] = data
	}
	if len(localFiles) == 0 {
		return dependencyresolution.Closure{}, fmt.Errorf("go_offline.local_empty")
	}
	localDigest, _, err := godependencyresolver.AnalyzeTree(localFiles)
	if err != nil {
		return dependencyresolution.Closure{}, err
	}
	zipPath := filepath.Join(proxy, filepath.FromSlash(in.Module), "@v", in.Version+".zip")
	externalFiles, err := readModuleZip(zipPath, in.Module, in.Version)
	if err != nil {
		return dependencyresolution.Closure{}, err
	}
	externalDigest, _, err := godependencyresolver.AnalyzeTree(externalFiles)
	if err != nil {
		return dependencyresolution.Closure{}, err
	}
	return godependencyresolver.Resolve(godependencyresolver.Input{GoMod: goMod, GoSum: goSum, Local: godependencyresolver.Local{From: in.LocalFrom, To: in.LocalTo, Source: "workspace:" + in.LocalTo, Digest: localDigest, Files: localFiles}, External: godependencyresolver.External{Module: in.Module, Version: in.Version, Source: "go-proxy:" + in.Module + "@" + in.Version, Digest: externalDigest, Files: externalFiles}, PackageImports: imports})
}

func verifyPackageFiles(root string, listed []string) error {
	if len(listed) == 0 {
		return fmt.Errorf("go_offline.local_empty")
	}
	dir := path.Dir(listed[0])
	directoryPath := filepath.Join(root, filepath.FromSlash(dir))
	resolved, err := filepath.EvalSymlinks(directoryPath)
	if err != nil || resolved != directoryPath {
		return fmt.Errorf("go_offline.local_directory_symlink")
	}
	want := map[string]bool{}
	for _, name := range listed {
		if !relative(name) || path.Dir(name) != dir || want[name] {
			return fmt.Errorf("go_offline.local_file_set")
		}
		want[name] = true
	}
	entries, err := os.ReadDir(directoryPath)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		info, er := entry.Info()
		if er != nil {
			return er
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("go_offline.local_symlink")
		}
		if entry.IsDir() {
			continue
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("go_offline.local_nonregular")
		}
		if filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		name := path.Join(dir, entry.Name())
		if !want[name] {
			return fmt.Errorf("go_offline.local_unlisted:%s", name)
		}
		seen[name] = true
	}
	if len(seen) != len(want) {
		return fmt.Errorf("go_offline.local_file_set")
	}
	return nil
}

func readModuleZip(name, module, version string) (map[string][]byte, error) {
	raw, err := read(name, maxZip)
	if err != nil {
		return nil, fmt.Errorf("go_offline.zip:%w", err)
	}
	z, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, fmt.Errorf("go_offline.zip_format")
	}
	if len(z.File) == 0 || len(z.File) > maxFiles {
		return nil, fmt.Errorf("go_offline.zip_count")
	}
	prefix := module + "@" + version + "/"
	out := map[string][]byte{}
	var total int64
	for _, f := range z.File {
		if !strings.HasPrefix(f.Name, prefix) {
			return nil, fmt.Errorf("go_offline.zip_prefix:%s", f.Name)
		}
		rel := strings.TrimPrefix(f.Name, prefix)
		if f.FileInfo().IsDir() {
			if rel != "" || f.Name != prefix {
				return nil, fmt.Errorf("go_offline.zip_directory")
			}
			continue
		}
		if !relative(rel) || f.Mode()&os.ModeSymlink != 0 || !f.Mode().IsRegular() || f.UncompressedSize64 > uint64(maxFile) {
			return nil, fmt.Errorf("go_offline.zip_entry:%s", f.Name)
		}
		if _, ok := out[f.Name]; ok {
			return nil, fmt.Errorf("go_offline.zip_duplicate:%s", f.Name)
		}
		r, er := f.Open()
		if er != nil {
			return nil, er
		}
		data, er := io.ReadAll(io.LimitReader(r, maxFile+1))
		closeErr := r.Close()
		if er != nil || closeErr != nil || int64(len(data)) > maxFile || uint64(len(data)) != f.UncompressedSize64 {
			return nil, fmt.Errorf("go_offline.zip_read:%s", f.Name)
		}
		total += int64(len(data))
		if total > maxTotal {
			return nil, fmt.Errorf("go_offline.zip_total")
		}
		out[f.Name] = data
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("go_offline.zip_empty")
	}
	return out, nil
}

func directory(p string) (string, error) {
	if !filepath.IsAbs(p) || filepath.Clean(p) != p {
		return "", fmt.Errorf("absolute")
	}
	i, err := os.Lstat(p)
	if err != nil {
		return "", err
	}
	if !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("plain_directory")
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil || resolved != p {
		return "", fmt.Errorf("symlink_component")
	}
	return p, nil
}
func readContained(root, name string, limit int64) ([]byte, error) {
	joined := filepath.Join(root, filepath.FromSlash(name))
	rel, err := filepath.Rel(root, joined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("go_offline.containment")
	}
	resolved, err := filepath.EvalSymlinks(joined)
	if err != nil || resolved != joined {
		return nil, fmt.Errorf("go_offline.symlink_component")
	}
	return read(joined, limit)
}
func read(name string, limit int64) ([]byte, error) {
	before, err := os.Lstat(name)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("plain_file")
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	opened, err := f.Stat()
	after, pathErr := os.Lstat(name)
	if err != nil || pathErr != nil || !os.SameFile(before, opened) || !os.SameFile(before, after) || after.Mode()&os.ModeSymlink != 0 || before.Size() != opened.Size() || before.Size() != after.Size() || int64(len(data)) != before.Size() || !before.ModTime().Equal(opened.ModTime()) || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("changed_file")
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file_limit")
	}
	return data, nil
}
func relative(x string) bool {
	return x != "" && x != "." && x != ".." && !strings.HasPrefix(x, "/") && !strings.Contains(x, "\\") && !strings.Contains(x, "//") && path.Clean(x) == x && !strings.HasPrefix(x, "../")
}
