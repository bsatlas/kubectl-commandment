package kubectl

import (
	"fmt"
	"strconv"
	"time"

	"github.com/abstractionsinc/commandment/cli"
)

func int32Language() cli.ValueCodec[int32] {
	language := cli.ValueCodec[int32]{
		Name: "int32",
		Parse: func(value string) (int32, error) {
			parsed, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return 0, cli.Rejection.Message("int32", fmt.Sprintf("expected signed 32-bit integer, got %q", value), cli.Rejection.At(0, 0, len(value)))
			}
			return int32(parsed), nil
		},
		Format:   func(value int32) string { return strconv.FormatInt(int64(value), 10) },
		Snapshot: func(value int32) int32 { return value },
	}
	return language
}

func int64Language() cli.ValueCodec[int64] {
	language := cli.ValueCodec[int64]{
		Name: "int64",
		Parse: func(value string) (int64, error) {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return 0, cli.Rejection.Message("int64", fmt.Sprintf("expected signed integer, got %q", value), cli.Rejection.At(0, 0, len(value)))
			}
			return parsed, nil
		},
		Format:   func(value int64) string { return strconv.FormatInt(value, 10) },
		Snapshot: func(value int64) int64 { return value },
	}
	return language
}

func durationLanguage() cli.ValueCodec[time.Duration] {
	language := cli.ValueCodec[time.Duration]{
		Name: "duration",
		Parse: func(value string) (time.Duration, error) {
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return 0, cli.Rejection.Message("duration", fmt.Sprintf("expected duration, got %q", value), cli.Rejection.At(0, 0, len(value)))
			}
			return parsed, nil
		},
		Format:   func(value time.Duration) string { return value.String() },
		Snapshot: func(value time.Duration) time.Duration { return value },
	}
	return language
}

func flagProvided(state cli.OptionalValue[cli.ValueState]) bool {
	if !state.Present {
		return false
	}
	for _, candidate := range state.Value.Contributors() {
		if candidate.ProducerKind() == cli.ValueProducerKindFlag {
			return true
		}
	}
	return false
}
