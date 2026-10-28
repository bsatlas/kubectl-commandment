package main

import "k8s.io/utils/exec"

func runCodeExitError(err error, code int) error {
	failure := exec.CodeExitError{
		Err:  err,
		Code: code,
	}
	return failure
}
