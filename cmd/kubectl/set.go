package main

import (
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	corev1 "k8s.io/api/core/v1"
)

type setConfig struct{}

func setCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[setConfig] {
	cmd := cli.Command[setConfig]{
		Name: "set",
		Description: cli.Description{
			Short: "Set specific features on objects",
			Long:  "Configure application resources.\n\n These commands help you make changes to existing application resources.",
		},
		Declare: func(cb *cli.CommandBuilder[setConfig]) {
			env := setEnvCommand(client)
			cb.Subcommands.Command(env)
			image := setImageCommand(client)
			cb.Subcommands.Command(image)
			resources := setResourcesCommand(client)
			cb.Subcommands.Command(resources)
			selector := setSelectorCommand(client)
			cb.Subcommands.Command(selector)
			serviceAccount := setServiceAccountCommand(client)
			cb.Subcommands.Command(serviceAccount)
			subject := setSubjectCommand(client)
			cb.Subcommands.Command(subject)
		},
	}
	return cmd
}

func selectContainers(containers []corev1.Container, pattern string) []*corev1.Container {
	selected := make([]*corev1.Container, 0, len(containers))
	for index := range containers {
		if matchContainerName(containers[index].Name, pattern) {
			selected = append(selected, &containers[index])
		}
	}
	return selected
}

func matchContainerName(name, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return name == pattern
	}

	position := 0
	parts := strings.Split(pattern, "*")
	for index, part := range parts {
		if part == "" {
			continue
		}

		next := strings.Index(name[position:], part)
		switch {
		case next < position:
			return false
		case index == 0 && position != 0:
			return false
		case index == len(parts)-1 && len(name) != len(part)+next:
			return false
		default:
			position = next
		}
	}
	return true
}
