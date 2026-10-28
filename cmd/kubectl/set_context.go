package main

import (
	"context"
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type configSetContextConfig struct {
	Kubeconfig     string
	Client         clientConfig
	Names          []string
	Cluster        string
	Current        bool
	Namespace      string
	User           string
	ClusterState   cli.OptionalValue[cli.ValueState]
	NamespaceState cli.OptionalValue[cli.ValueState]
	UserState      cli.OptionalValue[cli.ValueState]
	Stdout         io.Writer
}

func configSetContextCommand(kubeconfig cli.Destination[rootConfig, string], client cli.Destination[rootConfig, clientConfig]) cli.Command[configSetContextConfig] {
	cmd := cli.Command[configSetContextConfig]{
		Name: "set-context",
		Description: cli.Description{
			Short: "Set a context entry in kubeconfig",
			Long:  "Set a context entry in kubeconfig.\n\n Specifying a name that already exists will merge new fields on top of existing values for those fields.",
		},
		Examples: []cli.Example{
			{
				Description: "Set the user field on the gce context entry without touching other values",
				CommandLine: "kubectl config set-context gce --user=cluster-admin",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configSetContextConfig]) {
			kubeconfig := deriveKubeconfig(cb, &cb.Config.Kubeconfig, kubeconfig)
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdout(&cb.Config.Stdout)

			names := cb.Values.Sequence(&cb.Config.Names, cli.Value.String())
			cb.Arguments.Sequence(names, "NAME", cli.Count.Optional()).
				CompleteBy(kubeconfigNameCompleter(
					func(cfg *configSetContextConfig) string { return cfg.Kubeconfig },
					kubeconfigContextName,
				)).
				From(kubeconfig)

			cluster := cb.Values.String(&cb.Config.Cluster)
			cb.Flags.ValueFor(cluster).Description("cluster for the context entry in kubeconfig").Long("cluster")
			cb.Access.ValueState(cluster, &cb.Config.ClusterState)

			currentValue := cb.Values.Bool(&cb.Config.Current)
			current := cb.Flags.ValueFor(currentValue)
			current.Description("Modify the current context").Long("current").ImplicitValue(true)

			namespace := cb.Values.String(&cb.Config.Namespace)
			namespaceFlag := cb.Flags.ValueFor(namespace).Description("namespace for the context entry in kubeconfig")
			namespaceFlag.CompleteBy(fixedResourceArgumentCompleter(
				func(cfg *configSetContextConfig) (clientConfig, []string) { return cfg.Client, nil },
				"namespace",
			)).From(client)
			namespaceFlag.Long("namespace")
			namespaceFlag.Short('n')
			cb.Access.ValueState(namespace, &cb.Config.NamespaceState)

			user := cb.Values.String(&cb.Config.User)
			cb.Flags.ValueFor(user).Description("user for the context entry in kubeconfig").Long("user")
			cb.Access.ValueState(user, &cb.Config.UserState)

			cb.Rule.Check("kubectl-config-set-context-name", func(_ context.Context, cfg *configSetContextConfig, _ cli.Dependencies) (bool, error) {
				name := ""
				if len(cfg.Names) == 1 {
					name = cfg.Names[0]
				}
				return (name == "") == cfg.Current, nil
			}).
				DependsOn(names.OptionalDependency()).
				DependsOn(currentValue.OptionalDependency())
		},
		Run: runConfigSetContext,
	}
	return cmd
}
