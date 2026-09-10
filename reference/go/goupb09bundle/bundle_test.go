package goupb09bundle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"seme.local/reference/controlledeffectsinstance"
	"strings"
	"testing"
)

func TestLoadRejectsWithoutPartialAuthority(t *testing.T) {
	got, err := Load(context.Background(), Input{})
	if err == nil || len(got.Artifacts.ProjectV12) != 0 || got.Effects.Artifact != nil || got.Project.Composed != nil || got.Blobs != nil {
		t.Fatal("unauthenticated load returned partial authority")
	}
}

const replayDoc = `{"version":"seme.controlled-replay/v1","initial_seed":1,"initial_state":"00","duplicate_policy":"cached-no-new-effects","rejection_policy":"atomic-no-effects","steps":[]}`

func TestManifestCloneAndReplayStrictness(t *testing.T) {
	a := fakeArtifacts()
	blob := []byte("blob")
	sum := sha256.Sum256(blob)
	blobs := map[[32]byte][]byte{sum: blob}
	m, ok := manifest("seme-go-upb09-bundle-v1", files(a), blobs)
	if !ok || !validManifest(m, a, blobs) {
		t.Fatal("valid rejected")
	}
	c := clone(a)
	c.ProjectV12[0] ^= 1
	if validManifest(m, c, blobs) {
		t.Fatal("tamper accepted")
	}
	if a.ProjectV12[0] == c.ProjectV12[0] {
		t.Fatal("artifact clone aliased")
	}
	cb := cloneBlobs(blobs)
	cb[sum][0] ^= 1
	if bytes.Equal(cb[sum], blobs[sum]) {
		t.Fatal("blob clone aliased")
	}
	z := controlledeffectsinstance.Bounds{MaximumSteps: 256, MaximumInitialStateBytes: 524288, MaximumCommandBytes: 4096, MaximumTranscriptBytes: 1048576}
	parsed, e := parseReplay([]byte(replayDoc), z)
	if e != nil {
		t.Fatal(e)
	}
	encoded, e := EncodeReplayAuthority(parsed, z)
	if e != nil || !bytes.Equal(encoded, []byte(replayDoc)) {
		t.Fatalf("noncanonical encoding: %v %q", e, encoded)
	}
	for name, bad := range map[string]string{"unknown": strings.Replace(replayDoc, `"version":`, `"extra":1,"version":`, 1), "version": strings.Replace(replayDoc, replayVersion, "wrong", 1), "hex": strings.Replace(replayDoc, `"00"`, `"0z"`, 1), "policy": strings.Replace(replayDoc, "cached-no-new-effects", "cached", 1), "trailing": replayDoc + `{}`} {
		t.Run(name, func(t *testing.T) {
			if _, e := parseReplay([]byte(bad), z); e == nil {
				t.Fatal("accepted")
			}
		})
	}
	noncanonical := strings.Replace(replayDoc, `,"initial_seed"`, `, "initial_seed"`, 1)
	if _, e := parseReplay([]byte(noncanonical), z); e == nil {
		t.Fatal("noncanonical JSON accepted")
	}
	// A runtime transcript changes static project authority and therefore cannot
	// silently replace the zero-step replay authenticated by COMPLETE.
	dynamic := clone(a)
	dynamic.ReplayAuthority = append(bytes.Clone(dynamic.ReplayAuthority), '\n')
	if validManifest(m, dynamic, blobs) {
		t.Fatal("runtime transcript replaced project authority")
	}
}

func TestDirectoryClosedWorldAndAtomicPublish(t *testing.T) {
	a := fakeArtifacts()
	parent := t.TempDir()
	destination := filepath.Join(parent, "bundle")
	if e := WriteDirectory(destination, a, nil); e != nil {
		t.Fatal(e)
	}
	got, complete, blobs, e := ReadDirectory(destination)
	if e != nil || len(blobs) != 0 || !bytes.Equal(got.ProjectV12, a.ProjectV12) || !validManifest(complete, got, blobs) {
		t.Fatal(e)
	}
	if e = WriteDirectory(destination, a, nil); e == nil {
		t.Fatal("existing destination accepted")
	}
	if e = os.WriteFile(filepath.Join(destination, "extra"), []byte("x"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, _, _, e = ReadDirectory(destination); e == nil {
		t.Fatal("extra accepted")
	}
	t.Run("missing", func(t *testing.T) {
		d := filepath.Join(parent, "missing")
		if e := WriteDirectory(d, a, nil); e != nil {
			t.Fatal(e)
		}
		if e = os.Remove(filepath.Join(d, "project-v12.seme")); e != nil {
			t.Fatal(e)
		}
		if _, _, _, e = ReadDirectory(d); e == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("symlink", func(t *testing.T) {
		d := filepath.Join(parent, "linked")
		if e := os.Symlink(destination, d); e != nil {
			t.Fatal(e)
		}
		if _, _, _, e = ReadDirectory(d); e == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("symlink-file", func(t *testing.T) {
		d := filepath.Join(parent, "file-link")
		if e := WriteDirectory(d, a, nil); e != nil {
			t.Fatal(e)
		}
		if e := os.Remove(filepath.Join(d, "project-v12.seme")); e != nil {
			t.Fatal(e)
		}
		if e := os.Symlink(filepath.Join(destination, "project-v12.seme"), filepath.Join(d, "project-v12.seme")); e != nil {
			t.Fatal(e)
		}
		if _, _, _, e = ReadDirectory(d); e == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("failed-publish-atomic", func(t *testing.T) {
		d := filepath.Join(parent, "failed")
		badKey := sha256.Sum256([]byte("expected"))
		if e := WriteDirectory(d, a, map[[32]byte][]byte{badKey: []byte("wrong")}); e == nil {
			t.Fatal("accepted")
		}
		if _, e := os.Lstat(d); !os.IsNotExist(e) {
			t.Fatal("partial destination")
		}
	})
}

func fakeArtifacts() Artifacts {
	a := Artifacts{}
	for i, f := range files(a) {
		set(&a, f.name, []byte{byte(i + 1)})
	}
	a.ReplayAuthority = []byte(replayDoc)
	return a
}
func set(a *Artifacts, n string, b []byte) {
	switch n {
	case "construction-v36.g1":
		a.Base.Base.Construction = b
	case "execution-v36.seme":
		a.Base.Base.Execution = b
	case "project-base-v8.seme":
		a.Base.Base.ProjectBase = b
	case "inventory-v8.seme":
		a.Base.Base.Inventory = b
	case "package-detail-v4.seme":
		a.Base.Base.PackageDetail = b
	case "package-v4.seme":
		a.Base.Base.PackageV4 = b
	case "dependency-v1.seme":
		a.Base.Base.Dependency = b
	case "configuration-v3.seme":
		a.Base.Base.ConfigurationV3 = b
	case "project-v8.seme":
		a.Base.Base.ProjectV8 = b
	case "resource-v1.seme":
		a.Base.Base.Resource = b
	case "project-v9.seme":
		a.Base.Base.ProjectV9 = b
	case "durable-state-v1.seme":
		a.Base.Base.Durable = b
	case "source-presentation-v1.seme":
		a.Base.Base.Presentation = b
	case "project-v10.seme":
		a.Base.Base.ProjectV10 = b
	case "ordered-transport-v1.seme":
		a.Base.Transport = b
	case "project-v11.seme":
		a.Base.ProjectV11 = b
	case "controlled-effects-v1.seme":
		a.ControlledEffects = b
	case "project-v12.seme":
		a.ProjectV12 = b
	case "controlled-replay-v1.json":
		a.ReplayAuthority = b
	}
}
