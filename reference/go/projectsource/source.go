// Package projectsource discovers a bounded, deterministic project source set.
package projectsource

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

type Class string

const (
	Tracked   Class = "tracked"
	Ignored   Class = "ignored"
	Generated Class = "generated"
	Vendored  Class = "vendored"
	Opaque    Class = "opaque"
)

type Preservation string

const (
	ByteExact          Preservation = "byte-exact"
	SemanticProjection Preservation = "semantic-projection"
)

type Toolchain struct {
	Language, Toolchain, Profile, SemanticRevision string
}

type Policy struct {
	TrackedExtensions, IgnoredPrefixes, VendoredPrefixes []string
	GeneratedHeader                                      []byte
	MaxFiles                                             int
	MaxFileBytes, MaxTotalBytes                          int64
}
type Unit struct {
	Path         string
	Class        Class
	Preservation Preservation
	Size         int64
	SHA256       string
}
type Snapshot struct {
	RootIdentity, ContentRevision string
	Toolchain                     Toolchain
	Units                         []Unit
}

func Discover(root, identity string, toolchain Toolchain, policy Policy) (Snapshot, error) {
	if identity == "" || toolchain.Language == "" || toolchain.Toolchain == "" || toolchain.Profile == "" || toolchain.SemanticRevision == "" {
		return Snapshot{}, fmt.Errorf("project_source.identity")
	}
	if err := validatePolicy(&policy); err != nil {
		return Snapshot{}, err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Snapshot{}, err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return Snapshot{}, err
	}
	if !info.IsDir() {
		return Snapshot{}, fmt.Errorf("project_source.root_not_directory")
	}
	var units []Unit
	var total int64
	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == abs {
			return nil
		}
		relative, e := filepath.Rel(abs, path)
		if e != nil {
			return e
		}
		slash := filepath.ToSlash(relative)
		if e = validPath(slash); e != nil {
			return e
		}
		info, e = d.Info()
		if e != nil {
			return e
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("project_source.symlink:%s", slash)
		}
		if d.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("project_source.nonregular:%s", slash)
		}
		if len(units) >= policy.MaxFiles {
			return fmt.Errorf("project_source.file_limit")
		}
		if info.Size() > policy.MaxFileBytes {
			return fmt.Errorf("project_source.file_size:%s", slash)
		}
		total += info.Size()
		if total > policy.MaxTotalBytes {
			return fmt.Errorf("project_source.total_size")
		}
		data, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		if int64(len(data)) != info.Size() {
			return fmt.Errorf("project_source.digest_drift:%s", slash)
		}
		sum := sha256.Sum256(data)
		class, e := classify(slash, data, policy)
		if e != nil {
			return e
		}
		preservation := ByteExact
		if class == Tracked {
			preservation = SemanticProjection
		}
		units = append(units, Unit{Path: slash, Class: class, Preservation: preservation, Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:])})
		return nil
	})
	if err != nil {
		return Snapshot{}, err
	}
	sort.Slice(units, func(i, j int) bool { return units[i].Path < units[j].Path })
	out := Snapshot{RootIdentity: identity, Toolchain: toolchain, Units: units}
	out.ContentRevision = revision(out)
	return out, nil
}

func Verify(root string, expected Snapshot, policy Policy) error {
	actual, err := Discover(root, expected.RootIdentity, expected.Toolchain, policy)
	if err != nil {
		return err
	}
	if actual.ContentRevision != expected.ContentRevision || len(actual.Units) != len(expected.Units) {
		return fmt.Errorf("project_source.digest_drift")
	}
	for i := range actual.Units {
		if actual.Units[i] != expected.Units[i] {
			return fmt.Errorf("project_source.digest_drift:%s", actual.Units[i].Path)
		}
	}
	return nil
}

func validatePolicy(p *Policy) error {
	if p.MaxFiles <= 0 || p.MaxFileBytes <= 0 || p.MaxTotalBytes <= 0 || p.MaxFileBytes > p.MaxTotalBytes {
		return fmt.Errorf("project_source.bounds")
	}
	if len(p.GeneratedHeader) == 0 {
		return fmt.Errorf("project_source.generated_header")
	}
	ext := map[string]bool{}
	for _, x := range p.TrackedExtensions {
		if x == "" || !strings.HasPrefix(x, ".") || x != strings.ToLower(x) || strings.ContainsAny(x, "/\\") || ext[x] {
			return fmt.Errorf("project_source.extension_policy:%s", x)
		}
		ext[x] = true
	}
	seen := map[string]Class{}
	for _, group := range []struct {
		class    Class
		prefixes []string
	}{{Ignored, p.IgnoredPrefixes}, {Vendored, p.VendoredPrefixes}} {
		for _, x := range group.prefixes {
			if err := validPrefix(x); err != nil {
				return err
			}
			if prior, ok := seen[x]; ok {
				return fmt.Errorf("project_source.prefix_ambiguity:%s:%s:%s", x, prior, group.class)
			}
			seen[x] = group.class
		}
	}
	keys := make([]string, 0, len(seen))
	for x := range seen {
		keys = append(keys, x)
	}
	sort.Strings(keys)
	for i, a := range keys {
		for _, b := range keys[i+1:] {
			if strings.HasPrefix(b, a) {
				return fmt.Errorf("project_source.prefix_overlap:%s:%s", a, b)
			}
		}
	}
	return nil
}
func validPrefix(x string) error {
	if x == "" || strings.Contains(x, "\\") || strings.HasPrefix(x, "/") || !strings.HasSuffix(x, "/") {
		return fmt.Errorf("project_source.prefix:%s", x)
	}
	return validPath(strings.TrimSuffix(x, "/"))
}
func validPath(x string) error {
	if x == "" || !utf8.ValidString(x) || strings.Contains(x, "\\") || strings.HasPrefix(x, "/") || filepath.ToSlash(filepath.Clean(filepath.FromSlash(x))) != x || x == ".." || strings.HasPrefix(x, "../") {
		return fmt.Errorf("project_source.path:%s", x)
	}
	return nil
}
func classify(path string, data []byte, p Policy) (Class, error) {
	ignored, vendored := false, false
	for _, x := range p.IgnoredPrefixes {
		ignored = ignored || strings.HasPrefix(path, x)
	}
	for _, x := range p.VendoredPrefixes {
		vendored = vendored || strings.HasPrefix(path, x)
	}
	if ignored && vendored {
		return "", fmt.Errorf("project_source.classification_ambiguity:%s", path)
	}
	if ignored {
		return Ignored, nil
	}
	if vendored {
		return Vendored, nil
	}
	if bytes.HasPrefix(data, p.GeneratedHeader) {
		return Generated, nil
	}
	extension := strings.ToLower(filepath.Ext(path))
	for _, x := range p.TrackedExtensions {
		if extension == x {
			return Tracked, nil
		}
	}
	return Opaque, nil
}
func revision(s Snapshot) string {
	h := sha256.New()
	h.Write([]byte("seme-project-source-v1\x00"))
	field := func(x string) { h.Write([]byte(fmt.Sprintf("%d:", len(x)))); h.Write([]byte(x)) }
	field(s.RootIdentity)
	field(s.Toolchain.Language)
	field(s.Toolchain.Toolchain)
	field(s.Toolchain.Profile)
	field(s.Toolchain.SemanticRevision)
	for _, u := range s.Units {
		field(u.Path)
		field(string(u.Class))
		field(string(u.Preservation))
		field(fmt.Sprint(u.Size))
		field(u.SHA256)
	}
	return hex.EncodeToString(h.Sum(nil))
}
