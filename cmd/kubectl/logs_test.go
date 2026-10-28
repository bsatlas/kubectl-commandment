package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestLogsFrontendContract(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"logs", "pod-a"},
	}).Require(t, clitest.InspectionReady())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"logs", "--selector=app=demo"},
	}).Require(t, clitest.InspectionReady())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"logs"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"logs", "pod-a", "--selector=app=demo"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"logs", "pod-a", "inline", "--container=flagged"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"logs", "pod-a", "--all-containers", "--container=flagged"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"logs", "pod-a", "--since=1m", "--since-time=2026-10-28T12:00:00Z"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"logs", "pod-a", "--tail=-2"},
	}).Require(t, clitest.InspectionRejected())
}
