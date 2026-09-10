// Command go-upb07-report emits deterministic source-free Project-v10,
// durable-state, resource, and source-presentation authority only after the
// complete UPB07 bundle independently reproduces.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/durableruntime"
	"seme.local/reference/goconfigurationmanifest"
	"seme.local/reference/godurablemanifest"
	"seme.local/reference/goupb07bundle"
	"seme.local/reference/wire"
)

type report struct {
	ProjectContract                 string                 `json:"project_contract_revision"`
	DurableContract                 string                 `json:"durable_contract_revision"`
	PresentationContract            string                 `json:"presentation_contract_revision"`
	ResourceContract                string                 `json:"resource_contract_revision"`
	ProjectRevision                 string                 `json:"project_content_revision"`
	DurableRevision                 string                 `json:"durable_content_revision"`
	SourceBoundPresentationRevision string                 `json:"source_bound_presentation_revision"`
	Durable                         durableAuthority       `json:"durable_authority"`
	Resources                       []resourceAuthority    `json:"resources"`
	NormalizedAliases               []aliasAuthority       `json:"normalized_alias_authority"`
	AliasSources                    []aliasSourceAuthority `json:"source_bound_alias_authority"`
	HostBoundaryProfileProbe        hostBoundaryProbe      `json:"host_boundary_profile_probe"`
}
type hostBoundaryProbe struct {
	Family                  string `json:"family"`
	LoadIdentity            string `json:"load_identity"`
	CompareExchangeIdentity string `json:"compare_exchange_identity"`
	LoadSequence            uint64 `json:"load_sequence"`
	CompareExchangeSequence uint64 `json:"compare_exchange_sequence"`
	Committed               bool   `json:"committed"`
}
type resourceAuthority struct {
	Identity    string `json:"identity"`
	Path        string `json:"path"`
	Destination string `json:"destination"`
	MediaType   string `json:"media_type"`
	SHA256      string `json:"sha256"`
	Size        uint64 `json:"size"`
	Kind        uint64 `json:"kind"`
}
type durableAuthority struct {
	FamilyIdentity      string               `json:"family_identity"`
	StateOwner          string               `json:"state_owner"`
	CurrentType         string               `json:"current_type"`
	PortIdentity        string               `json:"port_identity"`
	PortOwner           string               `json:"port_owner"`
	CodecIdentity       string               `json:"codec_identity"`
	KeyType             string               `json:"key_type"`
	DomainErrorType     string               `json:"domain_error_type"`
	CurrentVersion      uint64               `json:"current_version"`
	MaximumPayloadBytes uint64               `json:"maximum_payload_bytes"`
	MaximumKeyBytes     uint64               `json:"maximum_key_bytes"`
	Versions            []versionAuthority   `json:"versions"`
	Migrations          []migrationAuthority `json:"migrations"`
	Load                operationAuthority   `json:"load"`
	CompareExchange     operationAuthority   `json:"compare_exchange"`
}
type versionAuthority struct {
	Number    uint64 `json:"number"`
	Type      string `json:"type"`
	Validator string `json:"validator"`
}
type migrationAuthority struct {
	From     uint64 `json:"from"`
	To       uint64 `json:"to"`
	Function string `json:"function"`
}
type operationAuthority struct {
	Sequence   uint64 `json:"sequence"`
	Identity   string `json:"identity"`
	Capability string `json:"capability"`
}
type aliasAuthority struct {
	Owner             string   `json:"owner"`
	StableName        string   `json:"stable_name"`
	TargetType        string   `json:"target_type"`
	Visibility        uint64   `json:"visibility"`
	ReferencedImports []string `json:"referenced_imports"`
}
type aliasSourceAuthority struct {
	Owner       string `json:"owner"`
	StableName  string `json:"stable_name"`
	Path        string `json:"path"`
	SHA256      string `json:"sha256"`
	Start       uint64 `json:"byte_start"`
	End         uint64 `json:"byte_end"`
	StartLine   uint64 `json:"start_line"`
	StartColumn uint64 `json:"start_column"`
	EndLine     uint64 `json:"end_line"`
	EndColumn   uint64 `json:"end_column"`
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb07-report:", err)
		os.Exit(65)
	}
}

