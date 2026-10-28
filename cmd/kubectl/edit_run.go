package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/evanphx/json-patch.v4"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/cmd/util/editor"
	"k8s.io/kubectl/pkg/cmd/util/editor/crlf"
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

// editOperation contains all the options for running edit cli command.
type editOperation struct {
	ToPrinter func(string) (printers.ResourcePrinter, error)

	OutputPatch         bool
	WindowsLineEndings  bool
	ValidationDirective string

	OriginalResult *resource.Result
	Mode           editMode

	CmdNamespace    string
	ApplyAnnotation bool

	managedFields map[types.UID][]v1.ManagedFieldsEntry

	genericiooptions.IOStreams

	Recorder            genericclioptions.Recorder
	f                   util.Factory
	editPrinterOptions  *editPrinterOptions
	updatedResultGetter func(data []byte) *resource.Result

	FieldManager string

	Subresource string
}

type editPrinterOptions struct {
	printFlags *genericclioptions.PrintFlags
	ext        string
	addHeader  bool
}

func (e *editPrinterOptions) Complete(fromPrintFlags *genericclioptions.PrintFlags) error {
	if e.printFlags == nil {
		return fmt.Errorf("missing PrintFlags in editor printer options")
	}

	// bind output format from existing printflags
	if fromPrintFlags != nil && len(*fromPrintFlags.OutputFormat) > 0 {
		e.printFlags.OutputFormat = fromPrintFlags.OutputFormat
	}

	// prevent a commented header at the top of the user's
	// default editor if presenting contents as json.
	if *e.printFlags.OutputFormat == "json" {
		e.addHeader = false
		e.ext = ".json"
		return nil
	}

	// we default to yaml if check above is false, as only json or yaml are supported
	e.addHeader = true
	e.ext = ".yaml"
	return nil
}

func (e *editPrinterOptions) PrintObj(obj runtime.Object, out io.Writer) error {
	p, err := e.printFlags.ToPrinter()
	if err != nil {
		return err
	}

	return p.PrintObj(obj, out)
}

