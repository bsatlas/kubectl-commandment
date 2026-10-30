package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestApplyDeclaresSourceAndPolicyRelationships(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name     string
		args     []string
		env      cli.Env
		rejected bool
	}{
		{
			name: "filename source",
			args: []string{"apply", "-f", "object.yaml"},
		},
		{
			name:     "source required",
			args:     []string{"apply"},
			rejected: true,
		},
		{
			name: "kustomize source",
			args: []string{"apply", "-k", "overlay"},
		},
		{
			name:     "filename and kustomize conflict",
			args:     []string{"apply", "-f", "object.yaml", "-k", "overlay"},
			rejected: true,
		},
		{
			name:     "recursive kustomize conflict",
			args:     []string{"apply", "-k", "overlay", "-R"},
			rejected: true,
		},
		{
			name:     "force conflicts requires server side",
			args:     []string{"apply", "-f", "object.yaml", "--force-conflicts"},
			rejected: true,
		},
		{
			name: "force conflicts with server side",
			args: []string{"apply", "-f", "object.yaml", "--server-side", "--force-conflicts"},
		},
		{
			name:     "server side forbids force",
			args:     []string{"apply", "-f", "object.yaml", "--server-side", "--force"},
			rejected: true,
		},
		{
			name:     "all conflicts with selector",
			args:     []string{"apply", "-f", "object.yaml", "--all", "--selector=app=web"},
			rejected: true,
		},
		{
			name:     "subresource requires server side",
			args:     []string{"apply", "-f", "object.yaml", "--subresource=status"},
			rejected: true,
		},
		{
			name: "subresource with server side",
			args: []string{"apply", "-f", "object.yaml", "--server-side", "--subresource=status"},
		},
		{
			name:     "client dry run conflicts with server side",
			args:     []string{"apply", "-f", "object.yaml", "--dry-run=client", "--server-side"},
			rejected: true,
		},
		{
			name:     "server dry run forbids force",
			args:     []string{"apply", "-f", "object.yaml", "--dry-run=server", "--force"},
			rejected: true,
		},
		{
			name:     "prune forbids force",
			args:     []string{"apply", "-f", "object.yaml", "--prune", "--all", "--force"},
			rejected: true,
		},
		{
			name:     "server-side prune requires applyset",
			args:     []string{"apply", "-f", "object.yaml", "--prune", "--all", "--server-side"},
			rejected: true,
		},
		{
			name:     "prune requires selection",
			args:     []string{"apply", "-f", "object.yaml", "--prune"},
			rejected: true,
		},
		{
			name: "prune with all",
			args: []string{"apply", "-f", "object.yaml", "--prune", "--all"},
		},
		{
			name: "prune with selector",
			args: []string{"apply", "-f", "object.yaml", "--prune", "--selector=app=web"},
		},
		{
			name:     "applyset unavailable without feature environment",
			args:     []string{"apply", "-f", "object.yaml", "--prune", "--applyset=example"},
			rejected: true,
		},
		{
			name: "applyset enabled by environment",
			args: []string{"apply", "-f", "object.yaml", "--prune", "--applyset=example"},
			env: cli.Env{
				{
					Name:  "KUBECTL_APPLYSET",
					Value: "TrUe",
				},
			},
		},
		{
			name: "applyset permits server-side prune",
			args: []string{"apply", "-f", "object.yaml", "--prune", "--applyset=example", "--server-side"},
			env: cli.Env{
				{
					Name:  "KUBECTL_APPLYSET",
					Value: "true",
				},
			},
		},
		{
			name: "applyset conflicts with all",
			args: []string{"apply", "-f", "object.yaml", "--prune", "--applyset=example", "--all"},
			env: cli.Env{
				{
					Name:  "KUBECTL_APPLYSET",
					Value: "true",
				},
			},
			rejected: true,
		},
		{
			name: "applyset conflicts with selector",
			args: []string{"apply", "-f", "object.yaml", "--prune", "--applyset=example", "--selector=app=web"},
			env: cli.Env{
				{
					Name:  "KUBECTL_APPLYSET",
					Value: "true",
				},
			},
			rejected: true,
		},
		{
			name: "applyset forbids prune allowlist",
			args: []string{"apply", "-f", "object.yaml", "--prune", "--applyset=example", "--prune-allowlist=core/v1/ConfigMap"},
			env: cli.Env{
				{
					Name:  "KUBECTL_APPLYSET",
					Value: "true",
				},
			},
			rejected: true,
		},
		{
			name: "applyset still requires prune",
			args: []string{"apply", "-f", "object.yaml", "--applyset=example"},
			env: cli.Env{
				{
					Name:  "KUBECTL_APPLYSET",
					Value: "true",
				},
			},
			rejected: true,
		},
		{
			name: "non-true applyset environment leaves flag unavailable",
			args: []string{"apply", "-f", "object.yaml", "--prune", "--applyset=example"},
			env: cli.Env{
				{
					Name:  "KUBECTL_APPLYSET",
					Value: "1",
				},
			},
			rejected: true,
		},
		{
			name: "assigned short filename",
			args: []string{"apply", "-f=object.yaml"},
		},
		{
			name: "joined short filename",
			args: []string{"apply", "-fobject.yaml"},
		},
		{
			name: "repeated field manager keeps last",
			args: []string{"apply", "-f", "object.yaml", "--field-manager=first", "--field-manager=second"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := cli.InspectionRequest{
				Args: test.args,
				Env:  test.env,
			}
			inspection := clitest.Inspect(t.Context(), app, request)
			if !test.rejected {
				inspection.Require(t, clitest.InspectionReady())
				return
			}
			inspection.Require(t, clitest.InspectionRejected(), clitest.FailsWith(clitest.DiagnosticAt(0).Present()))
		})
	}
}

