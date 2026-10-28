package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type rolloutHistoryConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Resources                resourceArguments
	Filename                 filenameOptions
	AllowMissingTemplateKeys bool
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Revision                 int64
	Selector                 string
	ShowManagedFields        bool
	Template                 string
}

func rolloutHistoryCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[rolloutHistoryConfig] {
	cmd := cli.Command[rolloutHistoryConfig]{
		Name: "history",
		Description: cli.Description{
			Short: "View rollout history",
			Long:  "View previous rollout revisions and configurations.",
		},
		Examples: []cli.Example{
			{
				Description: "View the rollout history of a deployment",
				CommandLine: "kubectl rollout history deployment/abc",
			},
			{
				Description: "View the details of daemonset revision 3",
				CommandLine: "kubectl rollout history daemonset/abc --revision=3",
			},
		},
		Declare: func(cb *cli.CommandBuilder[rolloutHistoryConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			resources := declareResourceArguments(
				cb,
				&cb.Config.Resources,
				client,
				func(cfg *rolloutHistoryConfig) (clientConfig, resourceArguments) { return cfg.Client, cfg.Resources },
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

			revisionFlag := cb.Flags.Value(&cb.Config.Revision, int64Language()).Description("See the details, including podTemplate of the revision specified")
			revisionFlag.Long("revision")

			selectorFlag := cb.Flags.String(&cb.Config.Selector).Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			requireResourceOrFilenameSource(cb, resources, files)
			cb.Rule.AtLeast(revisionFlag.Destination(), int64(0))
		},
		Run: runRolloutHistory,
	}
	return cmd
}
