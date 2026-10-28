package main

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/cmd/set"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
)

func runSetSubject(_ context.Context, cfg *setSubjectConfig) error {
	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	if cfg.Local && strategy == util.DryRunServer {
		return fmt.Errorf("cannot specify --local and --dry-run=server - did you mean --dry-run=client?")
	}
	if cfg.All && cfg.Selector != "" {
		return fmt.Errorf("cannot set --all and --selector at the same time")
	}
	if len(cfg.User) == 0 && len(cfg.Group) == 0 && len(cfg.ServiceAccounts) == 0 {
		return fmt.Errorf("you must specify at least one value of user, group or serviceaccount")
	}

	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil && !(cfg.Local && clientcmd.IsEmptyConfig(err)) {
		return err
	}
	filename := resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		LocalParam(cfg.Local).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &filename).
		Flatten()
	if cfg.Local {
		if len(cfg.Arguments) > 0 {
			return resource.LocalResourceError
		}
	} else {
		builder = builder.LabelSelectorParam(cfg.Selector).ResourceTypeOrNameArgs(cfg.All, cfg.Arguments...).Latest()
	}
	infos, err := builder.Do().Infos()
	if err != nil {
		return err
	}

	for _, serviceAccount := range cfg.ServiceAccounts {
		parts := strings.Split(serviceAccount, ":")
		if len(parts) != 2 || parts[1] == "" {
			return fmt.Errorf("serviceaccount must be <namespace>:<name>")
		}
		for _, info := range infos {
			if _, ok := info.Object.(*v1.ClusterRoleBinding); ok && parts[0] == "" {
				return fmt.Errorf("serviceaccount must be <namespace>:<name>, namespace must be specified")
			}
		}
	}

	printFlags := genericclioptions.NewPrintFlags("subjects updated").WithTypeSetter(scheme.Scheme)
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	printObj, err := printObject(printFlags, strategy, cfg.Streams.Out)
	if err != nil {
		return err
	}

	subjects := make([]v1.Subject, 0, len(cfg.User)+len(cfg.Group)+len(cfg.ServiceAccounts))
	for _, user := range sets.List(sets.New[string](cfg.User...)) {
		subjects = append(subjects, v1.Subject{
			Kind:     v1.UserKind,
			APIGroup: v1.GroupName,
			Name:     user,
		})
	}
	for _, group := range sets.List(sets.New[string](cfg.Group...)) {
		subjects = append(subjects, v1.Subject{
			Kind:     v1.GroupKind,
			APIGroup: v1.GroupName,
			Name:     group,
		})
	}
	for _, serviceAccount := range sets.List(sets.New[string](cfg.ServiceAccounts...)) {
		parts := strings.Split(serviceAccount, ":")
		subjectNamespace := parts[0]
		if subjectNamespace == "" {
			subjectNamespace = namespace
		}
		subjects = append(subjects, v1.Subject{
			Kind:      v1.ServiceAccountKind,
			Namespace: subjectNamespace,
			Name:      parts[1],
		})
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
			allErrs = append(allErrs, fmt.Errorf("error: %s %v\n", info.ObjectName(), patch.Err))
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
			allErrs = append(allErrs, fmt.Errorf("failed to patch subjects to rolebinding: %v", err))
			continue
		}
		if err := printObj(actual); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	return errors.NewAggregate(allErrs)
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
		return false, fmt.Errorf("setting subjects is only supported for RoleBinding/ClusterRoleBinding")
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
