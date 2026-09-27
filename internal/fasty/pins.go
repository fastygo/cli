package fasty

// Version is the fastygo release. The same number is the git tag, the
// npm package, and the set of module versions this release installs.
const Version = "0.1.0"

const (
	goLine      = "1.25.0"
	framework   = "github.com/fastygo/framework@v0.4.0"
	renderMod   = "github.com/fastygo/modules/render@v0.1.0"
	markdownMod = "github.com/fastygo/modules/markdown@v0.1.0"
	viewMod     = "github.com/fastygo/modules/view@v0.1.0"
	templTool   = "github.com/a-h/templ/cmd/templ@v0.3.1001"
	codexPkg    = "github.com/fastygo/backend/cmd/server@v0.1.0"
)

const helpText = `fastygo ` + Version + ` installs a site once. The project keeps its own go.mod after that.

Usage:
  fastygo init [dir] --module <path> [--target local|vps] [--with markdown,view,codex] [--storage bbolt] [--dry-run] [--yes]
  fastygo add <markdown|view|codex> [dir] [--yes]
  fastygo help

init always adds Framework v0.4.0 and github.com/fastygo/modules/render v0.1.0.
templ v0.3.1001 is recorded with a tool directive. Run it as: go tool templ generate

--target    local or vps. Both listen on 127.0.0.1. vps stays behind TLS.
--with      markdown, view, and codex. Repeat the flag or separate values with commas.
--storage   bbolt, and only when codex is selected. fastygo ` + Version + ` has no other storage.
--dry-run   print the files and commands without writing or downloading.
--yes       download a missing Go 1.25 without a prompt. Required when stdin is not a terminal.

Codex is github.com/fastygo/backend/cmd/server v0.1.0 placed in .bin.
The site calls it over HTTP and does not require that module.

A second init stops when go.mod already exists. add updates that module.
There is no project YAML and no packageManager field.

Launchers for this installer: npx fastygo, pnpm dlx fastygo, yarn dlx fastygo, bunx fastygo.
From the module: go install github.com/fastygo/cli/cmd/fastygo@v` + Version + `
`
