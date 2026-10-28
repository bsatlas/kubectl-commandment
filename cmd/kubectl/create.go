package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type createConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	Edit                     bool
	FieldManager             string
	Filename                 []string
	Kustomize                string
	Output                   string
	Raw                      string
	Record                   bool
	Recursive                bool
	SaveConfig               bool
	Selector                 string
	ShowManagedFields        bool
	Template                 string
	Validate                 string
	WindowsLineEndings       bool
}

func createCommand(client cli.Destination[rootConfig, clientConfig], plugins []pluginNode) cli.Command[createConfig] {
	cmd := cli.Command[createConfig]{
		Name: "create",
		Description: cli.Description{
			Short: "Create a resource from a file or from stdin",
			Long:  "Create a resource from a file or from stdin.\n\n JSON and YAML formats are accepted.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a pod using the data in pod.json",
				CommandLine: "kubectl create -f ./pod.json",
			},
			{
				Description: "Create a pod based on the JSON passed into stdin",
				CommandLine: "cat pod.json | kubectl create -f -",
			},
			{
				Description: "Edit the data in registry.yaml in JSON then create the resource using the edited data",
				CommandLine: "kubectl create -f registry.yaml --edit -o json",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createConfig]) {
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

			editFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Edit))
			editFlag = editFlag.Description("Edit the API resource before creating")
			editFlag.Long("edit").ImplicitValue(true)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("Filename, directory, or URL to files to use to create the resource")
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

			rawFlag := cb.Flags.String(&cb.Config.Raw)
			rawFlag = rawFlag.Description("Raw URI to POST to the server.  Uses the transport specified by the kubeconfig file.")
			rawFlag.Long("raw")

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			recordFlag = recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag = recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag = selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag = validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			windowsLineEndingsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.WindowsLineEndings))
			windowsLineEndingsFlag = windowsLineEndingsFlag.Description("Only relevant if --edit=true. Defaults to the line ending native to your platform.")
			windowsLineEndingsFlag.Long("windows-line-endings").ImplicitValue(true)

			clusterRole := createClusterRoleCommand(client)
			cb.Subcommands.Command(clusterRole)
			clusterRoleBinding := createClusterRoleBindingCommand(client)
			cb.Subcommands.Command(clusterRoleBinding)
			configMap := createConfigMapCommand(client)
			cb.Subcommands.Command(configMap)
			cronJob := createCronJobCommand(client)
			cb.Subcommands.Command(cronJob)
			deployment := createDeploymentCommand(client)
			cb.Subcommands.Command(deployment)
			ingress := createIngressCommand(client)
			cb.Subcommands.Command(ingress)
			job := createJobCommand(client)
			cb.Subcommands.Command(job)
			namespace := createNamespaceCommand(client)
			cb.Subcommands.Command(namespace)
			podDisruptionBudget := createPodDisruptionBudgetCommand(client)
			cb.Subcommands.Command(podDisruptionBudget)
			priorityClass := createPriorityClassCommand(client)
			cb.Subcommands.Command(priorityClass)
			quota := createQuotaCommand(client)
			cb.Subcommands.Command(quota)
			role := createRoleCommand(client)
			cb.Subcommands.Command(role)
			roleBinding := createRoleBindingCommand(client)
			cb.Subcommands.Command(roleBinding)
			secret := createSecretCommand(client)
			cb.Subcommands.Command(secret)
			service := createServiceCommand(client)
			cb.Subcommands.Command(service)
			serviceAccount := createServiceAccountCommand(client)
			cb.Subcommands.Command(serviceAccount)
			token := createTokenCommand(client)
			cb.Subcommands.Command(token)

			occupied := pluginOccupiedNames(
				clusterRole.Name, clusterRoleBinding.Name, configMap.Name, cronJob.Name, deployment.Name, ingress.Name,
				job.Name, namespace.Name, podDisruptionBudget.Name, priorityClass.Name, quota.Name, role.Name, roleBinding.Name,
				secret.Name, service.Name, serviceAccount.Name, token.Name,
			)
			addPluginProviders(&cb.Subcommands, plugins, occupied)
		},
		Run: runCreate,
	}
	return cmd
}
