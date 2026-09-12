package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprojector"
	"seme.local/reference/upb12authority"
	"strings"
)

func main() {
	authority := flag.String("authority", "", "source-free authority")
	projectRoot := flag.String("project-root", "", "projected Go root")
	module := flag.String("module", "", "Go module identity")
	foundation := flag.String("foundation", "", "Foundation contract")
	execution := flag.String("execution", "", "Execution contract")
	packages := flag.String("packages", "", "Package contract")
	dependency := flag.String("dependency", "", "Dependency contract")
	configuration := flag.String("configuration", "", "Configuration contract")
	project := flag.String("project", "", "Project-v8 contract")
	out := flag.String("out", "", "new canonical G1")
	nativeTest := flag.String("native-test", "", "trusted projected native test")
	flag.Parse()
	if flag.NArg() != 0 || *authority == "" || *projectRoot == "" || *module == "" || *nativeTest == "" || *out == "" {
		fatal("arguments")
	}
	files, err := upb12authority.Load(*authority)
	if err != nil {
		fatal(err)
	}
	read := func(name string) []byte {
		value, e := strictRead(name)
		if e != nil {
			fatal(e)
		}
		return value
	}
	contracts, err := contractcatalog.ResolveProjectContractSetV8(read(*foundation), read(*execution), read(*packages), read(*dependency), read(*configuration), read(*project))
	if err != nil {
		fatal(err)
	}
	expected, err := goprojector.ProjectPackagesV4Universal(files["construction-v36.g1"], contracts, files["package-detail-v4.seme"], files["package-v4.seme"])
	if err != nil {
		fatal(err)
	}
	want := map[string][]byte{}
	for identity, value := range expected {
		relative, ok := packageRelative(*module, identity)
		if !ok {
			fatal("package_identity")
		}
		want[filepath.Join(relative, "seme_projected.go")] = value
	}
	err = filepath.WalkDir(*projectRoot, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == *projectRoot {
			return nil
		}
		relative, e := filepath.Rel(*projectRoot, name)
		if e != nil {
			return e
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink")
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("special")
		}
		if strings.HasSuffix(relative, ".go") && !strings.HasSuffix(relative, "_test.go") {
			expectedValue, ok := want[relative]
			if !ok {
				return fmt.Errorf("unexpected_source:%s", relative)
			}
			actual, e := strictRead(name)
			if e != nil || !bytes.Equal(actual, expectedValue) {
				return fmt.Errorf("source_mismatch:%s", relative)
			}
			delete(want, relative)
		}
		return nil
	})
	if err != nil || len(want) != 0 {
		fatal(fmt.Errorf("project:%w:missing=%d", err, len(want)))
	}
	gotTest, err := strictRead(filepath.Join(*projectRoot, "controlled", "upb12_native_test.go"))
	if err != nil {
		fatal(err)
	}
	wantTest, err := strictRead(*nativeTest)
	if err != nil || !bytes.Equal(gotTest, wantTest) {
		fatal("native_test")
	}
	if !filepath.IsAbs(*out) || filepath.Clean(*out) != *out {
		fatal("output")
	}
	handle, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		fatal(err)
	}
	keep := false
	defer func() {
		handle.Close()
		if !keep {
			os.Remove(*out)
		}
	}()
	if _, err = handle.Write(files["construction-v36.g1"]); err != nil {
		fatal(err)
	}
	if err = handle.Sync(); err != nil {
		fatal(err)
	}
	if err = handle.Close(); err != nil {
		fatal(err)
	}
	keep = true
}
func packageRelative(module, identity string) (string, bool) {
	if identity == module {
		return ".", true
	}
	prefix := module + "/"
	if !strings.HasPrefix(identity, prefix) {
		return "", false
	}
	value := strings.TrimPrefix(identity, prefix)
	if value == "" || filepath.Clean(value) != value || value == ".." || strings.HasPrefix(value, "../") {
		return "", false
	}
	return value, true
}
func strictRead(name string) ([]byte, error) {
	real, err := filepath.EvalSymlinks(name)
	if err != nil || real != name {
		return nil, fmt.Errorf("symlink")
	}
	before, err := os.Lstat(name)
	if err != nil || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > 64<<20 {
		return nil, fmt.Errorf("regular")
	}
	value, err := os.ReadFile(name)
	if err != nil || int64(len(value)) != before.Size() {
		return nil, fmt.Errorf("changed")
	}
	after, err := os.Lstat(name)
	if err != nil || !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("changed")
	}
	return value, nil
}
func fatal(value any) { fmt.Fprintln(os.Stderr, "upb12-go-relift:", value); os.Exit(1) }
