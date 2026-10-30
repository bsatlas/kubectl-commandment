package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/set/env"
	"k8s.io/kubectl/pkg/cmd/util"
)

type setEnvConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []envArgument
	All                      bool
	AllowMissingTemplateKeys bool
	Containers               string
	DryRun                   util.DryRunStrategy
	Env                      []envChange
	FieldManager             string
	Filename                 filenameOptions
	From                     string
	Keys                     []string
	List                     bool
	Local                    bool
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Overwrite                bool
	Prefix                   string
	Resolve                  bool
	Selector                 string
	ShowManagedFields        bool
	Template                 string
}

type envArgumentKind uint8

const (
	envArgumentResource envArgumentKind = iota
	envArgumentChange
)

type envArgument struct {
	kind     envArgumentKind
	resource string
	change   envChange
}

type envChangeKind uint8

const (
	envChangeSet envChangeKind = iota
	envChangeRemove
	envChangeStdin
)

type envChange struct {
	kind  envChangeKind
	name  string
	value string
}

func envArgumentLike(value string) bool {
	return env.IsEnvironmentArgument(value) || strings.HasSuffix(value, "-")
}

func parseEnvChange(value string) (envChange, error) {
	switch {
	case value == "-":
		change := envChange{
			kind: envChangeStdin,
		}
		return change, nil
	case env.IsEnvironmentArgument(value):
		name, envValue, _ := strings.Cut(value, "=")
		if problems := validation.IsEnvVarName(name); len(problems) != 0 {
			return envChange{}, cli.Rejection.Message(
				"environment-variable",
				fmt.Sprintf("%q is not a valid key name: %s", name, strings.Join(problems, ";")),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		change := envChange{
			kind:  envChangeSet,
			name:  name,
			value: envValue,
		}
		return change, nil
	case strings.HasSuffix(value, "-"):
		change := envChange{
			kind: envChangeRemove,
			name: strings.TrimSuffix(value, "-"),
		}
		return change, nil
	default:
		return envChange{}, cli.Rejection.Message(
			"environment-variable",
			fmt.Sprintf("unknown environment variable: %v", value),
			cli.Rejection.At(0, 0, len(value)),
		)
	}
}

var envArgumentLanguage = cli.ValueCodec[envArgument]{
	Name: "kubectl-env-argument",
	Parse: func(value string) (envArgument, error) {
		if !envArgumentLike(value) {
			argument := envArgument{
				kind:     envArgumentResource,
				resource: value,
			}
			return argument, nil
		}
		change, err := parseEnvChange(value)
		if err != nil {
			return envArgument{}, err
		}
		argument := envArgument{
			kind:   envArgumentChange,
			change: change,
		}
		return argument, nil
	},
	Format: func(argument envArgument) string {
		if argument.kind == envArgumentResource {
			return argument.resource
		}
		return formatEnvChange(argument.change)
	},
	Snapshot: cli.Value.Assign[envArgument](),
}

var envChangeLanguage = cli.ValueCodec[envChange]{
	Name:     "kubectl-env-change",
	Parse:    parseEnvChange,
	Format:   formatEnvChange,
	Snapshot: cli.Value.Assign[envChange](),
}

func formatEnvChange(change envChange) string {
	switch change.kind {
	case envChangeSet:
		return change.name + "=" + change.value
	case envChangeRemove:
		return change.name + "-"
	case envChangeStdin:
		return "-"
	default:
		return ""
	}
}

func splitEnvArguments(arguments []envArgument) ([]string, []envChange) {
	resources := make([]string, 0, len(arguments))
	changes := make([]envChange, 0, len(arguments))
	for _, argument := range arguments {
		switch argument.kind {
		case envArgumentResource:
			resources = append(resources, argument.resource)
		case envArgumentChange:
			changes = append(changes, argument.change)
		}
	}
	return resources, changes
}

func envArgumentOrderValid(arguments []envArgument) bool {
	changeSeen := false
	for _, argument := range arguments {
		if argument.kind == envArgumentChange {
			changeSeen = true
			continue
		}
		if changeSeen {
			return false
		}
	}
	return true
}

func compatibleEnvChanges(changes ...[]envChange) bool {
	set := map[string]struct{}{}
	remove := map[string]struct{}{}
	for _, group := range changes {
		for _, change := range group {
			switch change.kind {
			case envChangeSet:
				set[change.name] = struct{}{}
			case envChangeRemove:
				remove[change.name] = struct{}{}
			}
		}
	}
	for name := range remove {
		if _, ok := set[name]; ok {
			return false
		}
	}
	return true
}

func setEnvCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[setEnvConfig] {
	cmd := cli.Command[setEnvConfig]{
		Name: "env",
		Description: cli.Description{
			Short: "Update environment variables on a pod template",
			Long:  "Update environment variables on a pod template.\n\n List environment variable definitions in one or more pods, pod templates. Add, update, or remove container environment variable definitions in one or more pod templates (within replication controllers or deployment configurations). View or modify the environment variable definitions on all containers in the specified pods or pod templates, or just those that match a wildcard.\n\n If \"--env -\" is passed, environment variables can be read from STDIN using the standard env syntax.\n\n Possible resources include (case insensitive):\n\n        pod (po), replicationcontroller (rc), deployment (deploy), daemonset (ds), statefulset (sts), cronjob (cj), replicaset (rs)",
		},
		Examples: []cli.Example{
			{
				Description: "Update deployment 'registry' with a new environment variable",
				CommandLine: "kubectl set env deployment/registry STORAGE_DIR=/local",
			},
			{
				Description: "List the environment variables defined on a deployments 'sample-build'",
				CommandLine: "kubectl set env deployment/sample-build --list",
			},
			{
				Description: "List the environment variables defined on all pods",
				CommandLine: "kubectl set env pods --all --list",
			},
			{
				Description: "Output modified deployment in YAML, and does not alter the object on the server",
				CommandLine: "kubectl set env deployment/sample-build STORAGE_DIR=/data -o yaml",
			},
			{
				Description: "Update all containers in all replication controllers in the project to have ENV=prod",
				CommandLine: "kubectl set env rc --all ENV=prod",
			},
			{
				Description: "Import environment from a secret",
				CommandLine: "kubectl set env --from=secret/mysecret deployment/myapp",
			},
			{
				Description: "Import environment from a config map with a prefix",
				CommandLine: "kubectl set env --from=configmap/myconfigmap --prefix=MYSQL_ deployment/myapp",
			},
			{
				Description: "Import specific keys from a config map",
				CommandLine: "kubectl set env --keys=my-example-key --from=configmap/myconfigmap deployment/myapp",
			},
			{
				Description: "Remove the environment variable ENV from container 'c1' in all deployment configs",
				CommandLine: "kubectl set env deployments --all --containers=\"c1\" ENV-",
			},
			{
				Description: "Remove the environment variable ENV from a deployment definition on disk and\nupdate the deployment config on the server",
				CommandLine: "kubectl set env -f deploy.json ENV-",
			},
			{
				Description: "Set some of the local shell environment into a deployment config on the server",
				CommandLine: "env | grep RAILS_ | kubectl set env -e - deployment/registry",
			},
		},
		Declare: func(cb *cli.CommandBuilder[setEnvConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, envArgumentLanguage)
			resourceCompleter := resourceArgumentCompleter(
				func(cfg *setEnvConfig) (clientConfig, []string) {
					resources, _ := splitEnvArguments(cfg.Arguments)
					return cfg.Client, resources
				},
				[]string{"daemonsets", "deployments", "pods", "cronjobs", "replicasets", "replicationcontrollers", "statefulsets"},
				resourceArgumentSingle,
			)
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[setEnvConfig]) {
				form.Sequence(arguments, "RESOURCE|KEY=VALUE|KEY-", cli.Count.ZeroOrMore()).
					CompleteBy(func(ctx context.Context, cfg *setEnvConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
						_, changes := splitEnvArguments(cfg.Arguments)
						if len(changes) != 0 {
							return cli.CompletionContribution{}, nil
						}
						return resourceCompleter(ctx, cfg, request)
					}).
					From(client)
			})

			all := cb.Values.Bool(&cb.Config.All)
			allFlag := cb.Flags.ValueFor(all)
			cb.Values.ResolveWith(allFlag, cli.Value.KeepLast())
			allFlag.Description("If true, select all resources in the namespace of the specified resource types")
			allFlag.Long("all").ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			cb.Values.ResolveWith(allowMissingTemplateKeysFlag, cli.Value.KeepLast())
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			containersFlag := cb.Flags.String(&cb.Config.Containers)
			cb.Values.ResolveWith(containersFlag, cli.Value.KeepLast())
			containersFlag.Description("The names of containers in the selected pod templates to change - may use wildcards")
			cb.Values.Default(containersFlag, "*")
			containersFlag.Long("containers")
			containersFlag.Short('c').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, util.DryRunNone)
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue(util.DryRunClient)

			envValue := cb.Values.Sequence(&cb.Config.Env, envChangeLanguage)
			envFlag := cb.Flags.Sequence(envValue)
			envFlag.Description("Specify a key-value pair for an environment variable to set into each container.")
			envFlag.Long("env")
			envFlag.Short('e').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-set")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename.Groups, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.Description("Filename, directory, or URL to files the resource to update the env")
			filenameFlag.CompleteWith(cli.CompletionNativePaths)
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			from := cb.Values.String(&cb.Config.From)
			fromFlag := cb.Flags.ValueFor(from)
			cb.Values.ResolveWith(fromFlag, cli.Value.KeepLast())
			fromFlag.Description("The name of a resource from which to inject environment variables")
			fromFlag.Long("from")

			keysValue := cb.Values.Sequence(&cb.Config.Keys, cli.Value.String())
			keysFlag := cb.Flags.Sequence(keysValue)
			keysFlag.DelimitedBy(",")
			keysFlag.Description("Comma-separated list of keys to import from specified resource")
			keysFlag.Long("keys")

			kustomize := cb.Values.String(&cb.Config.Filename.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			cb.Values.ResolveWith(kustomizeFlag, cli.Value.KeepLast())
			kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.CompleteWith(cli.CompletionNativeDirectories)
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			list := cb.Values.Bool(&cb.Config.List)
			listFlag := cb.Flags.ValueFor(list)
			cb.Values.ResolveWith(listFlag, cli.Value.KeepLast())
			listFlag.Description("If true, display the environment and any changes in the standard format. this flag will removed when we have kubectl view env.")
			listFlag.Long("list").ImplicitValue(true)

			local := cb.Values.Bool(&cb.Config.Local)
			localFlag := cb.Flags.ValueFor(local)
			cb.Values.ResolveWith(localFlag, cli.Value.KeepLast())
			localFlag.Description("If true, set env will NOT contact api-server but run locally.")
			localFlag.Long("local").ImplicitValue(true)

			output := cb.Values.Value(&cb.Config.Output, printOutputLanguage)
			outputFlag := cb.Flags.ValueFor(output)
			cb.Values.ResolveWith(outputFlag, cli.Value.KeepLast())
			outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.CompleteBy(completePrintOutputFormat[setEnvConfig])
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)
			cb.Access.ValueState(output, &cb.Config.OutputState)

			overwriteFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Overwrite))
			cb.Values.ResolveWith(overwriteFlag, cli.Value.KeepLast())
			overwriteFlag.Description("If true, allow environment to be overwritten, otherwise reject updates that overwrite existing environment.")
			cb.Values.Default(overwriteFlag, true)
			overwriteFlag.Long("overwrite").ImplicitValue(true)

			prefixFlag := cb.Flags.String(&cb.Config.Prefix)
			cb.Values.ResolveWith(prefixFlag, cli.Value.KeepLast())
			prefixFlag.Description("Prefix to append to variable names")
			prefixFlag.Long("prefix")

			recursive := cb.Values.Bool(&cb.Config.Filename.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive)
			cb.Values.ResolveWith(recursiveFlag, cli.Value.KeepLast())
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			resolveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Resolve))
			cb.Values.ResolveWith(resolveFlag, cli.Value.KeepLast())
			resolveFlag.Description("If true, show secret or configmap references when listing variables")
			resolveFlag.Long("resolve").ImplicitValue(true)

			selector := cb.Values.String(&cb.Config.Selector)
			selectorFlag := cb.Flags.ValueFor(selector)
			cb.Values.ResolveWith(selectorFlag, cli.Value.KeepLast())
			selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			cb.Values.ResolveWith(showManagedFieldsFlag, cli.Value.KeepLast())
			showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			template := cb.Values.String(&cb.Config.Template)
			templateFlag := cb.Flags.ValueFor(template)
			cb.Values.ResolveWith(templateFlag, cli.Value.KeepLast())
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.CompleteWith(cli.CompletionNativeFiles)
			templateFlag.Long("template")

			declarePrintOutputTemplateRule(cb, outputFlag, templateFlag)

			cb.Rule.When(cb.When.Equal(all, true)).Equal(selector, "")
			cb.Rule.Check("kubectl-set-env-filename-kustomize", func(_ context.Context, cfg *setEnvConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Filename.Filenames()) == 0 || cfg.Filename.Kustomize == "", nil
			}).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
			cb.Rule.When(cb.When.Equal(recursive, true)).Forbid(kustomize)
			cb.Rule.Check("kubectl-set-env-argument-order", func(_ context.Context, cfg *setEnvConfig, _ cli.Dependencies) (bool, error) {
				return envArgumentOrderValid(cfg.Arguments), nil
			}).DependsOn(arguments.OptionalDependency())
			cb.Rule.Check("kubectl-set-env-source", func(_ context.Context, cfg *setEnvConfig, _ cli.Dependencies) (bool, error) {
				resources, _ := splitEnvArguments(cfg.Arguments)
				return len(resources) != 0 || len(cfg.Filename.Filenames()) != 0, nil
			}).
				DependsOn(arguments.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency())
			cb.Rule.When(cb.When.Equal(list, true)).Equal(output, "")
			cb.Rule.Check("kubectl-set-env-keys-from", func(_ context.Context, cfg *setEnvConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Keys) == 0 || cfg.From != "", nil
			}).
				DependsOn(keysValue.OptionalDependency()).
				DependsOn(from.OptionalDependency())
			cb.Rule.Check("kubectl-set-env-change-conflict", func(_ context.Context, cfg *setEnvConfig, _ cli.Dependencies) (bool, error) {
				_, positional := splitEnvArguments(cfg.Arguments)
				return compatibleEnvChanges(cfg.Env, positional), nil
			}).
				DependsOn(envValue.OptionalDependency()).
				DependsOn(arguments.OptionalDependency())
			cb.Rule.Check("kubectl-set-env-local-dry-run", func(_ context.Context, cfg *setEnvConfig, _ cli.Dependencies) (bool, error) {
				return !cfg.Local || cfg.DryRun != util.DryRunServer, nil
			}).
				DependsOn(local.OptionalDependency()).
				DependsOn(dryRun.OptionalDependency())
		},
		Run: runSetEnv,
	}
	return cmd
}
