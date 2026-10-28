package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCreateFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "create requires a source",
			args: []string{"create"},
		},
		{
			name: "create source is exclusive",
			args: []string{"create", "-f", "object.yaml", "-k", "overlay"},
		},
		{
			name: "kustomize rejects recursive",
			args: []string{"create", "-k", "overlay", "-R"},
		},
		{
			name: "raw rejects edit",
			args: []string{"create", "-f", "object.yaml", "--raw=/api/v1/namespaces", "--edit"},
		},
		{
			name: "raw requires one local file",
			args: []string{"create", "-f", "one.yaml", "-f", "two.yaml", "--raw=/api/v1/namespaces"},
		},
		{
			name: "raw rejects remote file",
			args: []string{"create", "--filename=https://example.invalid/object.yaml", "--raw=/api/v1/namespaces"},
		},
		{
			name: "raw rejects recursive",
			args: []string{"create", "-f", "object.yaml", "--raw=/api/v1/namespaces", "-R"},
		},
		{
			name: "raw rejects selector",
			args: []string{"create", "-f", "object.yaml", "--raw=/api/v1/namespaces", "-l", "app=web"},
		},
		{
			name: "raw rejects output",
			args: []string{"create", "-f", "object.yaml", "--raw=/api/v1/namespaces", "-o", "yaml"},
		},
		{
			name: "raw URI has kubectl syntax",
			args: []string{"create", "-f", "object.yaml", "--raw=%"},
		},
		{
			name: "clusterrole requires verbs",
			args: []string{"create", "clusterrole", "reader", "--resource=pods"},
		},
		{
			name: "clusterrole requires a target",
			args: []string{"create", "clusterrole", "reader", "--verb=get"},
		},
		{
			name: "clusterrole aggregation excludes rules",
			args: []string{"create", "clusterrole", "reader", "--aggregation-rule=app=reader", "--verb=get", "--resource=pods"},
		},
		{
			name: "clusterrole non resource verbs are bounded",
			args: []string{"create", "clusterrole", "reader", "--verb=watch", "--non-resource-url=/healthz"},
		},
		{
			name: "clusterrole non resource url has kubectl syntax",
			args: []string{"create", "clusterrole", "reader", "--verb=get", "--non-resource-url=healthz"},
		},
		{
			name: "clusterrole non resource wildcard is suffix only",
			args: []string{"create", "clusterrole", "reader", "--verb=get", "--non-resource-url=/health*/ready"},
		},
		{
			name: "clusterrole resource has kubectl syntax",
			args: []string{"create", "clusterrole", "reader", "--verb=get", "--resource=.apps"},
		},
		{
			name: "clusterrole validates nonempty name",
			args: []string{"create", "clusterrole", "", "--verb=get", "--resource=pods"},
		},
		{
			name: "clusterrolebinding requires clusterrole flag",
			args: []string{"create", "clusterrolebinding", "reader"},
		},
		{
			name: "clusterrolebinding serviceaccount has kubectl syntax",
			args: []string{"create", "clusterrolebinding", "reader", "--clusterrole=reader", "--serviceaccount=ns:sa:extra"},
		},
		{
			name: "configmap env file excludes file",
			args: []string{"create", "configmap", "settings", "--from-env-file=.env", "--from-file=settings.ini"},
		},
		{
			name: "configmap validates nonempty name",
			args: []string{"create", "configmap", "", "--from-literal=a=b"},
		},
		{
			name: "configmap file source has kubectl syntax",
			args: []string{"create", "configmap", "settings", "--from-file==path"},
		},
		{
			name: "configmap literal source has kubectl syntax",
			args: []string{"create", "configmap", "settings", "--from-literal==value"},
		},
		{
			name: "cronjob requires image",
			args: []string{"create", "cronjob", "backup", "--schedule=* * * * *"},
		},
		{
			name: "cronjob requires schedule",
			args: []string{"create", "cronjob", "backup", "--image=busybox"},
		},
		{
			name: "deployment requires image",
			args: []string{"create", "deployment", "web"},
		},
		{
			name: "deployment command rejects multiple images",
			args: []string{"create", "deployment", "web", "--image=one", "--image=two", "--", "date"},
		},
		{
			name: "ingress requires routing",
			args: []string{"create", "ingress", "web"},
		},
		{
			name: "ingress rejects malformed rule",
			args: []string{"create", "ingress", "web", "--rule=not-a-rule"},
		},
		{
			name: "ingress annotation has kubectl syntax",
			args: []string{"create", "ingress", "web", "--rule=example.com/=api:80", "--annotation=key"},
		},
		{
			name: "ingress backend has kubectl syntax",
			args: []string{"create", "ingress", "web", "--default-backend=api"},
		},
		{
			name: "job requires image or from",
			args: []string{"create", "job", "work"},
		},
		{
			name: "job source is exclusive",
			args: []string{"create", "job", "work", "--image=busybox", "--from=cronjob/nightly"},
		},
		{
			name: "job from rejects command",
			args: []string{"create", "job", "work", "--from=cronjob/nightly", "--", "date"},
		},
		{
			name: "pdb requires selector",
			args: []string{"create", "poddisruptionbudget", "web", "--min-available=1"},
		},
		{
			name: "pdb availability is exclusive",
			args: []string{"create", "poddisruptionbudget", "web", "--selector=app=web", "--min-available=1", "--max-unavailable=1"},
		},
		{
			name: "pdb availability has kubectl syntax",
			args: []string{"create", "poddisruptionbudget", "web", "--selector=app=web", "--min-available=1Gi"},
		},
		{
			name: "pdb requires availability",
			args: []string{"create", "poddisruptionbudget", "web", "--selector=app=web"},
		},
		{
			name: "pdb selector has kubectl syntax",
			args: []string{"create", "poddisruptionbudget", "web", "--selector=app in (", "--min-available=1"},
		},
		{
			name: "pdb validates nonempty name",
			args: []string{"create", "poddisruptionbudget", "", "--selector=app=web", "--min-available=1"},
		},
		{
			name: "quota hard has kubectl syntax",
			args: []string{"create", "quota", "team", "--hard=cpu"},
		},
		{
			name: "quota scopes reject empty element",
			args: []string{"create", "quota", "team", "--scopes=BestEffort,"},
		},
		{
			name: "quota validates nonempty name",
			args: []string{"create", "quota", "", "--hard=cpu=1"},
		},
		{
			name: "role requires verbs",
			args: []string{"create", "role", "reader", "--resource=pods"},
		},
		{
			name: "role requires resources",
			args: []string{"create", "role", "reader", "--verb=get"},
		},
		{
			name: "role resource has kubectl syntax",
			args: []string{"create", "role", "reader", "--verb=get", "--resource=.apps"},
		},
		{
			name: "role validates nonempty name",
			args: []string{"create", "role", "", "--verb=get", "--resource=pods"},
		},
		{
			name: "rolebinding role reference is exclusive",
			args: []string{"create", "rolebinding", "reader", "--role=reader", "--clusterrole=reader"},
		},
		{
			name: "rolebinding requires role reference",
			args: []string{"create", "rolebinding", "reader"},
		},
		{
			name: "rolebinding serviceaccount has kubectl syntax",
			args: []string{"create", "rolebinding", "reader", "--role=reader", "--serviceaccount=ns:sa:extra"},
		},
		{
			name: "rolebinding validates nonempty name",
			args: []string{"create", "rolebinding", "", "--role=reader"},
		},
		{
			name: "secret env file excludes literal",
			args: []string{"create", "secret", "generic", "settings", "--from-env-file=.env", "--from-literal=a=b"},
		},
		{
			name: "secret generic validates nonempty name",
			args: []string{"create", "secret", "generic", "", "--from-literal=a=b"},
		},
		{
			name: "secret generic file source has kubectl syntax",
			args: []string{"create", "secret", "generic", "settings", "--from-file==path"},
		},
		{
			name: "secret generic literal source has kubectl syntax",
			args: []string{"create", "secret", "generic", "settings", "--from-literal==value"},
		},
		{
			name: "docker secret requires credentials or file",
			args: []string{"create", "secret", "docker-registry", "registry"},
		},
		{
			name: "docker secret validates nonempty name",
			args: []string{"create", "secret", "docker-registry", "", "--docker-username=user", "--docker-password=pass", "--docker-server=registry.example"},
		},
		{
			name: "docker secret file source has kubectl syntax",
			args: []string{"create", "secret", "docker-registry", "registry", "--from-file==path"},
		},
		{
			name: "tls secret requires key and cert",
			args: []string{"create", "secret", "tls", "site", "--cert=site.crt"},
		},
		{
			name: "tls secret requires both sources",
			args: []string{"create", "secret", "tls", "site"},
		},
		{
			name: "clusterip requires tcp unless headless",
			args: []string{"create", "service", "clusterip", "api"},
		},
		{
			name: "clusterip port has kubectl syntax",
			args: []string{"create", "service", "clusterip", "api", "--tcp=not-a-port"},
		},
		{
			name: "externalname requires external name",
			args: []string{"create", "service", "externalname", "api"},
		},
		{
			name: "externalname validates DNS",
			args: []string{"create", "service", "externalname", "api", "--external-name=NOT_DNS"},
		},
		{
			name: "loadbalancer requires tcp",
			args: []string{"create", "service", "loadbalancer", "api"},
		},
		{
			name: "nodeport requires tcp",
			args: []string{"create", "service", "nodeport", "api"},
		},
		{
			name: "serviceaccount validates nonempty name",
			args: []string{"create", "serviceaccount", ""},
		},
		{
			name: "token validates nonempty name",
			args: []string{"create", "token", ""},
		},
		{
			name: "token duration is nonnegative",
			args: []string{"create", "token", "builder", "--duration=-1s"},
		},
		{
			name: "token duration is whole seconds",
			args: []string{"create", "token", "builder", "--duration=1500ms"},
		},
		{
			name: "token bound name requires kind",
			args: []string{"create", "token", "builder", "--bound-object-name=pod"},
		},
		{
			name: "token bound kind requires name",
			args: []string{"create", "token", "builder", "--bound-object-kind=Pod"},
		},
		{
			name: "token bound kind is constrained",
			args: []string{"create", "token", "builder", "--bound-object-kind=Deployment", "--bound-object-name=web"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := cli.InspectionRequest{
				Args: test.args,
			}
			clitest.Inspect(t.Context(), app, request).Require(
				t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}

func TestCreateFrontendAcceptsKubectlBoundaryCases(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	tests := []struct {
		name         string
		args         []string
		expectations []clitest.InspectionExpectation
	}{
		{
			name: "direct create accepts filename source",
			args: []string{"create", "-f", "object.yaml"},
		},
		{
			name: "direct create accepts assigned shorthand filename",
			args: []string{"create", "-f=object.yaml"},
		},
		{
			name: "subcommand accepts assigned shorthand output",
			args: []string{"create", "namespace", "work", "-o=name"},
		},
		{
			name: "subcommand does not inherit direct create source requirement",
			args: []string{"create", "namespace", "work"},
		},
		{
			name: "cronjob applies kubectl restart default",
			args: []string{"create", "cronjob", "backup", "--image=busybox", "--schedule=* * * * *"},
			expectations: []clitest.InspectionExpectation{
				clitest.Command("create", "cronjob").Flag("--restart").Value().Text("OnFailure"),
			},
		},
		{
			name: "headless clusterip does not require tcp",
			args: []string{"create", "service", "clusterip", "api", "--clusterip=None"},
		},
		{
			name: "priorityclass value remains optional",
			args: []string{"create", "priorityclass", "normal"},
		},
		{
			name: "clusterrole aggregation stands alone",
			args: []string{"create", "clusterrole", "aggregate", "--aggregation-rule=app=reader"},
		},
		{
			name: "token bound UID is optional",
			args: []string{"create", "token", "builder", "--bound-object-kind=Pod", "--bound-object-name=web"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			expectations := []clitest.InspectionExpectation{clitest.InspectionReady()}
			expectations = append(expectations, test.expectations...)
			request := cli.InspectionRequest{
				Args: test.args,
			}
			clitest.Inspect(t.Context(), app, request).Require(
				t,
				expectations...,
			)
		})
	}
}
