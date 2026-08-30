package goprovider

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"seme.local/reference/wire"
)

const ManifestVersion = 1

type Manifest struct {
	Version      int            `json:"version"`
	Contract     string         `json:"contract"`
	Provider     string         `json:"provider"`
	Profile      Profile        `json:"profile"`
	Revision     string         `json:"revision"`
	Files        []NativeFile   `json:"files"`
	Declarations []Declaration  `json:"declarations"`
	Opaque       []OpaqueRegion `json:"opaque_regions"`
}
type Profile struct {
	Identity              string   `json:"identity"`
	ImplementationVersion string   `json:"implementation_version"`
	Language              string   `json:"language"`
	LanguageVersion       string   `json:"language_version"`
	Toolchain             string   `json:"toolchain"`
	Target                string   `json:"target"`
	Environment           string   `json:"environment"`
	Supported             []string `json:"supported_constructs"`
	Exclusions            []string `json:"exclusions"`
}
type NativeFile struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Digest string `json:"digest"`
}
type Occurrence struct {
	ID    string `json:"id"`
	File  string `json:"file"`
	Start int    `json:"start"`
	End   int    `json:"end"`
	Role  uint64 `json:"role"`
}
type Declaration struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Qualified        string       `json:"qualified_name"`
	Signature        string       `json:"signature"`
	NativeKey        string       `json:"native_key"`
	Fingerprint      string       `json:"semantic_fingerprint"`
	MatchFingerprint string       `json:"matching_fingerprint"`
	EvidenceID       string       `json:"identity_evidence_id"`
	Occurrences      []Occurrence `json:"occurrences"`
}
type OpaqueRegion struct {
	ID     string `json:"id"`
	File   string `json:"file"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Digest string `json:"digest"`
}
type IngestOptions struct {
	Project    string
	ModuleG1   string
	Prior      *Manifest
	ProviderID string
}
type ProjectionReport struct {
	BaseRevision     string   `json:"base_revision"`
	ResultRevision   string   `json:"result_revision"`
	ChangedFiles     []string `json:"changed_files"`
	ChangedRanges    []Range  `json:"changed_ranges"`
	Validation       string   `json:"validation_command"`
	ValidationStatus int      `json:"validation_status"`
}
type Range struct {
	File  string `json:"file"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

func Ingest(options IngestOptions) (Manifest, string, error) {
	project, err := filepath.Abs(options.Project)
	if err != nil {
		return Manifest{}, "", err
	}
	module, err := os.ReadFile(options.ModuleG1)
	if err != nil {
		return Manifest{}, "", err
	}
	packagePath, err := goOutput(project, "list", "-f", "{{.ImportPath}}", ".")
	if err != nil {
		return Manifest{}, "", err
	}
	goVersion, err := goOutput(project, "env", "GOVERSION")
	if err != nil {
		return Manifest{}, "", err
	}
	target, err := goOutput(project, "env", "GOOS", "GOARCH")
	if err != nil {
		return Manifest{}, "", err
	}
	files, sources, err := nativeFiles(project)
	if err != nil {
		return Manifest{}, "", err
	}
	revision := revisionOf(files)
	decls, err := declarations(project, packagePath, files, sources, options.Prior)
	if err != nil {
		return Manifest{}, "", err
	}
	providerID := options.ProviderID
	if providerID == "" {
		providerID = "seme.go-provider.v1"
	}
	profile := Profile{
		Identity: providerID, ImplementationVersion: "1", Language: "go",
		LanguageVersion: strings.TrimPrefix(goVersion, "go"), Toolchain: goVersion,
		Target: strings.Join(strings.Fields(target), "/"), Environment: "ordinary-module",
		Supported:  []string{"package-level-functions", "resolved-function-occurrences", "identity-preserving-rename", "module-source-closure-revisions", "multi-package-declarations"},
		Exclusions: []string{"cgo", "generated-files", "build-tag-variants", "methods", "native-concurrent-edits"},
	}
	manifest := Manifest{Version: ManifestVersion, Contract: "provider-v1", Provider: providerID, Profile: profile, Revision: revision, Files: files, Declarations: decls}
	for _, file := range files {
		manifest.Opaque = append(manifest.Opaque, OpaqueRegion{ID: stableID("opaque", file.ID), File: file.ID, Start: 0, End: len(sources[file.Path]), Digest: file.Digest})
	}
	g1, err := emitG1(module, manifest)
	return manifest, g1, err
}

func ReadManifest(path string) (Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, err
	}
	if m.Version != ManifestVersion || m.Contract != "provider-v1" {
		return Manifest{}, errors.New("provider.unsupported_manifest")
	}
	return m, nil
}

