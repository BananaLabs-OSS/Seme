package goupb08bundle_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/godurablemanifest"
	"seme.local/reference/goorderedtransportadapter"
	"seme.local/reference/goprovider"
	"seme.local/reference/goupb08bundle"
	"seme.local/reference/goupb08portruntime"
	"seme.local/reference/goupb08report"
	"seme.local/reference/internal/upb07testfixture"
	"seme.local/reference/orderedtransportinstance"
	"seme.local/reference/orderedtransportruntime"
	"seme.local/reference/projectv11instance"
	"seme.local/reference/wasmtarget"
)

func TestLoadAuthenticatesAndRejectsMixedFinalArtifacts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	work := t.TempDir()
	fx, err := upb07testfixture.Build(ctx, work)
	if err != nil {
		t.Fatal(err)
	}
	v11 := resolveV11(t, fx.Root)
	selection := goorderedtransportadapter.Selection{Streams: []goorderedtransportadapter.OwnedIdentity{{Identity: "primary", Package: "example.test/go-uab-11/service"}}, CommandKinds: []goorderedtransportadapter.KindSelection{{Identity: "configuration.initialize.v1", Owner: "example.test/go-uab-11/configuration", Payload: goorderedtransportadapter.Named{Package: "example.test/go-uab-11/configuration", Name: "Input"}}}, EventKinds: []goorderedtransportadapter.KindSelection{{Identity: "configuration.initialized.v1", Owner: "example.test/go-uab-11/configuration", Payload: goorderedtransportadapter.Named{Package: "example.test/go-uab-11/configuration", Name: "Initialized"}}}, Ports: []goorderedtransportadapter.OwnedIdentity{{Identity: "host.frames.v1", Package: "example.test/go-uab-11/service"}}, Dispatch: goorderedtransportadapter.Named{Package: "example.test/go-uab-11/service", Name: "Assemble"}, Replay: goorderedtransportadapter.Named{Package: "example.test/go-uab-11/service", Name: "Initialize"}}
	model, err := goorderedtransportadapter.Resolve(fx.Result.Project, selection)
	if err != nil {
		t.Fatal(err)
	}
	ti := orderedtransportinstance.Inputs{Contracts: v11, ProjectV10: fx.Result.Project, Model: model}
	ti.Artifact, err = orderedtransportinstance.Emit(ti)
	if err != nil {
		t.Fatal(err)
	}
	pi := projectv11instance.Inputs{Contracts: v11, ProjectV10: fx.Result.Project, Transport: ti}
	pi.Composed, err = projectv11instance.Emit(pi)
	if err != nil {
		t.Fatal(err)
	}
	a := goupb08bundle.Artifacts{Base: fx.Result.Artifacts, Transport: ti.Artifact, ProjectV11: pi.Composed}
	manifest := completion(a, fx.Result.Blobs)
	configBytes, _ := os.ReadFile(filepath.Join(fx.Root, "fixtures/go-upb05-configuration-overlay/configuration-selection.json"))
	config, err := goconfigurationmanifest.Parse(configBytes)
	if err != nil {
		t.Fatal(err)
	}
	durableBytes, _ := os.ReadFile(filepath.Join(fx.Source, "durable-selection.json"))
	durable, err := godurablemanifest.Parse(durableBytes)
	if err != nil {
		t.Fatal(err)
	}
	compile := func(ctx context.Context, source []byte) ([]byte, error) {
		d, e := os.MkdirTemp(work, "compile-")
		if e != nil {
			return nil, e
		}
		defer os.RemoveAll(d)
		in, out := filepath.Join(d, "in.g1"), filepath.Join(d, "out.seme")
		if e = os.WriteFile(in, source, 0600); e != nil {
			return nil, e
		}
		b, e := exec.CommandContext(ctx, filepath.Join(fx.Root, "bootstrap/seme-k0-linux-amd64"), filepath.Join(fx.Root, "compiler/g1-compiler.k0"), in, out).CombinedOutput()
		if e != nil {
			return nil, fmt.Errorf("compile:%w:%s", e, b)
		}
		return os.ReadFile(out)
	}
	in := goupb08bundle.Input{Contracts: v11, V10: fx.Result.Project.Contracts, V9: fx.Result.Base.Project.Contracts, V8: fx.Result.Base.Base.ProjectV8Input.Contracts, Artifacts: a, Manifest: manifest, Selection: config, DurableSelection: durable, TransportSelection: selection, Compile: compile, Blobs: fx.Result.Blobs}
	loaded, err := goupb08bundle.Load(ctx, in)
	if err != nil || !bytes.Equal(loaded.Artifacts.ProjectV11, a.ProjectV11) {
		t.Fatal(err)
	}
	report, err := goupb08report.Inspect(loaded)
	if err != nil || report.MaximumRetainedPayloadBytes != 4096 || len(report.Streams) != 1 || len(report.CommandKinds) != 1 || len(report.EventKinds) != 1 {
		t.Fatal("source-free authority report", err)
	}
	projected, err := goupb08bundle.Project(loaded)
	if err != nil {
		t.Fatal(err)
	}
	executionG1, err := os.ReadFile(filepath.Join(fx.Root, "modules/execution/v36/module.g1"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for pkg, source := range projected {
		rel := "seme_projected.go"
		if pkg != "example.test/go-uab-11" {
			rel = filepath.ToSlash(filepath.Join(strings.TrimPrefix(pkg, "example.test/go-uab-11/"), rel))
		}
		files[rel] = string(source)
	}
	session, err := goprovider.NewIncrementalSession(executionG1)
	if err != nil {
		t.Fatal(err)
	}
	relifted := session.Apply(goprovider.DocumentSnapshot{Revision: 1, ModulePath: "example.test/go-uab-11", PackagePath: "example.test/go-uab-11/service", Entry: "ApplyConfiguredResource", Files: files})
	if !relifted.Valid || !bytes.Equal([]byte(relifted.CanonicalG1), a.Base.Construction) {
		t.Fatal("ordinary Go projection did not re-lift to its exact canonical graph")
	}
	profile, err := orderedtransportruntime.AuthenticatedProfile(v11.OrderedTransport())
	if err != nil {
		t.Fatal(err)
	}
	layout := wasmtarget.PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "bytes", FixedSize: 8, VariablePayload: true, MaximumPayload: 4096, Encoding: "u32le-offset-u32le-byte-length/opaque"}
	payload := make([]byte, 9)
	binary.LittleEndian.PutUint32(payload[:4], 8)
	binary.LittleEndian.PutUint32(payload[4:8], 1)
	payload[8] = 7
	frame, err := orderedtransportruntime.EncodeFrame(profile, orderedtransportruntime.Layouts{Command: layout, Response: layout}, orderedtransportruntime.Frame{Kind: orderedtransportruntime.FrameCommand, Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	port := &integrationPort{frame: frame}
	hostResult, err := goupb08portruntime.ExecuteSelected(loaded.Transport, "host.frames.v1", orderedtransportruntime.Layouts{Command: layout, Response: layout}, port, integrationHandler{payload: payload})
	if err != nil || !hostResult.Sent || len(hostResult.Trace) != 2 {
		t.Fatal("selected host port", err, hostResult)
	}
	bad := in
	bad.Artifacts = cloneArtifacts(a)
	bad.Artifacts.ProjectV11[len(bad.Artifacts.ProjectV11)-1] ^= 1
	bad.Manifest = completion(bad.Artifacts, bad.Blobs)
	if out, e := goupb08bundle.Load(ctx, bad); e == nil || len(out.Artifacts.ProjectV11) != 0 {
		t.Fatal("mixed final artifact accepted")
	}
	bad = in
	bad.Artifacts = cloneArtifacts(a)
	bad.Artifacts.Transport[len(bad.Artifacts.Transport)-1] ^= 1
	bad.Manifest = completion(bad.Artifacts, bad.Blobs)
	if _, e := goupb08bundle.Load(ctx, bad); e == nil {
		t.Fatal("tampered transport accepted")
	}
}

type integrationPort struct {
	frame []byte
	sent  map[string][]byte
}

func (p *integrationPort) Receive(orderedtransportruntime.ReceiveRequest) orderedtransportruntime.ReceiveOutcome {
	return orderedtransportruntime.ReceiveOutcome{Frame: bytes.Clone(p.frame)}
}
func (p *integrationPort) Send(r orderedtransportruntime.SendRequest) orderedtransportruntime.SendOutcome {
	if p.sent == nil {
		p.sent = map[string][]byte{}
	}
	s := sha256.Sum256(r.Frame)
	receipt := bytes.Clone(s[:])
	p.sent[string(receipt)] = bytes.Clone(r.Frame)
	return orderedtransportruntime.SendOutcome{Receipt: receipt, AcceptedSHA256: s}
}
func (p *integrationPort) Evidence(r []byte) ([]byte, bool) {
	b, ok := p.sent[string(r)]
	return bytes.Clone(b), ok
}

type integrationHandler struct{ payload []byte }

func (h integrationHandler) Handle(orderedtransportruntime.Frame) orderedtransportruntime.SemanticResult {
	return orderedtransportruntime.SemanticResult{Response: orderedtransportruntime.Frame{Kind: orderedtransportruntime.FrameResponse, Payload: bytes.Clone(h.payload)}, Committed: true}
}

func resolveV11(t *testing.T, root string) contractcatalog.ProjectContractSetV11 {
	t.Helper()
	read := func(p string) []byte {
		b, e := os.ReadFile(filepath.Join(root, p))
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	s, e := contractcatalog.ResolveProjectContractSetV11(read("modules/foundation/v1/module.seme"), read("modules/execution/v36/module.seme"), read("modules/package/v4/module.seme"), read("modules/dependency/v1/module.seme"), read("modules/configuration/v3/module.seme"), read("modules/resource/v1/module.seme"), read("modules/durable-state/v1/module.seme"), read("modules/source-presentation/v1/module.seme"), read("modules/ordered-transport/v1/module.seme"), read("modules/project/v9/module.seme"), read("modules/project/v10/module.seme"), read("modules/project/v11/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func completion(a goupb08bundle.Artifacts, blobs map[[32]byte][]byte) []byte {
	fs := [][]byte{a.Base.Construction, a.Base.Execution, a.Base.ProjectBase, a.Base.Inventory, a.Base.PackageDetail, a.Base.PackageV4, a.Base.Dependency, a.Base.ConfigurationV3, a.Base.ProjectV8, a.Base.Resource, a.Base.ProjectV9, a.Base.Durable, a.Base.Presentation, a.Base.ProjectV10, a.Transport, a.ProjectV11}
	names := []string{"construction-v36.g1", "execution-v36.seme", "project-base-v8.seme", "inventory-v8.seme", "package-detail-v4.seme", "package-v4.seme", "dependency-v1.seme", "configuration-v3.seme", "project-v8.seme", "resource-v1.seme", "project-v9.seme", "durable-state-v1.seme", "source-presentation-v1.seme", "project-v10.seme", "ordered-transport-v1.seme", "project-v11.seme"}
	r := []byte("seme-go-upb08-bundle-v1\n")
	for i, b := range fs {
		s := sha256.Sum256(b)
		r = append(r, []byte(names[i]+" "+hex.EncodeToString(s[:])+"\n")...)
	}
	keys := make([][32]byte, 0, len(blobs))
	for k := range blobs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i][:], keys[j][:]) < 0 })
	for _, k := range keys {
		h := hex.EncodeToString(k[:])
		r = append(r, []byte("blobs/"+h+" "+h+"\n")...)
	}
	return r
}
func cloneArtifacts(a goupb08bundle.Artifacts) goupb08bundle.Artifacts {
	p := func(x []byte) []byte { return append([]byte(nil), x...) }
	b := a
	b.Transport = p(a.Transport)
	b.ProjectV11 = p(a.ProjectV11)
	return b
}
