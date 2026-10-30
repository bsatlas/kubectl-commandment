package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type setImageConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []imageArgument
	All                      bool
	AllowMissingTemplateKeys bool
	DryRun                   util.DryRunStrategy
	FieldManager             string
	Filename                 filenameOptions
	Local                    bool
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Record                   bool
	Selector                 string
	ShowManagedFields        bool
	Template                 string
}

type imageArgumentKind uint8

const (
	imageArgumentResource imageArgumentKind = iota
	imageArgumentUpdate
)

type imageArgument struct {
	kind     imageArgumentKind
	resource string
	update   imageUpdate
}

type imageUpdate struct {
	container string
	image     string
}

func parseImageUpdate(value string) (imageUpdate, error) {
	if !strings.Contains(value, "=") || value[0] == '=' {
		return imageUpdate{}, cli.Rejection.Message(
			"image",
			fmt.Sprintf("invalid image format: %s", value),
			cli.Rejection.At(0, 0, len(value)),
		)
	}
	container, image, _ := strings.Cut(value, "=")
	update := imageUpdate{
		container: container,
		image:     image,
	}
	return update, nil
}

var imageArgumentLanguage = cli.ValueCodec[imageArgument]{
	Name: "kubectl-image-argument",
	Parse: func(value string) (imageArgument, error) {
		if !imagePairLike(value) {
			argument := imageArgument{
				kind:     imageArgumentResource,
				resource: value,
			}
			return argument, nil
		}
		update, err := parseImageUpdate(value)
		if err != nil {
			return imageArgument{}, err
		}
		argument := imageArgument{
			kind:   imageArgumentUpdate,
			update: update,
		}
		return argument, nil
	},
	Format: func(argument imageArgument) string {
		if argument.kind == imageArgumentResource {
			return argument.resource
		}
		return argument.update.container + "=" + argument.update.image
	},
	Snapshot: cli.Value.Assign[imageArgument](),
}

func splitImageArguments(arguments []imageArgument) ([]string, map[string]string) {
	resources := make([]string, 0, len(arguments))
	images := make(map[string]string)
	for _, argument := range arguments {
		switch argument.kind {
		case imageArgumentResource:
			resources = append(resources, argument.resource)
		case imageArgumentUpdate:
			images[argument.update.container] = argument.update.image
		}
	}
	return resources, images
}

func imageArgumentOrderValid(arguments []imageArgument) bool {
	updateSeen := false
	for _, argument := range arguments {
		if argument.kind == imageArgumentUpdate {
			updateSeen = true
			continue
		}
		if updateSeen {
			return false
		}
	}
	return true
}

func imagePairLike(value string) bool {
	return (strings.Contains(value, "=") && value[0] != '=') || (strings.HasSuffix(value, "-") && value != "-")
}

func validImageUpdates(images map[string]string) bool {
	_, wildcard := images["*"]
	return !wildcard || len(images) == 1
}

func setImageCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[setImageConfig] {
	cmd := cli.Command[setImageConfig]{
		Name: "image",
		Description: cli.Description{
			Short: "Update the image of a pod template",
			Long:  "Update existing container image(s) of resources.\n\n Possible resources include (case insensitive):\n\n        pod (po), replicationcontroller (rc), deployment (deploy), daemonset (ds), statefulset (sts), cronjob (cj), replicaset (rs)",
		},
		Examples: []cli.Example{
			{
				Description: "Set a deployment's nginx container image to 'nginx:1.9.1', and its busybox container image to 'busybox'",
				CommandLine: "kubectl set image deployment/nginx busybox=busybox nginx=nginx:1.9.1",
			},
			{
				Description: "Update all deployments' and rc's nginx container's image to 'nginx:1.9.1'",
				CommandLine: "kubectl set image deployments,rc nginx=nginx:1.9.1 --all",
			},
			{
				Description: "Update image of all containers of daemonset abc to 'nginx:1.9.1'",
				CommandLine: "kubectl set image daemonset abc *=nginx:1.9.1",
			},
			{
				Description: "Print result (in yaml format) of updating nginx container image from local file, without hitting the server",
				CommandLine: "kubectl set image -f path/to/file.yaml nginx=nginx:1.9.1 --local -o yaml",
			},
		},
		Declare: func(cb *cli.CommandBuilder[setImageConfig]) {
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
			arguments := cb.Values.Sequence(&cb.Config.Arguments, imageArgumentLanguage)
			resourceCompleter := resourceArgumentCompleter(
				func(cfg *setImageConfig) (clientConfig, []string) {
					resources, _ := splitImageArguments(cfg.Arguments)
					return cfg.Client, resources
				},
				[]string{"daemonsets", "deployments", "pods", "cronjobs", "replicasets", "replicationcontrollers", "statefulsets"},
				resourceArgumentSingle,
			)
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[setImageConfig]) {
				form.Sequence(arguments, "RESOURCE|CONTAINER=IMAGE", cli.Count.OneOrMore()).
					CompleteBy(func(ctx context.Context, cfg *setImageConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
						_, images := splitImageArguments(cfg.Arguments)
						if len(images) != 0 {
							return cli.CompletionContribution{}, nil
						}
						return resourceCompleter(ctx, cfg, request)
					}).
					From(client)
			})

			all := cb.Values.Bool(&cb.Config.All)
			allFlag := cb.Flags.ValueFor(all)
			cb.Values.ResolveWith(allFlag, cli.Value.KeepLast())
			allFlag.Description("Select all resources, in the namespace of the specified resource types")
			allFlag.Long("all").ImplicitValue(true)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, util.DryRunNone)
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue(util.DryRunClient)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-set")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename.Groups, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.Description("Filename, directory, or URL to files identifying the resource to get from a server.")
			filenameFlag.CompleteWith(cli.CompletionNativePaths)
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			kustomize := cb.Values.String(&cb.Config.Filename.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			cb.Values.ResolveWith(kustomizeFlag, cli.Value.KeepLast())
			kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.CompleteWith(cli.CompletionNativeDirectories)
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			local := cb.Values.Bool(&cb.Config.Local)
			localFlag := cb.Flags.ValueFor(local)
			cb.Values.ResolveWith(localFlag, cli.Value.KeepLast())
			localFlag.Description("If true, set image will NOT contact api-server but run locally.")
			localFlag.Long("local").ImplicitValue(true)

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			cb.Values.ResolveWith(recordFlag, cli.Value.KeepLast())
			recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursive := cb.Values.Bool(&cb.Config.Filename.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive)
			cb.Values.ResolveWith(recursiveFlag, cli.Value.KeepLast())
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			selector := cb.Values.String(&cb.Config.Selector)
			selectorFlag := cb.Flags.ValueFor(selector)
			cb.Values.ResolveWith(selectorFlag, cli.Value.KeepLast())
			selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			cb.Rule.When(cb.When.Equal(all, true)).Equal(selector, "")
			cb.Rule.Check("kubectl-set-image-filename-kustomize", func(_ context.Context, cfg *setImageConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Filename.Filenames()) == 0 || cfg.Filename.Kustomize == "", nil
			}).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
			cb.Rule.When(cb.When.Equal(recursive, true)).Forbid(kustomize)
			cb.Rule.Check("kubectl-set-image-argument-order", func(_ context.Context, cfg *setImageConfig, _ cli.Dependencies) (bool, error) {
				return imageArgumentOrderValid(cfg.Arguments), nil
			}).DependsOn(arguments.OptionalDependency())
			cb.Rule.Check("kubectl-set-image-local-resources", func(_ context.Context, cfg *setImageConfig, _ cli.Dependencies) (bool, error) {
				resources, _ := splitImageArguments(cfg.Arguments)
				return !cfg.Local || len(resources) == 0, nil
			}).
				DependsOn(local.OptionalDependency()).
				DependsOn(arguments.OptionalDependency())
			cb.Rule.Check("kubectl-set-image-source", func(_ context.Context, cfg *setImageConfig, _ cli.Dependencies) (bool, error) {
				resources, _ := splitImageArguments(cfg.Arguments)
				return len(resources) != 0 || len(cfg.Filename.Filenames()) != 0 || cfg.Filename.Kustomize != "", nil
			}).
				DependsOn(arguments.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
			cb.Rule.Check("kubectl-set-image-update", func(_ context.Context, cfg *setImageConfig, _ cli.Dependencies) (bool, error) {
				_, images := splitImageArguments(cfg.Arguments)
				return len(images) != 0, nil
			}).DependsOn(arguments.OptionalDependency())
			cb.Rule.Check("kubectl-set-image-wildcard", func(_ context.Context, cfg *setImageConfig, _ cli.Dependencies) (bool, error) {
				_, images := splitImageArguments(cfg.Arguments)
				return validImageUpdates(images), nil
			}).DependsOn(arguments.OptionalDependency())
			cb.Rule.Check("kubectl-set-image-local-dry-run", func(_ context.Context, cfg *setImageConfig, _ cli.Dependencies) (bool, error) {
				return !cfg.Local || cfg.DryRun != util.DryRunServer, nil
			}).
				DependsOn(local.OptionalDependency()).
				DependsOn(dryRun.OptionalDependency())
		},
		Run: runSetImage,
	}
	return cmd
}