func WriteIngestion(directory string, manifest Manifest, g1 string) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := atomicWrite(filepath.Join(directory, "manifest.json"), b, 0o644); err != nil {
		return err
	}
	return atomicWrite(filepath.Join(directory, "program.g1"), []byte(g1), 0o644)
}

func RenameFromGraphs(project string, manifest Manifest, target, basePath, candidatePath string, validate bool) (ProjectionReport, error) {
	base, err := wire.Read(basePath)
	if err != nil {
		return ProjectionReport{}, err
	}
	candidate, err := wire.Read(candidatePath)
	if err != nil {
		return ProjectionReport{}, err
	}
	if base.Revision.String() != manifest.Revision {
		return ProjectionReport{}, errors.New("provider.stale_semantic_revision")
	}
	if candidate.Revision == base.Revision {
		return ProjectionReport{}, errors.New("provider.uncommitted_candidate")
	}
	targetID, err := wire.ParseID(target)
	if err != nil {
		return ProjectionReport{}, err
	}
	nameField, _ := wire.ParseID("00000000000000000000000000007130")
	baseEntity, ok := base.Entities[targetID]
	if !ok {
		return ProjectionReport{}, errors.New("provider.unknown_semantic_target")
	}
	candidateEntity, ok := candidate.Entities[targetID]
	if !ok {
		return ProjectionReport{}, errors.New("provider.deleted_semantic_target")
	}
	baseName, ok := baseEntity.Fields[nameField]
	if !ok || baseName.Tag != 5 {
		return ProjectionReport{}, errors.New("provider.invalid_base_name")
	}
	candidateName, ok := candidateEntity.Fields[nameField]
	if !ok || candidateName.Tag != 5 {
		return ProjectionReport{}, errors.New("provider.invalid_candidate_name")
	}
	if bytes.Equal(baseName.Bytes, candidateName.Bytes) {
		return ProjectionReport{}, errors.New("provider.missing_semantic_change")
	}
	if len(base.Entities) != len(candidate.Entities) {
		return ProjectionReport{}, errors.New("provider.unsupported_semantic_change")
	}
	for identity, before := range base.Entities {
		after, exists := candidate.Entities[identity]
		if !exists {
			return ProjectionReport{}, errors.New("provider.unsupported_semantic_change")
		}
		if identity == targetID {
			before.Fields = cloneWireFields(before.Fields)
			after.Fields = cloneWireFields(after.Fields)
			delete(before.Fields, nameField)
			delete(after.Fields, nameField)
		}
		if !reflect.DeepEqual(before, after) {
			return ProjectionReport{}, errors.New("provider.unsupported_semantic_change")
		}
	}
	report, err := rename(project, manifest, target, string(baseName.Bytes), string(candidateName.Bytes), validate)
	report.ResultRevision = candidate.Revision.String()
	return report, err
}

