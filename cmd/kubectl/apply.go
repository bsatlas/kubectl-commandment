package main

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type applyConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	All                      bool
	AllowMissingTemplateKeys bool
	Cascade                  string
	DryRun                   string
	FieldManager             string
	FieldManagerState        cli.OptionalValue[cli.ValueState]
	Filename                 []string
	Force                    bool
	ForceConflicts           bool
	GracePeriod              int
	Kustomize                string
	OpenapiPatch             bool
	Output                   string
	Overwrite                bool
	Prune                    bool
	PruneAllowlist           []string
	Record                   bool
	Recursive                bool
	Selector                 string
	ServerSide               bool
	ShowManagedFields        bool
	Subresource              string
	Template                 string
	Timeout                  time.Duration
	Validate                 string
	Wait                     bool
}

func applyCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[applyConfig] {
	cmd := cli.Command[applyConfig]{
		Name: "apply",
		Description: cli.Description{
			Short: "Apply a configuration to a resource by file name or stdin",
			Long:  "Apply a configuration to a resource by file name or stdin. The resource name must be specified. This resource will be created if it doesn't exist yet. To use 'apply', always create the resource initially with either 'apply' or 'create --save-config'.\n\n JSON and YAML formats are accepted.\n\n Alpha Disclaimer: the --prune functionality is not yet complete. Do not use unless you are aware of what the current state is. See https://issues.k8s.io/34274.",
		},
		Examples: []cli.Example{
			{
				Description: "Apply the configuration in pod.json to a pod",
				CommandLine: "kubectl apply -f ./pod.json",
			},
			{
				Description: "Apply resources from a directory containing kustomization.yaml - e.g. dir/kustomization.yaml",
				CommandLine: "kubectl apply -k dir/",
			},
			{
				Description: "Apply the JSON passed into stdin to a pod",
				CommandLine: "cat pod.json | kubectl apply -f -",
			},
			{
				Description: "Apply the configuration from all files that end with '.json'",
				CommandLine: "kubectl apply -f '*.json'",
			},
			{
				Description: "Note: --prune is still in Alpha\nApply the configuration in manifest.yaml that matches label app=nginx and delete all other resources that are not in the file and match label app=nginx",
				CommandLine: "kubectl apply --prune -f manifest.yaml -l app=nginx",
			},
			{
				Description: "Apply the configuration in manifest.yaml and delete all the other config maps that are not in the file",
				CommandLine: "kubectl apply --prune -f manifest.yaml --all --prune-allowlist=core/v1/ConfigMap",
			},
		},
		Declare: func(cb *cli.CommandBuilder[applyConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			allFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.All))
			allFlag = allFlag.Description("Select all resources in the namespace of the specified resource types.")
			allFlag.Long("all").ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			cascadeFlag := cb.Flags.String(&cb.Config.Cascade)
			cascadeFlag = cascadeFlag.Description("Must be \"background\", \"orphan\", or \"foreground\". Selects the deletion cascading strategy for the dependents (e.g. Pods created by a ReplicationController). Defaults to background.")
			cb.Values.Default(cascadeFlag, "background")
			cascadeFlagLong := cascadeFlag.Long("cascade")
			cascadeFlagLong.ImplicitValue("background")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-client-side-apply")
			fieldManagerFlag.Long("field-manager")
			cb.Access.ValueState(fieldManagerFlag, &cb.Config.FieldManagerState)

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("The files, directories or URLs that contain the configurations to apply.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			forceFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Force))
			forceFlag = forceFlag.Description("If true, immediately remove resources from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
			forceFlag.Long("force").ImplicitValue(true)

			forceConflictsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ForceConflicts))
			forceConflictsFlag = forceConflictsFlag.Description("If true, server-side apply will force the changes against conflicts.")
			forceConflictsFlag.Long("force-conflicts").ImplicitValue(true)

			gracePeriodFlag := cb.Flags.Int(&cb.Config.GracePeriod)
			gracePeriodFlag = gracePeriodFlag.Description("Period of time in seconds given to the resource to terminate gracefully. Ignored if negative. Set to 1 for immediate shutdown. Can only be set to 0 when --force is true (force deletion).")
			cb.Values.Default(gracePeriodFlag, -1)
			gracePeriodFlag.Long("grace-period")

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process a kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			openapiPatchFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.OpenapiPatch))
			openapiPatchFlag = openapiPatchFlag.Description("If true, use openapi to calculate diff when the openapi presents and the resource can be found in the openapi spec. Otherwise, fall back to use baked-in types.")
			cb.Values.Default(openapiPatchFlag, true)
			openapiPatchFlag.Long("openapi-patch").ImplicitValue(true)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			overwriteFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Overwrite))
			overwriteFlag = overwriteFlag.Description("Automatically resolve conflicts between the modified and live configuration by using values from the modified configuration")
			cb.Values.Default(overwriteFlag, true)
			overwriteFlag.Long("overwrite").ImplicitValue(true)

			pruneFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Prune))
			pruneFlag = pruneFlag.Description("Automatically delete resource objects, that do not appear in the configs and are created by either apply or create --save-config. Should be used with either -l or --all.")
			pruneFlag.Long("prune").ImplicitValue(true)

			pruneAllowlistValue := cb.Values.Sequence(&cb.Config.PruneAllowlist, cli.Value.String())
			pruneAllowlistFlag := cb.Flags.Sequence(pruneAllowlistValue)
			pruneAllowlistFlag = pruneAllowlistFlag.Description("Overwrite the default allowlist with <group/version/kind> for --prune")
			pruneAllowlistFlag.Long("prune-allowlist")

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

			serverSideFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ServerSide))
			serverSideFlag = serverSideFlag.Description("If true, apply runs in the server instead of the client.")
			serverSideFlag.Long("server-side").ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			subresourceFlag := cb.Flags.String(&cb.Config.Subresource)
			subresourceFlag = subresourceFlag.Description("If specified, apply will operate on the subresource of the requested object.  Only allowed when using --server-side.")
			subresourceFlag.CompleteBy(literalCompleter[applyConfig](
				cli.CompletionCandidateValue,
				"status", "scale", "resize",
			))
			subresourceFlag.Long("subresource")

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			timeoutFlag := cb.Flags.Value(&cb.Config.Timeout, durationLanguage())
			timeoutFlag = timeoutFlag.Description("The length of time to wait before giving up on a delete, zero means determine a timeout from the size of the object")
			timeoutFlag.Long("timeout")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag = validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			waitFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Wait))
			waitFlag = waitFlag.Description("If true, wait for resources to be gone before returning. This waits for finalizers.")
			waitFlag.Long("wait").ImplicitValue(true)

			editLastApplied := applyEditLastAppliedCommand(client)
			cb.Subcommands.Command(editLastApplied)
			setLastApplied := applySetLastAppliedCommand(client)
			cb.Subcommands.Command(setLastApplied)
			viewLastApplied := applyViewLastAppliedCommand(client)
			cb.Subcommands.Command(viewLastApplied)
		},
		Run: runApply,
	}
	return cmd
}
