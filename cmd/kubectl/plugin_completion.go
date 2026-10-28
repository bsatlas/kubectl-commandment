package main

import (
	"context"

	"github.com/abstractionsinc/commandment/cli"
)

func externalPluginCompletion(lookupName string) cli.CompletionDelegator[externalPluginConfig] {
	return func(_ context.Context, _ *externalPluginConfig, request cli.CompletionDelegationRequest) (cli.CompletionDelegation, error) {
		args := make([]string, 1, len(request.Args)+1)
		args[0] = "kubectl_complete-" + lookupName
		args = append(args, request.Args...)
		delegation := cli.CompletionDelegation{
			Args: args,
			Cursor: cli.CompletionCursor{
				Argument: request.Cursor.Argument + 1,
				Byte:     request.Cursor.Byte,
			},
		}
		return delegation, nil
	}
}
