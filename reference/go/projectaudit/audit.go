// Package projectaudit discovers project ecosystems without interpreting source.
package projectaudit

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Project struct {
	Name       string   `json:"name"`
	Path       string   `json:"path"`
	Languages  []string `json:"languages"`
	BuildFiles []string `json:"build_files"`
	Provider   string   `json:"provider"`
	Fidelity   string   `json:"fidelity"`
	Execution  string   `json:"execution"`
	Status     string   `json:"status"`
}

type Report struct {
	Version  int       `json:"version"`
	Root     string    `json:"root"`
	Projects []Project `json:"projects"`
	Summary  Summary   `json:"summary"`
}

type Summary struct {
	Total            int            `json:"total"`
	ByLanguage       map[string]int `json:"by_language"`
	GoProviderReady  int            `json:"go_provider_ready"`
	ProviderRequired int            `json:"provider_required"`
	MetadataOnly     int            `json:"metadata_only"`
}

var markers = map[string]string{
	"go.mod": "go", "Cargo.toml": "rust", "package.json": "javascript-typescript",
	"pyproject.toml": "python", "requirements.txt": "python", "pom.xml": "java",
	"build.gradle": "java-kotlin", "build.gradle.kts": "java-kotlin", "CMakeLists.txt": "c-cpp",
}

func Scan(root string) (Report, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return Report{}, err
	}
	entries, err := os.ReadDir(absolute)
	if err != nil {
		return Report{}, err
	}
	report := Report{Version: 1, Root: absolute, Summary: Summary{ByLanguage: map[string]int{}}}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		project, err := inspect(filepath.Join(absolute, entry.Name()), entry.Name())
		if err != nil {
			return Report{}, err
		}
		report.Projects = append(report.Projects, project)
		for _, language := range project.Languages {
			report.Summary.ByLanguage[language]++
		}
		switch project.Status {
		case "go-provider-ready":
			report.Summary.GoProviderReady++
		case "provider-required":
			report.Summary.ProviderRequired++
		default:
			report.Summary.MetadataOnly++
		}
	}
	sort.Slice(report.Projects, func(i, j int) bool { return report.Projects[i].Name < report.Projects[j].Name })
	report.Summary.Total = len(report.Projects)
	return report, nil
}

func inspect(path, name string) (Project, error) {
	project := Project{Name: name, Path: path, Provider: "metadata", Fidelity: "native", Execution: "external-native-island", Status: "metadata-only"}
	languages := map[string]bool{}
	err := filepath.WalkDir(path, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(path, current)
		if err != nil {
			return err
		}
		depth := 0
		if relative != "." {
			depth = len(strings.Split(filepath.ToSlash(relative), "/"))
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" || entry.Name() == "node_modules" || entry.Name() == ".seme" || depth > 2 {
				return filepath.SkipDir
			}
			return nil
		}
		language, ok := markers[entry.Name()]
		if !ok || depth > 3 {
			return nil
		}
		languages[language] = true
		project.BuildFiles = append(project.BuildFiles, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return Project{}, err
	}
	for language := range languages {
		project.Languages = append(project.Languages, language)
	}
	sort.Strings(project.Languages)
	sort.Strings(project.BuildFiles)
	if languages["go"] {
		project.Provider = "seme.go-provider.v1"
		project.Status = "go-provider-ready"
	}
	if len(languages) > 1 || !languages["go"] && len(languages) > 0 {
		project.Status = "provider-required"
		if languages["go"] {
			project.Provider = "seme.go-provider.v1+additional-provider-required"
		}
	}
	return project, nil
}

func Write(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
