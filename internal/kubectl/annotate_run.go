package kubectl

import (
	"context"
	"slices"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/annotate"
	"k8s.io/kubectl/pkg/cmd/util"
)

func runAnnotate(_ context.Context, cfg *annotateConfig) error {
	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}

	factory := cfg.Client.factory()
	flags := annotate.NewAnnotateFlags(factory, cfg.Streams)
	flags.FieldManager = cfg.FieldManager
	flags.List = cfg.List
	flags.Overwrite = cfg.Overwrite
	flags.ResourceVersion = cfg.ResourceVersion
	*flags.ResourceBuilderFlags.LabelSelector = cfg.Selector
	*flags.ResourceBuilderFlags.FieldSelector = cfg.FieldSelector
	*flags.ResourceBuilderFlags.AllNamespaces = cfg.AllNamespaces
	*flags.ResourceBuilderFlags.All = cfg.All
	*flags.ResourceBuilderFlags.Local = cfg.Local
	filenames := slices.Clone(cfg.Filename)
	flags.ResourceBuilderFlags.FileNameFlags.Filenames = &filenames
	flags.ResourceBuilderFlags.FileNameFlags.Kustomize = &cfg.Kustomize
	flags.ResourceBuilderFlags.FileNameFlags.Recursive = &cfg.Recursive
	configurePrintFlags(flags.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	util.PrintFlagsWithDryRunStrategy(flags.PrintFlags, strategy)
	printer, err := flags.PrintFlags.ToPrinter()
	if err != nil {
		return err
	}

	recordFlags := genericclioptions.NewRecordFlags()
	*recordFlags.Record = cfg.Record
	if err := recordFlags.CompleteWithChangeCause("kubectl annotate"); err != nil {
		return err
	}
	recorder, err := recordFlags.ToRecorder()
	if err != nil {
		return err
	}

	options := annotate.AnnotateOptions{
		PrintObj:          printer.PrintObj,
		Recorder:          recorder,
		ResourceBuilder:   flags.ResourceBuilderFlags.ToBuilder(factory, cfg.Operands.Resources),
		DryRunStrategy:    strategy,
		FieldManager:      cfg.FieldManager,
		List:              cfg.List,
		Local:             cfg.Local,
		NewAnnotations:    cfg.Operands.New,
		Overwrite:         cfg.Overwrite,
		RemoveAnnotations: cfg.Operands.Remove,
		ResourceVersion:   cfg.ResourceVersion,
		IOStreams:         cfg.Streams,
	}
	return options.RunAnnotate()
}
