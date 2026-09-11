// Command project-v9-compose binds detached project resources to an
// authenticated Project-v8 bundle. Its inputs and outputs are language-neutral.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/goresourceadapter"
	"seme.local/reference/goresourcemanifest"
	"seme.local/reference/goupb05bundle"
	"seme.local/reference/projectsource"
	"seme.local/reference/projectv9instance"
	"seme.local/reference/resourceinstance"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "project-v9-compose:", err)
		os.Exit(1)
	}
}
func run() error {
	names := []string{"root", "snapshot", "bundle", "configuration-selection", "resource-selection", "owner-package", "foundation-contract", "execution-contract", "package-contract", "dependency-contract", "configuration-contract", "project-v8-contract", "resource-contract", "project-v9-contract", "k0", "g1-compiler", "out"}
	values := map[string]*string{}
	for _, name := range names {
		v := ""
		values[name] = &v
		flag.StringVar(values[name], name, "", name)
	}
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	for _, name := range names {
		if *values[name] == "" {
			return fmt.Errorf("flag:%s", name)
		}
	}
	read := func(path string) ([]byte, error) {
		i, e := os.Lstat(path)
		if e != nil || !i.Mode().IsRegular() || i.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("not_regular:%s", path)
		}
		return os.ReadFile(path)
	}
	must := func(name string) ([]byte, error) { return read(*values[name]) }
	inputs := map[string][]byte{}
	for _, name := range []string{"snapshot", "configuration-selection", "resource-selection", "foundation-contract", "execution-contract", "package-contract", "dependency-contract", "configuration-contract", "project-v8-contract", "resource-contract", "project-v9-contract"} {
		b, e := must(name)
		if e != nil {
			return e
		}
		inputs[name] = b
	}
	v8, e := contractcatalog.ResolveProjectContractSetV8(inputs["foundation-contract"], inputs["execution-contract"], inputs["package-contract"], inputs["dependency-contract"], inputs["configuration-contract"], inputs["project-v8-contract"])
	if e != nil {
		return e
	}
	v9, e := contractcatalog.ResolveProjectContractSetV9(inputs["foundation-contract"], inputs["execution-contract"], inputs["package-contract"], inputs["dependency-contract"], inputs["configuration-contract"], inputs["resource-contract"], inputs["project-v9-contract"])
	if e != nil {
		return e
	}
	selection, e := goconfigurationmanifest.Parse(inputs["configuration-selection"])
	if e != nil {
		return e
	}
	resources, e := goresourcemanifest.Parse(inputs["resource-selection"])
	if e != nil {
		return e
	}
	var snapshot projectsource.Snapshot
	if e = json.Unmarshal(inputs["snapshot"], &snapshot); e != nil {
		return e
	}
	bundle := *values["bundle"]
	artifact := func(name string) ([]byte, error) { return read(filepath.Join(bundle, name)) }
	a := goupb05bundle.V8Artifacts{}
	parts := []struct {
		name string
		to   *[]byte
	}{{"construction-v36.g1", &a.Construction}, {"execution-v36.seme", &a.Execution}, {"project-base-v8.seme", &a.ProjectBase}, {"inventory-v8.seme", &a.Inventory}, {"package-detail-v4.seme", &a.PackageDetail}, {"package-v4.seme", &a.PackageV4}, {"dependency-v1.seme", &a.Dependency}, {"configuration-v3.seme", &a.ConfigurationV3}, {"project-v8.seme", &a.ProjectV8}}
	for _, part := range parts {
		*part.to, e = artifact(part.name)
		if e != nil {
			return e
		}
	}
	manifest, e := artifact("COMPLETE.sha256")
	if e != nil {
		return e
	}
	compile := func(ctx context.Context, source []byte) ([]byte, error) {
		d, e := os.MkdirTemp("", "seme-v9-compile-*")
		if e != nil {
			return nil, e
		}
		defer os.RemoveAll(d)
		in, out := filepath.Join(d, "in.g1"), filepath.Join(d, "out.seme")
		if e = os.WriteFile(in, source, 0600); e != nil {
			return nil, e
		}
		c := exec.CommandContext(ctx, *values["k0"], *values["g1-compiler"], in, out)
		if b, x := c.CombinedOutput(); x != nil {
			return nil, fmt.Errorf("compile:%w:%s", x, b)
		}
		return os.ReadFile(out)
	}
	base, e := goupb05bundle.LoadV8(context.Background(), goupb05bundle.V8Input{Contracts: v8, Artifacts: a, Manifest: manifest, Selection: selection, Compile: compile})
	if e != nil {
		return e
	}
	model, store, e := goresourceadapter.Resolve(*values["root"], snapshot, base.ProjectV8Input, *values["owner-package"], resources)
	if e != nil {
		return e
	}
	ri := resourceinstance.Inputs{Contracts: v9, ProjectV8: base.ProjectV8Input, Model: goresourceadapter.InstanceModel(model)}
	ri.Artifact, e = resourceinstance.Emit(ri)
	if e != nil {
		return e
	}
	blobs := map[[32]byte][]byte{}
	for _, blob := range store.Blobs {
		blobs[blob.SHA256] = blob.Bytes
	}
	if e = resourceinstance.VerifyDetached(ri, blobs); e != nil {
		return e
	}
	pi := projectv9instance.Inputs{Contracts: v9, ProjectV8: base.ProjectV8Input, Resource: ri}
	pi.Composed, e = projectv9instance.Emit(pi)
	if e != nil {
		return e
	}
	if e = projectv9instance.Validate(pi); e != nil {
		return e
	}
	out := *values["out"]
	if _, e = os.Lstat(out); !os.IsNotExist(e) {
		return fmt.Errorf("output_exists")
	}
	tmp, e := os.MkdirTemp(filepath.Dir(out), ".project-v9-compose-*")
	if e != nil {
		return e
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(tmp)
		}
	}()
	if e = os.Mkdir(filepath.Join(tmp, "blobs"), 0700); e != nil {
		return e
	}
	files := []struct {
		name string
		data []byte
	}{{"resource-v1.seme", ri.Artifact}, {"project-v9.seme", pi.Composed}}
	complete := []byte("seme-project-v9-compose-v1\n")
	for _, f := range files {
		if e = os.WriteFile(filepath.Join(tmp, f.name), f.data, 0600); e != nil {
			return e
		}
		sum := sha256.Sum256(f.data)
		complete = append(complete, []byte(f.name+" "+hex.EncodeToString(sum[:])+"\n")...)
	}
	keys := make([][32]byte, 0, len(blobs))
	for key := range blobs {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i][:], keys[j][:]) < 0 })
	for _, key := range keys {
		name := hex.EncodeToString(key[:])
		if e = os.WriteFile(filepath.Join(tmp, "blobs", name), blobs[key], 0600); e != nil {
			return e
		}
		complete = append(complete, []byte("blobs/"+name+" "+name+"\n")...)
	}
	if e = os.WriteFile(filepath.Join(tmp, "COMPLETE.sha256"), complete, 0600); e != nil {
		return e
	}
	if e = os.Rename(tmp, out); e != nil {
		return e
	}
	ok = true
	return nil
}
