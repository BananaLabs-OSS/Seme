// Package projectsource discovers a bounded, deterministic project source set.
package projectsource

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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
	TrackedExtensions, IgnoredPrefixes, IgnoredSuffixes, VendoredPrefixes []string
	GeneratedHeader                                                       []byte
	MaxFiles                                                              int
	MaxFileBytes, MaxTotalBytes                                           int64
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

// ValidateSnapshot checks a detached inventory without reading source bytes.
func ValidateSnapshot(s Snapshot) error {
	if s.RootIdentity == "" || s.Toolchain.Language == "" || s.Toolchain.Toolchain == "" || s.Toolchain.Profile == "" || s.Toolchain.SemanticRevision == "" {
		return fmt.Errorf("project_source.identity")
	}
	if len(s.Units) == 0 {
		return fmt.Errorf("project_source.empty")
	}
	if len(s.ContentRevision) != 64 {
		return fmt.Errorf("project_source.revision_shape")
	}
	if _, err := hex.DecodeString(s.ContentRevision); err != nil {
		return fmt.Errorf("project_source.revision_shape")
	}
	prior := ""
	for i, u := range s.Units {
		if err := validPath(u.Path); err != nil {
			return err
		}
		if i > 0 && u.Path <= prior {
			return fmt.Errorf("project_source.unit_order:%s", u.Path)
		}
		prior = u.Path
		if u.Size < 0 {
			return fmt.Errorf("project_source.unit_size:%s", u.Path)
		}
		digest, err := hex.DecodeString(u.SHA256)
		if err != nil || len(digest) != sha256.Size {
			return fmt.Errorf("project_source.unit_digest:%s", u.Path)
		}
		switch u.Class {
		case Tracked, Ignored, Generated, Vendored, Opaque:
		default:
			return fmt.Errorf("project_source.unit_class:%s", u.Path)
		}
		switch u.Preservation {
		case ByteExact, SemanticProjection:
		default:
			return fmt.Errorf("project_source.unit_preservation:%s", u.Path)
		}
		if (u.Class == Tracked) != (u.Preservation == SemanticProjection) {
			return fmt.Errorf("project_source.unit_preservation_mapping:%s", u.Path)
		}
	}
	if revision(s) != s.ContentRevision {
		return fmt.Errorf("project_source.revision_mismatch")
	}
	return nil
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
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil || resolved != abs {
		return Snapshot{}, fmt.Errorf("project_source.root_symlink_component")
	}
	rootInfo, err := os.Lstat(abs)
	if err != nil || rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return Snapshot{}, fmt.Errorf("project_source.root_not_plain_directory")
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
		info, e := d.Info()
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
		data, e := readRegular(abs, path, policy.MaxFileBytes)
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
	if err := ValidateSnapshot(out); err != nil {
		return Snapshot{}, err
	}
	return out, nil
}

func readRegular(root, path string, limit int64) ([]byte, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path || !strings.HasPrefix(path, root+string(filepath.Separator)) {
		return nil, fmt.Errorf("project_source.symlink_component:%s", path)
	}
	before, err := os.Lstat(path)
	if err != nil || before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("project_source.not_plain_file:%s", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("project_source.changed_file:%s", path)
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, fmt.Errorf("project_source.read_file:%s", path)
	}
	after, err := file.Stat()
	pathAfter, pathErr := os.Lstat(path)
	resolvedAfter, resolveErr := filepath.EvalSymlinks(path)
	if err != nil || pathErr != nil || resolveErr != nil || resolvedAfter != path || pathAfter.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, after) || !os.SameFile(before, pathAfter) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("project_source.changed_file:%s", path)
	}
	return data, nil
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

// ReadVerified returns one inventoried file only when the root is plain and
// the currently opened bytes still match the unit's declared size and digest.
func ReadVerified(root string, unit Unit) ([]byte, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil || resolved != abs {
		return nil, fmt.Errorf("project_source.root_symlink_component")
	}
	if err := validPath(unit.Path); err != nil {
		return nil, err
	}
	data, err := readRegular(abs, filepath.Join(abs, filepath.FromSlash(unit.Path)), unit.Size)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	if int64(len(data)) != unit.Size || hex.EncodeToString(digest[:]) != unit.SHA256 {
		return nil, fmt.Errorf("project_source.digest_drift:%s", unit.Path)
	}
	return data, nil
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
	suffix := map[string]bool{}
	for _, x := range p.IgnoredSuffixes {
		if x == "" || strings.ContainsAny(x, "/\\") || suffix[x] {
			return fmt.Errorf("project_source.suffix_policy:%s", x)
		}
		suffix[x] = true
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
	for _, x := range p.IgnoredSuffixes {
		ignored = ignored || strings.HasSuffix(path, x)
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
