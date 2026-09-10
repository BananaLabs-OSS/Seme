package gopackageadapter

import (
	"crypto/sha256"
	"fmt"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprovider"
	"seme.local/reference/packagedetail"
	"seme.local/reference/sourceinventory"
	"seme.local/reference/wire"
)

var (
	unitSchema     = mustID("e015")
	classSchema    = mustID("e013")
	pathField      = mustID("e150")
	digestField    = mustID("e151")
	sizeField      = mustID("e152")
	classField     = mustID("e153")
	classCodeField = mustID("e130")
)

// EvidenceFrom validates the detached inventory against its semantic project,
// then reconciles its tracked units with the complete in-memory provider
// snapshot. Raw source bytes are used only to verify digest and size.
func EvidenceFrom(snapshot goprovider.DocumentSnapshot, resolution goprovider.ResolutionManifest, contract contractcatalog.Contract, project, inventory []byte) (Evidence, error) {
	if err := sourceinventory.Validate(contract, project, inventory); err != nil {
		return Evidence{}, fmt.Errorf("go_package_adapter.inventory:%w", err)
	}
	return evidence(snapshot, resolution, project, inventory)
}

func evidence(snapshot goprovider.DocumentSnapshot, resolution goprovider.ResolutionManifest, project, inventory []byte) (Evidence, error) {
	envelope, err := wire.Decode(inventory)
	if err != nil {
		return Evidence{}, err
	}
	projectEnvelope, err := wire.Decode(project)
	if err != nil {
		return Evidence{}, err
	}
	tracked := map[string]packagedetail.Source{}
	for id, entity := range envelope.Entities {
		if entity.Schema != unitSchema {
			continue
		}
		cv := entity.Fields[classField]
		class := envelope.Entities[cv.Reference]
		if cv.Tag != 6 || class.Schema != classSchema || class.Fields[classCodeField].Tag != 3 {
			return Evidence{}, fmt.Errorf("go_package_adapter.inventory_class")
		}
		if class.Fields[classCodeField].Unsigned != 0 {
			continue
		}
		pv, dv, sv := entity.Fields[pathField], entity.Fields[digestField], entity.Fields[sizeField]
		if pv.Tag != 5 || dv.Tag != 5 || len(dv.Bytes) != sha256.Size || sv.Tag != 3 {
			return Evidence{}, fmt.Errorf("go_package_adapter.inventory_unit")
		}
		path := string(pv.Bytes)
		var digest [32]byte
		copy(digest[:], dv.Bytes)
		if _, ok := tracked[path]; ok {
			return Evidence{}, fmt.Errorf("go_package_adapter.inventory_path_duplicate")
		}
		tracked[path] = packagedetail.Source{Identity: id.String(), Path: path, ContentDigest: digest, ByteSize: sv.Unsigned}
	}
	wanted := map[string]bool{}
	for _, p := range resolution.Packages {
		for _, file := range p.Files {
			if wanted[file] {
				return Evidence{}, fmt.Errorf("go_package_adapter.resolution_source_duplicate")
			}
			wanted[file] = true
		}
	}
	if len(wanted) != len(snapshot.Files) || len(tracked) != len(snapshot.Files) {
		return Evidence{}, fmt.Errorf("go_package_adapter.source_set")
	}
	e := Evidence{Sources: map[string]packagedetail.Source{}, Origins: map[Key]packagedetail.Origin{}, ConsumedImports: map[Key]string{}}
	for path, data := range snapshot.Files {
		if !wanted[path] {
			return Evidence{}, fmt.Errorf("go_package_adapter.snapshot_extra")
		}
		source, ok := tracked[path]
		if !ok {
			return Evidence{}, fmt.Errorf("go_package_adapter.inventory_missing")
		}
		digest := sha256.Sum256([]byte(data))
		if digest != source.ContentDigest || uint64(len(data)) != source.ByteSize {
			return Evidence{}, fmt.Errorf("go_package_adapter.source_drift")
		}
		e.Sources[path] = source
	}
	for _, p := range resolution.Packages {
		for _, d := range p.Declarations {
			if err := addOrigin(e, d.Location); err != nil {
				return Evidence{}, err
			}
		}
		for _, im := range p.Imports {
			if err := addOrigin(e, im.Location); err != nil {
				return Evidence{}, err
			}
			if im.Consumed {
				if im.Local || im.Realization == "" || !realizationPresent(projectEnvelope, im.Realization) {
					return Evidence{}, fmt.Errorf("go_package_adapter.consumed_realization:%s", im.Path)
				}
				k := key(im.Location)
				if _, exists := e.ConsumedImports[k]; exists {
					return Evidence{}, fmt.Errorf("go_package_adapter.consumed_duplicate")
				}
				e.ConsumedImports[k] = im.Realization
			} else if im.Realization != "" {
				return Evidence{}, fmt.Errorf("go_package_adapter.realization_without_consumption")
			}
		}
	}
	return e, nil
}

// EvidenceFromV8 derives the identical neutral provenance model from a source
// inventory authenticated directly by Project v8.
func EvidenceFromV8(snapshot goprovider.DocumentSnapshot, resolution goprovider.ResolutionManifest, contract contractcatalog.Contract, project, inventory []byte) (Evidence, error) {
	if err := sourceinventory.ValidateV8(contract, project, inventory); err != nil {
		return Evidence{}, fmt.Errorf("go_package_adapter.inventory:%w", err)
	}
	return evidence(snapshot, resolution, project, inventory)
}

func realizationPresent(e wire.Envelope, realization string) bool {
	wantSchema := wire.ID{}
	switch realization {
	case "go-consumed:bytes:Equal":
		wantSchema = mustID("a065")
	case "go-consumed:maps:Clone":
		wantSchema = mustID("a043")
	case "go-consumed:slices:Clone,Replace":
		wantSchema = mustID("90fc")
	case "go-consumed:log:Print":
		for _, q := range e.Entities {
			if q.Schema == mustID("15") && string(q.Fields[mustID("150")].Bytes) == "observability.log" {
				return true
			}
		}
		return false
	default:
		return false
	}
	for _, q := range e.Entities {
		if q.Schema == wantSchema {
			return true
		}
	}
	return false
}
func addOrigin(e Evidence, l goprovider.ProjectLocation) error {
	s, ok := e.Sources[l.File]
	if !ok {
		return fmt.Errorf("go_package_adapter.origin_source")
	}
	if l.ByteStart < 0 || l.ByteEnd <= l.ByteStart || l.EndLine <= 0 || l.EndColumn <= 0 || uint64(l.ByteEnd) > s.ByteSize {
		return fmt.Errorf("go_package_adapter.origin_span")
	}
	k := key(l)
	origin := packagedetail.Origin{SourceIdentity: s.Identity, Path: s.Path, ContentDigest: s.ContentDigest, ByteStart: uint64(l.ByteStart), ByteEnd: uint64(l.ByteEnd), StartLine: uint32(l.Line), StartColumn: uint32(l.Column), EndLine: uint32(l.EndLine), EndColumn: uint32(l.EndColumn)}
	if prior, exists := e.Origins[k]; exists {
		// Several concrete generic realizations may truthfully originate at the
		// same generic TypeSpec. Only byte-identical provenance may be shared.
		if prior != origin {
			return fmt.Errorf("go_package_adapter.origin_duplicate")
		}
		return nil
	}
	e.Origins[k] = origin
	return nil
}
func mustID(short string) wire.ID {
	for len(short) < 32 {
		short = "0" + short
	}
	x, err := wire.ParseID(short)
	if err != nil {
		panic(err)
	}
	return x
}
