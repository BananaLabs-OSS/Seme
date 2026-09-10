package goupb03pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"seme.local/reference/configurationinstance"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/packagedetail"
	"seme.local/reference/packagev3instance"
	"seme.local/reference/projectdependencyinstance"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/projectv5instance"
	"seme.local/reference/projectv5report"
	"seme.local/reference/projectv6instance"
	"seme.local/reference/projectv7instance"
	"seme.local/reference/wire"
)

func TestPackageV3AndProjectV5InstancesRepeatValidateAndRejectTamper(t *testing.T) {
	in := fixture(t)
	base, err := Build(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	repo, _ := filepath.Abs("../../../")
	read := func(p string) []byte {
		b, e := os.ReadFile(filepath.Join(repo, p))
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	v5, err := contractcatalog.ResolveProjectContractSetV5(read("modules/execution/v35/module.seme"), read("modules/package/v3/module.seme"), read("modules/dependency/v1/module.seme"), read("modules/project/v5/module.seme"))
	if err != nil {
		t.Fatal(err)
	}
	pkgInput := packagev3instance.Inputs{Contracts: v5, PackageV2: base.Base.PackageV2}
	p3a, err := packagev3instance.Emit(pkgInput)
	if err != nil {
		t.Fatal(err)
	}
	p3b, err := packagev3instance.Emit(pkgInput)
	if err != nil || !bytes.Equal(p3a, p3b) {
		t.Fatal("nondeterministic package v3")
	}
	if err = packagev3instance.Validate(v5, base.Base.PackageV2, p3a); err != nil {
		t.Fatal(err)
	}
	untrusted := pkgInput
	untrusted.Contracts = contractcatalog.ProjectContractSetV5{}
	if out, e := packagev3instance.Emit(untrusted); e == nil || out != nil {
		t.Fatal("untrusted package contracts accepted")
	}
	v3 := projectgraphinstance.Inputs{Contracts: in.Base.Contracts.V3, ProjectV2: in.Base.Contracts.V2.Project(), Project: base.Base.ProjectV1, Inventory: base.Base.InventoryV2, PackageGraph: base.Base.PackageV2, Composed: base.Base.ProjectV3}
	v4 := projectdependencyinstance.Inputs{Contracts: in.Contracts, ProjectV3: v3, Dependency: base.DependencyV1, Composed: base.ProjectV4}
	pi := projectv5instance.Inputs{Contracts: v5, ProjectV4: v4, PackageV2: base.Base.PackageV2, PackageV3: p3a}
	p5a, err := projectv5instance.Emit(pi)
	if err != nil {
		t.Fatal(err)
	}
	p5b, err := projectv5instance.Emit(pi)
	if err != nil || !bytes.Equal(p5a, p5b) {
		t.Fatal("nondeterministic project v5")
	}
	pi.Composed = p5a
	if err = projectv5instance.Validate(pi); err != nil {
		t.Fatal(err)
	}
	reportA, err := projectv5report.Inspect(pi)
	if err != nil {
		t.Fatal(err)
	}
	reportB, err := projectv5report.Inspect(pi)
	if err != nil || !bytes.Equal(mustJSON(t, reportA), mustJSON(t, reportB)) {
		t.Fatal("unstable v5 report")
	}
	if reportA.ContractRevision != "0000000000000000000000000000e005" || reportA.Snapshot.ID == "" || reportA.Project.Project.Root == "" {
		t.Fatal("incomplete v5 report")
	}
	if reportA.Pins.Execution.Revision != "00000000000000000000000000009023" || reportA.Pins.Package.Revision != "0000000000000000000000000000b003" || reportA.Pins.Dependency.Revision != "0000000000000000000000000000f001" || reportA.Pins.Project.Revision != "0000000000000000000000000000e005" {
		t.Fatalf("wrong authenticated pins: %#v", reportA.Pins)
	}
	if len(reportA.PackageGraph.Members) == 0 {
		t.Fatal("v2 members omitted")
	}
	for _, member := range reportA.PackageGraph.Members {
		if member.Category != "function" && member.Category != "nonfunction" {
			t.Fatalf("unclassified v2 member: %#v", member)
		}
	}
	if bytes.Contains(mustJSON(t, reportA), []byte("package application")) {
		t.Fatal("source bytes leaked")
	}
	v6, err := contractcatalog.ResolveProjectContractSetV6(read("modules/foundation/v1/module.seme"), read("modules/execution/v35/module.seme"), read("modules/package/v3/module.seme"), read("modules/dependency/v1/module.seme"), read("modules/configuration/v1/module.seme"), read("modules/project/v6/module.seme"))
	if err != nil {
		t.Fatal(err)
	}
	configuration := configurationinstance.Input{Contracts: v6, ProjectV5: pi, Model: configurationinstance.Model{}}
	configA, err := configurationinstance.Emit(configuration)
	if err != nil {
		t.Fatal(err)
	}
	configB, err := configurationinstance.Emit(configuration)
	if err != nil || !bytes.Equal(configA, configB) {
		t.Fatal("nondeterministic configuration instance")
	}
	configuration.Artifact = configA
	if err = configurationinstance.Validate(configuration); err != nil {
		t.Fatal(err)
	}
	staleConfiguration, err := wire.Decode(configA)
	if err != nil {
		t.Fatal(err)
	}
	for x, entity := range staleConfiguration.Entities {
		if entity.Schema == testID("4010") {
			entity.Fields[testID("4105")] = wire.Value{Tag: 5, Bytes: make([]byte, 32)}
			staleConfiguration.Entities[x] = entity
			break
		}
	}
	staleConfiguration.Revision = configurationinstance.ArtifactRevision(staleConfiguration)
	configuration.Artifact, err = wire.Encode(staleConfiguration)
	if err != nil {
		t.Fatal(err)
	}
	if configurationinstance.Validate(configuration) == nil {
		t.Fatal("stale configuration graph revision accepted")
	}
	configuration.Artifact = configA
	p6input := projectv6instance.Inputs{Contracts: v6, ProjectV5: pi, Configuration: configuration}
	p6a, err := projectv6instance.Emit(p6input)
	if err != nil {
		t.Fatal(err)
	}
	p6b, err := projectv6instance.Emit(p6input)
	if err != nil || !bytes.Equal(p6a, p6b) {
		t.Fatal("nondeterministic project v6")
	}
	p6input.Composed = p6a
	v7, err := contractcatalog.ResolveProjectContractSetV7(read("modules/foundation/v1/module.seme"), read("modules/execution/v35/module.seme"), read("modules/package/v3/module.seme"), read("modules/dependency/v1/module.seme"), read("modules/configuration/v2/module.seme"), read("modules/project/v7/module.seme"))
	if err != nil {
		t.Fatal(err)
	}
	bound := configurationinstance.BoundInput{Contracts: v7, Base: configuration, Model: configurationinstance.BoundModel{}}
	boundA, err := configurationinstance.EmitBound(bound)
	if err != nil {
		t.Fatal(err)
	}
	boundB, err := configurationinstance.EmitBound(bound)
	if err != nil || !bytes.Equal(boundA, boundB) {
		t.Fatal("nondeterministic bound configuration")
	}
	bound.Artifact = boundA
	if err = configurationinstance.ValidateBound(bound); err != nil {
		t.Fatal(err)
	}
	staleBound, err := wire.Decode(boundA)
	if err != nil {
		t.Fatal(err)
	}
	for x, entity := range staleBound.Entities {
		if entity.Schema == testID("401f") {
			entity.Fields[testID("41f3")] = wire.Value{Tag: 5, Bytes: make([]byte, 32)}
			staleBound.Entities[x] = entity
			break
		}
	}
	staleBound.Revision = configurationinstance.BoundArtifactRevision(staleBound)
	bound.Artifact, err = wire.Encode(staleBound)
	if err != nil {
		t.Fatal(err)
	}
	if configurationinstance.ValidateBound(bound) == nil {
		t.Fatal("stale bound-configuration revision accepted")
	}
	bound.Artifact = boundA
	p7input := projectv7instance.Inputs{Contracts: v7, ProjectV6: p6input, BoundConfiguration: bound}
	p7a, err := projectv7instance.Emit(p7input)
	if err != nil {
		t.Fatal(err)
	}
	p7b, err := projectv7instance.Emit(p7input)
	if err != nil || !bytes.Equal(p7a, p7b) {
		t.Fatal("nondeterministic Project v7")
	}
	p7input.Composed = p7a
	if err = projectv7instance.Validate(p7input); err != nil {
		t.Fatal(err)
	}
	staleP7, err := wire.Decode(p7a)
	if err != nil {
		t.Fatal(err)
	}
	for x, entity := range staleP7.Entities {
		if entity.Schema == testID("e022") {
			entity.Fields[testID("e222")] = wire.Value{Tag: 5, Bytes: make([]byte, 32)}
			staleP7.Entities[x] = entity
			break
		}
	}
	staleP7.Revision = projectv7instance.ArtifactRevision(staleP7)
	p7input.Composed, err = wire.Encode(staleP7)
	if err != nil {
		t.Fatal(err)
	}
	if projectv7instance.Validate(p7input) == nil {
		t.Fatal("stale Project v7 snapshot revision accepted")
	}
	p7input.Composed = p7a
	badBound := bound
	badBound.Contracts = contractcatalog.ProjectContractSetV7{}
	if out, e := configurationinstance.EmitBound(badBound); e == nil || out != nil {
		t.Fatal("untrusted bound configuration contracts accepted")
	}
	badP7 := p7input
	badP7.Contracts = contractcatalog.ProjectContractSetV7{}
	if out, e := projectv7instance.Emit(badP7); e == nil || out != nil {
		t.Fatal("untrusted Project v7 contracts accepted")
	}
	if err = projectv6instance.Validate(p6input); err != nil {
		t.Fatal(err)
	}
	staleProject, err := wire.Decode(p6a)
	if err != nil {
		t.Fatal(err)
	}
	for x, entity := range staleProject.Entities {
		if entity.Schema == testID("e021") {
			entity.Fields[testID("e212")] = wire.Value{Tag: 5, Bytes: make([]byte, 32)}
			staleProject.Entities[x] = entity
			break
		}
	}
	staleProject.Revision = projectv6instance.ArtifactRevision(staleProject)
	p6input.Composed, err = wire.Encode(staleProject)
	if err != nil {
		t.Fatal(err)
	}
	if projectv6instance.Validate(p6input) == nil {
		t.Fatal("stale Project v6 snapshot revision accepted")
	}
	p6input.Composed = p6a
	badP6 := p6input
	badP6.Contracts = contractcatalog.ProjectContractSetV6{}
	if out, e := projectv6instance.Emit(badP6); e == nil || out != nil {
		t.Fatal("untrusted project v6 contracts accepted")
	}
	badConfiguration := configuration
	badConfiguration.Contracts = contractcatalog.ProjectContractSetV6{}
	if out, e := configurationinstance.Emit(badConfiguration); e == nil || out != nil {
		t.Fatal("untrusted configuration contracts accepted")
	}
	badPI := pi
	badPI.Contracts = contractcatalog.ProjectContractSetV5{}
	if out, e := projectv5instance.Emit(badPI); e == nil || out != nil {
		t.Fatal("untrusted project contracts accepted")
	}
	tampered, _ := wire.Decode(p3a)
	for x, q := range tampered.Entities {
		if q.Schema == testID("b029") {
			q.Fields[testID("b292")] = wire.Value{Tag: 5, Bytes: make([]byte, 32)}
			tampered.Entities[x] = q
			break
		}
	}
	bad, _ := wire.Encode(tampered)
	if packagev3instance.Validate(v5, base.Base.PackageV2, bad) == nil {
		t.Fatal("stale package revision accepted")
	}
	project, _ := wire.Decode(p5a)
	project.Entities[testID("ffffffffffffffffffffffffffffffff")] = wire.Entity{ID: testID("ffffffffffffffffffffffffffffffff"), Schema: testID("e020"), Version: 1, Fields: map[wire.ID]wire.Value{}}
	badProject, _ := wire.Encode(project)
	pi.Composed = badProject
	if projectv5instance.Validate(pi) == nil {
		t.Fatal("orphan project entity accepted")
	}
	p3a[0] ^= 1
	if err = packagev3instance.Validate(v5, base.Base.PackageV2, p3b); err != nil {
		t.Fatal("result aliasing")
	}
	// A canonical record absent from v2 function members must be owned through
	// the supplemental v3 graph and may carry a newly emitted source origin.
	baseGraph, _ := wire.Decode(base.Base.PackageV2)
	recordID := testID("77777777777777777777777777777777")
	baseGraph.Entities[recordID] = wire.Entity{ID: recordID, Schema: testID("9030"), Version: 1, Fields: map[wire.ID]wire.Value{testID("9300"): {Tag: 5, Bytes: []byte("Thing")}, testID("9301"): {Tag: 7, List: []wire.Value{}}}}
	richBase, _ := wire.Encode(baseGraph)
	var owner wire.ID
	var source wire.ID
	for x, q := range baseGraph.Entities {
		if q.Schema == testID("b021") && owner == (wire.ID{}) {
			owner = x
		}
		if q.Schema == testID("b026") && source == (wire.ID{}) {
			source = q.Fields[testID("b260")].Reference
		}
	}
	rich := packagev3instance.Inputs{Contracts: v5, PackageV2: richBase, Declarations: []packagev3instance.Declaration{{Identity: recordID.String(), OwnerDetail: owner.String(), Kind: packagev3instance.DataType, Name: "Thing", ExportName: "Thing", Visibility: packagedetail.Public, Origin: packagedetail.Origin{SourceIdentity: source.String(), Path: "model/thing.go", ContentDigest: [32]byte{1}, ByteEnd: 10, StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 10}}}}
	richOut, err := packagev3instance.Emit(rich)
	if err != nil {
		t.Fatal(err)
	}
	if err = packagev3instance.Validate(v5, richBase, richOut); err != nil {
		t.Fatal(err)
	}
	richReport, err := projectv5report.InspectPackage(v5, richBase, richOut)
	if err != nil || len(richReport.Declarations) != 1 || richReport.Declarations[0].Kind != "data-type" || richReport.Declarations[0].Owner == "" || richReport.Declarations[0].Origin.Path != "model/thing.go" {
		t.Fatalf("rich report=%#v err=%v", richReport, err)
	}
	wrongKind := rich
	wrongKind.Declarations[0].Kind = packagev3instance.BehavioralInterface
	if out, e := packagev3instance.Emit(wrongKind); e == nil || out != nil {
		t.Fatal("schema-kind mismatch accepted")
	}
	rich.Declarations[0].Kind = packagev3instance.DataType
	duplicate := rich
	duplicate.Declarations = append(duplicate.Declarations, duplicate.Declarations[0])
	if out, e := packagev3instance.Emit(duplicate); e == nil || out != nil {
		t.Fatal("duplicate ownership accepted")
	}
	badOrigin := rich
	badOrigin.Declarations[0].Origin.ByteEnd = 0
	if out, e := packagev3instance.Emit(badOrigin); e == nil || out != nil {
		t.Fatal("invalid origin accepted")
	}
	richGraph, _ := wire.Decode(richOut)
	delete(richGraph.Entities, recordID)
	for x, q := range richGraph.Entities {
		if q.Schema == testID("b028") {
			q.Fields[testID("b280")] = wire.Value{Tag: 6, Reference: recordID}
			richGraph.Entities[x] = q
		}
	}
	missing, _ := wire.Encode(richGraph)
	if packagev3instance.Validate(v5, richBase, missing) == nil {
		t.Fatal("missing canonical declaration accepted")
	}
}

func testID(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
