package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func TestCreateFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "create requires a source",
			args: []string{"create"},
		},
		{
			name: "create source is exclusive",
			args: []string{"create", "-f", "object.yaml", "-k", "overlay"},
		},
		{
			name: "kustomize rejects recursive",
			args: []string{"create", "-k", "overlay", "-R"},
		},
		{
			name: "raw rejects edit",
			args: []string{"create", "-f", "object.yaml", "--raw=/api/v1/namespaces", "--edit"},
		},
		{
			name: "raw requires one local file",
			args: []string{"create", "-f", "one.yaml", "-f", "two.yaml", "--raw=/api/v1/namespaces"},
		},
		{
			name: "raw counts CSV filename entries",
			args: []string{"create", "-f", "one.yaml,two.yaml", "--raw=/api/v1/namespaces"},
		},
		{
			name: "raw rejects remote file",
			args: []string{"create", "--filename=https://example.invalid/object.yaml", "--raw=/api/v1/namespaces"},
		},
		{
			name: "raw rejects recursive",
			args: []string{"create", "-f", "object.yaml", "--raw=/api/v1/namespaces", "-R"},
		},
		{
			name: "raw rejects selector",
			args: []string{"create", "-f", "object.yaml", "--raw=/api/v1/namespaces", "-l", "app=web"},
		},
		{
			name: "raw rejects output",
			args: []string{"create", "-f", "object.yaml", "--raw=/api/v1/namespaces", "-o", "yaml"},
		},
		{
			name: "raw URI has kubectl syntax",
			args: []string{"create", "-f", "object.yaml", "--raw=%"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := cli.InspectionRequest{
				Args: test.args,
			}
			clitest.Inspect(t.Context(), app, request).Require(
				t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}

func TestCreateFrontendAcceptsKubectlBoundaryCases(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name         string
		args         []string
		expectations []clitest.InspectionExpectation
	}{
		{
			name: "direct create applies platform line ending default",
			args: []string{"create", "-f", "object.yaml"},
			expectations: []clitest.InspectionExpectation{
				clitest.Command("create").Flag("--windows-line-endings").Value().Text(strconv.FormatBool(runtime.GOOS == "windows")),
			},
		},
		{
			name: "direct create preserves quoted CSV filename containing comma",
			args: []string{"create", `--filename="one,two.yaml"`, "--raw=/api/v1/namespaces"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			expectations := []clitest.InspectionExpectation{clitest.InspectionReady()}
			expectations = append(expectations, test.expectations...)
			request := cli.InspectionRequest{
				Args: test.args,
			}
			clitest.Inspect(t.Context(), app, request).Require(t, expectations...)
		})
	}
}

func TestCreateFrontendUsesPflagLastOccurrenceSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"create", "-f", "object.yaml", "--field-manager=first", "--field-manager=second"},
	}
	clitest.Inspect(t.Context(), app, request).Require(
		t,
		clitest.InspectionReady(),
		clitest.Command("create").Flag("--field-manager").Value().Text("second"),
	)
}

func TestCreateRuntimeReadsInvocationStdin(t *testing.T) {
	responses := map[string]string{
		"/api":    `{"kind":"APIVersions","apiVersion":"v1","versions":["v1"],"serverAddressByClientCIDRs":[]}`,
		"/apis":   `{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`,
		"/api/v1": `{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"v1","resources":[{"name":"namespaces","singularName":"namespace","namespaced":false,"kind":"Namespace","verbs":["create"]}]}`,
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, ok := responses[request.URL.Path]
		if !ok {
			http.Error(writer, "unexpected path: "+request.URL.Path, http.StatusNotFound)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfg := createConfig{
		Client: clientConfig{
			Server:   server.URL,
			Insecure: true,
		},
		Streams: genericiooptions.IOStreams{
			In:     strings.NewReader("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: from-stdin\n"),
			Out:    &stdout,
			ErrOut: &stderr,
		},
		DryRun:       cmdutil.DryRunClient,
		Filename:     [][]string{{"-"}},
		FieldManager: "kubectl-create",
		Validate:     "false",
	}
	if err := runCreate(t.Context(), &cfg); err != nil {
		t.Fatalf("runCreate() error = %v; stderr = %q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "from-stdin") {
		t.Fatalf("stdout = %q, want created object name", stdout.String())
	}
}
