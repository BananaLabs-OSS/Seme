// Package goupb09cmdload contains the shared strict loader for UPB09 commands.
package goupb09cmdload

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/gocontrolledeffectsmanifest"
	"seme.local/reference/godurablemanifest"
	"seme.local/reference/goorderedtransportmanifest"
	"seme.local/reference/goupb08cmdload"
	"seme.local/reference/goupb09bundle"
)

type Paths struct {
	Base                                            goupb08cmdload.Paths
	ControlledEffects, ProjectV12, EffectsSelection string
}
type Result struct {
	Bundle                                                                         goupb09bundle.Result
	ConfigurationSelection, DurableSelection, TransportSelection, EffectsSelection []byte
}

func (p *Paths) Bind(f *flag.FlagSet) {
	p.Base.Bind(f)
	f.StringVar(&p.ControlledEffects, "controlled-effects", "", "controlled-effects")
	f.StringVar(&p.ProjectV12, "project-v12", "", "project-v12")
	f.StringVar(&p.EffectsSelection, "effects-selection", "", "effects-selection")
}
func Load(ctx context.Context, p Paths) (Result, error) {
	b := map[string][]byte{}
	for n, path := range p.Base.Values() {
		if n != "bundle" {
			x, e := read(path)
			if e != nil {
				return Result{}, fmt.Errorf("%s:%w", n, e)
			}
			b[n] = x
		}
	}
	for n, path := range map[string]string{"controlled-effects": p.ControlledEffects, "project-v12": p.ProjectV12, "effects-selection": p.EffectsSelection} {
		x, e := read(path)
		if e != nil {
			return Result{}, fmt.Errorf("%s:%w", n, e)
		}
		b[n] = x
	}
	v8, e := contractcatalog.ResolveProjectContractSetV8(b["foundation"], b["execution"], b["package"], b["dependency"], b["configuration"], b["project-v8"])
	if e != nil {
		return Result{}, e
	}
	v9, e := contractcatalog.ResolveProjectContractSetV9(b["foundation"], b["execution"], b["package"], b["dependency"], b["configuration"], b["resource"], b["project-v9"])
	if e != nil {
		return Result{}, e
	}
	v10, e := contractcatalog.ResolveProjectContractSetV10(b["foundation"], b["execution"], b["package"], b["dependency"], b["configuration"], b["resource"], b["durable"], b["presentation"], b["project-v9"], b["project-v10"])
	if e != nil {
		return Result{}, e
	}
	v11, e := contractcatalog.ResolveProjectContractSetV11(b["foundation"], b["execution"], b["package"], b["dependency"], b["configuration"], b["resource"], b["durable"], b["presentation"], b["ordered-transport"], b["project-v9"], b["project-v10"], b["project-v11"])
	if e != nil {
		return Result{}, e
	}
	v12, e := contractcatalog.ResolveProjectContractSetV12(b["foundation"], b["execution"], b["package"], b["dependency"], b["configuration"], b["resource"], b["durable"], b["presentation"], b["ordered-transport"], b["controlled-effects"], b["project-v9"], b["project-v10"], b["project-v11"], b["project-v12"])
	if e != nil {
		return Result{}, e
	}
	c, e := goconfigurationmanifest.Parse(b["selection"])
	if e != nil {
		return Result{}, e
	}
	d, e := godurablemanifest.Parse(b["durable-selection"])
	if e != nil {
		return Result{}, e
	}
	t, e := goorderedtransportmanifest.Parse(b["transport-selection"])
	if e != nil {
		return Result{}, e
	}
	effects, e := gocontrolledeffectsmanifest.Parse(b["effects-selection"])
	if e != nil {
		return Result{}, e
	}
	a, m, blobs, e := goupb09bundle.ReadDirectory(p.Base.Bundle)
	if e != nil {
		return Result{}, e
	}
	compile := func(ctx context.Context, source []byte) ([]byte, error) {
		dir, e := os.MkdirTemp("", "seme-upb09-compile-")
		if e != nil {
			return nil, e
		}
		defer os.RemoveAll(dir)
		in, out := filepath.Join(dir, "in.g1"), filepath.Join(dir, "out.seme")
		if e = os.WriteFile(in, source, 0600); e != nil {
			return nil, e
		}
		data, e := exec.CommandContext(ctx, p.Base.K0, p.Base.Compiler, in, out).CombinedOutput()
		if e != nil {
			return nil, fmt.Errorf("compile:%w:%s", e, data)
		}
		return os.ReadFile(out)
	}
	loaded, e := goupb09bundle.Load(ctx, goupb09bundle.Input{Contracts: v12, V11: v11, V10: v10, V9: v9, V8: v8, Artifacts: a, Manifest: m, Selection: c, DurableSelection: d, TransportSelection: t, EffectsSelection: effects, Compile: compile, Blobs: blobs})
	if e != nil {
		return Result{}, e
	}
	clone := func(x []byte) []byte { return append([]byte(nil), x...) }
	return Result{Bundle: loaded, ConfigurationSelection: clone(b["selection"]), DurableSelection: clone(b["durable-selection"]), TransportSelection: clone(b["transport-selection"]), EffectsSelection: clone(b["effects-selection"])}, nil
}
func read(p string) ([]byte, error) {
	if p == "" || !filepath.IsAbs(p) || filepath.Clean(p) != p {
		return nil, fmt.Errorf("path")
	}
	before, e := os.Lstat(p)
	if e != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 || before.Size() <= 0 || before.Size() > 64<<20 {
		return nil, fmt.Errorf("regular")
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("changed")
	}
	b, e := io.ReadAll(io.LimitReader(f, 64<<20+1))
	if e != nil || int64(len(b)) != opened.Size() {
		return nil, fmt.Errorf("changed")
	}
	after, e := os.Lstat(p)
	if e != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("changed")
	}
	return b, nil
}
