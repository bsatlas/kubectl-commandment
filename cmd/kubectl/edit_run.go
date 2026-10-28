package main

import (
	"bytes"
	"context"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/scheme"
)

type editRunConfig struct {
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

func runEditOperation(cfg editRunConfig, mode editMode) error {
	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	filename := resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	result := factory.NewBuilder().
		Unstructured().
		ResourceTypeOrNameArgs(true, cfg.Arguments...).
		Latest().
		NamespaceParam(namespace).
		DefaultNamespace().
		FilenameParam(enforceNamespace, &filename).
		Subresource(cfg.Subresource).
		ContinueOnError().
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	validation, err := validationDirective(cfg.Validate)
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("edited").WithTypeSetter(scheme.Scheme)
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	toPrinter := func(operation string) (printers.ResourcePrinter, error) {
		printFlags.NamePrintFlags.Operation = operation
		return printFlags.ToPrinter()
	}

	editorFlags := (&genericclioptions.PrintFlags{
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
	}).WithDefaultOutput("yaml")
	editorPrinter := &editPrinterOptions{
		printFlags: editorFlags,
		ext:        ".yaml",
		addHeader:  true,
	}
	if err := editorPrinter.Complete(printFlags); err != nil {
		return err
	}

	recordFlags := genericclioptions.NewRecordFlags()
	*recordFlags.Record = cfg.Record
	if cfg.Record {
		if err := recordFlags.CompleteWithChangeCause("kubectl edit"); err != nil {
			return err
		}
	}
	recorder, err := recordFlags.ToRecorder()
	if err != nil {
		return err
	}

	operation := editOperation{
		ToPrinter:           toPrinter,
		OutputPatch:         cfg.OutputPatch,
		WindowsLineEndings:  cfg.WindowsLineEndings,
		ValidationDirective: validation,
		OriginalResult:      result,
		Mode:                mode,
		CmdNamespace:        namespace,
		ApplyAnnotation:     cfg.SaveConfig,
		IOStreams:           cfg.Streams,
		Recorder:            recorder,
		f:                   factory,
		editPrinterOptions:  editorPrinter,
		updatedResultGetter: func(data []byte) *resource.Result {
			return factory.NewBuilder().
				Unstructured().
				Stream(bytes.NewReader(data), "edited-file").
				Subresource(cfg.Subresource).
				ContinueOnError().
				Flatten().
				Do()
		},
		FieldManager: cfg.FieldManager,
		Subresource:  cfg.Subresource,
	}
	return operation.Run()
}

func runEdit(_ context.Context, cfg *editConfig) error {
	runConfig := editRunConfig{
		Client:                   cfg.Client,
		Streams:                  cfg.Streams,
		Arguments:                cfg.Arguments,
		AllowMissingTemplateKeys: cfg.AllowMissingTemplateKeys,
		FieldManager:             cfg.FieldManager,
		Filename:                 cfg.Filename,
		Kustomize:                cfg.Kustomize,
		Output:                   cfg.Output,
		OutputPatch:              cfg.OutputPatch,
		Record:                   cfg.Record,
		Recursive:                cfg.Recursive,
		SaveConfig:               cfg.SaveConfig,
		ShowManagedFields:        cfg.ShowManagedFields,
		Subresource:              cfg.Subresource,
		Template:                 cfg.Template,
		Validate:                 cfg.Validate,
		WindowsLineEndings:       cfg.WindowsLineEndings,
	}
	return runEditOperation(runConfig, editModeNormal)
}

func runApplyEditLastApplied(_ context.Context, cfg *applyEditLastAppliedConfig) error {
	runConfig := editRunConfig{
		Client:                   cfg.Client,
		Streams:                  cfg.Streams,
		Arguments:                cfg.Arguments,
		AllowMissingTemplateKeys: cfg.AllowMissingTemplateKeys,
		FieldManager:             cfg.FieldManager,
		Filename:                 cfg.Filename,
		Kustomize:                cfg.Kustomize,
		Output:                   cfg.Output,
		Record:                   cfg.Record,
		Recursive:                cfg.Recursive,
		ShowManagedFields:        cfg.ShowManagedFields,
		Template:                 cfg.Template,
		Validate:                 cfg.Validate,
		WindowsLineEndings:       cfg.WindowsLineEndings,
	}
	return runEditOperation(runConfig, editModeApply)
}
