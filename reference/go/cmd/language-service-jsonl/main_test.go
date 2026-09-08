package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"seme.local/reference/executionmodule"
)

func TestLiveLoopRetainsLastValidAndRejectsStale(t *testing.T) {
	input := strings.Join([]string{
		`{"id":1,"command":"initialize","session":"s","package_path":"example.test/live"}`,
		`{"id":2,"command":"update","session":"s","revision":1,"files":{"main.go":"package live\nfunc Decide(enabled bool, value, limit int64) bool {\nif enabled || (\"λ\" + \"!\" == \"never\") {\nif value <= limit { return \"exact\" + \"-text\" == \"exact-text\" }\nreturn false\n}\nreturn false\n}\n"}}`,
		`{"id":3,"command":"update","session":"s","revision":2,"files":{"main.go":"package live\nfunc Decide("}}`,
		`{"id":4,"command":"update","session":"s","revision":2,"files":{"main.go":"package live\nfunc Decide(enabled bool, value, limit int64) bool { return enabled && value <= limit }\n"}}`,
		`{"id":5,"command":"snapshot","session":"s"}`,
	}, "\n")
	var output bytes.Buffer
	if err := serve(strings.NewReader(input), &output, moduleVersion(t, 13), defaultMaxMessage); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.Bytes())
	if len(responses) != 5 {
		t.Fatalf("got %d responses", len(responses))
	}
	if !responses[1].OK || responses[1].State.Disposition != "accepted-valid" || responses[1].State.CanonicalG1 == "" {
		t.Fatalf("valid update: %+v", responses[1])
	}
	if responses[2].State.Disposition != "accepted-invalid" || responses[2].State.LastValidRevision != 1 || responses[2].State.CanonicalG1 == "" {
		t.Fatalf("invalid update: %+v", responses[2])
	}
	if responses[3].State.Disposition != "rejected-stale" || responses[3].State.Accepted {
		t.Fatalf("stale update: %+v", responses[3])
	}
	if responses[4].State.Disposition != "rejected-stale" || responses[4].State.LastValidRevision != 1 {
		t.Fatalf("snapshot: %+v", responses[4])
	}
}

func TestMalformedOversizedAndUnknownSessionRemainFramed(t *testing.T) {
	input := "not-json\n" + strings.Repeat("x", 300) + "\n" + `{"id":"after","command":"snapshot","session":"missing"}`
	var output bytes.Buffer
	if err := serve(strings.NewReader(input), &output, moduleV12(t), 256); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.Bytes())
	if len(responses) != 3 {
		t.Fatalf("got %d responses: %s", len(responses), output.String())
	}
	if responses[0].Error.Code != "transport.invalid_json" {
		t.Fatalf("first: %+v", responses[0])
	}
	if responses[1].Error.Code != "transport.message_too_large" {
		t.Fatalf("second: %+v", responses[1])
	}
	if responses[2].Error.Code != "session.not_initialized" || string(responses[2].ID) != `"after"` {
		t.Fatalf("third: %+v", responses[2])
	}
}

func TestRepeatedTranscriptIsDeterministic(t *testing.T) {
	input := `{"id":"i","command":"initialize","session":"s","package_path":"example.test/deterministic"}` + "\n" +
		`{"id":"u","command":"update","session":"s","revision":9,"files":{"z.go":"package deterministic\nfunc Z(v int64) int64 { return v }\n","a.go":"package deterministic\nfunc A(v int64) int64 { return v + 1 }\n"}}`
	var first, second bytes.Buffer
	if err := serve(strings.NewReader(input), &first, moduleV12(t), defaultMaxMessage); err != nil {
		t.Fatal(err)
	}
	if err := serve(strings.NewReader(input), &second, moduleV12(t), defaultMaxMessage); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Fatal("identical transcripts produced different responses")
	}
}

func TestOversizedResponseBecomesCorrelatedError(t *testing.T) {
	input := `{"id":"i","command":"initialize","session":"s","package_path":"example.test/bounded"}` + "\n" +
		`{"id":"large","command":"update","session":"s","revision":1,"files":{"main.go":"package bounded\nfunc Same(v int64) int64 { return v }\n"}}`
	var output bytes.Buffer
	if err := serve(strings.NewReader(input), &output, moduleV12(t), 512); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.Bytes())
	if len(responses) != 2 || responses[1].Error == nil || responses[1].Error.Code != "transport.response_too_large" || string(responses[1].ID) != `"large"` {
		t.Fatalf("responses: %+v", responses)
	}
	for _, line := range bytes.Split(bytes.TrimSuffix(output.Bytes(), []byte{'\n'}), []byte{'\n'}) {
		if len(line)+1 > 512 {
			t.Fatalf("response frame has %d bytes", len(line)+1)
		}
	}
}

func TestGracefulEmptyEOF(t *testing.T) {
	var output bytes.Buffer
	if err := serve(strings.NewReader("\n"), &output, moduleV12(t), 256); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("unexpected output %q", output.String())
	}
}

func moduleV12(t *testing.T) []byte {
	return moduleVersion(t, 12)
}

func moduleVersion(t *testing.T, version int) []byte {
	t.Helper()
	var module bytes.Buffer
	if err := executionmodule.Emit(&module, version); err != nil {
		t.Fatal(err)
	}
	return module.Bytes()
}

func decodeResponses(t *testing.T, encoded []byte) []response {
	t.Helper()
	var result []response
	scanner := bufio.NewScanner(bytes.NewReader(encoded))
	for scanner.Scan() {
		var item response
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			t.Fatal(err)
		}
		result = append(result, item)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}
