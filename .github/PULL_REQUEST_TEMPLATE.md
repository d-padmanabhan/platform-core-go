### Motivation/context of the change(s)

- What problem does this PR solve?
- Why is this change needed now?

### What changed

- Summarize key code/documentation updates

### Please check if the PR fulfills these requirements

- [ ] PR title is meaningful (relevant, clear, concise)
- [ ] No sensitive information is part of this PR (passwords, keys, etc.)
- [ ] Changes adhere to repo conventions
- [ ] Documentation has been added/updated (if needed)
- [ ] This PR does not combine refactoring with feature-related changes
- [ ] Tests were added/updated or rationale provided
- [ ] Behavior contract impact reviewed (`docs/behavior-contract.md`)

### Validation

- Commands run locally:
  - `go test ./...`
  - `go test -race ./...`
  - `gofmt -s -w .`
  - `go vet ./...`
  - `staticcheck ./...`
  - `govulncheck ./...`
  - `golangci-lint run` (if configured)