func rename(project string, manifest Manifest, target, expected, replacement string, validate bool) (ProjectionReport, error) {
	if expected == "" || replacement == "" {
		return ProjectionReport{}, errors.New("provider.invalid_rename")
	}
	currentFiles, _, err := nativeFiles(project)
	if err != nil {
		return ProjectionReport{}, err
	}
	if revisionOf(currentFiles) != manifest.Revision {
		return ProjectionReport{}, errors.New("provider.stale_native_revision")
	}
	var declaration *Declaration
	for i := range manifest.Declarations {
		if manifest.Declarations[i].ID == target {
			declaration = &manifest.Declarations[i]
			break
		}
	}
	if declaration == nil {
		return ProjectionReport{}, errors.New("provider.unknown_target")
	}
	if declaration.Name != expected {
		return ProjectionReport{}, errors.New("provider.precondition_failed")
	}
	byPath := map[string][]Occurrence{}
	filePath := map[string]string{}
	for _, file := range manifest.Files {
		filePath[file.ID] = file.Path
	}
	for _, occurrence := range declaration.Occurrences {
		byPath[filePath[occurrence.File]] = append(byPath[filePath[occurrence.File]], occurrence)
	}
	if len(byPath) != 1 {
		return ProjectionReport{}, errors.New("provider.profile_requires_single_changed_file")
	}
	report := ProjectionReport{BaseRevision: manifest.Revision}
	for relative, occurrences := range byPath {
		path := filepath.Join(project, filepath.FromSlash(relative))
		original, err := os.ReadFile(path)
		if err != nil {
			return ProjectionReport{}, err
		}
		sort.Slice(occurrences, func(i, j int) bool { return occurrences[i].Start > occurrences[j].Start })
		changed := append([]byte(nil), original...)
		for _, occurrence := range occurrences {
			if occurrence.Start < 0 || occurrence.End > len(changed) || occurrence.Start > occurrence.End || string(changed[occurrence.Start:occurrence.End]) != expected {
				return ProjectionReport{}, errors.New("provider.occurrence_precondition_failed")
			}
			changed = append(changed[:occurrence.Start], append([]byte(replacement), changed[occurrence.End:]...)...)
			report.ChangedRanges = append(report.ChangedRanges, Range{relative, occurrence.Start, occurrence.End})
		}
		if _, err := parser.ParseFile(token.NewFileSet(), relative, changed, parser.AllErrors); err != nil {
			return ProjectionReport{}, fmt.Errorf("provider.projected_syntax:%w", err)
		}
		stage, err := os.MkdirTemp(filepath.Dir(project), ".seme-go-provider-v1-")
		if err != nil {
			return ProjectionReport{}, err
		}
		defer os.RemoveAll(stage)
		if err := copyTree(project, stage); err != nil {
			return ProjectionReport{}, err
		}
		stagePath := filepath.Join(stage, filepath.FromSlash(relative))
		if err := atomicWrite(stagePath, changed, 0o644); err != nil {
			return ProjectionReport{}, err
		}
		if validate {
			report.Validation = "go test ./..."
			cmd := exec.Command("go", "test", "./...")
			cmd.Dir = stage
			if output, err := cmd.CombinedOutput(); err != nil {
				return ProjectionReport{}, fmt.Errorf("provider.native_validation:%w:%s", err, output)
			}
		}
		if err := atomicWrite(path, changed, 0o644); err != nil {
			return ProjectionReport{}, err
		}
		report.ChangedFiles = []string{relative}
	}
	sort.Slice(report.ChangedRanges, func(i, j int) bool {
		if report.ChangedRanges[i].File != report.ChangedRanges[j].File {
			return report.ChangedRanges[i].File < report.ChangedRanges[j].File
		}
		return report.ChangedRanges[i].Start < report.ChangedRanges[j].Start
	})
	return report, nil
}

func cloneWireFields(fields map[wire.ID]wire.Value) map[wire.ID]wire.Value {
	out := make(map[wire.ID]wire.Value, len(fields))
	for identity, value := range fields {
		out[identity] = value
	}
	return out
}

func WriteProjectionReport(path string, report ProjectionReport) error {
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, append(b, '\n'), 0o644)
}

func ReadProjectionReport(path string) (ProjectionReport, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return ProjectionReport{}, err
	}
	var report ProjectionReport
	if err := json.Unmarshal(b, &report); err != nil {
		return ProjectionReport{}, err
	}
	return report, nil
}

