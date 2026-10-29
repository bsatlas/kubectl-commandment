package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestTaintDeclaresResourceAndChangeGrammar(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name  string
		args  []string
		ready bool
	}{
		{
			name:  "named node",
			args:  []string{"taint", "nodes", "node-a", "dedicated=special:NoSchedule"},
			ready: true,
		},
		{
			name:  "selector nodes",
			args:  []string{"taint", "nodes", "dedicated=special:NoSchedule", "--selector=role=worker"},
			ready: true,
		},
		{
			name:  "all nodes",
			args:  []string{"taint", "nodes", "dedicated=special:NoSchedule", "--all"},
			ready: true,
		},
		{
			name:  "missing update",
			args:  []string{"taint", "nodes", "node-a"},
			ready: false,
		},
		{
			name:  "missing target",
			args:  []string{"taint", "nodes", "dedicated=special:NoSchedule"},
			ready: false,
		},
		{
			name:  "all with selector",
			args:  []string{"taint", "nodes", "dedicated=special:NoSchedule", "--all", "--selector=role=worker"},
			ready: false,
		},
		{
			name:  "add and remove same taint",
			args:  []string{"taint", "nodes", "node-a", "dedicated=special:NoSchedule", "dedicated:NoSchedule-"},
			ready: false,
		},
		{
			name:  "invalid taint key",
			args:  []string{"taint", "nodes", "node-a", "bad$key=value:NoSchedule"},
			ready: false,
		},
		{
			name:  "invalid taint value",
			args:  []string{"taint", "nodes", "node-a", "dedicated=bad$value:NoSchedule"},
			ready: false,
		},
		{
			name:  "invalid effect",
			args:  []string{"taint", "nodes", "node-a", "dedicated=special:Sometimes"},
			ready: false,
		},
		{
			name:  "remove without effect treats equals as invalid key syntax",
			args:  []string{"taint", "nodes", "node-a", "dedicated=special-"},
			ready: false,
		},
		{
			name:  "remove with value and effect follows upstream parser",
			args:  []string{"taint", "nodes", "node-a", "dedicated=special:NoSchedule-"},
			ready: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := cli.InspectionRequest{
				Args: test.args,
			}
			inspection := clitest.Inspect(t.Context(), app, request)
			if test.ready {
				inspection.Require(t, clitest.InspectionReady())
				return
			}
			inspection.Require(t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}

func TestTaintReportsMutationOperation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		taints     string
		change     string
		wantOutput string
	}{
		{
			name:       "addition",
			taints:     `[]`,
			change:     "dedicated=special:NoSchedule",
			wantOutput: "node/node-a tainted\n",
		},
		{
			name:       "removal",
			taints:     `[{"key":"dedicated","value":"special","effect":"NoSchedule"}]`,
			change:     "dedicated:NoSchedule-",
			wantOutput: "node/node-a untainted\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := taintTestAPIServer(t, test.taints)
			kubeconfig := writeGetTestKubeconfig(t, server.URL)
			app, err := application(pluginCatalog{})
			if err != nil {
				t.Fatalf("Application() error = %v", err)
			}
			var stdout bytes.Buffer
			request := cli.RunRequest{
				Args: []string{
					"taint", "nodes", "node-a", test.change,
					"--dry-run=client",
					"--kubeconfig", kubeconfig,
				},
				Stdout: &stdout,
				Stderr: io.Discard,
			}
			if err := app.Run(t.Context(), request); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if got := stdout.String(); got != test.wantOutput {
				t.Fatalf("stdout = %q, want %q", got, test.wantOutput)
			}
		})
	}
}

func taintTestAPIServer(t *testing.T, taints string) *httptest.Server {
	t.Helper()

	responses := map[string]string{
		"/api":                 `{"kind":"APIVersions","apiVersion":"v1","versions":["v1"],"serverAddressByClientCIDRs":[]}`,
		"/apis":                `{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`,
		"/api/v1":              `{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"v1","resources":[{"name":"nodes","singularName":"node","namespaced":false,"kind":"Node","verbs":["get","list","patch","update"],"shortNames":["no"]}]}`,
		"/api/v1/nodes/node-a": `{"apiVersion":"v1","kind":"Node","metadata":{"name":"node-a"},"spec":{"taints":` + taints + `}}`,
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
	return server
}
