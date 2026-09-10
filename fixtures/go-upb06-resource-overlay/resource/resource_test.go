package resource

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

type declaration struct {
	Version   string          `json:"version"`
	Resources []resourceEntry `json:"resources"`
}

type resourceEntry struct {
	Identity    string `json:"identity"`
	Path        string `json:"path"`
	Destination string `json:"destination"`
	MediaType   string `json:"media_type"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
}

func TestValidate(t *testing.T) {
	valid := Set{Notice: "Seme resources — 世界\n", Marker: []byte{0x00, 0xff, 'S', 'E', 'M', 'E', '\n'}}
	if got := Validate(valid); !got.Ok || !got.Value || got.Error != 0 {
		t.Fatalf("valid resources: %+v", got)
	}
	for _, test := range []struct {
		name string
		set  Set
		err  int64
	}{
		{"empty notice", Set{Marker: valid.Marker}, 40},
		{"notice without newline", Set{Notice: strings.TrimSuffix(valid.Notice, "\n"), Marker: valid.Marker}, 40},
		{"nil marker", Set{Notice: valid.Notice}, 41},
		{"changed marker", Set{Notice: valid.Notice, Marker: []byte{0x00, 0xfe, 'S', 'E', 'M', 'E', '\n'}}, 41},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Validate(test.set)
			if got.Ok || got.Error != test.err {
				t.Fatalf("got %+v, want error %d", got, test.err)
			}
		})
	}
}

func TestDeclaredResourcesMatchFiles(t *testing.T) {
	root := filepath.Clean("..")
	manifest, err := os.Open(filepath.Join(root, "resources.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer manifest.Close()
	decoder := json.NewDecoder(manifest)
	decoder.DisallowUnknownFields()
	var got declaration
	if err := decoder.Decode(&got); err != nil {
		t.Fatal(err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("trailing JSON value: %v", err)
	}
	want := []resourceEntry{
		{"example.test/go-uab-11/resource.notice.v1", "resources/notice.txt", "resources/notice.txt", "text/plain; charset=utf-8", 26, "dde7c6f27c3a259a48b5c9e6b886a7f3c1c75f87c73699536eccb7149d1e7d48"},
		{"example.test/go-uab-11/resource.marker.v1", "resources/marker.bin", "resources/marker.bin", "application/octet-stream", 7, "d57b9b19e900f27112610f80812bf55d383d69ffd0e5796b7e1c84164ef15f9d"},
	}
	if got.Version != "seme.resources/v1" || !reflect.DeepEqual(got.Resources, want) {
		t.Fatalf("unexpected declaration: %+v", got)
	}
	for _, entry := range got.Resources {
		if filepath.IsAbs(entry.Path) || filepath.Clean(entry.Path) != entry.Path || strings.HasPrefix(entry.Path, "../") {
			t.Fatalf("unsafe path %q", entry.Path)
		}
		info, err := os.Lstat(filepath.Join(root, entry.Path))
		if err != nil {
			t.Fatal(err)
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("%s is not a regular file", entry.Path)
		}
		data, err := os.ReadFile(filepath.Join(root, entry.Path))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		if int64(len(data)) != entry.Size || hex.EncodeToString(digest[:]) != entry.SHA256 {
			t.Fatalf("content mismatch for %s", entry.Path)
		}
		if entry.MediaType == "text/plain; charset=utf-8" && !utf8.Valid(data) {
			t.Fatalf("invalid UTF-8: %s", entry.Path)
		}
	}
}
