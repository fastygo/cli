package fasty

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writeSite(dir string, plan Plan) error {
	name := moduleName(plan.Module)
	files := map[string]string{
		"AGENTS.md":                             agents(plan),
		".gitignore":                            gitignore,
		".env.example":                          siteEnv(plan),
		"package.json":                          packageJSON(name),
		filepath.Join("cmd", "site", "main.go"): mainGo,
		filepath.Join("cmd", "site", "home.templ"): homeTempl,
	}
	if plan.With["markdown"] || plan.With["view"] {
		files[filepath.Join("cmd", "site", "features.go")] = featuresGo(plan)
	}
	if plan.With["codex"] {
		files["codex.env.example"] = codexEnv
	}
	for rel, body := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func moduleName(module string) string {
	name := module
	if i := strings.LastIndex(module, "/"); i >= 0 {
		name = module[i+1:]
	}
	if name == "" {
		return "site"
	}
	return name
}

func agents(plan Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, `# Agent notes

This site was installed by fastygo %s. fastygo does not manage it after that.
Versions live in go.mod. Change them there.

## Stack

- Process: github.com/fastygo/framework v0.4.0. Call Handler() for a host that owns the server socket, and Run(ctx) for the long-running process. Do not rename either method.
- HTML: github.com/fastygo/modules/render v0.1.0
- templ CLI: v0.3.1001, installed with the tool directive. Generate with `+"`go tool templ generate ./cmd/site`"+`.
- Target recorded at install: %s. The process listens on 127.0.0.1. A public TLS proxy forwards to that address.

`, Version, plan.Target)
	if plan.With["markdown"] {
		b.WriteString("- Markdown: github.com/fastygo/modules/markdown v0.1.0. The package name is contentmarkdown.\n")
	}
	if plan.With["view"] {
		b.WriteString("- Theme and language data: github.com/fastygo/modules/view v0.1.0. Templ files for those controls stay in this application.\n")
	}
	if plan.With["codex"] {
		b.WriteString(`- Codex: the server binary in .bin, from github.com/fastygo/backend/cmd/server v0.1.0, storage bbolt.
  Call it over HTTP. Do not add github.com/fastygo/backend to go.mod.
  Variable names are in codex.env.example. Secrets stay out of the repository.
  The product manifest is supplied by the operator. This install does not write one.
`)
	}
	b.WriteString(`
## Commands

` + "```text" + `
go test ./...
go vet ./...
go run ./cmd/site
go tool templ generate ./cmd/site
` + "```" + `

The package.json dev script runs the same go run command. npm, pnpm, yarn, and bun can all start it. This project does not set packageManager.
`)
	return b.String()
}

func siteEnv(plan Plan) string {
	origin := ""
	if plan.With["codex"] {
		origin = "CODEX_ORIGIN=http://127.0.0.1:8081\n"
	}
	return "APP_BIND=127.0.0.1:8080\n" + origin
}

func packageJSON(name string) string {
	return fmt.Sprintf(`{
  "name": %q,
  "private": true,
  "scripts": {
    "dev": "go run ./cmd/site"
  }
}
`, name)
}

const gitignore = `.bin/
data/
.env
`

const codexEnv = `APP_BIND=127.0.0.1:8081
HEADLESS_STORAGE=bbolt
HEADLESS_BBOLT_PATH=./data/codex.db
HEADLESS_MEDIA_ROOT=./data/media
# HEADLESS_MANIFEST_PATH=
# HEADLESS_TOKEN_SECRET=
`

const mainGo = `package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/modules/render"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	builder := app.New(cfg)
	// LoadConfig sets StaticDir, so Build would register /static/.
	// Go rejects that pattern next to a GET / subtree, and this site
	// has no static files. The home route matches only /.
	builder.DisableStatic()
	builder.Mux().HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if err := render.Render(r.Context(), w, home()); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	})
	if err := builder.Build().Run(ctx); err != nil {
		log.Fatal(err)
	}
}
`

const homeTempl = `package main

templ home() {
	<!DOCTYPE html>
	<html lang="en">
		<head><title>Home</title></head>
		<body><h1>Home</h1></body>
	</html>
}
`

func featuresGo(plan Plan) string {
	var b strings.Builder
	b.WriteString("// Installed by fastygo. fastygo add rewrites this file.\n\npackage main\n\nimport (\n")
	if plan.With["markdown"] {
		b.WriteString("\t\"github.com/fastygo/modules/markdown\"\n")
	}
	if plan.With["view"] {
		b.WriteString("\t\"github.com/fastygo/modules/view\"\n")
	}
	b.WriteString(")\n")
	if plan.With["markdown"] {
		b.WriteString(`
func markdownRenderer() contentmarkdown.Renderer {
	return contentmarkdown.DefaultRenderer()
}
`)
	}
	if plan.With["view"] {
		b.WriteString(`
func languageToggle() view.LanguageToggleData {
	return view.LanguageToggleData{Label: "Language"}
}
`)
	}
	return b.String()
}
