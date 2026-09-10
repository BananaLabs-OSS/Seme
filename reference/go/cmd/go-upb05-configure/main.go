// Command go-upb05-configure executes one typed startup request only after the
// complete Project-v8 bundle authenticates and reproduces.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"seme.local/reference/canonicaleval"
	"seme.local/reference/configurationexecutor"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/goupb05bundle"
	"seme.local/reference/wire"
)

type request struct {
	ConfigurationInputs map[string]canonicaleval.Value `json:"configuration_inputs"`
	RuntimeInputs       map[string]canonicaleval.Value `json:"runtime_inputs"`
	CapabilityGrants    []string                       `json:"capability_grants"`
}

type response struct {
	Authority authority                    `json:"authority"`
	Execution configurationexecutor.Result `json:"execution"`
}

type authority struct {
	ContractRevision string `json:"contract_revision"`
	SnapshotRevision string `json:"snapshot_revision"`
	PackageCount     int    `json:"package_count"`
	InitializerCount int    `json:"initializer_count"`
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb05-configure:", err)
		os.Exit(65)
	}
}

func run(parent context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	set := flag.NewFlagSet("go-upb05-configure", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	paths := map[string]*string{}
	for _, name := range []string{"bundle", "selection", "foundation", "execution", "package", "dependency", "configuration", "project", "k0", "g1-compiler"} {
		paths[name] = new(string)
		set.StringVar(paths[name], name, "", name)
	}
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	for name, value := range paths {
		if *value == "" || !filepath.IsAbs(*value) || filepath.Clean(*value) != *value {
			return fmt.Errorf("path:%s", name)
		}
	}
	read := func(path string) ([]byte, error) {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() < 0 || info.Size() > 64<<20 {
			return nil, fmt.Errorf("input")
		}
		return os.ReadFile(path)
	}
	mustRead := func(path string) ([]byte, error) { return read(path) }
	foundation, err := mustRead(*paths["foundation"])
	if err != nil {
		return err
	}
	execution, err := mustRead(*paths["execution"])
	if err != nil {
		return err
	}
	packageContract, err := mustRead(*paths["package"])
	if err != nil {
		return err
	}
	dependency, err := mustRead(*paths["dependency"])
	if err != nil {
		return err
	}
	configuration, err := mustRead(*paths["configuration"])
	if err != nil {
		return err
	}
	project, err := mustRead(*paths["project"])
	if err != nil {
		return err
	}
	contracts, err := contractcatalog.ResolveProjectContractSetV8(foundation, execution, packageContract, dependency, configuration, project)
	if err != nil {
		return err
	}
	selectionBytes, err := mustRead(*paths["selection"])
	if err != nil {
		return err
	}
	selection, err := goconfigurationmanifest.Parse(selectionBytes)
	if err != nil {
		return err
	}
	bundle := *paths["bundle"]
	artifact := func(name string) ([]byte, error) { return mustRead(filepath.Join(bundle, name)) }
	construction, err := artifact("construction-v36.g1")
	if err != nil {
		return err
	}
	executable, err := artifact("execution-v36.seme")
	if err != nil {
		return err
	}
	projectBase, err := artifact("project-base-v8.seme")
	if err != nil {
		return err
	}
	inventory, err := artifact("inventory-v8.seme")
	if err != nil {
		return err
	}
	packageDetail, err := artifact("package-detail-v4.seme")
	if err != nil {
		return err
	}
	packageV4, err := artifact("package-v4.seme")
	if err != nil {
		return err
	}
	dependencyArtifact, err := artifact("dependency-v1.seme")
	if err != nil {
		return err
	}
	configurationV3, err := artifact("configuration-v3.seme")
	if err != nil {
		return err
	}
	projectV8, err := artifact("project-v8.seme")
	if err != nil {
		return err
	}
	manifest, err := artifact("COMPLETE.sha256")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	loaded, err := goupb05bundle.LoadV8(ctx, goupb05bundle.V8Input{Contracts: contracts, Manifest: manifest, Selection: selection, Artifacts: goupb05bundle.V8Artifacts{Construction: construction, Execution: executable, ProjectBase: projectBase, Inventory: inventory, PackageDetail: packageDetail, PackageV4: packageV4, Dependency: dependencyArtifact, ConfigurationV3: configurationV3, ProjectV8: projectV8}, Compile: func(ctx context.Context, source []byte) ([]byte, error) {
		directory, tempErr := os.MkdirTemp("", "seme-upb05-configure-")
		if tempErr != nil {
			return nil, tempErr
		}
		defer os.RemoveAll(directory)
		input, output := filepath.Join(directory, "input.g1"), filepath.Join(directory, "output.seme")
		if tempErr = os.WriteFile(input, source, 0600); tempErr != nil {
			return nil, tempErr
		}
		command := exec.CommandContext(ctx, *paths["k0"], *paths["g1-compiler"], input, output)
		if diagnostic, commandErr := command.CombinedOutput(); commandErr != nil {
			return nil, fmt.Errorf("compile:%w:%s", commandErr, diagnostic)
		}
		return os.ReadFile(output)
	}})
	if err != nil {
		return err
	}
	var input request
	decoder := json.NewDecoder(stdin)
	if err = decoder.Decode(&input); err != nil {
		return err
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("request_count")
	}
	grants := map[string]bool{}
	for _, grant := range input.CapabilityGrants {
		if grant == "" || grants[grant] {
			return fmt.Errorf("capability_grants")
		}
		grants[grant] = true
	}
	result, err := configurationexecutor.ExecuteV3(configurationexecutor.V3Input{Project: loaded.ProjectV8Input, ConfigurationInputs: input.ConfigurationInputs, RuntimeInputs: input.RuntimeInputs, CapabilityGrants: grants})
	if err != nil {
		return err
	}
	report, err := authorityReport(contracts, projectV8, len(result.Outputs))
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(response{Authority: report, Execution: result})
}

func authorityReport(contracts contractcatalog.ProjectContractSetV8, artifact []byte, initializers int) (authority, error) {
	graph, err := wire.Decode(artifact)
	if err != nil {
		return authority{}, err
	}
	report := authority{ContractRevision: contracts.Project().Pin().Revision.String(), InitializerCount: initializers}
	for _, entity := range graph.Entities {
		switch entity.Schema.String() {
		case "0000000000000000000000000000e023":
			value, ok := entity.Fields[mustID("e235")]
			if !ok || value.Tag != 5 || report.SnapshotRevision != "" {
				return authority{}, fmt.Errorf("project_snapshot")
			}
			report.SnapshotRevision = hex.EncodeToString(value.Bytes)
		case "0000000000000000000000000000b021":
			report.PackageCount++
		}
	}
	if report.SnapshotRevision == "" || report.PackageCount == 0 {
		return authority{}, fmt.Errorf("project_authority")
	}
	return report, nil
}

func mustID(short string) wire.ID {
	for len(short) < 32 {
		short = "0" + short
	}
	value, err := wire.ParseID(short)
	if err != nil {
		panic(err)
	}
	return value
}
