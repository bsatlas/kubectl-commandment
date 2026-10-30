package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
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
			clusterFlag := cb.Flags.ValueFor(cluster)
			cb.Values.ResolveWith(clusterFlag, cli.Value.KeepLast())
			clusterFlag.Description("cluster for the context entry in kubeconfig").Long("cluster")
			cb.Access.ValueState(cluster, &cb.Config.ClusterState)

			currentValue := cb.Values.Bool(&cb.Config.Current)
			current := cb.Flags.ValueFor(currentValue)
			cb.Values.ResolveWith(current, cli.Value.KeepLast())
			current.Description("Modify the current context").Long("current").ImplicitValue(true)

			namespace := cb.Values.String(&cb.Config.Namespace)
			namespaceFlag := cb.Flags.ValueFor(namespace)
			cb.Values.ResolveWith(namespaceFlag, cli.Value.KeepLast())
			namespaceFlag.Description("namespace for the context entry in kubeconfig")
			namespaceFlag.CompleteBy(fixedResourceArgumentCompleter(
				func(cfg *configSetContextConfig) (clientConfig, []string) { return cfg.Client, nil },
				"namespace",
			)).From(client)
			namespaceFlag.Long("namespace")
			namespaceFlag.Short('n')
			cb.Access.ValueState(namespace, &cb.Config.NamespaceState)

			user := cb.Values.String(&cb.Config.User)
			userFlag := cb.Flags.ValueFor(user)
			cb.Values.ResolveWith(userFlag, cli.Value.KeepLast())
			userFlag.Description("user for the context entry in kubeconfig").Long("user")
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

func runConfigSetContext(_ context.Context, cfg *configSetContextConfig) error {
	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}

	name := ""
	if len(cfg.Names) == 1 {
		name = cfg.Names[0]
	}
	if cfg.Current {
		if config.CurrentContext == "" {
			return errors.New("no current context is set")
		}
		name = config.CurrentContext
	}

	starting, exists := config.Contexts[name]
	if !exists {
		starting = api.NewContext()
	}
	modified := *starting
	if flagProvided(cfg.ClusterState) {
		modified.Cluster = cfg.Cluster
	}
	if flagProvided(cfg.UserState) {
		modified.AuthInfo = cfg.User
	}
	if flagProvided(cfg.NamespaceState) {
		modified.Namespace = cfg.Namespace
	}
	config.Contexts[name] = &modified

	if err := clientcmd.ModifyConfig(access, *config, true); err != nil {
		return err
	}
	if exists {
		_, err = fmt.Fprintf(cfg.Stdout, "Context %q modified.\n", name)
	} else {
		_, err = fmt.Fprintf(cfg.Stdout, "Context %q created.\n", name)
	}
	return err
}
