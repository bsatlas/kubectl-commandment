package kubectlcommandment_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	kubectlcommandment "github.com/abstractionsinc/kubectl-commandment"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestGetReadsRealKubernetesAPIThroughResourceBuilder(t *testing.T) {
	server := newKubernetesAPIServer(t)
	path := writeServerKubeconfig(t, server.URL, "test")
	app := compileApplication(t)

	var stdout bytes.Buffer
	err := app.Run(context.Background(), runRequest(
		[]string{"--kubeconfig", path, "get", "pods", "-o", "json"},
		&stdout,
	))
	if err != nil {
		t.Fatalf("run get pods: %v", err)
	}

	var output struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
		} `json:"items"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode get json: %v\noutput:\n%s", err, stdout.String())
	}
	if len(output.Items) != 1 || output.Items[0].Metadata.Name != "pod-a" {
		t.Fatalf("items = %#v, want pod-a", output.Items)
	}
}

func TestGetAcceptsParameterizedOutputFormat(t *testing.T) {
	server := newKubernetesAPIServer(t)
	path := writeServerKubeconfig(t, server.URL, "test")
	app := compileApplication(t)

	var stdout bytes.Buffer
	err := app.Run(context.Background(), runRequest(
		[]string{"get", "pods", "--kubeconfig", path, "-o", "custom-columns=NAME:.metadata.name"},
		&stdout,
	))
	if err != nil {
		t.Fatalf("run parameterized output: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "pod-a") {
		t.Fatalf("stdout = %q, want pod-a", got)
	}
}

func TestGetRawUsesSelectedKubeconfigRESTClient(t *testing.T) {
	server := newKubernetesAPIServer(t)
	path := writeServerKubeconfig(t, server.URL, "test")
	app := compileApplication(t)

	var stdout bytes.Buffer
	err := app.Run(context.Background(), runRequest(
		[]string{"get", "--raw", "/healthz", "--kubeconfig", path},
		&stdout,
	))
	if err != nil {
		t.Fatalf("run raw get: %v", err)
	}
	if got, want := stdout.String(), "ok\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestGetResourceCompletionUsesSelectedKubernetesDiscovery(t *testing.T) {
	server := newKubernetesAPIServer(t)
	path := writeServerKubeconfig(t, server.URL, "test")
	app := compileApplication(t)

	completion, err := app.Complete(context.Background(), cli.CompletionRequest{
		Args: []string{"--kubeconfig", path, "get", "po"},
		Cursor: cli.CompletionCursor{
			Argument: 3,
			Byte:     2,
		},
	})
	if err != nil {
		t.Fatalf("complete resource type: %v", err)
	}
	if !completionHasValue(completion, "pods") {
		t.Fatalf("completion values = %v, want pods", completionValues(completion))
	}
}

func TestGetResourceNameCompletionUsesSelectedKubernetesAPI(t *testing.T) {
	server := newKubernetesAPIServer(t)
	path := writeServerKubeconfig(t, server.URL, "test")
	app := compileApplication(t)

	completion, err := app.Complete(context.Background(), cli.CompletionRequest{
		Args: []string{"--kubeconfig", path, "get", "pods", "pod"},
		Cursor: cli.CompletionCursor{
			Argument: 4,
			Byte:     3,
		},
	})
	if err != nil {
		t.Logf("completion on error: %#v", completion)
		t.Logf("completion cause: %v", errors.Unwrap(err))
		for i, diagnostic := range completion.Diagnostics {
			t.Logf("diagnostic[%d]: %#v", i, diagnostic)
		}
		t.Fatalf("complete resource name: %v", err)
	}
	if !completionHasValue(completion, "pod-a") {
		t.Fatalf("completion values = %v, want pod-a", completionValues(completion))
	}
}

func TestGetPolicyRejectsSelectedInvalidCombinationsBeforeExecution(t *testing.T) {
	app := compileApplication(t)
	cases := []struct {
		name string
		args []string
	}{
		{name: "raw with resources", args: []string{"get", "pods", "--raw", "/api"}},
		{name: "raw with watch", args: []string{"get", "--raw", "/api", "--watch"}},
		{name: "raw with selector", args: []string{"get", "--raw", "/api", "--selector", "app=test"}},
		{name: "raw with output", args: []string{"get", "--raw", "/api", "-o", "json"}},
		{name: "malformed raw", args: []string{"get", "--raw", "://bad"}},
		{name: "show labels with json", args: []string{"get", "pods", "--show-labels", "-o", "json"}},
		{name: "label columns with custom columns", args: []string{"get", "pods", "-L", "app", "-o", "custom-columns=NAME:.metadata.name"}},
		{name: "watch events without watch", args: []string{"get", "pods", "--output-watch-events"}},
		{name: "filename and kustomize", args: []string{"get", "-f", "a.yaml", "-k", "dir"}},
		{name: "recursive and kustomize", args: []string{"get", "-R", "-k", "dir"}},
		{name: "invalid output", args: []string{"get", "pods", "-o", "nonsense"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			if err := app.Run(context.Background(), runRequest(tc.args, &stdout)); err == nil {
				t.Fatalf("Run(%v) error = nil, want rejection", tc.args)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
		})
	}
}

func compileApplication(t *testing.T) cli.Application {
	t.Helper()
	app, err := kubectlcommandment.Application()
	if err != nil {
		t.Fatalf("compile application: %v", err)
	}
	return app
}

func completionHasValue(completion cli.Completion, value string) bool {
	for _, item := range completion.Items {
		if item.Value == value {
			return true
		}
	}
	return false
}

func completionValues(completion cli.Completion) []string {
	values := make([]string, 0, len(completion.Items))
	for _, item := range completion.Items {
		values = append(values, item.Value)
	}
	return values
}

func writeServerKubeconfig(t *testing.T, server, namespace string) string {
	t.Helper()
	config := clientcmdapi.NewConfig()
	config.CurrentContext = "fixture"
	config.Clusters["fixture"] = &clientcmdapi.Cluster{Server: server}
	config.Contexts["fixture"] = &clientcmdapi.Context{
		Cluster:   "fixture",
		Namespace: namespace,
	}
	path := filepath.Join(t.TempDir(), "config")
	if err := clientcmd.WriteToFile(*config, path); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	return path
}

func newKubernetesAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Logf("fixture request: %s %s", request.Method, request.URL.String())
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/healthz":
			writer.Header().Set("Content-Type", "text/plain")
			_, _ = writer.Write([]byte("ok\n"))
		case "/api":
			_, _ = writer.Write([]byte(`{"kind":"APIVersions","apiVersion":"v1","versions":["v1"],"serverAddressByClientCIDRs":[]}`))
		case "/apis":
			_, _ = writer.Write([]byte(`{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`))
		case "/api/v1":
			_, _ = writer.Write([]byte(`{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"v1","resources":[{"name":"pods","singularName":"pod","namespaced":true,"kind":"Pod","verbs":["get","list"],"shortNames":["po"]}]}`))
		case "/api/v1/namespaces/test/pods":
			_, _ = writer.Write([]byte(`{"kind":"PodList","apiVersion":"v1","metadata":{"resourceVersion":"1"},"items":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"pod-a","namespace":"test"}}]}`))
		case "/api/v1/namespaces/test/pods/pod-a":
			_, _ = writer.Write([]byte(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"pod-a","namespace":"test"}}`))
		default:
			http.Error(writer, "unexpected path: "+request.URL.Path, http.StatusNotFound)
		}
	})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}
