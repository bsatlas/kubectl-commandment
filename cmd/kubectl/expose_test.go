package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestExposeDeclaresResourceSourceGrammar(t *testing.T) {
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
			args:  []string{"expose", "deployment/foo", "--dry-run=wat"},
			ready: false,
		},
		{
			name:  "invalid output format",
			args:  []string{"expose", "deployment/foo", "--output=wat"},
			ready: false,
		},
		{
			name:  "invalid session affinity",
			args:  []string{"expose", "deployment/foo", "--session-affinity=Sticky"},
			ready: false,
		},
		{
			name:  "invalid selector syntax",
			args:  []string{"expose", "deployment/foo", "--selector=a=b=c"},
			ready: false,
		},
		{
			name:  "invalid labels syntax",
			args:  []string{"expose", "deployment/foo", "--labels=a=b=c"},
			ready: false,
		},
		{
			name:  "invalid port",
			args:  []string{"expose", "deployment/foo", "--port=wat"},
			ready: false,
		},
		{
			name:  "multiple ports",
			args:  []string{"expose", "deployment/foo", "--port=80,443"},
			ready: true,
		},
		{
			name:  "invalid override type when overrides are used",
			args:  []string{"expose", "deployment/foo", "--overrides={}", "--override-type=wat"},
			ready: false,
		},
		{
			name:  "unused invalid override type remains compatible",
			args:  []string{"expose", "deployment/foo", "--override-type=wat"},
			ready: true,
		},
		{
			name:  "resource argument",
			args:  []string{"expose", "deployment/foo"},
			ready: true,
		},
		{
			name:  "assigned short filename",
			args:  []string{"expose", "-f=deployment.yaml"},
			ready: true,
		},
		{
			name:  "filename",
			args:  []string{"expose", "-f", "deployment.yaml"},
			ready: true,
		},
		{
			name:  "kustomize",
			args:  []string{"expose", "-k", "dir"},
			ready: true,
		},
		{
			name:  "missing source",
			args:  []string{"expose"},
			ready: false,
		},
		{
			name:  "filename and kustomize",
			args:  []string{"expose", "-f", "a.yaml", "-k", "dir"},
			ready: false,
		},
		{
			name:  "recursive kustomize",
			args:  []string{"expose", "-k", "dir", "--recursive"},
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

func TestExposeRepeatedScalarUsesLastOccurrence(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"expose", "deployment/foo", "--name=first", "--name=second"},
	}
	inspection := clitest.Inspect(t.Context(), app, request)
	inspection.Require(t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("expose").Flag("--name").Value().Text("second"),
	)
}
