# Repository Guidelines

## Project Structure & Module Organization

```text
├─ cmd/alexandria       # Main binary source (main.go, server.go)
├─ alexandria           # Core library code and unit tests
├─ web                  # Templating and static assets (css, fonts)
├─ Dockerfile           # Container image definition
├─ manifest/            # Kustomize manifests for prod/staging
├─ package.json         # Node tooling for asset pipeline
└─ go.mod / go.sum      # Go module definition
```

- Go source lives under `cmd/` for executables and top‑level packages for
  reusable code.
- Tests are co‑located with the package they cover (`*_test.go`).
- Static web assets are under `web/xess/static` and compiled templates under
  `web/`.

## Production Environment

The service is deployed to a Kubernetes cluster. Production manifests are stored
under the `manifest/` directory and are applied with Kustomize (e.g.,
`kustomize build manifest/prod | kubectl apply -f -`). The Docker image built
via `docker buildx bake --load` is pushed to the container registry and
referenced in the Kubernetes Deployment spec.

## Build, Test, and Development Commands

| Command                        | Description                                       |
| ------------------------------ | ------------------------------------------------- |
| `go build ./cmd/alexandria`    | Compiles the server binary.                       |
| `go run ./cmd/alexandria`      | Runs the server locally (reads env vars).         |
| `npm run test`                 | Executes all repository tests (via npm script).   |
| `npm install && npm run build` | Installs Node dependencies and builds web assets. |
| `docker buildx bake --load`    | Builds the Docker image using the bake file.      |

## Coding Style & Naming Conventions

- Use **gofmt** (or `go fmt ./...`) for formatting.
- Follow **golint/go vet** recommendations – avoid exported names with
  underscores.
- Indentation: tabs (default Go style).
- Naming: `camelCase` for variables, `PascalCase` for exported types and
  functions.
- Linting: `go vet ./...` and `staticcheck ./...` are run in CI.

## Testing Guidelines

- Test files end with `_test.go` and use the standard `testing` package.
- Name test functions `Test<Thing>` and keep them focused.
- Run coverage locally with `go test -cover ./...`.
- Aim for ≥ 80 % coverage on new code.

## Commit & Pull Request Guidelines

- **Commit messages** follow Conventional Commits:
  `<type>(<scope>): <description>`. _Examples:_
  `feat(server): add health endpoint`, `chore: update license`.
- PR description must include:
  1. A brief summary of the change.
  2. Linked issue number (`Fixes #123`).
  3. Any required migration steps.
  4. Screenshots for UI or asset changes.
- Keep PRs small and focused; request review after CI passes.
- **Assisted‑by footer** – when a commit is generated with AI assistance, append
  a footer line to the commit message: `Assisted-by: GPT-OSS 120b`

## Security & Configuration Tips

- Secrets are loaded via environment variables (see `README.md`).
- Do not commit `.env` files; add them to `.gitignore`.
- Run `go vet` and `staticcheck` before pushing to catch insecure patterns.

---

These guidelines aim to keep the codebase consistent and easy to contribute to.
If you notice a gap, feel free to open an issue or submit a PR.
