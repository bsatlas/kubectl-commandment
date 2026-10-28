/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubectl

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/set"
	"k8s.io/kubectl/pkg/cmd/set/env"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

var invalidEnvName = regexp.MustCompile("[^a-zA-Z0-9_]")

func resolveEnvChanges(in io.Reader, groups ...[]envChange) ([]v1.EnvVar, []string, bool, error) {
	var environment []v1.EnvVar
	var remove []string
	usedStdin := false
	for _, group := range groups {
		for _, change := range group {
			switch change.kind {
			case envChangeSet:
				variable := v1.EnvVar{
					Name:  change.name,
					Value: change.value,
				}
				environment = append(environment, variable)
			case envChangeRemove:
				remove = append(remove, change.name)
			case envChangeStdin:
				fromStdin, _, used, err := env.ParseEnv([]string{"-"}, in)
				if err != nil {
					return nil, nil, usedStdin, err
				}
				environment = append(environment, fromStdin...)
				usedStdin = usedStdin || used
			}
		}
	}
	if envSetAndRemoved(environment, remove) {
		return nil, nil, usedStdin, fmt.Errorf("can not both modify and remove the same environment variable in the same command")
	}
	return environment, remove, usedStdin, nil
}

func envSetAndRemoved(environment []v1.EnvVar, remove []string) bool {
	set := make(map[string]struct{}, len(environment))
	for _, variable := range environment {
		set[variable.Name] = struct{}{}
	}
	for _, name := range remove {
		if _, ok := set[name]; ok {
			return true
		}
	}
	return false
}

