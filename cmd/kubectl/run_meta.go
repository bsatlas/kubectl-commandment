package main

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func runObjectMeta(name string, labels, annotations map[string]string) v1.ObjectMeta {
	meta := v1.ObjectMeta{
		Name:        name,
		Labels:      labels,
		Annotations: annotations,
	}
	return meta
}

func runPodListWatch(clientset kubernetes.Interface, namespace, name string, ctx context.Context) *cache.ListWatch {
	selector := fields.OneTermEqualSelector("metadata.name", name).String()
	listWatch := cache.ListWatch{
		ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = selector
			return clientset.CoreV1().Pods(namespace).List(ctx, options)
		},
		WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = selector
			return clientset.CoreV1().Pods(namespace).Watch(ctx, options)
		},
	}
	return &listWatch
}

func runTimestampNow() *v1.Time {
	now := v1.Now()
	return &now
}
