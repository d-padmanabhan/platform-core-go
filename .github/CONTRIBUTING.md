## Contributing

### Local setup

- Verify Go toolchain:

```bash
go version
```

- Initialize dependencies:

```bash
go mod tidy
```

- Install and enable pre-commit hooks (if configured):

```bash
python3 -m pip install --upgrade pip
python3 -m pip install pre-commit
pre-commit install
```

### Run checks

```bash
go test ./...
go test -race ./...
gofmt -s -w .
go vet ./...
staticcheck ./...
govulncheck ./...
```

If `golangci-lint` is configured:

```bash
golangci-lint run
```

If pre-commit is configured:

```bash
pre-commit run --all-files
```

### Updating hook versions

```bash
pre-commit autoupdate
pre-commit run --all-files
```

> [!NOTE]
> If `ggshield` is configured in hooks, set `GITGUARDIAN_API_KEY` in your environment.

### Commit messages

This repository follows **Conventional Commits**.

Example:

```text
feat(cloudflare): add retry-aware api client
```

### Pull requests

- Use the PR template
- Keep changes focused (avoid bundling refactors with feature work)
- Do not commit secrets (keys, tokens, passwords, credentials)
