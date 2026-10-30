package main

import (
	"context"
	"runtime"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type applyEditLastAppliedConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	FieldManager             string
	Filename                 filenameOptions
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Record                   bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
	WindowsLineEndings       bool
}

func applyEditLastAppliedCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[applyEditLastAppliedConfig] {
	cmd := cli.Command[applyEditLastAppliedConfig]{
		Name: "edit-last-applied",
		Description: cli.Description{
			Short: "Edit latest last-applied-configuration annotations of a resource/object",
			Long:  "Edit the latest last-applied-configuration annotations of resources from the default editor.\n\n The edit-last-applied command allows you to directly edit any API resource you can retrieve via the command-line tools. It will open the editor defined by your KUBE_EDITOR, or EDITOR environment variables, or fall back to 'vi' for Linux or 'notepad' for Windows. You can edit multiple objects, although changes are applied one at a time. The command accepts file names as well as command-line arguments, although the files you point to must be previously saved versions of resources.\n\n The default format is YAML. To edit in JSON, specify \"-o json\".\n\n The flag --windows-line-endings can be used to force Windows line endings, otherwise the default for your operating system will be used.\n\n In the event an error occurs while updating, a temporary file will be created on disk that contains your unapplied changes. The most common error when updating a resource is another editor changing the resource on the server. When this occurs, you will have to apply your changes to the newer version of the resource, or update your temporary saved copy to include the latest resource version.",
		},
		Examples: []cli.Example{
			{
				Description: "Edit the last-applied-configuration annotations by type/name in YAML",
				CommandLine: "kubectl apply edit-last-applied deployment/nginx",
			},
			{
				Description: "Edit the last-applied-configuration annotations by file in JSON",
				CommandLine: "kubectl apply edit-last-applied -f deploy.yaml -o json",
			},
		},
		Declare: func(cb *cli.CommandBuilder[applyEditLastAppliedConfig]) {
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

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[applyEditLastAppliedConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *applyEditLastAppliedConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						nil,
						resourceArgumentRepeated,
					)).
					From(client)
			})

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-client-side-apply")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename.Groups, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.Description("Filename, directory, or URL to files to use to edit the resource")
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

			cb.Rule.When(cb.When.Equal(recursive, true)).Equal(kustomize, "")
			cb.Rule.Check("kubectl-edit-last-applied-source", func(_ context.Context, cfg *applyEditLastAppliedConfig, _ cli.Dependencies) (bool, error) {
				hasFilename := len(cfg.Filename.Filenames()) > 0
				hasKustomize := cfg.Filename.Kustomize != ""
				if hasFilename && hasKustomize {
					return false, nil
				}
				return len(cfg.Arguments) > 0 || hasFilename || hasKustomize, nil
			}).
				DependsOn(arguments.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			windowsLineEndingsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.WindowsLineEndings))
			cb.Values.ResolveWith(windowsLineEndingsFlag, cli.Value.KeepLast())
			windowsLineEndingsFlag.Description("Defaults to the line ending native to your platform.")
			cb.Values.Default(windowsLineEndingsFlag, runtime.GOOS == "windows")
			windowsLineEndingsFlag.Long("windows-line-endings").ImplicitValue(true)
		},
		Run: func(_ context.Context, cfg *applyEditLastAppliedConfig) error {
			runConfig := editRunConfig{
				Client:                   cfg.Client,
				Streams:                  cfg.Streams,
				Arguments:                cfg.Arguments,
				AllowMissingTemplateKeys: cfg.AllowMissingTemplateKeys,
				FieldManager:             cfg.FieldManager,
				Filename:                 cfg.Filename.Filenames(),
				Kustomize:                cfg.Filename.Kustomize,
				Output:                   cfg.Output,
				Record:                   cfg.Record,
				Recursive:                cfg.Filename.Recursive,
				ShowManagedFields:        cfg.ShowManagedFields,
				Template:                 cfg.Template,
				Validate:                 cfg.Validate,
				WindowsLineEndings:       cfg.WindowsLineEndings,
			}
			return runEditOperation(runConfig, editModeApply)
		},
	}
	return cmd
}
