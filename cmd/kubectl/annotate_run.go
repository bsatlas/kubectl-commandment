package main

import (
	"context"
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/annotate"
	"k8s.io/kubectl/pkg/cmd/util"
)

func runAnnotate(_ context.Context, cfg *annotateConfig) error {
	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	resources, pairs, err := util.GetResourcesAndPairs(cfg.Arguments, "annotation")
	if err != nil {
		return err
	}
	newAnnotations, removeAnnotations, err := util.ParsePairs(pairs, "annotation", true)
	if err != nil {
		return err
	}
	for _, key := range removeAnnotations {
		if _, ok := newAnnotations[key]; ok {
			return fmt.Errorf("can not both modify and remove annotation %q in the same command", key)
		}
	}
	if cfg.List && cfg.Output != "" {
		return fmt.Errorf("--list and --output may not be specified together")
	}
	if cfg.All && cfg.Selector != "" {
		return fmt.Errorf("cannot set --all and --selector at the same time")
	}
	if cfg.All && cfg.FieldSelector != "" {
		return fmt.Errorf("cannot set --all and --field-selector at the same time")
	}
	if cfg.Local {
		if strategy == util.DryRunServer {
			return fmt.Errorf("cannot specify --local and --dry-run=server - did you mean --dry-run=client?")
		}
		if len(resources) != 0 {
			return fmt.Errorf("can only use local files by -f rsrc.yaml or --filename=rsrc.json when --local=true is set")
		}
		if util.IsFilenameSliceEmpty(cfg.Filename, cfg.Kustomize) {
			return fmt.Errorf("one or more files must be specified as -f rsrc.yaml or --filename=rsrc.json")
		}
	} else if len(resources) == 0 && util.IsFilenameSliceEmpty(cfg.Filename, cfg.Kustomize) {
		return fmt.Errorf("one or more resources must be specified as <resource> <name> or <resource>/<name>")
	}
	if len(newAnnotations) == 0 && len(removeAnnotations) == 0 && !cfg.List {
		return fmt.Errorf("at least one annotation update is required")
	}
	if cfg.List && (len(newAnnotations) != 0 || len(removeAnnotations) != 0) {
		return fmt.Errorf("cannot modify annotations when --list is specified")
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
	filenames := append([]string(nil), cfg.Filename...)
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
		ResourceBuilder:   flags.ResourceBuilderFlags.ToBuilder(factory, resources),
		DryRunStrategy:    strategy,
		FieldManager:      cfg.FieldManager,
		List:              cfg.List,
		Local:             cfg.Local,
		NewAnnotations:    newAnnotations,
		Overwrite:         cfg.Overwrite,
		RemoveAnnotations: removeAnnotations,
		ResourceVersion:   cfg.ResourceVersion,
		IOStreams:         cfg.Streams,
	}
	return options.RunAnnotate()
}
