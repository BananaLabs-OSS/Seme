// Command package-detail-emit converts explicit provider evidence into a
// validated neutral Package-v2 graph bound to a canonical Project execution.
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"seme.local/reference/packagedetail"
	"seme.local/reference/packagedetailemitter"
	"seme.local/reference/wire"
)

type origin struct {
	SourceIdentity, Path, ContentDigest        string
	ByteStart, ByteEnd                         uint64
	StartLine, StartColumn, EndLine, EndColumn uint32
}
type source struct {
	Identity, Path, ContentDigest string
	ByteSize                      uint64
}
type member struct {
	Identity, Name, ExportName string
	Visibility                 uint8
	Origin                     origin
	Callable                   bool
	Parameters, Results        []string
}
type imported struct {
	Alias, Requested, Resolved string
	Class                      uint8
	Origin                     origin
}
type detail struct {
	Identity string
	Root     bool
	Sources  []source
	Members  []member
	Imports  []imported
}
type graph struct{ Packages []detail }

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	var basePath, graphPath, out string
	flag.StringVar(&basePath, "base", "", "canonical Project-v1 artifact")
	flag.StringVar(&graphPath, "graph", "", "provider package graph JSON")
	flag.StringVar(&out, "out", "", "new Package-v2 artifact")
	flag.Parse()
	if basePath == "" || graphPath == "" || out == "" {
		return fmt.Errorf("package_detail_emit.missing")
	}
	baseBytes, err := read(basePath)
	if err != nil {
		return err
	}
	base, err := wire.Decode(baseBytes)
	if err != nil {
		return err
	}
	canonical, err := wire.Encode(base)
	if err != nil || !bytes.Equal(canonical, baseBytes) {
		return fmt.Errorf("package_detail_emit.base_noncanonical")
	}
	raw, err := read(graphPath)
	if err != nil {
		return err
	}
	var input graph
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&input); err != nil {
		return fmt.Errorf("package_detail_emit.graph:%w", err)
	}
	var extra any
	if trailing := decoder.Decode(&extra); trailing != io.EOF {
		return fmt.Errorf("package_detail_emit.graph_trailing")
	}
	g := packagedetail.Graph{}
	for _, p := range input.Packages {
		d := packagedetail.Detail{Identity: p.Identity, Root: p.Root}
		for _, s := range p.Sources {
			digest, e := digest(s.ContentDigest)
			if e != nil {
				return e
			}
			d.Sources = append(d.Sources, packagedetail.Source{Identity: s.Identity, Path: s.Path, ContentDigest: digest, ByteSize: s.ByteSize})
		}
		for _, m := range p.Members {
			o, e := convertOrigin(m.Origin)
			if e != nil {
				return e
			}
			d.Members = append(d.Members, packagedetail.Member{Identity: m.Identity, Name: m.Name, ExportName: m.ExportName, Visibility: packagedetail.Visibility(m.Visibility), Origin: o, Callable: m.Callable, Parameters: m.Parameters, Results: m.Results})
		}
		for _, im := range p.Imports {
			o, e := convertOrigin(im.Origin)
			if e != nil {
				return e
			}
			d.Imports = append(d.Imports, packagedetail.Import{Alias: im.Alias, Requested: im.Requested, Resolved: im.Resolved, Class: packagedetail.ImportClass(im.Class), Origin: o})
		}
		g.Packages = append(g.Packages, d)
	}
	artifact, err := packagedetailemitter.Emit(base, g)
	if err != nil {
		return err
	}
	return publish(out, artifact)
}
func convertOrigin(x origin) (packagedetail.Origin, error) {
	d, e := digest(x.ContentDigest)
	if e != nil {
		return packagedetail.Origin{}, e
	}
	return packagedetail.Origin{SourceIdentity: x.SourceIdentity, Path: x.Path, ContentDigest: d, ByteStart: x.ByteStart, ByteEnd: x.ByteEnd, StartLine: x.StartLine, StartColumn: x.StartColumn, EndLine: x.EndLine, EndColumn: x.EndColumn}, nil
}
func digest(raw string) ([32]byte, error) {
	var out [32]byte
	b, e := hex.DecodeString(raw)
	if e != nil || len(b) != len(out) {
		return out, fmt.Errorf("package_detail_emit.digest")
	}
	copy(out[:], b)
	return out, nil
}
func read(path string) ([]byte, error) {
	i, e := os.Lstat(path)
	if e != nil || !i.Mode().IsRegular() {
		return nil, fmt.Errorf("package_detail_emit.not_regular:%s", path)
	}
	return os.ReadFile(path)
}
func publish(path string, data []byte) (err error) {
	absolute, e := filepath.Abs(path)
	if e != nil {
		return e
	}
	if _, e = os.Lstat(absolute); e == nil {
		return fmt.Errorf("package_detail_emit.output_exists")
	} else if !os.IsNotExist(e) {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(absolute), ".package-detail-emit-*")
	if e != nil {
		return e
	}
	name := f.Name()
	linked := false
	defer func() {
		if !linked {
			_ = os.Remove(name)
		}
	}()
	if _, e = f.Write(data); e != nil {
		_ = f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		_ = f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = os.Link(name, absolute); e != nil {
		return e
	}
	linked = true
	return os.Remove(name)
}
