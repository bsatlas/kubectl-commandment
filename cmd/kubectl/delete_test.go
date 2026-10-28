package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestDeleteDeclaresSelectionDefaultsAndCompatibility(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	tests := []struct {
		name          string
		args          []string
		ready         bool
		ignoreMissing string
	}{
		{
			name:          "all defaults ignore not found",
			args:          []string{"delete", "pods", "--all"},
			ready:         true,
			ignoreMissing: "true",
		},
		{
			name:          "selector defaults ignore not found",
			args:          []string{"delete", "pods", "--selector=app=web"},
			ready:         true,
			ignoreMissing: "true",
		},
		{
			name:          "field selector defaults ignore not found",
			args:          []string{"delete", "pods", "--field-selector=metadata.name=foo"},
			ready:         true,
			ignoreMissing: "true",
		},
		{
			name:          "explicit ignore not found wins",
			args:          []string{"delete", "pods", "--all", "--ignore-not-found=false"},
			ready:         true,
			ignoreMissing: "false",
		},
		{
			name:  "deprecated boolean cascade remains accepted",
			args:  []string{"delete", "pod/foo", "--cascade=true"},
			ready: true,
		},
		{
			name: "invalid cascade",
			args: []string{"delete", "pod/foo", "--cascade=immediate"},
		},
		{
			name: "missing source",
			args: []string{"delete"},
		},
		{
			name: "all with selector",
			args: []string{"delete", "pods", "--all", "--selector=app=web"},
		},
		{
			name: "all with field selector",
			args: []string{"delete", "pods", "--all", "--field-selector=metadata.name=foo"},
		},
		{
			name: "filename and kustomize",
			args: []string{"delete", "-f", "a.yaml", "-k", "dir"},
		},
		{
			name:  "filename and explicit empty kustomize",
			args:  []string{"delete", "-f", "a.yaml", "--kustomize="},
			ready: true,
		},
		{
			name: "explicit empty kustomize is not a source",
			args: []string{"delete", "--kustomize="},
		},
		{
			name: "recursive kustomize",
			args: []string{"delete", "-k", "dir", "--recursive"},
		},
		{
			name:  "grace zero remains compatible without force",
			args:  []string{"delete", "pod/foo", "--grace-period=0"},
			ready: true,
		},
		{
			name: "positive grace with force",
			args: []string{"delete", "pod/foo", "--grace-period=1", "--force"},
		},
		{
			name: "now with force",
			args: []string{"delete", "pod/foo", "--now", "--force"},
		},
		{
			name: "now with requested grace",
			args: []string{"delete", "pod/foo", "--now", "--grace-period=2"},
		},
		{
			name:  "raw URI",
			args:  []string{"delete", "--raw=/api/v1/namespaces/default/pods/foo"},
			ready: true,
		},
		{
			name:  "name output",
			args:  []string{"delete", "pod/foo", "--output=name"},
			ready: true,
		},
		{
			name: "invalid raw URI",
			args: []string{"delete", "--raw=://bad"},
		},
		{
			name: "raw output",
			args: []string{"delete", "--raw=/api/v1/pods/foo", "--output=name"},
		},
		{
			name: "raw interactive",
			args: []string{"delete", "--raw=/api/v1/pods/foo", "--interactive"},
		},
		{
			name: "raw multiple files",
			args: []string{"delete", "--raw=/api/v1/pods/foo", "-f", "a.yaml,b.yaml"},
		},
		{
			name: "raw remote filename",
			args: []string{"delete", "--raw=/api/v1/pods/foo", "-f", "https://example.invalid/a.yaml"},
		},
		{
			name: "raw recursive",
			args: []string{"delete", "--raw=/api/v1/pods/foo", "--recursive"},
		},
		{
			name: "unsupported output",
			args: []string{"delete", "pod/foo", "--output=json"},
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
				expectations := []clitest.InspectionExpectation{clitest.InspectionReady()}
				if test.ignoreMissing != "" {
					expectations = append(expectations, clitest.Command("delete").Flag("--ignore-not-found").Value().Text(test.ignoreMissing))
				}
				inspection.Require(t, expectations...)
				return
			}
			inspection.Require(t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}
