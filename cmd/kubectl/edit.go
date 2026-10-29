package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

var editValidationGroupKind = v1.SchemeGroupVersion.WithKind("").GroupKind()

type editConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	FieldManager             string
	Filename                 []string
	Kustomize                string
	Output                   string
	OutputPatch              bool
	Record                   bool
	Recursive                bool
	SaveConfig               bool
	ShowManagedFields        bool
	Subresource              string
	Template                 string
	Validate                 string
	WindowsLineEndings       bool
}

func editCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[editConfig] {
	cmd := cli.Command[editConfig]{
		Name: "edit",
		Description: cli.Description{
			Short: "Edit a resource on the server",
			Long:  "Edit a resource from the default editor.\n\n The edit command allows you to directly edit any API resource you can retrieve via the command-line tools. It will open the editor defined by your KUBE_EDITOR, or EDITOR environment variables, or fall back to 'vi' for Linux or 'notepad' for Windows. When attempting to open the editor, it will first attempt to use the shell that has been defined in the 'SHELL' environment variable. If this is not defined, the default shell will be used, which is '/bin/bash' for Linux or 'cmd' for Windows.\n\n You can edit multiple objects, although changes are applied one at a time. The command accepts file names as well as command-line arguments, although the files you point to must be previously saved versions of resources.\n\n Editing is done with the API version used to fetch the resource. To edit using a specific API version, fully-qualify the resource, version, and group.\n\n The default format is YAML. To edit in JSON, specify \"-o json\".\n\n The flag --windows-line-endings can be used to force Windows line endings, otherwise the default for your operating system will be used.\n\n In the event an error occurs while updating, a temporary file will be created on disk that contains your unapplied changes. The most common error when updating a resource is another editor changing the resource on the server. When this occurs, you will have to apply your changes to the newer version of the resource, or update your temporary saved copy to include the latest resource version.",
		},
		Examples: []cli.Example{
			{
				Description: "Edit the service named 'registry'",
				CommandLine: "kubectl edit svc/registry",
			},
			{
				Description: "Use an alternative editor",
				CommandLine: "KUBE_EDITOR=\"nano\" kubectl edit svc/registry",
			},
			{
				Description: "Edit the job 'myjob' in JSON using the v1 API format",
				CommandLine: "kubectl edit job.v1.batch/myjob -o json",
			},
			{
				Description: "Edit the deployment 'mydeployment' in YAML and save the modified config in its annotation",
				CommandLine: "kubectl edit deployment/mydeployment -o yaml --save-config",
			},
			{
				Description: "Edit the 'status' subresource for the 'mydeployment' deployment",
				CommandLine: "kubectl edit deployment mydeployment --subresource='status'",
			},
		},
		Declare: func(cb *cli.CommandBuilder[editConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[editConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *editConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						nil,
						resourceArgumentRepeated,
					)).
					From(client)
			})

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-edit")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.DelimitedBy(",")
			filenameFlag.Description("Filename, directory, or URL to files to use to edit the resource")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomize := cb.Values.String(&cb.Config.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			outputPatchFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.OutputPatch))
			outputPatchFlag.Description("Output the patch if the resource is edited.")
			outputPatchFlag.Long("output-patch").ImplicitValue(true)

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursive := cb.Values.Bool(&cb.Config.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive)
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			cb.Rule.When(cb.When.Equal(recursive, true)).Equal(kustomize, "")
			cb.Rule.Check("kubectl-edit-source", func(_ context.Context, cfg *editConfig, _ cli.Dependencies) (bool, error) {
				hasFilename := len(cfg.Filename) > 0
				hasKustomize := cfg.Kustomize != ""
				if hasFilename && hasKustomize {
					return false, nil
				}
				return len(cfg.Arguments) > 0 || hasFilename || hasKustomize, nil
			}).
				DependsOn(arguments.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			subresourceFlag := cb.Flags.String(&cb.Config.Subresource)
			subresourceFlag.Description("If specified, edit will operate on the subresource of the requested object.")
			subresourceFlag.CompleteBy(literalCompleter[editConfig](
				cli.CompletionCandidateValue,
				"status", "scale", "resize",
			))
			subresourceFlag.Long("subresource")

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			windowsLineEndingsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.WindowsLineEndings))
			windowsLineEndingsFlag.Description("Defaults to the line ending native to your platform.")
			cb.Values.Default(windowsLineEndingsFlag, runtime.GOOS == "windows")
			windowsLineEndingsFlag.Long("windows-line-endings").ImplicitValue(true)
		},
		Run: runEdit,
	}
	return cmd
}

// preservedFile writes the location of the editor recovery file when it still exists.
func preservedFile(err error, path string, out io.Writer) error {
	if path == "" {
		return err
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, fs.ErrNotExist) {
		fmt.Fprintf(out, "A copy of your changes has been stored to %q\n", path)
	}
	return err
}
