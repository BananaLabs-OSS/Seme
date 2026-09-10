// Package goupb08cmdload contains the shared strict loader for UPB08 commands.
package goupb08cmdload

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/godurablemanifest"
	"seme.local/reference/goorderedtransportmanifest"
	"seme.local/reference/goupb08bundle"
)

type Paths struct{ Bundle, ConfigurationSelection, DurableSelection, TransportSelection, Foundation, Execution, Package, Dependency, Configuration, Resource, Durable, Presentation, ProjectV8, ProjectV9, ProjectV10, OrderedTransport, ProjectV11, K0, Compiler string }

func (p *Paths) Bind(f *flag.FlagSet) {
	for n, v := range map[string]*string{"bundle": &p.Bundle, "selection": &p.ConfigurationSelection, "durable-selection": &p.DurableSelection, "transport-selection": &p.TransportSelection, "foundation": &p.Foundation, "execution": &p.Execution, "package": &p.Package, "dependency": &p.Dependency, "configuration": &p.Configuration, "resource": &p.Resource, "durable-state": &p.Durable, "source-presentation": &p.Presentation, "project-v8": &p.ProjectV8, "project-v9": &p.ProjectV9, "project-v10": &p.ProjectV10, "ordered-transport": &p.OrderedTransport, "project-v11": &p.ProjectV11, "k0": &p.K0, "g1-compiler": &p.Compiler} {
		f.StringVar(v, n, "", n)
	}
}
func (p Paths) Values() map[string]string {
	return map[string]string{"bundle": p.Bundle, "selection": p.ConfigurationSelection, "durable-selection": p.DurableSelection, "transport-selection": p.TransportSelection, "foundation": p.Foundation, "execution": p.Execution, "package": p.Package, "dependency": p.Dependency, "configuration": p.Configuration, "resource": p.Resource, "durable": p.Durable, "presentation": p.Presentation, "project-v8": p.ProjectV8, "project-v9": p.ProjectV9, "project-v10": p.ProjectV10, "ordered-transport": p.OrderedTransport, "project-v11": p.ProjectV11, "k0": p.K0, "compiler": p.Compiler}
}
func Load(ctx context.Context, p Paths) (goupb08bundle.Result, error) {
	b := map[string][]byte{}
	for n, path := range p.Values() {
		if n == "bundle" {
			continue
		}
		x, e := read(path)
		if e != nil {
			return goupb08bundle.Result{}, fmt.Errorf("%s:%w", n, e)
		}
		b[n] = x
	}
	v8, e := contractcatalog.ResolveProjectContractSetV8(b["foundation"], b["execution"], b["package"], b["dependency"], b["configuration"], b["project-v8"])
	if e != nil {
		return goupb08bundle.Result{}, e
	}
	v9, e := contractcatalog.ResolveProjectContractSetV9(b["foundation"], b["execution"], b["package"], b["dependency"], b["configuration"], b["resource"], b["project-v9"])
	if e != nil {
		return goupb08bundle.Result{}, e
	}
	v10, e := contractcatalog.ResolveProjectContractSetV10(b["foundation"], b["execution"], b["package"], b["dependency"], b["configuration"], b["resource"], b["durable"], b["presentation"], b["project-v9"], b["project-v10"])
	if e != nil {
		return goupb08bundle.Result{}, e
	}
	v11, e := contractcatalog.ResolveProjectContractSetV11(b["foundation"], b["execution"], b["package"], b["dependency"], b["configuration"], b["resource"], b["durable"], b["presentation"], b["ordered-transport"], b["project-v9"], b["project-v10"], b["project-v11"])
	if e != nil {
		return goupb08bundle.Result{}, e
	}
	c, e := goconfigurationmanifest.Parse(b["selection"])
	if e != nil {
		return goupb08bundle.Result{}, e
	}
	d, e := godurablemanifest.Parse(b["durable-selection"])
	if e != nil {
		return goupb08bundle.Result{}, e
	}
	t, e := goorderedtransportmanifest.Parse(b["transport-selection"])
	if e != nil {
		return goupb08bundle.Result{}, e
	}
	a, m, blobs, e := goupb08bundle.ReadDirectory(p.Bundle)
	if e != nil {
		return goupb08bundle.Result{}, e
	}
	compile := func(ctx context.Context, source []byte) ([]byte, error) {
		dir, e := os.MkdirTemp("", "seme-upb08-compile-")
		if e != nil {
			return nil, e
		}
		defer os.RemoveAll(dir)
		in, out := filepath.Join(dir, "in.g1"), filepath.Join(dir, "out.seme")
		if e = os.WriteFile(in, source, 0600); e != nil {
			return nil, e
		}
		data, e := exec.CommandContext(ctx, p.K0, p.Compiler, in, out).CombinedOutput()
		if e != nil {
			return nil, fmt.Errorf("compile:%w:%s", e, data)
		}
		return os.ReadFile(out)
	}
	return goupb08bundle.Load(ctx, goupb08bundle.Input{Contracts: v11, V10: v10, V9: v9, V8: v8, Artifacts: a, Manifest: m, Selection: c, DurableSelection: d, TransportSelection: t, Compile: compile, Blobs: blobs})
}
func read(p string) ([]byte, error) {
	if p == "" || !filepath.IsAbs(p) || filepath.Clean(p) != p {
		return nil, fmt.Errorf("path")
	}
	r, e := filepath.EvalSymlinks(p)
	if e != nil || r != p {
		return nil, fmt.Errorf("symlink")
	}
	i, e := os.Lstat(p)
	if e != nil || !i.Mode().IsRegular() || i.Size() <= 0 || i.Size() > 64<<20 {
		return nil, fmt.Errorf("regular")
	}
	return os.ReadFile(p)
}
