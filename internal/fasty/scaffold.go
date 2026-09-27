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
		filepath.Join("cmd", "site", "main.go"): processMain(plan),
		filepath.Join("internal", "site", "app.go"):     siteApp(plan),
		filepath.Join("internal", "site", "page.go"):    pageGo,
		filepath.Join("internal", "site", "home.templ"): homeTempl,
	}
	if plan.Content == "fixtures" {
		files[filepath.Join("content", "manifest.json")] = fixtureManifest
		files[filepath.Join("content", "entries", "home.json")] = fixtureHome
		files[filepath.Join("content", "load.go")] = fixtureLoad
		files[filepath.Join("internal", "site", "fixtures.go")] = fixturesGo(plan)
	} else {
		files[filepath.Join("internal", "site", "remote.go")] = remoteGoSource()
	}
	if plan.Target == "vercel" {
		files["vercel.json"] = vercelJSON
		files[filepath.Join("api", "index.go")] = vercelHandler(plan)
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
- templ CLI: v0.3.1001, installed with the tool directive. Generate with `+"`go tool templ generate ./internal/site`"+`.
- Target recorded at install: %s.
`, Version, plan.Target)
	switch plan.Target {
	case "vercel":
		b.WriteString("- Vercel runs api/index.go. Handler delegates to the shared app and does not listen or start workers.\n")
		b.WriteString("- Local development still uses cmd/site and Run(ctx) on 127.0.0.1.\n")
	default:
		b.WriteString("- The process listens on 127.0.0.1. A public TLS proxy forwards to that address.\n")
	}
	switch plan.Content {
	case "fixtures":
		b.WriteString("- Content: embedded Codex fixtures in content/, loaded by github.com/fastygo/modules/content-json v0.1.0.\n")
	case "codex":
		b.WriteString("- Content: an external GoBackend. Set CODEX_ORIGIN. Do not commit CODEX_TOKEN.\n")
		b.WriteString("- Do not add github.com/fastygo/backend to go.mod.\n")
	}
	if plan.With["markdown"] {
		b.WriteString("- Markdown: github.com/fastygo/modules/markdown v0.1.0. The package name is contentmarkdown.\n")
	}
	if plan.With["view"] {
		b.WriteString("- Theme and language data: github.com/fastygo/modules/view v0.1.0. Templ files for those controls stay in this application.\n")
	}
	if plan.With["codex"] {
		b.WriteString(`- Local Codex server: the binary in .bin, from github.com/fastygo/backend/cmd/server v0.1.0, storage bbolt.
  The site does not import that module. Variable names are in codex.env.example.
  Secrets and the product manifest stay out of the repository.
`)
	}
	b.WriteString(`
## Commands

` + "```text" + `
go test ./...
go vet ./...
go run ./cmd/site
go tool templ generate ./internal/site
` + "```" + `
`)
	if plan.Target == "vercel" {
		b.WriteString("\nPublish the repository with the Vercel Go runtime. vercel.json routes public paths to api/index.go.\n")
	}
	b.WriteString("\nThe package.json dev script runs the same go run command. npm, pnpm, yarn, and bun can all start it. This project does not set packageManager.\n")
	return b.String()
}

func siteEnv(plan Plan) string {
	var b strings.Builder
	b.WriteString("APP_BIND=127.0.0.1:8080\n")
	if plan.Content == "codex" {
		b.WriteString(`CODEX_ORIGIN=
CODEX_COLLECTION=pages
CODEX_ENTRY_SLUG=home
CODEX_LOCALE=en
# CODEX_TOKEN=
`)
	}
	return b.String()
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

const vercelJSON = `{
  "$schema": "https://openapi.vercel.sh/vercel.json",
  "routes": [
    {
      "src": "/(.*)",
      "dest": "/api/index",
      "transforms": [
        {
          "type": "request.path",
          "op": "set",
          "args": "/$1"
        }
      ]
    }
  ]
}
`

const fixtureManifest = `{
  "name": "site",
  "version": "1",
  "resources": [
    {
      "record": {
        "id": "page",
        "label": "Pages",
        "scope": "tenant",
        "fields": [
          {
            "id": "content",
            "label": "Content",
            "type": "markdown",
            "localized": true
          }
        ]
      }
    }
  ]
}
`

const fixtureHome = `{
  "id": "home",
  "kind": "page",
  "status": "published",
  "visibility": "public",
  "slug": {
    "en": "home"
  },
  "title": {
    "en": "Home"
  },
  "content": {
    "en": "Hello from Codex"
  },
  "locales": {
    "en": {
      "data": {
        "content": "Hello from Codex"
      },
      "status": "published",
      "updated_at": "2026-09-28T00:00:00Z"
    }
  },
  "version": 1,
  "created_at": "2026-09-28T00:00:00Z",
  "updated_at": "2026-09-28T00:00:00Z",
  "published_at": "2026-09-28T00:00:00Z"
}
`

const fixtureLoad = `package sitecontent

import "embed"

// Files is the closed Codex snapshot embedded in this application.
//
//go:embed manifest.json
//go:embed entries/home.json
var Files embed.FS
`

const pageGo = `package site

import (
	"context"
	"errors"
)

// Page is the view model rendered by templ. It does not reveal whether the
// content came from fixtures or HTTP.
type Page struct {
	Title string
	Body  string
}

// ErrNotFound is returned when the requested slug is absent.
var ErrNotFound = errors.New("page not found")

type source interface {
	Page(ctx context.Context, slug string) (Page, error)
}
`

const homeTempl = `package site

templ Home(page Page) {
	<!DOCTYPE html>
	<html lang="en">
		<head><title>{ page.Title }</title></head>
		<body>
			<h1>{ page.Title }</h1>
			<p>{ page.Body }</p>
		</body>
	</html>
}
`

func processMain(plan Plan) string {
	return `package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"` + plan.Module + `/internal/site"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	application, err := site.New()
	if err != nil {
		log.Fatal(err)
	}
	if err := application.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
`
}

func vercelHandler(plan Plan) string {
	return `package handler

import (
	"net/http"
	"sync"

	"` + plan.Module + `/internal/site"
)

var (
	loadOnce    sync.Once
	application http.Handler
	loadErr     error
)

// Handler is the Vercel Go Function entrypoint. It does not listen and it
// does not start framework workers.
func Handler(w http.ResponseWriter, r *http.Request) {
	loadOnce.Do(func() {
		built, err := site.New()
		if err != nil {
			loadErr = err
			return
		}
		application = built.Handler()
	})
	if loadErr != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if r.URL.Path == "/api" || r.URL.Path == "/api/index" {
		clone := r.Clone(r.Context())
		clone.URL.Path = "/"
		r = clone
	}
	application.ServeHTTP(w, r)
}
`
}

func siteApp(plan Plan) string {
	return `package site

import (
	"net/http"

	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/modules/render"
)

// New builds the shared application. Run serves it as a process. The Vercel
// function serves Handler without starting that process.
func New() (*app.App, error) {
	pages, err := newSource()
	if err != nil {
		return nil, err
	}
	cfg, err := app.LoadConfig()
	if err != nil {
		return nil, err
	}
	builder := app.New(cfg)
	// This site has no static files. Disable the default /static/ route so
	// it cannot collide with the home route.
	builder.DisableStatic()
	builder.Mux().HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	builder.Mux().HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, r, pages, "home")
	})
	builder.Mux().HandleFunc("GET /{slug}", func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, r, pages, r.PathValue("slug"))
	})
	return builder.Build(), nil
}

