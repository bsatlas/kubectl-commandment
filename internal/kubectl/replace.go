package kubectl

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type replaceConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	Cascade                  deletionPropagation
	DryRun                   string
	FieldManager             string
	Filename                 []string
	Force                    bool
	GracePeriod              int
	RequestedGrace           int
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
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			cascade := cb.Values.Value(&cb.Config.Cascade, deletionPropagationLanguage)
			backgroundCascade := deletionPropagation{
				Propagation: v1.DeletePropagationBackground,
			}
			cb.Values.Default(cascade, backgroundCascade)
			cascadeFlag := cb.Flags.ValueFor(cascade)
			cascadeFlag.Description("Must be \"background\", \"orphan\", or \"foreground\". Selects the deletion cascading strategy for the dependents (e.g. Pods created by a ReplicationController). Defaults to background.")
			cascadeFlag.Long("cascade").ImplicitValue(backgroundCascade)

			dryRun := cb.Values.String(&cb.Config.DryRun)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-replace")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.DelimitedBy(",")
			filenameFlag.Description("The files, directories or URLs that contain the configurations to replace.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			force := cb.Values.Bool(&cb.Config.Force)
			forceFlag := cb.Flags.ValueFor(force)
			forceFlag.Description("If true, immediately remove resources from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
			forceFlag.Long("force").ImplicitValue(true)

			requestedGrace := cb.Values.Int(&cb.Config.RequestedGrace)
			cb.Values.Default(requestedGrace, -1)
			gracePeriodFlag := cb.Flags.ValueFor(requestedGrace)
			gracePeriodFlag.Description("Period of time in seconds given to the resource to terminate gracefully. Ignored if negative. Set to 1 for immediate shutdown. Can only be set to 0 when --force is true (force deletion).")
			gracePeriodFlag.Long("grace-period")

			kustomize := cb.Values.String(&cb.Config.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			kustomizeFlag.Description("Process a kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			output := cb.Values.String(&cb.Config.Output)
			outputFlag := cb.Flags.ValueFor(output)
			outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			raw := cb.Values.String(&cb.Config.Raw)
			rawFlag := cb.Flags.ValueFor(raw)
			rawFlag.Description("Raw URI to PUT to the server.  Uses the transport specified by the kubeconfig file.")
			rawFlag.Long("raw")

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

			subresourceFlag := cb.Flags.String(&cb.Config.Subresource)
			subresourceFlag.Description("If specified, replace will operate on the subresource of the requested object.")
			subresourceFlag.CompleteBy(literalCompleter[replaceConfig](
				cli.CompletionCandidateValue,
				"status", "scale", "resize",
			))
			subresourceFlag.Long("subresource")

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			timeout := cb.Values.Value(&cb.Config.Timeout, durationLanguage())
			timeoutFlag := cb.Flags.ValueFor(timeout)
			timeoutFlag.Description("The length of time to wait before giving up on a delete, zero means determine a timeout from the size of the object")
			timeoutFlag.Long("timeout")

			cb.Rule.When(cb.When.Equal(recursive, true)).Equal(kustomize, "")
			cb.Rule.Check("kubectl-replace-source", func(_ context.Context, cfg *replaceConfig, _ cli.Dependencies) (bool, error) {
				hasFilename := len(cfg.Filename) > 0
				hasKustomize := cfg.Kustomize != ""
				return hasFilename != hasKustomize, nil
			}).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
			cb.Rule.When(cb.When.AtLeast(requestedGrace, 0)).Equal(force, true)
			cb.Rule.When(cb.When.NotEqual(timeout, time.Duration(0))).Equal(force, true)
			cb.Rule.When(cb.When.Equal(force, true)).Equal(dryRun, "none")
			rawReplace := cb.When.NotEqual(raw, "")
			cb.Rule.When(rawReplace).ItemCount(filenameValue, cli.Count.Exactly(1))
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
				if cfg.Raw == "" || len(cfg.Filename) == 0 {
					return true, nil
				}
				filename := cfg.Filename[0]
				return !strings.HasPrefix(filename, "http://") && !strings.HasPrefix(filename, "https://"), nil
			}).
				DependsOn(raw.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency())
			cb.Values.Derive(&cb.Config.GracePeriod, cli.Value.ComputedBy(func(_ context.Context, _ *replaceConfig, dependencies cli.Dependencies) (int, error) {
				gracePeriod, ok := dependencies.Get(requestedGrace)
				if !ok {
					return 0, fmt.Errorf("replace grace-period dependency is unavailable")
				}
				if gracePeriod == 0 {
					return 1, nil
				}
				return gracePeriod, nil
			}).From(requestedGrace))
			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			wait := cb.Values.Bool(&cb.Config.Wait)
			waitFlag := cb.Flags.ValueFor(wait)
			waitFlag.Description("If true, wait for resources to be gone before returning. This waits for finalizers.")
			waitFlag.Long("wait").ImplicitValue(true)
			cb.Values.Derive(&cb.Config.WaitForDeletion, cli.Value.ComputedBy(func(_ context.Context, _ *replaceConfig, dependencies cli.Dependencies) (bool, error) {
				waitForDeletion, waitOK := dependencies.Get(wait)
				gracePeriod, graceOK := dependencies.Get(requestedGrace)
				if !waitOK || !graceOK {
					return false, fmt.Errorf("replace deletion wait dependency is unavailable")
				}
				return waitForDeletion || gracePeriod == 0, nil
			}).From(wait, requestedGrace))
		},
		Run: runReplace,
	}
	return cmd
}