// Run performs the execution
func (o *editOperation) Run() error {
	edit := editor.NewDefaultEditor(editorEnvs())
	// editFn is invoked for each edit session (once with a list for normal edit, once for each individual resource in a edit-on-create invocation)
	editFn := func(infos []*resource.Info) error {
		var (
			results  = editResults{}
			original = []byte{}
			edited   = []byte{}
			file     string
			err      error
		)

		containsError := false
		// loop until we succeed or cancel editing
		for {
			// get the object we're going to serialize as input to the editor
			var originalObj runtime.Object
			switch len(infos) {
			case 1:
				originalObj = infos[0].Object
			default:
				l := &unstructured.UnstructuredList{
					Object: map[string]interface{}{
						"kind":       "List",
						"apiVersion": "v1",
						"metadata":   map[string]interface{}{},
					},
				}
				for _, info := range infos {
					l.Items = append(l.Items, *info.Object.(*unstructured.Unstructured))
				}
				originalObj = l
			}

			// generate the file to edit
			buf := &bytes.Buffer{}
			var w io.Writer = buf
			if o.WindowsLineEndings {
				w = crlf.NewCRLFWriter(w)
			}

			if o.editPrinterOptions.addHeader {
				results.header.writeTo(w, o.Mode)
			}

			if !containsError {
				if err := o.extractManagedFields(originalObj); err != nil {
					return preservedFile(err, results.file, o.ErrOut)
				}

				if err := o.editPrinterOptions.PrintObj(originalObj, w); err != nil {
					return preservedFile(err, results.file, o.ErrOut)
				}
				original = buf.Bytes()
			} else {
				// In case of an error, preserve the edited file.
				// Remove the comments (header) from it since we already
				// have included the latest header in the buffer above.
				buf.Write(util.ManualStrip(edited))
			}

			// launch the editor
			editedDiff := edited
			edited, file, err = edit.LaunchTempFile(fmt.Sprintf("%s-edit-", filepath.Base(os.Args[0])), o.editPrinterOptions.ext, buf)
			if err != nil {
				return preservedFile(err, results.file, o.ErrOut)
			}

			// If we're retrying the loop because of an error, and no change was made in the file, short-circuit
			if containsError && bytes.Equal(util.StripComments(editedDiff), util.StripComments(edited)) {
				return preservedFile(fmt.Errorf("%s", "Edit cancelled, no valid changes were saved."), file, o.ErrOut)
			}
			// cleanup any file from the previous pass
			if len(results.file) > 0 {
				os.Remove(results.file)
			}
			klog.V(4).Infof("User edited:\n%s", string(edited))

			// Apply validation
			schema, err := o.f.Validator(o.ValidationDirective)
			if err != nil {
				return preservedFile(err, file, o.ErrOut)
			}
			err = schema.ValidateBytes(util.StripComments(edited))
			if err != nil {
				results = editResults{
					file: file,
				}
				containsError = true
				fmt.Fprintln(o.ErrOut, results.addError(editInvalidError(err), infos[0]))
				continue
			}

			// Compare content without comments
			if bytes.Equal(util.StripComments(original), util.StripComments(edited)) {
				os.Remove(file)
				fmt.Fprintln(o.ErrOut, "Edit cancelled, no changes made.")
				return nil
			}

			lines, err := hasLines(bytes.NewBuffer(edited))
			if err != nil {
				return preservedFile(err, file, o.ErrOut)
			}
			if !lines {
				os.Remove(file)
				fmt.Fprintln(o.ErrOut, "Edit cancelled, saved file was empty.")
				return nil
			}

			results = editResults{
				file: file,
			}

			// parse the edited file
			updatedInfos, err := o.updatedResultGetter(edited).Infos()
			if err != nil {
				// syntax error
				containsError = true
				reason := editReason{
					head: fmt.Sprintf("The edited file had a syntax error: %v", err),
				}
				results.header.reasons = append(results.header.reasons, reason)
				continue
			}

			// not a syntax error as it turns out...
			containsError = false
			updatedVisitor := resource.InfoListVisitor(updatedInfos)

			// we need to add back managedFields to both updated and original object
			if err := o.restoreManagedFields(updatedInfos); err != nil {
				return preservedFile(err, file, o.ErrOut)
			}
			if err := o.restoreManagedFields(infos); err != nil {
				return preservedFile(err, file, o.ErrOut)
			}

			// need to make sure the original namespace wasn't changed while editing
			if err := updatedVisitor.Visit(resource.RequireNamespace(o.CmdNamespace)); err != nil {
				return preservedFile(err, file, o.ErrOut)
			}

			// iterate through all items to apply annotations
			if err := o.visitAnnotation(updatedVisitor); err != nil {
				return preservedFile(err, file, o.ErrOut)
			}

			switch o.Mode {
			case editModeNormal:
				err = o.visitToPatch(infos, updatedVisitor, &results)
			case editModeApply:
				err = o.visitToApplyEditPatch(infos, updatedVisitor)
			case editModeBeforeCreate:
				err = o.visitToCreate(updatedVisitor)
			default:
				err = fmt.Errorf("unsupported edit mode %q", o.Mode)
			}
			if err != nil {
				return preservedFile(err, results.file, o.ErrOut)
			}

			// Handle all possible errors
			//
			// 1. retryable: propose kubectl replace -f
			// 2. notfound: indicate the location of the saved configuration of the deleted resource
			// 3. invalid: retry those on the spot by looping ie. reloading the editor
			if results.retryable > 0 {
				fmt.Fprintf(o.ErrOut, "You can run `%s replace -f %s` to try this update again.\n", filepath.Base(os.Args[0]), file)
				return util.ErrExit
			}
			if results.notfound > 0 {
				fmt.Fprintf(o.ErrOut, "The edits you made on deleted resources have been saved to %q\n", file)
				return util.ErrExit
			}

			if len(results.edit) == 0 {
				if results.notfound == 0 {
					os.Remove(file)
				} else {
					fmt.Fprintf(o.Out, "The edits you made on deleted resources have been saved to %q\n", file)
				}
				return nil
			}

			if len(results.header.reasons) > 0 {
				containsError = true
			}
		}
	}

	switch o.Mode {
	// If doing normal edit we cannot use Visit because we need to edit a list for convenience. Ref: #20519
	case editModeNormal:
		infos, err := o.OriginalResult.Infos()
		if err != nil {
			return err
		}
		if len(infos) == 0 {
			return fmt.Errorf("edit cancelled, no objects found")
		}
		return editFn(infos)
	case editModeApply:
		infos, err := o.OriginalResult.Infos()
		if err != nil {
			return err
		}
		var annotationInfos []*resource.Info
		for i := range infos {
			data, err := originalConfiguration(infos[i].Object)
			if err != nil {
				return err
			}
			if data == nil {
				continue
			}

			tempInfos, err := o.updatedResultGetter(data).Infos()
			if err != nil {
				return err
			}
			annotationInfos = append(annotationInfos, tempInfos[0])
		}
		if len(annotationInfos) == 0 {
			return fmt.Errorf("no last-applied-configuration annotation found on resources, to create the annotation, use command `kubectl apply set-last-applied --create-annotation`")
		}
		return editFn(annotationInfos)
	case editModeBeforeCreate:
		return o.OriginalResult.Visit(func(info *resource.Info, err error) error {
			if err != nil {
				return err
			}
			return editFn([]*resource.Info{info})
		})
	default:
		return fmt.Errorf("unsupported edit mode %q", o.Mode)
	}
}