func renderPage(w http.ResponseWriter, r *http.Request, pages source, slug string) {
	page, err := pages.Page(r.Context(), slug)
	if err == ErrNotFound {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := render.Render(r.Context(), w, Home(page)); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
`
}

func fixturesGo(plan Plan) string {
	return `package site

import (
	"context"

	"github.com/fastygo/codex/content"
	contentjson "github.com/fastygo/modules/content-json"

	"` + plan.Module + `/content"
)

type fixtureSource struct {
	library *contentjson.Library
}

func newSource() (source, error) {
	library, err := contentjson.Load(contentjson.Options{FS: sitecontent.Files})
	if err != nil {
		return nil, err
	}
	if _, ok := library.Get("home"); !ok {
		return nil, ErrNotFound
	}
	return fixtureSource{library: library}, nil
}

func (source fixtureSource) Page(_ context.Context, slug string) (Page, error) {
	for _, entry := range source.library.Entries() {
		if entry.Slug.Value("en", "en") == slug {
			return pageFromEntry(entry), nil
		}
	}
	return Page{}, ErrNotFound
}

func pageFromEntry(entry content.Entry) Page {
	return Page{
		Title: entry.Title.Value("en", "en"),
		Body:  entry.Content.Value("en", "en"),
	}
}
`
}

func remoteGoSource() string {
	return `package site

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type remoteSource struct {
	origin     string
	collection string
	locale     string
	client     *http.Client
	token      string
}

func newSource() (source, error) {
	origin := strings.TrimRight(strings.TrimSpace(os.Getenv("CODEX_ORIGIN")), "/")
	if origin == "" {
		return nil, fmt.Errorf("CODEX_ORIGIN is required")
	}
	collection := strings.Trim(strings.TrimSpace(os.Getenv("CODEX_COLLECTION")), "/")
	if collection == "" {
		collection = "pages"
	}
	locale := strings.TrimSpace(os.Getenv("CODEX_LOCALE"))
	if locale == "" {
		locale = "en"
	}
	return remoteSource{
		origin:     origin,
		collection: collection,
		locale:     locale,
		token:      strings.TrimSpace(os.Getenv("CODEX_TOKEN")),
		client:     &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (source remoteSource) Page(ctx context.Context, slug string) (Page, error) {
	if strings.TrimSpace(slug) == "" {
		slug = strings.TrimSpace(os.Getenv("CODEX_ENTRY_SLUG"))
	}
	if slug == "" {
		slug = "home"
	}
	endpoint := source.origin + "/go-json/go/v2/" + url.PathEscape(source.collection) + "/by-slug/" + url.PathEscape(slug)
	query := url.Values{}
	query.Set("locale", source.locale)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return Page{}, err
	}
	if source.token != "" {
		request.Header.Set("Authorization", "Bearer "+source.token)
	}
	response, err := source.client.Do(request)
	if err != nil {
		return Page{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return Page{}, err
	}
	if response.StatusCode == http.StatusNotFound {
		return Page{}, ErrNotFound
	}
	if response.StatusCode != http.StatusOK {
		return Page{}, fmt.Errorf("codex returned %s", response.Status)
	}
	var envelope struct {
		Data struct {
			Title   map[string]string ` + "`json:\"title\"`" + `
			Content map[string]string ` + "`json:\"content\"`" + `
		} ` + "`json:\"data\"`" + `
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return Page{}, err
	}
	return Page{
		Title: localized(envelope.Data.Title, source.locale),
		Body:  localized(envelope.Data.Content, source.locale),
	}, nil
}

func localized(values map[string]string, locale string) string {
	if value := strings.TrimSpace(values[locale]); value != "" {
		return value
	}
	return strings.TrimSpace(values["en"])
}
`
}

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
