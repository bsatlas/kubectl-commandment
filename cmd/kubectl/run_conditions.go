package main

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

func runPodCompleted(event watch.Event) (bool, error) {
	if event.Type == watch.Deleted {
		resource := schema.GroupResource{
			Resource: "pods",
		}
		return false, errors.NewNotFound(resource, "")
	}
	pod, ok := event.Object.(*v1.Pod)
	if !ok {
		return false, nil
	}
	return pod.Status.Phase == v1.PodFailed || pod.Status.Phase == v1.PodSucceeded, nil
}

func runPodSucceeded(event watch.Event) (bool, error) {
	if event.Type == watch.Deleted {
		resource := schema.GroupResource{
			Resource: "pods",
		}
		return false, errors.NewNotFound(resource, "")
	}
	pod, ok := event.Object.(*v1.Pod)
	if !ok {
		return false, nil
	}
	return pod.Status.Phase == v1.PodSucceeded, nil
}

func runPodRunningAndReady(event watch.Event) (bool, error) {
	if event.Type == watch.Deleted {
		resource := schema.GroupResource{
			Resource: "pods",
		}
		return false, errors.NewNotFound(resource, "")
	}
	pod, ok := event.Object.(*v1.Pod)
	if !ok {
		return false, nil
	}
	switch pod.Status.Phase {
	case v1.PodFailed, v1.PodSucceeded:
		return false, errRunPodCompleted
	case v1.PodRunning:
		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
				return true, nil
			}
		}
	}
	return false, nil
}
