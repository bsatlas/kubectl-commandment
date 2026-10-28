package kubectl

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type autoscaleMetricTarget struct {
	Text        string
	Quantity    resource.Quantity
	Utilization int32
	Type        v2.MetricTargetType
}

type autoscaleConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	CPU                      autoscaleMetricTarget
	CPUPercent               int32
	DryRun                   string
	FieldManager             string
	Filename                 []string
	Kustomize                string
	Max                      int32
	Memory                   autoscaleMetricTarget
	Min                      int32
	Name                     string
	Output                   string
	Record                   bool
	Recursive                bool
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
}

func autoscaleMetricTargetLanguage(resourceName string) cli.ValueCodec[autoscaleMetricTarget] {
	name := "kubectl-autoscale-" + resourceName + "-target"
	language := cli.ValueCodec[autoscaleMetricTarget]{
		Name: name,
		Parse: func(text string) (autoscaleMetricTarget, error) {
			if text == "" {
				return autoscaleMetricTarget{}, nil
			}
			quantity, utilization, metricType, err := parseAutoscaleResource(text, resourceName)
			if err != nil {
				return autoscaleMetricTarget{}, cli.Rejection.Message(name, err.Error(), cli.Rejection.At(0, 0, len(text)))
			}
			target := autoscaleMetricTarget{
				Text:        text,
				Quantity:    quantity,
				Utilization: utilization,
				Type:        metricType,
			}
			return target, nil
		},
		Format: func(target autoscaleMetricTarget) string {
			return target.Text
		},
		Snapshot: func(target autoscaleMetricTarget) autoscaleMetricTarget {
			target.Quantity = target.Quantity.DeepCopy()
			return target
		},
	}
	return language
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
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			cpu := cb.Values.Value(&cb.Config.CPU, autoscaleMetricTargetLanguage("cpu"))
			cpuFlag := cb.Flags.ValueFor(cpu)
			cpuFlag.Description("Target CPU utilization over all the pods. When specified as a percentage (e.g.\"70%\" for 70% of requested CPU) it will target average utilization. When specified as quantity (e.g.\"500m\" for 500 milliCPU) it will target average value. Value without units is treated as a quantity with miliCPU being the unit (e.g.\"500\" is \"500m\").")
			cpuFlag.Long("cpu")

			cpuPercent := cb.Values.Value(&cb.Config.CPUPercent, int32Language())
			cpuPercentFlag := cb.Flags.ValueFor(cpuPercent)
			cpuPercentFlag.Description("The target average CPU utilization (represented as a percent of requested CPU) over all the pods. If it's not specified or negative, a default autoscaling policy will be used.")
			cb.Values.Default(cpuPercent, int32(-1))
			cpuPercentFlag.Hidden().Release(preexistingDeprecation("Use --cpu with percentage or resource quantity format (e.g., '70%' for utilization or '500m' for milliCPU)."))
			cpuPercentFlag.Long("cpu-percent")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-autoscale")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.DelimitedBy(",")
			filenameFlag.Description("Filename, directory, or URL to files identifying the resource to autoscale.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomize := cb.Values.String(&cb.Config.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			maxPods := cb.Values.Value(&cb.Config.Max, int32Language())
			maxFlag := cb.Flags.ValueFor(maxPods)
			maxFlag.Description("The upper limit for the number of pods that can be set by the autoscaler. Required.")
			maxFlag.Long("max")

			memory := cb.Values.Value(&cb.Config.Memory, autoscaleMetricTargetLanguage("memory"))
			memoryFlag := cb.Flags.ValueFor(memory)
			memoryFlag.Description("Target memory utilization over all the pods. When specified  as a percentage (e.g.\"60%\" for 60% of requested memory) it will target average utilization. When specified as quantity (e.g.\"200Mi\" for 200 MiB, \"1Gi\" for 1 GiB) it will target average value. Value without units is treated as a quantity with mebibytes being the unit (e.g.\"200\" is \"200Mi\").")
			memoryFlag.Long("memory")

			minPods := cb.Values.Value(&cb.Config.Min, int32Language())
			minFlag := cb.Flags.ValueFor(minPods)
			minFlag.Description("The lower limit for the number of pods that can be set by the autoscaler. If it's not specified or negative, the server will apply a default value.")
			cb.Values.Default(minPods, int32(-1))
			minFlag.Long("min")

			nameFlag := cb.Flags.String(&cb.Config.Name)
			nameFlag.Description("The name for the newly created object. If not specified, the name of the input resource will be used.")
			nameFlag.Long("name")

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

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			cb.Rule.Require(maxPods)
			cb.Rule.AtLeast(maxPods, int32(1))
			cb.Rule.Compare.AtLeast(maxPods, minPods)
			cb.Rule.Check("kubectl-autoscale-cpu-input", func(_ context.Context, cfg *autoscaleConfig, _ cli.Dependencies) (bool, error) {
				return cfg.CPUPercent <= 0 || cfg.CPU.Text == "", nil
			}).
				DependsOn(cpuPercent.OptionalDependency()).
				DependsOn(cpu.OptionalDependency())
			cb.Rule.When(cb.When.Equal(recursive, true)).Equal(kustomize, "")
			cb.Rule.Check("kubectl-autoscale-source", func(_ context.Context, cfg *autoscaleConfig, _ cli.Dependencies) (bool, error) {
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
		Run: runAutoscale,
	}
	return cmd
}

func parseAutoscaleResource(input, resourceName string) (resource.Quantity, int32, v2.MetricTargetType, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return resource.Quantity{}, 0, "", fmt.Errorf("empty input")
	}
	if trimmed, ok := strings.CutSuffix(input, "%"); ok {
		value, err := strconv.ParseInt(trimmed, 10, 32)
		if err != nil || value < 0 {
			return resource.Quantity{}, 0, "", fmt.Errorf("invalid percentage value: %s", trimmed)
		}
		return resource.Quantity{}, int32(value), v2.UtilizationMetricType, nil
	}
	if value, err := strconv.ParseFloat(input, 64); err == nil {
		unit := ""
		switch resourceName {
		case "cpu":
			unit = "m"
		case "memory":
			unit = "Mi"
		default:
			return resource.Quantity{}, 0, "", fmt.Errorf("unsupported resource type: %v", resourceName)
		}
		quantity, err := resource.ParseQuantity(fmt.Sprintf("%g%s", value, unit))
		if err != nil {
			return resource.Quantity{}, 0, "", err
		}
		return quantity, 0, v2.AverageValueMetricType, nil
	}
	quantity, err := resource.ParseQuantity(input)
	if err != nil {
		return resource.Quantity{}, 0, "", fmt.Errorf("invalid resource %s value: %s", resourceName, input)
	}
	return quantity, 0, v2.AverageValueMetricType, nil
}
