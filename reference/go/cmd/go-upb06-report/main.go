// Command go-upb06-report emits a deterministic source-free authority report
// only after goupb06bundle authenticates and reproduces the complete bundle.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/goupb06bundle"
	"seme.local/reference/wire"
)

type report struct {
	ProjectContract  string     `json:"project_contract_revision"`
	ResourceContract string     `json:"resource_contract_revision"`
	SnapshotRevision string     `json:"snapshot_revision"`
	Resources        []resource `json:"resources"`
}
type resource struct {
	Identity    string `json:"identity"`
	Path        string `json:"path"`
	Destination string `json:"destination"`
	MediaType   string `json:"media_type"`
	SHA256      string `json:"sha256"`
	Size        uint64 `json:"size"`
	Kind        uint64 `json:"kind"`
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb06-report:", err)
		os.Exit(65)
	}
}

func run(parent context.Context, args []string, stdout io.Writer) error {
	f := flag.NewFlagSet("go-upb06-report", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	paths := map[string]*string{}
	for _, name := range []string{"bundle", "selection", "foundation", "execution", "package", "dependency", "configuration", "resource", "project-v8", "project-v9", "k0", "g1-compiler"} {
		paths[name] = new(string)
		f.StringVar(paths[name], name, "", name)
	}
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	for name, path := range paths {
		if *path == "" || !filepath.IsAbs(*path) || filepath.Clean(*path) != *path {
			return fmt.Errorf("path:%s", name)
		}
	}
	read := func(path string) ([]byte, error) {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 64<<20 {
			return nil, fmt.Errorf("input:%s", filepath.Base(path))
		}
		return os.ReadFile(path)
	}
	contract := func(name string) ([]byte, error) { return read(*paths[name]) }
	foundation, err := contract("foundation")
	if err != nil {
		return err
	}
	execution, err := contract("execution")
	if err != nil {
		return err
	}
	packageV4, err := contract("package")
	if err != nil {
		return err
	}
	dependency, err := contract("dependency")
	if err != nil {
		return err
	}
	configuration, err := contract("configuration")
	if err != nil {
		return err
	}
	resourceV1, err := contract("resource")
	if err != nil {
		return err
	}
	projectV8, err := contract("project-v8")
	if err != nil {
		return err
	}
	projectV9, err := contract("project-v9")
	if err != nil {
		return err
	}
	v8, err := contractcatalog.ResolveProjectContractSetV8(foundation, execution, packageV4, dependency, configuration, projectV8)
	if err != nil {
		return err
	}
	v9, err := contractcatalog.ResolveProjectContractSetV9(foundation, execution, packageV4, dependency, configuration, resourceV1, projectV9)
	if err != nil {
		return err
	}
	selectionBytes, err := read(*paths["selection"])
	if err != nil {
		return err
	}
	selection, err := goconfigurationmanifest.Parse(selectionBytes)
	if err != nil {
		return err
	}
	a, manifest, blobs, err := loadFiles(*paths["bundle"], read)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	loaded, err := goupb06bundle.Load(ctx, goupb06bundle.Input{Contracts: v9, V8: v8, Artifacts: a, Manifest: manifest, Selection: selection, Blobs: blobs, Compile: func(ctx context.Context, source []byte) ([]byte, error) {
		directory, e := os.MkdirTemp("", "seme-upb06-report-")
		if e != nil {
			return nil, e
		}
		defer os.RemoveAll(directory)
		input, output := filepath.Join(directory, "input.g1"), filepath.Join(directory, "output.seme")
		if e = os.WriteFile(input, source, 0600); e != nil {
			return nil, e
		}
		data, e := exec.CommandContext(ctx, *paths["k0"], *paths["g1-compiler"], input, output).CombinedOutput()
		if e != nil {
			return nil, fmt.Errorf("compile:%w:%s", e, data)
		}
		return os.ReadFile(output)
	}})
	if err != nil {
		return err
	}
	out, err := makeReport(v9, loaded)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(out)
}

func loadFiles(directory string, read func(string) ([]byte, error)) (goupb06bundle.Artifacts, []byte, map[[32]byte][]byte, error) {
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return goupb06bundle.Artifacts{}, nil, nil, fmt.Errorf("bundle")
	}
	names := []string{"construction-v36.g1", "execution-v36.seme", "project-base-v8.seme", "inventory-v8.seme", "package-detail-v4.seme", "package-v4.seme", "dependency-v1.seme", "configuration-v3.seme", "project-v8.seme", "resource-v1.seme", "project-v9.seme"}
	values := make([][]byte, len(names))
	for i, name := range names {
		values[i], err = read(filepath.Join(directory, name))
		if err != nil {
			return goupb06bundle.Artifacts{}, nil, nil, err
		}
	}
	manifest, err := read(filepath.Join(directory, "COMPLETE.sha256"))
	if err != nil {
		return goupb06bundle.Artifacts{}, nil, nil, err
	}
	blobDir := filepath.Join(directory, "blobs")
	entries, err := os.ReadDir(blobDir)
	if err != nil || len(entries) == 0 || len(entries) > 16 {
		return goupb06bundle.Artifacts{}, nil, nil, fmt.Errorf("blobs")
	}
	blobs := map[[32]byte][]byte{}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || len(entry.Name()) != 64 {
			return goupb06bundle.Artifacts{}, nil, nil, fmt.Errorf("blob_entry")
		}
		raw, e := hex.DecodeString(entry.Name())
		if e != nil {
			return goupb06bundle.Artifacts{}, nil, nil, fmt.Errorf("blob_name")
		}
		var digest [32]byte
		copy(digest[:], raw)
		data, e := read(filepath.Join(blobDir, entry.Name()))
		if e != nil || sha256.Sum256(data) != digest {
			return goupb06bundle.Artifacts{}, nil, nil, fmt.Errorf("blob_digest")
		}
		blobs[digest] = data
	}
	return goupb06bundle.Artifacts{Construction: values[0], Execution: values[1], ProjectBase: values[2], Inventory: values[3], PackageDetail: values[4], PackageV4: values[5], Dependency: values[6], ConfigurationV3: values[7], ProjectV8: values[8], Resource: values[9], ProjectV9: values[10]}, manifest, blobs, nil
}