func (o *editOperation) extractManagedFields(obj runtime.Object) error {
	o.managedFields = make(map[types.UID][]v1.ManagedFieldsEntry)
	if meta.IsListType(obj) {
		err := meta.EachListItem(obj, func(obj runtime.Object) error {
			uid, mf, err := clearManagedFields(obj)
			if err != nil {
				return err
			}
			o.managedFields[uid] = mf
			return nil
		})
		return err
	}
	uid, mf, err := clearManagedFields(obj)
	if err != nil {
		return err
	}
	o.managedFields[uid] = mf
	return nil
}

func clearManagedFields(obj runtime.Object) (types.UID, []v1.ManagedFieldsEntry, error) {
	metaObjs, err := meta.Accessor(obj)
	if err != nil {
		return "", nil, err
	}
	mf := metaObjs.GetManagedFields()
	metaObjs.SetManagedFields(nil)
	return metaObjs.GetUID(), mf, nil
}

func (o *editOperation) restoreManagedFields(infos []*resource.Info) error {
	for _, info := range infos {
		metaObjs, err := meta.Accessor(info.Object)
		if err != nil {
			return err
		}
		mf := o.managedFields[metaObjs.GetUID()]
		metaObjs.SetManagedFields(mf)
	}
	return nil
}

func (o *editOperation) visitToApplyEditPatch(originalInfos []*resource.Info, patchVisitor resource.Visitor) error {
	err := patchVisitor.Visit(func(info *resource.Info, incomingErr error) error {
		editObjUID, err := meta.NewAccessor().UID(info.Object)
		if err != nil {
			return err
		}

		var originalInfo *resource.Info
		for _, i := range originalInfos {
			originalObjUID, err := meta.NewAccessor().UID(i.Object)
			if err != nil {
				return err
			}
			if editObjUID == originalObjUID {
				originalInfo = i
				break
			}
		}
		if originalInfo == nil {
			return fmt.Errorf("no original object found for %#v", info.Object)
		}

		originalJS, err := encodeToJSON(originalInfo.Object.(runtime.Unstructured))
		if err != nil {
			return err
		}

		editedJS, err := encodeToJSON(info.Object.(runtime.Unstructured))
		if err != nil {
			return err
		}

		if bytes.Equal(originalJS, editedJS) {
			printer, err := o.ToPrinter("skipped")
			if err != nil {
				return err
			}
			return printer.PrintObj(info.Object, o.Out)
		}
		err = o.annotationPatch(info)
		if err != nil {
			return err
		}

		printer, err := o.ToPrinter("edited")
		if err != nil {
			return err
		}
		return printer.PrintObj(info.Object, o.Out)
	})
	return err
}

func (o *editOperation) annotationPatch(update *resource.Info) error {
	patch, _, patchType, err := editor.GetApplyPatch(update.Object.(runtime.Unstructured))
	if err != nil {
		return err
	}
	mapping := update.ResourceMapping()
	client, err := o.f.UnstructuredClientForMapping(mapping)
	if err != nil {
		return err
	}
	helper := resource.NewHelper(client, mapping).
		WithFieldManager(o.FieldManager).
		WithFieldValidation(o.ValidationDirective).
		WithSubresource(o.Subresource)
	_, err = helper.Patch(o.CmdNamespace, update.Name, patchType, patch, nil)
	return err
}

func encodeToJSON(obj runtime.Unstructured) ([]byte, error) {
	serialization, err := runtime.Encode(unstructured.UnstructuredJSONScheme, obj)
	if err != nil {
		return nil, err
	}
	js, err := yaml.ToJSON(serialization)
	if err != nil {
		return nil, err
	}
	return js, nil
}

