package main

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type applyConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	All                      bool
	ApplySet                 string
	ApplySetEnabled          bool
	AllowMissingTemplateKeys bool
	Cascade                  string
	DryRun                   util.DryRunStrategy
	FieldManager             string
	Filename                 filenameOptions
	Force                    bool
	ForceConflicts           bool
	GracePeriod              int
	OpenapiPatch             bool
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Overwrite                bool
	Prune                    bool
	PruneAllowlist           []string
	Record                   bool
	Selector                 string
	ServerSide               bool
	ShowManagedFields        bool
	Subresource              string
	Template                 string
	Timeout                  time.Duration
	Validate                 string
	Wait                     bool
}

var applySetFeatureLanguage = cli.ValueCodec[bool]{
	Name: "kubectl-applyset-feature",
	Parse: func(value string) (bool, error) {
		return strings.EqualFold(value, "true"), nil
	},
	Format:   strconv.FormatBool,
	Snapshot: cli.Value.Assign[bool](),
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

			declarePrintFlags(
				cb,
				&cb.Config.Output,
				&cb.Config.OutputState,
				&cb.Config.Template,
				&cb.Config.AllowMissingTemplateKeys,
				&cb.Config.ShowManagedFields,
			)

			all := cb.Values.Bool(&cb.Config.All)
			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			fieldManager := cb.Values.String(&cb.Config.FieldManager)
			filename := cb.Values.Sequence(&cb.Config.Filename.Groups, filenameListLanguage)
			force := cb.Values.Bool(&cb.Config.Force)
			forceConflicts := cb.Values.Bool(&cb.Config.ForceConflicts)
			kustomize := cb.Values.String(&cb.Config.Filename.Kustomize)
			prune := cb.Values.Bool(&cb.Config.Prune)
			recursive := cb.Values.Bool(&cb.Config.Filename.Recursive)
			selector := cb.Values.String(&cb.Config.Selector)
			serverSide := cb.Values.Bool(&cb.Config.ServerSide)
			subresource := cb.Values.String(&cb.Config.Subresource)

			allFlag := cb.Flags.ValueFor(all)
			cb.Values.ResolveWith(allFlag, cli.Value.KeepLast())
			allFlag.Description("Select all resources in the namespace of the specified resource types.")
			allFlag.Long("all").ImplicitValue(true)

			applySet := cb.Values.String(&cb.Config.ApplySet)
			applySetEnabled := cb.Values.Value(&cb.Config.ApplySetEnabled, applySetFeatureLanguage)
			cb.Values.Environment(applySetEnabled, string(util.ApplySet))

			applySetFlag := cb.Flags.ValueFor(applySet)
			cb.Values.ResolveWith(applySetFlag, cli.Value.KeepLast())
			applySetFlag.Description("[alpha] The name of the ApplySet that tracks which resources are being managed, for the purposes of determining what to prune. Live resources that are part of the ApplySet but have been removed from the provided configs will be deleted. Format: [RESOURCE][.GROUP]/NAME. A Secret will be used if no resource or group is specified.")
			applySetFlag.AvailableWhen(cb.When.Equal(applySetEnabled, true))
			applySetFlag.Long("applyset")

			cb.Profiles.Switch(applySetEnabled, func(cases *cli.GrammarProfileCases[applyConfig, bool]) {
				cases.Case(true, func(active *cli.CommandBuilder[applyConfig]) {
					pruneFlag := active.Flags.ValueFor(prune)
					active.Values.ResolveWith(pruneFlag, cli.Value.KeepLast())
					pruneFlag.Description("Automatically delete previously applied resource objects that do not appear in the provided configs. For alpha1, use with either -l or --all. For alpha2, use with --applyset.")
					pruneFlag.Long("prune").ImplicitValue(true)
				})
				cases.Default(func(active *cli.CommandBuilder[applyConfig]) {
					pruneFlag := active.Flags.ValueFor(prune)
					active.Values.ResolveWith(pruneFlag, cli.Value.KeepLast())
					pruneFlag.Description("Automatically delete resource objects, that do not appear in the configs and are created by either apply or create --save-config. Should be used with either -l or --all.")
					pruneFlag.Long("prune").ImplicitValue(true)
				})
			})

			cascadeFlag := cb.Flags.String(&cb.Config.Cascade)
			cb.Values.ResolveWith(cascadeFlag, cli.Value.KeepLast())
			cascadeFlag.Description("Must be \"background\", \"orphan\", or \"foreground\". Selects the deletion cascading strategy for the dependents (e.g. Pods created by a ReplicationController). Defaults to background.")
			cb.Values.Default(cascadeFlag, "background")
			cascadeFlagLong := cascadeFlag.Long("cascade")
			cascadeFlagLong.ImplicitValue("background")

			dryRunFlag := cb.Flags.ValueFor(dryRun)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, util.DryRunNone)
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue(util.DryRunClient)

			cb.Values.OptionalComputedDefault(
				fieldManager,
				cli.Value.OptionallyComputedBy(func(_ context.Context, _ *applyConfig, dependencies cli.Dependencies) (cli.ValueContribution[string], error) {
					server, _ := dependencies.Get(serverSide)
					if server {
						return cli.ContributeValue("kubectl"), nil
					}
					return cli.OmitValue[string](), nil
				}).From(serverSide.OptionalDependency()),
			)
			cb.Values.Default(fieldManager, "kubectl-client-side-apply")
			fieldManagerFlag := cb.Flags.ValueFor(fieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			fieldManagerFlag.Long("field-manager")

			filenameFlag := cb.Flags.Sequence(filename)
			filenameFlag.Description("The files, directories or URLs that contain the configurations to apply.")
			filenameFlag.CompleteWith(cli.CompletionNativePaths)
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			forceFlag := cb.Flags.ValueFor(force)
			cb.Values.ResolveWith(forceFlag, cli.Value.KeepLast())
			forceFlag.Description("If true, immediately remove resources from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
			forceFlag.Long("force").ImplicitValue(true)

			forceConflictsFlag := cb.Flags.ValueFor(forceConflicts)
			cb.Values.ResolveWith(forceConflictsFlag, cli.Value.KeepLast())
			forceConflictsFlag.Description("If true, server-side apply will force the changes against conflicts.")
			forceConflictsFlag.Long("force-conflicts").ImplicitValue(true)

			gracePeriodFlag := cb.Flags.Int(&cb.Config.GracePeriod)
			cb.Values.ResolveWith(gracePeriodFlag, cli.Value.KeepLast())
			gracePeriodFlag.Description("Period of time in seconds given to the resource to terminate gracefully. Ignored if negative. Set to 1 for immediate shutdown. Can only be set to 0 when --force is true (force deletion).")
			cb.Values.Default(gracePeriodFlag, -1)
			gracePeriodFlag.Long("grace-period")

			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			cb.Values.ResolveWith(kustomizeFlag, cli.Value.KeepLast())
			kustomizeFlag.Description("Process a kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.CompleteWith(cli.CompletionNativeDirectories)
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			openapiPatchFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.OpenapiPatch))
			cb.Values.ResolveWith(openapiPatchFlag, cli.Value.KeepLast())
			openapiPatchFlag.Description("If true, use openapi to calculate diff when the openapi presents and the resource can be found in the openapi spec. Otherwise, fall back to use baked-in types.")
			cb.Values.Default(openapiPatchFlag, true)
			openapiPatchFlag.Long("openapi-patch").ImplicitValue(true)

			overwriteFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Overwrite))
			cb.Values.ResolveWith(overwriteFlag, cli.Value.KeepLast())
			overwriteFlag.Description("Automatically resolve conflicts between the modified and live configuration by using values from the modified configuration")
			cb.Values.Default(overwriteFlag, true)
			overwriteFlag.Long("overwrite").ImplicitValue(true)

			pruneAllowlistValue := cb.Values.Sequence(&cb.Config.PruneAllowlist, cli.Value.String())
			pruneAllowlistFlag := cb.Flags.Sequence(pruneAllowlistValue)
			pruneAllowlistFlag.Description("Overwrite the default allowlist with <group/version/kind> for --prune")
			pruneAllowlistFlag.Long("prune-allowlist")

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			cb.Values.ResolveWith(recordFlag, cli.Value.KeepLast())
			recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursiveFlag := cb.Flags.ValueFor(recursive)
			cb.Values.ResolveWith(recursiveFlag, cli.Value.KeepLast())
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			selectorFlag := cb.Flags.ValueFor(selector)
			cb.Values.ResolveWith(selectorFlag, cli.Value.KeepLast())
			selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			serverSideFlag := cb.Flags.ValueFor(serverSide)
			cb.Values.ResolveWith(serverSideFlag, cli.Value.KeepLast())
			serverSideFlag.Description("If true, apply runs in the server instead of the client.")
			serverSideFlag.Long("server-side").ImplicitValue(true)

			subresourceFlag := cb.Flags.ValueFor(subresource)
			cb.Values.ResolveWith(subresourceFlag, cli.Value.KeepLast())
			subresourceFlag.Description("If specified, apply will operate on the subresource of the requested object.  Only allowed when using --server-side.")
			subresourceFlag.CompleteBy(literalCompleter[applyConfig](
				cli.CompletionCandidateValue,
				"status", "scale", "resize",
			))
			subresourceFlag.Long("subresource")

			timeoutFlag := cb.Flags.Value(&cb.Config.Timeout, durationLanguage())
			cb.Values.ResolveWith(timeoutFlag, cli.Value.KeepLast())
			timeoutFlag.Description("The length of time to wait before giving up on a delete, zero means determine a timeout from the size of the object")
			timeoutFlag.Long("timeout")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			waitFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Wait))
			cb.Values.ResolveWith(waitFlag, cli.Value.KeepLast())
			waitFlag.Description("If true, wait for resources to be gone before returning. This waits for finalizers.")
			waitFlag.Long("wait").ImplicitValue(true)

			cb.Rule.Check("kubectl-apply-filename-kustomize", func(_ context.Context, cfg *applyConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Filename.Filenames()) == 0 || cfg.Filename.Kustomize == "", nil
			}).
				DependsOn(filename.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
			cb.Rule.Check("kubectl-apply-source", func(_ context.Context, cfg *applyConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Filename.Filenames()) != 0 || cfg.Filename.Kustomize != "", nil
			}).
				DependsOn(filename.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
			cb.Rule.When(cb.When.Equal(recursive, true)).Forbid(kustomize)
			cb.Rule.When(cb.When.Equal(forceConflicts, true)).Require(serverSide)
			cb.Rule.When(cb.When.Equal(forceConflicts, true)).Equal(serverSide, true)
			cb.Rule.When(cb.When.Equal(serverSide, true)).Equal(force, false)
			cb.Rule.When(cb.When.Equal(all, true)).Equal(selector, "")
			cb.Rule.When(cb.When.NotEqual(subresource, "")).Require(serverSide)
			cb.Rule.When(cb.When.NotEqual(subresource, "")).Equal(serverSide, true)
			cb.Rule.When(cb.When.Equal(dryRun, util.DryRunClient)).Forbid(serverSide)
			cb.Rule.When(cb.When.Equal(dryRun, util.DryRunServer)).Forbid(force)
			cb.Rule.When(cb.When.Equal(prune, true)).Equal(force, false)
			cb.Rule.Check("kubectl-apply-prune-server-side", func(_ context.Context, cfg *applyConfig, _ cli.Dependencies) (bool, error) {
				return !cfg.Prune || cfg.ApplySet != "" || !cfg.ServerSide, nil
			}).
				DependsOn(prune.OptionalDependency()).
				DependsOn(applySet.OptionalDependency()).
				DependsOn(serverSide.OptionalDependency())
			cb.Rule.Check("kubectl-apply-prune-selection", func(_ context.Context, cfg *applyConfig, _ cli.Dependencies) (bool, error) {
				return !cfg.Prune || cfg.ApplySet != "" || cfg.All || cfg.Selector != "", nil
			}).
				DependsOn(prune.OptionalDependency()).
				DependsOn(applySet.OptionalDependency()).
				DependsOn(all.OptionalDependency()).
				DependsOn(selector.OptionalDependency())
			selected := cb.When.NotEqual(applySet, "")
			cb.Rule.When(selected).Require(prune)
			cb.Rule.When(selected).Equal(prune, true)
			cb.Rule.When(selected).Equal(all, false)
			cb.Rule.When(selected).Equal(selector, "")
			cb.Rule.When(selected).Forbid(pruneAllowlistValue)

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
