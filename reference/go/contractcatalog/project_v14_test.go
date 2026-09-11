package contractcatalog

import (
	"os"
	"testing"
)

func TestResolveProjectContractSetV14AuthenticatesCompactRevision(t *testing.T) {
	read := func(path string) []byte {
		value, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	set, err := ResolveProjectContractSetV14(
		read("../../../modules/foundation/v1/module.seme"), read("../../../modules/execution/v36/module.seme"),
		read("../../../modules/package/v4/module.seme"), read("../../../modules/dependency/v1/module.seme"),
		read("../../../modules/configuration/v3/module.seme"), read("../../../modules/resource/v1/module.seme"),
		read("../../../modules/durable-state/v1/module.seme"), read("../../../modules/source-presentation/v1/module.seme"),
		read("../../../modules/ordered-transport/v1/module.seme"), read("../../../modules/controlled-effects/v1/module.seme"),
		read("../../../modules/target/v1/module.seme"), read("../../../modules/patch/v1/module.seme"),
		read("../../../modules/language-service/v1/module.seme"), read("../../../modules/project/v9/module.seme"),
		read("../../../modules/project/v10/module.seme"), read("../../../modules/project/v11/module.seme"),
		read("../../../modules/project/v12/module.seme"), read("../../../modules/project/v13/module.seme"),
		read("../../../modules/project/v14/module.seme"),
	)
	if err != nil || !set.Validated() || set.Project().Pin() != (Pin{projectModule, projectRevV14}) || set.ProjectV13().Pin() != (Pin{projectModule, projectRevV13}) || set.Patch().Pin() != (Pin{patchModule, patchRev}) || set.LanguageService().Pin() != (Pin{languageServiceModule, languageServiceRev}) {
		t.Fatal("v14 authority", err)
	}
}
