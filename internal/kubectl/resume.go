package kubectl

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type rolloutResumeConfig struct {
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

func rolloutResumeCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[rolloutResumeConfig] {
	cmd := cli.Command[rolloutResumeConfig]{
		Name: "resume",
		Description: cli.Description{
			Short: "Resume a paused resource",
			Long:  "Resume a paused resource.\n\nPaused resources will not be reconciled by a controller. By resuming a resource, we allow it to be reconciled again. Currently only deployments support being resumed.",
		},
		Examples: []cli.Example{
			{
				Description: "Resume an already paused deployment",
				CommandLine: "kubectl rollout resume deployment/nginx",
			},
		},
		Declare: func(cb *cli.CommandBuilder[rolloutResumeConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			resources := declareResourceArguments(
				cb,
				&cb.Config.Resources,
				client,
				func(cfg *rolloutResumeConfig) (clientConfig, resourceArguments) { return cfg.Client, cfg.Resources },
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

			fieldManager := cb.Values.String(&cb.Config.FieldManager)
			cb.Values.Default(fieldManager, "kubectl-rollout")
			cb.Flags.ValueFor(fieldManager).Description("Name of the manager used to track field ownership.").Long("field-manager")

			selectorFlag := cb.Flags.String(&cb.Config.Selector).Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			requireResourceOrFilenameSource(cb, resources, files)
		},
		Run: runRolloutResume,
	}
	return cmd
}
