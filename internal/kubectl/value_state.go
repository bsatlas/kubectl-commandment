package kubectl

import "github.com/abstractionsinc/commandment/cli"

func explicitFlagValue(state cli.OptionalValue[cli.ValueState]) bool {
	return state.Present && explicitFlagState(state.Value)
}

func explicitFlagState(state cli.ValueState) bool {
	for _, producer := range state.Producers() {
		if producer.Kind == cli.ValueProducerKindFlag && producer.Tier == cli.ResolutionTierExplicit && producer.Outcome == cli.ValueProducerOutcomeContributed {
			return true
		}
	}
	return false
}
