package godependencyresolver

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"path"
	"regexp"
	"seme.local/reference/dependencyresolution"
	"sort"
	"strings"
)

type Local struct {
	From, To, Source, Digest string
	Files                    map[string][]byte
}
type External struct {
	Module, Version, Source, Digest string
	Files                           map[string][]byte
}
type Input struct {
	GoMod, GoSum   []byte
	Local          Local
	External       External
	PackageImports map[string][]string
}

func Resolve(in Input) (dependencyresolution.Closure, error) {
	module, rm, rv, err := parseMod(in.GoMod)
	if err != nil {
		return dependencyresolution.Closure{}, err
	}
	if !strings.HasPrefix(in.Local.From, module+"/") || !strings.HasPrefix(in.Local.To, module+"/") || in.Local.From == in.Local.To || !contains(in.PackageImports[in.Local.From], in.Local.To) {
		return dependencyresolution.Closure{}, fmt.Errorf("go_dependency.local")
	}
	ld, lh, err := tree(in.Local.Files)
	if err != nil || ld != in.Local.Digest || in.Local.Source == "" {
		return dependencyresolution.Closure{}, fmt.Errorf("go_dependency.local_integrity")
	}
	if rm != in.External.Module || rv != in.External.Version {
		return dependencyresolution.Closure{}, fmt.Errorf("go_dependency.undeclared")
	}
	if !pinned(rv) || in.External.Source == "" {
		return dependencyresolution.Closure{}, fmt.Errorf("go_dependency.external")
	}
	ed, eh, err := tree(in.External.Files)
	if err != nil || ed != in.External.Digest {
		return dependencyresolution.Closure{}, fmt.Errorf("go_dependency.external_integrity")
	}
	sum, err := findSum(in.GoSum, rm, rv)
	if err != nil {
		return dependencyresolution.Closure{}, err
	}
	if sum != eh {
		return dependencyresolution.Closure{}, fmt.Errorf("go_dependency.h1_mismatch")
	}
	localVersion := "source:" + ld
	out := dependencyresolution.Closure{Requirements: []dependencyresolution.Requirement{{Identity: in.Local.To, Requirement: localVersion, Kind: dependencyresolution.Local, Metadata: []dependencyresolution.Metadata{{Key: "go.import.from", Value: in.Local.From}}}, {Identity: rm, Requirement: rv, Kind: dependencyresolution.External, Metadata: []dependencyresolution.Metadata{{Key: "go.mod.direct", Value: "true"}}}}, Entries: []dependencyresolution.Entry{
		{Identity: in.Local.To, Version: localVersion, Kind: dependencyresolution.Local, Integrity: lh, IntegrityAlgorithm: "go-h1-tree", Source: in.Local.Source, SourceKind: "explicit-local-tree", Digest: ld, Metadata: []dependencyresolution.Metadata{{Key: "direct", Value: "true"}, {Key: "go.import.from", Value: in.Local.From}}},
		{Identity: rm, Ecosystem: "go", Version: rv, Kind: dependencyresolution.External, Integrity: eh, IntegrityAlgorithm: "go-h1-tree", Source: in.External.Source, SourceKind: "explicit-offline-module-tree", Digest: ed, Metadata: []dependencyresolution.Metadata{{Key: "direct", Value: "true"}, {Key: "go.mod.require", Value: rm + "@" + rv}, {Key: "go.sum.integrity", Value: eh}}},
	}}
	dependencyresolution.Normalize(&out)
	if err = dependencyresolution.Validate(out); err != nil {
		return dependencyresolution.Closure{}, err
	}
	return out, nil
}
func parseMod(b []byte) (string, string, string, error) {
	s := bufio.NewScanner(strings.NewReader(string(b)))
	module := ""
	var req [][]string
	for s.Scan() {
		line := strings.TrimSpace(strings.SplitN(s.Text(), "//", 2)[0])
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if f[0] == "replace" || strings.Contains(line, "=>") {
			return "", "", "", fmt.Errorf("go_dependency.replace")
		}
		if f[0] == "module" && len(f) == 2 {
			module = f[1]
		}
		if f[0] == "require" && len(f) == 3 {
			req = append(req, f[1:])
		} else if f[0] == "require" {
			return "", "", "", fmt.Errorf("go_dependency.require_shape")
		}
	}
	if s.Err() != nil {
		return "", "", "", s.Err()
	}
	if module == "" || len(req) != 1 {
		return "", "", "", fmt.Errorf("go_dependency.mod_shape")
	}
	return module, req[0][0], req[0][1], nil
}
func findSum(b []byte, m, v string) (string, error) {
	found := ""
	s := bufio.NewScanner(strings.NewReader(string(b)))
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) == 3 && f[0] == m && f[1] == v {
			if found != "" || !strings.HasPrefix(f[2], "h1:") {
				return "", fmt.Errorf("go_dependency.sum_duplicate")
			}
			found = f[2]
		}
	}
	if found == "" {
		return "", fmt.Errorf("go_dependency.sum_missing")
	}
	return found, nil
}
func tree(files map[string][]byte) (string, string, error) {
	if len(files) == 0 {
		return "", "", fmt.Errorf("empty")
	}
	names := make([]string, 0, len(files))
	for n := range files {
		if !validTreePath(n) {
			return "", "", fmt.Errorf("path")
		}
		names = append(names, n)
	}
	sort.Strings(names)
	raw := sha256.New()
	lines := sha256.New()
	for _, n := range names {
		h := sha256.Sum256(files[n])
		fmt.Fprintf(raw, "%s\x00%d\x00", n, len(files[n]))
		raw.Write(files[n])
		fmt.Fprintf(lines, "%x  %s\n", h, n)
	}
	return hex.EncodeToString(raw.Sum(nil)), "h1:" + base64.StdEncoding.EncodeToString(lines.Sum(nil)), nil
}

// AnalyzeTree exposes the resolver's exact bounded content/integrity
// calculation so filesystem loaders cannot silently drift from it.
func AnalyzeTree(files map[string][]byte) (digest, integrity string, err error) {
	return tree(files)
}

var stableVersion = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func validTreePath(n string) bool {
	if n == "" || n == "." || n == ".." || strings.HasPrefix(n, "/") || strings.HasSuffix(n, "/") || strings.Contains(n, "\\") || strings.Contains(n, "//") || path.Clean(n) != n {
		return false
	}
	for _, r := range n {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}
func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
func pinned(v string) bool { return stableVersion.MatchString(v) }
