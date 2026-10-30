package main

import (
	"strconv"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/net"
)

var servicePortLanguage = cli.ValueCodec[string]{
	Name: "service-port",
	Parse: func(value string) (string, error) {
		parts := strings.Split(value, ":")
		if len(parts) > 2 {
			return "", cli.Rejection.Message("service-port", "expected port or port:targetPort", cli.Rejection.At(0, 0, len(value)))
		}
		if _, err := net.ParsePort(parts[0], true); err != nil {
			return "", cli.Rejection.Message("service-port", err.Error(), cli.Rejection.At(0, 0, len(value)))
		}
		if len(parts) == 2 {
			if port, err := strconv.Atoi(parts[1]); err != nil {
				if problems := validation.IsValidPortName(parts[1]); len(problems) != 0 {
					return "", cli.Rejection.Message("service-port", strings.Join(problems, ","), cli.Rejection.At(0, 0, len(value)))
				}
			} else if problems := validation.IsValidPortNum(port); len(problems) != 0 {
				return "", cli.Rejection.Message("service-port", strings.Join(problems, ","), cli.Rejection.At(0, 0, len(value)))
			}
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

func createServiceCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name:    "service",
		Aliases: cli.Aliases("svc"),
		Description: cli.Description{
			Short: "Create a service using a specified subcommand",
			Long:  "Create a service using a specified subcommand.",
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			clusterIP := createServiceClusterIPCommand(client)
			cb.Subcommands.Command(clusterIP)
			externalName := createServiceExternalNameCommand(client)
			cb.Subcommands.Command(externalName)
			loadBalancer := createServiceLoadBalancerCommand(client)
			cb.Subcommands.Command(loadBalancer)
			nodePort := createServiceNodePortCommand(client)
			cb.Subcommands.Command(nodePort)
		},
	}
	return cmd
}
