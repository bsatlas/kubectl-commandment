package compatibility_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	legacyKubectlEnv      = "KUBECTL_LEGACY"
	commandmentKubectlEnv = "KUBECTL_COMMANDMENT"
)

type processResult struct {
	stdout []byte
	stderr []byte
	err    error
}

type resourceObservation struct {
	Kind      string
	Name      string
	Namespace string
}

func TestWorkstream1DifferentialCompatibility(t *testing.T) {
	legacy, candidate := comparisonBinaries(t)
	server := newKubernetesAPIServer(t)

	t.Run("use context", func(t *testing.T) {
		legacyConfig := writeKubeconfig(t, server.URL, "fixture", "fixture", "dev")
		candidateConfig := writeKubeconfig(t, server.URL, "fixture", "fixture", "dev")
		legacyResult := run(t, legacy, "--kubeconfig", legacyConfig, "config", "use-context", "dev")
		candidateResult := run(t, candidate, "--kubeconfig", candidateConfig, "config", "use-context", "dev")
		requireSuccess(t, "legacy", legacyResult)
		requireSuccess(t, "candidate", candidateResult)
		requireExact(t, "stdout", legacyResult.stdout, candidateResult.stdout)
		requireExact(t, "stdout", []byte("Switched to context \"dev\".\n"), candidateResult.stdout)
		if got := currentContext(t, legacyConfig); got != "dev" {
			t.Fatalf("legacy current context = %q, want dev", got)
		}
		if got := currentContext(t, candidateConfig); got != "dev" {
			t.Fatalf("candidate current context = %q, want dev", got)
		}
	})

	t.Run("current context", func(t *testing.T) {
		path := writeKubeconfig(t, server.URL, "dev", "fixture", "dev")
		legacyResult := run(t, legacy, "--kubeconfig", path, "config", "current-context")
		candidateResult := run(t, candidate, "--kubeconfig", path, "config", "current-context")
		requireSuccess(t, "legacy", legacyResult)
		requireSuccess(t, "candidate", candidateResult)
		requireExact(t, "stdout", legacyResult.stdout, candidateResult.stdout)
	})

	t.Run("raw get", func(t *testing.T) {
		path := writeKubeconfig(t, server.URL, "fixture", "fixture")
		legacyResult := run(t, legacy, "--kubeconfig", path, "get", "--raw", "/healthz")
		candidateResult := run(t, candidate, "--kubeconfig", path, "get", "--raw", "/healthz")
		requireSuccess(t, "legacy", legacyResult)
		requireSuccess(t, "candidate", candidateResult)
		requireExact(t, "stdout", legacyResult.stdout, candidateResult.stdout)
	})

	t.Run("json resource read", func(t *testing.T) {
		path := writeKubeconfig(t, server.URL, "fixture", "fixture")
		legacyResult := run(t, legacy, "--kubeconfig", path, "get", "pods", "-o", "json")
		candidateResult := run(t, candidate, "--kubeconfig", path, "get", "pods", "-o", "json")
		requireSuccess(t, "legacy", legacyResult)
		requireSuccess(t, "candidate", candidateResult)
		legacyResources := observeResources(t, legacyResult.stdout)
		candidateResources := observeResources(t, candidateResult.stdout)
		requireResourceObservations(t, legacyResources, candidateResources)
	})

	t.Run("namespace override", func(t *testing.T) {
		path := writeKubeconfig(t, server.URL, "fixture", "fixture")
		args := []string{"--kubeconfig", path, "get", "pods", "--namespace", "other", "-o", "json"}
		legacyResult := run(t, legacy, args...)
		candidateResult := run(t, candidate, args...)
		requireSuccess(t, "legacy", legacyResult)
		requireSuccess(t, "candidate", candidateResult)
		legacyResources := observeResources(t, legacyResult.stdout)
		candidateResources := observeResources(t, candidateResult.stdout)
		requireResourceObservations(t, legacyResources, candidateResources)
		if len(candidateResources) != 1 || candidateResources[0].Namespace != "other" {
			t.Fatalf("candidate namespace observation = %#v, want one resource in other", candidateResources)
		}
	})

	t.Run("context override", func(t *testing.T) {
		path := writeKubeconfig(t, server.URL, "fixture", "fixture")
		addKubeconfigContext(t, path, "other", "other")
		args := []string{"--kubeconfig", path, "get", "pods", "--context", "other", "-o", "json"}
		legacyResult := run(t, legacy, args...)
		candidateResult := run(t, candidate, args...)
		requireSuccess(t, "legacy", legacyResult)
		requireSuccess(t, "candidate", candidateResult)
		legacyResources := observeResources(t, legacyResult.stdout)
		candidateResources := observeResources(t, candidateResult.stdout)
		requireResourceObservations(t, legacyResources, candidateResources)
		if len(candidateResources) != 1 || candidateResources[0].Namespace != "other" {
			t.Fatalf("candidate context observation = %#v, want one resource in other", candidateResources)
		}
	})

	t.Run("all namespaces", func(t *testing.T) {
		path := writeKubeconfig(t, server.URL, "fixture", "fixture")
		args := []string{"--kubeconfig", path, "get", "pods", "--all-namespaces", "-o", "json"}
		legacyResult := run(t, legacy, args...)
		candidateResult := run(t, candidate, args...)
		requireSuccess(t, "legacy", legacyResult)
		requireSuccess(t, "candidate", candidateResult)
		legacyResources := observeResources(t, legacyResult.stdout)
		candidateResources := observeResources(t, candidateResult.stdout)
		requireResourceObservations(t, legacyResources, candidateResources)
		if len(candidateResources) != 2 {
			t.Fatalf("candidate all-namespace observation = %#v, want two resources", candidateResources)
		}
	})

	t.Run("chunk size int64 max", func(t *testing.T) {
		path := writeKubeconfig(t, server.URL, "fixture", "fixture")
		args := []string{"--kubeconfig", path, "get", "pods", "--chunk-size", "9223372036854775807", "-o", "json"}
		legacyResult := run(t, legacy, args...)
		candidateResult := run(t, candidate, args...)
		requireSuccess(t, "legacy", legacyResult)
		requireSuccess(t, "candidate", candidateResult)
		legacyResources := observeResources(t, legacyResult.stdout)
		candidateResources := observeResources(t, candidateResult.stdout)
		requireResourceObservations(t, legacyResources, candidateResources)
	})

	invalidCases := []struct {
		name string
		args []string
	}{
		{name: "raw with resources", args: []string{"get", "pods", "--raw", "/api"}},
		{name: "raw with watch", args: []string{"get", "--raw", "/api", "--watch"}},
		{name: "raw with selector", args: []string{"get", "--raw", "/api", "--selector", "app=test"}},
		{name: "raw with output", args: []string{"get", "--raw", "/api", "-o", "json"}},
		{name: "show labels with json", args: []string{"get", "pods", "--show-labels", "-o", "json"}},
		{name: "watch events without watch", args: []string{"get", "pods", "--output-watch-events"}},
		{name: "filename and kustomize", args: []string{"get", "-f", "a.yaml", "-k", "dir"}},
		{name: "recursive and kustomize", args: []string{"get", "-R", "-k", "dir"}},
		{name: "missing input", args: []string{"get"}},
		{name: "chunk size int64 overflow", args: []string{"get", "pods", "--chunk-size", "9223372036854775808"}},
		{name: "get suggest-for list is not alias", args: []string{"list", "pods"}},
		{name: "get suggest-for ls is not alias", args: []string{"ls", "pods"}},
	}
	for _, tc := range invalidCases {
		t.Run("reject "+tc.name, func(t *testing.T) {
			path := writeKubeconfig(t, server.URL, "fixture", "fixture")
			args := append([]string{"--kubeconfig", path}, tc.args...)
			legacyResult := run(t, legacy, args...)
			candidateResult := run(t, candidate, args...)
			if legacyResult.err == nil {
				t.Fatalf("legacy unexpectedly accepted %v", tc.args)
			}
			if candidateResult.err == nil {
				t.Fatalf("candidate unexpectedly accepted %v", tc.args)
			}
			if len(legacyResult.stdout) != 0 || len(candidateResult.stdout) != 0 {
				t.Fatalf("rejected invocation stdout: legacy=%q candidate=%q", legacyResult.stdout, candidateResult.stdout)
			}
		})
	}
}

