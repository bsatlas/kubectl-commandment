package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestKustomizeDeclaresArgumentAndTypedFlagDomains(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name     string
		args     []string
		rejected bool
	}{
		{
			name: "default directory",
			args: []string{"kustomize"},
		},
		{
			name: "one directory",
			args: []string{"kustomize", "overlay"},
		},
		{
			name:     "only one directory",
			args:     []string{"kustomize", "one", "two"},
			rejected: true,
		},
		{
			name: "root only load restrictor",
			args: []string{"kustomize", "--load-restrictor=LoadRestrictionsRootOnly"},
		},
		{
			name: "none load restrictor",
			args: []string{"kustomize", "--load-restrictor=LoadRestrictionsNone"},
		},
		{
			name:     "invalid load restrictor",
			args:     []string{"kustomize", "--load-restrictor=outside"},
			rejected: true,
		},
		{
			name: "legacy reorder",
			args: []string{"kustomize", "--reorder=legacy"},
		},
		{
			name: "none reorder",
			args: []string{"kustomize", "--reorder=none"},
		},
		{
			name:     "invalid reorder",
			args:     []string{"kustomize", "--reorder=random"},
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
			inspection.Require(t, clitest.InspectionRejected(), clitest.FailsWith(clitest.DiagnosticAt(0).Present()))
		})
	}
}

func TestKustomizeDeclaresManagedbyLabelEnvironment(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"kustomize"},
		Env: cli.Env{
			{
				Name:  "KUSTOMIZE_ENABLE_MANAGEDBY_LABEL",
				Value: "on",
			},
		},
	}
	managedbyLabel := clitest.Command("kustomize").Environment("KUSTOMIZE_ENABLE_MANAGEDBY_LABEL").Value()
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		managedbyLabel.Resolved(),
		managedbyLabel.Text("true"),
	)
}
