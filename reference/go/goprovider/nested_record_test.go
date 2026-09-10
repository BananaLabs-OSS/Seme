package goprovider

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestNestedRecordsComposeAcrossPackagesAndAliases(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v36/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"model/model.go": `package model
type Inner struct { Value int64 }
type Alias = Inner
`,
		"app/app.go": `package app
import "example.test/nested/model"
type Outer struct { Inner model.Alias }
func Apply(value int64) int64 {
	outer := Outer{Inner: model.Inner{Value: value}}
	return outer.Inner.Value
}
`,
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/nested", PackagePath: "example.test/nested/app", Entry: "Apply", Files: files})
	if !result.Valid {
		t.Fatalf("diagnostics=%#v", result.Diagnostics)
	}
	if strings.Count(result.CanonicalG1, " 00000000000000000000000000009030 1 ") != 2 {
		t.Fatal("nested record types were not emitted exactly once")
	}
}

func TestNestedRecordDiscoveryIsFileOrderIndependent(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v36/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	apply := func(revision uint64, files map[string]string) string {
		session, sessionErr := NewIncrementalSession(module)
		if sessionErr != nil {
			t.Fatal(sessionErr)
		}
		result := session.Apply(DocumentSnapshot{Revision: revision, ModulePath: "example.test/order", PackagePath: "example.test/order", Entry: "Read", Files: files})
		if !result.Valid {
			t.Fatalf("diagnostics=%#v", result.Diagnostics)
		}
		return result.CanonicalG1
	}
	a := apply(1, map[string]string{"z.go": "package order\ntype Outer struct { Inner Inner }\nfunc Read(v Outer) int64 { return v.Inner.Value }\n", "a.go": "package order\ntype Inner struct { Value int64 }\n"})
	b := apply(1, map[string]string{"a.go": "package order\ntype Inner struct { Value int64 }\n", "z.go": "package order\ntype Outer struct { Inner Inner }\nfunc Read(v Outer) int64 { return v.Inner.Value }\n"})
	if a != b {
		t.Fatal("canonical nested records depend on file or client revision order")
	}
}

func TestNestedRecordsRejectPointerCyclesAndDepthOverflow(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v36/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	deep := strings.Builder{}
	deep.WriteString("package depth\n")
	for index := 0; index < 33; index++ {
		deep.WriteString("type T")
		deep.WriteString(strconv.Itoa(index))
		deep.WriteString(" struct { Next T")
		deep.WriteString(strconv.Itoa(index + 1))
		deep.WriteString(" }\n")
	}
	deep.WriteString("type T33 struct { Value int64 }\nfunc Read(v T0) int64 { return 0 }\n")
	for name, source := range map[string]string{
		"pointer-cycle": "package cycle\ntype Node struct { Next *Node }\nfunc Read(v Node) int64 { return 0 }\n",
		"depth":         deep.String(),
	} {
		t.Run(name, func(t *testing.T) {
			session, sessionErr := NewIncrementalSession(module)
			if sessionErr != nil {
				t.Fatal(sessionErr)
			}
			result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/" + name, Entry: "Read", Files: map[string]string{"main.go": source}})
			if result.Valid {
				t.Fatal("unsupported recursive record graph accepted")
			}
		})
	}
}
