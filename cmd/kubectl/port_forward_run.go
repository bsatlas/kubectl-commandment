package main

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubectl/pkg/cmd/portforward"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util"
)

func runPortForward(ctx context.Context, cfg *portForwardConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	namespace, _, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		ResourceNames("pods", cfg.Resource)
	object, err := builder.Do().Object()
	if err != nil {
		return err
	}
	pod, err := polymorphichelpers.AttachablePodForObjectFn(factory, object, cfg.PodRunningTimeout)
	if err != nil {
		return err
	}

	ports := cfg.Ports
	var forwardPorts []string
	switch object := object.(type) {
	case *v1.Service:
		if err := rejectUDPServicePorts(ports, object); err != nil {
			return err
		}
		forwardPorts, err = serviceForwardPorts(ports, *object, *pod)
	case *v1.Pod:
		if err := rejectUDPPodPorts(ports, object); err != nil {
			return err
		}
		forwardPorts, err = podForwardPorts(ports, *pod)
	default:
		if err := rejectUDPPodPorts(ports, pod); err != nil {
			return err
		}
		forwardPorts, err = podForwardPorts(ports, *pod)
	}
	if err != nil {
		return err
	}

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	restClient, err := factory.RESTClient()
	if err != nil {
		return err
	}

	options := portforward.NewDefaultPortForwardOptions(cfg.Streams)
	options.Namespace = namespace
	options.PodName = pod.Name
	options.PodClient = clientset.CoreV1()
	options.Config = restConfig
	options.RESTClient = restClient
	options.Address = slices.Clone(cfg.Address)
	options.Ports = forwardPorts
	options.StopChannel = make(chan struct{}, 1)
	options.ReadyChannel = make(chan struct{})
	return options.RunPortForwardContext(ctx)
}

func serviceForwardPorts(ports []portForwardPort, service v1.Service, pod v1.Pod) ([]string, error) {
	forwarded := make([]string, 0, len(ports))
	for _, port := range ports {
		localPort, remotePort := port.Local, port.Remote
		portNumber, err := strconv.Atoi(remotePort)
		if err != nil {
			servicePort, err := util.LookupServicePortNumberByName(service, remotePort)
			if err != nil {
				return nil, err
			}
			portNumber = int(servicePort)
			if localPort == remotePort {
				localPort = strconv.Itoa(portNumber)
			}
		}
		containerPort, err := util.LookupContainerPortNumberByServicePort(service, pod, int32(portNumber))
		if err != nil {
			return nil, err
		}
		remotePort = strconv.Itoa(int(containerPort))
		if localPort != remotePort {
			forwarded = append(forwarded, fmt.Sprintf("%s:%s", localPort, remotePort))
		} else {
			forwarded = append(forwarded, remotePort)
		}
	}
	return forwarded, nil
}

func podForwardPorts(ports []portForwardPort, pod v1.Pod) ([]string, error) {
	forwarded := make([]string, 0, len(ports))
	for _, port := range ports {
		localPort, remotePort := port.Local, port.Remote
		containerPort := remotePort
		if _, err := strconv.Atoi(remotePort); err != nil {
			portNumber, err := util.LookupContainerPortNumberByName(pod, remotePort)
			if err != nil {
				return nil, err
			}
			containerPort = strconv.Itoa(int(portNumber))
		}
		if localPort != remotePort {
			forwarded = append(forwarded, fmt.Sprintf("%s:%s", localPort, containerPort))
		} else {
			forwarded = append(forwarded, containerPort)
		}
	}
	return forwarded, nil
}

func rejectUDPServicePorts(ports []portForwardPort, service *v1.Service) error {
	udp := sets.New[int]()
	tcp := sets.New[int]()
	for _, port := range service.Spec.Ports {
		number := int(port.Port)
		switch port.Protocol {
		case v1.ProtocolUDP:
			udp.Insert(number)
		case v1.ProtocolTCP:
			tcp.Insert(number)
		}
	}
	return rejectUDPPorts(udp.Difference(tcp), ports, func(name string) (int, error) {
		port, err := util.LookupServicePortNumberByName(*service, name)
		return int(port), err
	})
}

func rejectUDPPodPorts(ports []portForwardPort, pod *v1.Pod) error {
	udp := sets.New[int]()
	tcp := sets.New[int]()
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			number := int(port.ContainerPort)
			switch port.Protocol {
			case v1.ProtocolUDP:
				udp.Insert(number)
			case v1.ProtocolTCP:
				tcp.Insert(number)
			}
		}
	}
	return rejectUDPPorts(udp.Difference(tcp), ports, func(name string) (int, error) {
		port, err := util.LookupContainerPortNumberByName(*pod, name)
		return int(port), err
	})
}

func rejectUDPPorts(udp sets.Set[int], ports []portForwardPort, namedPort func(string) (int, error)) error {
	for _, port := range ports {
		remotePort := port.Remote
		portNumber, err := strconv.Atoi(remotePort)
		if err != nil {
			portNumber, err = namedPort(remotePort)
			if err != nil {
				return err
			}
		}
		if udp.Has(portNumber) {
			return fmt.Errorf("UDP protocol is not supported for %s", remotePort)
		}
	}
	return nil
}
