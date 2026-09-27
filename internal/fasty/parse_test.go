package fasty

import (
	"strings"
	"testing"
)

func TestParseInit(t *testing.T) {
	plan, err := parseArgs([]string{"init", "shop", "--module", "github.com/acme/shop", "--target", "vps", "--with", "markdown,view", "--with", "codex"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Dir != "shop" || plan.Module != "github.com/acme/shop" || plan.Target != "vps" {
		t.Fatalf("plan = %+v", plan)
	}
	for _, name := range []string{"render", "markdown", "view", "codex"} {
		if !plan.With[name] {
			t.Fatalf("missing %s in %+v", name, plan.With)
		}
	}
	if plan.Storage != "bbolt" {
		t.Fatalf("storage %s", plan.Storage)
	}
}

func TestParseRejectsUnknownTargetAndStorage(t *testing.T) {
	if _, err := parseArgs([]string{"init", "--module", "github.com/acme/shop", "--target", "serverless"}); err == nil {
		t.Fatal("expected unknown target")
	}
	if _, err := parseArgs([]string{"init", "--module", "github.com/acme/shop", "--with", "codex", "--storage", "postgres"}); err == nil {
		t.Fatal("expected unknown storage")
	}
	if _, err := parseArgs([]string{"init", "--module", "github.com/acme/shop", "--storage", "bbolt"}); err == nil {
		t.Fatal("expected storage without codex to fail")
	}
}

func TestParseAdd(t *testing.T) {
	plan, err := parseArgs([]string{"add", "markdown", "app"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Add != "markdown" || plan.Dir != "app" {
		t.Fatalf("plan = %+v", plan)
	}
	if _, err := parseArgs([]string{"add", "mail"}); err == nil {
		t.Fatal("expected unknown feature")
	}
}

func TestHelpMentionsLaunchersAndPins(t *testing.T) {
	for _, phrase := range []string{
		"npx fastygo",
		"pnpm dlx fastygo",
		"yarn dlx fastygo",
		"bunx fastygo",
		"v0.4.0",
		"bbolt",
		"no project YAML",
		"no packageManager",
	} {
		if !strings.Contains(helpText, phrase) {
			t.Fatalf("help missing %q", phrase)
		}
	}
}
