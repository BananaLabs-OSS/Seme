package goprovider

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type graphEntity struct{ id, text string }

func emitG1(module []byte, manifest Manifest) (string, error) {
	profileID := stableID("profile", manifest.Provider, manifest.Profile.Toolchain, manifest.Profile.Target)
	ingestionID := stableID("ingestion", manifest.Revision)
	var entities []graphEntity
	entities = append(entities, graphEntity{profileID, emitProfile(profileID, manifest.Profile)})
	for _, file := range manifest.Files {
		entities = append(entities, graphEntity{file.ID, entity(file.ID, "00000000000000000000000000007011", []graphField{
			bytesField(0x7110, file.Path), bytesHexField(0x7111, file.Digest),
		})})
	}
	for _, declaration := range manifest.Declarations {
		entities = append(entities, graphEntity{declaration.EvidenceID, entity(declaration.EvidenceID, "00000000000000000000000000007014", []graphField{
			bytesField(0x7140, declaration.NativeKey), bytesHexField(0x7141, declaration.Fingerprint), bytesHexField(0x7142, declaration.MatchFingerprint),
		})})
		var occurrenceIDs []string
		for _, occurrence := range declaration.Occurrences {
			occurrenceIDs = append(occurrenceIDs, occurrence.ID)
			entities = append(entities, graphEntity{occurrence.ID, entity(occurrence.ID, "00000000000000000000000000007012", []graphField{
				refField(0x7120, occurrence.File), unsignedField(0x7121, uint64(occurrence.Start)), unsignedField(0x7122, uint64(occurrence.End)), unsignedField(0x7123, occurrence.Role),
			})})
		}
		entities = append(entities, graphEntity{declaration.ID, entity(declaration.ID, "00000000000000000000000000007013", []graphField{
			bytesField(0x7130, declaration.Name), bytesField(0x7131, declaration.Qualified), bytesField(0x7132, declaration.Signature), unsignedField(0x7133, 0), refsField(0x7134, occurrenceIDs), refField(0x7135, declaration.EvidenceID),
		})})
	}
	for _, opaque := range manifest.Opaque {
		entities = append(entities, graphEntity{opaque.ID, entity(opaque.ID, "00000000000000000000000000007015", []graphField{
			refField(0x7150, opaque.File), unsignedField(0x7151, uint64(opaque.Start)), unsignedField(0x7152, uint64(opaque.End)), bytesHexField(0x7153, opaque.Digest),
		})})
	}
	var fileIDs, declarationIDs, opaqueIDs []string
	for _, file := range manifest.Files {
		fileIDs = append(fileIDs, file.ID)
	}
	for _, declaration := range manifest.Declarations {
		declarationIDs = append(declarationIDs, declaration.ID)
	}
	for _, opaque := range manifest.Opaque {
		opaqueIDs = append(opaqueIDs, opaque.ID)
	}
	entities = append(entities, graphEntity{ingestionID, entity(ingestionID, "00000000000000000000000000007016", []graphField{
		bytesHexField(0x7160, manifest.Revision), refField(0x7161, profileID), refsField(0x7162, fileIDs), refsField(0x7163, declarationIDs), refsField(0x7164, opaqueIDs),
	})})
	sort.Slice(entities, func(i, j int) bool { return entities[i].id < entities[j].id })
	seen := map[string]bool{}
	for _, e := range entities {
		if seen[e.id] {
			return "", fmt.Errorf("provider.identity_collision:%s", e.id)
		}
		seen[e.id] = true
	}
	lines := strings.Split(strings.TrimSpace(string(module)), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "rv ") {
			lines[i] = "rv " + manifest.Revision
		}
		if strings.HasPrefix(line, "ec ") {
			count, err := strconv.Atoi(strings.TrimPrefix(line, "ec "))
			if err != nil {
				return "", err
			}
			lines[i] = fmt.Sprintf("ec %d", count+len(entities))
		}
	}
	var out strings.Builder
	out.WriteString(strings.Join(lines, "\n"))
	out.WriteString("\n")
	for _, e := range entities {
		out.WriteString("\n")
		out.WriteString(e.text)
	}
	return out.String(), nil
}

