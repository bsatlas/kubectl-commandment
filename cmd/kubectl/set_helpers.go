package main

import (
	"strings"

	"k8s.io/api/core/v1"
)

func selectContainers(containers []v1.Container, pattern string) []*v1.Container {
	selected := make([]*v1.Container, 0, len(containers))
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
	parts := strings.Split(pattern, "*")
	position := 0
	for index, part := range parts {
		if part == "" {
			continue
		}
		relative := strings.Index(name[position:], part)
		if relative < 0 {
			return false
		}
		absolute := position + relative
		if index == 0 && absolute != 0 {
			return false
		}
		position = absolute + len(part)
		if index == len(parts)-1 && position != len(name) {
			return false
		}
	}
	return true
}
