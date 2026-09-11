package goupb10bundle

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPlacementDirectoryRoundTripAndAtomicRejection(t *testing.T) {
	parent := t.TempDir()
	base := []byte("seme-go-upb09-bundle-v1\nexample digest\n")
	files := PlacementFiles{TargetPlan: []byte("plan"), ProjectV13: []byte("project"), Report: []byte("report")}
	first := filepath.Join(parent, "first")
	second := filepath.Join(parent, "second")
	if err := WriteDirectory(first, files, base); err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadDirectory(first, base)
	if err != nil || !reflect.DeepEqual(loaded, files) {
		t.Fatal("round trip", err)
	}
	if err = WriteDirectory(second, files, base); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(filepath.Join(first, "COMPLETE.sha256"))
	b, _ := os.ReadFile(filepath.Join(second, "COMPLETE.sha256"))
	if !bytes.Equal(a, b) {
		t.Fatal("manifest drift")
	}
	if err = WriteDirectory(first, files, base); err == nil {
		t.Fatal("overwrote output")
	}
	if err = WriteDirectory(filepath.Join(parent, "empty"), PlacementFiles{}, base); err == nil {
		t.Fatal("published empty artifacts")
	}
	if _, statErr := os.Lstat(filepath.Join(parent, "empty")); !os.IsNotExist(statErr) {
		t.Fatal("empty failure left partial output")
	}
}

func TestPlacementDirectoryRejectsTamperWrongBaseAndExtras(t *testing.T) {
	parent := t.TempDir()
	base := []byte("base")
	files := PlacementFiles{TargetPlan: []byte("plan"), ProjectV13: []byte("project"), Report: []byte("report")}
	for _, test := range []struct {
		name   string
		mutate func(string)
		base   []byte
	}{
		{"tamper", func(root string) {
			_ = os.WriteFile(filepath.Join(root, "target-plan-v1.seme"), []byte("forged"), 0o600)
		}, base},
		{"extra", func(root string) { _ = os.WriteFile(filepath.Join(root, "extra"), []byte("x"), 0o600) }, base},
		{"wrong-base", func(string) {}, []byte("other")},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(parent, test.name)
			if err := WriteDirectory(root, files, base); err != nil {
				t.Fatal(err)
			}
			test.mutate(root)
			if _, err := ReadDirectory(root, test.base); err == nil {
				t.Fatal("accepted invalid directory")
			}
		})
	}
}
