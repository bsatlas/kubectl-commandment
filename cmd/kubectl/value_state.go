package main

import "github.com/abstractionsinc/commandment/cli"

func explicitFlagValue(state cli.OptionalValue[cli.ValueState]) bool {
	if !state.Present {
		return false
	}
	for _, producer := range state.Value.Producers() {
		if producer.Kind == cli.ValueProducerKindFlag && producer.Tier == cli.ResolutionTierExplicit && producer.Outcome == cli.ValueProducerOutcomeContributed {
			return true
		}
	}
	return false
}