func run(parent context.Context, args []string, stdout io.Writer) error {
	f := flag.NewFlagSet("go-upb07-report", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	paths := map[string]*string{}
	for _, name := range []string{"bundle", "selection", "durable-selection", "foundation", "execution", "package", "dependency", "configuration", "resource", "durable-state", "presentation", "project-v8", "project-v9", "project-v10", "k0", "g1-compiler"} {
		paths[name] = new(string)
		f.StringVar(paths[name], name, "", name)
	}
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	for name, p := range paths {
		if *p == "" || !filepath.IsAbs(*p) || filepath.Clean(*p) != *p {
			return fmt.Errorf("path:%s", name)
		}
	}
	read := func(path string) ([]byte, error) {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > 64<<20 {
			return nil, fmt.Errorf("input:%s", filepath.Base(path))
		}
		return os.ReadFile(path)
	}
	inputs := map[string][]byte{}
	for _, name := range []string{"selection", "durable-selection", "foundation", "execution", "package", "dependency", "configuration", "resource", "durable-state", "presentation", "project-v8", "project-v9", "project-v10"} {
		b, err := read(*paths[name])
		if err != nil {
			return err
		}
		inputs[name] = b
	}
	v8, err := contractcatalog.ResolveProjectContractSetV8(inputs["foundation"], inputs["execution"], inputs["package"], inputs["dependency"], inputs["configuration"], inputs["project-v8"])
	if err != nil {
		return err
	}
	v9, err := contractcatalog.ResolveProjectContractSetV9(inputs["foundation"], inputs["execution"], inputs["package"], inputs["dependency"], inputs["configuration"], inputs["resource"], inputs["project-v9"])
	if err != nil {
		return err
	}
	v10, err := contractcatalog.ResolveProjectContractSetV10(inputs["foundation"], inputs["execution"], inputs["package"], inputs["dependency"], inputs["configuration"], inputs["resource"], inputs["durable-state"], inputs["presentation"], inputs["project-v9"], inputs["project-v10"])
	if err != nil {
		return err
	}
	selection, err := goconfigurationmanifest.Parse(inputs["selection"])
	if err != nil {
		return err
	}
	durableSelection, err := godurablemanifest.Parse(inputs["durable-selection"])
	if err != nil {
		return err
	}
	a, manifest, blobs, err := goupb07bundle.ReadDirectory(*paths["bundle"])
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	loaded, err := goupb07bundle.Load(ctx, goupb07bundle.Input{Contracts: v10, V9: v9, V8: v8, Artifacts: a, Manifest: manifest, Selection: selection, DurableSelection: durableSelection, Blobs: blobs, Compile: func(ctx context.Context, source []byte) ([]byte, error) {
		return compile(ctx, *paths["k0"], *paths["g1-compiler"], source)
	}})
	if err != nil {
		return err
	}
	probe, err := probeHostBoundary(loaded)
	if err != nil {
		return err
	}
	out, err := makeReport(v10, loaded)
	if err != nil {
		return err
	}
	out.HostBoundaryProfileProbe = probe
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(out)
}

type probeTransformer struct{ version uint64 }

const hostBoundaryProbeKey = "k"

var hostBoundaryProbePayload = []byte("p")

func (p probeTransformer) Prepare(*durableruntime.Payload) (durableruntime.Payload, *durableruntime.DomainError) {
	b := append([]byte(nil), hostBoundaryProbePayload...)
	return durableruntime.Payload{Version: p.version, Bytes: b, SHA256: sha256.Sum256(b)}, nil
}
func (p probeTransformer) Canonical(v durableruntime.Payload) bool {
	b := hostBoundaryProbePayload
	return v.Version == p.version && bytes.Equal(v.Bytes, b) && v.SHA256 == sha256.Sum256(b)
}

type probePort struct {
	missing durableruntime.Token
	load    *durableruntime.LoadRequest
	compare *durableruntime.CompareExchangeRequest
}

func (p *probePort) Load(r durableruntime.LoadRequest) durableruntime.LoadOutcome {
	p.load = &r
	return durableruntime.LoadOutcome{Variant: durableruntime.LoadMissing, MissingToken: append(durableruntime.Token(nil), p.missing...)}
}
func (p *probePort) CompareExchange(r durableruntime.CompareExchangeRequest) durableruntime.CompareExchangeOutcome {
	p.compare = &r
	committed := r.Payload
	return durableruntime.CompareExchangeOutcome{Variant: durableruntime.CompareExchangeSaved, Token: durableruntime.Token("opaque-saved-probe"), Committed: &committed}
}

func probeHostBoundary(loaded goupb07bundle.Result) (hostBoundaryProbe, error) {
	profile, err := durableruntime.AuthenticatedProfile(loaded.Durable)
	if err != nil {
		return hostBoundaryProbe{}, err
	}
	port := &probePort{missing: durableruntime.Token("opaque-absence-probe")}
	grants := durableruntime.Grants{profile.Load.Capability: true, profile.CompareExchange.Capability: true}
	r, err := durableruntime.ExecuteAuthenticated(loaded.Durable, grants, durableruntime.Request{Family: profile.FamilyIdentity, Key: hostBoundaryProbeKey}, port, probeTransformer{version: profile.CurrentVersion})
	if err != nil || !r.Committed || r.Failure != "" || len(r.Trace) != 2 || r.Trace[0].Sequence != profile.Load.Sequence || r.Trace[0].Operation != profile.Load.Identity || r.Trace[1].Sequence != profile.CompareExchange.Sequence || r.Trace[1].Operation != profile.CompareExchange.Identity || port.load == nil || port.compare == nil || !bytes.Equal(port.compare.ExpectedToken, port.missing) {
		return hostBoundaryProbe{}, fmt.Errorf("host_boundary_profile_probe")
	}
	return hostBoundaryProbe{Family: profile.FamilyIdentity, LoadIdentity: profile.Load.Identity, CompareExchangeIdentity: profile.CompareExchange.Identity, LoadSequence: profile.Load.Sequence, CompareExchangeSequence: profile.CompareExchange.Sequence, Committed: true}, nil
}

func compile(ctx context.Context, k0, compiler string, source []byte) ([]byte, error) {
	d, err := os.MkdirTemp("", "seme-upb07-report-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(d)
	in, out := filepath.Join(d, "input.g1"), filepath.Join(d, "output.seme")
	if err = os.WriteFile(in, source, 0600); err != nil {
		return nil, err
	}
	data, err := exec.CommandContext(ctx, k0, compiler, in, out).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compile:%w:%s", err, data)
	}
	return os.ReadFile(out)
}

func makeReport(contracts contractcatalog.ProjectContractSetV10, loaded goupb07bundle.Result) (report, error) {
	out := report{ProjectContract: contracts.Project().Pin().Revision.String(), DurableContract: contracts.DurableState().Pin().Revision.String(), PresentationContract: contracts.Presentation().Pin().Revision.String(), ResourceContract: contracts.Resource().Pin().Revision.String()}
	project, err := wire.Decode(loaded.Artifacts.ProjectV10)
	if err != nil {
		return report{}, err
	}
	root, err := one(project, "e025")
	if err != nil {
		return report{}, err
	}
	out.ProjectRevision, err = digestField(root, "e252")
	if err != nil {
		return report{}, err
	}
	durable, err := wire.Decode(loaded.Artifacts.Durable)
	if err != nil {
		return report{}, err
	}
	plan, err := one(durable, "8010")
	if err != nil {
		return report{}, err
	}
	out.DurableRevision, err = digestField(plan, "8102")
	if err != nil {
		return report{}, err
	}
	out.Durable, err = durableReport(durable)
	if err != nil {
		return report{}, err
	}
	presentation, err := wire.Decode(loaded.Artifacts.Presentation)
	if err != nil {
		return report{}, err
	}
	manifest, err := one(presentation, "1010")
	if err != nil {
		return report{}, err
	}
	out.SourceBoundPresentationRevision, err = digestField(manifest, "1102")
	if err != nil {
		return report{}, err
	}
	out.NormalizedAliases, out.AliasSources, err = aliasReports(presentation, loaded)
	if err != nil {
		return report{}, err
	}
	destinations := map[string]string{}
	for _, placement := range loaded.Base.Resource.Model.Placements {
		destinations[placement.Resource] = placement.Destination
	}
	for _, item := range loaded.Base.Resource.Model.Resources {
		out.Resources = append(out.Resources, resourceAuthority{Identity: item.Identity, Path: item.Path, Destination: destinations[item.Identity], MediaType: item.MediaType, SHA256: hex.EncodeToString(item.SHA256[:]), Size: item.Size, Kind: item.Kind})
	}
	sort.Slice(out.Resources, func(i, j int) bool { return out.Resources[i].Identity < out.Resources[j].Identity })
	return out, nil
}

func durableReport(e wire.Envelope) (durableAuthority, error) {
	family, err := one(e, "8011")
	if err != nil {
		return durableAuthority{}, err
	}
	port, err := one(e, "8015")
	if err != nil {
		return durableAuthority{}, err
	}
	owners := map[wire.ID]string{}
	for x, q := range e.Entities {
		if q.Schema == wid("b010") {
			owners[x] = string(q.Fields[wid("b100")].Bytes)
		}
	}
	out := durableAuthority{FamilyIdentity: string(family.Fields[wid("8110")].Bytes), StateOwner: owners[family.Fields[wid("8111")].Reference], PortIdentity: string(port.Fields[wid("8150")].Bytes), PortOwner: owners[port.Fields[wid("8151")].Reference], CodecIdentity: string(port.Fields[wid("8159")].Bytes), KeyType: port.Fields[wid("8157")].Reference.String(), DomainErrorType: port.Fields[wid("815b")].Reference.String(), MaximumPayloadBytes: port.Fields[wid("8158")].Unsigned, MaximumKeyBytes: port.Fields[wid("815a")].Unsigned, Versions: []versionAuthority{}, Migrations: []migrationAuthority{}}
	current := family.Fields[wid("8112")].Reference
	validators := map[wire.ID]string{}
	for _, v := range family.Fields[wid("8114")].List {
		q := e.Entities[v.Reference]
		validators[q.Fields[wid("8130")].Reference] = q.Fields[wid("8131")].Reference.String()
	}
	for _, v := range family.Fields[wid("8113")].List {
		q := e.Entities[v.Reference]
		x := versionAuthority{Number: q.Fields[wid("8121")].Unsigned, Type: q.Fields[wid("8122")].Reference.String(), Validator: validators[q.ID]}
		out.Versions = append(out.Versions, x)
		if q.ID == current {
			out.CurrentVersion = x.Number
			out.CurrentType = x.Type
		}
	}
	for _, v := range family.Fields[wid("8115")].List {
		q := e.Entities[v.Reference]
		from, to := e.Entities[q.Fields[wid("8140")].Reference], e.Entities[q.Fields[wid("8141")].Reference]
		out.Migrations = append(out.Migrations, migrationAuthority{From: from.Fields[wid("8121")].Unsigned, To: to.Fields[wid("8121")].Unsigned, Function: q.Fields[wid("8142")].Reference.String()})
	}
	op := func(effectField, capField string, seq uint64) (operationAuthority, error) {
		effect, ok := e.Entities[port.Fields[wid(effectField)].Reference]
		cap, cok := e.Entities[port.Fields[wid(capField)].Reference]
		if !ok || !cok || effect.Schema != wid("15") || cap.Schema != wid("16") || effect.Fields[wid("151")].Reference != cap.ID {
			return operationAuthority{}, fmt.Errorf("operation")
		}
		return operationAuthority{Sequence: seq, Identity: string(effect.Fields[wid("150")].Bytes), Capability: string(cap.Fields[wid("160")].Bytes)}, nil
	}
	out.Load, err = op("8155", "8153", 0)
	if err != nil {
		return durableAuthority{}, err
	}
	out.CompareExchange, err = op("8156", "8154", 1)
	if err != nil {
		return durableAuthority{}, err
	}
	if out.FamilyIdentity == "" || out.StateOwner == "" || out.PortIdentity == "" || out.PortOwner == "" || out.CurrentVersion == 0 || out.CodecIdentity == "" || out.MaximumPayloadBytes == 0 || out.MaximumKeyBytes == 0 {
		return durableAuthority{}, fmt.Errorf("durable_authority")
	}
	return out, nil
}

func aliasReports(e wire.Envelope, loaded goupb07bundle.Result) ([]aliasAuthority, []aliasSourceAuthority, error) {
	owners := map[wire.ID]string{}
	bindings := map[wire.ID]string{}
	for x, q := range e.Entities {
		if q.Schema == wid("b010") {
			owners[x] = string(q.Fields[wid("b100")].Bytes)
		}
		if q.Schema == wid("b024") {
			bindings[x] = string(q.Fields[wid("b241")].Bytes)
		}
	}
	aliases := []aliasAuthority{}
	sources := []aliasSourceAuthority{}
	for _, a := range loaded.Presentation.Model.Aliases {
		owner := owners[a.Owner]
		if owner == "" {
			return nil, nil, fmt.Errorf("alias_owner")
		}
		imports := []string{}
		for _, b := range a.ImportBindings {
			x := bindings[b]
			if x == "" {
				return nil, nil, fmt.Errorf("alias_binding")
			}
			imports = append(imports, x)
		}
		sort.Strings(imports)
		aliases = append(aliases, aliasAuthority{Owner: owner, StableName: a.Name, Visibility: a.Visibility, TargetType: a.Target.String(), ReferencedImports: imports})
		sources = append(sources, aliasSourceAuthority{Owner: owner, StableName: a.Name, Path: a.Path, SHA256: hex.EncodeToString(a.Digest[:]), Start: a.Start, End: a.End, StartLine: a.StartLine, StartColumn: a.StartColumn, EndLine: a.EndLine, EndColumn: a.EndColumn})
	}
	sort.Slice(aliases, func(i, j int) bool {
		return aliases[i].Owner < aliases[j].Owner || aliases[i].Owner == aliases[j].Owner && aliases[i].StableName < aliases[j].StableName
	})
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Owner < sources[j].Owner || sources[i].Owner == sources[j].Owner && sources[i].StableName < sources[j].StableName
	})
	return aliases, sources, nil
}
func one(e wire.Envelope, s string) (wire.Entity, error) {
	var out wire.Entity
	n := 0
	for _, q := range e.Entities {
		if q.Schema == wid(s) {
			out = q
			n++
		}
	}
	if n != 1 {
		return wire.Entity{}, fmt.Errorf("count:%s", s)
	}
	return out, nil
}
func digestField(q wire.Entity, f string) (string, error) {
	v := q.Fields[wid(f)]
	if v.Tag != 5 || len(v.Bytes) != 32 {
		return "", fmt.Errorf("digest:%s", f)
	}
	return hex.EncodeToString(v.Bytes), nil
}
func wid(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, err := wire.ParseID(s)
	if err != nil {
		panic(err)
	}
	return x
}
