package kubectl

import "context"

func runUncordon(_ context.Context, cfg *uncordonConfig) error {
	return cordonNodes(cfg.Client, cfg.Streams, cfg.Arguments, cfg.Selector, cfg.DryRun, nodeSchedulable)
}