func makeReport(contracts contractcatalog.ProjectContractSetV9, loaded goupb06bundle.Result) (report, error) {
	out := report{ProjectContract: contracts.Project().Pin().Revision.String(), ResourceContract: contracts.Resource().Pin().Revision.String()}
	graph, err := wire.Decode(loaded.Artifacts.ProjectV9)
	if err != nil {
		return report{}, err
	}
	for _, entity := range graph.Entities {
		if entity.Schema.String() == "0000000000000000000000000000e024" {
			value := entity.Fields[mustID("e242")]
			if value.Tag != 5 || len(value.Bytes) != 32 || out.SnapshotRevision != "" {
				return report{}, fmt.Errorf("snapshot")
			}
			out.SnapshotRevision = hex.EncodeToString(value.Bytes)
		}
	}
	destinations := map[string]string{}
	for _, placement := range loaded.Resource.Model.Placements {
		destinations[placement.Resource] = placement.Destination
	}
	for _, item := range loaded.Resource.Model.Resources {
		out.Resources = append(out.Resources, resource{Identity: item.Identity, Path: item.Path, Destination: destinations[item.Identity], MediaType: item.MediaType, SHA256: hex.EncodeToString(item.SHA256[:]), Size: item.Size, Kind: item.Kind})
	}
	sort.Slice(out.Resources, func(i, j int) bool { return out.Resources[i].Identity < out.Resources[j].Identity })
	if out.SnapshotRevision == "" || len(out.Resources) == 0 {
		return report{}, fmt.Errorf("authority")
	}
	return out, nil
}

func mustID(value string) wire.ID {
	for len(value) < 32 {
		value = "0" + value
	}
	id, err := wire.ParseID(value)
	if err != nil {
		panic(err)
	}
	return id
}
