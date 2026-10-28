package kubectl

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type setResourcesConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Resources                []string
	All                      bool
	AllowMissingTemplateKeys bool
	Containers               string
	DryRun                   util.DryRunStrategy
	FieldManager             string
	Filename                 []string
	Kustomize                string
	Limits                   v1.ResourceList
	Local                    bool
	Output                   string
	Record                   bool
	Recursive                bool
	Requests                 v1.ResourceList
	Selector                 string
	ShowManagedFields        bool
	Template                 string
}

func setResourcesCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[setResourcesConfig] {
	cmd := cli.Command[setResourcesConfig]{
		Name: "resources",
		Description: cli.Description{
			Short: "Update resource requests/limits on objects with pod templates",
			Long:  "Specify compute resource requirements (CPU, memory) for any resource that defines a pod template.  If a pod is successfully scheduled, it is guaranteed the amount of resource requested, but may burst up to its specified limits.\n\n For each compute resource, if a limit is specified and a request is omitted, the request will default to the limit.\n\n Possible resources include (case insensitive): Use \"kubectl api-resources\" for a complete list of supported resources..",
		},
		Examples: []cli.Example{
			{
				Description: "Set a deployments nginx container cpu limits to \"200m\" and memory to \"512Mi\"",
				CommandLine: "kubectl set resources deployment nginx -c=nginx --limits=cpu=200m,memory=512Mi",
			},
			{
				Description: "Set the resource request and limits for all containers in nginx",
				CommandLine: "kubectl set resources deployment nginx --limits=cpu=200m,memory=512Mi --requests=cpu=100m,memory=256Mi",
			},
			{
				Description: "Remove the resource requests for resources on containers in nginx",
				CommandLine: "kubectl set resources deployment nginx --limits=cpu=0,memory=0 --requests=cpu=0,memory=0",
			},
			{
				Description: "Print the result (in yaml format) of updating nginx container limits from a local, without hitting the server",
				CommandLine: "kubectl set resources -f path/to/file.yaml --limits=cpu=200m,memory=512Mi --local -o yaml",
			},
		},
		Declare: func(cb *cli.CommandBuilder[setResourcesConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			resources := cb.Values.Sequence(&cb.Config.Resources, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[setResourcesConfig]) {
				form.Sequence(resources, "RESOURCE", cli.Count.ZeroOrMore())
			})

			all := cb.Values.Bool(&cb.Config.All)
			allFlag := cb.Flags.ValueFor(all)
			allFlag.Description("Select all resources, in the namespace of the specified resource types")
			allFlag.Long("all").ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			containersFlag := cb.Flags.String(&cb.Config.Containers)
			containersFlag.Description("The names of containers in the selected pod templates to change, all containers are selected by default - may use wildcards")
			cb.Values.Default(containersFlag, "*")
			containersFlag.Long("containers")
			containersFlag.Short('c')

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

			limits := cb.Values.Value(&cb.Config.Limits, resourceListLanguage)
			limitsFlag := cb.Flags.ValueFor(limits)
			limitsFlag.Description("The resource requirement requests for this container.  For example, 'cpu=100m,memory=256Mi'.  Note that server side components may assign requests depending on the server configuration, such as limit ranges.")
			limitsFlag.Long("limits")

			local := cb.Values.Bool(&cb.Config.Local)
			localFlag := cb.Flags.ValueFor(local)
			localFlag.Description("If true, set resources will NOT contact api-server but run locally.")
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

			requests := cb.Values.Value(&cb.Config.Requests, resourceListLanguage)
			requestsFlag := cb.Flags.ValueFor(requests)
			requestsFlag.Description("The resource requirement requests for this container.  For example, 'cpu=100m,memory=256Mi'.  Note that server side components may assign requests depending on the server configuration, such as limit ranges.")
			requestsFlag.Long("requests")

			selector := cb.Values.String(&cb.Config.Selector)
			selectorFlag := cb.Flags.ValueFor(selector)
			selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			cb.Rule.When(cb.When.Equal(all, true)).Equal(selector, "")
			cb.Rule.AtMostOne(filenameValue, kustomize)
			cb.Rule.When(cb.When.Equal(recursive, true)).Forbid(kustomize)
			cb.Rule.When(cb.When.Equal(local, true)).Forbid(resources)
			cb.Rule.AtLeastOne(resources, filenameValue).And(kustomize)
			cb.Rule.Check("kubectl-set-resources-update", func(_ context.Context, cfg *setResourcesConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Limits) > 0 || len(cfg.Requests) > 0, nil
			}).
				DependsOn(limits.OptionalDependency()).
				DependsOn(requests.OptionalDependency())
			cb.Rule.Check("kubectl-set-resources-local-dry-run", func(_ context.Context, cfg *setResourcesConfig, _ cli.Dependencies) (bool, error) {
				return !cfg.Local || cfg.DryRun != util.DryRunServer, nil
			}).
				DependsOn(local.OptionalDependency()).
				DependsOn(dryRun.OptionalDependency())
		},
		Run: runSetResources,
	}
	return cmd
}

var resourceListLanguage = cli.ValueCodec[v1.ResourceList]{
	Name: "kubectl-resource-list",
	Parse: func(value string) (v1.ResourceList, error) {
		if value == "" {
			return nil, nil
		}
		result := v1.ResourceList{}
		for statement := range strings.SplitSeq(value, ",") {
			name, quantityText, found := strings.Cut(statement, "=")
			if !found || strings.Contains(quantityText, "=") {
				return nil, cli.Rejection.Message(
					"resource-list",
					fmt.Sprintf("invalid argument syntax %v, expected <resource>=<value>", statement),
					cli.Rejection.At(0, 0, len(value)),
				)
			}
			quantity, err := resource.ParseQuantity(quantityText)
			if err != nil {
				return nil, cli.Rejection.Message(
					"resource-list",
					err.Error(),
					cli.Rejection.At(0, 0, len(value)),
				)
			}
			result[v1.ResourceName(name)] = quantity
		}
		return result, nil
	},
	Format: func(value v1.ResourceList) string {
		keys := slices.Sorted(maps.Keys(value))
		parts := make([]string, 0, len(keys))
		for _, name := range keys {
			quantity := value[name]
			parts = append(parts, string(name)+"="+quantity.String())
		}
		return strings.Join(parts, ",")
	},
	Snapshot: func(value v1.ResourceList) v1.ResourceList {
		if value == nil {
			return nil
		}
		clone := make(v1.ResourceList, len(value))
		for name, quantity := range value {
			clone[name] = quantity.DeepCopy()
		}
		return clone
	},
}
