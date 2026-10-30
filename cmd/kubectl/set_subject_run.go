package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/cmd/set"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
)

var (
	errSetSubjectServiceAccountNamespace = errors.New("serviceaccount must be <namespace>:<name>, namespace must be specified")
	errSetSubjectUnsupportedKind         = errors.New("setting subjects is only supported for RoleBinding/ClusterRoleBinding")
)

func runSetSubject(ctx context.Context, cfg *setSubjectConfig) error {
	strategy := cfg.DryRun

	factory := cfg.Client.factoryForInvocation(ctx)
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil && !(cfg.Local && clientcmd.IsEmptyConfig(err)) {
		return err
	}
	filename := resource.FilenameOptions{
		Filenames: cfg.Filename.Filenames(),
		Kustomize: cfg.Filename.Kustomize,
		Recursive: cfg.Filename.Recursive,
	}
	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		LocalParam(cfg.Local).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace()
	addFilenameSources(builder, enforceNamespace, filename, cfg.Streams.In)
	builder.Flatten()
	if !cfg.Local {
		builder.LabelSelectorParam(cfg.Selector).ResourceTypeOrNameArgs(cfg.All, cfg.Resources...).Latest()
	}
	infos, err := builder.Do().Infos()
	if err != nil {
		return err
	}

	for _, serviceAccount := range cfg.ServiceAccounts {
		for _, info := range infos {
			if _, ok := info.Object.(*v1.ClusterRoleBinding); ok && serviceAccount.namespace == "" {
				return errSetSubjectServiceAccountNamespace
			}
		}
	}

	printFlags := genericclioptions.NewPrintFlags("subjects updated").WithTypeSetter(scheme.Scheme)
	configurePrintFlagsWithOutputState(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	printObj, err := printObject(printFlags, strategy, cfg.Streams.Out)
	if err != nil {
		return err
	}

	subjects := make([]v1.Subject, 0, len(cfg.User)+len(cfg.Group)+len(cfg.ServiceAccounts))
	for _, user := range sets.List(sets.New[string](cfg.User...)) {
		subject := v1.Subject{
			Kind:     v1.UserKind,
			APIGroup: v1.GroupName,
			Name:     user,
		}
		subjects = append(subjects, subject)
	}
	for _, group := range sets.List(sets.New[string](cfg.Group...)) {
		subject := v1.Subject{
			Kind:     v1.GroupKind,
			APIGroup: v1.GroupName,
			Name:     group,
		}
		subjects = append(subjects, subject)
	}
	serviceAccounts := slices.Clone(cfg.ServiceAccounts)
	slices.SortFunc(serviceAccounts, func(a, b subjectServiceAccount) int {
		return cmp.Compare(a.namespace+":"+a.name, b.namespace+":"+b.name)
	})
	serviceAccounts = slices.Compact(serviceAccounts)
	for _, serviceAccount := range serviceAccounts {
		subjectNamespace := serviceAccount.namespace
		if subjectNamespace == "" {
			subjectNamespace = namespace
		}
		subject := v1.Subject{
			Kind:      v1.ServiceAccountKind,
			Namespace: subjectNamespace,
			Name:      serviceAccount.name,
		}
		subjects = append(subjects, subject)
	}

	patches := set.CalculatePatches(infos, scheme.DefaultJSONEncoder(), func(object runtime.Object) ([]byte, error) {
		changed, err := addBindingSubjects(object, subjects)
		if !changed || err != nil {
			return nil, err
		}
		return runtime.Encode(scheme.DefaultJSONEncoder(), object)
	})

	var allErrs []error
	for _, patch := range patches {
		info := patch.Info
		if patch.Err != nil {
			allErrs = append(allErrs, fmt.Errorf("error: %s %w\n", info.ObjectName(), patch.Err))
			continue
		}
		if len(patch.Patch) == 0 || string(patch.Patch) == "{}" {
			continue
		}
		if cfg.Local || strategy == util.DryRunClient {
			if err := printObj(info.Object); err != nil {
				allErrs = append(allErrs, err)
			}
			continue
		}
		actual, err := resource.NewHelper(info.Client, info.Mapping).
			DryRun(strategy == util.DryRunServer).
			WithFieldManager(cfg.FieldManager).
			Patch(info.Namespace, info.Name, types.StrategicMergePatchType, patch.Patch, nil)
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("failed to patch subjects to rolebinding: %w", err))
			continue
		}
		if err := printObj(actual); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	return utilerrors.NewAggregate(allErrs)
}

func addBindingSubjects(object runtime.Object, subjects []v1.Subject) (bool, error) {
	switch binding := object.(type) {
	case *v1.RoleBinding:
		updated, changed := appendMissingSubjects(binding.Subjects, subjects)
		binding.Subjects = updated
		return changed, nil
	case *v1.ClusterRoleBinding:
		updated, changed := appendMissingSubjects(binding.Subjects, subjects)
		binding.Subjects = updated
		return changed, nil
	default:
		return false, errSetSubjectUnsupportedKind
	}
}

func appendMissingSubjects(existing, targets []v1.Subject) ([]v1.Subject, bool) {
	updated := existing
	changed := false
	for _, target := range targets {
		found := false
		for _, current := range existing {
			if current == target {
				found = true
				break
			}
		}
		if !found {
			updated = append(updated, target)
			changed = true
		}
	}
	return updated, changed
}
