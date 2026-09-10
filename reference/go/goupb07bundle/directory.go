package goupb07bundle

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ReadDirectory accepts exactly the UPB07 artifact set, COMPLETE, blobs
// directory, and digest-named regular blob files. It follows no symlinks.
func ReadDirectory(root string) (Artifacts, []byte, map[[32]byte][]byte, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb07_directory.path")
	}
	i, err := os.Lstat(root)
	if err != nil || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb07_directory.root")
	}
	if r, e := filepath.EvalSymlinks(root); e != nil || r != root {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb07_directory.symlink")
	}
	allowed := map[string]bool{"COMPLETE.sha256": true}
	for _, f := range files(Artifacts{}) {
		allowed[f.name] = true
	}
	data := map[string][]byte{}
	blobs := map[[32]byte][]byte{}
	sawBlobs := false
	err = filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
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
			return fmt.Errorf("go_upb07_directory.extra:%s", rel)
		}
		if d.Type()&os.ModeSymlink != 0 || !d.Type().IsRegular() {
			return fmt.Errorf("go_upb07_directory.nonregular")
		}
		b, e := readFile(p)
		if e != nil {
			return e
		}
		if strings.HasPrefix(rel, "blobs/") {
			name := strings.TrimPrefix(rel, "blobs/")
			raw, e := hex.DecodeString(name)
			if e != nil || len(raw) != 32 {
				return fmt.Errorf("go_upb07_directory.blob_name")
			}
			var sum [32]byte
			copy(sum[:], raw)
			if _, ok := blobs[sum]; ok {
				return fmt.Errorf("go_upb07_directory.blob_duplicate")
			}
			blobs[sum] = b
			return nil
		}
		if !allowed[rel] {
			return fmt.Errorf("go_upb07_directory.extra:%s", rel)
		}
		data[rel] = b
		return nil
	})
	if err != nil {
		return Artifacts{}, nil, nil, err
	}
	if !sawBlobs {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb07_directory.blobs")
	}
	for _, f := range files(Artifacts{}) {
		if data[f.name] == nil {
			return Artifacts{}, nil, nil, fmt.Errorf("go_upb07_directory.missing:%s", f.name)
		}
	}
	if data["COMPLETE.sha256"] == nil {
		return Artifacts{}, nil, nil, fmt.Errorf("go_upb07_directory.complete")
	}
	a := Artifacts{data["construction-v36.g1"], data["execution-v36.seme"], data["project-base-v8.seme"], data["inventory-v8.seme"], data["package-detail-v4.seme"], data["package-v4.seme"], data["dependency-v1.seme"], data["configuration-v3.seme"], data["project-v8.seme"], data["resource-v1.seme"], data["project-v9.seme"], data["durable-state-v1.seme"], data["source-presentation-v1.seme"], data["project-v10.seme"]}
	return a, data["COMPLETE.sha256"], blobs, nil
}
func readFile(p string) ([]byte, error) {
	before, e := os.Lstat(p)
	if e != nil || !before.Mode().IsRegular() || before.Size() > 64<<20 {
		return nil, fmt.Errorf("go_upb07_directory.file")
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("go_upb07_directory.changed")
	}
	b, e := io.ReadAll(io.LimitReader(f, 64<<20+1))
	if e != nil || int64(len(b)) != opened.Size() {
		return nil, fmt.Errorf("go_upb07_directory.changed")
	}
	after, e := os.Lstat(p)
	if e != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("go_upb07_directory.changed")
	}
	return b, nil
}
