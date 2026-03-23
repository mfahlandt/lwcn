package cncf

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/mfahlandt/lwcn/internal/models"
)

const (
	// LandscapeURL is the CNCF landscape page that embeds all project data
	LandscapeURL = "https://landscape.cncf.io"
)

// landscapeItem represents a single item from the embedded landscape data
type landscapeItem struct {
	Name        string `json:"name"`
	Maturity    string `json:"maturity,omitempty"`
	Category    string `json:"category"`
	Subcategory string `json:"subcategory"`
	OSS         bool   `json:"oss,omitempty"`
}

// baseDS represents the window.baseDS structure embedded in the landscape HTML
type baseDS struct {
	Items []landscapeItem `json:"items"`
}

// knownRepos maps CNCF project names to their primary GitHub owner/repo.
var knownRepos = map[string]struct{ Owner, Repo string }{
	// Graduated
	"Kubernetes":                 {"kubernetes", "kubernetes"},
	"Prometheus":                 {"prometheus", "prometheus"},
	"Envoy":                      {"envoyproxy", "envoy"},
	"CoreDNS":                    {"coredns", "coredns"},
	"containerd":                 {"containerd", "containerd"},
	"etcd":                       {"etcd-io", "etcd"},
	"Flux":                       {"fluxcd", "flux2"},
	"Argo":                       {"argoproj", "argo-cd"},
	"Helm":                       {"helm", "helm"},
	"Harbor":                     {"goharbor", "harbor"},
	"TiKV":                       {"tikv", "tikv"},
	"Vitess":                     {"vitessio", "vitess"},
	"Jaeger":                     {"jaegertracing", "jaeger"},
	"Linkerd":                    {"linkerd", "linkerd2"},
	"Fluentd":                    {"fluent", "fluentd"},
	"CRI-O":                      {"cri-o", "cri-o"},
	"Rook":                       {"rook", "rook"},
	"SPIRE":                      {"spiffe", "spire"},
	"Open Policy Agent (OPA)":    {"open-policy-agent", "opa"},
	"Cilium":                     {"cilium", "cilium"},
	"Crossplane":                 {"crossplane", "crossplane"},
	"cert-manager":               {"cert-manager", "cert-manager"},
	"Falco":                      {"falcosecurity", "falco"},
	"in-toto":                    {"in-toto", "in-toto"},
	"The Update Framework (TUF)": {"theupdateframework", "python-tuf"},
	"Kyverno":                    {"kyverno", "kyverno"},
	"CubeFS":                     {"cubefs", "cubefs"},
	"Dragonfly":                  {"dragonflyoss", "dragonfly2"},
	"KubeEdge":                   {"kubeedge", "kubeedge"},
	"SPIFFE":                     {"spiffe", "spiffe"},
	// Incubating
	"Dapr":                              {"dapr", "dapr"},
	"KEDA":                              {"kedacore", "keda"},
	"KubeVirt":                          {"kubevirt", "kubevirt"},
	"Longhorn":                          {"longhorn", "longhorn"},
	"Knative Serving":                   {"knative", "serving"},
	"Knative Eventing":                  {"knative", "eventing"},
	"Thanos":                            {"thanos-io", "thanos"},
	"Istio":                             {"istio", "istio"},
	"NATS":                              {"nats-io", "nats-server"},
	"Strimzi":                           {"strimzi", "strimzi-kafka-operator"},
	"Backstage":                         {"backstage", "backstage"},
	"Karmada":                           {"karmada-io", "karmada"},
	"Volcano":                           {"volcano-sh", "volcano"},
	"Notary Project":                    {"notaryproject", "notary"},
	"OpenFGA":                           {"openfga", "openfga"},
	"Keycloak":                          {"keycloak", "keycloak"},
	"Kubescape":                         {"kubescape", "kubescape"},
	"Cloud Custodian":                   {"cloud-custodian", "cloud-custodian"},
	"metal3-io":                         {"metal3-io", "baremetal-operator"},
	"Container Network Interface (CNI)": {"containernetworking", "cni"},
	"OpenYurt":                          {"openyurtio", "openyurt"},
	"Fluid":                             {"fluid-cloudnative", "fluid"},
	"Lima":                              {"lima-vm", "lima"},
	"KServe":                            {"kserve", "kserve"},
	// Sandbox (notable)
	"OpenTelemetry":           {"open-telemetry", "opentelemetry-collector"},
	"Fluent Bit":              {"fluent", "fluent-bit"},
	"CloudEvents":             {"cloudevents", "spec"},
	"Tekton Pipelines":        {"tektoncd", "pipeline"},
	"Chaos Mesh":              {"chaos-mesh", "chaos-mesh"},
	"Litmus":                  {"litmuschaos", "litmus"},
	"OpenCost":                {"opencost", "opencost"},
	"Kepler":                  {"sustainable-computing-io", "kepler"},
	"Inspektor Gadget":        {"inspektor-gadget", "inspektor-gadget"},
	"kube-vip":                {"kube-vip", "kube-vip"},
	"Dex":                     {"dexidp", "dex"},
	"Distribution":            {"distribution", "distribution"},
	"OpenEBS":                 {"openebs", "openebs"},
	"Kuma":                    {"kumahq", "kuma"},
	"Meshery":                 {"meshery", "meshery"},
	"Telepresence":            {"telepresenceio", "telepresence"},
	"Gateway API":             {"kubernetes-sigs", "gateway-api"},
	"external-secrets":        {"external-secrets", "external-secrets"},
	"Paralus":                 {"paralus", "paralus"},
	"Confidential Containers": {"confidential-containers", "operator"},
	"Clusternet":              {"clusternet", "clusternet"},
	"ChaosBlade":              {"chaosblade-io", "chaosblade"},
	"Keptn":                   {"keptn", "lifecycle-toolkit"},
	"KubeVela":                {"kubevela", "kubevela"},
	"OpenObserve":             {"openobserve", "openobserve"},
	"Podman Container Tools":  {"containers", "podman"},
	"Capsule":                 {"projectcapsule", "capsule"},
	"KubeArmor":               {"kubearmor", "KubeArmor"},
	"Kubewarden":              {"kubewarden", "kubewarden-controller"},
	"Antrea":                  {"antrea-io", "antrea"},
	"Submariner":              {"submariner-io", "submariner"},
	"Kube-OVN":                {"kubeovn", "kube-ovn"},
	"Spin":                    {"fermyon", "spin"},
	"Cloud Native Buildpacks": {"buildpacks", "pack"},
}

