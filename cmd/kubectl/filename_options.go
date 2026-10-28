package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"slices"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
)

type filenameOptions struct {
	Groups    [][]string
	Kustomize string
	Recursive bool
}

func (options filenameOptions) Filenames() []string {
	return slices.Concat(options.Groups...)
}

type filenameOptionDestinations[C any] struct {
	Filenames cli.Destination[C, [][]string]
	Kustomize cli.Destination[C, string]
	Recursive cli.Destination[C, bool]
}

func declareFilenameOptions[C any](cb *cli.CommandBuilder[C], storage *filenameOptions) filenameOptionDestinations[C] {
	filenames := cb.Values.Sequence(&storage.Groups, filenameListLanguage)
	filenameFlag := cb.Flags.Sequence(filenames).
		Description("Filename, directory, or URL to files identifying the resource to get from a server.").
		CompleteWith(cli.CompletionNativePaths)
	filenameFlag.Long("filename")
	filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

	kustomize := cb.Values.String(&storage.Kustomize)
	kustomizeFlag := cb.Flags.ValueFor(kustomize).
		Description("Process the kustomization directory. This flag can't be used together with -f or -R.").
		CompleteWith(cli.CompletionNativeDirectories)
	kustomizeFlag.Long("kustomize")
	kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

	recursive := cb.Values.Bool(&storage.Recursive)
	recursiveFlag := cb.Flags.ValueFor(recursive).Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
	recursiveFlag.Long("recursive").ImplicitValue(true)
	recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

	cb.Rule.Check("kubectl-filename-kustomize", func(_ context.Context, _ *C, dependencies cli.Dependencies) (bool, error) {
		groups, _ := dependencies.Get(filenames)
		kustomizeValue, _ := dependencies.Get(kustomize)
		return len(slices.Concat(groups...)) == 0 || kustomizeValue == "", nil
	}).
		Subject(filenames).
		Subject(kustomize).
		DependsOn(filenames.OptionalDependency()).
		DependsOn(kustomize.OptionalDependency())

	cb.Rule.Check("kubectl-kustomize-recursive", func(_ context.Context, _ *C, dependencies cli.Dependencies) (bool, error) {
		kustomizeValue, _ := dependencies.Get(kustomize)
		recursiveValue, _ := dependencies.Get(recursive)
		return kustomizeValue == "" || !recursiveValue, nil
	}).
		Subject(kustomize).
		Subject(recursive).
		DependsOn(kustomize.OptionalDependency()).
		DependsOn(recursive.OptionalDependency())

	destinations := filenameOptionDestinations[C]{
		Filenames: filenames,
		Kustomize: kustomize,
		Recursive: recursive,
	}
	return destinations
}

var filenameListLanguage = cli.ValueCodec[[]string]{
	Name: "kubectl-filename-list",
	Parse: func(value string) ([]string, error) {
		if value == "" {
			return []string{}, nil
		}
		values, err := csv.NewReader(strings.NewReader(value)).Read()
		if err != nil {
			return nil, cli.Rejection.Message(
				"filename-list",
				fmt.Sprintf("invalid CSV filename list %q: %v", value, err),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return values, nil
	},
	Format: func(values []string) string {
		if len(values) == 0 {
			return ""
		}
		var text strings.Builder
		writer := csv.NewWriter(&text)
		if err := writer.Write(values); err != nil {
			panic(fmt.Sprintf("format filename list: %v", err))
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			panic(fmt.Sprintf("format filename list: %v", err))
		}
		return strings.TrimSuffix(text.String(), "\n")
	},
	Snapshot: func(values []string) []string { return slices.Clone(values) },
}
