// Command project-source-discover publishes a deterministic, language-neutral
// source snapshot. Ecosystem adapters supply policy; this command does not infer
// language semantics from filenames or host state.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"seme.local/reference/projectsource"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var root, identity, language, toolchain, profile, semanticRevision, tracked, ignoredPrefixes, ignoredSuffixes, vendoredPrefixes, generatedHeader, out string
	var maxFiles int
	var maxFileBytes, maxTotalBytes int64
	flag.StringVar(&root, "root", "", "plain project root directory")
	flag.StringVar(&identity, "identity", "", "stable project root identity")
	flag.StringVar(&language, "language", "", "language adapter identity")
	flag.StringVar(&toolchain, "toolchain", "", "pinned native toolchain identity")
	flag.StringVar(&profile, "profile", "", "source classification profile")
	flag.StringVar(&semanticRevision, "semantic-revision", "", "semantic provider revision")
	flag.StringVar(&tracked, "tracked-extensions", "", "comma-separated semantic source extensions")
	flag.StringVar(&ignoredPrefixes, "ignored-prefixes", "", "comma-separated ignored path prefixes")
	flag.StringVar(&ignoredSuffixes, "ignored-suffixes", "", "comma-separated ignored path suffixes")
	flag.StringVar(&vendoredPrefixes, "vendored-prefixes", "", "comma-separated vendored path prefixes")
	flag.StringVar(&generatedHeader, "generated-header", "", "exact generated-file header")
	flag.IntVar(&maxFiles, "max-files", 1024, "maximum regular files")
	flag.Int64Var(&maxFileBytes, "max-file-bytes", 2<<20, "maximum bytes per file")
	flag.Int64Var(&maxTotalBytes, "max-total-bytes", 32<<20, "maximum aggregate bytes")
	flag.StringVar(&out, "out", "", "new JSON snapshot destination")
	flag.Parse()
	for name, value := range map[string]string{"root": root, "identity": identity, "language": language, "toolchain": toolchain, "profile": profile, "semantic-revision": semanticRevision, "tracked-extensions": tracked, "out": out} {
		if value == "" {
			return fmt.Errorf("project_source_discover.missing:%s", name)
		}
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	if _, err = os.Lstat(absOut); err == nil {
		return fmt.Errorf("project_source_discover.output_exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	snapshot, err := projectsource.Discover(absRoot, identity, projectsource.Toolchain{Language: language, Toolchain: toolchain, Profile: profile, SemanticRevision: semanticRevision}, projectsource.Policy{
		TrackedExtensions: split(tracked), IgnoredPrefixes: split(ignoredPrefixes), IgnoredSuffixes: split(ignoredSuffixes), VendoredPrefixes: split(vendoredPrefixes), GeneratedHeader: []byte(generatedHeader), MaxFiles: maxFiles, MaxFileBytes: maxFileBytes, MaxTotalBytes: maxTotalBytes,
	})
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	parent := filepath.Dir(absOut)
	temp, err := os.CreateTemp(parent, ".project-source-discover-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tempName)
		}
	}()
	if _, err = temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err = temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	if _, err = os.Lstat(absOut); err == nil {
		return fmt.Errorf("project_source_discover.output_exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err = os.Link(tempName, absOut); err != nil {
		return fmt.Errorf("project_source_discover.publish:%w", err)
	}
	ok = true
	if err = os.Remove(tempName); err != nil {
		return err
	}
	return nil
}

func split(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	for _, part := range parts {
		if part == "" {
			return []string{""}
		}
	}
	return parts
}