// FetchCNCFProjects fetches the list of all CNCF projects from the Landscape page
// by parsing the embedded window.baseDS JSON data.
func FetchCNCFProjects() ([]models.Repository, error) {
	log.Println("Fetching CNCF projects from Landscape...")

	resp, err := http.Get(LandscapeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CNCF landscape: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CNCF landscape returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Extract window.baseDS JSON from the HTML
	data, err := extractBaseDS(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to extract landscape data: %w", err)
	}

	log.Printf("Found %d items in CNCF Landscape", len(data.Items))

	var repos []models.Repository
	seen := make(map[string]bool)
	var unmapped []string

	for _, item := range data.Items {
		// Only include CNCF projects (items with maturity field)
		if item.Maturity == "" || item.Maturity == "archived" {
			continue
		}

		repoInfo, ok := knownRepos[item.Name]
		if !ok {
			unmapped = append(unmapped, fmt.Sprintf("%s (%s)", item.Name, item.Maturity))
			continue
		}

		key := strings.ToLower(repoInfo.Owner + "/" + repoInfo.Repo)
		if seen[key] {
			continue
		}
		seen[key] = true

		category := normalizeCategory(item.Subcategory, item.Category)

		repos = append(repos, models.Repository{
			Owner:      repoInfo.Owner,
			Repo:       repoInfo.Repo,
			Name:       item.Name,
			Category:   category,
			CNCFStatus: item.Maturity,
		})
	}

	if len(unmapped) > 0 {
		log.Printf("Note: %d CNCF projects not in known repos mapping (add to landscape.go knownRepos if needed):", len(unmapped))
		for _, name := range unmapped {
			log.Printf("  - %s", name)
		}
	}

	log.Printf("Extracted %d GitHub repositories from CNCF projects", len(repos))
	return repos, nil
}

// extractBaseDS parses the window.baseDS JSON from the landscape HTML page
func extractBaseDS(html string) (*baseDS, error) {
	re := regexp.MustCompile(`window\.baseDS\s*=\s*(\{.+?\});\s*\n`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not find window.baseDS in landscape HTML")
	}

	var data baseDS
	if err := json.Unmarshal([]byte(matches[1]), &data); err != nil {
		return nil, fmt.Errorf("failed to parse baseDS JSON: %w", err)
	}

	return &data, nil
}

// normalizeCategory converts CNCF landscape categories to simpler categories
func normalizeCategory(subcategory, category string) string {
	sub := strings.ToLower(subcategory)

	categoryMap := map[string]string{
		"container runtime":                    "container-runtime",
		"cloud native storage":                 "storage",
		"container registry":                   "registry",
		"service mesh":                         "service-mesh",
		"api gateway":                          "networking",
		"service proxy":                        "networking",
		"cloud native network":                 "networking",
		"coordination & service discovery":     "networking",
		"scheduling & orchestration":           "orchestration",
		"monitoring":                           "monitoring",
		"logging":                              "logging",
		"tracing":                              "tracing",
		"observability":                        "observability",
		"chaos engineering":                    "chaos-engineering",
		"continuous integration & delivery":    "ci-cd",
		"continuous optimization":              "ci-cd",
		"security & compliance":                "security",
		"key management":                       "security",
		"streaming & messaging":                "messaging",
		"database":                             "database",
		"application definition & image build": "build",
		"automation & configuration":           "configuration",
		"serverless":                           "serverless",
		"installable platform":                 "platform",
		"feature flagging":                     "observability",
		"ml serving":                           "ml-serving",
		"general orchestration":                "orchestration",
	}

	if mapped, ok := categoryMap[sub]; ok {
		return mapped
	}
	if sub != "" {
		return strings.ReplaceAll(strings.ToLower(sub), " ", "-")
	}
	if category != "" {
		return strings.ReplaceAll(strings.ToLower(category), " ", "-")
	}
	return "other"
}
