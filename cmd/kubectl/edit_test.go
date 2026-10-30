package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestEditDeclaresResourceSourceGrammar(t *testing.T) {
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
			name:  "invalid output format",
			args:  []string{"edit", "deployment/foo", "--output=wat"},
			ready: false,
		},
		{
			name:  "resource argument",
			args:  []string{"edit", "deployment/foo"},
			ready: true,
		},
		{
			name:  "assigned short filename",
			args:  []string{"edit", "-f=deployment.yaml"},
			ready: true,
		},
		{
			name:  "filename",
			args:  []string{"edit", "-f", "deployment.yaml"},
			ready: true,
		},
		{
			name:  "missing source",
			args:  []string{"edit"},
			ready: false,
		},
		{
			name:  "filename and kustomize",
			args:  []string{"edit", "-f", "a.yaml", "-k", "dir"},
			ready: false,
		},
		{
			name:  "recursive kustomize",
			args:  []string{"edit", "-k", "dir", "--recursive"},
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

func TestEditRepeatedScalarUsesLastOccurrence(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"edit", "deployment/foo", "--field-manager=first", "--field-manager=second"},
	}
	inspection := clitest.Inspect(t.Context(), app, request)
	inspection.Require(t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("edit").Flag("--field-manager").Value().Text("second"),
	)
}