func (o *editOperation) visitToPatch(originalInfos []*resource.Info, patchVisitor resource.Visitor, results *editResults) error {
	err := patchVisitor.Visit(func(info *resource.Info, incomingErr error) error {
		editObjUID, err := meta.NewAccessor().UID(info.Object)
		if err != nil {
			return err
		}

		var originalInfo *resource.Info
		for _, i := range originalInfos {
			originalObjUID, err := meta.NewAccessor().UID(i.Object)
			if err != nil {
				return err
			}
			if editObjUID == originalObjUID {
				originalInfo = i
				break
			}
		}
		if originalInfo == nil {
			return fmt.Errorf("no original object found for %#v", info.Object)
		}

		originalJS, err := encodeToJSON(originalInfo.Object.(runtime.Unstructured))
		if err != nil {
			return err
		}

		editedJS, err := encodeToJSON(info.Object.(runtime.Unstructured))
		if err != nil {
			return err
		}

		if bytes.Equal(originalJS, editedJS) {
			// no edit, so just skip it.
			printer, err := o.ToPrinter("skipped")
			if err != nil {
				return err
			}
			return printer.PrintObj(info.Object, o.Out)
		}

		preconditions := []mergepatch.PreconditionFunc{
			mergepatch.RequireKeyUnchanged("apiVersion"),
			mergepatch.RequireKeyUnchanged("kind"),
			mergepatch.RequireMetadataKeyUnchanged("name"),
			mergepatch.RequireKeyUnchanged("managedFields"),
		}

		// Create the versioned struct from the type defined in the mapping
		// (which is the API version we'll be submitting the patch to)
		versionedObject, err := scheme.Scheme.New(info.Mapping.GroupVersionKind)
		var patchType types.PatchType
		var patch []byte
		switch {
		case runtime.IsNotRegisteredError(err):
			// fall back to generic JSON merge patch
			patchType = types.MergePatchType
			patch, err = jsonpatch.CreateMergePatch(originalJS, editedJS)
			if err != nil {
				klog.V(4).Infof("Unable to calculate diff, no merge is possible: %v", err)
				return err
			}
			var patchMap map[string]interface{}
			err = json.Unmarshal(patch, &patchMap)
			if err != nil {
				klog.V(4).Infof("Unable to calculate diff, no merge is possible: %v", err)
				return err
			}
			for _, precondition := range preconditions {
				if !precondition(patchMap) {
					klog.V(4).Infof("Unable to calculate diff, no merge is possible: %v", err)
					return fmt.Errorf("%s", "At least one of apiVersion, kind and name was changed")
				}
			}
		case err != nil:
			return err
		default:
			patchType = types.StrategicMergePatchType
			patch, err = strategicpatch.CreateTwoWayMergePatch(originalJS, editedJS, versionedObject, preconditions...)
			if err != nil {
				klog.V(4).Infof("Unable to calculate diff, no merge is possible: %v", err)
				if mergepatch.IsPreconditionFailed(err) {
					return fmt.Errorf("%s", "At least one of apiVersion, kind and name was changed")
				}
				return err
			}
		}

		if o.OutputPatch {
			fmt.Fprintf(o.Out, "Patch: %s\n", string(patch))
		}

		patched, err := resource.NewHelper(info.Client, info.Mapping).
			WithFieldManager(o.FieldManager).
			WithFieldValidation(o.ValidationDirective).
			WithSubresource(o.Subresource).
			Patch(info.Namespace, info.Name, patchType, patch, nil)
		if err != nil {
			fmt.Fprintln(o.ErrOut, results.addError(err, info))
			return nil
		}
		info.Refresh(patched, true)
		printer, err := o.ToPrinter("edited")
		if err != nil {
			return err
		}
		return printer.PrintObj(info.Object, o.Out)
	})
	return err
}

func (o *editOperation) visitToCreate(createVisitor resource.Visitor) error {
	return createVisitor.Visit(func(info *resource.Info, incomingErr error) error {
		if incomingErr != nil {
			return incomingErr
		}
		object, err := resource.NewHelper(info.Client, info.Mapping).
			WithFieldManager(o.FieldManager).
			WithFieldValidation(o.ValidationDirective).
			Create(info.Namespace, true, info.Object)
		if err != nil {
			return err
		}
		info.Refresh(object, true)
		printer, err := o.ToPrinter("created")
		if err != nil {
			return err
		}
		return printer.PrintObj(info.Object, o.Out)
	})
}

func (o *editOperation) visitAnnotation(annotationVisitor resource.Visitor) error {
	// iterate through all items to apply annotations
	err := annotationVisitor.Visit(func(info *resource.Info, incomingErr error) error {
		// put configuration annotation in "updates"
		if o.ApplyAnnotation {
			if err := saveConfigAnnotation(true, info.Object); err != nil {
				return err
			}
		}
		if err := o.Recorder.Record(info.Object); err != nil {
			klog.V(4).Infof("error recording current command: %v", err)
		}

		return nil
	})
	return err
}

// editMode selects the editor operation mode used by kubectl frontend commands.
type editMode string

