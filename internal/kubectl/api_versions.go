package kubectl

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type apiVersionsConfig struct {
	Client clientConfig
	Stdout io.Writer
}

func apiVersionsCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[apiVersionsConfig] {
	cmd := cli.Command[apiVersionsConfig]{
		Name: "api-versions",
		Description: cli.Description{
			Short: "Print the supported API versions on the server, in the form of \"group/version\"",
			Long:  "Print the supported API versions on the server, in the form of \"group/version\".",
		},
		Examples: []cli.Example{
			{
				Description: "Print the supported API versions",
				CommandLine: "kubectl api-versions",
			},
		},
		Declare: func(cb *cli.CommandBuilder[apiVersionsConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdout(&cb.Config.Stdout)
		},
		Run: runAPIVersions,
	}
	return cmd
}

func runAPIVersions(_ context.Context, cfg *apiVersionsConfig) error {
	discoveryClient, err := restClientGetter(cfg.Client).ToDiscoveryClient()
	if err != nil {
		return err
	}
	discoveryClient.Invalidate()

	groups, err := discoveryClient.ServerGroups()
	if err != nil {
		return fmt.Errorf("couldn't get available api versions from server: %v", err)
	}
	versions := v1.ExtractGroupVersions(groups)
	slices.Sort(versions)
	for _, version := range versions {
		_, _ = fmt.Fprintln(cfg.Stdout, version)
	}
	return nil
}
