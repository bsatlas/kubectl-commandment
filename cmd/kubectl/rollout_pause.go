package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type rolloutPauseConfig struct {
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

func rolloutPauseCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[rolloutPauseConfig] {
	cmd := cli.Command[rolloutPauseConfig]{
		Name: "pause",
		Description: cli.Description{
			Short: "Mark the provided resource as paused",
			Long:  "Mark the provided resource as paused.\n\nPaused resources will not be reconciled by a controller. Use \"kubectl rollout resume\" to resume a paused resource. Currently only deployments support being paused.",
		},
		Examples: []cli.Example{
			{
				Description: "Mark the nginx deployment as paused\nAny current state of the deployment will continue its function; new updates\nto the deployment will not have an effect as long as the deployment is paused",
				CommandLine: "kubectl rollout pause deployment/nginx",
			},
		},
		Declare: func(cb *cli.CommandBuilder[rolloutPauseConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			resources := declareResourceArguments(
				cb,
				&cb.Config.Resources,
				client,
				func(cfg *rolloutPauseConfig) (clientConfig, resourceArguments) { return cfg.Client, cfg.Resources },
				[]string{"deployment"},
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
		Run: runRolloutPause,
	}
	return cmd
}
