# Agent notes

`fastygo` is a one-shot installer. It does not keep owning the site it creates.

## Rules

1. Do not add a project YAML format or a `packageManager` field to generated apps.
2. Pins for this release live in `internal/fasty/pins.go`: Framework v0.4.0, modules v0.1.0, templ v0.3.1001, Codex server v0.1.0, Go 1.25.0.
3. The site must not require `github.com/fastygo/backend`. A local Codex server is a binary in `.bin`. Vercel content uses `CODEX_ORIGIN`.
4. `init` accepts targets `local`, `vps`, and `vercel`, content `fixtures` or `codex`, and storage `bbolt` only for a local Codex server.
5. Comments and help text are written in English.

Run before completion:

```text
go test ./...
go vet ./...
gofmt -w .
```
