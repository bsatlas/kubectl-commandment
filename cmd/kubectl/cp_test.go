package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCopyFrontendContract(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"cp", "./local", "pod-a:/remote"},
	}).Require(t, clitest.InspectionReady())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"cp", "./left", "./right"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"cp", "pod-a:/left", "pod-b:/right"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"cp", "./local", "pod-a:"},
	}).Require(t, clitest.InspectionRejected())
}
