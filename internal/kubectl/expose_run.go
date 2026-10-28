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

package kubectl

import (
	"context"
	"fmt"
	"slices"
	"strconv"
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
	selector    string
	labels      string
	port        string
	ports       string
	protocols   string
}

func runExpose(_ context.Context, cfg *exposeConfig) error {
	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}

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
		Filenames: slices.Clone(cfg.Filename),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
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
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
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
		selector := cfg.Selector
		if selector == "" {
			selector, err = polymorphichelpers.MapBasedSelectorForObjectFn(info.Object)
			if err != nil {
				return fmt.Errorf("couldn't retrieve selectors via --selector flag or introspection: %v", err)
			}
		}

		port := cfg.Port
		ports := ""
		if port == "" {
			discovered, err := polymorphichelpers.PortsForObjectFn(info.Object)
			if err != nil {
				return fmt.Errorf("couldn't find port via --port flag or introspection: %v", err)
			}
			switch len(discovered) {
			case 0:
				if cfg.ClusterIP != "None" {
					return fmt.Errorf("couldn't find port via --port flag or introspection")
				}
			case 1:
				port = discovered[0]
			default:
				ports = strings.Join(discovered, ",")
			}
		}

		protocolMap, err := polymorphichelpers.MultiProtocolsForObjectFn(info.Object)
		if err != nil {
			return fmt.Errorf("couldn't find protocol via introspection: %v", err)
		}
		protocols := exposeProtocols(protocolMap)

		labels := cfg.Labels
		if labels == "" {
			objectLabels, err := meta.NewAccessor().Labels(info.Object)
			if err != nil {
				return err
			}
			labels = polymorphichelpers.MakeLabels(objectLabels)
		}

		exposure := serviceExposure{
			defaultName: defaultName,
			selector:    selector,
			labels:      labels,
			port:        port,
			ports:       ports,
			protocols:   protocols,
		}
		service, err := newExposedService(cfg, exposure)
		if err != nil {
			return err
		}
		overridden, err := applyOverrides(service, &v1.Service{}, cfg.Overrides, cfg.OverrideType)
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
	selector, err := parseExposeLabels(exposure.selector)
	if err != nil {
		return nil, err
	}
	var labels map[string]string
	if exposure.labels != "" {
		labels, err = parseExposeLabels(exposure.labels)
		if err != nil {
			return nil, err
		}
	}

	name := cfg.Name
	if name == "" {
		name = exposure.defaultName
	}
	if name == "" {
		return nil, fmt.Errorf("name must be specified")
	}

	var protocolMap map[string][]string
	if exposure.protocols != "" {
		protocolMap, err = parseExposeProtocols(exposure.protocols)
		if err != nil {
			return nil, err
		}
	}
	portSpec := exposure.ports
	if portSpec == "" {
		portSpec = exposure.port
	}
	var servicePorts []v1.ServicePort
	if portSpec != "" {
		items := strings.Split(portSpec, ",")
		for index, item := range items {
			value, err := strconv.Atoi(item)
			if err != nil {
				return nil, err
			}
			portName := ""
			if len(items) > 1 {
				portName = fmt.Sprintf("port-%d", index+1)
			}
			protocol := cfg.Protocol
			switch {
			case protocol == "" && len(protocolMap) == 0:
				protocol = "TCP"
			case protocol == "" && len(protocolMap) > 0:
				protocol = "TCP"
				if values, ok := protocolMap[item]; ok {
					if len(values) == 1 {
						protocol = values[0]
					} else {
						for _, candidate := range values {
							servicePorts = append(servicePorts, v1.ServicePort{
								Name:     fmt.Sprintf("port-%d-%s", index+1, strings.ToLower(candidate)),
								Port:     int32(value),
								Protocol: v1.Protocol(candidate),
							})
						}
						continue
					}
				}
			}
			servicePorts = append(servicePorts, v1.ServicePort{
				Name:     portName,
				Port:     int32(value),
				Protocol: v1.Protocol(protocol),
			})
		}
	}

	service := &v1.Service{}
	service.Name = name
	service.Labels = labels
	service.Spec.Selector = selector
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
		switch v1.ServiceAffinity(cfg.SessionAffinity) {
		case v1.ServiceAffinityNone:
			service.Spec.SessionAffinity = v1.ServiceAffinityNone
		case v1.ServiceAffinityClientIP:
			service.Spec.SessionAffinity = v1.ServiceAffinityClientIP
		default:
			return nil, fmt.Errorf("unknown session affinity: %s", cfg.SessionAffinity)
		}
	}
	if cfg.ClusterIP != "" {
		service.Spec.ClusterIP = cfg.ClusterIP
	}
	return service, nil
}

func parseExposeLabels(spec string) (map[string]string, error) {
	if spec == "" {
		return nil, fmt.Errorf("no label spec passed")
	}
	result := make(map[string]string)
	for item := range strings.SplitSeq(spec, ",") {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			return nil, fmt.Errorf("unexpected label spec: %s", item)
		}
		if key == "" {
			return nil, fmt.Errorf("unexpected empty label key")
		}
		result[key] = value
	}
	return result, nil
}

func exposeProtocols(protocols map[string][]string) string {
	var values []string
	for port, candidates := range protocols {
		for _, protocol := range candidates {
			values = append(values, fmt.Sprintf("%s/%s", port, protocol))
		}
	}
	return strings.Join(values, ",")
}

func parseExposeProtocols(spec string) (map[string][]string, error) {
	if spec == "" {
		return nil, fmt.Errorf("no protocols passed")
	}
	result := make(map[string][]string)
	for item := range strings.SplitSeq(spec, ",") {
		port, protocol, ok := strings.Cut(item, "/")
		if !ok || port == "" || protocol == "" {
			return nil, fmt.Errorf("unexpected port protocol mapping: %s", item)
		}
		result[port] = append(result[port], protocol)
	}
	return result, nil
}
