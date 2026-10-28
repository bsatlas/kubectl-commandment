package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestConfigViewDeclaresOutputLanguage(t *testing.T) {
	t.Parallel()

	app := compileConfigTestApplication(t, func(config *cli.CommandBuilder[struct{}], destinations configTestDestinations) {
		config.Subcommands.Command(configViewCommand(destinations.Kubeconfig, destinations.Context))
	})
	tests := []struct {
		name     string
		args     []string
		rejected bool
	}{
		{
			name: "yaml",
			args: []string{"config", "view", "--output=yaml"},
		},
		{
			name: "uppercase yaml",
			args: []string{"config", "view", "--output=YAML"},
		},
		{
			name: "go template inline",
			args: []string{"config", "view", "--output=go-template={{.kind}}"},
		},
		{
			name: "jsonpath inline",
			args: []string{"config", "view", "--output=jsonpath={.kind}"},
		},
		{
			name: "template flag supplies go template",
			args: []string{"config", "view", "--template={{.kind}}"},
		},
		{
			name:     "template output requires template",
			args:     []string{"config", "view", "--output=go-template"},
			rejected: true,
		},
		{
			name:     "inline template conflicts with template flag",
			args:     []string{"config", "view", "--output=go-template={{.kind}}", "--template={{.apiVersion}}"},
			rejected: true,
		},
		{
			name:     "unsupported output",
			args:     []string{"config", "view", "--output=wide"},
			rejected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := cli.InspectionRequest{
				Args: test.args,
			}
			inspection := clitest.Inspect(t.Context(), app, request)
			if !test.rejected {
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

func TestConfigViewMergeFalseRequiresExplicitKubeconfig(t *testing.T) {
	t.Parallel()

	app := compileConfigTestApplication(t, func(config *cli.CommandBuilder[struct{}], destinations configTestDestinations) {
		config.Subcommands.Command(configViewCommand(destinations.Kubeconfig, destinations.Context))
	})
	rejected := cli.InspectionRequest{
		Args: []string{"config", "view", "--merge=false"},
	}
	clitest.Inspect(t.Context(), app, rejected).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
	)

	path := writeConfigTestKubeconfig(t, "dev", "dev")
	ready := cli.InspectionRequest{
		Args: []string{"--kubeconfig", path, "config", "view", "--merge=false"},
	}
	clitest.Inspect(t.Context(), app, ready).Require(t, clitest.InspectionReady())
}

func TestConfigViewTemplateUsesNativeFileCompletion(t *testing.T) {
	t.Parallel()

	app := compileConfigTestApplication(t, func(config *cli.CommandBuilder[struct{}], destinations configTestDestinations) {
		config.Subcommands.Command(configViewCommand(destinations.Kubeconfig, destinations.Context))
	})
	request := cli.CompletionRequest{
		Args: []string{"config", "view", "--template", ""},
		Cursor: cli.CompletionCursor{
			Argument: 3,
		},
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				Native: cli.CompletionNativeCapabilities{
					Files: true,
				},
			},
		},
	}
	clitest.Complete(t.Context(), app, request).Require(t,
		clitest.CompletionNativeRequestsExactly(clitest.CompletionNativeRequestAt(0)),
		clitest.CompletionNativeRequestAt(0).Kind(cli.CompletionNativeFiles),
	)
}
