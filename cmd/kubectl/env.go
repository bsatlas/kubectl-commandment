package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type setEnvConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	All                      bool
	AllowMissingTemplateKeys bool
	Containers               string
	DryRun                   string
	Env                      []string
	FieldManager             string
	Filename                 []string
	From                     string
	Keys                     []string
	Kustomize                string
	List                     bool
	Local                    bool
	Output                   string
	Overwrite                bool
	Prefix                   string
	Recursive                bool
	Resolve                  bool
	Selector                 string
	ShowManagedFields        bool
	Template                 string
}

func setEnvCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[setEnvConfig] {
	cmd := cli.Command[setEnvConfig]{
		Name: "env",
		Description: cli.Description{
			Short: "Update environment variables on a pod template",
			Long:  "Update environment variables on a pod template.\n\n List environment variable definitions in one or more pods, pod templates. Add, update, or remove container environment variable definitions in one or more pod templates (within replication controllers or deployment configurations). View or modify the environment variable definitions on all containers in the specified pods or pod templates, or just those that match a wildcard.\n\n If \"--env -\" is passed, environment variables can be read from STDIN using the standard env syntax.\n\n Possible resources include (case insensitive):\n\n        pod (po), replicationcontroller (rc), deployment (deploy), daemonset (ds), statefulset (sts), cronjob (cj), replicaset (rs)",
		},
		Examples: []cli.Example{
			{
				Description: "Update deployment 'registry' with a new environment variable",
				CommandLine: "kubectl set env deployment/registry STORAGE_DIR=/local",
			},
			{
				Description: "List the environment variables defined on a deployments 'sample-build'",
				CommandLine: "kubectl set env deployment/sample-build --list",
			},
			{
				Description: "List the environment variables defined on all pods",
				CommandLine: "kubectl set env pods --all --list",
			},
			{
				Description: "Output modified deployment in YAML, and does not alter the object on the server",
				CommandLine: "kubectl set env deployment/sample-build STORAGE_DIR=/data -o yaml",
			},
			{
				Description: "Update all containers in all replication controllers in the project to have ENV=prod",
				CommandLine: "kubectl set env rc --all ENV=prod",
			},
			{
				Description: "Import environment from a secret",
				CommandLine: "kubectl set env --from=secret/mysecret deployment/myapp",
			},
			{
				Description: "Import environment from a config map with a prefix",
				CommandLine: "kubectl set env --from=configmap/myconfigmap --prefix=MYSQL_ deployment/myapp",
			},
			{
				Description: "Import specific keys from a config map",
				CommandLine: "kubectl set env --keys=my-example-key --from=configmap/myconfigmap deployment/myapp",
			},
			{
				Description: "Remove the environment variable ENV from container 'c1' in all deployment configs",
				CommandLine: "kubectl set env deployments --all --containers=\"c1\" ENV-",
			},
			{
				Description: "Remove the environment variable ENV from a deployment definition on disk and\nupdate the deployment config on the server",
				CommandLine: "kubectl set env -f deploy.json ENV-",
			},
			{
				Description: "Set some of the local shell environment into a deployment config on the server",
				CommandLine: "env | grep RAILS_ | kubectl set env -e - deployment/registry",
			},
		},
		Declare: func(cb *cli.CommandBuilder[setEnvConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[setEnvConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore())
			})

			allFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.All))
			allFlag = allFlag.Description("If true, select all resources in the namespace of the specified resource types")
			allFlag.Long("all").ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			containersFlag := cb.Flags.String(&cb.Config.Containers)
			containersFlag = containersFlag.Description("The names of containers in the selected pod templates to change - may use wildcards")
			cb.Values.Default(containersFlag, "*")
			containersFlag.Long("containers")
			containersFlag.Short('c')

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			envValue := cb.Values.Sequence(&cb.Config.Env, cli.Value.String())
			envFlag := cb.Flags.Sequence(envValue)
			envFlag = envFlag.Description("Specify a key-value pair for an environment variable to set into each container.")
			envFlag.Long("env")
			envFlag.Short('e')

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-set")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("Filename, directory, or URL to files the resource to update the env")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			fromFlag := cb.Flags.String(&cb.Config.From)
			fromFlag = fromFlag.Description("The name of a resource from which to inject environment variables")
			fromFlag.Long("from")

			keysValue := cb.Values.Sequence(&cb.Config.Keys, cli.Value.String())
			keysFlag := cb.Flags.Sequence(keysValue)
			keysFlag = keysFlag.DelimitedBy(",")
			keysFlag = keysFlag.Description("Comma-separated list of keys to import from specified resource")
			keysFlag.Long("keys")

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			listFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.List))
			listFlag = listFlag.Description("If true, display the environment and any changes in the standard format. this flag will removed when we have kubectl view env.")
			listFlag.Long("list").ImplicitValue(true)

			localFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Local))
			localFlag = localFlag.Description("If true, set env will NOT contact api-server but run locally.")
			localFlag.Long("local").ImplicitValue(true)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			overwriteFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Overwrite))
			overwriteFlag = overwriteFlag.Description("If true, allow environment to be overwritten, otherwise reject updates that overwrite existing environment.")
			cb.Values.Default(overwriteFlag, true)
			overwriteFlag.Long("overwrite").ImplicitValue(true)

			prefixFlag := cb.Flags.String(&cb.Config.Prefix)
			prefixFlag = prefixFlag.Description("Prefix to append to variable names")
			prefixFlag.Long("prefix")

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			resolveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Resolve))
			resolveFlag = resolveFlag.Description("If true, show secret or configmap references when listing variables")
			resolveFlag.Long("resolve").ImplicitValue(true)

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
		},
		Run: runSetEnv,
	}
	return cmd
}
