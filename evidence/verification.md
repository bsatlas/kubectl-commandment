# Workstream 1 verification

## Exact kubectl source candidate

```text
commit  aac6731c524e03f6039d79be1b52a2bb0d799700
tree    c9d2c1871faf87b318766648c83700b0541b375d
Go      1.27.1 linux/amd64
```

The exact source commit was checked in a clean detached worktree. The portable Kubernetes verification projection requires its staged module versions to be represented as `v0.0.0` while the public repository `go.mod` retains the exact Kubernetes pseudo-versions; verification temporarily rewrote only those module-version tokens and restored `go.mod` afterward. Commandment was supplied through the exact local composed correction donor below.

```text
Commandment parent  a02adf48fc503a57a0898caf235f800b08205270
Commandment donor   73621550f7ceba1b58c9c71d92a669580beec981
Commandment tree    595d71d81bdff0788613e21bff715b90372d5bf0
```

Returned kubectl-candidate gates:

```text
go test -count=1 ./...                         PASS
golangci-lint run --tests=false ./...          0 issues
go build ./cmd/kubectl                         PASS
20-case subprocess differential oracle          PASS
git diff HEAD^..HEAD --check                    PASS
direct Cobra/pflag imports in production Go     NONE
verification worktree after restore             CLEAN
```

Exact candidate binary:

```text
sha256 bf98b66f086313e584f42207773368e939ec8b57b6b3add59989b1647862ba1d
```

A Windows/amd64 cross-build of the exact source candidate was attempted once and did not return within the sandbox command window. It is **NOT RETURNED**, neither PASS nor FAIL, and was not retried. Platform closure therefore remains downstream.

## Commandment correction donor

The exact composed donor returned:

```text
go test -count=1 ./cli/test/conformance/completion/core       PASS
go test -count=1 ./cli/test/conformance/completion/dynamic    PASS
go test -count=1 ./cli/test/conformance/dynamic/providers     PASS
go test -count=1 ./cli/test/conformance/runtime/values        PASS
golangci-lint run --tests=false ./cli                         0 issues
go build ./cli                                                 PASS
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cli         PASS
git diff a02adf4..7362155 --check                              PASS
```

The Commandment root external `cli_test` package was **NOT RUN**. It is target-only under current sandbox law; maintainer target acceptance remains required after integration.

## Authority synchronization evidence

`evidence/DEC-B1-001-authoring-style.patch` was checked with `git apply --check` against exact workspace baseline `efaab903d63568a5fcb212e7e549e8785fca29b9`: **PASS**. The patch is evidence for the Director/Maintainer-owned authority correction; this standalone repository does not mutate workspace governance authority.
