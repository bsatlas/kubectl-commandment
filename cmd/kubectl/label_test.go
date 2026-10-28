package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestLabelDeclaresResourceAndUpdateGrammar(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	tests := []struct {
		name  string
		args  []string
		ready bool
	}{
		{
			name:  "resource update",
			args:  []string{"label", "pods", "foo", "tier=frontend"},
			ready: true,
		},
		{
			name:  "list without update",
			args:  []string{"label", "pods", "foo", "--list"},
			ready: true,
		},
		{
			name:  "missing update",
			args:  []string{"label", "pods", "foo"},
			ready: false,
		},
		{
			name:  "list with update",
			args:  []string{"label", "pods", "foo", "tier=frontend", "--list"},
			ready: false,
		},
		{
			name:  "invalid label value",
			args:  []string{"label", "pods", "foo", "tier=bad$value"},
			ready: false,
		},
		{
			name:  "all with selector",
			args:  []string{"label", "pods", "tier=frontend", "--all", "--selector=app=web"},
			ready: false,
		},
		{
			name:  "list with output",
			args:  []string{"label", "pods", "foo", "--list", "--output=name"},
			ready: false,
		},
		{
			name:  "all with field selector",
			args:  []string{"label", "pods", "tier=frontend", "--all", "--field-selector=metadata.name=foo"},
			ready: false,
		},
		{
			name:  "missing resource source",
			args:  []string{"label", "tier=frontend"},
			ready: false,
		},
		{
			name:  "modify and remove same label",
			args:  []string{"label", "pods", "foo", "tier=frontend", "tier-"},
			ready: false,
		},
		{
			name:  "local file update",
			args:  []string{"label", "-f", "pod.yaml", "tier=frontend", "--local"},
			ready: true,
		},
		{
			name:  "filename and kustomize",
			args:  []string{"label", "-f", "pod.yaml", "-k", "dir", "tier=frontend"},
			ready: false,
		},
		{
			name:  "filename and explicit empty kustomize",
			args:  []string{"label", "-f", "pod.yaml", "--kustomize=", "tier=frontend"},
			ready: true,
		},
		{
			name:  "explicit empty kustomize is not a source",
			args:  []string{"label", "--kustomize=", "tier=frontend"},
			ready: false,
		},
		{
			name:  "recursive kustomize",
			args:  []string{"label", "-k", "dir", "--recursive", "tier=frontend"},
			ready: false,
		},
		{
			name:  "local server dry run",
			args:  []string{"label", "-f", "pod.yaml", "tier=frontend", "--local", "--dry-run=server"},
			ready: false,
		},
		{
			name:  "local without file",
			args:  []string{"label", "tier=frontend", "--local"},
			ready: false,
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
