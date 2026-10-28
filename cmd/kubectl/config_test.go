package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type configTestDestinations struct {
	Kubeconfig cli.Destination[rootConfig, string]
	Context    cli.Destination[rootConfig, string]
	Client     cli.Destination[rootConfig, clientConfig]
}

func compileConfigTestApplication(t *testing.T, declare func(*cli.CommandBuilder[struct{}], configTestDestinations)) cli.Application {
	t.Helper()

	definition := cli.Definition[rootConfig]{
		Name: "kubectl",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderInterspersed)

			kubeconfig := root.Values.String(&root.Config.Kubeconfig)
			root.Values.Default(kubeconfig, "")
			root.Flags.ValueFor(kubeconfig).Placement(cli.FlagPlacementSubtree).Long("kubeconfig")
			contextName := root.Values.String(&root.Config.Context)
			root.Values.Default(contextName, "")
			client := root.Values.Derive(&root.Config.Client, cli.Value.ComputedBy(func(context.Context, *rootConfig, cli.Dependencies) (clientConfig, error) {
				return clientConfig{}, nil
			}).From(kubeconfig))

			destinations := configTestDestinations{
				Kubeconfig: kubeconfig,
				Context:    contextName,
				Client:     client,
			}
			config := cli.Command[struct{}]{
				Name: "config",
				Declare: func(config *cli.CommandBuilder[struct{}]) {
					declare(config, destinations)
				},
			}
			root.Subcommands.Command(config)
		},
	}
	return clitest.Compile(t, definition)
}

func writeConfigTestKubeconfig(t *testing.T, current string, contexts ...string) string {
	t.Helper()

	config := api.NewConfig()
	config.CurrentContext = current
	config.Clusters["cluster"] = &api.Cluster{
		Server: "https://127.0.0.1",
	}
	config.AuthInfos["user"] = &api.AuthInfo{}
	for _, name := range contexts {
		config.Contexts[name] = &api.Context{
			Cluster:  "cluster",
			AuthInfo: "user",
		}
	}

	path := filepath.Join(t.TempDir(), "config")
	if err := clientcmd.WriteToFile(*config, path); err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}
	return path
}
