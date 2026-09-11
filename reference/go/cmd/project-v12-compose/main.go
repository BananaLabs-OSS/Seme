// Command project-v12-compose binds a neutral controlled-effects selection to
// an authenticated, closed Project-v11 bundle.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/controlledeffectsinstance"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/gocontrolledeffectsadapter"
	"seme.local/reference/gocontrolledeffectsmanifest"
	"seme.local/reference/godurablemanifest"
	"seme.local/reference/goorderedtransportmanifest"
	"seme.local/reference/goupb08bundle"
	"seme.local/reference/goupb09bundle"
	"seme.local/reference/projectv12instance"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "project-v12-compose:", err)
		os.Exit(1)
	}
}
func run() error {
	names := []string{"bundle-v11", "configuration-selection", "durable-selection", "transport-selection", "effects-selection", "foundation-contract", "execution-contract", "package-contract", "dependency-contract", "configuration-contract", "resource-contract", "durable-state-contract", "source-presentation-contract", "ordered-transport-contract", "controlled-effects-contract", "project-v8-contract", "project-v9-contract", "project-v10-contract", "project-v11-contract", "project-v12-contract", "k0", "g1-compiler", "out"}
	v := map[string]*string{}
	for _, n := range names {
		v[n] = new(string)
		flag.StringVar(v[n], n, "", n)
	}
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	for _, n := range names {
		if *v[n] == "" {
			return fmt.Errorf("flag:%s", n)
		}
	}
	read := func(path string) ([]byte, error) {
		i, e := os.Lstat(path)
		if e != nil || !i.Mode().IsRegular() || i.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("not_regular:%s", path)
		}
		return os.ReadFile(path)
	}
	c := map[string][]byte{}
	for _, n := range []string{"foundation-contract", "execution-contract", "package-contract", "dependency-contract", "configuration-contract", "resource-contract", "durable-state-contract", "source-presentation-contract", "ordered-transport-contract", "controlled-effects-contract", "project-v8-contract", "project-v9-contract", "project-v10-contract", "project-v11-contract", "project-v12-contract"} {
		b, e := read(*v[n])
		if e != nil {
			return e
		}
		c[n] = b
	}
	v8, e := contractcatalog.ResolveProjectContractSetV8(c["foundation-contract"], c["execution-contract"], c["package-contract"], c["dependency-contract"], c["configuration-contract"], c["project-v8-contract"])
	if e != nil {
		return e
	}
	v9, e := contractcatalog.ResolveProjectContractSetV9(c["foundation-contract"], c["execution-contract"], c["package-contract"], c["dependency-contract"], c["configuration-contract"], c["resource-contract"], c["project-v9-contract"])
	if e != nil {
		return e
	}
	v10, e := contractcatalog.ResolveProjectContractSetV10(c["foundation-contract"], c["execution-contract"], c["package-contract"], c["dependency-contract"], c["configuration-contract"], c["resource-contract"], c["durable-state-contract"], c["source-presentation-contract"], c["project-v9-contract"], c["project-v10-contract"])
	if e != nil {
		return e
	}
	v11, e := contractcatalog.ResolveProjectContractSetV11(c["foundation-contract"], c["execution-contract"], c["package-contract"], c["dependency-contract"], c["configuration-contract"], c["resource-contract"], c["durable-state-contract"], c["source-presentation-contract"], c["ordered-transport-contract"], c["project-v9-contract"], c["project-v10-contract"], c["project-v11-contract"])
	if e != nil {
		return e
	}
	v12, e := contractcatalog.ResolveProjectContractSetV12(c["foundation-contract"], c["execution-contract"], c["package-contract"], c["dependency-contract"], c["configuration-contract"], c["resource-contract"], c["durable-state-contract"], c["source-presentation-contract"], c["ordered-transport-contract"], c["controlled-effects-contract"], c["project-v9-contract"], c["project-v10-contract"], c["project-v11-contract"], c["project-v12-contract"])
	if e != nil {
		return e
	}
	parse := func(name string) ([]byte, error) { return read(*v[name]) }
	b, e := parse("configuration-selection")
	if e != nil {
		return e
	}
	configuration, e := goconfigurationmanifest.Parse(b)
	if e != nil {
		return e
	}
	b, e = parse("durable-selection")
	if e != nil {
		return e
	}
	durable, e := godurablemanifest.Parse(b)
	if e != nil {
		return e
	}
	b, e = parse("transport-selection")
	if e != nil {
		return e
	}
	transport, e := goorderedtransportmanifest.Parse(b)
	if e != nil {
		return e
	}
	b, e = parse("effects-selection")
	if e != nil {
		return e
	}
	effects, e := gocontrolledeffectsmanifest.Parse(b)
	if e != nil {
		return e
	}
	a, manifest, blobs, e := goupb08bundle.ReadDirectory(*v["bundle-v11"])
	if e != nil {
		return e
	}
	compile := func(ctx context.Context, source []byte) ([]byte, error) {
		d, x := os.MkdirTemp("", "seme-v12-compile-*")
		if x != nil {
			return nil, x
		}
		defer os.RemoveAll(d)
		in, out := filepath.Join(d, "in.g1"), filepath.Join(d, "out.seme")
		if x = os.WriteFile(in, source, 0600); x != nil {
			return nil, x
		}
		data, x := exec.CommandContext(ctx, *v["k0"], *v["g1-compiler"], in, out).CombinedOutput()
		if x != nil {
			return nil, fmt.Errorf("compile:%w:%s", x, data)
		}
		return os.ReadFile(out)
	}
	base, e := goupb08bundle.Load(context.Background(), goupb08bundle.Input{Contracts: v11, V10: v10, V9: v9, V8: v8, Artifacts: a, Manifest: manifest, Selection: configuration, DurableSelection: durable, TransportSelection: transport, Compile: compile, Blobs: blobs})
	if e != nil {
		return e
	}
	model, e := gocontrolledeffectsadapter.Resolve(base.Project, effects)
	if e != nil {
		return e
	}
	ci := controlledeffectsinstance.Inputs{Contracts: v12, ProjectV11: base.Project, Model: model}
	ci.Artifact, e = controlledeffectsinstance.Emit(ci)
	if e != nil {
		return e
	}
	if e = controlledeffectsinstance.Validate(ci); e != nil {
		return e
	}
	pi := projectv12instance.Inputs{Contracts: v12, ProjectV11: base.Project, Effects: ci}
	pi.Composed, e = projectv12instance.Emit(pi)
	if e != nil {
		return e
	}
	if e = projectv12instance.Validate(pi); e != nil {
		return e
	}
	replay, e := goupb09bundle.EncodeReplayAuthority(model.Replay, model.Bounds)
	if e != nil {
		return e
	}
	return publish(*v["out"], []named{{"controlled-effects-v1.seme", ci.Artifact}, {"project-v12.seme", pi.Composed}, {"controlled-replay-v1.json", replay}})
}

type named struct {
	name string
	data []byte
}

func publish(out string, files []named) error {
	if _, e := os.Lstat(out); !os.IsNotExist(e) {
		return fmt.Errorf("output_exists")
	}
	tmp, e := os.MkdirTemp(filepath.Dir(out), ".project-v12-compose-*")
	if e != nil {
		return e
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(tmp)
		}
	}()
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	m := []byte("seme-project-v12-compose-v1\n")
	for _, f := range files {
		if e = os.WriteFile(filepath.Join(tmp, f.name), f.data, 0600); e != nil {
			return e
		}
		s := sha256.Sum256(f.data)
		m = append(m, []byte(f.name+" "+hex.EncodeToString(s[:])+"\n")...)
	}
	if e = os.WriteFile(filepath.Join(tmp, "COMPLETE.sha256"), m, 0600); e != nil {
		return e
	}
	if e = os.Rename(tmp, out); e != nil {
		return e
	}
	ok = true
	return nil
}
