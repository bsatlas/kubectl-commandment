/*
Copyright 2014 The Kubernetes Authors.

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

package main

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

type serviceExposure struct {
	defaultName string
	selector    map[string]string
	labels      map[string]string
	ports       []exposePort
	protocols   map[string][]string
}

func runExpose(_ context.Context, cfg *exposeConfig) error {
	strategy := cfg.DryRun

	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	mapper, err := factory.ToRESTMapper()
	if err != nil {
		return err
	}

	filename := resource.FilenameOptions{
		Filenames: cfg.Filename.Filenames(),
		Kustomize: cfg.Filename.Kustomize,
		Recursive: cfg.Filename.Recursive,
	}
	result := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &filename).
		ResourceTypeOrNameArgs(false, cfg.Arguments...).
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("exposed").WithTypeSetter(scheme.Scheme)
	configurePrintFlagsWithOutputState(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	util.PrintFlagsWithDryRunStrategy(printFlags, strategy)
	printer, err := printFlags.ToPrinter()
	if err != nil {
		return err
	}

	recordFlags := genericclioptions.NewRecordFlags()
	*recordFlags.Record = cfg.Record
	if cfg.Record {
		if err := recordFlags.CompleteWithChangeCause("kubectl expose"); err != nil {
			return err
		}
	}
	recorder, err := recordFlags.ToRecorder()
	if err != nil {
		return err
	}

	return result.Visit(func(info *resource.Info, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		if err := polymorphichelpers.CanBeExposedFn(info.ResourceMapping().GroupVersionKind.GroupKind()); err != nil {
			return err
		}

		defaultName := info.Name
		if len(defaultName) > validation.DNS1035LabelMaxLength {
			defaultName = defaultName[:validation.DNS1035LabelMaxLength]
		}
		selector := cfg.Selector.Values
		if cfg.Selector.Text == "" {
			discovered, err := polymorphichelpers.MapBasedSelectorForObjectFn(info.Object)
			if err != nil {
				return fmt.Errorf("couldn't retrieve selectors via --selector flag or introspection: %v", err)
			}
			parsed, err := parseExposeLabelSet(discovered)
			if err != nil {
				return err
			}
			selector = parsed.Values
		}

		ports := cfg.Port.Values
		if cfg.Port.Text == "" {
			discovered, err := polymorphichelpers.PortsForObjectFn(info.Object)
			if err != nil {
				return fmt.Errorf("couldn't find port via --port flag or introspection: %v", err)
			}
			if len(discovered) == 0 && cfg.ClusterIP != "None" {
				return fmt.Errorf("couldn't find port via --port flag or introspection")
			}
			if len(discovered) > 0 {
				parsed, err := parseExposePortSet(strings.Join(discovered, ","))
				if err != nil {
					return err
				}
				ports = parsed.Values
			}
		}

		protocols, err := polymorphichelpers.MultiProtocolsForObjectFn(info.Object)
		if err != nil {
			return fmt.Errorf("couldn't find protocol via introspection: %v", err)
		}

		labels := cfg.Labels.Values
		if cfg.Labels.Text == "" {
			objectLabels, err := meta.NewAccessor().Labels(info.Object)
			if err != nil {
				return err
			}
			labels = maps.Clone(objectLabels)
		}

		exposure := serviceExposure{
			defaultName: defaultName,
			selector:    selector,
			labels:      labels,
			ports:       ports,
			protocols:   protocols,
		}
		service, err := newExposedService(cfg, exposure)
		if err != nil {
			return err
		}
		overridden, err := applyOverrides(service, &v1.Service{}, cfg.Overrides, string(cfg.OverrideType))
		if err != nil {
			return err
		}
		if err := recorder.Record(overridden); err != nil {
			klog.V(4).Infof("error recording current command: %v", err)
		}

		if strategy == util.DryRunClient {
			if accessor, err := meta.Accessor(overridden); err == nil && enforceNamespace {
				accessor.SetNamespace(namespace)
			}
			return printer.PrintObj(overridden, cfg.Streams.Out)
		}
		if err := saveConfigAnnotation(cfg.SaveConfig, overridden); err != nil {
			return err
		}

		object := &unstructured.Unstructured{}
		if err := scheme.Scheme.Convert(overridden, object, nil); err != nil {
			return err
		}
		gvks, _, err := unstructuredscheme.NewUnstructuredObjectTyper().ObjectKinds(object)
		if err != nil {
			return err
		}
		mapping, err := mapper.RESTMapping(gvks[0].GroupKind(), gvks[0].Version)
		if err != nil {
			return err
		}
		client, err := factory.ClientForMapping(mapping)
		if err != nil {
			return err
		}
		created, err := resource.NewHelper(client, mapping).
			DryRun(strategy == util.DryRunServer).
			WithFieldManager(cfg.FieldManager).
			Create(namespace, false, object)
		if err != nil {
			return err
		}
		return printer.PrintObj(created, cfg.Streams.Out)
	})
}

func newExposedService(cfg *exposeConfig, exposure serviceExposure) (*v1.Service, error) {
	name := cfg.Name
	if name == "" {
		name = exposure.defaultName
	}
	if name == "" {
		return nil, fmt.Errorf("name must be specified")
	}

	var servicePorts []v1.ServicePort
	for index, port := range exposure.ports {
		portName := ""
		if len(exposure.ports) > 1 {
			portName = fmt.Sprintf("port-%d", index+1)
		}
		protocol := cfg.Protocol
		switch {
		case protocol == "" && len(exposure.protocols) == 0:
			protocol = "TCP"
		case protocol == "" && len(exposure.protocols) > 0:
			protocol = "TCP"
			if values, ok := exposure.protocols[port.Text]; ok {
				if len(values) == 1 {
					protocol = values[0]
				} else {
					for _, candidate := range values {
						servicePorts = append(servicePorts, v1.ServicePort{
							Name:     fmt.Sprintf("port-%d-%s", index+1, strings.ToLower(candidate)),
							Port:     port.Number,
							Protocol: v1.Protocol(candidate),
						})
					}
					continue
				}
			}
		}
		servicePorts = append(servicePorts, v1.ServicePort{
			Name:     portName,
			Port:     port.Number,
			Protocol: v1.Protocol(protocol),
		})
	}

	service := &v1.Service{}
	service.Name = name
	service.Labels = maps.Clone(exposure.labels)
	service.Spec.Selector = maps.Clone(exposure.selector)
	service.Spec.Ports = servicePorts
	for index := range service.Spec.Ports {
		if cfg.TargetPort != "" {
			service.Spec.Ports[index].TargetPort = intstr.Parse(cfg.TargetPort)
		} else {
			service.Spec.Ports[index].TargetPort = intstr.FromInt32(service.Spec.Ports[index].Port)
		}
	}
	if cfg.ExternalIP != "" {
		service.Spec.ExternalIPs = []string{cfg.ExternalIP}
	}
	if cfg.Type != "" {
		service.Spec.Type = v1.ServiceType(cfg.Type)
	}
	if service.Spec.Type == v1.ServiceTypeLoadBalancer {
		service.Spec.LoadBalancerIP = cfg.LoadBalancerIP
	}
	if cfg.SessionAffinity != "" {
		service.Spec.SessionAffinity = cfg.SessionAffinity
	}
	if cfg.ClusterIP != "" {
		service.Spec.ClusterIP = cfg.ClusterIP
	}
	return service, nil
}
