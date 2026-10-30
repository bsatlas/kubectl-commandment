package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type rolloutRestartConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Resources                resourceArguments
	Filename                 filenameOptions
	AllowMissingTemplateKeys bool
	FieldManager             string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Selector                 string
	ShowManagedFields        bool
	Template                 string
}

func rolloutRestartCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[rolloutRestartConfig] {
	cmd := cli.Command[rolloutRestartConfig]{
		Name: "restart",
		Description: cli.Description{
			Short: "Restart a resource",
			Long:  "Restart a resource.\n\nResource rollout will be restarted.",
		},
		Examples: []cli.Example{
			{
				Description: "Restart all deployments in the test-namespace namespace",
				CommandLine: "kubectl rollout restart deployment -n test-namespace",
			},
			{
				Description: "Restart a deployment",
				CommandLine: "kubectl rollout restart deployment/nginx",
			},
			{
				Description: "Restart a daemon set",
				CommandLine: "kubectl rollout restart daemonset/abc",
			},
			{
				Description: "Restart deployments with the app=nginx label",
				CommandLine: "kubectl rollout restart deployment --selector=app=nginx",
			},
		},
		Declare: func(cb *cli.CommandBuilder[rolloutRestartConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			resources := declareResourceArguments(
				cb,
				&cb.Config.Resources,
				client,
				func(cfg *rolloutRestartConfig) (clientConfig, resourceArguments) { return cfg.Client, cfg.Resources },
				[]string{"deployment", "daemonset", "statefulset"},
			)
			files := declareFilenameOptions(cb, &cb.Config.Filename)
			declarePrintFlags(
				cb,
				&cb.Config.Output,
				&cb.Config.OutputState,
				&cb.Config.Template,
				&cb.Config.AllowMissingTemplateKeys,
				&cb.Config.ShowManagedFields,
			)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager).Description("Name of the manager used to track field ownership.")
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			cb.Values.Default(fieldManagerFlag, "kubectl-rollout")
			fieldManagerFlag.Long("field-manager")

			selectorFlag := cb.Flags.String(&cb.Config.Selector).Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			cb.Values.ResolveWith(selectorFlag, cli.Value.KeepLast())
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			requireResourceOrFilenameSource(cb, resources, files)
		},
		Run: runRolloutRestart,
	}
	return cmd
}
