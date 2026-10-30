package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type copyFileSpec struct {
	PodName      string
	PodNamespace string
	File         string
}

func (spec copyFileSpec) String() string {
	if spec.PodName == "" {
		return spec.File
	}
	pod := spec.PodName
	if spec.PodNamespace != "" {
		pod = spec.PodNamespace + "/" + pod
	}
	return pod + ":" + spec.File
}

type cpConfig struct {
	Client      clientConfig
	Streams     genericiooptions.IOStreams
	Source      copyFileSpec
	Destination copyFileSpec
	Container   string
	NoPreserve  bool
	Retries     int
}

var copyFileSpecLanguage = cli.ValueCodec[copyFileSpec]{
	Name: "kubectl-copy-file-spec",
	Parse: func(value string) (copyFileSpec, error) {
		fileSpec, err := parseCopyFileSpec(value)
		if err != nil {
			return copyFileSpec{}, cli.Rejection.Message(
				"file-spec",
				err.Error(),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return fileSpec, nil
	},
	Format: func(value copyFileSpec) string { return value.String() },
	Snapshot: func(value copyFileSpec) copyFileSpec {
		return value
	},
}

func parseCopyFileSpec(value string) (copyFileSpec, error) {
	index := strings.Index(value, ":")
	if index == 0 {
		return copyFileSpec{}, fmt.Errorf("filespec must match the canonical format: [[namespace/]pod:]file/path")
	}
	if index == -1 {
		if value == "" {
			return copyFileSpec{}, fmt.Errorf("filepath can not be empty")
		}
		fileSpec := copyFileSpec{
			File: value,
		}
		return fileSpec, nil
	}
	pod, file := value[:index], value[index+1:]
	if file == "" {
		return copyFileSpec{}, fmt.Errorf("filepath can not be empty")
	}
	pieces := strings.Split(pod, "/")
	switch len(pieces) {
	case 1:
		fileSpec := copyFileSpec{
			PodName: pieces[0],
			File:    file,
		}
		return fileSpec, nil
	case 2:
		fileSpec := copyFileSpec{
			PodNamespace: pieces[0],
			PodName:      pieces[1],
			File:         file,
		}
		return fileSpec, nil
	default:
		return copyFileSpec{}, fmt.Errorf("filespec must match the canonical format: [[namespace/]pod:]file/path")
	}
}

func cpCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[cpConfig] {
	cmd := cli.Command[cpConfig]{
		Name: "cp",
		Description: cli.Description{
			Short: "Copy files and directories to and from containers",
			Long:  "Copy files and directories to and from containers.",
		},
		Examples: []cli.Example{
			{
				Description: "!!!Important Note!!!\nRequires that the 'tar' binary is present in your container\nimage.  If 'tar' is not present, 'kubectl cp' will fail.\n\nFor advanced use cases, such as symlinks, wildcard expansion or\nfile mode preservation, consider using 'kubectl exec'.\n\nCopy /tmp/foo local file to /tmp/bar in a remote pod in namespace <some-namespace>",
				CommandLine: "tar cf - /tmp/foo | kubectl exec -i -n <some-namespace> <some-pod> -- tar xf - -C /tmp/bar",
			},
			{
				Description: "Copy /tmp/foo from a remote pod to /tmp/bar locally",
				CommandLine: "kubectl exec -n <some-namespace> <some-pod> -- tar cf - /tmp/foo | tar xf - -C /tmp/bar",
			},
			{
				Description: "Copy /tmp/foo_dir local directory to /tmp/bar_dir in a remote pod in the default namespace",
				CommandLine: "kubectl cp /tmp/foo_dir <some-pod>:/tmp/bar_dir",
			},
			{
				Description: "Copy /tmp/foo local file to /tmp/bar in a remote pod in a specific container",
				CommandLine: "kubectl cp /tmp/foo <some-pod>:/tmp/bar -c <specific-container>",
			},
			{
				Description: "Copy /tmp/foo local file to /tmp/bar in a remote pod in namespace <some-namespace>",
				CommandLine: "kubectl cp /tmp/foo <some-namespace>/<some-pod>:/tmp/bar",
			},
			{
				Description: "Copy /tmp/foo from a remote pod to /tmp/bar locally",
				CommandLine: "kubectl cp <some-namespace>/<some-pod>:/tmp/foo /tmp/bar",
			},
		},
		Declare: func(cb *cli.CommandBuilder[cpConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			var source, destination cli.Destination[cpConfig, copyFileSpec]
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[cpConfig]) {
				sourceArgument := form.Value(&form.Config.Source, "file-spec-src", copyFileSpecLanguage)
				source = sourceArgument.Destination()
				sourceArgument.CompleteBy(completeCopySource).
					From(client).
					FallbackTo(cli.CompletionNativeFiles)
				destinationArgument := form.Value(&form.Config.Destination, "file-spec-dest", copyFileSpecLanguage)
				destination = destinationArgument.Destination()
				destinationArgument.CompleteWith(cli.CompletionNativeFiles)
			})
			cb.Rule.Check("kubectl-cp-one-remote", func(_ context.Context, cfg *cpConfig, _ cli.Dependencies) (bool, error) {
				sourceRemote := cfg.Source.PodName != ""
				destinationRemote := cfg.Destination.PodName != ""
				return sourceRemote != destinationRemote, nil
			}).
				DependsOn(source).
				DependsOn(destination)
			containerFlag := cb.Flags.String(&cb.Config.Container)
			cb.Values.ResolveWith(containerFlag, cli.Value.KeepLast())
			containerFlag.Description("Container name. If omitted, use the kubectl.kubernetes.io/default-container annotation for selecting the container to be attached or the first container in the pod will be chosen")
			containerFlag.Long("container")
			containerFlag.Short('c').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)
			noPreserveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.NoPreserve))
			cb.Values.ResolveWith(noPreserveFlag, cli.Value.KeepLast())
			noPreserveFlag.Description("The copied file/directory's ownership and permissions will not be preserved in the container")
			noPreserveFlag.Long("no-preserve").ImplicitValue(true)
			retriesFlag := cb.Flags.Int(&cb.Config.Retries)
			cb.Values.ResolveWith(retriesFlag, cli.Value.KeepLast())
			retriesFlag.Description("Set number of retries to complete a copy operation from a container. Specify 0 to disable or any negative value for infinite retrying. The default is 0 (no retry).")
			retriesFlag.Long("retries")
		},
		Run: runCopy,
	}
	return cmd
}

