package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

var podResourceCompletionTypes = [...]string{
	"daemonsets",
	"deployments",
	"pods",
	"jobs",
	"replicasets",
	"replicationcontrollers",
	"services",
	"statefulsets",
}

func podResourceArgumentCompleter[C any](inputs resourceCompletionInputs[C]) cli.ValueCompleter[C] {
	return func(ctx context.Context, cfg *C, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		client, arguments := inputs(cfg)
		if len(arguments) != 0 {
			return cli.CompletionContribution{}, nil
		}
		return completePodResource(ctx, client, request)
	}
}

func podResourceAndContainerArgumentCompleter[C any](inputs resourceCompletionInputs[C]) cli.ValueCompleter[C] {
	return func(ctx context.Context, cfg *C, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		client, arguments := inputs(cfg)
		switch len(arguments) {
		case 0:
			return completePodResource(ctx, client, request)
		case 1:
			podName, err := podNameForResource(client, arguments[0])
			if err != nil || podName == "" {
				return cli.CompletionContribution{}, err
			}
			return completePodContainers(ctx, client, podName, request.Text)
		default:
			return cli.CompletionContribution{}, nil
		}
	}
}

func containerCompleter[C any](client func(*C) clientConfig, resourceName func(*C) string) cli.ValueCompleter[C] {
	return func(ctx context.Context, cfg *C, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		name := resourceName(cfg)
		if name == "" {
			return cli.CompletionContribution{}, nil
		}
		selection := client(cfg)
		podName, err := podNameForResource(selection, name)
		if err != nil || podName == "" {
			return cli.CompletionContribution{}, err
		}
		return completePodContainers(ctx, selection, podName, request.Text)
	}
}

func resourceAndPortArgumentCompleter[C any](inputs resourceCompletionInputs[C]) cli.ValueCompleter[C] {
	return func(ctx context.Context, cfg *C, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		client, arguments := inputs(cfg)
		switch len(arguments) {
		case 0:
			return completePodResource(ctx, client, request)
		case 1:
			return completeResourcePorts(ctx, client, arguments[0], request.Text)
		default:
			return cli.CompletionContribution{}, nil
		}
	}
}

func completePodResource(ctx context.Context, client clientConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	text := request.Text
	slash := strings.Index(text, "/")
	if slash >= 0 {
		resourceType := text[:slash]
		completion, err := completeResourceNames(ctx, restClientGetter(client), resourceType, resourceCompletionQualified)
		if err != nil {
			return cli.CompletionContribution{}, err
		}
		completion.Candidates = filterCompletionCandidates(completion.Candidates, text, nil)
		return completion, nil
	}

	pods, err := completeResourceNames(ctx, restClientGetter(client), "pod", resourceCompletionBare)
	if err != nil {
		return cli.CompletionContribution{}, err
	}
	pods.Candidates = filterCompletionCandidates(pods.Candidates, text, nil)
	for _, resourceType := range podResourceCompletionTypes {
		if !strings.HasPrefix(resourceType, text) {
			continue
		}
		pods.Candidates = append(pods.Candidates, cli.CompletionCandidate{
			Value:   resourceType + "/",
			Kind:    cli.CompletionCandidateResource,
			Spacing: cli.CompletionSpacingNoSpace,
		})
	}
	sort.Slice(pods.Candidates, func(i, j int) bool {
		return pods.Candidates[i].Value < pods.Candidates[j].Value
	})
	return pods, nil
}

func podNameForResource(client clientConfig, resourceName string) (string, error) {
	if !strings.Contains(resourceName, "/") {
		return resourceName, nil
	}

	factory := client.factory()
	namespace, _, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return "", err
	}
	object, err := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		ContinueOnError().
		NamespaceParam(namespace).
		DefaultNamespace().
		ResourceNames("pods", resourceName).
		Do().
		Object()
	if err != nil {
		return "", err
	}
	pod, err := polymorphichelpers.AttachablePodForObjectFn(factory, object, 100*time.Millisecond)
	if err != nil {
		return "", err
	}
	return pod.Name, nil
}

func completePodContainers(ctx context.Context, client clientConfig, podName, prefix string) (cli.CompletionContribution, error) {
	factory := client.factory()
	namespace, _, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return cli.CompletionContribution{}, err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return cli.CompletionContribution{}, err
	}
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, v1.GetOptions{})
	if err != nil {
		return cli.CompletionContribution{}, err
	}
	values := make([]string, 0, len(pod.Spec.InitContainers)+len(pod.Spec.Containers))
	for _, container := range pod.Spec.InitContainers {
		if strings.HasPrefix(container.Name, prefix) {
			values = append(values, container.Name)
		}
	}
	for _, container := range pod.Spec.Containers {
		if strings.HasPrefix(container.Name, prefix) {
			values = append(values, container.Name)
		}
	}
	sort.Strings(values)
	return completionFromStrings(values, cli.CompletionCandidateResource, cli.CompletionSpacingDefault), nil
}

func completeResourcePorts(ctx context.Context, client clientConfig, resourceName, prefix string) (cli.CompletionContribution, error) {
	parts := strings.SplitN(resourceName, "/", 2)
	resourceType := "pod"
	name := parts[0]
	if len(parts) == 2 {
		resourceType = parts[0]
		name = parts[1]
	}

	factory := client.factory()
	namespace, _, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return cli.CompletionContribution{}, err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return cli.CompletionContribution{}, err
	}

	localPort := strings.SplitN(prefix, ":", 2)[0]
	values := make([]string, 0)
	if resourceType == "service" || resourceType == "services" || resourceType == "svc" {
		service, getErr := clientset.CoreV1().Services(namespace).Get(ctx, name, v1.GetOptions{})
		if getErr != nil {
			return cli.CompletionContribution{}, getErr
		}
		for _, port := range service.Spec.Ports {
			remote := strconv.Itoa(int(port.Port))
			local := localPort
			if local == "" {
				local = remote
			}
			value := fmt.Sprintf("%s:%s", local, remote)
			if strings.HasPrefix(value, prefix) {
				values = append(values, value)
			}
		}
	} else {
		podName, podErr := podNameForResource(client, resourceName)
		if podErr != nil {
			return cli.CompletionContribution{}, podErr
		}
		pod, getErr := clientset.CoreV1().Pods(namespace).Get(ctx, podName, v1.GetOptions{})
		if getErr != nil {
			return cli.CompletionContribution{}, getErr
		}
		for _, container := range pod.Spec.Containers {
			for _, port := range container.Ports {
				remote := strconv.Itoa(int(port.ContainerPort))
				local := localPort
				if local == "" {
					local = remote
				}
				value := fmt.Sprintf("%s:%s", local, remote)
				if strings.HasPrefix(value, prefix) {
					values = append(values, value)
				}
			}
		}
	}
	sort.Strings(values)
	return completionFromStrings(values, cli.CompletionCandidateValue, cli.CompletionSpacingDefault), nil
}
