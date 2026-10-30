package main

import (
	"context"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var quotaHardLanguage = cli.ValueCodec[string]{
	Name: "resource-quota-hard",
	Parse: func(value string) (string, error) {
		if value == "" {
			return value, nil
		}
		for _, statement := range strings.Split(value, ",") {
			parts := strings.Split(statement, "=")
			if len(parts) != 2 {
				return "", cli.Rejection.Message("resource-quota-hard", "expected resource=value", cli.Rejection.At(0, 0, len(value)))
			}
			if _, err := resource.ParseQuantity(parts[1]); err != nil {
				return "", cli.Rejection.Message("resource-quota-hard", err.Error(), cli.Rejection.At(0, 0, len(value)))
			}
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

var quotaScopesLanguage = cli.ValueCodec[string]{
	Name: "resource-quota-scopes",
	Parse: func(value string) (string, error) {
		if value == "" {
			return value, nil
		}
		for _, scope := range strings.Split(value, ",") {
			if scope == "" {
				return "", cli.Rejection.Message("resource-quota-scopes", "scope must not be empty", cli.Rejection.At(0, 0, len(value)))
			}
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

type createQuotaConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	DryRun                   cmdutil.DryRunStrategy
	FieldManager             string
	Hard                     string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	SaveConfig               bool
	Scopes                   string
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createQuotaCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createQuotaConfig] {
	cmd := cli.Command[createQuotaConfig]{
		Name:    "quota",
		Aliases: cli.Aliases("resourcequota"),
		Description: cli.Description{
			Short: "Create a quota with the specified name",
			Long:  "Create a resource quota with the specified name, hard limits, and optional scopes.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a new resource quota named my-quota",
				CommandLine: "kubectl create quota my-quota --hard=cpu=1,memory=1G,pods=2,services=3,replicationcontrollers=2,resourcequotas=1,secrets=5,persistentvolumeclaims=10",
			},
			{
				Description: "Create a new resource quota named best-effort",
				CommandLine: "kubectl create quota best-effort --hard=pods=100 --scopes=BestEffort",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createQuotaConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createQuotaConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createQuotaConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			cb.Rule.NotEmpty(name)

			declarePrintFlags(
				cb,
				&cb.Config.Output,
				&cb.Config.OutputState,
				&cb.Config.Template,
				&cb.Config.AllowMissingTemplateKeys,
				&cb.Config.ShowManagedFields,
			)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.").Long("dry-run").ImplicitValue(cmdutil.DryRunClient)
			cb.Values.Default(dryRun, cmdutil.DryRunNone)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			hardFlag := cb.Flags.Value(&cb.Config.Hard, quotaHardLanguage)
			cb.Values.ResolveWith(hardFlag, cli.Value.KeepLast())
			hardFlag.Description("A comma-delimited set of resource=quantity pairs that define a hard limit.")
			hardFlag.Long("hard")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			cb.Values.ResolveWith(saveConfigFlag, cli.Value.KeepLast())
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			scopesFlag := cb.Flags.Value(&cb.Config.Scopes, quotaScopesLanguage)
			cb.Values.ResolveWith(scopesFlag, cli.Value.KeepLast())
			scopesFlag.Description("A comma-delimited set of quota scopes that must all match each object tracked by the quota.")
			scopesFlag.Long("scopes")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")
		},
		Run: runCreateQuota,
	}
	return cmd
}

func runCreateQuota(ctx context.Context, cfg *createQuotaConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewQuotaOpts(cfg.Streams)
	options.Name = cfg.Name
	options.Hard = cfg.Hard
	options.Scopes = cfg.Scopes
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = corev1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy = cfg.DryRun
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlagsWithOutputState(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}
