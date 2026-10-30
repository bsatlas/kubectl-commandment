package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type taintOperands struct {
	Resources []string
	Add       []v1.Taint
	Remove    []v1.Taint
}

func splitTaintArguments(args []string) ([]string, []string, error) {
	var resources []string
	var taints []string
	metTaint := false
	for _, arg := range args {
		isTaint := strings.Contains(arg, "=") || strings.Contains(arg, ":") || strings.HasSuffix(arg, "-")
		if !metTaint && isTaint {
			metTaint = true
		}
		if metTaint {
			if !isTaint {
				return nil, nil, fmt.Errorf("all resources must be specified before taint changes: %s", arg)
			}
			taints = append(taints, arg)
			continue
		}
		resources = append(resources, arg)
	}
	return resources, taints, nil
}

func parseTaintChanges(specs []string) ([]v1.Taint, []v1.Taint, error) {
	var add []v1.Taint
	var remove []v1.Taint
	unique := make(map[v1.TaintEffect]sets.Set[string])
	for _, spec := range specs {
		if value, ok := strings.CutSuffix(spec, "-"); ok {
			taint, err := parseTaint(value)
			if err != nil {
				return nil, nil, err
			}
			taintToRemove := v1.Taint{
				Key:    taint.Key,
				Effect: taint.Effect,
			}
			remove = append(remove, taintToRemove)
			continue
		}
		taint, err := parseTaint(spec)
		if err != nil {
			return nil, nil, err
		}
		if taint.Effect == "" {
			return nil, nil, fmt.Errorf("invalid taint spec: %v", spec)
		}
		if unique[taint.Effect] == nil {
			unique[taint.Effect] = sets.New[string]()
		}
		if unique[taint.Effect].Has(taint.Key) {
			return nil, nil, fmt.Errorf("duplicated taints with the same key and effect: %v", taint)
		}
		unique[taint.Effect].Insert(taint.Key)
		add = append(add, taint)
	}
	return add, remove, nil
}

func parseTaint(spec string) (v1.Taint, error) {
	var taint v1.Taint
	keyValue, effect, hasEffect := strings.Cut(spec, ":")
	if !hasEffect {
		taint.Key = keyValue
	} else {
		if strings.Contains(effect, ":") {
			return taint, fmt.Errorf("invalid taint spec: %v", spec)
		}
		taint.Effect = v1.TaintEffect(effect)
		if taint.Effect != v1.TaintEffectNoSchedule && taint.Effect != v1.TaintEffectPreferNoSchedule && taint.Effect != v1.TaintEffectNoExecute {
			return taint, fmt.Errorf("invalid taint effect: %v, unsupported taint effect", taint.Effect)
		}
		key, value, hasValue := strings.Cut(keyValue, "=")
		if strings.Contains(value, "=") {
			return taint, fmt.Errorf("invalid taint spec: %v", spec)
		}
		taint.Key = key
		if hasValue {
			taint.Value = value
			if errs := validation.IsValidLabelValue(taint.Value); len(errs) > 0 {
				return taint, fmt.Errorf("invalid taint spec: %v, %s", spec, strings.Join(errs, "; "))
			}
		}
	}
	if errs := validation.IsQualifiedName(taint.Key); len(errs) > 0 {
		return taint, fmt.Errorf("invalid taint spec: %v, %s", spec, strings.Join(errs, "; "))
	}
	return taint, nil
}

func parseTaintOperands(arguments []string) (taintOperands, error) {
	resources, changes, err := splitTaintArguments(arguments)
	if err != nil {
		return taintOperands{}, err
	}
	add, remove, err := parseTaintChanges(changes)
	if err != nil {
		return taintOperands{}, err
	}
	for _, added := range add {
		for _, removed := range remove {
			if added.Key == removed.Key && (removed.Effect == "" || added.Effect == removed.Effect) {
				return taintOperands{}, fmt.Errorf("can not both modify and remove the following taint(s) in the same command: %s=%s", removed.Key, removed.Effect)
			}
		}
	}
	input := taintOperands{
		Resources: resources,
		Add:       add,
		Remove:    remove,
	}
	return input, nil
}

type taintConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	Operands                 taintOperands
	All                      bool
	AllowMissingTemplateKeys bool
	DryRun                   util.DryRunStrategy
	FieldManager             string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Overwrite                bool
	Selector                 string
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func taintCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[taintConfig] {
	cmd := cli.Command[taintConfig]{
		Name: "taint",
		Description: cli.Description{
			Short: "Update the taints on one or more nodes",
			Long:  "Update the taints on one or more nodes.\n\n  *  A taint consists of a key, value, and effect. As an argument here, it is expressed as key=value:effect.\n  *  The key must begin with a letter or number, and may contain letters, numbers, hyphens, dots, and underscores, up to 253 characters.\n  *  Optionally, the key can begin with a DNS subdomain prefix and a single '/', like example.com/my-app.\n  *  The value is optional. If given, it must begin with a letter or number, and may contain letters, numbers, hyphens, dots, and underscores, up to 63 characters.\n  *  The effect must be NoSchedule, PreferNoSchedule or NoExecute.\n  *  Currently taint can only apply to node.",
		},
		Examples: []cli.Example{
			{
				Description: "Update node 'foo' with a taint with key 'dedicated' and value 'special-user' and effect 'NoSchedule'\nIf a taint with that key and effect already exists, its value is replaced as specified",
				CommandLine: "kubectl taint nodes foo dedicated=special-user:NoSchedule",
			},
			{
				Description: "Remove from node 'foo' the taint with key 'dedicated' and effect 'NoSchedule' if one exists",
				CommandLine: "kubectl taint nodes foo dedicated:NoSchedule-",
			},
			{
				Description: "Remove from node 'foo' all the taints with key 'dedicated'",
				CommandLine: "kubectl taint nodes foo dedicated-",
			},
			{
				Description: "Add a taint with key 'dedicated' on nodes having label myLabel=X",
				CommandLine: "kubectl taint node -l myLabel=X  dedicated=foo:PreferNoSchedule",
			},
			{
				Description: "Add to node 'foo' a taint with key 'bar' and no value",
				CommandLine: "kubectl taint nodes foo bar:NoSchedule",
			},
		},
		Declare: func(cb *cli.CommandBuilder[taintConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[taintConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *taintConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						[]string{"node"},
						resourceArgumentRepeated,
					)).
					From(client)
			})
			input := cb.Values.Derive(&cb.Config.Operands, cli.Value.ComputedBy(func(_ context.Context, _ *taintConfig, dependencies cli.Dependencies) (taintOperands, error) {
				values, ok := dependencies.Get(arguments)
				if !ok {
					return taintOperands{}, fmt.Errorf("taint arguments dependency is unavailable")
				}
				return parseTaintOperands(values)
			}).From(arguments))

			all := cb.Values.Bool(&cb.Config.All)
			allFlag := cb.Flags.ValueFor(all)
			cb.Values.ResolveWith(allFlag, cli.Value.KeepLast())
			allFlag.Description("Select all nodes in the cluster")
			allFlag.Long("all").ImplicitValue(true)

			declarePrintFlags(cb, &cb.Config.Output, &cb.Config.OutputState, &cb.Config.Template, &cb.Config.AllowMissingTemplateKeys, &cb.Config.ShowManagedFields)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRun, util.DryRunNone)
			dryRunFlag.Long("dry-run").ImplicitValue(util.DryRunClient)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-taint")
			fieldManagerFlag.Long("field-manager")

			overwriteFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Overwrite))
			cb.Values.ResolveWith(overwriteFlag, cli.Value.KeepLast())
			overwriteFlag.Description("If true, allow taints to be overwritten, otherwise reject taint updates that overwrite existing taints.")
			overwriteFlag.Long("overwrite").ImplicitValue(true)

			selector := cb.Values.String(&cb.Config.Selector)
			selectorFlag := cb.Flags.ValueFor(selector)
			cb.Values.ResolveWith(selectorFlag, cli.Value.KeepLast())
			selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			cb.Rule.When(cb.When.Equal(all, true)).Equal(selector, "")
			cb.Rule.Check("kubectl-taint-resource", func(_ context.Context, cfg *taintConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Operands.Resources) > 0, nil
			}).DependsOn(input.OptionalDependency())
			cb.Rule.Check("kubectl-taint-update", func(_ context.Context, cfg *taintConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Operands.Add)+len(cfg.Operands.Remove) > 0, nil
			}).DependsOn(input.OptionalDependency())
			cb.Rule.Check("kubectl-taint-target", func(_ context.Context, cfg *taintConfig, _ cli.Dependencies) (bool, error) {
				if cfg.All || cfg.Selector != "" {
					return true, nil
				}
				return len(cfg.Operands.Resources) >= 2, nil
			}).
				DependsOn(input.OptionalDependency()).
				DependsOn(all.OptionalDependency()).
				DependsOn(selector.OptionalDependency())

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")
		},
		Run: runTaint,
	}
	return cmd
}
