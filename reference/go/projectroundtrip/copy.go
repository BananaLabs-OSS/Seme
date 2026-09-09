// Package projectroundtrip materializes a verified source snapshot without
// confusing preserved source bytes with executable canonical semantics.
package projectroundtrip

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"seme.local/reference/projectsource"
)

// CopyVerified publishes an exact byte-preserving copy of snapshot at dest.
// The destination must not already exist. Publication is a same-directory
// rename, so verification failures never expose a partial destination.
func CopyVerified(source, dest string, snapshot projectsource.Snapshot, policy projectsource.Policy) (err error) {
	if err := projectsource.Verify(source, snapshot, policy); err != nil {
		return err
	}
	dest, err = filepath.Abs(dest)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(dest); err == nil {
		return fmt.Errorf("project_round_trip.destination_exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	parent := filepath.Dir(dest)
	if info, err := os.Lstat(parent); err != nil || !info.IsDir() {
		return fmt.Errorf("project_round_trip.parent")
	}
	stage, err := os.MkdirTemp(parent, ".seme-project-stage-")
	if err != nil {
		return err
	}
	defer func() {
		if stage != "" {
			_ = os.RemoveAll(stage)
		}
	}()

	for _, unit := range snapshot.Units {
		input := filepath.Join(source, filepath.FromSlash(unit.Path))
		data, err := os.ReadFile(input)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		if int64(len(data)) != unit.Size || hex.EncodeToString(digest[:]) != unit.SHA256 {
			return fmt.Errorf("project_round_trip.digest_drift:%s", unit.Path)
		}
		output := filepath.Join(stage, filepath.FromSlash(unit.Path))
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return err
		}
		file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return err
		}
		written, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil {
			return writeErr
		}
		if written != len(data) {
			return fmt.Errorf("project_round_trip.short_write:%s", unit.Path)
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if err := projectsource.Verify(stage, snapshot, policy); err != nil {
		return err
	}
	if _, err := os.Lstat(dest); err == nil {
		return fmt.Errorf("project_round_trip.destination_exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(stage, dest); err != nil {
		return err
	}
	stage = ""
	return nil
}
