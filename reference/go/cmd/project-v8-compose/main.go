// Command project-v8-compose binds a provider-neutral canonical project,
// dependency closure, and declarative lifecycle into Configuration v3 and
// Project v8 authorities.
package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"seme.local/reference/configurationinstance"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationadapter"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/projectv8instance"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "project-v8-compose:", err)
		os.Exit(1)
	}
}
func run() error {
	names := []string{"construction", "project-base", "inventory", "package-v2", "package-v4", "dependency", "selection", "foundation-contract", "execution-contract", "package-contract", "dependency-contract", "configuration-contract", "project-contract", "k0", "g1-compiler", "out"}
	paths := map[string]*string{}
	for _, name := range names {
		value := ""
		paths[name] = &value
		flag.StringVar(paths[name], name, "", name)
	}
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	for _, name := range names {
		value := *paths[name]
		if value == "" || !filepath.IsAbs(value) || filepath.Clean(value) != value {
			return fmt.Errorf("flag:%s", name)
		}
	}
	read := func(name string) ([]byte, error) {
		info, err := os.Lstat(*paths[name])
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("not_regular:%s", name)
		}
		return os.ReadFile(*paths[name])
	}
	inputs := map[string][]byte{}
	for _, name := range names[:15] {
		value, err := read(name)
		if err != nil {
			return err
		}
		inputs[name] = value
	}
	contracts, err := contractcatalog.ResolveProjectContractSetV8(inputs["foundation-contract"], inputs["execution-contract"], inputs["package-contract"], inputs["dependency-contract"], inputs["configuration-contract"], inputs["project-contract"])
	if err != nil {
		return err
	}
	selection, err := goconfigurationmanifest.Parse(inputs["selection"])
	if err != nil {
		return err
	}
	compile := func(ctx context.Context, source []byte) ([]byte, error) {
		dir, err := os.MkdirTemp("", "seme-project-v8-compile-*")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(dir)
		in, out := filepath.Join(dir, "in.g1"), filepath.Join(dir, "out.seme")
		if err = os.WriteFile(in, source, 0600); err != nil {
			return nil, err
		}
		command := exec.CommandContext(ctx, *paths["k0"], *paths["g1-compiler"], in, out)
		if result, runErr := command.CombinedOutput(); runErr != nil {
			return nil, fmt.Errorf("compiler:%w:%s", runErr, result)
		}
		return os.ReadFile(out)
	}
	plan, err := goconfigurationadapter.ResolveNeutralV8(context.Background(), goconfigurationadapter.NeutralInputV8{CanonicalG1: inputs["construction"], Compile: compile, Contracts: contracts, PackageV2: inputs["package-v2"], PackageV4: inputs["package-v4"], Fields: selection.Fields, Runtime: selection.Runtime, Units: selection.Units})
	if err != nil {
		return err
	}
	baseModel, boundModel, err := goconfigurationadapter.Models(plan)
	if err != nil {
		return err
	}
	base := configurationinstance.V3BaseInput{Contracts: contracts, PackageV2: inputs["package-v2"], PackageV4: inputs["package-v4"], Model: baseModel}
	base.Artifact, err = configurationinstance.EmitV3Base(base)
	if err != nil {
		return err
	}
	configuration := configurationinstance.V3Input{Contracts: contracts, Base: base, Model: boundModel}
	configuration.Artifact, err = configurationinstance.EmitV3(configuration)
	if err != nil {
		return err
	}
	execution, err := compile(context.Background(), inputs["construction"])
	if err != nil {
		return err
	}
	project := projectv8instance.Inputs{Contracts: contracts, ProjectBase: inputs["project-base"], Inventory: inputs["inventory"], PackageV2: inputs["package-v2"], PackageV4: inputs["package-v4"], Dependency: inputs["dependency"], Construction: execution, Configuration: configuration}
	project.Composed, err = projectv8instance.Emit(project)
	if err != nil {
		return err
	}
	if err = projectv8instance.Validate(project); err != nil {
		return err
	}
	out := *paths["out"]
	if _, err = os.Lstat(out); !os.IsNotExist(err) {
		return fmt.Errorf("output_exists")
	}
	parent := filepath.Dir(out)
	tmp, err := os.MkdirTemp(parent, ".project-v8-compose-*")
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(tmp)
		}
	}()
	files := map[string][]byte{"configuration-v3.seme": configuration.Artifact, "project-v8.seme": project.Composed}
	manifest := []byte("seme-project-v8-compose-v1\n")
	for _, name := range []string{"configuration-v3.seme", "project-v8.seme"} {
		if err = os.WriteFile(filepath.Join(tmp, name), files[name], 0600); err != nil {
			return err
		}
		sum := sha256.Sum256(files[name])
		manifest = append(manifest, []byte(fmt.Sprintf("%s %x\n", name, sum))...)
	}
	if err = os.WriteFile(filepath.Join(tmp, "COMPLETE.sha256"), manifest, 0600); err != nil {
		return err
	}
	if err = os.Rename(tmp, out); err != nil {
		return err
	}
	ok = true
	return nil
}
