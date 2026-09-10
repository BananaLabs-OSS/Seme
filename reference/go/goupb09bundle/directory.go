package goupb09bundle

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"seme.local/reference/goupb07bundle"
	"seme.local/reference/goupb08bundle"
	"strings"
)

func ReadDirectory(root string) (Artifacts, []byte, map[[32]byte][]byte, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb09_directory.path")
	}
	i, e := os.Lstat(root)
	if e != nil || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb09_directory.root")
	}
	if r, e := filepath.EvalSymlinks(root); e != nil || r != root {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb09_directory.symlink")
	}
	allowed := map[string]bool{"COMPLETE.sha256": true}
	for _, f := range files(Artifacts{}) {
		allowed[f.name] = true
	}
	data := map[string][]byte{}
	blobs := map[[32]byte][]byte{}
	sawBlobs := false
	e = filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if rel == "blobs" {
				sawBlobs = true
				return nil
			}
			return fmt.Errorf("go_upb09_directory.extra:%s", rel)
		}
		if d.Type()&os.ModeSymlink != 0 || !d.Type().IsRegular() {
			return fmt.Errorf("go_upb09_directory.nonregular")
		}
		b, e := readFile(p)
		if e != nil {
			return e
		}
		if strings.HasPrefix(rel, "blobs/") {
			raw, e := hex.DecodeString(strings.TrimPrefix(rel, "blobs/"))
			if e != nil || len(raw) != 32 {
				return fmt.Errorf("go_upb09_directory.blob_name")
			}
			var s [32]byte
			copy(s[:], raw)
			if _, ok := blobs[s]; ok {
				return fmt.Errorf("go_upb09_directory.blob_duplicate")
			}
			blobs[s] = b
			return nil
		}
		if !allowed[rel] {
			return fmt.Errorf("go_upb09_directory.extra:%s", rel)
		}
		if _, ok := data[rel]; ok {
			return fmt.Errorf("go_upb09_directory.duplicate")
		}
		data[rel] = b
		return nil
	})
	if e != nil {
		return Artifacts{}, nil, nil, e
	}
	if !sawBlobs {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb09_directory.blobs")
	}
	for _, f := range files(Artifacts{}) {
		if data[f.name] == nil {
			return Artifacts{}, nil, nil, fmt.Errorf("go_upb09_directory.missing:%s", f.name)
		}
	}
	if data["COMPLETE.sha256"] == nil {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb09_directory.complete")
	}
	b := goupb07bundle.Artifacts{Construction: data["construction-v36.g1"], Execution: data["execution-v36.seme"], ProjectBase: data["project-base-v8.seme"], Inventory: data["inventory-v8.seme"], PackageDetail: data["package-detail-v4.seme"], PackageV4: data["package-v4.seme"], Dependency: data["dependency-v1.seme"], ConfigurationV3: data["configuration-v3.seme"], ProjectV8: data["project-v8.seme"], Resource: data["resource-v1.seme"], ProjectV9: data["project-v9.seme"], Durable: data["durable-state-v1.seme"], Presentation: data["source-presentation-v1.seme"], ProjectV10: data["project-v10.seme"]}
	base := goupb08bundle.Artifacts{Base: b, Transport: data["ordered-transport-v1.seme"], ProjectV11: data["project-v11.seme"]}
	return Artifacts{Base: base, ControlledEffects: data["controlled-effects-v1.seme"], ProjectV12: data["project-v12.seme"], ReplayAuthority: data["controlled-replay-v1.json"]}, data["COMPLETE.sha256"], blobs, nil
}

func WriteDirectory(destination string, a Artifacts, blobs map[[32]byte][]byte) error {
	if !filepath.IsAbs(destination) || filepath.Clean(destination) != destination {
		return fmt.Errorf("go_upb09_publish.path")
	}
	if _, e := os.Lstat(destination); !os.IsNotExist(e) {
		return fmt.Errorf("go_upb09_publish.exists")
	}
	parent := filepath.Dir(destination)
	real, e := filepath.EvalSymlinks(parent)
	if e != nil || real != parent {
		return fmt.Errorf("go_upb09_publish.parent")
	}
	complete, ok := manifest("seme-go-upb09-bundle-v1", files(a), blobs)
	if !ok {
		return fmt.Errorf("go_upb09_publish.authority")
	}
	tmp, e := os.MkdirTemp(parent, ".seme-upb09-")
	if e != nil {
		return e
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(tmp)
		}
	}()
	if e = os.Mkdir(filepath.Join(tmp, "blobs"), 0700); e != nil {
		return e
	}
	for _, f := range files(a) {
		if e = os.WriteFile(filepath.Join(tmp, f.name), f.data, 0600); e != nil {
			return e
		}
	}
	for k, b := range blobs {
		if e = os.WriteFile(filepath.Join(tmp, "blobs", hex.EncodeToString(k[:])), b, 0600); e != nil {
			return e
		}
	}
	if e = os.WriteFile(filepath.Join(tmp, "COMPLETE.sha256"), complete, 0600); e != nil {
		return e
	}
	if e = os.Rename(tmp, destination); e != nil {
		return fmt.Errorf("go_upb09_publish.commit:%w", e)
	}
	keep = true
	return nil
}
func readFile(p string) ([]byte, error) {
	before, e := os.Lstat(p)
	if e != nil || !before.Mode().IsRegular() || before.Size() > 64<<20 {
		return nil, fmt.Errorf("go_upb09_directory.file")
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("go_upb09_directory.changed")
	}
	b, e := io.ReadAll(io.LimitReader(f, 64<<20+1))
	if e != nil || int64(len(b)) != opened.Size() {
		return nil, fmt.Errorf("go_upb09_directory.changed")
	}
	after, e := os.Lstat(p)
	if e != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("go_upb09_directory.changed")
	}
	return b, nil
}
