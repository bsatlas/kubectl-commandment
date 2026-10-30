package main

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type replaceConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	Cascade                  deletionPropagation
	DryRun                   util.DryRunStrategy
	FieldManager             string
	Filename                 filenameOptions
	Force                    bool
	GracePeriod              int
	RequestedGrace           int
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Raw                      string
	SaveConfig               bool
	ShowManagedFields        bool
	Subresource              string
	Template                 string
	Timeout                  time.Duration
	Validate                 string
	Wait                     bool
	WaitForDeletion          bool
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
			cb.Values.ResolveWith(allowMissingTemplateKeysFlag, cli.Value.KeepLast())
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			cascade := cb.Values.Value(&cb.Config.Cascade, deletionPropagationLanguage)
			backgroundCascade := deletionPropagation{
				Propagation: v1.DeletePropagationBackground,
			}
			cb.Values.Default(cascade, backgroundCascade)
			cascadeFlag := cb.Flags.ValueFor(cascade)
			cb.Values.ResolveWith(cascadeFlag, cli.Value.KeepLast())
			cascadeFlag.Description("Must be \"background\", \"orphan\", or \"foreground\". Selects the deletion cascading strategy for the dependents (e.g. Pods created by a ReplicationController). Defaults to background.")
			cascadeFlag.Long("cascade").ImplicitValue(backgroundCascade)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRun, util.DryRunNone)
			dryRunFlag.Long("dry-run").ImplicitValue(util.DryRunClient)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-replace")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename.Groups, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.Description("The files, directories or URLs that contain the configurations to replace.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			force := cb.Values.Bool(&cb.Config.Force)
			forceFlag := cb.Flags.ValueFor(force)
			cb.Values.ResolveWith(forceFlag, cli.Value.KeepLast())
			forceFlag.Description("If true, immediately remove resources from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
			forceFlag.Long("force").ImplicitValue(true)

			requestedGrace := cb.Values.Int(&cb.Config.RequestedGrace)
			cb.Values.Default(requestedGrace, -1)
			gracePeriodFlag := cb.Flags.ValueFor(requestedGrace)
			cb.Values.ResolveWith(gracePeriodFlag, cli.Value.KeepLast())
			gracePeriodFlag.Description("Period of time in seconds given to the resource to terminate gracefully. Ignored if negative. Set to 1 for immediate shutdown. Can only be set to 0 when --force is true (force deletion).")
			gracePeriodFlag.Long("grace-period")

			kustomize := cb.Values.String(&cb.Config.Filename.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			cb.Values.ResolveWith(kustomizeFlag, cli.Value.KeepLast())
			kustomizeFlag.Description("Process a kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			output := cb.Values.Value(&cb.Config.Output, printOutputLanguage)
			outputFlag := cb.Flags.ValueFor(output)
			cb.Values.ResolveWith(outputFlag, cli.Value.KeepLast())
			outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.CompleteBy(completePrintOutputFormat[replaceConfig])
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)
			cb.Access.ValueState(output, &cb.Config.OutputState)

			raw := cb.Values.String(&cb.Config.Raw)
			rawFlag := cb.Flags.ValueFor(raw)
			cb.Values.ResolveWith(rawFlag, cli.Value.KeepLast())
			rawFlag.Description("Raw URI to PUT to the server.  Uses the transport specified by the kubeconfig file.")
			rawFlag.Long("raw")

			recursive := cb.Values.Bool(&cb.Config.Filename.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive)
			cb.Values.ResolveWith(recursiveFlag, cli.Value.KeepLast())
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			cb.Values.ResolveWith(saveConfigFlag, cli.Value.KeepLast())
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			cb.Values.ResolveWith(showManagedFieldsFlag, cli.Value.KeepLast())
			showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			subresourceFlag := cb.Flags.String(&cb.Config.Subresource)
			cb.Values.ResolveWith(subresourceFlag, cli.Value.KeepLast())
			subresourceFlag.Description("If specified, replace will operate on the subresource of the requested object.")
			subresourceFlag.CompleteBy(literalCompleter[replaceConfig](
				cli.CompletionCandidateValue,
				"status", "scale", "resize",
			))
			subresourceFlag.Long("subresource")

			templateFlag := cb.Flags.String(&cb.Config.Template)
			cb.Values.ResolveWith(templateFlag, cli.Value.KeepLast())
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.CompleteWith(cli.CompletionNativeFiles)
			templateFlag.Long("template")
			declarePrintOutputTemplateRule(cb, outputFlag, templateFlag)

			timeout := cb.Values.Value(&cb.Config.Timeout, durationLanguage())
			timeoutFlag := cb.Flags.ValueFor(timeout)
			cb.Values.ResolveWith(timeoutFlag, cli.Value.KeepLast())
			timeoutFlag.Description("The length of time to wait before giving up on a delete, zero means determine a timeout from the size of the object")
			timeoutFlag.Long("timeout")

			cb.Rule.When(cb.When.Equal(recursive, true)).Equal(kustomize, "")
			cb.Rule.Check("kubectl-replace-source", func(_ context.Context, cfg *replaceConfig, _ cli.Dependencies) (bool, error) {
				hasFilename := len(cfg.Filename.Filenames()) > 0
				hasKustomize := cfg.Filename.Kustomize != ""
				return hasFilename != hasKustomize, nil
			}).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
			graceRequiresForce := cb.When.AtLeast(requestedGrace, 0)
			cb.Rule.When(graceRequiresForce).Require(force)
			cb.Rule.When(graceRequiresForce).Equal(force, true)
			timeoutRequiresForce := cb.When.NotEqual(timeout, time.Duration(0))
			cb.Rule.When(timeoutRequiresForce).Require(force)
			cb.Rule.When(timeoutRequiresForce).Equal(force, true)
			cb.Rule.When(cb.When.Equal(force, true)).Equal(dryRun, util.DryRunNone)
			rawReplace := cb.When.NotEqual(raw, "")
			cb.Rule.Check("kubectl-replace-raw-filename-count", func(_ context.Context, cfg *replaceConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Raw == "" || len(cfg.Filename.Filenames()) == 1, nil
			}).
				DependsOn(raw.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency())
			cb.Rule.When(rawReplace).Equal(recursive, false)
			cb.Rule.When(rawReplace).Equal(output, "")
			cb.Rule.Check("kubectl-replace-raw-uri", func(_ context.Context, cfg *replaceConfig, _ cli.Dependencies) (bool, error) {
				if cfg.Raw == "" {
					return true, nil
				}
				_, err := url.ParseRequestURI(cfg.Raw)
				return err == nil, nil
			}).DependsOn(raw.OptionalDependency())
			cb.Rule.Check("kubectl-replace-raw-filename", func(_ context.Context, cfg *replaceConfig, _ cli.Dependencies) (bool, error) {
				if cfg.Raw == "" || len(cfg.Filename.Filenames()) == 0 {
					return true, nil
				}
				filename := cfg.Filename.Filenames()[0]
				return !strings.HasPrefix(filename, "http://") && !strings.HasPrefix(filename, "https://"), nil
			}).
				DependsOn(raw.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency())
			cb.Values.Derive(&cb.Config.GracePeriod, cli.Value.ComputedBy(func(_ context.Context, _ *replaceConfig, dependencies cli.Dependencies) (int, error) {
				gracePeriod, _ := dependencies.Get(requestedGrace)
				if gracePeriod == 0 {
					return 1, nil
				}
				return gracePeriod, nil
			}).From(requestedGrace))
			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			wait := cb.Values.Bool(&cb.Config.Wait)
			waitFlag := cb.Flags.ValueFor(wait)
			cb.Values.ResolveWith(waitFlag, cli.Value.KeepLast())
			waitFlag.Description("If true, wait for resources to be gone before returning. This waits for finalizers.")
			waitFlag.Long("wait").ImplicitValue(true)
			cb.Values.Derive(&cb.Config.WaitForDeletion, cli.Value.ComputedBy(func(_ context.Context, _ *replaceConfig, dependencies cli.Dependencies) (bool, error) {
				waitForDeletion, _ := dependencies.Get(wait)
				gracePeriod, _ := dependencies.Get(requestedGrace)
				return waitForDeletion || gracePeriod == 0, nil
			}).From(wait.OptionalDependency(), requestedGrace))
		},
		Run: runReplace,
	}
	return cmd
}
