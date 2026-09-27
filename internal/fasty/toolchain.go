package fasty

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const goIndexURL = "https://go.dev/dl/?mode=json"

// Supported reports whether fastygo 0.1.0 installs a toolchain for this OS and CPU.
func Supported(goos, goarch string) bool {
	switch goos + "/" + goarch {
	case "windows/amd64", "linux/amd64", "darwin/arm64":
		return true
	default:
		return false
	}
}

func parseGoVersion(output string) (string, bool) {
	fields := strings.Fields(output)
	for _, field := range fields {
		if strings.HasPrefix(field, "go1.") {
			return strings.TrimPrefix(field, "go"), true
		}
	}
	return "", false
}

func goAtLeast125(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}
	return major > 1 || (major == 1 && minor >= 25)
}

type goFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	SHA256   string `json:"sha256"`
	Kind     string `json:"kind"`
}

type goRelease struct {
	Version string   `json:"version"`
	Stable  bool     `json:"stable"`
	Files   []goFile `json:"files"`
}

func selectGo125(index []goRelease, goos, goarch string) (goFile, string, error) {
	var best string
	var file goFile
	for _, rel := range index {
		if !rel.Stable || !strings.HasPrefix(rel.Version, "go1.25.") {
			continue
		}
		if best != "" && rel.Version <= best {
			continue
		}
		for _, candidate := range rel.Files {
			if candidate.OS == goos && candidate.Arch == goarch && candidate.Kind == "archive" {
				best = rel.Version
				file = candidate
			}
		}
	}
	if best == "" {
		return goFile{}, "", fmt.Errorf("no Go 1.25 archive for %s/%s", goos, goarch)
	}
	return file, best, nil
}

func confirm(in io.Reader, out io.Writer, tty, yes bool, message string) (bool, error) {
	if yes {
		fmt.Fprintln(out, message)
		fmt.Fprintln(out, "continuing because --yes was set")
		return true, nil
	}
	fmt.Fprintln(out, message)
	if !tty {
		fmt.Fprintln(out, "stdin is not a terminal. Re-run with --yes to download and continue.")
		return false, nil
	}
	fmt.Fprint(out, "Download and continue? [Y/n] ")
	var answer string
	_, err := fmt.Fscanln(in, &answer)
	if err != nil && answer == "" {
		// A bare Enter is a yes. Fscanln reports that as a newline error.
		return true, nil
	}
	if err != nil {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer == "" || answer == "y" || answer == "yes" {
		return true, nil
	}
	return false, nil
}

func installGoRelease(ctx context.Context, client *http.Client, indexURL, dest, goos, goarch string) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("go release index returned %s", res.Status)
	}
	var index []goRelease
	if err := json.NewDecoder(res.Body).Decode(&index); err != nil {
		return "", err
	}
	file, version, err := selectGo125(index, goos, goarch)
	if err != nil {
		return "", err
	}
	archiveURL, err := releaseURL(indexURL, file.Filename)
	if err != nil {
		return "", err
	}
	body, err := getBytes(ctx, client, archiveURL)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != file.SHA256 {
		return "", fmt.Errorf("checksum mismatch for %s", file.Filename)
	}
	root := filepath.Join(dest, version)
	if err := os.RemoveAll(root); err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	if err := extractGo(body, file.Filename, root); err != nil {
		return "", err
	}
	bin := filepath.Join(root, "go", "bin", "go")
	if goos == "windows" {
		bin += ".exe"
	}
	if _, err := os.Stat(bin); err != nil {
		return "", fmt.Errorf("go binary missing after extract: %s", bin)
	}
	return bin, nil
}

func releaseURL(indexURL, filename string) (string, error) {
	parsed, err := url.Parse(indexURL)
	if err != nil {
		return "", err
	}
	parsed.RawQuery = ""
	parsed.Path = path.Join(path.Dir(parsed.Path), filename)
	return parsed.String(), nil
}

func getBytes(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s returned %s", rawURL, res.Status)
	}
	return io.ReadAll(res.Body)
}

func extractGo(body []byte, filename, dest string) error {
	if strings.HasSuffix(filename, ".zip") {
		return extractZip(body, dest)
	}
	return extractTarGz(body, dest)
}

func extractZip(body []byte, dest string) error {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}
	for _, file := range reader.File {
		if err := extractZipFile(file, dest); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(file *zip.File, dest string) error {
	target, err := safeJoin(dest, file.Name)
	if err != nil {
		return err
	}
	if file.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	mode := file.Mode()
	if mode == 0 {
		mode = 0o644
	}
	return writeFile(target, src, mode)
}

func extractTarGz(body []byte, dest string) error {
	gz, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := safeJoin(dest, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(hdr.Mode) & 0o777
			if mode == 0 {
				mode = 0o644
			}
			if err := writeFile(target, tr, mode); err != nil {
				return err
			}
		}
	}
}

func writeFile(path string, src io.Reader, mode os.FileMode) error {
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func safeJoin(root, name string) (string, error) {
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive path escapes destination: %s", name)
	}
	return filepath.Join(root, clean), nil
}

func toolchainHome() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "fastygo", "go"), nil
}

func goBinaryName() string {
	if runtime.GOOS == "windows" {
		return "go.exe"
	}
	return "go"
}
