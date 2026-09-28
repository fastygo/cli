# fastygo

`fastygo` installs a site once. After that, versions live in the project's `go.mod`. There is no project YAML and no `packageManager` field.

This release installs:

- `github.com/fastygo/framework` v0.4.0
- `github.com/fastygo/modules/render` v0.1.0 and templ v0.3.1001 as `go tool templ`
- optional markdown, view, and a local Codex bbolt binary from `github.com/fastygo/backend/cmd/server` v0.1.0
- embedded Codex fixtures through `github.com/fastygo/modules/content-json` v0.1.0, or an external Codex origin

Targets are `local`, `vps`, and `vercel`. Local and VPS processes listen on `127.0.0.1`. Vercel gets a Go Function at `api/index.go`.

## Install the command

Any of these runs the binary packed inside the npm package. The npm version can be newer than the Go tag because npm versions cannot be replaced after publication. The git repository is private, so the package carries the binary instead of downloading the GitHub release.

```text
npx fastygo init . --module github.com/you/shop
pnpm dlx fastygo init . --module github.com/you/shop
yarn dlx fastygo init . --module github.com/you/shop
bunx fastygo init . --module github.com/you/shop
```

From the Go module:

```text
go install github.com/fastygo/cli/cmd/fastygo@v0.2.0
```

Linux amd64 and macOS arm64 can also use `install.sh`. It checks `SHA256SUMS` and prints the binary path.

If Go 1.25 is missing, `init` offers to download it into the user cache before it writes the project. `--yes` accepts that download when stdin is not a terminal.

## First site

```text
fastygo init . --module github.com/you/shop --target vps --with codex
fastygo init . --module github.com/you/site --target vercel --content fixtures
fastygo init . --module github.com/you/site --target vercel --content codex
go run ./cmd/site
```

`--content fixtures` is the default. It embeds `content/manifest.json` and `content/entries/home.json` and loads them with `content-json`. `--content codex` reads an external GoBackend through `CODEX_ORIGIN`; the token stays out of the repository. `--target vercel` writes `api/index.go` and `vercel.json` and does not install a local Codex server. Use `--with codex` only for `local` or `vps`.

The page template is `internal/site/home.templ`. Routes, including `DisableStatic`, `GET /healthz`, and `GET /{$}`, live in `internal/site/app.go`. `cmd/site/main.go` only calls `Run(ctx)`. After a template edit, run `go tool templ generate ./internal/site`.

`fastygo init --help` lists the flags. A second `init` stops when `go.mod` is already there. `fastygo add markdown` updates that module.
