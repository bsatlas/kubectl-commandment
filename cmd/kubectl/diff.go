package main

import (
	"context"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type diffConfig struct {
	Client            clientConfig
	Streams           genericiooptions.IOStreams
	Concurrency       int
	FieldManager      string
	Filename          []string
	ForceConflicts    bool
	Kustomize         string
	Prune             bool
	PruneAllowlist    []string
	Recursive         bool
	Selector          string
	ServerSide        bool
	ShowManagedFields bool
	ShowSecrets       bool
}

func diffCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[diffConfig] {
	cmd := cli.Command[diffConfig]{
		Name: "diff",
		Description: cli.Description{
			Short: "Diff the live version against a would-be applied version",
			Long:  "Diff configurations specified by file name or stdin between the current online configuration, and the configuration as it would be if applied.\n\n The output is always YAML.\n\n KUBECTL_EXTERNAL_DIFF environment variable can be used to select your own diff command. Users can use external commands with params too, example: KUBECTL_EXTERNAL_DIFF=\"colordiff -N -u\"\n\n By default, the \"diff\" command available in your path will be run with the \"-u\" (unified diff) and \"-N\" (treat absent files as empty) options.\n\n Exit status: 0 No differences were found. 1 Differences were found. >1 Kubectl or diff failed with an error.\n\n Note: KUBECTL_EXTERNAL_DIFF, if used, is expected to follow that convention.",
		},
		Examples: []cli.Example{
			{
				Description: "Diff resources included in pod.json",
				CommandLine: "kubectl diff -f pod.json",
			},
			{
				Description: "Diff file read from stdin",
				CommandLine: "cat service.yaml | kubectl diff -f -",
			},
		},
		Declare: func(cb *cli.CommandBuilder[diffConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			fieldManager := cb.Values.String(&cb.Config.FieldManager)
			filename := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			forceConflicts := cb.Values.Bool(&cb.Config.ForceConflicts)
			kustomize := cb.Values.String(&cb.Config.Kustomize)
			recursive := cb.Values.Bool(&cb.Config.Recursive)
			serverSide := cb.Values.Bool(&cb.Config.ServerSide)

			concurrencyFlag := cb.Flags.Int(&cb.Config.Concurrency)
			concurrencyFlag = concurrencyFlag.Description("Number of objects to process in parallel when diffing against the live version. Larger number = faster, but more memory, I/O and CPU over that shorter period of time.")
			cb.Values.Default(concurrencyFlag, 1)
			concurrencyFlag.Long("concurrency")

			cb.Values.OptionalComputedDefault(
				fieldManager,
				cli.Value.OptionallyComputedBy(func(_ context.Context, _ *diffConfig, dependencies cli.Dependencies) (cli.ValueContribution[string], error) {
					server, _ := dependencies.Get(serverSide)
					if server {
						return cli.ContributeValue("kubectl"), nil
					}
					return cli.OmitValue[string](), nil
				}).From(serverSide.OptionalDependency()),
			)
			cb.Values.Default(fieldManager, "kubectl-client-side-apply")
			fieldManagerFlag := cb.Flags.ValueFor(fieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			fieldManagerFlag.Long("field-manager")

			filenameValue := filename
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("Filename, directory, or URL to files contains the configuration to diff")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			forceConflictsFlag := cb.Flags.ValueFor(forceConflicts)
			forceConflictsFlag = forceConflictsFlag.Description("If true, server-side apply will force the changes against conflicts.")
			forceConflictsFlag.Long("force-conflicts").ImplicitValue(true)

			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			pruneFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Prune))
			pruneFlag = pruneFlag.Description("Include resources that would be deleted by pruning. Can be used with -l and default shows all resources would be pruned")
			pruneFlag.Long("prune").ImplicitValue(true)

			pruneAllowlistValue := cb.Values.Sequence(&cb.Config.PruneAllowlist, cli.Value.String())
			pruneAllowlistFlag := cb.Flags.Sequence(pruneAllowlistValue)
			pruneAllowlistFlag = pruneAllowlistFlag.Description("Overwrite the default allowlist with <group/version/kind> for --prune")
			pruneAllowlistFlag.Long("prune-allowlist")

			recursiveFlag := cb.Flags.ValueFor(recursive)
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag = selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			serverSideFlag := cb.Flags.ValueFor(serverSide)
			serverSideFlag = serverSideFlag.Description("If true, apply runs in the server instead of the client.")
			serverSideFlag.Long("server-side").ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, include managed fields in the diff.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			showSecretsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowSecrets))
			showSecretsFlag = showSecretsFlag.Description("If true, do not mask secret values in the diff.")
			showSecretsFlag.Long("show-secrets").ImplicitValue(true)

			cb.Rule.AtMostOne(filename, kustomize)
			cb.Rule.When(cb.When.Equal(recursive, true)).Forbid(kustomize)
			cb.Rule.AtLeastOne(filename, kustomize)
			cb.Rule.When(cb.When.Equal(forceConflicts, true)).Require(serverSide)
			cb.Rule.When(cb.When.Equal(forceConflicts, true)).Equal(serverSide, true)
		},
		Run: runDiff,
	}
	return cmd
}
