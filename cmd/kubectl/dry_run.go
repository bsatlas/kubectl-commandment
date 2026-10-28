package main

import (
	"fmt"
	"strconv"

	"k8s.io/kubectl/pkg/cmd/util"
)

func dryRunStrategy(value string) (util.DryRunStrategy, error) {
	boolean, err := strconv.ParseBool(value)
	if err == nil {
		if boolean {
			return util.DryRunClient, nil
		}
		return util.DryRunNone, nil
	}

	switch value {
	case "unchanged", "client":
		return util.DryRunClient, nil
	case "server":
		return util.DryRunServer, nil
	case "none", "":
		return util.DryRunNone, nil
	default:
		return util.DryRunNone, fmt.Errorf("invalid dry-run value %q; must be none, server, or client", value)
	}
}
