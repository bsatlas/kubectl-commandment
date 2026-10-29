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
	Filename                 []string
	Kustomize                string
	Local                    bool
	Output                   string
	Record                   bool
	Recursive                bool
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
			resources := cb.Values.Sequence(&cb.Config.Resources, cli.Value.String())
			serviceAccount := cb.Values.String(&cb.Config.ServiceAccount)
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[setServiceAccountConfig]) {
				form.Sequence(resources, "RESOURCE", cli.Count.ZeroOrMore()).Minimal()
				form.ValueFor(serviceAccount, "SERVICE_ACCOUNT")
			})

			allFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.All))
			allFlag.Description("Select all resources, in the namespace of the specified resource types")
			allFlag.Long("all").ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, util.DryRunNone)
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue(util.DryRunClient)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-set")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.DelimitedBy(",")
			filenameFlag.Description("Filename, directory, or URL to files identifying the resource to get from a server.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomize := cb.Values.String(&cb.Config.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			local := cb.Values.Bool(&cb.Config.Local)
			localFlag := cb.Flags.ValueFor(local)
			localFlag.Description("If true, set serviceaccount will NOT contact api-server but run locally.")
			localFlag.Long("local").ImplicitValue(true)

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

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			cb.Rule.AtMostOne(filenameValue, kustomize)
			cb.Rule.When(cb.When.Equal(recursive, true)).Forbid(kustomize)
			cb.Rule.AtLeastOne(resources, filenameValue).And(kustomize)
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
