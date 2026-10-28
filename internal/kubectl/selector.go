package kubectl

import (
	"context"
	"fmt"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type setSelectorConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Resources                []string
	Selector                 v1.LabelSelector
	All                      bool
	AllowMissingTemplateKeys bool
	DryRun                   util.DryRunStrategy
	FieldManager             string
	Filename                 []string
	Local                    bool
	Output                   string
	Record                   bool
	Recursive                bool
	ResourceVersion          string
	ShowManagedFields        bool
	Template                 string
}

func setSelectorCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[setSelectorConfig] {
	cmd := cli.Command[setSelectorConfig]{
		Name: "selector",
		Description: cli.Description{
			Short: "Set the selector on a resource",
			Long:  "Set the selector on a resource. Note that the new selector will overwrite the old selector if the resource had one prior to the invocation of 'set selector'.\n\n A selector must begin with a letter or number, and may contain letters, numbers, hyphens, dots, and underscores, up to 63 characters. If --resource-version is specified, then updates will use this resource version, otherwise the existing resource-version will be used. Note: currently selectors can only be set on Service objects.",
		},
		Examples: []cli.Example{
			{
				Description: "Set the labels and selector before creating a deployment/service pair",
				CommandLine: "kubectl create service clusterip my-svc --clusterip=\"None\" -o yaml --dry-run=client | kubectl set selector --local -f - 'environment=qa' -o yaml | kubectl create -f -\n  kubectl create deployment my-dep --image=nginx -o yaml --dry-run=client | kubectl label --local -f - environment=qa -o yaml | kubectl create -f -",
			},
		},
		Declare: func(cb *cli.CommandBuilder[setSelectorConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			resources := cb.Values.Sequence(&cb.Config.Resources, cli.Value.String())
			selector := cb.Values.Value(&cb.Config.Selector, selectorLanguage)
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[setSelectorConfig]) {
				form.Sequence(resources, "RESOURCE", cli.Count.ZeroOrMore()).Minimal()
				form.ValueFor(selector, "SELECTOR")
			})

			allFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.All))
			allFlag.Description("Select all resources in the namespace of the specified resource types")
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
			filenameFlag.Description("identifying the resource.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			local := cb.Values.Bool(&cb.Config.Local)
			localFlag := cb.Flags.ValueFor(local)
			localFlag.Description("If true, annotation will NOT contact api-server but run locally.")
			localFlag.Long("local").ImplicitValue(true)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			cb.Values.Default(recursiveFlag, true)
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			resourceVersionFlag := cb.Flags.String(&cb.Config.ResourceVersion)
			resourceVersionFlag.Description("If non-empty, the selectors update will only succeed if this is the current resource-version for the object. Only valid when specifying a single resource.")
			resourceVersionFlag.Long("resource-version")

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			cb.Rule.When(cb.When.Equal(local, true)).Forbid(resources)
			cb.Rule.AtLeastOne(resources, filenameValue)
			cb.Rule.Check("kubectl-set-selector-local-dry-run", func(_ context.Context, cfg *setSelectorConfig, _ cli.Dependencies) (bool, error) {
				return !cfg.Local || cfg.DryRun != util.DryRunServer, nil
			}).
				DependsOn(local.OptionalDependency()).
				DependsOn(dryRun.OptionalDependency())
		},
		Run: runSetSelector,
	}
	return cmd
}

var selectorLanguage = cli.ValueCodec[v1.LabelSelector]{
	Name: "kubectl-label-selector",
	Parse: func(value string) (v1.LabelSelector, error) {
		selector, err := v1.ParseToLabelSelector(value)
		if err != nil {
			return v1.LabelSelector{}, cli.Rejection.Message(
				"selector",
				fmt.Sprintf("invalid selector %q: %v", value, err),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return *selector, nil
	},
	Format: func(value v1.LabelSelector) string {
		return v1.FormatLabelSelector(&value)
	},
	Snapshot: func(value v1.LabelSelector) v1.LabelSelector {
		return *value.DeepCopy()
	},
}