func comparisonBinaries(t *testing.T) (string, string) {
	t.Helper()
	legacy := os.Getenv(legacyKubectlEnv)
	candidate := os.Getenv(commandmentKubectlEnv)
	if legacy == "" || candidate == "" {
		t.Skipf("set %s and %s to run differential compatibility", legacyKubectlEnv, commandmentKubectlEnv)
	}
	return legacy, candidate
}

func run(t *testing.T, binary string, args ...string) processResult {
	t.Helper()
	command := exec.CommandContext(t.Context(), binary, args...)
	stdout, err := command.Output()
	result := processResult{stdout: stdout, err: err}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.stderr = append([]byte(nil), exitErr.Stderr...)
	}
	return result
}

func requireSuccess(t *testing.T, subject string, result processResult) {
	t.Helper()
	if result.err != nil {
		t.Fatalf("%s failed: %v\nstderr: %s", subject, result.err, result.stderr)
	}
}

func requireExact(t *testing.T, subject string, expected, actual []byte) {
	t.Helper()
	if string(actual) != string(expected) {
		t.Fatalf("%s differs:\nexpected: %q\nactual:   %q", subject, expected, actual)
	}
}

func currentContext(t *testing.T, path string) string {
	t.Helper()
	config, err := clientcmd.LoadFromFile(path)
	if err != nil {
		t.Fatalf("load kubeconfig %s: %v", path, err)
	}
	return config.CurrentContext
}

