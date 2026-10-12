package cncf

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/mfahlandt/lwcn/internal/models"
)

const testLandscape = `
landscape:
  - category:
    name: Provisioning
    subcategories:
      - subcategory:
        name: Automation & Configuration
        items:
          - item:
            name: OpenTofu
            project: sandbox
            repo_url: https://github.com/opentofu/opentofu
          - item:
            name: Duplicate
            project: sandbox
            repo_url: https://github.com/OpenTofu/OpenTofu.git/
          - item:
            name: Commercial product
            repo_url: https://github.com/vendor/product
          - item:
            name: Archived project
            project: archived
            repo_url: https://github.com/archive/project
          - item:
            name: Other status
            project: member
            repo_url: https://github.com/member/project
          - item:
            name: Missing repo
            project: sandbox
          - item:
            name: Non-GitHub repo
            project: sandbox
            repo_url: https://gitlab.com/group/project
      - subcategory:
        name: Security & Compliance
        items:
          - item:
            name: Notary Project
            project: incubating
            repo_url: https://github.com/notaryproject/notation
          - item:
            name: Confidential Containers
            project: sandbox
            repo_url: https://github.com/confidential-containers/confidential-containers
  - category:
    name: Observability and Analysis
    subcategories:
      - subcategory:
        name: Observability
        items:
          - item:
            name: OpenTelemetry
            project: graduated
            repo_url: https://github.com/open-telemetry/community
  - category:
    name: Empty category
    subcategories:
      - subcategory:
        name: Empty subcategory
        items:
`

func TestParseLandscape(t *testing.T) {
	repos, err := parseLandscape([]byte(testLandscape))
	if err != nil {
		t.Fatal(err)
	}
	want := []models.Repository{
		{Owner: "opentofu", Repo: "opentofu", Name: "OpenTofu", Category: "configuration", CNCFStatus: "sandbox"},
		{Owner: "notaryproject", Repo: "notation", Name: "Notary Project", Category: "security", CNCFStatus: "incubating"},
		{Owner: "confidential-containers", Repo: "operator", Name: "Confidential Containers", Category: "security", CNCFStatus: "sandbox"},
		{Owner: "open-telemetry", Repo: "opentelemetry-collector", Name: "OpenTelemetry", Category: "observability", CNCFStatus: "graduated"},
	}
	if !reflect.DeepEqual(repos, want) {
		t.Errorf("got %+v, want %+v", repos, want)
	}
}

func TestParseLandscapeRejectsInvalidData(t *testing.T) {
	for _, data := range []string{
		"landscape: [",
		"",
		"items: []",
		"landscape: []",
		`
landscape:
  - name: Security
    subcategories:
      - name: Security & Compliance
        items:
          - name: No repo
            project: sandbox
`,
		`
landscape:
  - name: Security
    subcategories:
      - name: Security & Compliance
        items:
          - project: sandbox
            repo_url: https://github.com/owner/repo
`,
	} {
		if _, err := parseLandscape([]byte(data)); err == nil {
			t.Errorf("expected error for %q", data)
		}
	}
}

func TestParseGitHubRepoURL(t *testing.T) {
	for _, raw := range []string{
		"https://github.com/owner/repo",
		"https://github.com/owner/repo.git/",
		"http://GITHUB.com/owner/repo",
	} {
		owner, repo, err := parseGitHubRepoURL(raw)
		if err != nil || owner != "owner" || repo != "repo" {
			t.Errorf("%q: got %s/%s, err=%v", raw, owner, repo, err)
		}
	}
	for _, raw := range []string{
		"", "https://gitlab.com/owner/repo", "https://github.com.evil.example/owner/repo",
		"https://github.com/owner", "https://github.com/owner/repo/tree/main",
		"https://github.com//repo", "https://github.com/owner/.git", "https://github.com/owner/..",
		"https://user@github.com/owner/repo", "ftp://github.com/owner/repo", "https://%",
	} {
		if _, _, err := parseGitHubRepoURL(raw); err == nil {
			t.Errorf("expected error for %q", raw)
		}
	}
}

func TestFetchCNCFProjects(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		fail   bool
	}{
		{"valid YAML", http.StatusOK, testLandscape, false},
		{"HTTP failure", http.StatusServiceUnavailable, "unavailable", true},
		{"invalid YAML", http.StatusOK, "landscape: [", true},
		{"empty landscape", http.StatusOK, "landscape: []", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				if _, err := w.Write([]byte(tc.body)); err != nil {
					t.Errorf("write response: %v", err)
				}
			}))
			defer server.Close()
			repos, err := fetchCNCFProjects(server.Client(), server.URL)
			if (err != nil) != tc.fail {
				t.Fatalf("err=%v, want failure=%v", err, tc.fail)
			}
			if !tc.fail && len(repos) != 4 {
				t.Errorf("got %d repos, want 4", len(repos))
			}
		})
	}
}
