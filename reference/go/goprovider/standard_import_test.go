package goprovider

import (
	"os"
	"testing"
)

func TestStandardImportsDoNotDependOnHostModuleDirectory(t *testing.T) {
	execution, err := os.ReadFile("../../../modules/execution/v36/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	s, err := NewIncrementalSession(execution)
	if err != nil {
		t.Fatal(err)
	}
	r := s.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/cwd", PackagePath: "example.test/cwd", Entry: "Keep", Files: map[string]string{
		"main.go": "package cwd\nimport \"log\"\nvar _ *log.Logger\nfunc Keep(v int64) int64 { return v }\n",
	}})
	if !r.Valid {
		t.Fatal(r.Diagnostics)
	}
}
