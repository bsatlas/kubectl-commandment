package main

import (
	"github.com/abstractionsinc/commandment/cli"
)

func topCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name: "top",
		Description: cli.Description{
			Short: "Display resource (CPU/memory) usage",
			Long:  "Display resource (CPU/memory) usage.\n\n This command provides a view of recent resource consumption for nodes and pods. It fetches metrics from the Metrics Server, which aggregates this data from the kubelet on each node. The Metrics Server must be installed and running in the cluster for this command to work.\n\n The metrics shown are specifically optimized for Kubernetes autoscaling decisions, such as those made by the Horizontal Pod Autoscaler (HPA) and Vertical Pod Autoscaler (VPA). Because of this, the values may not match those from standard OS tools like 'top', as the metrics are designed to provide a stable signal for autoscalers rather than for pinpoint accuracy.\n\n FAQ for Metrics Server: https://github.com/kubernetes-sigs/metrics-server/blob/master/FAQ.md\n\n When to use this command:\n\n  *  For on-the-fly spot-checks of resource usage (e.g. identify which pods are consuming the most resources at a glance, or get a quick sense of the load on your nodes)\n  *  Understand current resource consumption patterns\n  *  Validate the behavior of your HPA or VPA configurations by seeing the metrics they use for scaling decisions.\n\n It is not intended to be a replacement for full-featured monitoring solutions. Its primary design goal is to provide a low-overhead signal for autoscalers, not to be a perfectly accurate monitoring tool. For high-accuracy reporting, historical analysis, dashboarding, or alerting, you should use a dedicated monitoring solution.",
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			node := topNodeCommand(client)
			cb.Subcommands.Command(node)
			pod := topPodCommand(client)
			cb.Subcommands.Command(pod)
		},
	}
	return cmd
}