func runSetEnv(_ context.Context, cfg *setEnvConfig) error {
	resources, positionalEnvironment := splitEnvArguments(cfg.Arguments)
	strategy := cfg.DryRun

	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}

	environment, remove, envFromStdin, err := resolveEnvChanges(cfg.Streams.In, cfg.Env, positionalEnvironment)
	if err != nil {
		return err
	}

	filename := resource.FilenameOptions{
		Filenames: slices.Clone(cfg.Filename),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	warningPrinter := printers.NewWarningPrinter(cfg.Streams.ErrOut, printers.WarningPrinterOptions{
		Color: printers.AllowsColorOutput(cfg.Streams.ErrOut),
	})
	if cfg.From != "" {
		builder := factory.NewBuilder().
			WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
			LocalParam(cfg.Local).
			ContinueOnError().
			NamespaceParam(namespace).DefaultNamespace().
			FilenameParam(enforceNamespace, &filename).
			Flatten()
		if !cfg.Local {
			builder.LabelSelectorParam(cfg.Selector).ResourceTypeOrNameArgs(cfg.All, cfg.From).Latest()
		}
		if envFromStdin {
			builder.StdinInUse()
		}
		infos, err := builder.Do().Infos()
		if err != nil {
			return err
		}
		for _, info := range infos {
			switch source := info.Object.(type) {
			case *v1.Secret:
				for key := range source.Data {
					if envKeySelected(key, cfg.Keys) {
						environment = append(environment, v1.EnvVar{
							Name: envName(key, warningPrinter),
							ValueFrom: &v1.EnvVarSource{
								SecretKeyRef: &v1.SecretKeySelector{
									LocalObjectReference: v1.LocalObjectReference{
										Name: source.Name,
									},
									Key: key,
								},
							},
						})
					}
				}
			case *v1.ConfigMap:
				for key := range source.Data {
					if envKeySelected(key, cfg.Keys) {
						environment = append(environment, v1.EnvVar{
							Name: envName(key, warningPrinter),
							ValueFrom: &v1.EnvVarSource{
								ConfigMapKeyRef: &v1.ConfigMapKeySelector{
									LocalObjectReference: v1.LocalObjectReference{
										Name: source.Name,
									},
									Key: key,
								},
							},
						})
					}
				}
			default:
				return fmt.Errorf("unsupported resource specified in --from")
			}
		}
	}
	if cfg.Prefix != "" {
		for index := range environment {
			environment[index].Name = cfg.Prefix + environment[index].Name
		}
	}

	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		LocalParam(cfg.Local).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &filename).
		Flatten()
	if !cfg.Local {
		builder.LabelSelectorParam(cfg.Selector).ResourceTypeOrNameArgs(cfg.All, resources...).Latest()
	}
	if envFromStdin {
		builder.StdinInUse()
	}
	infos, err := builder.Do().Infos()
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("env updated").WithTypeSetter(scheme.Scheme)
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	printObj, err := printObject(printFlags, strategy, cfg.Streams.Out)
	if err != nil {
		return err
	}

	patches := set.CalculatePatches(infos, scheme.DefaultJSONEncoder(), func(object runtime.Object) ([]byte, error) {
		resolutionFailed := false
		_, err := polymorphichelpers.UpdatePodSpecForObjectFn(object, func(spec *v1.PodSpec) error {
			containers := append(selectContainers(spec.Containers, cfg.Containers), selectContainers(spec.InitContainers, cfg.Containers)...)
			name, err := meta.NewAccessor().Name(object)
			if err != nil {
				return err
			}
			kind := objectKind(object)
			if len(containers) == 0 {
				warningPrinter.Print(fmt.Sprintf("%s/%s does not have any containers matching %q", kind, name, cfg.Containers))
				return nil
			}
			for _, container := range containers {
				if !cfg.Overwrite {
					if err := validateEnvNoOverwrite(container.Env, environment); err != nil {
						return err
					}
				}
				container.Env = mergeEnv(container.Env, environment, remove)
				if !cfg.List {
					continue
				}
				resolveErrors := map[string][]string{}
				store := env.NewResourceStore()
				fmt.Fprintf(cfg.Streams.Out, "# %s %s, container %s\n", kind, name, container.Name)
				for _, variable := range container.Env {
					if variable.ValueFrom == nil {
						fmt.Fprintf(cfg.Streams.Out, "%s=%s\n", variable.Name, variable.Value)
						continue
					}
					if !cfg.Resolve {
						fmt.Fprintf(cfg.Streams.Out, "# %s from %s\n", variable.Name, env.GetEnvVarRefString(variable.ValueFrom))
						continue
					}
					value, err := env.GetEnvVarRefValue(clientset, namespace, store, variable.ValueFrom, object, container)
					if err == nil {
						fmt.Fprintf(cfg.Streams.Out, "%s=%s\n", variable.Name, value)
						continue
					}
					fmt.Fprintf(cfg.Streams.Out, "# %s from %s\n", variable.Name, env.GetEnvVarRefString(variable.ValueFrom))
					resolveErrors[err.Error()] = append(resolveErrors[err.Error()], variable.Name)
					resolutionFailed = true
				}
				messages := make([]string, 0, len(resolveErrors))
				for message, variables := range resolveErrors {
					slices.Sort(variables)
					messages = append(messages, fmt.Sprintf("error retrieving reference for %s: %v", strings.Join(variables, ", "), message))
				}
				slices.Sort(messages)
				for _, message := range messages {
					fmt.Fprintln(cfg.Streams.ErrOut, message)
				}
			}
			if resolutionFailed {
				return fmt.Errorf("failed to retrieve valueFrom references")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return runtime.Encode(scheme.DefaultJSONEncoder(), object)
	})
	if cfg.List {
		return nil
	}

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
			allErrs = append(allErrs, fmt.Errorf("failed to patch env update to pod template: %v", err))
			continue
		}
		if len(environment) == 0 && len(positionalEnvironment) == 0 {
			return fmt.Errorf("at least one environment variable must be provided")
		}
		if err := printObj(actual); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	return errors.NewAggregate(allErrs)
}

func envKeySelected(key string, keys []string) bool {
	if len(keys) == 0 {
		return true
	}
	for _, candidate := range keys {
		if candidate == key {
			return true
		}
	}
	return false
}

func envName(key string, warningPrinter *printers.WarningPrinter) string {
	name := strings.ToUpper(invalidEnvName.ReplaceAllString(key, "_"))
	if name != key {
		warningPrinter.Print(fmt.Sprintf("key %s transferred to %s", key, name))
	}
	return name
}

func objectKind(object runtime.Object) string {
	kind := object.GetObjectKind().GroupVersionKind().Kind
	if kind != "" {
		return kind
	}
	gvks, _, err := scheme.Scheme.ObjectKinds(object)
	if err != nil {
		return kind
	}
	for _, gvk := range gvks {
		if gvk.Kind != "" && gvk.Version != "" && gvk.Version != runtime.APIVersionInternal {
			return gvk.Kind
		}
	}
	return kind
}

func validateEnvNoOverwrite(existing, updates []v1.EnvVar) error {
	for _, update := range updates {
		for _, current := range existing {
			if current.Name == update.Name && current.Value != update.Value {
				return fmt.Errorf("'%s' already has a value (%s), and --overwrite is false", current.Name, current.Value)
			}
		}
	}
	return nil
}

func mergeEnv(existing, updates []v1.EnvVar, remove []string) []v1.EnvVar {
	index := make(map[string]int, len(existing))
	result := slices.Clone(existing)
	for position := range result {
		index[result[position].Name] = position
	}
	for _, update := range updates {
		if position, ok := index[update.Name]; ok {
			result[position] = update
			continue
		}
		index[update.Name] = len(result)
		result = append(result, update)
	}
	if len(remove) == 0 {
		return result
	}
	removed := make(map[string]struct{}, len(remove))
	for _, name := range remove {
		removed[name] = struct{}{}
	}
	filtered := result[:0]
	for _, variable := range result {
		if _, ok := removed[variable.Name]; !ok {
			filtered = append(filtered, variable)
		}
	}
	return filtered
}
