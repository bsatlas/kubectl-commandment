package main

import (
	"fmt"
	"strconv"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/kubectl/pkg/cmd/util"
)

var dryRunValueLanguage = cli.Value.Domain(cli.Value.Assign[util.DryRunStrategy](), func(domain *cli.ValueDomainBuilder[util.DryRunStrategy]) {
	none := domain.Value("none", util.DryRunNone)
	client := domain.Value("client", util.DryRunClient)
	domain.Value("server", util.DryRunServer)
	for _, alias := range []string{"0", "f", "F", "FALSE", "false", "False"} {
		domain.Alias(alias, none).Hidden()
	}
	for _, alias := range []string{"1", "t", "T", "TRUE", "true", "True", "unchanged"} {
		domain.Alias(alias, client).Hidden()
	}
})

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
