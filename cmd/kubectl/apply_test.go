package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestApplyDeclaresSourceAndPolicyRelationships(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
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
