package main

import (
	"context"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestReplaceDeclaresSourceForceAndRawGrammar(t *testing.T) {
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
			args:  []string{"replace", "-f", "a.yaml", "--dry-run=wat"},
			ready: false,
		},
		{
			name:  "invalid output format",
			args:  []string{"replace", "-f", "a.yaml", "--output=wat"},
			ready: false,
		},
		{
			name:  "assigned short filename",
			args:  []string{"replace", "-f=resource.yaml"},
			ready: true,
		},
		{
			name:  "filename",
			args:  []string{"replace", "-f", "resource.yaml"},
			ready: true,
		},
		{
			name:  "kustomize",
			args:  []string{"replace", "-k", "dir"},
			ready: true,
		},
		{
			name:  "missing source",
			args:  []string{"replace"},
			ready: false,
		},
		{
			name:  "filename and kustomize",
			args:  []string{"replace", "-f", "a.yaml", "-k", "dir"},
			ready: false,
		},
		{
			name:  "recursive kustomize",
			args:  []string{"replace", "-k", "dir", "--recursive"},
			ready: false,
		},
		{
			name:  "grace without force",
			args:  []string{"replace", "-f", "a.yaml", "--grace-period=0"},
			ready: false,
		},
		{
			name:  "timeout without force",
			args:  []string{"replace", "-f", "a.yaml", "--timeout=1s"},
			ready: false,
		},
		{
			name:  "force server dry run",
			args:  []string{"replace", "-f", "a.yaml", "--force", "--dry-run=server"},
			ready: false,
		},
		{
			name:  "grace with force",
			args:  []string{"replace", "-f", "a.yaml", "--grace-period=0", "--force"},
			ready: true,
		},
		{
			name:  "timeout with force",
			args:  []string{"replace", "-f", "a.yaml", "--timeout=1s", "--force"},
			ready: true,
		},
		{
			name:  "raw filename",
			args:  []string{"replace", "--raw=/api/v1/pods/foo", "-f", "a.yaml"},
			ready: true,
		},
		{
			name:  "raw invalid URI",
			args:  []string{"replace", "--raw=://bad", "-f", "a.yaml"},
			ready: false,
		},
		{
			name:  "raw remote filename",
			args:  []string{"replace", "--raw=/api/v1/pods/foo", "-f", "https://example.invalid/a.yaml"},
			ready: false,
		},
		{
			name:  "raw output",
			args:  []string{"replace", "--raw=/api/v1/pods/foo", "-f", "a.yaml", "--output=name"},
			ready: false,
		},
		{
			name:  "raw multiple files",
			args:  []string{"replace", "--raw=/api/v1/pods/foo", "-f", "a.yaml,b.yaml"},
			ready: false,
		},
		{
			name:  "raw recursive",
			args:  []string{"replace", "--raw=/api/v1/pods/foo", "-f", "a.yaml", "--recursive"},
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

func TestReplaceCarriesEffectiveDeletionWaitIntoExecution(t *testing.T) {
	t.Parallel()

	var observed replaceConfig
	definition := cli.Definition[rootConfig]{
		Name:    "kubectl",
		Release: kubectlRelease,
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			client := root.Values.Derive(
				&root.Config.Client,
				cli.Value.ComputedBy(func(context.Context, *rootConfig, cli.Dependencies) (clientConfig, error) {
					return clientConfig{}, nil
				}),
			)
			command := replaceCommand(client)
			command.Run = func(_ context.Context, cfg *replaceConfig) error {
				observed = *cfg
				return nil
			}
			root.Subcommands.Command(command)
		},
	}
	app := clitest.Compile(t, definition)
	tests := []struct {
		name               string
		args               []string
		wantWait           bool
		wantGracePeriod    int
		wantRequestedGrace int
	}{
		{
			name:               "default",
			args:               []string{"replace", "-f", "resource.yaml"},
			wantGracePeriod:    -1,
			wantRequestedGrace: -1,
		},
		{
			name:               "explicit wait",
			args:               []string{"replace", "-f", "resource.yaml", "--wait=true"},
			wantWait:           true,
			wantGracePeriod:    -1,
			wantRequestedGrace: -1,
		},
		{
			name:               "zero grace forces wait",
			args:               []string{"replace", "-f", "resource.yaml", "--grace-period=0", "--force", "--wait=false"},
			wantWait:           true,
			wantGracePeriod:    1,
			wantRequestedGrace: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observed = replaceConfig{}
			if err := app.Run(t.Context(), cli.RunRequest{Args: test.args}); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if observed.WaitForDeletion != test.wantWait {
				t.Fatalf("WaitForDeletion = %v, want %v", observed.WaitForDeletion, test.wantWait)
			}
			if observed.GracePeriod != test.wantGracePeriod {
				t.Fatalf("GracePeriod = %d, want %d", observed.GracePeriod, test.wantGracePeriod)
			}
			if observed.RequestedGrace != test.wantRequestedGrace {
				t.Fatalf("RequestedGrace = %d, want %d", observed.RequestedGrace, test.wantRequestedGrace)
			}
		})
	}
}

func TestReplaceRepeatedScalarUsesLastOccurrence(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"replace", "-f", "a.yaml", "--field-manager=first", "--field-manager=second"},
	}
	inspection := clitest.Inspect(t.Context(), app, request)
	inspection.Require(t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("replace").Flag("--field-manager").Value().Text("second"),
	)
}
