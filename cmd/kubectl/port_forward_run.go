package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubectl/pkg/cmd/portforward"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util"
)

func runPortForward(ctx context.Context, cfg *portForwardConfig) error {
	if len(cfg.Arguments) < 2 {
		return fmt.Errorf("TYPE/NAME and list of ports are required for port-forward")
	}

	factory := cfg.Client.factory()
	namespace, _, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		ResourceNames("pods", cfg.Arguments[0])
	object, err := builder.Do().Object()
	if err != nil {
		return err
	}
	pod, err := polymorphichelpers.AttachablePodForObjectFn(factory, object, cfg.PodRunningTimeout)
	if err != nil {
		return err
	}

	ports := cfg.Arguments[1:]
	switch object := object.(type) {
	case *v1.Service:
		if err := rejectUDPServicePorts(ports, object); err != nil {
			return err
		}
		ports, err = serviceForwardPorts(ports, *object, *pod)
	case *v1.Pod:
		if err := rejectUDPPodPorts(ports, object); err != nil {
			return err
		}
		ports, err = podForwardPorts(ports, *pod)
	default:
		if err := rejectUDPPodPorts(ports, pod); err != nil {
			return err
		}
		ports, err = podForwardPorts(ports, *pod)
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
	options.Address = append([]string(nil), cfg.Address...)
	options.Ports = ports
	options.StopChannel = make(chan struct{}, 1)
	options.ReadyChannel = make(chan struct{})
	if err := options.Validate(); err != nil {
		return err
	}
	return options.RunPortForwardContext(ctx)
}

func splitForwardPort(port string) (string, string) {
	local, remote, found := strings.Cut(port, ":")
	if found {
		return local, remote
	}
	return port, port
}

func serviceForwardPorts(ports []string, service v1.Service, pod v1.Pod) ([]string, error) {
	forwarded := make([]string, 0, len(ports))
	for _, port := range ports {
		localPort, remotePort := splitForwardPort(port)
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

func podForwardPorts(ports []string, pod v1.Pod) ([]string, error) {
	forwarded := make([]string, 0, len(ports))
	for _, port := range ports {
		localPort, remotePort := splitForwardPort(port)
		if remotePort == "" {
			return nil, fmt.Errorf("remote port cannot be empty")
		}
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

func rejectUDPServicePorts(ports []string, service *v1.Service) error {
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

func rejectUDPPodPorts(ports []string, pod *v1.Pod) error {
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

func rejectUDPPorts(udp sets.Set[int], ports []string, namedPort func(string) (int, error)) error {
	for _, port := range ports {
		_, remotePort := splitForwardPort(port)
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
