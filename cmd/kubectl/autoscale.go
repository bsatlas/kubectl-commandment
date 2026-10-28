package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type autoscaleConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	CPU                      string
	CPUPercent               int32
	DryRun                   string
	FieldManager             string
	Filename                 []string
	Kustomize                string
	Max                      int32
	Memory                   string
	Min                      int32
	Name                     string
	Output                   string
	Record                   bool
	Recursive                bool
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
}

func autoscaleCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[autoscaleConfig] {
	cmd := cli.Command[autoscaleConfig]{
		Name: "autoscale",
		Description: cli.Description{
			Short: "Auto-scale a deployment, replica set, stateful set, or replication controller",
			Long:  "Creates an autoscaler that automatically chooses and sets the number of pods that run in a Kubernetes cluster. The command will attempt to use the autoscaling/v2 API first, in case of an error, it will fall back to autoscaling/v1 API.\n\n Looks up a deployment, replica set, stateful set, or replication controller by name and creates an autoscaler that uses the given resource as a reference. An autoscaler can automatically increase or decrease number of pods deployed within the system as needed.",
		},
		Examples: []cli.Example{
			{
				Description: "Auto scale a deployment \"foo\", with the number of pods between 2 and 10, no target CPU utilization specified so a default autoscaling policy will be used",
				CommandLine: "kubectl autoscale deployment foo --min=2 --max=10",
			},
			{
				Description: "Auto scale a replication controller \"foo\", with the number of pods between 1 and 5, target CPU utilization at 80%",
				CommandLine: "kubectl autoscale rc foo --max=5 --cpu=80%",
			},
			{
				Description: "Auto scale a deployment \"bar\", with the number of pods between 3 and 6, target average CPU of 500m and memory of 200Mi",
				CommandLine: "kubectl autoscale deployment bar --min=3 --max=6 --cpu=500m --memory=200Mi",
			},
			{
				Description: "Auto scale a deployment \"bar\", with the number of pods between 2 and 8, target CPU utilization 60% and memory utilization 70%",
				CommandLine: "kubectl autoscale deployment bar --min=2 --max=8 --cpu=60% --memory=70%",
			},
		},
		Declare: func(cb *cli.CommandBuilder[autoscaleConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[autoscaleConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *autoscaleConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						[]string{"deployment", "replicaset", "replicationcontroller", "statefulset"},
						resourceArgumentRepeated,
					)).
					From(client)
			})

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			cpuFlag := cb.Flags.String(&cb.Config.CPU)
			cpuFlag = cpuFlag.Description("Target CPU utilization over all the pods. When specified as a percentage (e.g.\"70%\" for 70% of requested CPU) it will target average utilization. When specified as quantity (e.g.\"500m\" for 500 milliCPU) it will target average value. Value without units is treated as a quantity with miliCPU being the unit (e.g.\"500\" is \"500m\").")
			cpuFlag.Long("cpu")

			cpuPercentFlag := cb.Flags.Value(&cb.Config.CPUPercent, int32Language())
			cpuPercentFlag = cpuPercentFlag.Description("The target average CPU utilization (represented as a percent of requested CPU) over all the pods. If it's not specified or negative, a default autoscaling policy will be used.")
			cb.Values.Default(cpuPercentFlag, int32(-1))
			cpuPercentFlag = cpuPercentFlag.Hidden().Release(preexistingDeprecation("Use --cpu with percentage or resource quantity format (e.g., '70%' for utilization or '500m' for milliCPU)."))
			cpuPercentFlag.Long("cpu-percent")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-autoscale")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("Filename, directory, or URL to files identifying the resource to autoscale.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			maxFlag := cb.Flags.Value(&cb.Config.Max, int32Language())
			maxFlag = maxFlag.Description("The upper limit for the number of pods that can be set by the autoscaler. Required.")
			cb.Values.Default(maxFlag, int32(-1))
			maxFlag.Long("max")

			memoryFlag := cb.Flags.String(&cb.Config.Memory)
			memoryFlag = memoryFlag.Description("Target memory utilization over all the pods. When specified  as a percentage (e.g.\"60%\" for 60% of requested memory) it will target average utilization. When specified as quantity (e.g.\"200Mi\" for 200 MiB, \"1Gi\" for 1 GiB) it will target average value. Value without units is treated as a quantity with mebibytes being the unit (e.g.\"200\" is \"200Mi\").")
			memoryFlag.Long("memory")

			minFlag := cb.Flags.Value(&cb.Config.Min, int32Language())
			minFlag = minFlag.Description("The lower limit for the number of pods that can be set by the autoscaler. If it's not specified or negative, the server will apply a default value.")
			cb.Values.Default(minFlag, int32(-1))
			minFlag.Long("min")

			nameFlag := cb.Flags.String(&cb.Config.Name)
			nameFlag = nameFlag.Description("The name for the newly created object. If not specified, the name of the input resource will be used.")
			nameFlag.Long("name")

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

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")
		},
		Run: runAutoscale,
	}
	return cmd
}
