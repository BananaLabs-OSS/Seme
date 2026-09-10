// Package upb07testfixture materializes one authenticated UPB07 bundle for
// integration tests. It is test infrastructure, not a provider or runtime.
package upb07testfixture

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/godurablemanifest"
	"seme.local/reference/goupb07bundle"
)

type Fixture struct {
	Result               goupb07bundle.Result
	Root, Source, Bundle string
}

func Build(ctx context.Context, work string) (Fixture, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return Fixture{}, fmt.Errorf("fixture caller")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(filename), "../../../.."))
	source, bundle, tool := filepath.Join(work, "source"), filepath.Join(work, "bundle"), filepath.Join(work, "go-upb07-build")
	if out, err := exec.CommandContext(ctx, filepath.Join(repo, "scripts/materialize-go-upb07-fixture.sh"), source).CombinedOutput(); err != nil {
		return Fixture{}, fmt.Errorf("materialize:%w:%s", err, out)
	}
	goCmd := exec.CommandContext(ctx, "go", "build", "-buildvcs=false", "-o", tool, "./cmd/go-upb07-build")
	goCmd.Dir = filepath.Join(repo, "reference/go")
	goCmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(work, "go-cache"))
	if out, err := goCmd.CombinedOutput(); err != nil {
		return Fixture{}, fmt.Errorf("build-tool:%w:%s", err, out)
	}
	args := []string{"-project", source, "-proxy", filepath.Join(repo, "fixtures/go-upb03-offline-proxy"), "-module", "example.test/go-uab-11", "-package", "example.test/go-uab-11/service", "-entry", "ApplyConfiguredResource", "-revision", "1", "-out", bundle, "-dependency", "example.test/seme/checksum", "-version", "v1.2.3", "-local-from", "example.test/go-uab-11/application", "-local-to", "example.test/go-uab-11/model", "-selection", filepath.Join(repo, "fixtures/go-upb05-configuration-overlay/configuration-selection.json"), "-resources", filepath.Join(source, "resources.json"), "-resource-owner", "example.test/go-uab-11/service", "-durable-selection", filepath.Join(source, "durable-selection.json"), "-execution-g1", filepath.Join(repo, "modules/execution/v36/module.g1"), "-execution-contract", filepath.Join(repo, "modules/execution/v36/module.seme"), "-foundation-contract", filepath.Join(repo, "modules/foundation/v1/module.seme"), "-package-v4", filepath.Join(repo, "modules/package/v4/module.seme"), "-project-v8", filepath.Join(repo, "modules/project/v8/module.seme"), "-dependency-v1", filepath.Join(repo, "modules/dependency/v1/module.seme"), "-configuration-v3", filepath.Join(repo, "modules/configuration/v3/module.seme"), "-resource-v1", filepath.Join(repo, "modules/resource/v1/module.seme"), "-project-v9", filepath.Join(repo, "modules/project/v9/module.seme"), "-durable-state-v1", filepath.Join(repo, "modules/durable-state/v1/module.seme"), "-source-presentation-v1", filepath.Join(repo, "modules/source-presentation/v1/module.seme"), "-project-v10", filepath.Join(repo, "modules/project/v10/module.seme"), "-k0", filepath.Join(repo, "bootstrap/seme-k0-linux-amd64"), "-g1-compiler", filepath.Join(repo, "compiler/g1-compiler.k0")}
	if out, err := exec.CommandContext(ctx, tool, args...).CombinedOutput(); err != nil {
		return Fixture{}, fmt.Errorf("build:%w:%s", err, out)
	}
	contractPaths := map[string]string{"foundation": "modules/foundation/v1/module.seme", "execution": "modules/execution/v36/module.seme", "package": "modules/package/v4/module.seme", "dependency": "modules/dependency/v1/module.seme", "configuration": "modules/configuration/v3/module.seme", "resource": "modules/resource/v1/module.seme", "durable": "modules/durable-state/v1/module.seme", "presentation": "modules/source-presentation/v1/module.seme", "project-v8": "modules/project/v8/module.seme", "project-v9": "modules/project/v9/module.seme", "project-v10": "modules/project/v10/module.seme"}
	contracts := map[string][]byte{}
	for name, path := range contractPaths {
		data, readErr := os.ReadFile(filepath.Join(repo, path))
		if readErr != nil {
			return Fixture{}, fmt.Errorf("contract %s: %w", name, readErr)
		}
		contracts[name] = data
	}
	f, e, p := contracts["foundation"], contracts["execution"], contracts["package"]
	d, c, r := contracts["dependency"], contracts["configuration"], contracts["resource"]
	ds, sp := contracts["durable"], contracts["presentation"]
	p8, p9, p10 := contracts["project-v8"], contracts["project-v9"], contracts["project-v10"]
	v8, err := contractcatalog.ResolveProjectContractSetV8(f, e, p, d, c, p8)
	if err != nil {
		return Fixture{}, err
	}
	v9, err := contractcatalog.ResolveProjectContractSetV9(f, e, p, d, c, r, p9)
	if err != nil {
		return Fixture{}, err
	}
	v10, err := contractcatalog.ResolveProjectContractSetV10(f, e, p, d, c, r, ds, sp, p9, p10)
	if err != nil {
		return Fixture{}, err
	}
	selBytes, err := os.ReadFile(filepath.Join(repo, "fixtures/go-upb05-configuration-overlay/configuration-selection.json"))
	if err != nil {
		return Fixture{}, err
	}
	sel, err := goconfigurationmanifest.Parse(selBytes)
	if err != nil {
		return Fixture{}, err
	}
	durableBytes, err := os.ReadFile(filepath.Join(source, "durable-selection.json"))
	if err != nil {
		return Fixture{}, err
	}
	durable, err := godurablemanifest.Parse(durableBytes)
	if err != nil {
		return Fixture{}, err
	}
	a, manifest, blobs, err := goupb07bundle.ReadDirectory(bundle)
	if err != nil {
		return Fixture{}, err
	}
	result, err := goupb07bundle.Load(ctx, goupb07bundle.Input{Contracts: v10, V9: v9, V8: v8, Artifacts: a, Manifest: manifest, Selection: sel, DurableSelection: durable, Blobs: blobs, Compile: func(ctx context.Context, source []byte) ([]byte, error) {
		dir, er := os.MkdirTemp(work, "compile-")
		if er != nil {
			return nil, er
		}
		defer os.RemoveAll(dir)
		in, out := filepath.Join(dir, "in.g1"), filepath.Join(dir, "out.seme")
		if er = os.WriteFile(in, source, 0600); er != nil {
			return nil, er
		}
		data, er := exec.CommandContext(ctx, filepath.Join(repo, "bootstrap/seme-k0-linux-amd64"), filepath.Join(repo, "compiler/g1-compiler.k0"), in, out).CombinedOutput()
		if er != nil {
			return nil, fmt.Errorf("compile:%w:%s", er, data)
		}
		return os.ReadFile(out)
	}})
	if err != nil {
		return Fixture{}, err
	}
	return Fixture{Result: result, Root: repo, Source: source, Bundle: bundle}, nil
}
