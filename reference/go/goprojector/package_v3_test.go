package goprojector

import (
	"os"
	"testing"

	"seme.local/reference/contractcatalog"
)

func TestProjectPackagesV3RejectsUntrustedAndTamperedAtomically(t *testing.T) {
	if out, err := ProjectPackagesV3(nil, contractcatalog.ProjectContractSetV5{}, nil, nil); err == nil || out != nil {
		t.Fatalf("untrusted out=%v err=%v", out, err)
	}
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	contracts, err := contractcatalog.ResolveProjectContractSetV5(read("../../../modules/execution/v35/module.seme"), read("../../../modules/package/v3/module.seme"), read("../../../modules/dependency/v1/module.seme"), read("../../../modules/project/v5/module.seme"))
	if err != nil {
		t.Fatal(err)
	}
	if out, err := ProjectPackagesV3(nil, contracts, []byte("bad"), []byte("bad")); err == nil || out != nil {
		t.Fatalf("tamper out=%v err=%v", out, err)
	}
}
