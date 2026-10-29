package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type rolloutUndoConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Resources                resourceArguments
	Filename                 filenameOptions
	AllowMissingTemplateKeys bool
	DryRun                   util.DryRunStrategy
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Selector                 string
	ShowManagedFields        bool
	Template                 string
	ToRevision               int64
}

func rolloutUndoCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[rolloutUndoConfig] {
	cmd := cli.Command[rolloutUndoConfig]{
		Name: "undo",
		Description: cli.Description{
			Short: "Undo a previous rollout",
			Long:  "Roll back to a previous rollout.",
		},
		Examples: []cli.Example{
			{
				Description: "Roll back to the previous deployment",
				CommandLine: "kubectl rollout undo deployment/abc",
			},
			{
				Description: "Roll back to daemonset revision 3",
				CommandLine: "kubectl rollout undo daemonset/abc --to-revision=3",
			},
			{
				Description: "Roll back to the previous deployment with dry-run",
				CommandLine: "kubectl rollout undo --dry-run=server deployment/abc",
			},
		},
		Declare: func(cb *cli.CommandBuilder[rolloutUndoConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			resources := declareResourceArguments(
				cb,
				&cb.Config.Resources,
				client,
				func(cfg *rolloutUndoConfig) (clientConfig, resourceArguments) { return cfg.Client, cfg.Resources },
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

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			cb.Values.Default(dryRun, util.DryRunNone)
			cb.Flags.ValueFor(dryRun).
				Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.").
				Long("dry-run").
				ImplicitValue(util.DryRunClient)

			selectorFlag := cb.Flags.String(&cb.Config.Selector).Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			toRevisionFlag := cb.Flags.Value(&cb.Config.ToRevision, int64Language()).Description("The revision to rollback to. Default to 0 (last revision).")
			toRevisionFlag.Long("to-revision")

			requireResourceOrFilenameSource(cb, resources, files)
			cb.Rule.AtLeast(toRevisionFlag.Destination(), int64(0))
		},
		Run: runRolloutUndo,
	}
	return cmd
}
