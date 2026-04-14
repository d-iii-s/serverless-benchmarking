# Contributing

Thanks for contributing to `slsbench`.

## Development Setup

```bash
git clone https://github.com/d-iii-s/serverless-benchmarking.git
cd serverless-benchmarking
go mod download
```

Optional local build:

```bash
go build -o slsbench ./cmd/slsbench
```

## Test Expectations

Before opening a PR, run:

```bash
go test ./internal/service/flowgen ./internal/service/datagen ./internal/service/dslvalidator ./internal/service/bodyprobe ./internal/service/harness
go build ./...
```

If your change touches release/docs paths, also verify:

- README command examples are valid and use current flags.
- `RELEASE_NOTES.md` and `CHANGELOG.md` are consistent.

## Pull Request Guidelines

- Keep PRs focused and atomic.
- Explain **why** the change is needed, not only what changed.
- Include test evidence in the PR description.
- If behavior changes, update docs in the same PR.

## Release Notes Guidance

For release-impacting changes, include an entry in:

- `CHANGELOG.md`
- `RELEASE_NOTES.md` (when preparing a release)

Use concise sections:

- Added
- Changed
- Removed
- Fixed (if applicable)
