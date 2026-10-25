package kubectlcommandment

import (
	"testing"

	"github.com/abstractionsinc/commandment/clitest"
)

func TestWorkstreamOneStaticSemanticsComeFromCompiledApplication(t *testing.T) {
	app := clitest.Compile(t, definition)
	root := clitest.RootCommand()
	config := clitest.Command("config")
	useContext := clitest.Command("config", "use-context")
	currentContext := clitest.Command("config", "current-context")
	get := clitest.Command("get")

	clitest.Schema(app).Require(t,
		root.Name("kubectl"),
		root.Flag("--kubeconfig").Exists(),
		root.Flag("--context").Exists(),
		root.Flag("--namespace").Exists(),
		root.Flag("-n").Exists(),
		config.Exists(),
		get.Exists(),
		root.Alias("list").DoesNotExist(),
		root.Alias("ls").DoesNotExist(),
		useContext.Exists(),
		currentContext.Exists(),
		config.Alias("use").Exists(),
		get.ArgumentFormsExactly(get.ArgumentFormAt(0), get.ArgumentFormAt(1), get.ArgumentFormAt(2)),
		get.Flag("--output").Exists(),
		get.Flag("-o").Exists(),
		get.Flag("--no-headers").Exists(),
		get.Flag("--show-managed-fields").Exists(),
		get.Flag("--template").Exists(),
		get.Flag("--allow-missing-template-keys").Exists(),
		get.Flag("--show-labels").Exists(),
		get.Flag("--sort-by").Exists(),
		get.Flag("--label-columns").Exists(),
		get.Flag("-L").Exists(),
		get.Flag("--show-kind").Exists(),
		get.Flag("--raw").Exists(),
		get.Flag("--watch").Exists(),
		get.Flag("-w").Exists(),
		get.Flag("--watch-only").Exists(),
		get.Flag("--output-watch-events").Exists(),
		get.Flag("--ignore-not-found").Exists(),
		get.Flag("--field-selector").Exists(),
		get.Flag("--all-namespaces").Exists(),
		get.Flag("-A").Exists(),
		get.Flag("--server-print").Exists(),
		get.Flag("--filename").Exists(),
		get.Flag("-f").Exists(),
		get.Flag("--kustomize").Exists(),
		get.Flag("-k").Exists(),
		get.Flag("--recursive").Exists(),
		get.Flag("-R").Exists(),
		get.Flag("--chunk-size").Exists(),
		get.Flag("--selector").Exists(),
		get.Flag("-l").Exists(),
		get.Flag("--subresource").Exists(),
	)
}
