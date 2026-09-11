// Command project-v10-compose binds a neutral durable-state selection and
// source-presentation manifest to an authenticated Project-v9 bundle.
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
	"seme.local/reference/durableinstance"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/godurableadapter"
	"seme.local/reference/godurablemanifest"
	"seme.local/reference/goupb05bundle"
	"seme.local/reference/presentationinstance"
	"seme.local/reference/projectv10instance"
	"seme.local/reference/projectv9instance"
	"seme.local/reference/resourceinstance"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "project-v10-compose:", err)
		os.Exit(1)
	}
}

func run() error {
	names := []string{"bundle-v8", "bundle-v9", "configuration-selection", "durable-selection", "foundation-contract", "execution-contract", "package-contract", "dependency-contract", "configuration-contract", "project-v8-contract", "resource-contract", "project-v9-contract", "durable-state-contract", "source-presentation-contract", "project-v10-contract", "k0", "g1-compiler", "out"}
	values := map[string]*string{}
	for _, name := range names {
		values[name] = new(string)
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
		i, err := os.Lstat(path)
		if err != nil || !i.Mode().IsRegular() || i.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("not_regular:%s", path)
		}
		return os.ReadFile(path)
	}
	contracts := map[string][]byte{}
	for _, name := range []string{"foundation-contract", "execution-contract", "package-contract", "dependency-contract", "configuration-contract", "project-v8-contract", "resource-contract", "project-v9-contract", "durable-state-contract", "source-presentation-contract", "project-v10-contract"} {
		data, err := read(*values[name])
		if err != nil {
			return err
		}
		contracts[name] = data
	}
	v8, err := contractcatalog.ResolveProjectContractSetV8(contracts["foundation-contract"], contracts["execution-contract"], contracts["package-contract"], contracts["dependency-contract"], contracts["configuration-contract"], contracts["project-v8-contract"])
	if err != nil {
		return err
	}
	v9, err := contractcatalog.ResolveProjectContractSetV9(contracts["foundation-contract"], contracts["execution-contract"], contracts["package-contract"], contracts["dependency-contract"], contracts["configuration-contract"], contracts["resource-contract"], contracts["project-v9-contract"])
	if err != nil {
		return err
	}
	v10, err := contractcatalog.ResolveProjectContractSetV10(contracts["foundation-contract"], contracts["execution-contract"], contracts["package-contract"], contracts["dependency-contract"], contracts["configuration-contract"], contracts["resource-contract"], contracts["durable-state-contract"], contracts["source-presentation-contract"], contracts["project-v9-contract"], contracts["project-v10-contract"])
	if err != nil {
		return err
	}
	configurationBytes, err := read(*values["configuration-selection"])
	if err != nil {
		return err
	}
	configuration, err := goconfigurationmanifest.Parse(configurationBytes)
	if err != nil {
		return err
	}
	durableBytes, err := read(*values["durable-selection"])
	if err != nil {
		return err
	}
	durableSelection, err := godurablemanifest.Parse(durableBytes)
	if err != nil {
		return err
	}
	v8dir, v9dir := *values["bundle-v8"], *values["bundle-v9"]
	artifact := func(dir, name string) ([]byte, error) { return read(filepath.Join(dir, name)) }
	a := goupb05bundle.V8Artifacts{}
	for _, part := range []struct {
		name string
		to   *[]byte
	}{{"construction-v36.g1", &a.Construction}, {"execution-v36.seme", &a.Execution}, {"project-base-v8.seme", &a.ProjectBase}, {"inventory-v8.seme", &a.Inventory}, {"package-detail-v4.seme", &a.PackageDetail}, {"package-v4.seme", &a.PackageV4}, {"dependency-v1.seme", &a.Dependency}, {"configuration-v3.seme", &a.ConfigurationV3}, {"project-v8.seme", &a.ProjectV8}} {
		*part.to, err = artifact(v8dir, part.name)
		if err != nil {
			return err
		}
	}
	manifest, err := artifact(v8dir, "COMPLETE.sha256")
	if err != nil {
		return err
	}
	compile := func(ctx context.Context, source []byte) ([]byte, error) {
		dir, e := os.MkdirTemp("", "seme-v10-compile-*")
		if e != nil {
			return nil, e
		}
		defer os.RemoveAll(dir)
		in, out := filepath.Join(dir, "in.g1"), filepath.Join(dir, "out.seme")
		if e = os.WriteFile(in, source, 0600); e != nil {
			return nil, e
		}
		data, e := exec.CommandContext(ctx, *values["k0"], *values["g1-compiler"], in, out).CombinedOutput()
		if e != nil {
			return nil, fmt.Errorf("compile:%w:%s", e, data)
		}
		return os.ReadFile(out)
	}
	base, err := goupb05bundle.LoadV8(context.Background(), goupb05bundle.V8Input{Contracts: v8, Artifacts: a, Manifest: manifest, Selection: configuration, Compile: compile})
	if err != nil {
		return err
	}
	resourceBytes, err := artifact(v9dir, "resource-v1.seme")
	if err != nil {
		return err
	}
	resourceModel, err := resourceinstance.ModelFromArtifact(resourceBytes)
	if err != nil {
		return err
	}
	resource := resourceinstance.Inputs{Contracts: v9, ProjectV8: base.ProjectV8Input, Artifact: resourceBytes, Model: resourceModel}
	blobs, err := readBlobs(v9dir, read)
	if err != nil {
		return err
	}
	if err = resourceinstance.VerifyDetached(resource, blobs); err != nil {
		return err
	}
	projectBytes, err := artifact(v9dir, "project-v9.seme")
	if err != nil {
		return err
	}
	project := projectv9instance.Inputs{Contracts: v9, ProjectV8: base.ProjectV8Input, Resource: resource, Composed: projectBytes}
	if err = projectv9instance.Validate(project); err != nil {
		return err
	}
	model, err := godurableadapter.Resolve(project, durableSelection)
	if err != nil {
		return err
	}
	durable := durableinstance.Inputs{Contracts: v10, ProjectV9: project, Model: model}
	durable.Artifact, err = durableinstance.Emit(durable)
	if err != nil {
		return err
	}
	if err = durableinstance.Validate(durable); err != nil {
		return err
	}
	presentation := presentationinstance.Inputs{Contracts: v10, ProjectV9: project, Model: presentationinstance.Model{}}
	presentation.Artifact, err = presentationinstance.Emit(presentation)
	if err != nil {
		return err
	}
	if err = presentationinstance.Validate(presentation); err != nil {
		return err
	}
	project10 := projectv10instance.Inputs{Contracts: v10, ProjectV9: project, Durable: durable, Presentation: presentation}
	project10.Composed, err = projectv10instance.Emit(project10)
	if err != nil {
		return err
	}
	if err = projectv10instance.Validate(project10); err != nil {
		return err
	}
	return publish(*values["out"], []named{{"durable-state-v1.seme", durable.Artifact}, {"source-presentation-v1.seme", presentation.Artifact}, {"project-v10.seme", project10.Composed}})
}

