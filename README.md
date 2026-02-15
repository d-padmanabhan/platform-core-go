# platform-core-go

Shared platform Go library (AWS/Cloudflare/Kubernetes/Vault utilities + core helpers).

[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-fe5196.svg?logo=conventionalcommits&logoColor=white)](https://www.conventionalcommits.org/)
![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)
![AWS](https://img.shields.io/badge/AWS-SDK-232F3E?logo=amazonaws&logoColor=white)
![Cloudflare](https://img.shields.io/badge/Cloudflare-API-F38020?logo=cloudflare&logoColor=white)
![Vault](https://img.shields.io/badge/Vault-KV%20v2-000000?logo=vault&logoColor=white)

> [!NOTE]
> This README follows the same onboarding style as `platform-core` to keep cross-language adoption simple.

## Table of Contents

- [What this is](#what-this-is)
- [Scope and non-goals](#scope-and-non-goals)
- [Current status](#current-status)
- [Planned packages](#planned-packages)
- [Behavior contract](#behavior-contract)
- [Development](#development)
- [Contributing](#contributing)
- [Security](#security)
- [License](#license)

## What this is

`platform-core-go` is the Go counterpart to `platform-core` (Python), focused on reusable,
minimal, dependency-light platform helpers:

- AWS helpers (SDK client factory, common STS patterns)
- Cloudflare helpers (HTTP client, retries, pagination)
- Vault helpers (KV v2 read/write with timeouts)
- Optional Kubernetes helpers (client initialization wrappers)
- Core utilities (logging, retry primitives, HTTP helpers)

## Scope and non-goals

- **In scope**
  - Small, explicit package APIs
  - Safe defaults for timeouts, retries, and error handling
  - Reusable code for internal platform services and tooling
- **Non-goals**
  - Product/business logic
  - Infrastructure-as-code generation/deployment logic
  - Wrapping entire provider SDK surfaces

## Current status

This repository is bootstrapped and currently documentation-first.
Initial package scaffolding and implementation are planned next.

## Planned packages

- `internal/httpx`
  - Shared retry/backoff + HTTP transport helpers
- `cloudflare`
  - API client with bounded retries and pagination support
- `vault`
  - KV v2 helpers (`ReadKVv2`, `WriteKVv2`)
- `awsx`
  - AWS SDK client/session utilities (STS account ID, assume-role helpers)
- `kubernetes` (optional)
  - Thin wrapper for kubeconfig/in-cluster client setup

## Behavior contract

Cross-language behavior alignment (Python <-> Go) is tracked in:

- `docs/behavior-contract.md`

This document is the source of truth for:

- timeout defaults
- retry/backoff policy
- env var precedence
- error handling conventions
- pagination behavior

## Development

### Local setup

```bash
go version
go mod tidy
```

### Common commands

```bash
go test ./...
go test -race ./...
gofmt -w .
```

If `golangci-lint` is configured in this repo:

```bash
golangci-lint run
```

## Contributing

See `.github/CONTRIBUTING.md`.

## Security

- Do not commit secrets (tokens, keys, credentials)
- Prefer least privilege IAM and API token scopes
- Keep retries bounded and avoid logging sensitive payloads

## License

See `LICENSE.md`.
