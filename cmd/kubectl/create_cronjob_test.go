package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCreateCronJobFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
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
			name: "cronjob requires image",
			args: []string{"create", "cronjob", "backup", "--schedule=* * * * *"},
		},
		{
			name: "cronjob requires schedule",
			args: []string{"create", "cronjob", "backup", "--image=busybox"},
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

func TestCreateCronJobFrontendAcceptsKubectlBoundaryCases(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name         string
		args         []string
		expectations []clitest.InspectionExpectation
	}{
		{
			name: "cronjob applies kubectl restart default",
			args: []string{"create", "cronjob", "backup", "--image=busybox", "--schedule=* * * * *"},
			expectations: []clitest.InspectionExpectation{
				clitest.Command("create", "cronjob").Flag("--restart").Value().Text("OnFailure"),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			expectations := []clitest.InspectionExpectation{clitest.InspectionReady()}
			expectations = append(expectations, test.expectations...)
			request := cli.InspectionRequest{
				Args: test.args,
			}
			clitest.Inspect(t.Context(), app, request).Require(t, expectations...)
		})
	}
}

func TestCreateCronJobRequiredFlagProjection(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	command := clitest.Command("create", "cronjob")
	clitest.Schema(app).Require(
		t,
		command.Flag("--image").Required(),
		command.Flag("--schedule").Required(),
	)
}

func TestCreateCronJobFrontendUsesPflagLastOccurrenceSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"create", "cronjob", "backup", "--image=busybox", "--schedule=* * * * *", "--restart=Never", "--restart=Always"},
	}
	clitest.Inspect(t.Context(), app, request).Require(
		t,
		clitest.InspectionReady(),
		clitest.Command("create", "cronjob").Flag("--restart").Value().Text("Always"),
	)
}