type named struct {
	name string
	data []byte
}

func readBlobs(dir string, read func(string) ([]byte, error)) (map[[32]byte][]byte, error) {
	entries, err := os.ReadDir(filepath.Join(dir, "blobs"))
	if err != nil {
		return nil, err
	}
	out := map[[32]byte][]byte{}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("blob_entry")
		}
		decoded, e := hex.DecodeString(entry.Name())
		if e != nil || len(decoded) != 32 {
			return nil, fmt.Errorf("blob_name")
		}
		var key [32]byte
		copy(key[:], decoded)
		data, e := read(filepath.Join(dir, "blobs", entry.Name()))
		if e != nil || sha256.Sum256(data) != key {
			return nil, fmt.Errorf("blob_digest")
		}
		out[key] = data
	}
	return out, nil
}

func publish(out string, files []named) error {
	if _, err := os.Lstat(out); !os.IsNotExist(err) {
		return fmt.Errorf("output_exists")
	}
	tmp, err := os.MkdirTemp(filepath.Dir(out), ".project-v10-compose-*")
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(tmp)
		}
	}()
	complete := []byte("seme-project-v10-compose-v1\n")
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	for _, file := range files {
		if err = os.WriteFile(filepath.Join(tmp, file.name), file.data, 0600); err != nil {
			return err
		}
		sum := sha256.Sum256(file.data)
		complete = append(complete, []byte(file.name+" "+hex.EncodeToString(sum[:])+"\n")...)
	}
	if err = os.WriteFile(filepath.Join(tmp, "COMPLETE.sha256"), complete, 0600); err != nil {
		return err
	}
	if err = os.Rename(tmp, out); err != nil {
		return err
	}
	ok = true
	return nil
}
