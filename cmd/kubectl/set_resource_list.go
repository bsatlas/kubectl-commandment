package main

import (
	"fmt"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func parseResourceList(spec string) (v1.ResourceList, error) {
	if spec == "" {
		return nil, nil
	}
	result := v1.ResourceList{}
	for _, statement := range strings.Split(spec, ",") {
		parts := strings.Split(statement, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid argument syntax %v, expected <resource>=<value>", statement)
		}
		quantity, err := resource.ParseQuantity(parts[1])
		if err != nil {
			return nil, err
		}
		result[v1.ResourceName(parts[0])] = quantity
	}
	return result, nil
}
