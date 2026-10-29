package main

import (
	"context"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type scaleConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	All                      bool
	AllowMissingTemplateKeys bool
	CurrentReplicas          int
	DryRun                   string
	Filename                 []string
	Kustomize                string
	Output                   string
	Record                   bool
	Recursive                bool
	Replicas                 int
	ResourceVersion          string
	Selector                 string
	ShowManagedFields        bool
	Template                 string
	Timeout                  time.Duration
}

func scaleCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[scaleConfig] {
	cmd := cli.Command[scaleConfig]{
		Name: "scale",
		Description: cli.Description{
			Short: "Set a new size for a deployment, replica set, or replication controller",
			Long:  "Set a new size for a deployment, replica set, replication controller, or stateful set.\n\n Scale also allows users to specify one or more preconditions for the scale action.\n\n If --current-replicas or --resource-version is specified, it is validated before the scale is attempted, and it is guaranteed that the precondition holds true when the scale is sent to the server.",
		},
		Examples: []cli.Example{
			{
				Description: "Scale a replica set named 'foo' to 3",
				CommandLine: "kubectl scale --replicas=3 rs/foo",
			},
			{
				Description: "Scale a resource identified by type and name specified in \"foo.yaml\" to 3",
				CommandLine: "kubectl scale --replicas=3 -f foo.yaml",
			},
			{
				Description: "If the deployment named mysql's current size is 2, scale mysql to 3",
				CommandLine: "kubectl scale --current-replicas=2 --replicas=3 deployment/mysql",
			},
			{
				Description: "Scale multiple replication controllers",
				CommandLine: "kubectl scale --replicas=5 rc/example1 rc/example2 rc/example3",
			},
			{
				Description: "Scale stateful set named 'web' to 3",
				CommandLine: "kubectl scale --replicas=3 statefulset/web",
			},
		},
		Declare: func(cb *cli.CommandBuilder[scaleConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[scaleConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *scaleConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						[]string{"deployment", "replicaset", "replicationcontroller", "statefulset"},
						resourceArgumentRepeated,
					)).
					From(client)
			})

			all := cb.Values.Bool(&cb.Config.All)
			allFlag := cb.Flags.ValueFor(all)
			allFlag.Description("Select all resources in the namespace of the specified resource types")
			allFlag.Long("all").ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			currentReplicas := cb.Values.Int(&cb.Config.CurrentReplicas)
			currentReplicasFlag := cb.Flags.ValueFor(currentReplicas)
			currentReplicasFlag.Description("Precondition for current size. Requires that the current size of the resource match this value in order to scale. -1 (default) for no condition.")
			cb.Values.Default(currentReplicas, -1)
			currentReplicasFlag.Long("current-replicas")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.DelimitedBy(",")
			filenameFlag.Description("Filename, directory, or URL to files identifying the resource to set a new size")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomize := cb.Values.String(&cb.Config.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursive := cb.Values.Bool(&cb.Config.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive)
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			replicas := cb.Values.Int(&cb.Config.Replicas)
			replicasFlag := cb.Flags.ValueFor(replicas)
			replicasFlag.Description("The new desired number of replicas. Required.")
			replicasFlag.Long("replicas")

			resourceVersionFlag := cb.Flags.String(&cb.Config.ResourceVersion)
			resourceVersionFlag.Description("Precondition for resource version. Requires that the current resource version match this value in order to scale.")
			resourceVersionFlag.Long("resource-version")

			selector := cb.Values.String(&cb.Config.Selector)
			selectorFlag := cb.Flags.ValueFor(selector)
			selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			timeoutFlag := cb.Flags.Value(&cb.Config.Timeout, durationLanguage())
			timeoutFlag.Description("The length of time to wait before giving up on a scale operation, zero means don't wait. Any other values should contain a corresponding time unit (e.g. 1s, 2m, 3h).")
			timeoutFlag.Long("timeout")

			cb.Rule.Require(replicas)
			cb.Rule.AtLeast(replicas, 0)
			cb.Rule.AtLeast(currentReplicas, -1)
			cb.Rule.When(cb.When.Equal(recursive, true)).Equal(kustomize, "")
			cb.Rule.Check("kubectl-scale-source", func(_ context.Context, cfg *scaleConfig, _ cli.Dependencies) (bool, error) {
				hasFilename := len(cfg.Filename) > 0
				hasKustomize := cfg.Kustomize != ""
				if hasFilename && hasKustomize {
					return false, nil
				}
				return len(cfg.Arguments) > 0 || hasFilename || hasKustomize, nil
			}).
				DependsOn(arguments.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
		},
		Run: runScale,
	}
	return cmd
}
