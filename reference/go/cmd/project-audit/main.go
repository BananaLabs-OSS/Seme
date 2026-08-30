// Command project-audit inventories ecosystem/provider coverage for a project root.
package main

import (
	"fmt"
	"os"
	"sort"

	"seme.local/reference/projectaudit"
)

func main() {
	if len(os.Args) != 3 {
		fatal(fmt.Errorf("usage: project-audit ROOT OUTPUT.json"))
	}
	report, err := projectaudit.Scan(os.Args[1])
	fatal(err)
	fatal(projectaudit.Write(os.Args[2], report))
	fmt.Printf("Projects: %d\n", report.Summary.Total)
	keys := make([]string, 0, len(report.Summary.ByLanguage))
	for language := range report.Summary.ByLanguage {
		keys = append(keys, language)
	}
	sort.Strings(keys)
	for _, language := range keys {
		fmt.Printf("  %-22s %d\n", language, report.Summary.ByLanguage[language])
	}
	fmt.Printf("Go provider ready: %d\nProvider required: %d\nMetadata only: %d\n", report.Summary.GoProviderReady, report.Summary.ProviderRequired, report.Summary.MetadataOnly)
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "project-audit:", err)
		os.Exit(65)
	}
}
