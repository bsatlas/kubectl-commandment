package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestCertificateConditionSourcePolicy(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	commands := []string{"approve", "deny"}
	tests := []struct {
		name     string
		args     []string
		rejected bool
	}{
		{
			name: "name",
			args: []string{"csr-a"},
		},
		{
			name: "filename",
			args: []string{"-f", "csr.yaml"},
		},
		{
			name: "joined short filename",
			args: []string{"-fcsr.yaml"},
		},
		{
			name: "assigned short filename",
			args: []string{"-f=csr.yaml"},
		},
		{
			name: "kustomize",
			args: []string{"-k", "overlay"},
		},
		{
			name: "name and filename",
			args: []string{"csr-a", "-f", "csr.yaml"},
		},
		{
			name: "name and kustomize",
			args: []string{"csr-a", "-k", "overlay"},
		},
		{
			name:     "missing source",
			rejected: true,
		},
		{
			name:     "filename and kustomize",
			args:     []string{"-f", "csr.yaml", "-k", "overlay"},
			rejected: true,
		},
		{
			name:     "recursive kustomize",
			args:     []string{"-k", "overlay", "-R"},
			rejected: true,
		},
	}

	for _, command := range commands {
		for _, test := range tests {
			t.Run(command+"/"+test.name, func(t *testing.T) {
				t.Parallel()

				args := []string{"certificate", command}
				args = append(args, test.args...)
				proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{Args: args})
				if !test.rejected {
					proof.Require(t, clitest.InspectionReady())
					return
				}
				proof.Require(t,
					clitest.InspectionRejected(),
					clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
				)
			})
		}
	}
}

func TestCertificateConditionUsesPflagLastOccurrenceSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	tests := []struct {
		name string
		args []string
		flag string
		want string
	}{
		{
			name: "force",
			args: []string{"certificate", "approve", "csr", "--force", "--force=false"},
			flag: "--force",
			want: "false",
		},
		{
			name: "kustomize",
			args: []string{"certificate", "approve", "-k", "first", "-k", "second"},
			flag: "--kustomize",
			want: "second",
		},
		{
			name: "recursive",
			args: []string{"certificate", "approve", "-f", "csr.yaml", "--recursive", "--recursive=false"},
			flag: "--recursive",
			want: "false",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			value := clitest.Command("certificate", "approve").Flag(test.flag).Value()
			clitest.Inspect(t.Context(), app, cli.InspectionRequest{Args: test.args}).Require(t,
				clitest.InspectionReady(),
				value.Resolved(),
				value.Text(test.want),
			)
		})
	}
}

func TestCertificateFilenameUsesStringSliceCSVLanguage(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"certificate", "approve", "--filename", `one.yaml,"two,three.yaml"`},
	}).Require(t, clitest.InspectionReady())

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"certificate", "approve", "--filename", `"unterminated`},
	}).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
	)
}

func TestCertificateReadsDashFilenameFromInvocationStdin(t *testing.T) {
	t.Parallel()

	server := certificateTestAPIServer(t, false)
	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	stdin := strings.NewReader(`apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: csr
`)
	request := cli.RunRequest{
		Args:   []string{"--server", server.URL, "certificate", "approve", "-f", "-"},
		Stdin:  stdin,
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestCertificatePreservesTypedNotFoundCause(t *testing.T) {
	t.Parallel()

	server := certificateTestAPIServer(t, true)
	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	request := cli.RunRequest{
		Args:   []string{"--server", server.URL, "certificate", "approve", "csr"},
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	err = app.Run(t.Context(), request)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("Run() error = %v, want wrapped NotFound cause", err)
	}
}

func TestCertificatePropagatesEmptyResultWriterError(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), "empty.yaml")
	if err := os.WriteFile(filename, nil, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	wantErr := errors.New("write failed")
	request := cli.RunRequest{
		Args:   []string{"--server", "http://127.0.0.1", "certificate", "approve", "-f", filename},
		Stdout: errorWriter{err: wantErr},
		Stderr: io.Discard,
	}

	err = app.Run(t.Context(), request)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func certificateTestAPIServer(t *testing.T, typedNotFound bool) *httptest.Server {
	t.Helper()

	var csrGets atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api":
			_, _ = writer.Write([]byte(`{"kind":"APIVersions","apiVersion":"v1","versions":["v1"],"serverAddressByClientCIDRs":[]}`))
		case "/apis":
			_, _ = writer.Write([]byte(`{"kind":"APIGroupList","apiVersion":"v1","groups":[{"name":"certificates.k8s.io","versions":[{"groupVersion":"certificates.k8s.io/v1","version":"v1"}],"preferredVersion":{"groupVersion":"certificates.k8s.io/v1","version":"v1"}}]}`))
		case "/api/v1":
			_, _ = writer.Write([]byte(`{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"v1","resources":[]}`))
		case "/apis/certificates.k8s.io/v1":
			_, _ = writer.Write([]byte(`{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"certificates.k8s.io/v1","resources":[{"name":"certificatesigningrequests","singularName":"certificatesigningrequest","namespaced":false,"kind":"CertificateSigningRequest","verbs":["get","update"]}]}`))
		case "/apis/certificates.k8s.io/v1/certificatesigningrequests/csr":
			if request.Method != http.MethodGet {
				http.Error(writer, "unexpected method", http.StatusMethodNotAllowed)
				return
			}
			if typedNotFound && csrGets.Add(1) > 1 {
				writer.WriteHeader(http.StatusNotFound)
				_, _ = writer.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"certificatesigningrequests.certificates.k8s.io \"csr\" not found","reason":"NotFound","details":{"name":"csr","group":"certificates.k8s.io","kind":"certificatesigningrequests"},"code":404}`))
				return
			}
			if !typedNotFound {
				csrGets.Add(1)
			}
			_, _ = writer.Write([]byte(`{"apiVersion":"certificates.k8s.io/v1","kind":"CertificateSigningRequest","metadata":{"name":"csr"}}`))
		case "/apis/certificates.k8s.io/v1/certificatesigningrequests/csr/approval":
			if request.Method != http.MethodPut {
				http.Error(writer, "unexpected method", http.StatusMethodNotAllowed)
				return
			}
			_, _ = writer.Write([]byte(`{"apiVersion":"certificates.k8s.io/v1","kind":"CertificateSigningRequest","metadata":{"name":"csr"},"status":{"conditions":[{"type":"Approved","status":"True"}]}}`))
		default:
			http.Error(writer, "unexpected path: "+request.URL.Path, http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}
