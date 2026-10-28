package kubectl_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestGetResourceNameCompletionUsesSelectedKubernetesAPI(t *testing.T) {
	t.Parallel()

	server := getTestAPIServer(t)
	path := writeGetTestKubeconfig(t, server.URL)
	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}

	request := cli.CompletionRequest{
		Args: []string{"--kubeconfig", path, "get", "pods", "pod"},
		Cursor: cli.CompletionCursor{
			Argument: 4,
			Byte:     3,
		},
	}

	proof := clitest.Complete(t.Context(), app, request)

	proof.Require(t, clitest.CompletionItem("pod-a").Present())
}

func TestGetRawUsesSelectedKubeconfig(t *testing.T) {
	t.Parallel()

	server := getTestAPIServer(t)
	path := writeGetTestKubeconfig(t, server.URL)
	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}

	var stdout bytes.Buffer
	request := cli.RunRequest{
		Args:   []string{"get", "--raw", "/healthz", "--kubeconfig", path},
		Stdout: &stdout,
		Stderr: io.Discard,
	}

	err := app.Run(t.Context(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := stdout.String(), "ok\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestGetPolicyRejectsRawWithResourceBeforeExecution(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"get", "pods", "--raw", "/api"},
	}

	proof := clitest.Inspect(t.Context(), app, request)

	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

func TestGetGrammarRejectsMixedResourceAndQualifiedName(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"get", "pods", "service/frontend"},
	}

	proof := clitest.Inspect(t.Context(), app, request)

	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidValue).Present()),
	)
}

func TestGetGrammarAcceptsMultipleTypesWithName(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"get", "pods,services", "frontend"},
	}

	proof := clitest.Inspect(t.Context(), app, request)

	proof.Require(t, clitest.InspectionReady())
}

func TestGetFilenameConsumesOneTokenPerOccurrence(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"get", "--filename", "pod.yaml", "pods"},
	}

	proof := clitest.Inspect(t.Context(), app, request)

	proof.Require(t,
		clitest.InspectionReady(),
		clitest.TokenAt(2).Role(cli.TokenRoleFlagValue),
		clitest.TokenAt(3).Role(cli.TokenRolePositional),
	)
}

func TestGetLabelColumnsConsumesOneTokenPerOccurrence(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"get", "pods", "--label-columns", "app", "frontend"},
	}

	proof := clitest.Inspect(t.Context(), app, request)

	proof.Require(t,
		clitest.InspectionReady(),
		clitest.TokenAt(3).Role(cli.TokenRoleFlagValue),
		clitest.TokenAt(4).Role(cli.TokenRolePositional),
	)
}

func TestGetRawAllowsExplicitFalseWatch(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"get", "--raw", "/api", "--watch=false"},
	}

	proof := clitest.Inspect(t.Context(), app, request)

	proof.Require(t, clitest.InspectionReady())
}

func TestGetCustomColumnsAcceptsTemplateFlag(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"get", "pods", "--output", "custom-columns", "--template", "NAME:.metadata.name"},
	}

	proof := clitest.Inspect(t.Context(), app, request)

	proof.Require(t, clitest.InspectionReady())
}

func TestGetInlineTemplateRejectsTemplateFlag(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"get", "pods", "--output", "go-template={{.metadata.name}}", "--template", "{{.metadata.namespace}}"},
	}

	proof := clitest.Inspect(t.Context(), app, request)

	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

func TestGetRawIgnoresFilenameKustomizeComposition(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"get", "--raw", "/api", "--filename", "pod.yaml", "--kustomize", "dir", "--recursive"},
	}

	proof := clitest.Inspect(t.Context(), app, request)

	proof.Require(t, clitest.InspectionReady())
}

func TestGetRejectsNamedResourceWithSelector(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"get", "pods", "pod-a", "--selector", "app=test"}}

	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

func TestGetRejectsQualifiedResourceWithFieldSelector(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"get", "pods/pod-a", "--field-selector", "metadata.name=pod-a"}}

	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

func TestGetAllowsBulkResourceWithSelector(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"get", "pods", "--selector", "app=test"}}

	clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
}

func TestGetRejectsBulkResourceWithSubresource(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"get", "pods", "--subresource", "status"}}

	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

func TestGetAllowsNamedResourceWithSubresource(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"get", "pods", "pod-a", "--subresource", "status"}}

	clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
}

func TestGetRejectsResourceArgumentsWithFilename(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"get", "pods", "--filename", "pod.yaml"}}

	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

func TestGetResourceTypeCompletionUsesCanonicalNames(t *testing.T) {
	t.Parallel()

	server := getTestAPIServer(t)
	path := writeGetTestKubeconfig(t, server.URL)
	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.CompletionRequest{
		Args:   []string{"--kubeconfig", path, "get", "po"},
		Cursor: cli.CompletionCursor{Argument: 3, Byte: 2},
	}

	proof := clitest.Complete(t.Context(), app, request)

	proof.Require(t,
		clitest.CompletionItem("pods").Present(),
		clitest.CompletionItem("po").Absent(),
	)
}

func TestGetResourceTypeCompletionPreservesCommaPrefix(t *testing.T) {
	t.Parallel()

	server := getTestAPIServer(t)
	path := writeGetTestKubeconfig(t, server.URL)
	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.CompletionRequest{
		Args:   []string{"--kubeconfig", path, "get", "pods,se"},
		Cursor: cli.CompletionCursor{Argument: 3, Byte: 7},
	}

	clitest.Complete(t.Context(), app, request).Require(t, clitest.CompletionItem("pods,secrets").Present())
}

func writeGetTestKubeconfig(t *testing.T, server string) string {
	t.Helper()

	config := api.NewConfig()
	config.CurrentContext = "fixture"
	config.Clusters["fixture"] = &api.Cluster{
		Server: server,
	}
	config.Contexts["fixture"] = &api.Context{
		Cluster:   "fixture",
		Namespace: "test",
	}
	path := filepath.Join(t.TempDir(), "config")
	if err := clientcmd.WriteToFile(*config, path); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	return path
}

func getTestAPIServer(t *testing.T) *httptest.Server {
	t.Helper()

	responses := map[string]struct {
		contentType string
		body        string
	}{
		"/healthz": {
			contentType: "text/plain",
			body:        "ok\n",
		},
		"/api": {
			contentType: "application/json",
			body:        `{"kind":"APIVersions","apiVersion":"v1","versions":["v1"],"serverAddressByClientCIDRs":[]}`,
		},
		"/apis": {
			contentType: "application/json",
			body:        `{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`,
		},
		"/api/v1": {
			contentType: "application/json",
			body:        `{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"v1","resources":[{"name":"pods","singularName":"pod","namespaced":true,"kind":"Pod","verbs":["get","list"],"shortNames":["po"]},{"name":"secrets","singularName":"secret","namespaced":true,"kind":"Secret","verbs":["get","list"],"shortNames":["sec"]}]}`,
		},
		"/api/v1/namespaces/test/pods": {
			contentType: "application/json",
			body:        `{"kind":"PodList","apiVersion":"v1","metadata":{"resourceVersion":"1"},"items":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"pod-a","namespace":"test"}}]}`,
		},
		"/api/v1/namespaces/test/pods/pod-a": {
			contentType: "application/json",
			body:        `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"pod-a","namespace":"test"}}`,
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Logf("fixture request: %s %s", request.Method, request.URL.String())
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
