package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const contractVersion = "seme.provider/v1"

type Profile struct {
	Language  string   `json:"language"`
	Version   string   `json:"version"`
	Toolchain string   `json:"toolchain"`
	Target    string   `json:"target"`
	Supports  []string `json:"supports"`
	Excludes  []string `json:"excludes"`
}

type Occurrence struct {
	File  string `json:"file"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

type Entity struct {
	ID          string       `json:"id"`
	Kind        string       `json:"kind"`
	Name        string       `json:"name"`
	Package     string       `json:"package"`
	File        string       `json:"file"`
	Declaration Occurrence   `json:"declaration"`
	Type        string       `json:"type"`
	Fidelity    string       `json:"fidelity"`
	Occurrences []Occurrence `json:"occurrences"`
}

type NativeFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type Program struct {
	Contract string       `json:"contract"`
	Provider string       `json:"provider"`
	Profile  Profile      `json:"profile"`
	Root     string       `json:"root"`
	Revision string       `json:"revision"`
	Files    []NativeFile `json:"files"`
	Entities []Entity     `json:"entities"`
}

type Rename struct {
	Entity       string `json:"entity"`
	ExpectedName string `json:"expected_name"`
	NewName      string `json:"new_name"`
}

type Patch struct {
	Contract     string   `json:"contract"`
	BaseRevision string   `json:"base_revision"`
	Renames      []Rename `json:"renames"`
}

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: seme-go <import|rename|project|verify> [flags]")
	}
	var err error
	switch os.Args[1] {
	case "import":
		err = importCommand(os.Args[2:])
	case "rename":
		err = renameCommand(os.Args[2:])
	case "project":
		err = projectCommand(os.Args[2:])
	case "verify":
		err = verifyCommand(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fatalf("%v", err)
	}
}

func importCommand(args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	root := fs.String("root", ".", "ordinary Go project root")
	out := fs.String("out", "seme-program.json", "ingestion output")
	previous := fs.String("previous", "", "previous ingestion for identity recovery")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var prior *Program
	if *previous != "" {
		p := new(Program)
		if err := readJSON(*previous, p); err != nil {
			return err
		}
		prior = p
	}
	program, err := ingest(*root, prior)
	if err != nil {
		return err
	}
	return writeJSON(*out, program)
}

func renameCommand(args []string) error {
	fs := flag.NewFlagSet("rename", flag.ContinueOnError)
	programPath := fs.String("program", "seme-program.json", "ingested program")
	entityID := fs.String("entity", "", "semantic entity identity")
	name := fs.String("name", "", "unique declaration name convenience selector")
	to := fs.String("to", "", "new declaration name")
	out := fs.String("out", "seme-patch.json", "semantic patch output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *to == "" || (*entityID == "" && *name == "") {
		return errors.New("rename requires -to and either -entity or -name")
	}
	var p Program
	if err := readJSON(*programPath, &p); err != nil {
		return err
	}
	var matches []Entity
	for _, entity := range p.Entities {
		if (*entityID != "" && entity.ID == *entityID) || (*entityID == "" && entity.Name == *name) {
			matches = append(matches, entity)
		}
	}
	if len(matches) != 1 {
		return fmt.Errorf("selector resolved to %d entities", len(matches))
	}
	patch := Patch{Contract: "seme.patch/v1", BaseRevision: p.Revision, Renames: []Rename{{Entity: matches[0].ID, ExpectedName: matches[0].Name, NewName: *to}}}
	return writeJSON(*out, patch)
}

func projectCommand(args []string) error {
	fs := flag.NewFlagSet("project", flag.ContinueOnError)
	programPath := fs.String("program", "seme-program.json", "ingested program")
	patchPath := fs.String("patch", "seme-patch.json", "semantic patch")
	validate := fs.Bool("validate", false, "run gofmt and go test after projection")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var p Program
	var patch Patch
	if err := readJSON(*programPath, &p); err != nil {
		return err
	}
	if err := readJSON(*patchPath, &patch); err != nil {
		return err
	}
	if patch.BaseRevision != p.Revision {
		return errors.New("patch base revision does not match program")
	}
	if err := verifyNativeDigests(p); err != nil {
		return err
	}
	if err := applyPatch(p, patch); err != nil {
		return err
	}
	if *validate {
		cmd := exec.Command("go", "fmt", "./...")
		cmd.Dir = p.Root
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("go fmt: %v: %s", err, output)
		}
		cmd = exec.Command("go", "test", "./...")
		cmd.Dir = p.Root
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("go test: %v: %s", err, output)
		}
	}
	return nil
}

func verifyCommand(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	beforePath := fs.String("before", "seme-program.json", "program before patch")
	afterPath := fs.String("after", "seme-program-after.json", "re-ingested program")
	patchPath := fs.String("patch", "seme-patch.json", "applied patch")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var before, after Program
	var patch Patch
	if err := readJSON(*beforePath, &before); err != nil {
		return err
	}
	if err := readJSON(*afterPath, &after); err != nil {
		return err
	}
	if err := readJSON(*patchPath, &patch); err != nil {
		return err
	}
	renamed := map[string]Rename{}
	for _, rename := range patch.Renames {
		renamed[rename.Entity] = rename
	}
	afterByID := map[string]Entity{}
	for _, entity := range after.Entities {
		afterByID[entity.ID] = entity
	}
	for _, entity := range before.Entities {
		got, ok := afterByID[entity.ID]
		if !ok {
			return fmt.Errorf("identity %s was not recovered", entity.ID)
		}
		if rename, changed := renamed[entity.ID]; changed {
			if got.Name != rename.NewName {
				return fmt.Errorf("identity %s has name %q, want %q", entity.ID, got.Name, rename.NewName)
			}
		} else if got.Name != entity.Name || got.Type != entity.Type || got.Kind != entity.Kind {
			return fmt.Errorf("unaffected identity %s changed semantics", entity.ID)
		}
	}
	fmt.Printf("verified %d stable identities across revision %s -> %s\n", len(before.Entities), before.Revision[:12], after.Revision[:12])
	return nil
}

func ingest(root string, previous *Program) (Program, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Program{}, err
	}
	module, err := modulePath(abs)
	if err != nil {
		return Program{}, err
	}
	fset := token.NewFileSet()
	var paths []string
	err = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "vendor") && path != abs {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return Program{}, err
	}
	sort.Strings(paths)
	parsed := make([]*ast.File, 0, len(paths))
	pathForFile := map[*ast.File]string{}
	var nativeFiles []NativeFile
	packageNames := map[string]bool{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return Program{}, err
		}
		file, err := parser.ParseFile(fset, path, data, parser.ParseComments)
		if err != nil {
			return Program{}, err
		}
		parsed = append(parsed, file)
		packageNames[file.Name.Name] = true
		pathForFile[file], _ = filepath.Rel(abs, path)
		nativeFiles = append(nativeFiles, NativeFile{Path: pathForFile[file], SHA256: digest(data)})
	}
	if len(packageNames) != 1 {
		return Program{}, fmt.Errorf("provider proof requires one package, found %d", len(packageNames))
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check(module, fset, parsed, info)
	if err != nil {
		return Program{}, fmt.Errorf("Go provider type check: %w", err)
	}
	packageName := module
	if pkg != nil {
		packageName = pkg.Path()
	}
	priorByAnchor := map[string]Entity{}
	priorByName := map[string][]Entity{}
	if previous != nil {
		for _, entity := range previous.Entities {
			priorByAnchor[anchor(entity.File, entity.Kind, entity.Declaration.Start)] = entity
			key := entity.File + ":" + entity.Kind + ":" + entity.Name
			priorByName[key] = append(priorByName[key], entity)
		}
	}
	type declaration struct {
		entity Entity
		object types.Object
	}
	var declarations []declaration
	for _, file := range parsed {
		rel := pathForFile[file]
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			start := fset.Position(fn.Name.Pos()).Offset
			end := fset.Position(fn.Name.End()).Offset
			id := stableID(module, rel, "function", start)
			nameMatches := priorByName[rel+":"+"function"+":"+fn.Name.Name]
			if len(nameMatches) == 1 {
				id = nameMatches[0].ID
			} else if old, ok := priorByAnchor[anchor(rel, "function", start)]; ok {
				id = old.ID
			}
			typeText := "unresolved"
			if obj := info.Defs[fn.Name]; obj != nil {
				typeText = obj.Type().String()
			}
			entity := Entity{ID: id, Kind: "function", Name: fn.Name.Name, Package: packageName, File: rel, Declaration: Occurrence{File: rel, Start: start, End: end}, Type: typeText, Fidelity: "refined"}
			declarations = append(declarations, declaration{entity: entity, object: info.Defs[fn.Name]})
		}
	}
	for i := range declarations {
		for ident, obj := range info.Defs {
			if obj != nil && obj == declarations[i].object {
				declarations[i].entity.Occurrences = append(declarations[i].entity.Occurrences, occurrence(fset, abs, ident))
			}
		}
		for ident, obj := range info.Uses {
			if obj != nil && obj == declarations[i].object {
				declarations[i].entity.Occurrences = append(declarations[i].entity.Occurrences, occurrence(fset, abs, ident))
			}
		}
		sort.Slice(declarations[i].entity.Occurrences, func(a, b int) bool {
			x, y := declarations[i].entity.Occurrences[a], declarations[i].entity.Occurrences[b]
			if x.File == y.File {
				return x.Start < y.Start
			}
			return x.File < y.File
		})
	}
	entities := make([]Entity, len(declarations))
	for i := range declarations {
		entities[i] = declarations[i].entity
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i].ID < entities[j].ID })
	p := Program{Contract: contractVersion, Provider: "seme.go/0.1", Profile: Profile{Language: "Go", Version: strings.TrimPrefix(runtime.Version(), "go"), Toolchain: "go/parser+go/types", Target: runtime.GOOS + "/" + runtime.GOARCH, Supports: []string{"single-package trees", "package functions", "resolved function references", "semantic rename"}, Excludes: []string{"multi-package trees", "methods", "cgo", "unsafe semantics", "generated files", "build-tag variants"}}, Root: abs, Files: nativeFiles, Entities: entities}
	p.Revision = programRevision(p)
	return p, nil
}

func applyPatch(p Program, patch Patch) error {
	entities := map[string]Entity{}
	for _, e := range p.Entities {
		entities[e.ID] = e
	}
	type edit struct {
		start, end       int
		old, replacement string
	}
	byFile := map[string][]edit{}
	for _, rename := range patch.Renames {
		entity, ok := entities[rename.Entity]
		if !ok {
			return fmt.Errorf("unknown entity %s", rename.Entity)
		}
		if entity.Name != rename.ExpectedName {
			return fmt.Errorf("rename precondition failed for %s", rename.Entity)
		}
		if !token.IsIdentifier(rename.NewName) || token.Lookup(rename.NewName).IsKeyword() {
			return fmt.Errorf("%q is not a valid Go identifier", rename.NewName)
		}
		for _, occ := range entity.Occurrences {
			byFile[occ.File] = append(byFile[occ.File], edit{occ.Start, occ.End, entity.Name, rename.NewName})
		}
	}
	changed := map[string][]byte{}
	for rel, edits := range byFile {
		path := filepath.Join(p.Root, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
		for _, e := range edits {
			if e.start < 0 || e.end > len(data) || string(data[e.start:e.end]) != e.old {
				return fmt.Errorf("source precondition failed at %s:%d", rel, e.start)
			}
			data = append(append(append([]byte{}, data[:e.start]...), e.replacement...), data[e.end:]...)
		}
		changed[path] = data
	}
	// Validate and materialize every edit before replacing any native file. This
	// keeps semantic/precondition failures atomic; an operating-system failure
	// during the final rename phase is reported as an I/O failure.
	for path, data := range changed {
		if err := os.WriteFile(path+".seme-tmp", data, 0o644); err != nil {
			return err
		}
	}
	for path := range changed {
		if err := os.Rename(path+".seme-tmp", path); err != nil {
			return err
		}
	}
	return nil
}

func verifyNativeDigests(p Program) error {
	for _, file := range p.Files {
		data, err := os.ReadFile(filepath.Join(p.Root, file.Path))
		if err != nil {
			return err
		}
		if digest(data) != file.SHA256 {
			return fmt.Errorf("native file changed since ingestion: %s", file.Path)
		}
	}
	return nil
}

func occurrence(fset *token.FileSet, root string, ident *ast.Ident) Occurrence {
	pos := fset.Position(ident.Pos())
	rel, _ := filepath.Rel(root, pos.Filename)
	return Occurrence{File: rel, Start: pos.Offset, End: fset.Position(ident.End()).Offset}
}
func anchor(file, kind string, start int) string { return fmt.Sprintf("%s:%s:%d", file, kind, start) }
func stableID(module, file, kind string, start int) string {
	sum := sha256.Sum256([]byte(anchor(module+"/"+file, kind, start)))
	return "seme:" + hex.EncodeToString(sum[:16])
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func programRevision(p Program) string {
	copy := p
	copy.Revision = ""
	data, _ := json.Marshal(copy)
	return digest(data)
}
func modulePath(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(data))
	for i := range fields {
		if fields[i] == "module" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}
	return "", errors.New("go.mod has no module directive")
}
func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}
func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "seme-go: "+format+"\n", args...)
	os.Exit(1)
}
