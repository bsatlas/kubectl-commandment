package main

import (
	"context"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

var patchTypeLanguage = cli.ValueCodec[types.PatchType]{
	Name: "patch-type",
	Parse: func(value string) (types.PatchType, error) {
		switch strings.ToLower(value) {
		case "":
			return types.StrategicMergePatchType, nil
		case "json":
			return types.JSONPatchType, nil
		case "merge":
			return types.MergePatchType, nil
		case "strategic":
			return types.StrategicMergePatchType, nil
		default:
			return "", cli.Rejection.Message(
				"patch-type",
				"--type must be one of [json merge strategic]",
				cli.Rejection.At(0, 0, len(value)),
			)
		}
	},
	Format: func(value types.PatchType) string {
		switch value {
		case types.JSONPatchType:
			return "json"
		case types.MergePatchType:
			return "merge"
		case types.StrategicMergePatchType:
			return "strategic"
		default:
			return string(value)
		}
	},
	Snapshot: func(value types.PatchType) types.PatchType { return value },
}

type patchConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	Filename                 []string
	Kustomize                string
	Local                    bool
	Output                   string
	Patch                    string
	PatchFile                string
	Record                   bool
	Recursive                bool
	ShowManagedFields        bool
	Subresource              string
	Template                 string
	Type                     types.PatchType
}

func patchCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[patchConfig] {
	cmd := cli.Command[patchConfig]{
		Name: "patch",
		Description: cli.Description{
			Short: "Update fields of a resource",
			Long:  "Update fields of a resource using strategic merge patch, a JSON merge patch, or a JSON patch.\n\n JSON and YAML formats are accepted.\n\n Note: Strategic merge patch is not supported for custom resources.",
		},
		Examples: []cli.Example{
			{
				Description: "Partially update a node using a strategic merge patch, specifying the patch as JSON",
				CommandLine: "kubectl patch node k8s-node-1 -p '{\"spec\":{\"unschedulable\":true}}'",
			},
			{
				Description: "Partially update a node using a strategic merge patch, specifying the patch as YAML",
				CommandLine: "kubectl patch node k8s-node-1 -p $'spec:\\n unschedulable: true'",
			},
			{
				Description: "Partially update a node identified by the type and name specified in \"node.json\" using strategic merge patch",
				CommandLine: "kubectl patch -f node.json -p '{\"spec\":{\"unschedulable\":true}}'",
			},
			{
				Description: "Update a container's image; spec.containers[*].name is required because it's a merge key",
				CommandLine: "kubectl patch pod valid-pod -p '{\"spec\":{\"containers\":[{\"name\":\"kubernetes-serve-hostname\",\"image\":\"new image\"}]}}'",
			},
			{
				Description: "Update a container's image using a JSON patch with positional arrays",
				CommandLine: "kubectl patch pod valid-pod --type='json' -p='[{\"op\": \"replace\", \"path\": \"/spec/containers/0/image\", \"value\":\"new image\"}]'",
			},
			{
				Description: "Update a deployment's replicas through the 'scale' subresource using a merge patch",
				CommandLine: "kubectl patch deployment nginx-deployment --subresource='scale' --type='merge' -p '{\"spec\":{\"replicas\":2}}'",
			},
		},
		Declare: func(cb *cli.CommandBuilder[patchConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[patchConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *patchConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						nil,
						resourceArgumentRepeated,
					)).
					From(client)
			})

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			dryRun := cb.Values.String(&cb.Config.DryRun)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRun, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-patch")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.DelimitedBy(",")
			filenameFlag.Description("Filename, directory, or URL to files identifying the resource to update")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomize := cb.Values.String(&cb.Config.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			local := cb.Values.Bool(&cb.Config.Local)
			localFlag := cb.Flags.ValueFor(local)
			localFlag.Description("If true, patch will operate on the content of the file, not the server-side resource.")
			localFlag.Long("local").ImplicitValue(true)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			patch := cb.Values.String(&cb.Config.Patch)
			patchFlag := cb.Flags.ValueFor(patch)
			patchFlag.Description("The patch to be applied to the resource JSON file.")
			patchFlag.Long("patch")
			patchFlag.Short('p')

			patchFile := cb.Values.String(&cb.Config.PatchFile)
			patchFileFlag := cb.Flags.ValueFor(patchFile)
			patchFileFlag.Description("A file containing a patch to be applied to the resource.")
			patchFileFlag.Long("patch-file")

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

			subresourceFlag := cb.Flags.String(&cb.Config.Subresource)
			subresourceFlag.Description("If specified, patch will operate on the subresource of the requested object.")
			subresourceFlag.CompleteBy(literalCompleter[patchConfig](
				cli.CompletionCandidateValue,
				"status", "scale", "resize",
			))
			subresourceFlag.Long("subresource")

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			patchType := cb.Values.Value(&cb.Config.Type, patchTypeLanguage)
			typeFlag := cb.Flags.ValueFor(patchType)
			typeFlag.Description("The type of patch being provided; one of [json merge strategic]")
			cb.Values.Default(patchType, types.StrategicMergePatchType)
			typeFlag.Long("type")

			cb.Rule.Check("kubectl-patch-input", func(_ context.Context, cfg *patchConfig, _ cli.Dependencies) (bool, error) {
				hasPatch := cfg.Patch != ""
				hasPatchFile := cfg.PatchFile != ""
				return hasPatch != hasPatchFile, nil
			}).
				DependsOn(patch.OptionalDependency()).
				DependsOn(patchFile.OptionalDependency())
			cb.Rule.When(cb.When.Equal(recursive, true)).Equal(kustomize, "")
			cb.Rule.Check("kubectl-patch-source", func(_ context.Context, cfg *patchConfig, _ cli.Dependencies) (bool, error) {
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
			cb.Rule.When(cb.When.Equal(local, true)).Forbid(arguments)
			cb.Rule.When(cb.When.Equal(local, true)).NotEqual(dryRun, "server")
		},
		Run: runPatch,
	}
	return cmd
}
