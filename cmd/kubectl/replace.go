package main

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type replaceConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	Cascade                  string
	DryRun                   string
	FieldManager             string
	Filename                 []string
	Force                    bool
	GracePeriod              int
	Kustomize                string
	Output                   string
	Raw                      string
	Recursive                bool
	SaveConfig               bool
	ShowManagedFields        bool
	Subresource              string
	Template                 string
	Timeout                  time.Duration
	Validate                 string
	Wait                     bool
}

func replaceCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[replaceConfig] {
	cmd := cli.Command[replaceConfig]{
		Name: "replace",
		Description: cli.Description{
			Short: "Replace a resource by file name or stdin",
			Long:  "Replace a resource by file name or stdin.\n\n JSON and YAML formats are accepted. If replacing an existing resource, the complete resource spec must be provided. This can be obtained by\n\n        $ kubectl get TYPE NAME -o yaml",
		},
		Examples: []cli.Example{
			{
				Description: "Replace a pod using the data in pod.json",
				CommandLine: "kubectl replace -f ./pod.json",
			},
			{
				Description: "Replace a pod based on the JSON passed into stdin",
				CommandLine: "cat pod.json | kubectl replace -f -",
			},
			{
				Description: "Update a single-container pod's image version (tag) to v4",
				CommandLine: "kubectl get pod mypod -o yaml | sed 's/\\(image: myimage\\):.*$/\\1:v4/' | kubectl replace -f -",
			},
			{
				Description: "Force replace, delete and then re-create the resource",
				CommandLine: "kubectl replace --force -f ./pod.json",
			},
		},
		Declare: func(cb *cli.CommandBuilder[replaceConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[replaceConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore())
			})

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
			cb.Values.Default(fieldManagerFlag, "kubectl-replace")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("The files, directories or URLs that contain the configurations to replace.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			forceFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Force))
			forceFlag = forceFlag.Description("If true, immediately remove resources from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
			forceFlag.Long("force").ImplicitValue(true)

			gracePeriodFlag := cb.Flags.Int(&cb.Config.GracePeriod)
			gracePeriodFlag = gracePeriodFlag.Description("Period of time in seconds given to the resource to terminate gracefully. Ignored if negative. Set to 1 for immediate shutdown. Can only be set to 0 when --force is true (force deletion).")
			cb.Values.Default(gracePeriodFlag, -1)
			gracePeriodFlag.Long("grace-period")

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process a kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			rawFlag := cb.Flags.String(&cb.Config.Raw)
			rawFlag = rawFlag.Description("Raw URI to PUT to the server.  Uses the transport specified by the kubeconfig file.")
			rawFlag.Long("raw")

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

			subresourceFlag := cb.Flags.String(&cb.Config.Subresource)
			subresourceFlag = subresourceFlag.Description("If specified, replace will operate on the subresource of the requested object.")
			subresourceFlag.CompleteBy(literalCompleter[replaceConfig](
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
		},
		Run: runReplace,
	}
	return cmd
}
