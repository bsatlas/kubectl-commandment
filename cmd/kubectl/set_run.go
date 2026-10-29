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
