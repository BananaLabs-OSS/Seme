package goprovider

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ReadDocumentSnapshot reads one bounded, ordinary multi-package Go project
// into the same complete in-memory form used by live language sessions.
func ReadDocumentSnapshot(root, module, packagePath, entry string, revision uint64) (DocumentSnapshot, error) {
	if revision == 0 || module == "" || packagePath == "" || entry == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return DocumentSnapshot{}, errors.New("provider.snapshot_options")
	}
	real, err := filepath.EvalSymlinks(root)
	if err != nil || real != root {
		return DocumentSnapshot{}, errors.New("provider.snapshot_root")
	}
	files := map[string]string{}
	var total int64
	err = filepath.WalkDir(root, func(path string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if item.IsDir() {
			if item.Type()&os.ModeSymlink != 0 {
				return errors.New("provider.snapshot_nonregular")
			}
			if item.Name() == ".git" || item.Name() == "vendor" || item.Name() == reconciliationDirectory {
				return filepath.SkipDir
			}
			return nil
		}
		if item.Type()&os.ModeSymlink != 0 {
			return errors.New("provider.snapshot_nonregular")
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		info, infoErr := item.Info()
		if infoErr != nil || !info.Mode().IsRegular() || info.Size() > 2<<20 || len(files) >= 1024 {
			return errors.New("provider.snapshot_bounds")
		}
		total += info.Size()
		if total > 32<<20 {
			return errors.New("provider.snapshot_bounds")
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil || int64(len(data)) != info.Size() {
			return errors.New("provider.snapshot_changed")
		}
		relative, relativeErr := filepath.Rel(root, path)
		if relativeErr != nil {
			return relativeErr
		}
		files[filepath.ToSlash(relative)] = string(data)
		return nil
	})
	if err != nil || len(files) == 0 {
		return DocumentSnapshot{}, errors.New("provider.snapshot_source")
	}
	return DocumentSnapshot{Revision: revision, ModulePath: module, PackagePath: packagePath, Entry: entry, Files: files}, nil
}
