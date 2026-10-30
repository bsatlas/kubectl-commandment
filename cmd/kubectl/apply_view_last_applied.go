package main

import (
	"context"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type applyViewLastAppliedConfig struct {
	Client    clientConfig
	Streams   genericiooptions.IOStreams
	Arguments []string
	All       bool
	Filename  filenameOptions
	Output    string
	Selector  string
}

func applyViewLastAppliedCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[applyViewLastAppliedConfig] {
	cmd := cli.Command[applyViewLastAppliedConfig]{
		Name: "view-last-applied",
		Description: cli.Description{
			Short: "View the latest last-applied-configuration annotations of a resource/object",
			Long:  "View the latest last-applied-configuration annotations by type/name or file.\n\n The default output will be printed to stdout in YAML format. You can use the -o option to change the output format.",
		},
		Examples: []cli.Example{
			{
				Description: "View the last-applied-configuration annotations by type/name in YAML",
				CommandLine: "kubectl apply view-last-applied deployment/nginx",
			},
			{
				Description: "View the last-applied-configuration annotations by file in JSON",
				CommandLine: "kubectl apply view-last-applied -f deploy.yaml -o json",
			},
		},
		Declare: func(cb *cli.CommandBuilder[applyViewLastAppliedConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[applyViewLastAppliedConfig]) {
				form.Sequence(arguments, "RESOURCE", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *applyViewLastAppliedConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						nil,
						resourceArgumentRepeated,
					)).
					From(client)
			})

			allFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.All))
			cb.Values.ResolveWith(allFlag, cli.Value.KeepLast())
			allFlag.Description("Select all resources in the namespace of the specified resource types").Long("all").ImplicitValue(true)

			filenameValue := cb.Values.Sequence(&cb.Config.Filename.Groups, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameValue).Description("Filename, directory, or URL to files that contains the last-applied-configuration annotations")
			filenameFlag.CompleteWith(cli.CompletionNativePaths)
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			kustomize := cb.Values.String(&cb.Config.Filename.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			cb.Values.ResolveWith(kustomizeFlag, cli.Value.KeepLast())
			kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.CompleteWith(cli.CompletionNativeDirectories)
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			cb.Values.ResolveWith(outputFlag, cli.Value.KeepLast())
			outputFlag.Description("Output format. Must be one of (yaml, json)").Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)
			cb.Values.Default(outputFlag, "yaml")
			cb.Rule.In(outputFlag, "yaml", "json")

			recursive := cb.Values.Bool(&cb.Config.Filename.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive)
			cb.Values.ResolveWith(recursiveFlag, cli.Value.KeepLast())
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.").Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			cb.Values.ResolveWith(selectorFlag, cli.Value.KeepLast())
			selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.").Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			cb.Rule.Check("kubectl-view-last-applied-filename-kustomize", func(_ context.Context, cfg *applyViewLastAppliedConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Filename.Filenames()) == 0 || cfg.Filename.Kustomize == "", nil
			}).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
			cb.Rule.When(cb.When.Equal(recursive, true)).Forbid(kustomize)
			cb.Rule.Check("kubectl-view-last-applied-source", func(_ context.Context, cfg *applyViewLastAppliedConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Arguments) != 0 || len(cfg.Filename.Filenames()) != 0 || cfg.Filename.Kustomize != "", nil
			}).
				DependsOn(arguments.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
		},
		Run: runApplyViewLastApplied,
	}
	return cmd
}
