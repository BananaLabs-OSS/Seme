package goupb10bundle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type PlacementFiles struct {
	TargetPlan, ProjectV13, Report []byte
}

func WriteDirectory(destination string, files PlacementFiles, baseManifest []byte) error {
	if !absoluteClean(destination) || len(baseManifest) == 0 {
		return fmt.Errorf("go_upb10_publish.path_or_base")
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		return fmt.Errorf("go_upb10_publish.exists")
	}
	parent := filepath.Dir(destination)
	real, err := filepath.EvalSymlinks(parent)
	if err != nil || real != parent {
		return fmt.Errorf("go_upb10_publish.parent")
	}
	values := placementMap(files)
	manifest, err := placementManifest(values, baseManifest)
	if err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(parent, ".seme-upb10-placement-")
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(temporary)
		}
	}()
	for name, value := range values {
		if err = os.WriteFile(filepath.Join(temporary, name), value, 0o600); err != nil {
			return err
		}
	}
	if err = os.WriteFile(filepath.Join(temporary, "COMPLETE.sha256"), manifest, 0o600); err != nil {
		return err
	}
	if err = os.Rename(temporary, destination); err != nil {
		return fmt.Errorf("go_upb10_publish.commit:%w", err)
	}
	keep = true
	return nil
}

func ReadDirectory(root string, baseManifest []byte) (PlacementFiles, error) {
	if !absoluteClean(root) || len(baseManifest) == 0 {
		return PlacementFiles{}, fmt.Errorf("go_upb10_directory.path_or_base")
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return PlacementFiles{}, fmt.Errorf("go_upb10_directory.root")
	}
	real, err := filepath.EvalSymlinks(root)
	if err != nil || real != root {
		return PlacementFiles{}, fmt.Errorf("go_upb10_directory.symlink")
	}
	allowed := map[string]bool{"COMPLETE.sha256": true, "target-plan-v1.seme": true, "project-v13.seme": true, "placement-report.json": true}
	values := map[string][]byte{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, _ := filepath.Rel(root, path)
		relative = filepath.ToSlash(relative)
		if relative == "." {
			return nil
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() || !allowed[relative] {
			return fmt.Errorf("go_upb10_directory.extra_or_nonregular:%s", relative)
		}
		value, readErr := readStable(path)
		if readErr != nil {
			return readErr
		}
		values[relative] = value
		return nil
	})
	if err != nil {
		return PlacementFiles{}, err
	}
	for name := range allowed {
		if len(values[name]) == 0 {
			return PlacementFiles{}, fmt.Errorf("go_upb10_directory.missing:%s", name)
		}
	}
	manifest, err := placementManifest(map[string][]byte{"target-plan-v1.seme": values["target-plan-v1.seme"], "project-v13.seme": values["project-v13.seme"], "placement-report.json": values["placement-report.json"]}, baseManifest)
	if err != nil || !bytes.Equal(manifest, values["COMPLETE.sha256"]) {
		return PlacementFiles{}, fmt.Errorf("go_upb10_directory.manifest")
	}
	return PlacementFiles{TargetPlan: bytes.Clone(values["target-plan-v1.seme"]), ProjectV13: bytes.Clone(values["project-v13.seme"]), Report: bytes.Clone(values["placement-report.json"])}, nil
}

func placementMap(files PlacementFiles) map[string][]byte {
	return map[string][]byte{"target-plan-v1.seme": files.TargetPlan, "project-v13.seme": files.ProjectV13, "placement-report.json": files.Report}
}

func placementManifest(values map[string][]byte, base []byte) ([]byte, error) {
	names := make([]string, 0, len(values))
	for name, value := range values {
		if len(value) == 0 || strings.Contains(name, "/") {
			return nil, fmt.Errorf("go_upb10_publish.artifact")
		}
		names = append(names, name)
	}
	sort.Strings(names)
	result := []byte("seme-go-upb10-placement-v1\n")
	digest := sha256.Sum256(base)
	result = append(result, []byte("base-complete "+hex.EncodeToString(digest[:])+"\n")...)
	for _, name := range names {
		digest = sha256.Sum256(values[name])
		result = append(result, []byte(name+" "+hex.EncodeToString(digest[:])+"\n")...)
	}
	return result, nil
}

func readStable(path string) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > 64<<20 {
		return nil, fmt.Errorf("go_upb10_directory.file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("go_upb10_directory.changed")
	}
	value, err := io.ReadAll(io.LimitReader(file, 64<<20+1))
	if err != nil || int64(len(value)) != opened.Size() {
		return nil, fmt.Errorf("go_upb10_directory.changed")
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("go_upb10_directory.changed")
	}
	return value, nil
}

func absoluteClean(path string) bool { return filepath.IsAbs(path) && filepath.Clean(path) == path }