const (
	// editModeNormal is an edit mode
	editModeNormal editMode = "normal_mode"

	// editModeApply is an edit mode
	editModeApply editMode = "edit_last_applied_mode"

	// editModeBeforeCreate edits objects before the create request is sent.
	editModeBeforeCreate editMode = "edit_before_create_mode"
)

// editReason preserves a message about the reason this file must be edited again
type editReason struct {
	head  string
	other []string
}

// editHeader includes a list of reasons the edit must be retried
type editHeader struct {
	reasons []editReason
}

// writeTo outputs the current header information into a stream
func (h *editHeader) writeTo(w io.Writer, editMode editMode) error {
	if editMode == editModeApply {
		fmt.Fprint(w, `# Please edit the 'last-applied-configuration' annotations below.
# Lines beginning with a '#' will be ignored, and an empty file will abort the edit.
#
`)
	} else {
		fmt.Fprint(w, `# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
`)
	}

	for _, r := range h.reasons {
		if len(r.other) > 0 {
			fmt.Fprintf(w, "# %s:\n", hashOnLineBreak(r.head))
		} else {
			fmt.Fprintf(w, "# %s\n", hashOnLineBreak(r.head))
		}
		for _, o := range r.other {
			fmt.Fprintf(w, "# * %s\n", hashOnLineBreak(o))
		}
		fmt.Fprintln(w, "#")
	}
	return nil
}

// editResults capture the result of an update
type editResults struct {
	header    editHeader
	retryable int
	notfound  int
	edit      []*resource.Info
	file      string
}

func (r *editResults) addError(err error, info *resource.Info) string {
	resourceString := info.Mapping.Resource.Resource
	if len(info.Mapping.Resource.Group) > 0 {
		resourceString = resourceString + "." + info.Mapping.Resource.Group
	}

	switch {
	case errors.IsInvalid(err):
		r.edit = append(r.edit, info)
		reason := editReason{
			head: fmt.Sprintf("%s %q was not valid", resourceString, info.Name),
		}
		if err, ok := err.(errors.APIStatus); ok {
			if details := err.Status().Details; details != nil {
				for _, cause := range details.Causes {
					reason.other = append(reason.other, fmt.Sprintf("%s: %s", cause.Field, cause.Message))
				}
			}
		}
		r.header.reasons = append(r.header.reasons, reason)
		return fmt.Sprintf("error: %s %q is invalid", resourceString, info.Name)
	case errors.IsNotFound(err):
		r.notfound++
		return fmt.Sprintf("error: %s %q could not be found on the server", resourceString, info.Name)
	default:
		r.retryable++
		return fmt.Sprintf("error: %s %q could not be patched: %v", resourceString, info.Name, err)
	}
}

// preservedFile writes out a message about the provided file if it exists to the
// provided output stream when an error happens. Used to notify the user where
// their updates were preserved.
func preservedFile(err error, path string, out io.Writer) error {
	if len(path) > 0 {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			fmt.Fprintf(out, "A copy of your changes has been stored to %q\n", path)
		}
	}
	return err
}

// hasLines returns true if any line in the provided stream is non empty - has non-whitespace
// characters, or the first non-whitespace character is a '#' indicating a comment. Returns
// any errors encountered reading the stream.
func hasLines(r io.Reader) (bool, error) {
	// TODO: if any files we read have > 64KB lines, we'll need to switch to bytes.ReadLine
	// TODO: probably going to be secrets
	s := bufio.NewScanner(r)
	for s.Scan() {
		if line := strings.TrimSpace(s.Text()); len(line) > 0 && line[0] != '#' {
			return true, nil
		}
	}
	if err := s.Err(); err != nil && err != io.EOF {
		return false, err
	}
	return false, nil
}

// hashOnLineBreak returns a string built from the provided string by inserting any necessary '#'
// characters after '\n' characters, indicating a comment.
func hashOnLineBreak(s string) string {
	r := ""
	for i, ch := range s {
		j := i + 1
		if j < len(s) && ch == '\n' && s[j] != '#' {
			r += "\n# "
		} else {
			r += string(ch)
		}
	}
	return r
}

// editorEnvs returns an ordered list of env vars to check for editor preferences.
func editorEnvs() []string {
	return []string{
		"KUBE_EDITOR",
		"EDITOR",
	}
}

func editInvalidError(err error) error {
	fieldErrors := field.ErrorList{
		field.Invalid(nil, "The edited file failed validation", fmt.Sprintf("%v", err)),
	}
	return errors.NewInvalid(
		editValidationGroupKind,
		"",
		fieldErrors,
	)
}
