package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCreateSecretDockerRegistryFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "docker secret requires credentials or file",
			args: []string{"create", "secret", "docker-registry", "registry"},
		},
		{
			name: "docker secret validates nonempty name",
			args: []string{"create", "secret", "docker-registry", "", "--docker-username=user", "--docker-password=pass", "--docker-server=registry.example"},
		},
		{
			name: "docker secret file source has kubectl syntax",
			args: []string{"create", "secret", "docker-registry", "registry", "--from-file==path"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := cli.InspectionRequest{
				Args: test.args,
			}
			clitest.Inspect(t.Context(), app, request).Require(
				t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}

func TestCreateSecretDockerRegistryFrontendUsesPflagLastOccurrenceSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"create", "secret", "docker-registry", "registry", "--docker-username=user", "--docker-password=pass", "--docker-server=first.example", "--docker-server=second.example"},
	}
	clitest.Inspect(t.Context(), app, request).Require(
		t,
		clitest.InspectionReady(),
		clitest.Command("create", "secret", "docker-registry").Flag("--docker-server").Value().Text("second.example"),
	)
}
