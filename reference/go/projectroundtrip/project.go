package projectroundtrip

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"seme.local/reference/projectbundle"
	"seme.local/reference/projectsource"
)

// PublishProjected combines canonically regenerated tracked files with
// byte-exact preserved regions from a detached bundle. It never reads the
// original project root. The projected map must contain exactly the tracked
// paths and no preserved path.
func PublishProjected(dest string, original projectsource.Snapshot, bundle projectbundle.Bundle, projected map[string][]byte, policy projectsource.Policy) (result projectsource.Snapshot, err error) {
	if err := projectsource.ValidateSnapshot(original); err != nil {
		return result, err
	}
	if err := projectbundle.Validate(original, bundle); err != nil {
		return result, err
	}
	tracked := map[string]bool{}
	for _, unit := range original.Units {
		if unit.Class == projectsource.Tracked {
			tracked[unit.Path] = true
		}
	}
	if len(projected) != len(tracked) {
		return result, fmt.Errorf("project_round_trip.projected_set")
	}
	for path := range projected {
		if !tracked[path] {
			return result, fmt.Errorf("project_round_trip.projected_path:%s", path)
		}
	}

	dest, err = filepath.Abs(dest)
	if err != nil {
		return result, err
	}
	if _, err := os.Lstat(dest); err == nil {
		return result, fmt.Errorf("project_round_trip.destination_exists")
	} else if !os.IsNotExist(err) {
		return result, err
	}
	parent := filepath.Dir(dest)
	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil || resolvedParent != parent {
		return result, fmt.Errorf("project_round_trip.parent")
	}
	stage, err := os.MkdirTemp(parent, ".seme-project-stage-")
	if err != nil {
		return result, err
	}
	defer func() {
		if stage != "" {
			_ = os.RemoveAll(stage)
		}
	}()

	for _, unit := range original.Units {
		data := projected[unit.Path]
		if unit.Class != projectsource.Tracked {
			var ok bool
			data, ok = bundle.Lookup(unit.SHA256)
			if !ok {
				return result, fmt.Errorf("project_round_trip.bundle_missing:%s", unit.Path)
			}
			digest := sha256.Sum256(data)
			if int64(len(data)) != unit.Size || hex.EncodeToString(digest[:]) != unit.SHA256 {
				return result, fmt.Errorf("project_round_trip.preservation:%s", unit.Path)
			}
		}
		path := filepath.Join(stage, filepath.FromSlash(unit.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return result, err
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return result, err
		}
		written, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil || written != len(data) {
			return result, fmt.Errorf("project_round_trip.write:%s", unit.Path)
		}
		if closeErr != nil {
			return result, closeErr
		}
	}

	result, err = projectsource.Discover(stage, original.RootIdentity, original.Toolchain, policy)
	if err != nil {
		return projectsource.Snapshot{}, err
	}
	if len(result.Units) != len(original.Units) {
		return projectsource.Snapshot{}, fmt.Errorf("project_round_trip.unit_count")
	}
	for i, before := range original.Units {
		after := result.Units[i]
		if before.Path != after.Path || before.Class != after.Class || before.Preservation != after.Preservation {
			return projectsource.Snapshot{}, fmt.Errorf("project_round_trip.classification:%s", before.Path)
		}
		if before.Class != projectsource.Tracked && before != after {
			return projectsource.Snapshot{}, fmt.Errorf("project_round_trip.preservation:%s", before.Path)
		}
	}
	if _, err := os.Lstat(dest); err == nil {
		return projectsource.Snapshot{}, fmt.Errorf("project_round_trip.destination_exists")
	} else if !os.IsNotExist(err) {
		return projectsource.Snapshot{}, err
	}
	if err := os.Rename(stage, dest); err != nil {
		return projectsource.Snapshot{}, err
	}
	stage = ""
	return result, nil
}
