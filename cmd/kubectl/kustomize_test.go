package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestKustomizeDeclaresArgumentAndTypedFlagDomains(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name     string
		args     []string
		rejected bool
	}{
		{
			name: "default directory",
			args: []string{"kustomize"},
		},
		{
			name: "one directory",
			args: []string{"kustomize", "overlay"},
		},
		{
			name:     "only one directory",
			args:     []string{"kustomize", "one", "two"},
			rejected: true,
		},
		{
			name: "root only load restrictor",
			args: []string{"kustomize", "--load-restrictor=LoadRestrictionsRootOnly"},
		},
		{
			name: "none load restrictor",
			args: []string{"kustomize", "--load-restrictor=LoadRestrictionsNone"},
		},
		{
			name:     "invalid load restrictor",
			args:     []string{"kustomize", "--load-restrictor=outside"},
			rejected: true,
		},
		{
			name: "legacy reorder",
			args: []string{"kustomize", "--reorder=legacy"},
		},
		{
			name: "none reorder",
			args: []string{"kustomize", "--reorder=none"},
		},
		{
			name:     "invalid reorder",
			args:     []string{"kustomize", "--reorder=random"},
			rejected: true,
		},
		{
			name: "repeated load restrictor keeps last",
			args: []string{"kustomize", "overlay", "--load-restrictor=LoadRestrictionsRootOnly", "--load-restrictor=LoadRestrictionsNone"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := cli.InspectionRequest{
				Args: test.args,
			}
			inspection := clitest.Inspect(t.Context(), app, request)
			if !test.rejected {
				inspection.Require(t, clitest.InspectionReady())
				return
			}
			inspection.Require(t, clitest.InspectionRejected(), clitest.FailsWith(clitest.DiagnosticAt(0).Present()))
		})
	}
}

func TestKustomizeDeclaresManagedByLabelEnvironment(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"kustomize"},
		Env: cli.Env{
			{
				Name:  "KUSTOMIZE_ENABLE_MANAGEDBY_LABEL",
				Value: "on",
			},
		},
	}
	managedByLabel := clitest.Command("kustomize").Environment("KUSTOMIZE_ENABLE_MANAGEDBY_LABEL").Value()
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		managedByLabel.Resolved(),
		managedByLabel.Text("true"),
	)
}

func TestKustomizeCompletesLoadRestrictorDomain(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.CompletionRequest{
		Args: []string{"kustomize", "--load-restrictor", "LoadRestrictions"},
		Cursor: cli.CompletionCursor{
			Argument: 2,
			Byte:     len("LoadRestrictions"),
		},
	}
	clitest.Complete(t.Context(), app, request).Require(t,
		clitest.CompletionItem("LoadRestrictionsNone").Present(),
		clitest.CompletionItem("LoadRestrictionsRootOnly").Present(),
	)
}

func TestPreexistingDeprecatedFlagsPreserveLifecycleAndVisibility(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
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
