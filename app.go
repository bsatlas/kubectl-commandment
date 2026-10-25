package kubectlcommandment

import (
	"github.com/abstractionsinc/commandment/cli"
	configcmd "github.com/abstractionsinc/kubectl-commandment/internal/config"
	getcmd "github.com/abstractionsinc/kubectl-commandment/internal/get"
	"github.com/abstractionsinc/kubectl-commandment/internal/root"
)

var definition = cli.Definition[root.Config]{
	Name:    "kubectl",
	Declare: declareRoot,
}

// Application compiles the kubectl Commandment frontend.
func Application() (cli.Application, error) {
	return cli.Compile(definition)
}

func declareRoot(cb *cli.CommandBuilder[root.Config]) {
	cb.Dialect.FlagOrder(cli.FlagOrderInterspersed)

	kubeconfig := cb.Values.String(&cb.Config.Kubeconfig)
	cb.Values.Default(kubeconfig, "")
	cb.Flags.ValueFor(kubeconfig).
		Placement(cli.FlagPlacementSubtree).
		Long("kubeconfig")

	contextName := cb.Values.String(&cb.Config.Context)
	cb.Values.Default(contextName, "")
	cb.Flags.ValueFor(contextName).
		Placement(cli.FlagPlacementSubtree).
		Long("context")

	namespace := cb.Values.String(&cb.Config.Namespace)
	cb.Values.Default(namespace, "")
	namespaceFlag := cb.Flags.ValueFor(namespace).
		Placement(cli.FlagPlacementSubtree)
	namespaceFlag.Long("namespace")
	namespaceFlag.Short('n')

	configCmd := configcmd.Command(kubeconfig, configcmd.Services{})
	cb.Subcommands.Command(configCmd)

	getCmd := getcmd.Command(kubeconfig, contextName, namespace, getcmd.Services{})
	cb.Subcommands.Command(getCmd)
}
