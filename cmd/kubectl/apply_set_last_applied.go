package main

import (
	"context"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type applySetLastAppliedConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	AllowMissingTemplateKeys bool
	CreateAnnotation         bool
	DryRun                   util.DryRunStrategy
	Filename                 [][]string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	ShowManagedFields        bool
	Template                 string
}

func applySetLastAppliedCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[applySetLastAppliedConfig] {
	cmd := cli.Command[applySetLastAppliedConfig]{
		Name: "set-last-applied",
		Description: cli.Description{
			Short: "Set the last-applied-configuration annotation on a live object to match the contents of a file",
			Long:  "Set the latest last-applied-configuration annotations by setting it to match the contents of a file. This results in the last-applied-configuration being updated as though 'kubectl apply -f<file> ' was run, without updating any other parts of the object.",
		},
		Examples: []cli.Example{
			{
				Description: "Set the last-applied-configuration of a resource to match the contents of a file",
				CommandLine: "kubectl apply set-last-applied -f deploy.yaml",
			},
			{
				Description: "Execute set-last-applied against each configuration file in a directory",
				CommandLine: "kubectl apply set-last-applied -f path/",
			},
			{
				Description: "Set the last-applied-configuration of a resource to match the contents of a file; will create the annotation if it does not already exist",
				CommandLine: "kubectl apply set-last-applied -f deploy.yaml --create-annotation=true",
			},
		},
		Declare: func(cb *cli.CommandBuilder[applySetLastAppliedConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			declarePrintFlags(
				cb,
				&cb.Config.Output,
				&cb.Config.OutputState,
				&cb.Config.Template,
				&cb.Config.AllowMissingTemplateKeys,
				&cb.Config.ShowManagedFields,
			)

			createAnnotationFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.CreateAnnotation))
			cb.Values.ResolveWith(createAnnotationFlag, cli.Value.KeepLast())
			createAnnotationFlag.Description("Will create 'last-applied-configuration' annotations if current objects doesn't have one").Long("create-annotation").ImplicitValue(true)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.").Long("dry-run").ImplicitValue(util.DryRunClient)
			cb.Values.Default(dryRun, util.DryRunNone)

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameValue).Description("Filename, directory, or URL to files that contains the last-applied-configuration annotations")
			filenameFlag.CompleteWith(cli.CompletionNativePaths)
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			cb.Rule.Check("kubectl-set-last-applied-filename", func(_ context.Context, cfg *applySetLastAppliedConfig, _ cli.Dependencies) (bool, error) {
				return len(slices.Concat(cfg.Filename...)) != 0, nil
			}).DependsOn(filenameValue.OptionalDependency())
		},
		Run: runApplySetLastApplied,
	}
	return cmd
}
