package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
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
			name:  "invalid dry run",
			args:  []string{"taint", "nodes", "node-a", "dedicated=special:NoSchedule", "--dry-run=wat"},
			ready: false,
		},
		{
			name:  "invalid output format",
			args:  []string{"taint", "nodes", "node-a", "dedicated=special:NoSchedule", "--output=wat"},
			ready: false,
		},
		{
			name:  "named node",
			args:  []string{"taint", "nodes", "node-a", "dedicated=special:NoSchedule"},
			ready: true,
		},
		{
			name:  "assigned short selector",
			args:  []string{"taint", "nodes", "dedicated=special:NoSchedule", "-l=role=worker"},
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
			wantOutput: "node/node-a tainted (dry run)\n",
		},
		{
			name:       "removal",
			taints:     `[{"key":"dedicated","value":"special","effect":"NoSchedule"}]`,
			change:     "dedicated:NoSchedule-",
			wantOutput: "node/node-a untainted (dry run)\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := taintTestAPIServer(t, test.taints)
			kubeconfig := writeTaintTestKubeconfig(t, server.URL)
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

func writeTaintTestKubeconfig(t *testing.T, server string) string {
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

func TestTaintRepeatedScalarUsesLastOccurrence(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"taint", "nodes", "node-a", "dedicated=special:NoSchedule", "--field-manager=first", "--field-manager=second"},
	}
	inspection := clitest.Inspect(t.Context(), app, request)
	inspection.Require(t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("taint").Flag("--field-manager").Value().Text("second"),
	)
}
