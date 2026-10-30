package main

import (
	"context"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type setServiceAccountConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Resources                []string
	ServiceAccount           string
	All                      bool
	AllowMissingTemplateKeys bool
	DryRun                   util.DryRunStrategy
	FieldManager             string
	Filename                 filenameOptions
	Local                    bool
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Record                   bool
	ShowManagedFields        bool
	Template                 string
}

func setServiceAccountCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[setServiceAccountConfig] {
	cmd := cli.Command[setServiceAccountConfig]{
		Name:    "serviceaccount",
		Aliases: cli.Aliases("sa"),
		Description: cli.Description{
			Short: "Update the service account of a resource",
			Long:  "Update the service account of pod template resources.\n\n Possible resources (case insensitive) can be:\n\n replicationcontroller (rc), deployment (deploy), daemonset (ds), job, replicaset (rs), statefulset",
		},
		Examples: []cli.Example{
			{
				Description: "Set deployment nginx-deployment's service account to serviceaccount1",
				CommandLine: "kubectl set serviceaccount deployment nginx-deployment serviceaccount1",
			},
			{
				Description: "Print the result (in YAML format) of updated nginx deployment with the service account from local file, without hitting the API server",
				CommandLine: "kubectl set sa -f nginx-deployment.yaml serviceaccount1 --local --dry-run=client -o yaml",
			},
		},
		Declare: func(cb *cli.CommandBuilder[setServiceAccountConfig]) {
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
			resources := cb.Values.Sequence(&cb.Config.Resources, cli.Value.String())
			serviceAccount := cb.Values.String(&cb.Config.ServiceAccount)
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[setServiceAccountConfig]) {
				form.Sequence(resources, "RESOURCE", cli.Count.ZeroOrMore()).Minimal()
				form.ValueFor(serviceAccount, "SERVICE_ACCOUNT")
			})

			allFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.All))
			cb.Values.ResolveWith(allFlag, cli.Value.KeepLast())
			allFlag.Description("Select all resources, in the namespace of the specified resource types")
			allFlag.Long("all").ImplicitValue(true)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, util.DryRunNone)
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue(util.DryRunClient)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-set")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename.Groups, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.Description("Filename, directory, or URL to files identifying the resource to get from a server.")
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

			local := cb.Values.Bool(&cb.Config.Local)
			localFlag := cb.Flags.ValueFor(local)
			cb.Values.ResolveWith(localFlag, cli.Value.KeepLast())
			localFlag.Description("If true, set serviceaccount will NOT contact api-server but run locally.")
			localFlag.Long("local").ImplicitValue(true)

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			cb.Values.ResolveWith(recordFlag, cli.Value.KeepLast())
			recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursive := cb.Values.Bool(&cb.Config.Filename.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive)
			cb.Values.ResolveWith(recursiveFlag, cli.Value.KeepLast())
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			cb.Rule.Check("kubectl-set-serviceaccount-filename-kustomize", func(_ context.Context, cfg *setServiceAccountConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Filename.Filenames()) == 0 || cfg.Filename.Kustomize == "", nil
			}).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
			cb.Rule.When(cb.When.Equal(recursive, true)).Forbid(kustomize)
			cb.Rule.Check("kubectl-set-serviceaccount-source", func(_ context.Context, cfg *setServiceAccountConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Resources) != 0 || len(cfg.Filename.Filenames()) != 0 || cfg.Filename.Kustomize != "", nil
			}).
				DependsOn(resources.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
			cb.Rule.Check("kubectl-set-serviceaccount-local-dry-run", func(_ context.Context, cfg *setServiceAccountConfig, _ cli.Dependencies) (bool, error) {
				return !cfg.Local || cfg.DryRun != util.DryRunServer, nil
			}).
				DependsOn(local.OptionalDependency()).
				DependsOn(dryRun.OptionalDependency())
		},
		Run: runSetServiceAccount,
	}
	return cmd
}
