package fasty

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedSites(t *testing.T) {
	if testing.Short() {
		t.Skip("generated site builds download published modules")
	}
	fixture := initGenerated(t, "fixtures")
	assertGeneratedModule(t, fixture)
	writeGenerated(t, filepath.Join(fixture, "internal", "site", "render_test.go"), fixtureRenderTest)
	goCommand(t, fixture, "test", "./internal/site")
	goCommand(t, fixture, "build", "./...")

	remote := initGenerated(t, "codex")
	assertGeneratedModule(t, remote)
	writeGenerated(t, filepath.Join(remote, "internal", "site", "remote_test.go"), remoteRenderTest)
	goCommand(t, remote, "test", "./internal/site")
	goCommand(t, remote, "build", "./...")
}

func initGenerated(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	err := Execute(context.Background(), []string{
		"init", dir,
		"--module", "github.com/acme/shop",
		"--target", "vercel",
		"--content", content,
		"--yes",
	}, IO{Out: os.Stdout, Err: os.Stderr})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func assertGeneratedModule(t *testing.T, dir string) {
	t.Helper()
	for _, rel := range []string{
		"vercel.json",
		"api/index.go",
		"cmd/site/main.go",
		"internal/site/app.go",
		"internal/site/home_templ.go",
	} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Fatal(rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".bin")); !os.IsNotExist(err) {
		t.Fatal(".bin exists")
	}
	mod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(mod)
	if strings.Contains(text, "github.com/fastygo/backend") || strings.Contains(text, "replace ") {
		t.Fatalf("go.mod:\n%s", text)
	}
	pkg, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(pkg), "packageManager") {
		t.Fatalf("package.json:\n%s", pkg)
	}
}

func goCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func writeGenerated(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const fixtureRenderTest = `package site

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFixturePages(t *testing.T) {
	application, err := New()
	if err != nil {
		t.Fatal(err)
	}
	home := httptest.NewRecorder()
	application.Handler().ServeHTTP(home, fixtureRequest("/"))
	if home.Code != http.StatusOK || !strings.Contains(home.Body.String(), "<h1>Home</h1>") || !strings.Contains(home.Body.String(), "Hello from Codex") {
		t.Fatalf("%d %s", home.Code, home.Body)
	}
	bySlug := httptest.NewRecorder()
	application.Handler().ServeHTTP(bySlug, fixtureRequest("/home"))
	if bySlug.Code != http.StatusOK {
		t.Fatalf("slug status %d", bySlug.Code)
	}
	health := httptest.NewRecorder()
	application.Handler().ServeHTTP(health, fixtureRequest("/healthz"))
	if health.Body.String() != "ok" {
		t.Fatalf("health %q", health.Body)
	}
	missing := httptest.NewRecorder()
	application.Handler().ServeHTTP(missing, fixtureRequest("/missing"))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status %d", missing.Code)
	}
}

func fixtureRequest(path string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("User-Agent", "fastygo-test")
	return request
}
`

const remoteRenderTest = `package site

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRemoteEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/go-json/go/v2/pages/by-slug/home" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(` + "`" + `{"data":{"title":{"en":"Home"},"content":{"en":"From Codex"}}}` + "`" + `))
	}))
	defer server.Close()
	t.Setenv("CODEX_ORIGIN", server.URL)
	t.Setenv("CODEX_COLLECTION", "pages")
	t.Setenv("CODEX_LOCALE", "en")
	t.Setenv("CODEX_TOKEN", "secret")
	application, err := New()
	if err != nil {
		t.Fatal(err)
	}
	home := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("User-Agent", "fastygo-test")
	application.Handler().ServeHTTP(home, request)
	if home.Code != http.StatusOK || !strings.Contains(home.Body.String(), "<h1>Home</h1>") || !strings.Contains(home.Body.String(), "From Codex") {
		t.Fatalf("%d %s", home.Code, home.Body)
	}
}
`