func completeCopySource(ctx context.Context, cfg *cpConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	text := request.Text
	if strings.IndexAny(text, "/.~") == 0 || strings.Contains(text, ":") {
		completion := cli.CompletionContribution{
			Spacing: cli.CompletionSpacingNoSpace,
		}
		return completion, nil
	}
	getter := restClientGetter(cfg.Client)
	values := make([]string, 0)
	if index := strings.Index(text, "/"); index > 0 {
		namespace := text[:index]
		completion, err := completeResourceNamesInNamespace(ctx, getter, "pod", namespace, resourceCompletionBare)
		if err == nil {
			for _, candidate := range completion.Candidates {
				value := namespace + "/" + candidate.Value + ":"
				if strings.HasPrefix(value, text) {
					values = append(values, value)
				}
			}
		}
	} else {
		if completion, err := completeResourceNames(ctx, getter, "namespace", resourceCompletionBare); err == nil {
			for _, candidate := range completion.Candidates {
				if strings.HasPrefix(candidate.Value, text) {
					values = append(values, fmt.Sprintf("%s/", candidate.Value))
				}
			}
		}
		if completion, err := completeResourceNames(ctx, getter, "pod", resourceCompletionBare); err == nil {
			for _, candidate := range completion.Candidates {
				if strings.HasPrefix(candidate.Value, text) {
					values = append(values, fmt.Sprintf("%s:", candidate.Value))
				}
			}
		}
		if len(values) > 0 && text != "" {
			directory := request.WorkingDirectory
			if directory == "" {
				directory = "."
			}
			files, err := os.ReadDir(directory)
			if err == nil {
				for _, file := range files {
					name := file.Name()
					if !strings.HasPrefix(name, text) {
						continue
					}
					if file.IsDir() {
						name += "/"
					}
					values = append(values, name)
				}
			}
		} else if text == "" {
			values = append(values, "./", "/")
		}
	}
	candidates := make([]cli.CompletionCandidate, 0, len(values))
	for _, value := range values {
		candidates = append(candidates, cli.CompletionCandidate{
			Value:   value,
			Kind:    cli.CompletionCandidateResource,
			Spacing: cli.CompletionSpacingNoSpace,
		})
	}
	completion := cli.CompletionContribution{
		Candidates: candidates,
		Spacing:    cli.CompletionSpacingNoSpace,
	}
	return completion, nil
}
