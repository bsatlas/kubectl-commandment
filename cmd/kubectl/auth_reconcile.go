package main

import (
	"context"
	"io"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/kubectl/pkg/cmd/util"
)

type authReconcileConfig struct {
	Client clientConfig
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	AllowMissingTemplateKeys bool
	DryRun                   util.DryRunStrategy
	Filename                 filenameOptions
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	RemoveExtraPermissions   bool
	RemoveExtraSubjects      bool
	ShowManagedFields        bool
	Template                 string
}

func authReconcileCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[authReconcileConfig] {
	cmd := cli.Command[authReconcileConfig]{
		Name: "reconcile",
		Description: cli.Description{
			Short: "Reconciles rules for RBAC role, role binding, cluster role, and cluster role binding objects",
			Long:  "Reconciles rules for RBAC role, role binding, cluster role, and cluster role binding objects.\n\n Missing objects are created, and the containing namespace is created for namespaced objects, if required.\n\n Existing roles are updated to include the permissions in the input objects, and remove extra permissions if --remove-extra-permissions is specified.\n\n Existing bindings are updated to include the subjects in the input objects, and remove extra subjects if --remove-extra-subjects is specified.\n\n This is preferred to 'apply' for RBAC resources so that semantically-aware merging of rules and subjects is done.",
		},
		Examples: []cli.Example{
			{
				Description: "Reconcile RBAC resources from a file",
				CommandLine: "kubectl auth reconcile -f my-rbac-rules.yaml",
			},
		},
		Declare: func(cb *cli.CommandBuilder[authReconcileConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Stdin)
			cb.Access.Stdout(&cb.Config.Stdout)
			cb.Access.Stderr(&cb.Config.Stderr)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			cb.Values.Default(dryRun, util.DryRunNone)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			dryRunFlag.
				Description("Must be \"none\" or \"client\". If client strategy, only print the object that would be sent, without sending it.").
				Long("dry-run").
				ImplicitValue(util.DryRunClient)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			cb.Rule.NotEqual(dryRun, util.DryRunServer)

			filenames := cb.Values.Sequence(&cb.Config.Filename.Groups, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenames).
				Description("Filename, directory, or URL to files identifying the resource to reconcile.").
				CompleteWith(cli.CompletionNativePaths)
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			kustomize := cb.Values.String(&cb.Config.Filename.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize).
				Description("Process the kustomization directory. This flag can't be used together with -f or -R.").
				CompleteWith(cli.CompletionNativeDirectories)
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)
			cb.Values.ResolveWith(kustomizeFlag, cli.Value.KeepLast())

			declarePrintFlags(
				cb,
				&cb.Config.Output,
				&cb.Config.OutputState,
				&cb.Config.Template,
				&cb.Config.AllowMissingTemplateKeys,
				&cb.Config.ShowManagedFields,
			)

			recursive := cb.Values.Bool(&cb.Config.Filename.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive).
				Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
			cb.Values.ResolveWith(recursiveFlag, cli.Value.KeepLast())

			removeExtraPermissions := cb.Values.Bool(&cb.Config.RemoveExtraPermissions)
			removeExtraPermissionsFlag := cb.Flags.ValueFor(removeExtraPermissions)
			removeExtraPermissionsFlag.
				Description("If true, removes extra permissions added to roles").
				Long("remove-extra-permissions").
				ImplicitValue(true)
			cb.Values.ResolveWith(removeExtraPermissionsFlag, cli.Value.KeepLast())

			removeExtraSubjects := cb.Values.Bool(&cb.Config.RemoveExtraSubjects)
			removeExtraSubjectsFlag := cb.Flags.ValueFor(removeExtraSubjects)
			removeExtraSubjectsFlag.
				Description("If true, removes extra subjects added to rolebindings").
				Long("remove-extra-subjects").
				ImplicitValue(true)
			cb.Values.ResolveWith(removeExtraSubjectsFlag, cli.Value.KeepLast())

			cb.Rule.Check("auth-reconcile-source", func(_ context.Context, cfg *authReconcileConfig, _ cli.Dependencies) (bool, error) {
				hasFilenames := len(cfg.Filename.Filenames()) != 0
				hasKustomize := cfg.Filename.Kustomize != ""
				return hasFilenames != hasKustomize, nil
			}).Subject(filenames).Subject(kustomize)
			cb.Rule.When(cb.When.Equal(recursive, true)).Equal(kustomize, "")
		},
		Run: runAuthReconcile,
	}
	return cmd
}
