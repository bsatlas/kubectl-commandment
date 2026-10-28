package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestGetResourceNameCompletionUsesSelectedKubernetesAPI(t *testing.T) {
	t.Parallel()

	server := getTestAPIServer(t)
	path := writeGetTestKubeconfig(t, server.URL)
	app := compileGetTestApplication(t)

	completion, err := app.Complete(t.Context(), cli.CompletionRequest{
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
		t.Fatalf("Complete() error = %v", err)
	}
	if !getCompletionContains(completion, "pod-a") {
		t.Fatalf("completion items = %#v, want pod-a", completion.Items)
	}
}
func TestGetRawUsesSelectedKubeconfig(t *testing.T) {
	t.Parallel()

	server := getTestAPIServer(t)
	path := writeGetTestKubeconfig(t, server.URL)
	app := compileGetTestApplication(t)

	var stdout bytes.Buffer
	err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"get", "--raw", "/healthz", "--kubeconfig", path},
		Stdout: &stdout,
		Stderr: io.Discard,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := stdout.String(), "ok\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestGetPolicyRejectsRawWithResourceBeforeExecution(t *testing.T) {
	t.Parallel()

	app := compileGetTestApplication(t)
	err := app.Run(t.Context(), cli.RunRequest{
		Args:   []string{"get", "pods", "--raw", "/api"},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if err == nil {
		t.Fatal("Run() error = nil, want policy rejection")
	}
}

func compileGetTestApplication(t *testing.T) cli.Application {
	t.Helper()

	definition := cli.Definition[rootConfig]{
		Name: "kubectl",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderInterspersed)
			kubeconfig := root.Values.String(&root.Config.Kubeconfig)
			root.Values.Default(kubeconfig, "")
			root.Flags.ValueFor(kubeconfig).Placement(cli.FlagPlacementSubtree).Long("kubeconfig")
			client := root.Values.Derive(&root.Config.Client, cli.Value.ComputedBy(func(
				_ context.Context,
				_ *rootConfig,
				dependencies cli.Dependencies,
			) (clientConfig, error) {
				path, ok := dependencies.Get(kubeconfig)
				if !ok {
					return clientConfig{}, fmt.Errorf("kubeconfig dependency is unavailable")
				}
				client := clientConfig{
					Kubeconfig: path,
				}
				return client, nil
			}).From(kubeconfig))
			root.Subcommands.Command(getCommand(client))
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return app
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
			body:        `{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"v1","resources":[{"name":"pods","singularName":"pod","namespaced":true,"kind":"Pod","verbs":["get","list"],"shortNames":["po"]}]}`,
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

func getCompletionContains(completion cli.Completion, value string) bool {
	for _, item := range completion.Items {
		if item.Value == value {
			return true
		}
	}
	return false
}
