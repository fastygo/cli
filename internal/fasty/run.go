package fasty

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// IO is the process boundary used by tests.
type IO struct {
	In       io.Reader
	Out      io.Writer
	Err      io.Writer
	LookPath func(string) (string, error)
	Run      func(ctx context.Context, name string, args []string, dir string, env []string) error
	TTY      bool
	GOOS     string
	GOARCH   string
}

// Main runs fastygo with the current process.
func Main(args []string) error {
	info, _ := os.Stdin.Stat()
	return Execute(context.Background(), args, IO{
		In:       os.Stdin,
		Out:      os.Stdout,
		Err:      os.Stderr,
		LookPath: exec.LookPath,
		Run:      runCommand,
		TTY:      info != nil && info.Mode()&os.ModeCharDevice != 0,
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
	})
}

func runCommand(ctx context.Context, name string, args []string, dir string, env []string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if dir == "" {
		cmd.Dir = "."
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Execute parses args and installs or prints a plan.
func Execute(ctx context.Context, args []string, io IO) error {
	if io.Out == nil {
		io.Out = os.Stdout
	}
	if io.Err == nil {
		io.Err = os.Stderr
	}
	if io.LookPath == nil {
		io.LookPath = exec.LookPath
	}
	if io.Run == nil {
		io.Run = runCommand
	}
	if io.GOOS == "" {
		io.GOOS = runtime.GOOS
	}
	if io.GOARCH == "" {
		io.GOARCH = runtime.GOARCH
	}

	plan, err := parseArgs(args)
	if err != nil {
		return err
	}
	switch plan.Command {
	case "help":
		fmt.Fprint(io.Out, helpText)
		return nil
	case "version":
		fmt.Fprintf(io.Out, "fastygo %s\n", Version)
		return nil
	}

	if !Supported(io.GOOS, io.GOARCH) {
		return fmt.Errorf("fastygo %s installs toolchains for windows/amd64, linux/amd64, and darwin/arm64 (this system is %s/%s)", Version, io.GOOS, io.GOARCH)
	}

	dir, err := filepath.Abs(plan.Dir)
	if err != nil {
		return err
	}
	plan.Dir = dir

	switch plan.Command {
	case "init":
		return initProject(ctx, plan, io)
	case "add":
		return addFeature(ctx, plan, io)
	default:
		return fmt.Errorf("unknown command %s", plan.Command)
	}
}

func initProject(ctx context.Context, plan Plan, io IO) error {
	if _, err := os.Stat(filepath.Join(plan.Dir, "go.mod")); err == nil {
		return fmt.Errorf("%s already has go.mod; fastygo init stops here. Use fastygo add to bring in markdown, view, or codex", plan.Dir)
	}
	goBin, note, err := resolveGo(ctx, plan, io)
	if err != nil {
		return err
	}
	if plan.DryRun {
		return printInit(io.Out, plan, note)
	}
	if err := os.MkdirAll(plan.Dir, 0o755); err != nil {
		return err
	}
	if err := writeSite(plan.Dir, plan); err != nil {
		return err
	}
	env := goEnv(goBin, "")
	if err := io.Run(ctx, goBin, []string{"mod", "init", plan.Module}, plan.Dir, env); err != nil {
		return fmt.Errorf("go mod init: %w", err)
	}
	gets := []string{"get", framework, renderMod}
	if plan.With["markdown"] {
		gets = append(gets, markdownMod)
	}
	if plan.With["view"] {
		gets = append(gets, viewMod)
	}
	if err := io.Run(ctx, goBin, gets, plan.Dir, env); err != nil {
		return fmt.Errorf("go get: %w", err)
	}
	if err := io.Run(ctx, goBin, []string{"get", "-tool", templTool}, plan.Dir, env); err != nil {
		return fmt.Errorf("go get -tool: %w", err)
	}
	if err := pinGoLine(plan.Dir); err != nil {
		return err
	}
	if err := io.Run(ctx, goBin, []string{"tool", "templ", "generate", "./cmd/site"}, plan.Dir, env); err != nil {
		return fmt.Errorf("templ generate: %w", err)
	}
	if err := io.Run(ctx, goBin, []string{"mod", "tidy"}, plan.Dir, env); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}
	if err := pinGoLine(plan.Dir); err != nil {
		return err
	}
	if plan.With["codex"] {
		if err := installCodex(ctx, plan.Dir, goBin, io); err != nil {
			return err
		}
	}
	fmt.Fprintf(io.Out, "installed %s\n", plan.Module)
	fmt.Fprintln(io.Out, "run: go run ./cmd/site")
	printManagers(io)
	return nil
}

func addFeature(ctx context.Context, plan Plan, io IO) error {
	if _, err := os.Stat(filepath.Join(plan.Dir, "go.mod")); err != nil {
		return fmt.Errorf("%s has no go.mod. Run fastygo init first", plan.Dir)
	}
	goBin, note, err := resolveGo(ctx, plan, io)
	if err != nil {
		return err
	}
	if plan.DryRun {
		fmt.Fprintf(io.Out, "would add %s in %s\n%s\n", plan.Add, plan.Dir, note)
		return nil
	}
	env := goEnv(goBin, "")
	switch plan.Add {
	case "markdown":
		if err := ensureFeatureFile(plan.Dir, "markdown"); err != nil {
			return err
		}
		if err := io.Run(ctx, goBin, []string{"get", markdownMod}, plan.Dir, env); err != nil {
			return err
		}
	case "view":
		if err := ensureFeatureFile(plan.Dir, "view"); err != nil {
			return err
		}
		if err := io.Run(ctx, goBin, []string{"get", viewMod}, plan.Dir, env); err != nil {
			return err
		}
	case "codex":
		if err := installCodex(ctx, plan.Dir, goBin, io); err != nil {
			return err
		}
	}
	if err := io.Run(ctx, goBin, []string{"mod", "tidy"}, plan.Dir, env); err != nil {
		return err
	}
	if err := pinGoLine(plan.Dir); err != nil {
		return err
	}
	appendAgents(plan.Dir, plan.Add)
	fmt.Fprintf(io.Out, "added %s\n", plan.Add)
	return nil
}

func resolveGo(ctx context.Context, plan Plan, io IO) (string, string, error) {
	if bin, err := io.LookPath(goBinaryName()); err == nil {
		versionOut, verErr := goVersion(ctx, bin, io)
		if verErr == nil && goAtLeast125(versionOut) {
			return bin, "go " + versionOut, nil
		}
	}
	note := "Go 1.25 was not found. fastygo can download it into the user cache and use it for this install."
	if plan.DryRun {
		return "", note, nil
	}
	ok, err := confirm(io.In, io.Out, io.TTY, plan.Yes, note)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", fmt.Errorf("Go 1.25 is required")
	}
	home, err := toolchainHome()
	if err != nil {
		return "", "", err
	}
	bin, err := installGoRelease(ctx, nil, goIndexURL, home, io.GOOS, io.GOARCH)
	if err != nil {
		return "", "", err
	}
	fmt.Fprintf(io.Out, "Go is at %s\nAdd its bin directory to the user PATH for later terminals.\n", bin)
	return bin, "downloaded " + bin, nil
}

