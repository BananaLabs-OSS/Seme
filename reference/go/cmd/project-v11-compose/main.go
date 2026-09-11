// Command project-v11-compose binds a neutral ordered-transport selection to
// an authenticated Project-v10 artifact chain.
package main

import (
	"bytes"
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
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/godurablemanifest"
	"seme.local/reference/goorderedtransportadapter"
	"seme.local/reference/goorderedtransportmanifest"
	"seme.local/reference/goupb07bundle"
	"seme.local/reference/orderedtransportinstance"
	"seme.local/reference/projectv11instance"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "project-v11-compose:", err)
		os.Exit(1)
	}
}

func run() error {
	names := []string{"bundle-v8", "bundle-v9", "bundle-v10", "configuration-selection", "durable-selection", "transport-selection", "foundation-contract", "execution-contract", "package-contract", "dependency-contract", "configuration-contract", "project-v8-contract", "resource-contract", "project-v9-contract", "durable-state-contract", "source-presentation-contract", "project-v10-contract", "ordered-transport-contract", "project-v11-contract", "k0", "g1-compiler", "out"}
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
	for _, n := range []string{"foundation-contract", "execution-contract", "package-contract", "dependency-contract", "configuration-contract", "project-v8-contract", "resource-contract", "project-v9-contract", "durable-state-contract", "source-presentation-contract", "project-v10-contract", "ordered-transport-contract", "project-v11-contract"} {
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
	configurationBytes, e := read(*v["configuration-selection"])
	if e != nil {
		return e
	}
	configuration, e := goconfigurationmanifest.Parse(configurationBytes)
	if e != nil {
		return e
	}
	durableBytes, e := read(*v["durable-selection"])
	if e != nil {
		return e
	}
	durable, e := godurablemanifest.Parse(durableBytes)
	if e != nil {
		return e
	}
	transportBytes, e := read(*v["transport-selection"])
	if e != nil {
		return e
	}
	transport, e := goorderedtransportmanifest.Parse(transportBytes)
	if e != nil {
		return e
	}
	a := goupb07bundle.Artifacts{}
	dirs := map[string]string{"v8": *v["bundle-v8"], "v9": *v["bundle-v9"], "v10": *v["bundle-v10"]}
	for _, x := range []struct {
		level, name string
		to          *[]byte
	}{{"v8", "construction-v36.g1", &a.Construction}, {"v8", "execution-v36.seme", &a.Execution}, {"v8", "project-base-v8.seme", &a.ProjectBase}, {"v8", "inventory-v8.seme", &a.Inventory}, {"v8", "package-detail-v4.seme", &a.PackageDetail}, {"v8", "package-v4.seme", &a.PackageV4}, {"v8", "dependency-v1.seme", &a.Dependency}, {"v8", "configuration-v3.seme", &a.ConfigurationV3}, {"v8", "project-v8.seme", &a.ProjectV8}, {"v9", "resource-v1.seme", &a.Resource}, {"v9", "project-v9.seme", &a.ProjectV9}, {"v10", "durable-state-v1.seme", &a.Durable}, {"v10", "source-presentation-v1.seme", &a.Presentation}, {"v10", "project-v10.seme", &a.ProjectV10}} {
		*x.to, e = read(filepath.Join(dirs[x.level], x.name))
		if e != nil {
			return e
		}
	}
	blobs, e := readBlobs(dirs["v9"], read)
	if e != nil {
		return e
	}
	manifest := bundleManifest(a, blobs)
	compile := func(ctx context.Context, source []byte) ([]byte, error) {
		d, x := os.MkdirTemp("", "seme-v11-compile-*")
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
	base, e := goupb07bundle.Load(context.Background(), goupb07bundle.Input{Contracts: v10, V9: v9, V8: v8, Artifacts: a, Manifest: manifest, Selection: configuration, DurableSelection: durable, Compile: compile, Blobs: blobs})
	if e != nil {
		return e
	}
	model, e := goorderedtransportadapter.Resolve(base.Project, transport)
	if e != nil {
		return e
	}
	ti := orderedtransportinstance.Inputs{Contracts: v11, ProjectV10: base.Project, Model: model}
	ti.Artifact, e = orderedtransportinstance.Emit(ti)
	if e != nil {
		return e
	}
	if e = orderedtransportinstance.Validate(ti); e != nil {
		return e
	}
	pi := projectv11instance.Inputs{Contracts: v11, ProjectV10: base.Project, Transport: ti}
	pi.Composed, e = projectv11instance.Emit(pi)
	if e != nil {
		return e
	}
	if e = projectv11instance.Validate(pi); e != nil {
		return e
	}
	return publish(*v["out"], []named{{"ordered-transport-v1.seme", ti.Artifact}, {"project-v11.seme", pi.Composed}})
}

type named struct {
	name string
	data []byte
}

func bundleManifest(a goupb07bundle.Artifacts, blobs map[[32]byte][]byte) []byte {
	files := []named{{"construction-v36.g1", a.Construction}, {"execution-v36.seme", a.Execution}, {"project-base-v8.seme", a.ProjectBase}, {"inventory-v8.seme", a.Inventory}, {"package-detail-v4.seme", a.PackageDetail}, {"package-v4.seme", a.PackageV4}, {"dependency-v1.seme", a.Dependency}, {"configuration-v3.seme", a.ConfigurationV3}, {"project-v8.seme", a.ProjectV8}, {"resource-v1.seme", a.Resource}, {"project-v9.seme", a.ProjectV9}, {"durable-state-v1.seme", a.Durable}, {"source-presentation-v1.seme", a.Presentation}, {"project-v10.seme", a.ProjectV10}}
	out := []byte("seme-go-upb07-bundle-v1\n")
	for _, f := range files {
		s := sha256.Sum256(f.data)
		out = append(out, []byte(f.name+" "+hex.EncodeToString(s[:])+"\n")...)
	}
	keys := make([][32]byte, 0, len(blobs))
	for k := range blobs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i][:], keys[j][:]) < 0 })
	for _, k := range keys {
		h := hex.EncodeToString(k[:])
		out = append(out, []byte("blobs/"+h+" "+h+"\n")...)
	}
	return out
}
func readBlobs(dir string, read func(string) ([]byte, error)) (map[[32]byte][]byte, error) {
	entries, e := os.ReadDir(filepath.Join(dir, "blobs"))
	if e != nil {
		return nil, e
	}
	out := map[[32]byte][]byte{}
	for _, x := range entries {
		if x.IsDir() || x.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("blob_entry")
		}
		raw, z := hex.DecodeString(x.Name())
		if z != nil || len(raw) != 32 {
			return nil, fmt.Errorf("blob_name")
		}
		var k [32]byte
		copy(k[:], raw)
		b, z := read(filepath.Join(dir, "blobs", x.Name()))
		if z != nil || sha256.Sum256(b) != k {
			return nil, fmt.Errorf("blob_digest")
		}
		out[k] = b
	}
	return out, nil
}
func publish(out string, files []named) error {
	if _, e := os.Lstat(out); !os.IsNotExist(e) {
		return fmt.Errorf("output_exists")
	}
	tmp, e := os.MkdirTemp(filepath.Dir(out), ".project-v11-compose-*")
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
	complete := []byte("seme-project-v11-compose-v1\n")
	for _, f := range files {
		if e = os.WriteFile(filepath.Join(tmp, f.name), f.data, 0600); e != nil {
			return e
		}
		s := sha256.Sum256(f.data)
		complete = append(complete, []byte(f.name+" "+hex.EncodeToString(s[:])+"\n")...)
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
