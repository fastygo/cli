# fastygo

`fastygo` installs a site once. After that, versions live in the project's `go.mod`. There is no project YAML and no `packageManager` field.

This release installs:

- `github.com/fastygo/framework` v0.4.0
- `github.com/fastygo/modules/render` v0.1.0 and templ v0.3.1001 as `go tool templ`
- optional markdown, view, and a local Codex bbolt binary from `github.com/fastygo/backend/cmd/server` v0.1.0

Targets are `local` and `vps`. The process listens on `127.0.0.1`.

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
go install github.com/fastygo/cli/cmd/fastygo@v0.1.1
```

Linux amd64 and macOS arm64 can also use `install.sh`. It checks `SHA256SUMS` and prints the binary path.

If Go 1.25 is missing, `init` offers to download it into the user cache before it writes the project. `--yes` accepts that download when stdin is not a terminal.

## First site

```text
fastygo init . --module github.com/you/shop --target vps --with codex
go run ./cmd/site
```

`fastygo init --help` lists the flags. A second `init` stops when `go.mod` is already there. `fastygo add markdown` updates that module.
