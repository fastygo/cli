package fasty

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	err := Execute(context.Background(), []string{
		"init", dir,
		"--module", "github.com/acme/shop",
		"--target", "vps",
		"--with", "codex",
		"--dry-run",
	}, IO{
		Out: &out,
		Err: &out,
		LookPath: func(name string) (string, error) {
			if name == goBinaryName() {
				return name, nil
			}
			return "", os.ErrNotExist
		},
		Run: func(ctx context.Context, name string, args []string, dir string, env []string) error {
			if len(args) > 0 && args[0] == "version" {
				return nil
			}
			t.Fatalf("dry-run executed %s %v", name, args)
			return nil
		},
		TTY:    false,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, phrase := range []string{"dry-run", "github.com/fastygo/framework@v0.4.0", "bbolt", "vps"} {
		if !strings.Contains(text, phrase) {
			t.Fatalf("output missing %q:\n%s", phrase, text)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		t.Fatal("dry-run wrote go.mod")
	}
}

func TestInitStopsWhenModuleExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/old\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Execute(context.Background(), []string{"init", dir, "--module", "github.com/acme/shop", "--dry-run"}, IO{
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	})
	if err == nil || !strings.Contains(err.Error(), "go.mod") {
		t.Fatalf("err = %v", err)
	}
}
