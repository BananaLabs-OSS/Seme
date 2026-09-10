package controlledeffectsinstance_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"seme.local/reference/canonicaleval"
	"seme.local/reference/contractcatalog"
	ceinstance "seme.local/reference/controlledeffectsinstance"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/gocontrolledeffectsadapter"
	"seme.local/reference/gocontrolledeffectsmanifest"
	"seme.local/reference/godurablemanifest"
	"seme.local/reference/goorderedtransportmanifest"
	"seme.local/reference/goupb08bundle"
	"seme.local/reference/goupb09replay"
	"seme.local/reference/wasmtarget"
	"seme.local/reference/wire"
)

func TestRealProjectEmitsDeterministicTamperSensitiveEffectsPlan(t *testing.T) {
	if testing.Short() {
		t.Skip("materialized project integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	repo, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	source, bundle, tool := filepath.Join(work, "source"), filepath.Join(work, "bundle"), filepath.Join(work, "go-upb08-build")
	run := func(c *exec.Cmd) {
		t.Helper()
		if out, e := c.CombinedOutput(); e != nil {
			t.Fatalf("%v: %v\n%s", c.Args, e, out)
		}
	}
	run(exec.CommandContext(ctx, filepath.Join(repo, "scripts/materialize-go-upb09-fixture.sh"), source))
	c := exec.CommandContext(ctx, "go", "build", "-buildvcs=false", "-o", tool, "./cmd/go-upb08-build")
	c.Dir = filepath.Join(repo, "reference/go")
	c.Env = append(os.Environ(), "GOCACHE="+filepath.Join(work, "cache"))
	run(c)
	p := func(x string) string { return filepath.Join(repo, x) }
	args := []string{"-project", source, "-proxy", p("fixtures/go-upb03-offline-proxy"), "-module", "example.test/go-uab-11", "-package", "example.test/go-uab-11/service", "-entry", "ApplyConfiguredResource", "-revision", "1", "-out", bundle, "-dependency", "example.test/seme/checksum", "-version", "v1.2.3", "-local-from", "example.test/go-uab-11/application", "-local-to", "example.test/go-uab-11/model", "-selection", p("fixtures/go-upb05-configuration-overlay/configuration-selection.json"), "-resources", filepath.Join(source, "resources.json"), "-resource-owner", "example.test/go-uab-11/service", "-durable-selection", filepath.Join(source, "durable-selection.json"), "-transport-selection", p("fixtures/go-upb08-transport-overlay/transport-selection.json"), "-execution-g1", p("modules/execution/v36/module.g1"), "-execution-contract", p("modules/execution/v36/module.seme"), "-foundation-contract", p("modules/foundation/v1/module.seme"), "-package-v4", p("modules/package/v4/module.seme"), "-project-v8", p("modules/project/v8/module.seme"), "-dependency-v1", p("modules/dependency/v1/module.seme"), "-configuration-v3", p("modules/configuration/v3/module.seme"), "-resource-v1", p("modules/resource/v1/module.seme"), "-project-v9", p("modules/project/v9/module.seme"), "-durable-state-v1", p("modules/durable-state/v1/module.seme"), "-source-presentation-v1", p("modules/source-presentation/v1/module.seme"), "-project-v10", p("modules/project/v10/module.seme"), "-ordered-transport-v1", p("modules/ordered-transport/v1/module.seme"), "-project-v11", p("modules/project/v11/module.seme"), "-k0", p("bootstrap/seme-k0-linux-amd64"), "-g1-compiler", p("compiler/g1-compiler.k0")}
	run(exec.CommandContext(ctx, tool, args...))
	read := func(path string) []byte {
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	f, e, pkg, dep, cfg, res, dur, pres, trans := read(p("modules/foundation/v1/module.seme")), read(p("modules/execution/v36/module.seme")), read(p("modules/package/v4/module.seme")), read(p("modules/dependency/v1/module.seme")), read(p("modules/configuration/v3/module.seme")), read(p("modules/resource/v1/module.seme")), read(p("modules/durable-state/v1/module.seme")), read(p("modules/source-presentation/v1/module.seme")), read(p("modules/ordered-transport/v1/module.seme"))
	p8, p9, p10, p11, p12, ce := read(p("modules/project/v8/module.seme")), read(p("modules/project/v9/module.seme")), read(p("modules/project/v10/module.seme")), read(p("modules/project/v11/module.seme")), read(p("modules/project/v12/module.seme")), read(p("modules/controlled-effects/v1/module.seme"))
	v8, _ := contractcatalog.ResolveProjectContractSetV8(f, e, pkg, dep, cfg, p8)
	v9, _ := contractcatalog.ResolveProjectContractSetV9(f, e, pkg, dep, cfg, res, p9)
	v10, _ := contractcatalog.ResolveProjectContractSetV10(f, e, pkg, dep, cfg, res, dur, pres, p9, p10)
	v11, err := contractcatalog.ResolveProjectContractSetV11(f, e, pkg, dep, cfg, res, dur, pres, trans, p9, p10, p11)
	if err != nil {
		t.Fatal(err)
	}
	v12, err := contractcatalog.ResolveProjectContractSetV12(f, e, pkg, dep, cfg, res, dur, pres, trans, ce, p9, p10, p11, p12)
	if err != nil {
		t.Fatal(err)
	}
	a, manifest, blobs, err := goupb08bundle.ReadDirectory(bundle)
	if err != nil {
		t.Fatal(err)
	}
	cs, _ := goconfigurationmanifest.Parse(read(p("fixtures/go-upb05-configuration-overlay/configuration-selection.json")))
	ds, _ := godurablemanifest.Parse(read(filepath.Join(source, "durable-selection.json")))
	ts, _ := goorderedtransportmanifest.Parse(read(p("fixtures/go-upb08-transport-overlay/transport-selection.json")))
	compile := func(ctx context.Context, source []byte) ([]byte, error) {
		d, er := os.MkdirTemp(work, "compile-")
		if er != nil {
			return nil, er
		}
		defer os.RemoveAll(d)
		in, out := filepath.Join(d, "in.g1"), filepath.Join(d, "out.seme")
		if er = os.WriteFile(in, source, 0600); er != nil {
			return nil, er
		}
		b, er := exec.CommandContext(ctx, p("bootstrap/seme-k0-linux-amd64"), p("compiler/g1-compiler.k0"), in, out).CombinedOutput()
		if er != nil {
			return nil, fmt.Errorf("compile:%w:%s", er, b)
		}
		return os.ReadFile(out)
	}
	loaded, err := goupb08bundle.Load(ctx, goupb08bundle.Input{Contracts: v11, V10: v10, V9: v9, V8: v8, Artifacts: a, Manifest: manifest, Selection: cs, DurableSelection: ds, TransportSelection: ts, Compile: compile, Blobs: blobs})
	if err != nil {
		t.Fatal(err)
	}
	selection, err := gocontrolledeffectsmanifest.Parse(read(p("fixtures/go-upb09-effects-overlay/controlled-effects-selection.json")))
	if err != nil {
		t.Fatal(err)
	}
	model, err := gocontrolledeffectsadapter.Resolve(loaded.Project, selection)
	if err != nil {
		t.Fatal(err)
	}
	in := ceinstance.Inputs{Contracts: v12, ProjectV11: loaded.Project, Model: model}
	first, err := ceinstance.Emit(in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ceinstance.Emit(in)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("nondeterministic", err)
	}
	in.Artifact = first
	if err = ceinstance.Validate(in); err != nil {
		t.Fatal(err)
	}
	// Reuse this exact materialized graph to build and execute one nonempty
	// replay. The native corpus supplies ABI-shaped values, but ReplayControlled
	// itself is invoked by the independent canonical evaluator below.
	corpus := filepath.Join(work, "effects-corpus")
	c = exec.CommandContext(ctx, "go", "test", "-count=1", "-buildvcs=false", "./streamservice", "-run", "^TestNativeControlledEffectsRuntimeCorpus$", "-args", "-native-effects-corpus-dir", corpus)
	c.Dir = source
	c.Env = append(os.Environ(), "GOCACHE="+filepath.Join(work, "native-cache"), "GOTOOLCHAIN=local", "GOPROXY=file://"+p("fixtures/go-upb03-offline-proxy"), "GOSUMDB=off")
	run(c)
	stateValue, commandValue := firstAcceptedRequest(t, filepath.Join(corpus, "requests.jsonl"), filepath.Join(corpus, "expected.jsonl"))
	program, err := wire.Decode(loaded.Project.ProjectV10.Composed)
	if err != nil {
		t.Fatal(err)
	}
	stateLayout, err := wasmtarget.CertifyPureValueLayout(program, model.State)
	if err != nil {
		t.Fatal(err)
	}
	commandLayout, err := wasmtarget.CertifyPureValueLayout(program, model.Command)
	if err != nil {
		t.Fatal(err)
	}
	executor := realReplayExecutor{graph: program, function: model.ReplayFunction, state: stateLayout, command: commandLayout}
	initialBytes, err := wasmtarget.EncodePureValue(stateLayout, stateValue)
	if err != nil {
		t.Fatal(err)
	}
	commandBytes, err := wasmtarget.EncodePureValue(commandLayout, commandValue)
	if err != nil {
		t.Fatal(err)
	}
	outcome := executor.Execute(goupb09replay.Request{State: initialBytes, CanonicalCommand: commandBytes, CommandSequence: 1, ClockSequence: 1, RandomDrawOrdinal: 1, UnixMilliseconds: valueI64(commandValue.Fields["Clock"].Fields["UnixMilliseconds"]), RandomBefore: valueI64(commandValue.Fields["Random"].Fields["Value"])})
	if !outcome.Committed || outcome.Failure != "" {
		t.Fatal(outcome.Failure)
	}
	model.Replay.InitialSeed = outcomeRequestRandom(commandValue)
	model.Replay.InitialState = initialBytes
	model.Replay.Steps = []ceinstance.ReplayStep{{CommandSequence: 1, ClockSequence: 1, UnixMilliseconds: valueI64(commandValue.Fields["Clock"].Fields["UnixMilliseconds"]), RandomBefore: model.Replay.InitialSeed, RandomAfter: outcome.RandomAfter, RandomValue: outcome.RandomValue, RandomDrawOrdinal: 1, EffectValue: outcome.EffectValue, CanonicalCommand: commandBytes, ResponseSHA256: sha256.Sum256(outcome.Response), EventsSHA256: sha256.Sum256(outcome.Events), StateBeforeSHA256: sha256.Sum256(initialBytes), StateAfterSHA256: sha256.Sum256(outcome.State)}}
	in.Model = model
	in.Artifact, err = ceinstance.Emit(in)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := goupb09replay.Verify(in, executor)
	if err != nil || replayed.Steps != 1 || !bytes.Equal(replayed.FinalState, outcome.State) {
		t.Fatal(replayed, err)
	}
	bad := in
	bad.Model.Replay.Steps = append([]ceinstance.ReplayStep(nil), in.Model.Replay.Steps...)
	bad.Model.Replay.Steps[0].ResponseSHA256[0] ^= 1
	bad.Artifact, err = ceinstance.Emit(bad)
	if err != nil {
		t.Fatal(err)
	}
	if partial, verifyErr := goupb09replay.Verify(bad, executor); verifyErr == nil || !reflect.DeepEqual(partial, goupb09replay.Result{}) {
		t.Fatal("accepted real replay tamper", partial, verifyErr)
	}
	g, err := wire.Decode(first)
	if err != nil {
		t.Fatal(err)
	}
	root := oneSchema(t, g, xid("13100"))
	replay := g.Entities[root.Fields[xid("13203")].Reference]
	bounds := g.Entities[root.Fields[xid("13204")].Reference]
	random := g.Entities[root.Fields[xid("13201")].Reference]
	if random.Fields[xid("13234")].Reference != model.NextFunction || len(replay.Fields[xid("13272")].Bytes) != 32 || len(replay.Fields[xid("13276")].Bytes) != 32 || bounds.Fields[xid("13288")].Unsigned != 4096 || bounds.Fields[xid("13289")].Unsigned != 524288 || bounds.Fields[xid("1328a")].Unsigned != 1048576 {
		t.Fatal("strengthened authority fields")
	}
	mutated := append([]byte(nil), first...)
	mutated[len(mutated)-1] ^= 1
	in.Artifact = mutated
	if err = ceinstance.Validate(in); err == nil {
		t.Fatal("accepted byte tamper")
	}
	q := random
	q.Fields[xid("13234")] = xref(model.ReplayFunction)
	g.Entities[q.ID] = q
	mutated, _ = wire.Encode(g)
	in.Artifact = mutated
	if err = ceinstance.Validate(in); err == nil {
		t.Fatal("accepted next binding substitution")
	}
}

type realReplayExecutor struct {
	graph          wire.Envelope
	function       wire.ID
	state, command wasmtarget.PureValueLayout
}

func (x realReplayExecutor) Execute(r goupb09replay.Request) goupb09replay.Outcome {
	s, e := wasmtarget.DecodePureValue(x.state, r.State)
	if e != nil {
		return goupb09replay.Outcome{Failure: e.Error()}
	}
	c, e := wasmtarget.DecodePureValue(x.command, r.CanonicalCommand)
	if e != nil {
		return goupb09replay.Outcome{Failure: e.Error()}
	}
	v, effects, e := canonicaleval.EvaluateFunctionAuthorized(x.graph, x.function, []canonicaleval.Value{s, c}, map[string]bool{})
	if e != nil || len(effects) != 0 {
		return goupb09replay.Outcome{Failure: fmt.Sprint(e)}
	}
	next, response := v.Fields["State"], v.Fields["Response"]
	nextBytes, e := wasmtarget.EncodePureValue(x.state, next)
	if e != nil {
		return goupb09replay.Outcome{Failure: e.Error()}
	}
	responseLayout, ok := layoutField(resultLayout(x.graph, x.function), "Response")
	if !ok {
		return goupb09replay.Outcome{Failure: "response layout"}
	}
	responseBytes, e := wasmtarget.EncodePureValue(responseLayout, response)
	if e != nil {
		return goupb09replay.Outcome{Failure: e.Error()}
	}
	transportLayout, ok := layoutField(x.state, "Transport")
	if !ok {
		return goupb09replay.Outcome{Failure: "events layout"}
	}
	eventBytes, e := wasmtarget.EncodePureValue(transportLayout, next.Fields["Transport"])
	if e != nil {
		return goupb09replay.Outcome{Failure: e.Error()}
	}
	after := r.RandomBefore*48271 + 1
	return goupb09replay.Outcome{Committed: true, State: nextBytes, Response: responseBytes, Events: eventBytes, RandomAfter: after, RandomValue: after, RandomDrawOrdinal: r.RandomDrawOrdinal, EffectValue: response.Fields["Accepted"].Bool}
}
func resultLayout(g wire.Envelope, function wire.ID) wasmtarget.PureValueLayout {
	return mustLayout(g, g.Entities[function].Fields[xid("9112")].Reference)
}
func mustLayout(g wire.Envelope, typ wire.ID) wasmtarget.PureValueLayout {
	x, e := wasmtarget.CertifyPureValueLayout(g, typ)
	if e != nil {
		panic(e)
	}
	return x
}
func layoutField(l wasmtarget.PureValueLayout, name string) (wasmtarget.PureValueLayout, bool) {
	for _, f := range l.Fields {
		if f.Name == name {
			return f.Value, true
		}
	}
	return wasmtarget.PureValueLayout{}, false
}
func valueI64(v canonicaleval.Value) int64 { var x int64; _, _ = fmt.Sscan(v.I64, &x); return x }
func outcomeRequestRandom(v canonicaleval.Value) int64 {
	return valueI64(v.Fields["Random"].Fields["Value"])
}
func firstAcceptedRequest(t *testing.T, requests, expected string) (canonicaleval.Value, canonicaleval.Value) {
	t.Helper()
	rq, e := os.Open(requests)
	if e != nil {
		t.Fatal(e)
	}
	defer rq.Close()
	ex, e := os.Open(expected)
	if e != nil {
		t.Fatal(e)
	}
	defer ex.Close()
	rs, xs := bufio.NewScanner(rq), bufio.NewScanner(ex)
	for rs.Scan() && xs.Scan() {
		var r struct {
			Arguments []canonicaleval.Value `json:"arguments"`
		}
		var x struct {
			Value canonicaleval.Value `json:"value"`
		}
		if json.Unmarshal(rs.Bytes(), &r) != nil || json.Unmarshal(xs.Bytes(), &x) != nil {
			t.Fatal("corpus JSON")
		}
		if len(r.Arguments) == 2 && x.Value.Fields["OK"].Bool && !x.Value.Fields["Duplicate"].Bool {
			return r.Arguments[0], r.Arguments[1]
		}
	}
	t.Fatal("no accepted corpus request")
	return canonicaleval.Value{}, canonicaleval.Value{}
}

func xid(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
func xref(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }

func oneSchema(t *testing.T, e wire.Envelope, s wire.ID) wire.Entity {
	t.Helper()
	var q wire.Entity
	n := 0
	for _, x := range e.Entities {
		if x.Schema == s {
			q = x
			n++
		}
	}
	if n != 1 {
		t.Fatalf("schema %s count %d", s, n)
	}
	return q
}
