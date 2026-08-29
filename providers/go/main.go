package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
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
	loaded, err := packages.Load(&packages.Config{
		Dir: abs,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedImports | packages.NeedModule,
		Tests: true,
	}, "./...")
	if err != nil {
		return Program{}, fmt.Errorf("Go package load: %w", err)
	}
	var loadErrors []string
	for _, pkg := range loaded {
		for _, problem := range pkg.Errors {
			loadErrors = append(loadErrors, problem.Error())
		}
	}
	if len(loadErrors) != 0 {
		sort.Strings(loadErrors)
		return Program{}, fmt.Errorf("Go package load: %s", strings.Join(loadErrors, "; "))
	}
	fileSet := map[string]bool{}
	for _, pkg := range loaded {
		for _, path := range append(append([]string{}, pkg.GoFiles...), pkg.CompiledGoFiles...) {
			if within(abs, path) {
				fileSet[path] = true
			}
		}
	}
	paths := make([]string, 0, len(fileSet))
	for path := range fileSet {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var nativeFiles []NativeFile
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return Program{}, err
		}
		rel, _ := filepath.Rel(abs, path)
		nativeFiles = append(nativeFiles, NativeFile{Path: rel, SHA256: digest(data)})
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
	entityByAnchor := map[string]Entity{}
	occurrenceSets := map[string]map[string]Occurrence{}
	for _, pkg := range loaded {
		for _, file := range pkg.Syntax {
			filename := pkg.Fset.Position(file.Pos()).Filename
			if !within(abs, filename) {
				continue
			}
			rel, _ := filepath.Rel(abs, filename)
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil {
					continue
				}
				obj := pkg.TypesInfo.Defs[fn.Name]
				if obj == nil {
					continue
				}
				start := pkg.Fset.Position(fn.Name.Pos()).Offset
				end := pkg.Fset.Position(fn.Name.End()).Offset
				key := anchor(rel, "function", start)
				entity, exists := entityByAnchor[key]
				if !exists {
					id := stableID(pkg.PkgPath, rel, "function", start)
					nameMatches := priorByName[rel+":"+"function"+":"+fn.Name.Name]
					if len(nameMatches) == 1 {
						id = nameMatches[0].ID
					} else if old, ok := priorByAnchor[key]; ok {
						id = old.ID
					}
					entity = Entity{ID: id, Kind: "function", Name: fn.Name.Name, Package: pkg.PkgPath, File: rel, Declaration: Occurrence{File: rel, Start: start, End: end}, Type: obj.Type().String(), Fidelity: "refined"}
					occurrenceSets[key] = map[string]Occurrence{}
				}
				for ident, used := range pkg.TypesInfo.Defs {
					if used == obj {
						addOccurrence(occurrenceSets[key], pkg.Fset, abs, ident)
					}
				}
				for ident, used := range pkg.TypesInfo.Uses {
					if used == obj {
						addOccurrence(occurrenceSets[key], pkg.Fset, abs, ident)
					}
				}
				entityByAnchor[key] = entity
			}
		}
	}
	entities := make([]Entity, 0, len(entityByAnchor))
	for key, entity := range entityByAnchor {
		for _, occ := range occurrenceSets[key] {
			entity.Occurrences = append(entity.Occurrences, occ)
		}
		sort.Slice(entity.Occurrences, func(i, j int) bool {
			x, y := entity.Occurrences[i], entity.Occurrences[j]
			if x.File == y.File {
				return x.Start < y.Start
			}
			return x.File < y.File
		})
		entities = append(entities, entity)
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i].ID < entities[j].ID })
	p := Program{Contract: contractVersion, Provider: "seme.go/0.2", Profile: Profile{Language: "Go", Version: strings.TrimPrefix(runtime.Version(), "go"), Toolchain: "go/packages", Target: runtime.GOOS + "/" + runtime.GOARCH, Supports: []string{"module package loading", "package functions", "resolved function references", "semantic rename", "native test variants"}, Excludes: []string{"methods", "cgo semantics", "unsafe semantics", "generated-file editing"}}, Root: abs, Files: nativeFiles, Entities: entities}
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
func addOccurrence(set map[string]Occurrence, fset *token.FileSet, root string, ident *ast.Ident) {
	occ := occurrence(fset, root, ident)
	if strings.HasPrefix(occ.File, "..") {
		return
	}
	set[fmt.Sprintf("%s:%d:%d", occ.File, occ.Start, occ.End)] = occ
}
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
