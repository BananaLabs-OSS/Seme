// Package projectbundle captures source bytes in a detached, digest-addressed
// bundle. Bundle bytes are transport material, never executable semantics.
package projectbundle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"seme.local/reference/projectsource"
)

type Blob struct {
	SHA256 string
	Bytes  []byte
}

type Bundle struct {
	Blobs []Blob
}

func Capture(root string, snapshot projectsource.Snapshot, policy projectsource.Policy) (Bundle, error) {
	if err := projectsource.Verify(root, snapshot, policy); err != nil {
		return Bundle{}, err
	}
	byDigest := map[string][]byte{}
	for _, unit := range snapshot.Units {
		data, err := projectsource.ReadVerified(root, unit)
		if err != nil {
			return Bundle{}, err
		}
		if prior, ok := byDigest[unit.SHA256]; ok && !bytes.Equal(prior, data) {
			return Bundle{}, fmt.Errorf("project_bundle.digest_collision:%s", unit.SHA256)
		}
		byDigest[unit.SHA256] = append([]byte(nil), data...)
	}
	digests := make([]string, 0, len(byDigest))
	for digest := range byDigest {
		digests = append(digests, digest)
	}
	sort.Strings(digests)
	out := Bundle{Blobs: make([]Blob, 0, len(digests))}
	for _, digest := range digests {
		out.Blobs = append(out.Blobs, Blob{SHA256: digest, Bytes: byDigest[digest]})
	}
	if err := Validate(snapshot, out); err != nil {
		return Bundle{}, err
	}
	return out, nil
}

func Validate(snapshot projectsource.Snapshot, bundle Bundle) error {
	if err := projectsource.ValidateSnapshot(snapshot); err != nil {
		return err
	}
	required := map[string]bool{}
	for _, unit := range snapshot.Units {
		required[unit.SHA256] = true
	}
	seen := map[string]bool{}
	prior := ""
	for i, blob := range bundle.Blobs {
		if i > 0 && blob.SHA256 <= prior {
			return fmt.Errorf("project_bundle.order")
		}
		prior = blob.SHA256
		decoded, err := hex.DecodeString(blob.SHA256)
		if err != nil || len(decoded) != sha256.Size {
			return fmt.Errorf("project_bundle.digest_shape")
		}
		digest := sha256.Sum256(blob.Bytes)
		if hex.EncodeToString(digest[:]) != blob.SHA256 {
			return fmt.Errorf("project_bundle.digest_mismatch:%s", blob.SHA256)
		}
		if seen[blob.SHA256] {
			return fmt.Errorf("project_bundle.duplicate:%s", blob.SHA256)
		}
		seen[blob.SHA256] = true
		if !required[blob.SHA256] {
			return fmt.Errorf("project_bundle.unreferenced:%s", blob.SHA256)
		}
	}
	for digest := range required {
		if !seen[digest] {
			return fmt.Errorf("project_bundle.missing:%s", digest)
		}
	}
	return nil
}

func (b Bundle) Lookup(digest string) ([]byte, bool) {
	i := sort.Search(len(b.Blobs), func(i int) bool { return b.Blobs[i].SHA256 >= digest })
	if i == len(b.Blobs) || b.Blobs[i].SHA256 != digest {
		return nil, false
	}
	return append([]byte(nil), b.Blobs[i].Bytes...), true
}