func goVersion(ctx context.Context, bin string, io IO) (string, error) {
	cmd := exec.CommandContext(ctx, bin, "version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	version, ok := parseGoVersion(string(out))
	if !ok {
		return "", fmt.Errorf("cannot parse %q", strings.TrimSpace(string(out)))
	}
	return version, nil
}

func printInit(out io.Writer, plan Plan, goNote string) error {
	fmt.Fprintf(out, "fastygo %s dry-run\n", Version)
	fmt.Fprintf(out, "module: %s\n", plan.Module)
	fmt.Fprintf(out, "dir: %s\n", plan.Dir)
	fmt.Fprintf(out, "target: %s\n", plan.Target)
	fmt.Fprintf(out, "go: %s\n", goNote)
	fmt.Fprintf(out, "would go get %s\n", framework)
	fmt.Fprintf(out, "would go get %s\n", renderMod)
	if plan.With["markdown"] {
		fmt.Fprintf(out, "would go get %s\n", markdownMod)
	}
	if plan.With["view"] {
		fmt.Fprintf(out, "would go get %s\n", viewMod)
	}
	fmt.Fprintf(out, "would go get -tool %s\n", templTool)
	fmt.Fprintln(out, "would write AGENTS.md package.json .env.example cmd/site/main.go cmd/site/home.templ")
	if plan.With["codex"] {
		fmt.Fprintf(out, "would go install %s into .bin (storage %s)\n", codexPkg, plan.Storage)
	}
	return nil
}

func installCodex(ctx context.Context, dir, goBin string, io IO) error {
	binDir := filepath.Join(dir, ".bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "data"), 0o755); err != nil {
		return err
	}
	example := filepath.Join(dir, "codex.env.example")
	if _, err := os.Stat(example); err != nil {
		if writeErr := os.WriteFile(example, []byte(codexEnv), 0o644); writeErr != nil {
			return writeErr
		}
	}
	env := goEnv(goBin, binDir)
	if err := io.Run(ctx, goBin, []string{"install", codexPkg}, dir, env); err != nil {
		return fmt.Errorf("go install codex: %w", err)
	}
	name := "server"
	if runtime.GOOS == "windows" {
		name = "server.exe"
	}
	fmt.Fprintf(io.Out, "codex binary: %s\n", filepath.Join(binDir, name))
	return nil
}

func goEnv(goBin, gobin string) []string {
	env := make([]string, 0, len(os.Environ())+4)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch strings.ToUpper(key) {
		case "GOWORK", "GOTOOLCHAIN", "GOBIN", "PATH":
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "GOWORK=off", "GOTOOLCHAIN=local")
	path := os.Getenv("PATH")
	if goBin != "" {
		path = filepath.Dir(goBin) + string(os.PathListSeparator) + path
	}
	env = append(env, "PATH="+path)
	if gobin != "" {
		env = append(env, "GOBIN="+gobin)
	}
	return env
}

func pinGoLine(dir string) error {
	path := filepath.Join(dir, "go.mod")
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var out []string
	seen := false
	for _, line := range strings.Split(string(body), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "toolchain ") {
			continue
		}
		if strings.HasPrefix(trim, "go ") {
			if !seen {
				out = append(out, "go "+goLine)
				seen = true
			}
			continue
		}
		out = append(out, line)
	}
	if !seen {
		out = append([]string{"go " + goLine}, out...)
	}
	text := strings.Join(out, "\n")
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

func ensureFeatureFile(dir, feature string) error {
	path := filepath.Join(dir, "cmd", "site", "features.go")
	body, err := os.ReadFile(path)
	if err != nil {
		plan := Plan{With: map[string]bool{feature: true}}
		if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
			return mkErr
		}
		return os.WriteFile(path, []byte(featuresGo(plan)), 0o644)
	}
	text := string(body)
	importPath := markdownMod
	if feature == "view" {
		importPath = viewMod
	}
	importPath = strings.Split(importPath, "@")[0]
	if strings.Contains(text, importPath) {
		return nil
	}
	plan := Plan{With: map[string]bool{"markdown": strings.Contains(text, "modules/markdown"), "view": strings.Contains(text, "modules/view")}}
	plan.With[feature] = true
	return os.WriteFile(path, []byte(featuresGo(plan)), 0o644)
}

func appendAgents(dir, feature string) {
	path := filepath.Join(dir, "AGENTS.md")
	body, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var line string
	switch feature {
	case "markdown":
		line = "Markdown: github.com/fastygo/modules/markdown v0.1.0. The package name is contentmarkdown.\n"
	case "view":
		line = "Theme and language data: github.com/fastygo/modules/view v0.1.0.\n"
	case "codex":
		line = "Codex binary is in .bin from github.com/fastygo/backend/cmd/server v0.1.0. Call it over HTTP.\n"
	}
	if strings.Contains(string(body), line) {
		return
	}
	_ = os.WriteFile(path, append(body, []byte("\n"+line)...), 0o644)
}

func printManagers(io IO) {
	found := false
	for _, name := range []string{"npm", "pnpm", "yarn", "bun"} {
		if _, err := io.LookPath(name); err == nil {
			found = true
			break
		}
		if runtime.GOOS == "windows" {
			if _, err := io.LookPath(name + ".cmd"); err == nil {
				found = true
				break
			}
		}
	}
	if !found {
		fmt.Fprintln(io.Out, "npm, pnpm, yarn, and bun were not found. The site still runs with: go run ./cmd/site")
	}
}
