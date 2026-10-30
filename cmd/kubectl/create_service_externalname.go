package main

import (
	"context"
	"slices"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var externalServiceNameLanguage = cli.ValueCodec[string]{
	Name: "external-service-name",
	Parse: func(value string) (string, error) {
		if problems := validation.IsDNS1123Subdomain(value); len(problems) != 0 {
			return "", cli.Rejection.Message("external-service-name", strings.Join(problems, ","), cli.Rejection.At(0, 0, len(value)))
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

type createServiceExternalNameConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	DryRun                   cmdutil.DryRunStrategy
	ExternalName             string
	FieldManager             string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	SaveConfig               bool
	ShowManagedFields        bool
	TCP                      []string
	Template                 string
	Validate                 string
}

func createServiceExternalNameCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createServiceExternalNameConfig] {
	cmd := cli.Command[createServiceExternalNameConfig]{
		Name: "externalname",
		Description: cli.Description{
			Short: "Create an ExternalName service",
			Long:  "Create an ExternalName service with the specified name.\n\n ExternalName service references to an external DNS address instead of only pods, which will allow application authors to reference services that exist off platform, on other clusters, or locally.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a new ExternalName service named my-ns",
				CommandLine: "kubectl create service externalname my-ns --external-name bar.com",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createServiceExternalNameConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createServiceExternalNameConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createServiceExternalNameConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

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

			externalNameFlag := cb.Flags.Value(&cb.Config.ExternalName, externalServiceNameLanguage)
			cb.Values.ResolveWith(externalNameFlag, cli.Value.KeepLast())
			externalNameFlag.Description("External name of service")
			externalNameFlag.Required()
			cb.Rule.Require(externalNameFlag.Destination())
			externalNameFlag.Long("external-name")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			cb.Values.ResolveWith(saveConfigFlag, cli.Value.KeepLast())
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			tcpValue := cb.Values.Sequence(&cb.Config.TCP, servicePortLanguage)
			tcpFlag := cb.Flags.Sequence(tcpValue)
			tcpFlag.DelimitedBy(",")
			tcpFlag.Description("Port pairs can be specified as '<port>:<targetPort>'.")
			tcpFlag.Long("tcp")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")
		},
		Run: runCreateServiceExternalName,
	}
	return cmd
}

func runCreateServiceExternalName(ctx context.Context, cfg *createServiceExternalNameConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewServiceOptions(cfg.Streams, "ExternalName")
	options.Name = cfg.Name
	options.TCP = slices.Clone(cfg.TCP)
	options.ExternalName = cfg.ExternalName
	options.FieldManager = cfg.FieldManager

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.CoreV1()
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
