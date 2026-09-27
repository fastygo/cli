package fasty

import (
	"fmt"
	"strings"
)

// Plan is one init or add invocation.
type Plan struct {
	Command string
	Dir     string
	Module  string
	Target  string
	Content string
	Storage string
	With    map[string]bool
	DryRun  bool
	Yes     bool
	Add     string
}

func parseArgs(args []string) (Plan, error) {
	plan := Plan{
		Dir:     ".",
		Target:  "local",
		Content: "fixtures",
		Storage: "bbolt",
		With:    map[string]bool{"render": true},
	}
	if len(args) == 0 {
		return Plan{}, fmt.Errorf("missing command\n\n%s", helpText)
	}
	switch args[0] {
	case "help", "--help", "-h":
		plan.Command = "help"
		return plan, nil
	case "version", "--version":
		plan.Command = "version"
		return plan, nil
	case "init", "add":
		plan.Command = args[0]
	default:
		return Plan{}, fmt.Errorf("unknown command %q\n\n%s", args[0], helpText)
	}

	positionals := make([]string, 0, 2)
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--dry-run":
			plan.DryRun = true
		case arg == "--yes", arg == "-y":
			plan.Yes = true
		case arg == "--help", arg == "-h":
			plan.Command = "help"
			return plan, nil
		case arg == "--module", arg == "--target", arg == "--content", arg == "--storage", arg == "--with", arg == "--dir":
			if i+1 >= len(args) {
				return Plan{}, fmt.Errorf("%s needs a value", arg)
			}
			i++
			if err := applyFlag(&plan, arg, args[i]); err != nil {
				return Plan{}, err
			}
		case strings.HasPrefix(arg, "--module="), strings.HasPrefix(arg, "--target="), strings.HasPrefix(arg, "--content="), strings.HasPrefix(arg, "--storage="), strings.HasPrefix(arg, "--with="), strings.HasPrefix(arg, "--dir="):
			key, value, _ := strings.Cut(arg, "=")
			if err := applyFlag(&plan, key, value); err != nil {
				return Plan{}, err
			}
		case strings.HasPrefix(arg, "-"):
			return Plan{}, fmt.Errorf("unknown flag %s\n\n%s", arg, helpText)
		default:
			positionals = append(positionals, arg)
		}
	}

	switch plan.Command {
	case "init":
		if len(positionals) > 1 {
			return Plan{}, fmt.Errorf("init accepts one directory")
		}
		if len(positionals) == 1 {
			plan.Dir = positionals[0]
		}
		if strings.TrimSpace(plan.Module) == "" {
			return Plan{}, fmt.Errorf("init requires --module, for example github.com/you/shop")
		}
		if err := validateModule(plan.Module); err != nil {
			return Plan{}, err
		}
	case "add":
		if len(positionals) == 0 || len(positionals) > 2 {
			return Plan{}, fmt.Errorf("usage: fastygo add <markdown|view|codex> [dir]")
		}
		plan.Add = positionals[0]
		if len(positionals) == 2 {
			plan.Dir = positionals[1]
		}
		if err := validateOptional(plan.Add); err != nil {
			return Plan{}, err
		}
	}

	if plan.Target != "local" && plan.Target != "vps" && plan.Target != "vercel" {
		return Plan{}, fmt.Errorf("unknown target %q (fastygo %s accepts local, vps, or vercel)", plan.Target, Version)
	}
	if plan.Content != "fixtures" && plan.Content != "codex" {
		return Plan{}, fmt.Errorf("unknown content %q (fastygo %s accepts fixtures or codex)", plan.Content, Version)
	}
	if plan.Storage != "bbolt" {
		return Plan{}, fmt.Errorf("unknown storage %q (fastygo %s accepts bbolt)", plan.Storage, Version)
	}
	if plan.Command == "init" && plan.Target == "vercel" && plan.With["codex"] {
		return Plan{}, fmt.Errorf("vercel does not install a local Codex server; use --content codex and set CODEX_ORIGIN")
	}
	if plan.Storage != "" && plan.Command == "init" && !plan.With["codex"] && storageSet(args) {
		return Plan{}, fmt.Errorf("storage applies to the local Codex server; pass --with codex")
	}
	return plan, nil
}

func applyFlag(plan *Plan, key, value string) error {
	switch key {
	case "--module":
		plan.Module = value
	case "--target":
		plan.Target = value
	case "--content":
		plan.Content = value
	case "--storage":
		plan.Storage = value
	case "--dir":
		plan.Dir = value
	case "--with":
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if part == "render" {
				plan.With["render"] = true
				continue
			}
			if err := validateOptional(part); err != nil {
				return err
			}
			plan.With[part] = true
		}
	default:
		return fmt.Errorf("unknown flag %s", key)
	}
	return nil
}

func validateOptional(name string) error {
	switch name {
	case "markdown", "view", "codex":
		return nil
	default:
		return fmt.Errorf("unknown feature %q (expected markdown, view, or codex)", name)
	}
}

func validateModule(module string) error {
	if strings.ContainsAny(module, " \t\\") || strings.Contains(module, "..") {
		return fmt.Errorf("invalid module path %q", module)
	}
	if !strings.Contains(module, "/") {
		return fmt.Errorf("invalid module path %q (expected a path such as github.com/you/shop)", module)
	}
	return nil
}

func storageSet(args []string) bool {
	for _, arg := range args {
		if arg == "--storage" || strings.HasPrefix(arg, "--storage=") {
			return true
		}
	}
	return false
}
