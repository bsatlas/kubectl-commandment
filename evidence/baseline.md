# Workstream 1 execution baseline

## Exact source identities

```text
workspace   efaab903d63568a5fcb212e7e549e8785fca29b9
Commandment a02adf48fc503a57a0898caf235f800b08205270
Kubernetes  3930cfbbb185e71ecb7b993a5582de6f775bc1f0
```

The reconstructed Git identities equal the preparation identities in `kubectl-commandment-large-application-frontloaded-plan-r001.md`; no source-sensitive frontload finding requires drift recomputation.

## Toolchain

```text
Go 1.27.1 linux/amd64
Kubernetes source Go floor 1.27.0
Commandment source Go floor 1.27.1
```

Verification uses the supplied offline Go 1.27.1 runtime plus a disposable exact-Kubernetes worktree and its committed vendor tree. Exact Commandment `cli` source is supplied as a local Go-workspace module at the authoritative commit. These environment adapters are verification infrastructure only and are not part of this repository.

## Sandbox restriction

The Commandment repository root external `cli_test` package is target-only and is not executed by this workstream.
