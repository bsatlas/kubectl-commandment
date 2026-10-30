package main

import (
	"context"
	"slices"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var dockerConfigFileLanguage = cli.ValueCodec[string]{
	Name: "docker-config-file",
	Parse: func(value string) (string, error) {
		if !strings.Contains(value, "=") {
			value = ".dockerconfigjson=" + value
		}
		return createFileSourceLanguage.Parse(value)
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

type createSecretDockerRegistryConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	AppendHash               bool
	DockerEmail              string
	DockerPassword           string
	DockerServer             string
	DockerUsername           string
	DryRun                   cmdutil.DryRunStrategy
	FieldManager             string
	FromFile                 []string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createSecretDockerRegistryCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createSecretDockerRegistryConfig] {
	cmd := cli.Command[createSecretDockerRegistryConfig]{
		Name: "docker-registry",
		Description: cli.Description{
			Short: "Create a secret for use with a Docker registry",
			Long:  "Create a new secret for use with Docker registries.\n        \n        Dockercfg secrets are used to authenticate against Docker registries.\n        \n        When using the Docker command line to push images, you can authenticate to a given registry by running:\n        '$ docker login DOCKER_REGISTRY_SERVER --username=DOCKER_USER --password=DOCKER_PASSWORD --email=DOCKER_EMAIL'.\n        \n That produces a ~/.dockercfg file that is used by subsequent 'docker push' and 'docker pull' commands to authenticate to the registry. The email address is optional.\n\n        When creating applications, you may have a Docker registry that requires authentication.  In order for the\n        nodes to pull images on your behalf, they must have the credentials.  You can provide this information\n        by creating a dockercfg secret and attaching it to your service account.",
		},
		Examples: []cli.Example{
			{
				Description: "If you do not already have a .dockercfg file, create a dockercfg secret directly",
				CommandLine: "kubectl create secret docker-registry my-secret --docker-server=DOCKER_REGISTRY_SERVER --docker-username=DOCKER_USER --docker-password=DOCKER_PASSWORD --docker-email=DOCKER_EMAIL",
			},
			{
				Description: "Create a new secret named my-secret from ~/.docker/config.json",
				CommandLine: "kubectl create secret docker-registry my-secret --from-file=path/to/.docker/config.json",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createSecretDockerRegistryConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createSecretDockerRegistryConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createSecretDockerRegistryConfig]) {
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

			appendHashFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AppendHash))
			cb.Values.ResolveWith(appendHashFlag, cli.Value.KeepLast())
			appendHashFlag.Description("Append a hash of the secret to its name.")
			appendHashFlag.Long("append-hash").ImplicitValue(true)

			dockerEmailFlag := cb.Flags.String(&cb.Config.DockerEmail)
			cb.Values.ResolveWith(dockerEmailFlag, cli.Value.KeepLast())
			dockerEmailFlag.Description("Email for Docker registry")
			dockerEmailFlag.Long("docker-email")

			dockerPasswordFlag := cb.Flags.String(&cb.Config.DockerPassword)
			cb.Values.ResolveWith(dockerPasswordFlag, cli.Value.KeepLast())
			dockerPasswordFlag.Description("Password for Docker registry authentication")
			dockerPasswordFlag.Long("docker-password")

			dockerServerFlag := cb.Flags.String(&cb.Config.DockerServer)
			cb.Values.ResolveWith(dockerServerFlag, cli.Value.KeepLast())
			dockerServerFlag.Description("Server location for Docker registry")
			cb.Values.Default(dockerServerFlag, "https://index.docker.io/v1/")
			dockerServerFlag.Long("docker-server")

			dockerUsernameFlag := cb.Flags.String(&cb.Config.DockerUsername)
			cb.Values.ResolveWith(dockerUsernameFlag, cli.Value.KeepLast())
			dockerUsernameFlag.Description("Username for Docker registry authentication")
			dockerUsernameFlag.Long("docker-username")

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

			fromFileValue := cb.Values.Sequence(&cb.Config.FromFile, dockerConfigFileLanguage)
			fromFileFlag := cb.Flags.Sequence(fromFileValue)
			fromFileFlag.DelimitedBy(",")
			fromFileFlag.Description("Key files can be specified using their file path, in which case a default name of .dockerconfigjson will be given to them, or optionally with a name and file path, in which case the given name will be used. Specifying a directory will iterate each named file in the directory that is a valid secret key. For this command, the key should always be .dockerconfigjson.")
			fromFileFlag.Long("from-file")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			cb.Values.ResolveWith(saveConfigFlag, cli.Value.KeepLast())
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			cb.Rule.Check("kubectl-create-secret-docker-registry-source", func(ctx context.Context, cfg *createSecretDockerRegistryConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.FromFile) > 0 || (cfg.DockerUsername != "" && cfg.DockerPassword != "" && cfg.DockerServer != ""), nil
			}).
				DependsOn(fromFileValue.OptionalDependency()).
				DependsOn(dockerUsernameFlag.Destination().OptionalDependency()).
				DependsOn(dockerPasswordFlag.Destination().OptionalDependency()).
				DependsOn(dockerServerFlag.Destination())
		},
		Run: runCreateSecretDockerRegistry,
	}
	return cmd
}

func runCreateSecretDockerRegistry(ctx context.Context, cfg *createSecretDockerRegistryConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewSecretDockerRegistryOptions(cfg.Streams)
	options.Name = cfg.Name
	options.FileSources = slices.Clone(cfg.FromFile)
	options.Username = cfg.DockerUsername
	options.Email = cfg.DockerEmail
	options.Password = cfg.DockerPassword
	options.Server = cfg.DockerServer
	options.AppendHash = cfg.AppendHash
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

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
