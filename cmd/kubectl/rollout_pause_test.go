package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const rolloutPauseTestTimeout = 5 * time.Second

func TestRolloutPauseDeclaresDeploymentCompletionAndFieldManagerDefault(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"rollout", "pause", "deployment/frontend"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		clitest.Command("rollout", "pause").Flag("--field-manager").Value().Text("kubectl-rollout"),
	)

	completion := cli.CompletionRequest{
		Args:   []string{"rollout", "pause", "d"},
		Cursor: cli.CompletionCursor{Argument: 2, Byte: 1},
	}
	clitest.Complete(t.Context(), app, completion).Require(t,
		clitest.CompletionItem("deployment").Present(),
		clitest.CompletionItem("daemonset").Absent(),
	)
}

func TestRolloutPauseUsesLastRepeatedScalarValue(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{
			"rollout", "pause", "deployment/frontend",
			"--field-manager=old", "--field-manager=new",
			"--selector=app=old", "--selector=app=new",
		},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.Command("rollout", "pause").Flag("--field-manager").Value().Text("new"),
		clitest.Command("rollout", "pause").Flag("--selector").Value().Text("app=new"),
	)
}

func TestRolloutPauseReadsFilenameDashFromInvocationStdin(t *testing.T) {
	t.Parallel()

	server, patchBodies := rolloutPauseTestAPIServer(t, false)
	kubeconfig := writeRolloutPauseTestKubeconfig(t, server.URL)
	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  namespace: test
spec:
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - name: frontend
        image: nginx
`
	request := cli.RunRequest{
		Args:   []string{"--kubeconfig", kubeconfig, "rollout", "pause", "-f", "-"},
		Stdin:  strings.NewReader(manifest),
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	select {
	case body := <-patchBodies:
		if !bytes.Contains(body, []byte(`"paused":true`)) {
			t.Fatalf("patch body = %s, want paused=true", body)
		}
	default:
		t.Fatal("rollout pause did not send a patch")
	}
}

func TestRolloutPauseCancelsInFlightRequestWithInvocation(t *testing.T) {
	t.Parallel()

	server, patchBodies := rolloutPauseTestAPIServer(t, true)
	kubeconfig := writeRolloutPauseTestKubeconfig(t, server.URL)
	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() {
		request := cli.RunRequest{
			Args:   []string{"--kubeconfig", kubeconfig, "rollout", "pause", "deployment/frontend"},
			Stdin:  strings.NewReader(""),
			Stdout: io.Discard,
			Stderr: io.Discard,
		}
		result <- app.Run(ctx, request)
	}()

	select {
	case <-patchBodies:
		cancel()
	case <-time.After(rolloutPauseTestTimeout):
		cancel()
		t.Fatal("rollout pause did not begin its patch request")
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run() error = %v, want context canceled", err)
		}
	case <-time.After(rolloutPauseTestTimeout):
		t.Fatal("rollout pause did not return after invocation cancellation")
	}
}

func rolloutPauseTestAPIServer(t *testing.T, blockPatch bool) (*httptest.Server, <-chan []byte) {
	t.Helper()

	patchBodies := make(chan []byte, 1)
	deployment := `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"frontend","namespace":"test"},"spec":{"paused":false,"selector":{"matchLabels":{"app":"frontend"}},"template":{"metadata":{"labels":{"app":"frontend"}},"spec":{"containers":[{"name":"frontend","image":"nginx"}]}}}}`
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api":
			writeRolloutPauseTestResponse(t, writer, `{"kind":"APIVersions","apiVersion":"v1","versions":["v1"],"serverAddressByClientCIDRs":[]}`)
		case "/apis":
			writeRolloutPauseTestResponse(t, writer, `{"kind":"APIGroupList","apiVersion":"v1","groups":[{"name":"apps","versions":[{"groupVersion":"apps/v1","version":"v1"}],"preferredVersion":{"groupVersion":"apps/v1","version":"v1"}}]}`)
		case "/apis/apps/v1":
			writeRolloutPauseTestResponse(t, writer, `{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"apps/v1","resources":[{"name":"deployments","singularName":"deployment","namespaced":true,"kind":"Deployment","verbs":["get","list","patch"]}]}`)
		case "/apis/apps/v1/namespaces/test/deployments/frontend":
			switch request.Method {
			case http.MethodGet:
				writeRolloutPauseTestResponse(t, writer, deployment)
			case http.MethodPatch:
				body, err := io.ReadAll(request.Body)
				if err != nil {
					http.Error(writer, "read patch body: "+err.Error(), http.StatusBadRequest)
					return
				}
				patchBodies <- body
				if blockPatch {
					select {
					case <-request.Context().Done():
					case <-time.After(rolloutPauseTestTimeout):
					}
					return
				}
				writeRolloutPauseTestResponse(t, writer, strings.Replace(deployment, `"paused":false`, `"paused":true`, 1))
			default:
				http.Error(writer, "unexpected method: "+request.Method, http.StatusMethodNotAllowed)
			}
		default:
			http.Error(writer, "unexpected path: "+request.URL.Path, http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server, patchBodies
}

func writeRolloutPauseTestResponse(t *testing.T, writer io.Writer, body string) {
	t.Helper()

	if _, err := io.WriteString(writer, body); err != nil {
		t.Errorf("write test response: %v", err)
	}
}

func writeRolloutPauseTestKubeconfig(t *testing.T, server string) string {
	t.Helper()

	config := clientcmdapi.NewConfig()
	config.CurrentContext = "fixture"
	config.Clusters["fixture"] = &clientcmdapi.Cluster{Server: server}
	config.Contexts["fixture"] = &clientcmdapi.Context{Cluster: "fixture", Namespace: "test"}
	path := filepath.Join(t.TempDir(), "config")
	if err := clientcmd.WriteToFile(*config, path); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	return path
}