func declarations(project, packagePath string, files []NativeFile, sources map[string][]byte, prior *Manifest) ([]Declaration, error) {
	directories := map[string]bool{"": true}
	for _, file := range files {
		if strings.HasSuffix(file.Path, ".go") && !strings.HasSuffix(file.Path, "_test.go") {
			directory := filepath.ToSlash(filepath.Dir(file.Path))
			if directory == "." {
				directory = ""
			}
			directories[directory] = true
		}
	}
	ordered := make([]string, 0, len(directories))
	for directory := range directories {
		ordered = append(ordered, directory)
	}
	sort.Strings(ordered)
	var out []Declaration
	for _, directory := range ordered {
		path := packagePath
		if directory != "" {
			path += "/" + directory
		}
		items, err := declarationsForPackage(project, path, directory, files, sources, prior)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func declarationsForPackage(project, packagePath, directory string, files []NativeFile, sources map[string][]byte, prior *Manifest) ([]Declaration, error) {
	fset := token.NewFileSet()
	parsed := make([]*ast.File, 0, len(files))
	pathFor := map[*ast.File]string{}
	for _, file := range files {
		if !strings.HasSuffix(file.Path, ".go") || strings.HasSuffix(file.Path, "_test.go") {
			continue
		}
		fileDirectory := filepath.ToSlash(filepath.Dir(file.Path))
		if fileDirectory == "." {
			fileDirectory = ""
		}
		if fileDirectory != directory {
			continue
		}
		parsedFile, err := parser.ParseFile(fset, filepath.Join(project, filepath.FromSlash(file.Path)), sources[file.Path], parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, parsedFile)
		pathFor[parsedFile] = file.Path
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	rootPackage := strings.TrimSuffix(packagePath, "/"+directory)
	config := types.Config{Importer: newSourceImporter(project, Manifest{Files: files}, rootPackage)}
	pkg, err := config.Check(packagePath, fset, parsed, info)
	if err != nil {
		return nil, err
	}
	fileIDs := map[string]string{}
	for _, file := range files {
		fileIDs[file.Path] = file.ID
	}
	var funcs []*types.Func
	declNode := map[*types.Func]*ast.FuncDecl{}
	declFile := map[*types.Func]string{}
	for _, file := range parsed {
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			object, ok := info.Defs[fn.Name].(*types.Func)
			if !ok || object.Parent() != pkg.Scope() {
				continue
			}
			funcs = append(funcs, object)
			declNode[object] = fn
			declFile[object] = pathFor[file]
		}
	}
	sort.Slice(funcs, func(i, j int) bool { return funcs[i].FullName() < funcs[j].FullName() })
	priorKey := map[string]Declaration{}
	priorFingerprint := map[string][]Declaration{}
	if prior != nil {
		for _, declaration := range prior.Declarations {
			priorKey[declaration.NativeKey] = declaration
			fingerprint := declaration.MatchFingerprint
			if fingerprint == "" {
				fingerprint = declaration.Fingerprint
			}
			priorFingerprint[fingerprint] = append(priorFingerprint[fingerprint], declaration)
		}
	}
	type candidate struct {
		object                     *types.Func
		node                       *ast.FuncDecl
		relative, nativeKey        string
		matchFingerprint, identity string
	}
	candidates := make([]candidate, 0, len(funcs))
	for _, object := range funcs {
		fn, relative := declNode[object], declFile[object]
		candidates = append(candidates, candidate{
			object: object, node: fn, relative: relative,
			nativeKey:        packagePath + "\x00func\x00" + object.Name(),
			matchFingerprint: functionFingerprint(fset, fn, info, sources[relative], map[*types.Func]string{object: "_seme_symbol_"}),
		})
	}
	used := map[string]bool{}
	for i := range candidates {
		candidate := &candidates[i]
		identity := ""
		if previous, ok := priorKey[candidate.nativeKey]; ok {
			identity = previous.ID
		}
		if identity == "" {
			var matches []Declaration
			for _, previous := range priorFingerprint[candidate.matchFingerprint] {
				if !used[previous.ID] {
					matches = append(matches, previous)
				}
			}
			if len(matches) > 1 {
				return nil, fmt.Errorf("provider.identity_ambiguous:%s", candidate.object.Name())
			}
			if len(matches) == 1 {
				identity = matches[0].ID
			}
		}
		if identity == "" {
			identity = stableID("declaration", candidate.nativeKey)
		}
		if used[identity] {
			return nil, fmt.Errorf("provider.identity_reused:%s", identity)
		}
		used[identity] = true
		candidate.identity = identity
	}
	semanticIDs := map[*types.Func]string{}
	for _, candidate := range candidates {
		semanticIDs[candidate.object] = candidate.identity
	}
	out := make([]Declaration, 0, len(funcs))
	for _, candidate := range candidates {
		object, fn, relative := candidate.object, candidate.node, candidate.relative
		fingerprint := functionFingerprint(fset, fn, info, sources[relative], semanticIDs)
		declaration := Declaration{ID: candidate.identity, Name: object.Name(), Qualified: packagePath + "." + object.Name(), Signature: types.TypeString(object.Type(), qualifier), NativeKey: candidate.nativeKey, Fingerprint: fingerprint, MatchFingerprint: candidate.matchFingerprint, EvidenceID: stableID("evidence", candidate.identity)}
		for identifier, definition := range info.Defs {
			if definition == object {
				declaration.Occurrences = append(declaration.Occurrences, occurrence(fset, identifier, relative, fileIDs[relative], candidate.identity, 0))
			}
		}
		for identifier, use := range info.Uses {
			if use == object {
				pos := fset.Position(identifier.Pos())
				declaration.Occurrences = append(declaration.Occurrences, occurrence(fset, identifier, filepath.ToSlash(relativePath(project, pos.Filename)), fileIDs[filepath.ToSlash(relativePath(project, pos.Filename))], candidate.identity, 1))
			}
		}
		sort.Slice(declaration.Occurrences, func(i, j int) bool {
			a, b := declaration.Occurrences[i], declaration.Occurrences[j]
			if a.File != b.File {
				return a.File < b.File
			}
			return a.Start < b.Start
		})
		out = append(out, declaration)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func occurrence(fset *token.FileSet, identifier *ast.Ident, relative, fileID, declarationID string, role uint64) Occurrence {
	position := fset.Position(identifier.Pos())
	start := position.Offset
	return Occurrence{ID: stableID("occurrence", declarationID, relative, strconv.Itoa(start), strconv.FormatUint(role, 10)), File: fileID, Start: start, End: start + len(identifier.Name), Role: role}
}
func functionFingerprint(fset *token.FileSet, fn *ast.FuncDecl, info *types.Info, source []byte, identities map[*types.Func]string) string {
	start := fset.Position(fn.Pos()).Offset
	end := fset.Position(fn.End()).Offset
	part := append([]byte(nil), source[start:end]...)
	type replacement struct {
		start, end int
		value      string
	}
	var replacements []replacement
	ast.Inspect(fn, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		object, _ := info.Defs[identifier].(*types.Func)
		if object == nil {
			object, _ = info.Uses[identifier].(*types.Func)
		}
		if value, ok := identities[object]; ok {
			s := fset.Position(identifier.Pos()).Offset - start
			replacements = append(replacements, replacement{s, s + len(identifier.Name), value})
		}
		return true
	})
	sort.Slice(replacements, func(i, j int) bool { return replacements[i].start > replacements[j].start })
	for _, r := range replacements {
		part = append(part[:r.start], append([]byte(r.value), part[r.end:]...)...)
	}
	sum := sha256.Sum256(part)
	return hex.EncodeToString(sum[:])
}
func nativeFiles(project string) ([]NativeFile, map[string][]byte, error) {
	var files []NativeFile
	sources := map[string][]byte{}
	err := filepath.WalkDir(project, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != project && (entry.Name() == ".git" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(project, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if !strings.HasSuffix(relative, ".go") && relative != "go.mod" && relative != "go.sum" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		digest := hex.EncodeToString(sum[:])
		files = append(files, NativeFile{ID: stableID("file", relative), Path: relative, Digest: digest})
		sources[relative] = b
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if len(files) == 0 {
		return nil, nil, errors.New("provider.no_supported_go_files")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, sources, nil
}
func revisionOf(files []NativeFile) string {
	h := sha256.New()
	for _, f := range files {
		h.Write([]byte(f.Path))
		h.Write([]byte{0})
		d, _ := hex.DecodeString(f.Digest)
		h.Write(d)
	}
	return hex.EncodeToString(h.Sum(nil)[:16])
}
func stableID(parts ...string) string {
	h := sha256.New()
	h.Write([]byte("seme.provider.identity.v1\x00"))
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	sum := h.Sum(nil)
	b := make([]byte, 16)
	b[0] = 0x80
	copy(b[1:], sum[:15])
	return hex.EncodeToString(b)
}
func qualifier(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}
	return pkg.Path()
}
func relativePath(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return relative
}
func goOutput(directory string, args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = directory
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("provider.go_%s:%w:%s", args[0], err, b)
	}
	return strings.TrimSpace(string(b)), nil
}
func atomicWrite(path string, data []byte, mode fs.FileMode) error {
	directory := filepath.Dir(path)
	f, err := os.CreateTemp(directory, ".seme-provider-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return atomicWrite(target, b, info.Mode().Perm())
	})
}
