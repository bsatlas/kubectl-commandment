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
			kubeconfig := cb.Values.Derive(&cb.Config.Kubeconfig, cli.Value.ComputedBy(func(_ context.Context, _ *configSetContextConfig, dependencies cli.Dependencies) (string, error) {
				value, ok := dependencies.Get(kubeconfig)
				if !ok {
					return "", fmt.Errorf("kubeconfig dependency is unavailable")
				}
				return value, nil
			}).From(kubeconfig))
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

			current := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Current))
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
		},
		Run: runConfigSetContext,
	}
	return cmd
}

func runConfigSetContext(_ context.Context, cfg *configSetContextConfig) error {
	if len(cfg.Names) == 0 && !cfg.Current {
		return errors.New("you must specify a non-empty context name or --current")
	}
	if len(cfg.Names) > 0 && cfg.Current {
		return errors.New("you cannot specify both a context name and --current")
	}

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
