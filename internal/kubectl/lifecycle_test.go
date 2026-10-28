package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestPreexistingDeprecatedFlagsPreserveLifecycleAndVisibility(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}

	schema := app.Schema()
	var command cli.CommandSchema
	for _, id := range schema.CommandOrder {
		candidate := schema.Commands[id]
		if candidate.Name == "kustomize" {
			command = candidate
			break
		}
	}
	if command.ID == 0 {
		t.Fatal("kustomize command is missing")
	}

	assertDeprecatedHiddenFlag(t, schema, command, "--enable-managedby-label")
	assertDeprecatedHiddenFlag(t, schema, command, "--reorder")
}

func assertDeprecatedHiddenFlag(t *testing.T, schema cli.Schema, command cli.CommandSchema, spelling string) {
	t.Helper()

	for _, flagID := range command.Flags {
		flag := schema.Flags[flagID]
		for _, spellingID := range flag.Spellings {
			candidate := schema.FlagSpellings[spellingID]
			if candidate.Spelling.Text != spelling {
				continue
			}
			if !flag.Hidden {
				t.Fatalf("flag %s hidden = false, want true", spelling)
			}
			if flag.LifecycleState != cli.LifecycleStateDeprecated {
				t.Fatalf("flag %s lifecycle = %v, want deprecated", spelling, flag.LifecycleState)
			}
			if flag.Release.Reason == "" {
				t.Fatalf("flag %s deprecation reason is empty", spelling)
			}
			return
		}
	}
	t.Fatalf("flag %s is missing", spelling)
}
