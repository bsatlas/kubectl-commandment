package main

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getTableV1MediaType() string {
	return fmt.Sprintf("application/json;as=Table;v=%s;g=%s", v1.SchemeGroupVersion.Version, v1.GroupName)
}
