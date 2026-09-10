package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissingExtraAndRelativeRejectWithoutOutput(t *testing.T) {
	if err := run(context.Background(), nil, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "flag_missing") {
		t.Fatalf("err=%v", err)
	}
	out := filepath.Join(t.TempDir(), "bundle")
	args := fixtureArgs(t, out)
	args[1] = "relative"
	if err := run(context.Background(), args, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "absolute_path") {
		t.Fatalf("err=%v", err)
	}
	assertAbsent(t, out)
	args = fixtureArgs(t, out)
	args = append(args, "extra")
	if err := run(context.Background(), args, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "arguments") {
		t.Fatalf("err=%v", err)
	}
	assertAbsent(t, out)
}

func TestProjectProxySymlinkAndContractTamperRejectWithoutOutput(t *testing.T) {
	out := filepath.Join(t.TempDir(), "bundle")
	args := fixtureArgs(t, out)
	root := t.TempDir()
	link := filepath.Join(root, "project")
	if err := os.Symlink(args[1], link); err != nil {
		t.Fatal(err)
	}
	args[1] = link
	if err := run(context.Background(), args, &bytes.Buffer{}); err == nil {
		t.Fatal("project symlink accepted")
	}
	assertAbsent(t, out)
	args = fixtureArgs(t, out)
	proxyIndex := index(args, "-proxy") + 1
	link = filepath.Join(root, "proxy")
	if err := os.Symlink(args[proxyIndex], link); err != nil {
		t.Fatal(err)
	}
	args[proxyIndex] = link
	if err := run(context.Background(), args, &bytes.Buffer{}); err == nil {
		t.Fatal("proxy symlink accepted")
	}
	assertAbsent(t, out)
	args = fixtureArgs(t, out)
	contractIndex := index(args, "-project-v4") + 1
	forged := filepath.Join(root, "v4.seme")
	b, err := os.ReadFile(args[contractIndex])
	if err != nil {
		t.Fatal(err)
	}
	b[len(b)-1] ^= 1
	if err = os.WriteFile(forged, b, 0600); err != nil {
		t.Fatal(err)
	}
	args[contractIndex] = forged
	if err = run(context.Background(), args, &bytes.Buffer{}); err == nil {
		t.Fatal("tampered contract accepted")
	}
	assertAbsent(t, out)
}

func TestRunPublishesCompleteBundleAndPreservesExisting(t *testing.T) {
	out := filepath.Join(t.TempDir(), "bundle")
	args := fixtureArgs(t, out)
	if err := run(context.Background(), args, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"construction.g1", "project-v1.seme", "inventory-v2.seme", "package-v2.seme", "project-v3.seme", "dependency-v1.seme", "project-v4.seme", "COMPLETE.sha256"} {
		if b, e := os.ReadFile(filepath.Join(out, name)); e != nil || len(b) == 0 {
			t.Fatalf("%s: %v", name, e)
		}
	}
	marker := filepath.Join(out, "project-v4.seme")
	before, _ := os.ReadFile(marker)
	if err := run(context.Background(), args, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "output_exists") {
		t.Fatalf("err=%v", err)
	}
	after, _ := os.ReadFile(marker)
	if !bytes.Equal(before, after) {
		t.Fatal("existing output changed")
	}
}

func fixtureArgs(t *testing.T, out string) []string {
	t.Helper()
	repo, err := filepath.Abs("../../../../")
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(repo, "fixtures/go-upb03-dependency-v1")
	return []string{"-project", project, "-proxy", filepath.Join(repo, "fixtures/go-upb03-offline-proxy"), "-module", "example.test/go-upb03-dependency-v1", "-package", "example.test/go-upb03-dependency-v1/application", "-entry", "Apply", "-out", out, "-revision", "1", "-dependency", "example.test/seme/checksum", "-version", "v1.2.3", "-local-from", "example.test/go-upb03-dependency-v1/application", "-local-to", "example.test/go-upb03-dependency-v1/model", "-execution-g1", filepath.Join(repo, "modules/execution/v35/module.g1"), "-execution-contract", filepath.Join(repo, "modules/execution/v35/module.seme"), "-package-v1", filepath.Join(repo, "modules/package/v1/module.seme"), "-package-v2", filepath.Join(repo, "modules/package/v2/module.seme"), "-project-v1", filepath.Join(repo, "modules/project/v1/module.seme"), "-project-v2", filepath.Join(repo, "modules/project/v2/module.seme"), "-project-v3", filepath.Join(repo, "modules/project/v3/module.seme"), "-project-v4", filepath.Join(repo, "modules/project/v4/module.seme"), "-dependency-v1", filepath.Join(repo, "modules/dependency/v1/module.seme"), "-k0", filepath.Join(repo, "bootstrap/seme-k0-linux-amd64"), "-g1-compiler", filepath.Join(repo, "compiler/g1-compiler.k0")}
}
func index(xs []string, want string) int {
	for i, x := range xs {
		if x == want {
			return i
		}
	}
	return -1
}
func assertAbsent(t *testing.T, p string) {
	t.Helper()
	if _, e := os.Lstat(p); !os.IsNotExist(e) {
		t.Fatalf("output exists: %v", e)
	}
}
