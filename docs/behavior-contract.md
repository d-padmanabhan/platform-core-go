# Behavior Contract (Python <-> Go)

This document defines shared behavior expectations between:

- `platform-core` (Python)
- `platform-core-go` (Go)

> [!IMPORTANT]
> When implementations differ, update this contract and call out intentional differences in release notes.

## Goals

- Keep user-facing behavior consistent across languages
- Reduce migration risk between Python and Go tools/services
- Make retry/timeout/error behavior explicit and testable

## Configuration and Environment Variables

### General

- Environment variables are strings
- Required values must fail fast with clear errors
- Boolean env values use `1` for true, anything else is false

### Shared env vars

- `LOG_LEVEL` (default: `INFO`)
- `CLOUDFLARE_HTTP_MAX_RETRIES` (default: `3`)
- `CLOUDFLARE_HTTP_RETRY_BASE_DELAY_SECONDS` (default: `1.0`)
- `CLOUDFLARE_HTTP_RETRY_MAX_DELAY_SECONDS` (default: `30.0`)
- `VAULT_HTTP_TIMEOUT_SECONDS` (default: `30`)
- `VAULT_ADDR` (required for Vault)
- `VAULT_TOKEN` (required for Vault)

## HTTP Timeout Defaults

- Cloudflare API requests: `30s`
- Vault API requests: `30s`
- Default policy: all network calls must use explicit timeouts

## Retry and Backoff Policy

### Cloudflare

- Retry on:
  - `408`
  - `429`
  - `5xx`
  - transport/network errors
- Respect `Retry-After` for `429` when present
- Exponential backoff with jitter
- Retries are bounded; never infinite

### AWS

- Use SDK standard retry mode with bounded attempts
- Avoid custom unbounded retry loops

### Vault

- Default: no hidden automatic retries for write/read operations unless explicitly documented
- Callers can wrap with shared retry helpers when needed

## Error Handling Conventions

- Return typed/sentinel errors where practical
- Include operation context in wrapped errors
- Do not include secrets in error messages
- Prefer stable, actionable error text for common failures

Examples of stable error categories:

- invalid configuration
- authentication/authorization failures
- not found
- timeout
- retry exhausted

## Pagination Behavior

### Cloudflare GET endpoints

- If endpoint returns `result_info.total_pages`, fetch all pages
- Aggregate ordered results by page
- For non-paginated responses, return `result` directly

## Logging and Redaction

- Default production log level is `INFO`
- `DEBUG` only via config
- Never log secret values (tokens, passwords, credentials, API keys)
- Mask sensitive headers and payload fields by key name

## Testing Expectations

- Unit tests for retry/backoff boundaries and error mapping
- Unit tests for pagination behavior and malformed response handling
- Contract tests for parity-critical behavior where feasible

## Change Management

When changing any behavior listed here:

1. Update this document
2. Add/adjust tests
3. Mention the contract change in PR description
4. Call out cross-language impact
