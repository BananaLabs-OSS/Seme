package goupb08bundle

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"seme.local/reference/goupb07bundle"
)

// ReadDirectory accepts exactly the UPB08 closed artifact set and digest-named blobs.
func ReadDirectory(root string) (Artifacts, []byte, map[[32]byte][]byte, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb08_directory.path")
	}
	i, e := os.Lstat(root)
	if e != nil || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb08_directory.root")
	}
	if r, e := filepath.EvalSymlinks(root); e != nil || r != root {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb08_directory.symlink")
	}
	allowed := map[string]bool{"COMPLETE.sha256": true}
	for _, f := range files(Artifacts{}) {
		allowed[f.name] = true
	}
	data := map[string][]byte{}
	blobs := map[[32]byte][]byte{}
	saw := false
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
				saw = true
				return nil
			}
			return fmt.Errorf("go_upb08_directory.extra:%s", rel)
		}
		if d.Type()&os.ModeSymlink != 0 || !d.Type().IsRegular() {
			return fmt.Errorf("go_upb08_directory.nonregular")
		}
		b, e := readFile(p)
		if e != nil {
			return e
		}
		if strings.HasPrefix(rel, "blobs/") {
			raw, e := hex.DecodeString(strings.TrimPrefix(rel, "blobs/"))
			if e != nil || len(raw) != 32 {
				return fmt.Errorf("go_upb08_directory.blob_name")
			}
			var s [32]byte
			copy(s[:], raw)
			if _, ok := blobs[s]; ok {
				return fmt.Errorf("go_upb08_directory.blob_duplicate")
			}
			blobs[s] = b
			return nil
		}
		if !allowed[rel] {
			return fmt.Errorf("go_upb08_directory.extra:%s", rel)
		}
		data[rel] = b
		return nil
	})
	if e != nil {
		return Artifacts{}, nil, nil, e
	}
	if !saw {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb08_directory.blobs")
	}
	for _, f := range files(Artifacts{}) {
		if data[f.name] == nil {
			return Artifacts{}, nil, nil, fmt.Errorf("go_upb08_directory.missing:%s", f.name)
		}
	}
	if data["COMPLETE.sha256"] == nil {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb08_directory.complete")
	}
	b := goupb07Artifacts(data)
	return Artifacts{Base: b, Transport: data["ordered-transport-v1.seme"], ProjectV11: data["project-v11.seme"]}, data["COMPLETE.sha256"], blobs, nil
}
func goupb07Artifacts(d map[string][]byte) goupb07bundle.Artifacts {
	return goupb07bundle.Artifacts{Construction: d["construction-v36.g1"], Execution: d["execution-v36.seme"], ProjectBase: d["project-base-v8.seme"], Inventory: d["inventory-v8.seme"], PackageDetail: d["package-detail-v4.seme"], PackageV4: d["package-v4.seme"], Dependency: d["dependency-v1.seme"], ConfigurationV3: d["configuration-v3.seme"], ProjectV8: d["project-v8.seme"], Resource: d["resource-v1.seme"], ProjectV9: d["project-v9.seme"], Durable: d["durable-state-v1.seme"], Presentation: d["source-presentation-v1.seme"], ProjectV10: d["project-v10.seme"]}
}
func readFile(p string) ([]byte, error) {
	before, e := os.Lstat(p)
	if e != nil || !before.Mode().IsRegular() || before.Size() > 64<<20 {
		return nil, fmt.Errorf("go_upb08_directory.file")
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("go_upb08_directory.changed")
	}
	b, e := io.ReadAll(io.LimitReader(f, 64<<20+1))
	if e != nil || int64(len(b)) != opened.Size() {
		return nil, fmt.Errorf("go_upb08_directory.changed")
	}
	after, e := os.Lstat(p)
	if e != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("go_upb08_directory.changed")
	}
	return b, nil
}
