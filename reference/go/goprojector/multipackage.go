package goprojector

import (
	"fmt"
	"go/format"
	"path"
	"sort"
	"strconv"
	"strings"

	"seme.local/reference/goprovider"
)

// ProjectPackages emits one ordinary Go file per typed package. Package
// ownership comes only from checked provider metadata, never from names.
func ProjectPackages(g1 []byte, packages []goprovider.PackageMetadata) (map[string][]byte, error) {
	graph, err := parse(g1)
	if err != nil {
		return nil, err
	}
	owners := map[string]string{}
	exported := map[string]bool{}
	metadata := map[string]goprovider.PackageMetadata{}
	rootPackage := ""
	for _, p := range packages {
		if p.Name == "" {
			return nil, fmt.Errorf("go_projection.package_name")
		}
		if _, ok := metadata[p.Name]; ok {
			return nil, fmt.Errorf("go_projection.package_duplicate:%s", p.Name)
		}
		metadata[p.Name] = p
		if p.Root {
			if rootPackage != "" {
				return nil, fmt.Errorf("go_projection.root_package_count")
			}
			rootPackage = p.Name
		}
		seenDependencies := map[string]bool{}
		for _, dependency := range p.Dependencies {
			if dependency == p.Name {
				return nil, fmt.Errorf("go_projection.self_dependency:%s", p.Name)
			}
			if seenDependencies[dependency] {
				return nil, fmt.Errorf("go_projection.duplicate_dependency:%s:%s", p.Name, dependency)
			}
			seenDependencies[dependency] = true
		}
		members, explicit := packageMembers(p)
		memberByID := map[string]goprovider.PackageFunctionMetadata{}
		for _, f := range members {
			if _, ok := owners[f.ID]; ok {
				return nil, fmt.Errorf("go_projection.function_multiple_owners:%s", f.ID)
			}
			if f.ID == "" || f.Name == "" {
				return nil, fmt.Errorf("go_projection.invalid_member:%s", p.Name)
			}
			owners[f.ID] = p.Name
			exported[f.ID] = f.Exported
			memberByID[f.ID] = f
		}
		if explicit {
			seenPublic := map[string]bool{}
			for _, f := range p.Functions {
				member, ok := memberByID[f.ID]
				if !ok || !member.Exported || seenPublic[f.ID] || !sameFunctionMetadata(member, f) {
					return nil, fmt.Errorf("go_projection.export_membership:%s", f.ID)
				}
				seenPublic[f.ID] = true
			}
			for _, member := range members {
				if member.Exported && !seenPublic[member.ID] {
					return nil, fmt.Errorf("go_projection.export_missing:%s", member.ID)
				}
			}
		}
	}
	if rootPackage == "" {
		return nil, fmt.Errorf("go_projection.root_package_count")
	}
	for _, p := range packages {
		for _, dependency := range p.Dependencies {
			if _, ok := metadata[dependency]; !ok {
				return nil, fmt.Errorf("go_projection.unknown_dependency:%s:%s", p.Name, dependency)
			}
		}
	}
	program, ids, err := programMembers(graph)
	if err != nil {
		return nil, err
	}
	entry, err := ref(program, "00000000000000000000000000009151")
	if err != nil || owners[entry] != rootPackage {
		return nil, fmt.Errorf("go_projection.root_entry")
	}
	programSet := map[string]bool{}
	for _, id := range ids {
		programSet[id] = true
		if _, ok := owners[id]; !ok {
			return nil, fmt.Errorf("go_projection.function_owner_missing:%s", id)
		}
	}
	for _, p := range packages {
		members, _ := packageMembers(p)
		for _, f := range members {
			if !programSet[f.ID] {
				return nil, fmt.Errorf("go_projection.function_not_program_member:%s", f.ID)
			}
			if err := validateFunctionMetadata(graph, f); err != nil {
				return nil, err
			}
			for callee := range calledFunctions(graph, f.ID) {
				calleeOwner, ok := owners[callee]
				if !ok {
					return nil, fmt.Errorf("go_projection.call_owner_missing:%s", callee)
				}
				if calleeOwner != p.Name {
					if !exported[callee] {
						return nil, fmt.Errorf("go_projection.inaccessible_member:%s:%s", p.Name, callee)
					}
					if !containsString(p.Dependencies, calleeOwner) {
						return nil, fmt.Errorf("go_projection.undeclared_dependency:%s:%s", p.Name, calleeOwner)
					}
				}
			}
		}
	}
	aliases, err := packageAliases(packages)
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	ordered := append([]goprovider.PackageMetadata(nil), packages...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	for _, p := range ordered {
		source, er := projectPackage(graph, p, owners, aliases)
		if er != nil {
			return nil, er
		}
		out[p.Name] = source
	}
	return out, nil
}

func packageMembers(p goprovider.PackageMetadata) ([]goprovider.PackageFunctionMetadata, bool) {
	if p.Members != nil {
		return p.Members, true
	}
	// Legacy canonical Package metadata exposes interfaces only. Until member
	// ownership is represented in that contract, those interfaces are the
	// complete known member set and are necessarily public.
	out := make([]goprovider.PackageFunctionMetadata, len(p.Functions))
	copy(out, p.Functions)
	for i := range out {
		out[i].Exported = true
	}
	return out, false
}

func sameFunctionMetadata(a, b goprovider.PackageFunctionMetadata) bool {
	if a.ID != b.ID || a.Name != b.Name || a.Result != b.Result || len(a.Parameters) != len(b.Parameters) {
		return false
	}
	for i := range a.Parameters {
		if a.Parameters[i] != b.Parameters[i] {
			return false
		}
	}
	return true
}

func validateFunctionMetadata(graph map[string]entity, metadata goprovider.PackageFunctionMetadata) error {
	fn, ok := graph[metadata.ID]
	if !ok || fn.schema != sFunction {
		return fmt.Errorf("go_projection.invalid_function_member:%s", metadata.ID)
	}
	name, err := text(fn, "00000000000000000000000000009110")
	if err != nil || name != metadata.Name {
		return fmt.Errorf("go_projection.function_name_mismatch:%s", metadata.ID)
	}
	parameters, err := refs(fn, "00000000000000000000000000009111")
	if err != nil || len(parameters) != len(metadata.Parameters) {
		return fmt.Errorf("go_projection.function_parameters_mismatch:%s", metadata.ID)
	}
	for i, parameterID := range parameters {
		parameter, ok := graph[parameterID]
		if !ok || parameter.schema != sParameter {
			return fmt.Errorf("go_projection.invalid_parameter:%s", parameterID)
		}
		typeID, e := ref(parameter, "00000000000000000000000000009121")
		if e != nil || typeID != metadata.Parameters[i] {
			return fmt.Errorf("go_projection.function_parameter_type_mismatch:%s", metadata.ID)
		}
	}
	result, err := ref(fn, "00000000000000000000000000009112")
	if err != nil || result != metadata.Result {
		return fmt.Errorf("go_projection.function_result_mismatch:%s", metadata.ID)
	}
	return nil
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func programMembers(graph map[string]entity) (entity, []string, error) {
	var programs []entity
	for _, e := range graph {
		if e.schema == sProgram {
			programs = append(programs, e)
		}
	}
	if len(programs) != 1 {
		return entity{}, nil, fmt.Errorf("go_projection.requires_one_program")
	}
	ids, err := refs(programs[0], "00000000000000000000000000009150")
	if err != nil || len(ids) == 0 {
		return entity{}, nil, fmt.Errorf("go_projection.invalid_program_members")
	}
	return programs[0], ids, nil
}

func packageAliases(packages []goprovider.PackageMetadata) (map[string]string, error) {
	used := map[string]bool{"bytes": true, "log": true, "maps": true, "slices": true}
	out := map[string]string{}
	ordered := append([]goprovider.PackageMetadata(nil), packages...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	for _, p := range ordered {
		base := path.Base(p.Name)
		if !identifier(base) {
			return nil, fmt.Errorf("go_projection.invalid_package_name:%s", p.Name)
		}
		alias := base
		for n := 2; used[alias]; n++ {
			alias = base + strconv.Itoa(n)
		}
		used[alias] = true
		out[p.Name] = alias
	}
	return out, nil
}

func projectPackage(graph map[string]entity, p goprovider.PackageMetadata, owners, aliases map[string]string) ([]byte, error) {
	packageName := path.Base(p.Name)
	local := map[string]bool{}
	localNames := map[string]bool{}
	functions := map[string]string{}
	for id, owner := range owners {
		fn, ok := graph[id]
		if !ok || fn.schema != sFunction {
			return nil, fmt.Errorf("go_projection.invalid_function_member:%s", id)
		}
		name, err := text(fn, "00000000000000000000000000009110")
		if err != nil || !identifier(name) {
			return nil, fmt.Errorf("go_projection.invalid_function_name")
		}
		if owner == p.Name {
			if localNames[name] {
				return nil, fmt.Errorf("go_projection.duplicate_local_function:%s", name)
			}
			localNames[name] = true
			local[id] = true
			functions[id] = name
		} else {
			functions[id] = aliases[owner] + "." + name
		}
	}
	imports := map[string]string{}
	for id := range local {
		for callee := range calledFunctions(graph, id) {
			owner, ok := owners[callee]
			if !ok {
				return nil, fmt.Errorf("go_projection.call_owner_missing:%s", callee)
			}
			if owner != p.Name {
				imports[owner] = aliases[owner]
			}
		}
	}
	if graphHasSchemaInClosure(graph, local, sBytesEqual) {
		imports["bytes"] = "bytes"
	}
	if graphHasSchemaInClosure(graph, local, sCollectionUpdate) || graphHasSchemaInClosure(graph, local, sSliceRemove) {
		imports["slices"] = "slices"
	}
	if graphHasSchemaInClosure(graph, local, sMapRemove) {
		imports["maps"] = "maps"
	}
	if graphHasSchemaInClosure(graph, local, sEffectInvoke) {
		imports["log"] = "log"
	}
	var out strings.Builder
	fmt.Fprintf(&out, "package %s\n\n", packageName)
	paths := make([]string, 0, len(imports))
	for x := range imports {
		paths = append(paths, x)
	}
	sort.Strings(paths)
	if len(paths) > 0 {
		out.WriteString("import (\n")
		for _, x := range paths {
			alias := imports[x]
			if alias != path.Base(x) {
				fmt.Fprintf(&out, "\t%s %q\n", alias, x)
			} else {
				fmt.Fprintf(&out, "\t%q\n", x)
			}
		}
		out.WriteString(")\n\n")
	}
	records, err := collectRecords(graph)
	if err != nil {
		return nil, err
	}
	if len(records) != 0 {
		return nil, fmt.Errorf("go_projection.multi_package_records_unsupported")
	}
	ids := make([]string, 0, len(local))
	for id := range local {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		fn := graph[id]
		paramIDs, er := refs(fn, "00000000000000000000000000009111")
		if er != nil {
			return nil, er
		}
		params := map[string]string{}
		declarations := []string{}
		for _, pid := range paramIDs {
			q := graph[pid]
			name, e := text(q, "00000000000000000000000000009120")
			if e != nil || !identifier(name) {
				return nil, fmt.Errorf("go_projection.invalid_parameter")
			}
			tid, e := ref(q, "00000000000000000000000000009121")
			if e != nil {
				return nil, e
			}
			typ, e := typeName(graph, tid)
			if e != nil {
				return nil, e
			}
			params[pid] = name
			declarations = append(declarations, name+" "+typ)
		}
		resultID, e := ref(fn, "00000000000000000000000000009112")
		if e != nil {
			return nil, e
		}
		result, e := typeName(graph, resultID)
		if e != nil {
			return nil, e
		}
		bodyID, e := ref(fn, "00000000000000000000000000009113")
		if e != nil {
			return nil, e
		}
		body, e := projectBlock(bodyID, context{graph: graph, functions: functions, parameters: params, records: records, locals: map[string]string{}, iterations: map[string]string{}, variants: map[string]string{}, methods: map[string]string{}, receivers: map[string]string{}, captures: map[string]string{}, transitions: map[string]string{}})
		if e != nil {
			return nil, e
		}
		fmt.Fprintf(&out, "//seme:id %s\nfunc %s(%s) %s {\n%s\n}\n\n", id, functions[id], strings.Join(declarations, ", "), result, body)
	}
	formatted, err := format.Source([]byte(out.String()))
	if err != nil {
		return nil, fmt.Errorf("go_projection.format:%w", err)
	}
	return formatted, nil
}

func calledFunctions(graph map[string]entity, root string) map[string]bool {
	out := map[string]bool{}
	seen := map[string]bool{}
	q := []string{root}
	for len(q) > 0 {
		x := q[0]
		q = q[1:]
		if seen[x] {
			continue
		}
		seen[x] = true
		e, ok := graph[x]
		if !ok {
			continue
		}
		if x != root && e.schema == sFunction {
			continue
		}
		if e.schema == sCall {
			if c, err := ref(e, "00000000000000000000000000009600"); err == nil {
				out[c] = true
			}
		}
		q = append(q, entityReferences(e, graph)...)
	}
	return out
}
func graphHasSchemaInClosure(graph map[string]entity, roots map[string]bool, schema string) bool {
	for root := range roots {
		seen := map[string]bool{}
		q := []string{root}
		for len(q) > 0 {
			x := q[0]
			q = q[1:]
			if seen[x] {
				continue
			}
			seen[x] = true
			e, ok := graph[x]
			if !ok {
				continue
			}
			if e.schema == schema {
				return true
			}
			q = append(q, entityReferences(e, graph)...)
		}
	}
	return false
}

func entityReferences(e entity, graph map[string]entity) []string {
	var out []string
	for _, values := range e.fields {
		for _, raw := range values {
			fields := strings.Fields(raw)
			for i := 0; i+1 < len(fields); i++ {
				if fields[i] == "rf" || fields[i] == "ho" {
					if _, ok := graph[fields[i+1]]; ok {
						out = append(out, fields[i+1])
					}
				}
			}
		}
	}
	return out
}
