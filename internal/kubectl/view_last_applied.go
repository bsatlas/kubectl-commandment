package kubectl

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type applyViewLastAppliedConfig struct {
	Client    clientConfig
	Streams   genericiooptions.IOStreams
	Arguments []string
	All       bool
	Filename  []string
	Kustomize string
	Output    string
	Recursive bool
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
			client := deriveClientConfig(cb, &cb.Config.Client, client)
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
			allFlag.Description("Select all resources in the namespace of the specified resource types").Long("all").ImplicitValue(true)

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.DelimitedBy(",").Description("Filename, directory, or URL to files that contains the last-applied-configuration annotations").Long("filename")
			filenameFlag.Short('f')

			kustomize := cb.Values.String(&cb.Config.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.").Long("kustomize")
			kustomizeFlag.Short('k')

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag.Description("Output format. Must be one of (yaml, json)").Long("output")
			outputFlag.Short('o')
			cb.Values.Default(outputFlag, "yaml")
			cb.Rule.In(outputFlag, "yaml", "json")

			recursive := cb.Values.Bool(&cb.Config.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive)
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.").Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.").Long("selector")
			selectorFlag.Short('l')

			cb.Rule.AtMostOne(filenameValue, kustomize)
			cb.Rule.When(cb.When.Equal(recursive, true)).Forbid(kustomize)
			cb.Rule.AtLeastOne(arguments, filenameValue).And(kustomize)
		},
		Run: runApplyViewLastApplied,
	}
	return cmd
}
