package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mfahlandt/lwcn/internal/cncf"
	"github.com/mfahlandt/lwcn/internal/models"
	"gopkg.in/yaml.v3"
)

func TestMergeRepositories(t *testing.T) {
	cncfRepos := []models.Repository{
		{Owner: "opentofu", Repo: "opentofu", Name: "OpenTofu", Category: "configuration", CNCFStatus: "sandbox"},
		{Owner: "podman-container-tools", Repo: "podman", Name: "Podman Container Tools", Category: "container-runtime", CNCFStatus: "graduated"},
		{Owner: "notaryproject", Repo: "notation", Name: "Notary Project", Category: "security", CNCFStatus: "incubating"},
	}
	additional := []models.Repository{
		{Owner: "Podman-Container-Tools", Repo: "Podman", Name: "Podman", Category: "container-runtime"},
		{Owner: "ossf", Repo: "scorecard", Name: "OpenSSF Scorecard", Category: "security"},
		{Owner: "OSSF", Repo: "SCORECARD", Name: "Duplicate"},
		{Owner: "sigstore", Repo: "cosign", Name: "Cosign", Category: "security"},
	}
	want := []models.Repository{cncfRepos[1], cncfRepos[2], cncfRepos[0], additional[3], additional[1]}
	merged := mergeRepositories(cncfRepos, additional)
	if !reflect.DeepEqual(merged, want) {
		t.Errorf("got %+v, want %+v", merged, want)
	}

	data, err := marshalRepositoriesYAML(models.RepositoryConfig{Repositories: merged})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), cncf.LandscapeURL) {
		t.Error("generated header does not identify the official YAML source")
	}
	var cfg models.RepositoryConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.Repositories, want) {
		t.Errorf("YAML round trip: got %+v, want %+v", cfg.Repositories, want)
	}
}
