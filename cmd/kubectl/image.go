package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type setImageConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	All                      bool
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	Filename                 []string
	Kustomize                string
	Local                    bool
	Output                   string
	Record                   bool
	Recursive                bool
	Selector                 string
	ShowManagedFields        bool
	Template                 string
}

func setImageCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[setImageConfig] {
	cmd := cli.Command[setImageConfig]{
		Name: "image",
		Description: cli.Description{
			Short: "Update the image of a pod template",
			Long:  "Update existing container image(s) of resources.\n\n Possible resources include (case insensitive):\n\n        pod (po), replicationcontroller (rc), deployment (deploy), daemonset (ds), statefulset (sts), cronjob (cj), replicaset (rs)",
		},
		Examples: []cli.Example{
			{
				Description: "Set a deployment's nginx container image to 'nginx:1.9.1', and its busybox container image to 'busybox'",
				CommandLine: "kubectl set image deployment/nginx busybox=busybox nginx=nginx:1.9.1",
			},
			{
				Description: "Update all deployments' and rc's nginx container's image to 'nginx:1.9.1'",
				CommandLine: "kubectl set image deployments,rc nginx=nginx:1.9.1 --all",
			},
			{
				Description: "Update image of all containers of daemonset abc to 'nginx:1.9.1'",
				CommandLine: "kubectl set image daemonset abc *=nginx:1.9.1",
			},
			{
				Description: "Print result (in yaml format) of updating nginx container image from local file, without hitting the server",
				CommandLine: "kubectl set image -f path/to/file.yaml nginx=nginx:1.9.1 --local -o yaml",
			},
		},
		Declare: func(cb *cli.CommandBuilder[setImageConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[setImageConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *setImageConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						[]string{"daemonsets", "deployments", "pods", "cronjobs", "replicasets", "replicationcontrollers", "statefulsets"},
						resourceArgumentSingle,
					)).
					From(client)
			})

			allFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.All))
			allFlag = allFlag.Description("Select all resources, in the namespace of the specified resource types")
			allFlag.Long("all").ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-set")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("Filename, directory, or URL to files identifying the resource to get from a server.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			localFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Local))
			localFlag = localFlag.Description("If true, set image will NOT contact api-server but run locally.")
			localFlag.Long("local").ImplicitValue(true)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			recordFlag = recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag = recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

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
		Run: runSetImage,
	}
	return cmd
}
