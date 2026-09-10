package goprovider

import (
	"os"
	"testing"
)

func TestProjectedResultMatchReliftsTypedZeroRecordArm(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v36/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	source := `package match
type Result[T, E any] struct { Ok bool; Value T; Error E }
type Settings struct { Limit int64 }
func Read(result Result[Settings, int64]) int64 {
	value := func() Settings {
		matched := result
		if matched.Ok { return matched.Value }
		return Settings{}
	}()
	return value.Limit
}`
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/result-match", Entry: "Read", Files: map[string]string{"match.go": source}})
	if !result.Valid {
		t.Fatalf("diagnostics=%#v", result.Diagnostics)
	}
}

func TestFunctionLiteralMatchRejectsExtraBehaviorAndStandalonePartialRecord(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v36/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"extra-behavior":  `value := func() Settings { matched := result; matched.Ok = false; if matched.Ok { return matched.Value }; return Settings{} }(); return value.Limit`,
		"standalone-zero": `value := Settings{}; return value.Limit`,
	} {
		t.Run(name, func(t *testing.T) {
			source := "package match\ntype Result[T, E any] struct { Ok bool; Value T; Error E }\ntype Settings struct { Limit int64 }\nfunc Read(result Result[Settings, int64]) int64 { " + body + " }"
			session, sessionErr := NewIncrementalSession(module)
			if sessionErr != nil {
				t.Fatal(sessionErr)
			}
			result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/result-match-adversary", Entry: "Read", Files: map[string]string{"match.go": source}})
			if result.Valid {
				t.Fatal("unsupported function-literal behavior accepted")
			}
		})
	}
}
