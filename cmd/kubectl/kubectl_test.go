package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/clitest"
	"sort"
	"strings"
)

func TestStaticFrontendSemanticsComeFromCompiledApplication(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	schema := app.Schema()
	if got, want := len(schema.CommandOrder), 110; got != want {
		t.Fatalf("static command count = %d, want %d", got, want)
	}

	root := clitest.RootCommand()
	expectations := []clitest.SchemaExpectation{
		root.Name("kubectl"),
		root.Flag("--kubeconfig").Exists(),
		root.Flag("--context").Exists(),
		root.Flag("--namespace").Exists(),
		root.Flag("-n").Exists(),
		clitest.Command("config").NotRunnable(),
		clitest.Command("config", "use-context").Runnable(),
		clitest.Command("get").Runnable(),
		clitest.Command("create", "deployment").Runnable(),
		clitest.Command("apply").Runnable(),
		clitest.Command("apply", "edit-last-applied").Runnable(),
		clitest.Command("rollout", "status").Runnable(),
		clitest.Command("set", "image").Runnable(),
		clitest.Command("auth", "can-i").Runnable(),
		clitest.Command("certificate", "approve").Runnable(),
		clitest.Command("cluster-info", "dump").Runnable(),
		clitest.Command("top", "pod").Runnable(),
		clitest.Command("cp").Runnable(),
		clitest.Command("debug").Runnable(),
		clitest.Command("diff").Runnable(),
		clitest.Command("edit").Runnable(),
		clitest.Command("exec").Runnable(),
		clitest.Command("logs").Runnable(),
		clitest.Command("port-forward").Runnable(),
		clitest.Command("proxy").Runnable(),
		clitest.Command("run").Runnable(),
		clitest.Command("plugin").NotRunnable(),
		clitest.Command("plugin", "list").Runnable(),
		clitest.Command("options").NotRunnable(),
	}
	clitest.Schema(app).Require(t, expectations...)
}
func TestStaticFrontendTopologyMatchesBoundKubectl(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	schema := app.Schema()

	got := make([]string, 0, len(schema.CommandOrder)-1)
	runnable := make(map[string]bool, len(schema.CommandOrder)-1)
	for _, id := range schema.CommandOrder {
		command := schema.Commands[id]
		if command.ID == schema.Root {
			continue
		}
		path := strings.Join(command.Path, " ")
		got = append(got, path)
		runnable[path] = command.Runnable
	}
	sort.Strings(got)

	want := []string{
		"alpha",
		"annotate",
		"api-resources",
		"api-versions",
		"apply",
		"apply edit-last-applied",
		"apply set-last-applied",
		"apply view-last-applied",
		"attach",
		"auth",
		"auth can-i",
		"auth reconcile",
		"auth whoami",
		"autoscale",
		"certificate",
		"certificate approve",
		"certificate deny",
		"cluster-info",
		"cluster-info dump",
		"config",
		"config current-context",
		"config delete-cluster",
		"config delete-context",
		"config delete-user",
		"config get-clusters",
		"config get-contexts",
		"config get-users",
		"config rename-context",
		"config set",
		"config set-cluster",
		"config set-context",
		"config set-credentials",
		"config unset",
		"config use-context",
		"config view",
		"cordon",
		"cp",
		"create",
		"create clusterrole",
		"create clusterrolebinding",
		"create configmap",
		"create cronjob",
		"create deployment",
		"create ingress",
		"create job",
		"create namespace",
		"create poddisruptionbudget",
		"create priorityclass",
		"create quota",
		"create role",
		"create rolebinding",
		"create secret",
		"create secret docker-registry",
		"create secret generic",
		"create secret tls",
		"create service",
		"create service clusterip",
		"create service externalname",
		"create service loadbalancer",
		"create service nodeport",
		"create serviceaccount",
		"create token",
		"debug",
		"delete",
		"describe",
		"diff",
		"drain",
		"edit",
		"events",
		"exec",
		"explain",
		"expose",
		"get",
		"kuberc",
		"kuberc set",
		"kuberc view",
		"kustomize",
		"label",
		"logs",
		"options",
		"patch",
		"plugin",
		"plugin list",
		"port-forward",
		"proxy",
		"replace",
		"rollout",
		"rollout history",
		"rollout pause",
		"rollout restart",
		"rollout resume",
		"rollout status",
		"rollout undo",
		"run",
		"scale",
		"set",
		"set env",
		"set image",
		"set resources",
		"set selector",
		"set serviceaccount",
		"set subject",
		"taint",
		"top",
		"top node",
		"top pod",
		"uncordon",
		"version",
		"wait",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("static command paths differ\n got: %q\nwant: %q", got, want)
	}

	structural := map[string]struct{}{
		"alpha":          {},
		"auth":           {},
		"certificate":    {},
		"config":         {},
		"create secret":  {},
		"create service": {},
		"kuberc":         {},
		"options":        {},
		"plugin":         {},
		"rollout":        {},
		"set":            {},
		"top":            {},
	}
	for _, path := range want {
		_, helpOnly := structural[path]
		if gotRunnable := runnable[path]; gotRunnable == helpOnly {
			t.Fatalf("command %q runnable = %v, want %v", path, gotRunnable, !helpOnly)
		}
	}
}
