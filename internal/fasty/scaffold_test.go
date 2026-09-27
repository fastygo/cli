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
		Module: "github.com/acme/shop",
		Target: "vps",
		With:   map[string]bool{"render": true, "markdown": true, "codex": true},
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
	for _, phrase := range []string{"v0.4.0", "contentmarkdown", "127.0.0.1", "github.com/fastygo/backend", "vps"} {
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
}
