// Package goresourcemanifest parses the bounded resource selection boundary.
package goresourcemanifest

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"
)

const MaxBytes = 1 << 20
const Version = "seme.resources/v1"

type Selection struct {
	Identity, Path, Destination, MediaType string
	Size                                   uint64
	SHA256                                 [32]byte
}
type document struct {
	Version   string `json:"version"`
	Resources []item `json:"resources"`
}
type item struct {
	Identity    string `json:"identity"`
	Path        string `json:"path"`
	Destination string `json:"destination"`
	MediaType   string `json:"media_type"`
	Size        uint64 `json:"size"`
	SHA256      string `json:"sha256"`
}

// Encode emits the sole deterministic JSON representation used when a
// canonical resource plan is projected back into an ordinary project.
func Encode(resources []Selection) ([]byte, error) {
	if len(resources) == 0 || len(resources) > 1024 {
		return nil, fmt.Errorf("resource_manifest.count")
	}
	x := append([]Selection(nil), resources...)
	sort.Slice(x, func(i, j int) bool { return x[i].Identity < x[j].Identity })
	doc := document{Version: Version, Resources: make([]item, len(x))}
	for i, r := range x {
		doc.Resources[i] = item{Identity: r.Identity, Path: r.Path, Destination: r.Destination, MediaType: r.MediaType, Size: r.Size, SHA256: hex.EncodeToString(r.SHA256[:])}
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	parsed, err := Parse(b)
	if err != nil || len(parsed) != len(x) {
		return nil, fmt.Errorf("resource_manifest.encode:%w", err)
	}
	for i := range x {
		if parsed[i] != x[i] {
			return nil, fmt.Errorf("resource_manifest.encode_roundtrip")
		}
	}
	return b, nil
}

func Parse(data []byte) ([]Selection, error) {
	if len(data) == 0 || len(data) > MaxBytes {
		return nil, fmt.Errorf("resource_manifest.size")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var x document
	if err := d.Decode(&x); err != nil {
		return nil, fmt.Errorf("resource_manifest.json:%w", err)
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("resource_manifest.trailing")
	}
	if x.Version != Version {
		return nil, fmt.Errorf("resource_manifest.version")
	}
	if len(x.Resources) == 0 || len(x.Resources) > 1024 {
		return nil, fmt.Errorf("resource_manifest.count")
	}
	out := make([]Selection, len(x.Resources))
	for i, r := range x.Resources {
		if !identity(r.Identity) || !path(r.Path) || !path(r.Destination) || !media(r.MediaType) || r.Size > 16<<20 || len(r.SHA256) != 64 || r.SHA256 != strings.ToLower(r.SHA256) {
			return nil, fmt.Errorf("resource_manifest.item:%d", i)
		}
		b, err := hex.DecodeString(r.SHA256)
		if err != nil {
			return nil, fmt.Errorf("resource_manifest.digest:%d", i)
		}
		var sum [32]byte
		copy(sum[:], b)
		out[i] = Selection{r.Identity, r.Path, r.Destination, r.MediaType, r.Size, sum}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Identity < out[j].Identity })
	paths, dest := map[string]bool{}, map[string]bool{}
	for i := range out {
		if (i > 0 && out[i].Identity == out[i-1].Identity) || paths[out[i].Path] || dest[out[i].Destination] {
			return nil, fmt.Errorf("resource_manifest.duplicate")
		}
		paths[out[i].Path], dest[out[i].Destination] = true, true
	}
	return out, nil
}
func identity(s string) bool {
	if len(s) == 0 || len(s) > 128 || !utf8.ValidString(s) {
		return false
	}
	for _, p := range strings.Split(s, "/") {
		if p == "" {
			return false
		}
		for i, c := range []byte(p) {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (i > 0 && (c == '.' || c == '_' || c == '-'))) {
				return false
			}
		}
	}
	return true
}
func path(s string) bool {
	return len(s) > 0 && len(s) <= 512 && utf8.ValidString(s) && !strings.Contains(s, "\\") && !strings.HasPrefix(s, "/") && !strings.Contains(s, "//") && s != "." && s != ".." && !strings.HasPrefix(s, "../") && !strings.Contains(s, "/../") && !strings.HasSuffix(s, "/..")
}
func media(s string) bool {
	if len(s) < 3 || len(s) > 128 || s != strings.ToLower(s) || strings.Count(s, "/") != 1 {
		return false
	}
	for _, c := range []byte(s) {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '/' || c == '.' || c == '+' || c == '-' || c == ';' || c == '=' || c == ' ') {
			return false
		}
	}
	a, b, _ := strings.Cut(s, "/")
	return a != "" && b != ""
}
