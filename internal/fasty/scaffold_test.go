package fasty

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSiteOmitsPackageManager(t *testing.T) {
	dir := t.TempDir()
	plan := Plan{
		Module:  "github.com/acme/shop",
		Target:  "vps",
		Content: "fixtures",
		With:    map[string]bool{"render": true, "markdown": true, "codex": true},
	}
	if err := writeSite(dir, plan); err != nil {
		t.Fatal(err)
	}
	pkg, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(pkg), "packageManager") {
		t.Fatalf("package.json sets packageManager:\n%s", pkg)
	}
	if !strings.Contains(string(pkg), `"dev": "go run ./cmd/site"`) {
		t.Fatalf("dev script missing:\n%s", pkg)
	}
	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(agents)
	for _, phrase := range []string{"v0.4.0", "contentmarkdown", "127.0.0.1", "github.com/fastygo/backend", "vps", "content-json"} {
		if !strings.Contains(body, phrase) {
			t.Fatalf("AGENTS.md missing %q", phrase)
		}
	}
	env, err := os.ReadFile(filepath.Join(dir, "codex.env.example"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(env), "HEADLESS_TOKEN_SECRET=") && strings.Contains(string(env), "replace-with") {
		t.Fatal("example contains a secret value")
	}
	main, err := os.ReadFile(filepath.Join(dir, "cmd", "site", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(main), "github.com/fastygo/backend") {
		t.Fatal("site imports the backend module")
	}
	if _, err := os.Stat(filepath.Join(dir, "content", "manifest.json")); err != nil {
		t.Fatal(err)
	}
}

func TestWriteVercelCodexSite(t *testing.T) {
	dir := t.TempDir()
	plan := Plan{
		Module:  "github.com/acme/shop",
		Target:  "vercel",
		Content: "codex",
		With:    map[string]bool{"render": true},
	}
	if err := writeSite(dir, plan); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"vercel.json", "api/index.go", "internal/site/remote.go", "cmd/site/main.go"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Fatal(rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".bin")); !os.IsNotExist(err) {
		t.Fatal("vercel site created a local codex directory")
	}
	env, err := os.ReadFile(filepath.Join(dir, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(env)
	for _, name := range []string{"CODEX_ORIGIN=", "CODEX_COLLECTION=pages", "CODEX_ENTRY_SLUG=home", "CODEX_LOCALE=en", "# CODEX_TOKEN="} {
		if !strings.Contains(body, name) {
			t.Fatalf("env missing %q:\n%s", name, body)
		}
	}
	if strings.Contains(body, "CODEX_TOKEN=") && !strings.Contains(body, "# CODEX_TOKEN=") {
		t.Fatal("token value was written")
	}
	vercel, err := os.ReadFile(filepath.Join(dir, "vercel.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(vercel), "/api/index") {
		t.Fatalf("vercel.json:\n%s", vercel)
	}
	handler, err := os.ReadFile(filepath.Join(dir, "api", "index.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(handler), "func Handler(") || strings.Contains(string(handler), "github.com/fastygo/backend") {
		t.Fatalf("handler:\n%s", handler)
	}
}
