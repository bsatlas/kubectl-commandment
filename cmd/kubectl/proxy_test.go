package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestProxyFrontendContract(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"proxy"},
	}).Require(t, clitest.InspectionReady())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"proxy", "--unix-socket=/tmp/kubectl.sock"},
	}).Require(t, clitest.InspectionReady())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"proxy", "--unix-socket=/tmp/kubectl.sock", "--port=9000"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"proxy", "--accept-hosts=["},
	}).Require(t, clitest.InspectionRejected())
}