func TestApplyPruneHelpTracksApplySetFeature(t *testing.T) {
	t.Parallel()

	const alpha1 = "Automatically delete resource objects, that do not appear in the configs and are created by either apply or create --save-config. Should be used with either -l or --all."
	const alpha2 = "Automatically delete previously applied resource objects that do not appear in the provided configs. For alpha1, use with either -l or --all. For alpha2, use with --applyset."

	tests := []struct {
		name        string
		env         cli.Env
		description string
	}{
		{
			name:        "applyset disabled",
			description: alpha1,
		},
		{
			name: "applyset enabled",
			env: cli.Env{
				{
					Name:  "KUBECTL_APPLYSET",
					Value: "true",
				},
			},
			description: alpha2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var page cli.HelpPage
			definition := applicationDefinition(pluginCatalog{})
			definition.Help.Renderer = func(_ context.Context, observed cli.HelpPage) (string, error) {
				page = observed
				return "help", nil
			}
			app, err := cli.Compile(definition)
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			err = app.Run(t.Context(), cli.RunRequest{
				Args:   []string{"apply", "--help"},
				Env:    test.env,
				Stdout: io.Discard,
			})
			if err != nil {
				t.Fatalf("Run(apply --help) error = %v", err)
			}
			clitest.RequireHelpPresentation(t, page,
				clitest.HelpFlagGroup("Options").Flag("--prune").Description(test.description),
			)
		})
	}
}

func TestApplyValidatesCustomApplySetParentAtRuntime(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api":
			_, _ = io.WriteString(writer, `{"kind":"APIVersions","apiVersion":"v1","versions":["v1"],"serverAddressByClientCIDRs":[]}`)
		case "/api/v1":
			_, _ = io.WriteString(writer, `{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"v1","resources":[]}`)
		case "/apis":
			_, _ = io.WriteString(writer, `{"kind":"APIGroupList","apiVersion":"v1","groups":[{"name":"example.test","versions":[{"groupVersion":"example.test/v1","version":"v1"}],"preferredVersion":{"groupVersion":"example.test/v1","version":"v1"}},{"name":"apiextensions.k8s.io","versions":[{"groupVersion":"apiextensions.k8s.io/v1","version":"v1"}],"preferredVersion":{"groupVersion":"apiextensions.k8s.io/v1","version":"v1"}}]}`)
		case "/apis/example.test/v1":
			_, _ = io.WriteString(writer, `{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"example.test/v1","resources":[{"name":"widgets","singularName":"widget","namespaced":true,"kind":"Widget","verbs":["get","list"]}]}`)
		case "/apis/apiextensions.k8s.io/v1":
			_, _ = io.WriteString(writer, `{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"apiextensions.k8s.io/v1","resources":[{"name":"customresourcedefinitions","singularName":"customresourcedefinition","namespaced":false,"kind":"CustomResourceDefinition","verbs":["get","list"]}]}`)
		case "/apis/apiextensions.k8s.io/v1/customresourcedefinitions":
			_, _ = io.WriteString(writer, `{"kind":"CustomResourceDefinitionList","apiVersion":"apiextensions.k8s.io/v1","metadata":{"resourceVersion":"1"},"items":[]}`)
		default:
			http.Error(writer, "unexpected path: "+request.URL.Path, http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	config := api.NewConfig()
	config.CurrentContext = "fixture"
	config.Clusters["fixture"] = &api.Cluster{Server: server.URL}
	config.Contexts["fixture"] = &api.Context{Cluster: "fixture", Namespace: "test"}
	kubeconfig := filepath.Join(t.TempDir(), "config")
	if err := clientcmd.WriteToFile(*config, kubeconfig); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	err = app.Run(t.Context(), cli.RunRequest{
		Args: []string{
			"--kubeconfig", kubeconfig,
			"apply", "-f", "object.yaml",
			"--prune", "--applyset=widgets.example.test/demo", "--validate=false",
		},
		Env:    cli.Env{{Name: "KUBECTL_APPLYSET", Value: "true"}},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if err == nil {
		t.Fatal("Run() error = nil, want custom ApplySet parent validation failure")
	}
	if !strings.Contains(err.Error(), `resource "widgets.example.test" is not permitted as an ApplySet parent`) {
		t.Fatalf("Run() error = %v, want custom ApplySet parent rejection", err)
	}
}