func writeKubeconfig(t *testing.T, server, current string, contexts ...string) string {
	t.Helper()
	config := clientcmdapi.NewConfig()
	config.CurrentContext = current
	config.Clusters["fixture"] = &clientcmdapi.Cluster{Server: server}
	for _, name := range contexts {
		config.Contexts[name] = &clientcmdapi.Context{
			Cluster:   "fixture",
			Namespace: "test",
		}
	}
	path := filepath.Join(t.TempDir(), "config")
	if err := clientcmd.WriteToFile(*config, path); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	return path
}

func addKubeconfigContext(t *testing.T, path, name, namespace string) {
	t.Helper()
	config, err := clientcmd.LoadFromFile(path)
	if err != nil {
		t.Fatalf("load kubeconfig %s: %v", path, err)
	}
	config.Contexts[name] = &clientcmdapi.Context{
		Cluster:   "fixture",
		Namespace: namespace,
	}
	if err := clientcmd.WriteToFile(*config, path); err != nil {
		t.Fatalf("rewrite kubeconfig %s: %v", path, err)
	}
}

func newKubernetesAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	responses := map[string]struct {
		contentType string
		body        string
	}{
		"/healthz":                      {contentType: "text/plain", body: "ok\n"},
		"/api":                          {contentType: "application/json", body: `{"kind":"APIVersions","apiVersion":"v1","versions":["v1"],"serverAddressByClientCIDRs":[]}`},
		"/apis":                         {contentType: "application/json", body: `{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`},
		"/api/v1":                       {contentType: "application/json", body: `{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"v1","resources":[{"name":"pods","singularName":"pod","namespaced":true,"kind":"Pod","verbs":["get","list"],"shortNames":["po"]}]}`},
		"/api/v1/namespaces/test/pods":  {contentType: "application/json", body: `{"kind":"PodList","apiVersion":"v1","metadata":{"resourceVersion":"1"},"items":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"pod-a","namespace":"test"}}]}`},
		"/api/v1/namespaces/other/pods": {contentType: "application/json", body: `{"kind":"PodList","apiVersion":"v1","metadata":{"resourceVersion":"1"},"items":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"pod-b","namespace":"other"}}]}`},
		"/api/v1/pods":                  {contentType: "application/json", body: `{"kind":"PodList","apiVersion":"v1","metadata":{"resourceVersion":"1"},"items":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"pod-a","namespace":"test"}},{"apiVersion":"v1","kind":"Pod","metadata":{"name":"pod-b","namespace":"other"}}]}`},
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		response, ok := responses[request.URL.Path]
		if !ok {
			http.Error(writer, "unexpected path: "+request.URL.Path, http.StatusNotFound)
			return
		}
		writer.Header().Set("Content-Type", response.contentType)
		_, _ = writer.Write([]byte(response.body))
	}))
	t.Cleanup(server.Close)
	return server
}

func observeResources(t *testing.T, data []byte) []resourceObservation {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode JSON output: %v\n%s", err, data)
	}
	items, _ := document["items"].([]any)
	if len(items) == 0 && document["kind"] != nil {
		items = []any{document}
	}
	observations := make([]resourceObservation, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		metadata, _ := object["metadata"].(map[string]any)
		observations = append(observations, resourceObservation{
			Kind:      stringValue(object["kind"]),
			Name:      stringValue(metadata["name"]),
			Namespace: stringValue(metadata["namespace"]),
		})
	}
	sort.Slice(observations, func(i, j int) bool {
		if observations[i].Kind != observations[j].Kind {
			return observations[i].Kind < observations[j].Kind
		}
		if observations[i].Namespace != observations[j].Namespace {
			return observations[i].Namespace < observations[j].Namespace
		}
		return observations[i].Name < observations[j].Name
	})
	return observations
}

func requireResourceObservations(t *testing.T, expected, actual []resourceObservation) {
	t.Helper()
	if fmt.Sprint(actual) != fmt.Sprint(expected) {
		t.Fatalf("resource observations differ:\nexpected: %#v\nactual:   %#v", expected, actual)
	}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
