package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/clitest"
)

func TestStaticFrontendSemanticsComeFromCompiledApplication(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	schema := app.Schema()
	if got, want := len(schema.CommandOrder), 110; got != want {
		t.Fatalf("static command count = %d, want %d", got, want)
	}

	root := clitest.RootCommand()
	expectations := []clitest.SchemaExpectation{
		root.Name("kubectl"),
		root.Flag("--kubeconfig").Exists(),
		root.Flag("--context").Exists(),
		root.Flag("--namespace").Exists(),
		root.Flag("-n").Exists(),
		clitest.Command("config").NotRunnable(),
		clitest.Command("config", "use-context").Runnable(),
		clitest.Command("get").Runnable(),
		clitest.Command("create", "deployment").Runnable(),
		clitest.Command("apply").Runnable(),
		clitest.Command("apply", "edit-last-applied").Runnable(),
		clitest.Command("rollout", "status").Runnable(),
		clitest.Command("set", "image").Runnable(),
		clitest.Command("auth", "can-i").Runnable(),
		clitest.Command("certificate", "approve").Runnable(),
		clitest.Command("cluster-info", "dump").Runnable(),
		clitest.Command("top", "pod").Runnable(),
		clitest.Command("cp").Runnable(),
		clitest.Command("debug").Runnable(),
		clitest.Command("diff").Runnable(),
		clitest.Command("edit").Runnable(),
		clitest.Command("exec").Runnable(),
		clitest.Command("logs").Runnable(),
		clitest.Command("port-forward").Runnable(),
		clitest.Command("proxy").Runnable(),
		clitest.Command("run").Runnable(),
		clitest.Command("plugin").NotRunnable(),
		clitest.Command("plugin", "list").Runnable(),
		clitest.Command("options").NotRunnable(),
	}
	clitest.Schema(app).Require(t, expectations...)
}
