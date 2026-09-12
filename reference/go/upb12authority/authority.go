// Package upb12authority publishes the closed, source-free canonical seed used
// by all native UPB12 projectors. It contains no native source or language
// preference.
package upb12authority

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"seme.local/reference/goupb10cmdload"
)

const Header = "seme-upb12-source-free-authority-v1"

var Required = []string{
	"construction-v36.g1", "execution-v36.seme", "package-detail-v4.seme", "package-v4.seme",
	"dependency-v1.seme", "configuration-v3.seme", "resource-v1.seme", "durable-state-v1.seme",
	"source-presentation-v1.seme", "ordered-transport-v1.seme", "controlled-effects-v1.seme",
	"project-v12.seme", "target-plan-v1.seme", "project-v13.seme", "controlled-replay-v1.json",
	"configuration-selection-v1.json", "durable-selection-v1.json", "transport-selection-v1.json", "effects-selection-v1.json",
}

type Selections struct{ Configuration, Durable, Transport, Effects []byte }

func Files(result goupb10cmdload.Result, selections Selections) map[string][]byte {
	a := result.Bundle.Artifacts
	b := a.Base.Base.Base
	files := map[string][]byte{
		"construction-v36.g1": b.Construction, "execution-v36.seme": b.Execution,
		"package-detail-v4.seme": b.PackageDetail, "package-v4.seme": b.PackageV4,
		"dependency-v1.seme": b.Dependency, "configuration-v3.seme": b.ConfigurationV3,
		"resource-v1.seme": b.Resource, "durable-state-v1.seme": b.Durable,
		"source-presentation-v1.seme": b.Presentation,
		"ordered-transport-v1.seme":   a.Base.Base.Transport,
		"controlled-effects-v1.seme":  a.Base.ControlledEffects,
		"project-v12.seme":            a.Base.ProjectV12, "controlled-replay-v1.json": a.Base.ReplayAuthority,
		"target-plan-v1.seme": a.TargetPlan, "project-v13.seme": a.ProjectV13,
		"configuration-selection-v1.json": selections.Configuration,
		"durable-selection-v1.json":       selections.Durable,
		"transport-selection-v1.json":     selections.Transport,
		"effects-selection-v1.json":       selections.Effects,
	}
	for digest, value := range result.Bundle.Base.Blobs {
		files["blobs/"+hex.EncodeToString(digest[:])] = value
	}
	return clone(files)
}

func Publish(destination string, files map[string][]byte) error {
	if !filepath.IsAbs(destination) || filepath.Clean(destination) != destination {
		return fmt.Errorf("upb12_authority.output")
	}
	parent := filepath.Dir(destination)
	real, err := filepath.EvalSymlinks(parent)
	if err != nil || real != parent {
		return fmt.Errorf("upb12_authority.parent")
	}
	if _, err = os.Lstat(destination); !os.IsNotExist(err) {
		return fmt.Errorf("upb12_authority.exists")
	}
	manifest, err := Manifest(files)
	if err != nil {
		return err
	}
	stage, err := os.MkdirTemp(parent, ".seme-upb12-authority-")
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(stage)
		}
	}()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		target := filepath.Join(stage, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err = os.WriteFile(target, files[name], 0600); err != nil {
			return err
		}
	}
	if err = os.WriteFile(filepath.Join(stage, "COMPLETE.sha256"), manifest, 0600); err != nil {
		return err
	}
	if err = os.Rename(stage, destination); err != nil {
		return err
	}
	keep = true
	return nil
}

func Manifest(files map[string][]byte) ([]byte, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("upb12_authority.empty")
	}
	names := make([]string, 0, len(files))
	for name, value := range files {
		if !validName(name) || len(value) == 0 || len(value) > 64<<20 {
			return nil, fmt.Errorf("upb12_authority.artifact:%s", name)
		}
		names = append(names, name)
	}
	for _, name := range Required {
		if len(files[name]) == 0 {
			return nil, fmt.Errorf("upb12_authority.required:%s", name)
		}
	}
	hasBlob := false
	for name := range files {
		hasBlob = hasBlob || strings.HasPrefix(name, "blobs/")
	}
	if !hasBlob {
		return nil, fmt.Errorf("upb12_authority.resource_blob")
	}
	sort.Strings(names)
	out := []byte(Header + "\n")
	for _, name := range names {
		sum := sha256.Sum256(files[name])
		out = append(out, []byte(name+" "+hex.EncodeToString(sum[:])+"\n")...)
	}
	return out, nil
}
func validName(name string) bool {
	if name == "" || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || strings.Contains(name, "..") || strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".lua") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(name)))
	return clean == name
}
func clone(in map[string][]byte) map[string][]byte {
	out := make(map[string][]byte, len(in))
	for name, value := range in {
		out[name] = append([]byte(nil), value...)
	}
	return out
}
