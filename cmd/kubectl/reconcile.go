package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type authReconcileConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	Filename                 []string
	Kustomize                string
	Output                   string
	Recursive                bool
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
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[authReconcileConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore())
			})

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("Filename, directory, or URL to files identifying the resource to reconcile.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			removeExtraPermissionsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.RemoveExtraPermissions))
			removeExtraPermissionsFlag = removeExtraPermissionsFlag.Description("If true, removes extra permissions added to roles")
			removeExtraPermissionsFlag.Long("remove-extra-permissions").ImplicitValue(true)

			removeExtraSubjectsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.RemoveExtraSubjects))
			removeExtraSubjectsFlag = removeExtraSubjectsFlag.Description("If true, removes extra subjects added to rolebindings")
			removeExtraSubjectsFlag.Long("remove-extra-subjects").ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")
		},
		Run: runAuthReconcile,
	}
	return cmd
}