func EmitProjectionG1(module []byte, report ProjectionReport, manifest Manifest) (string, error) {
	if manifest.Revision == report.BaseRevision {
		return "", fmt.Errorf("provider.projection_not_reingested")
	}
	fileByPath := map[string]NativeFile{}
	for _, file := range manifest.Files {
		fileByPath[file.Path] = file
	}
	var entities []graphEntity
	var changedIDs []string
	for _, path := range report.ChangedFiles {
		file, ok := fileByPath[path]
		if !ok {
			return "", fmt.Errorf("provider.changed_file_missing:%s", path)
		}
		changedIDs = append(changedIDs, file.ID)
		entities = append(entities, graphEntity{file.ID, entity(file.ID, "00000000000000000000000000007011", []graphField{bytesField(0x7110, file.Path), bytesHexField(0x7111, file.Digest)})})
	}
	reportID := stableID("projection", report.BaseRevision, report.ResultRevision)
	entities = append(entities, graphEntity{reportID, entity(reportID, "00000000000000000000000000007017", []graphField{bytesHexField(0x7170, report.BaseRevision), bytesHexField(0x7171, report.ResultRevision), refsField(0x7172, changedIDs), bytesField(0x7173, report.Validation), unsignedField(0x7174, uint64(report.ValidationStatus))})})
	sort.Slice(entities, func(i, j int) bool { return entities[i].id < entities[j].id })
	lines := strings.Split(strings.TrimSpace(string(module)), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "rv ") {
			lines[i] = "rv " + report.ResultRevision
		}
		if strings.HasPrefix(line, "ec ") {
			count, err := strconv.Atoi(strings.TrimPrefix(line, "ec "))
			if err != nil {
				return "", err
			}
			lines[i] = fmt.Sprintf("ec %d", count+len(entities))
		}
	}
	var out strings.Builder
	out.WriteString(strings.Join(lines, "\n"))
	out.WriteString("\n")
	for _, item := range entities {
		out.WriteString("\n")
		out.WriteString(item.text)
	}
	return out.String(), nil
}

type graphField struct {
	id    uint64
	value string
}

func entity(identity, schema string, fields []graphField) string {
	sort.Slice(fields, func(i, j int) bool { return fields[i].id < fields[j].id })
	var b strings.Builder
	fmt.Fprintf(&b, "en %s %s 1 %d\n", identity, schema, len(fields))
	for _, f := range fields {
		fmt.Fprintf(&b, "fi %032x %s\n", f.id, f.value)
	}
	return b.String()
}
func bytesField(id uint64, value string) graphField {
	return graphField{id, "by " + hex.EncodeToString([]byte(value))}
}
func bytesHexField(id uint64, value string) graphField {
	if value == "" {
		value = "-"
	}
	return graphField{id, "by " + value}
}
func unsignedField(id, value uint64) graphField   { return graphField{id, fmt.Sprintf("uu %d", value)} }
func refField(id uint64, value string) graphField { return graphField{id, "rf " + value} }
func refsField(id uint64, values []string) graphField {
	var b strings.Builder
	fmt.Fprintf(&b, "li %d", len(values))
	for _, value := range values {
		b.WriteString("\nrf ")
		b.WriteString(value)
	}
	return graphField{id, b.String()}
}
func bytesListField(id uint64, values []string) graphField {
	var b strings.Builder
	fmt.Fprintf(&b, "li %d", len(values))
	for _, value := range values {
		b.WriteString("\nby ")
		b.WriteString(hex.EncodeToString([]byte(value)))
	}
	return graphField{id, b.String()}
}
func emitProfile(identity string, profile Profile) string {
	return entity(identity, "00000000000000000000000000007010", []graphField{
		bytesField(0x7100, "1"), bytesField(0x7101, profile.Identity), bytesField(0x7102, profile.ImplementationVersion), bytesField(0x7103, profile.Language), bytesField(0x7104, profile.LanguageVersion), bytesField(0x7105, profile.Toolchain), bytesField(0x7106, profile.Target), bytesField(0x7107, profile.Environment), bytesListField(0x7108, profile.Supported), bytesListField(0x7109, profile.Exclusions),
	})
}
