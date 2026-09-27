package fasty

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoVersionFloor(t *testing.T) {
	version, ok := parseGoVersion("go version go1.25.5 windows/amd64")
	if !ok || version != "1.25.5" || !goAtLeast125(version) {
		t.Fatalf("version %q ok=%v", version, ok)
	}
	old, ok := parseGoVersion("go version go1.23.0 linux/amd64")
	if !ok || goAtLeast125(old) {
		t.Fatal("1.23 should be below the floor")
	}
}

func TestSelectGo125(t *testing.T) {
	index := []goRelease{
		{Version: "go1.24.4", Stable: true, Files: []goFile{{Filename: "old.zip", OS: "windows", Arch: "amd64", Kind: "archive"}}},
		{Version: "go1.25.0", Stable: true, Files: []goFile{{Filename: "a.zip", OS: "windows", Arch: "amd64", Kind: "archive", SHA256: "aa"}}},
		{Version: "go1.25.5", Stable: true, Files: []goFile{{Filename: "b.zip", OS: "windows", Arch: "amd64", Kind: "archive", SHA256: "bb"}}},
		{Version: "go1.25.9", Stable: false, Files: []goFile{{Filename: "c.zip", OS: "windows", Arch: "amd64", Kind: "archive"}}},
	}
	file, version, err := selectGo125(index, "windows", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if version != "go1.25.5" || file.Filename != "b.zip" {
		t.Fatalf("got %s %s", version, file.Filename)
	}
	if Supported("windows", "386") {
		t.Fatal("386 is outside this release")
	}
}

func TestConfirmNonTTYNeedsYes(t *testing.T) {
	var out bytes.Buffer
	ok, err := confirm(strings.NewReader(""), &out, false, false, "Go 1.25 was not found.")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected a stop without --yes")
	}
	if !strings.Contains(out.String(), "--yes") {
		t.Fatalf("hint missing: %s", out.String())
	}
	ok, err = confirm(strings.NewReader(""), &out, false, true, "Go 1.25 was not found.")
	if err != nil || !ok {
		t.Fatalf("yes=%v err=%v", ok, err)
	}
}

func TestInstallGoReleaseChecksSumAndExtracts(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	entry, err := zw.Create("go/bin/go.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("fake-go")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	payload := buf.Bytes()
	sum := sha256.Sum256(payload)
	index := `[{"version":"go1.25.5","stable":true,"files":[{"filename":"go1.25.5.windows-amd64.zip","os":"windows","arch":"amd64","sha256":"` + hex.EncodeToString(sum[:]) + `","kind":"archive"}]}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dl/":
			_, _ = w.Write([]byte(index))
		case "/dl/go1.25.5.windows-amd64.zip":
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	bin, err := installGoRelease(context.Background(), srv.Client(), srv.URL+"/dl/?mode=json", t.TempDir(), "windows", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "fake-go" {
		t.Fatalf("extracted %q", body)
	}
	if filepath.Base(bin) != "go.exe" {
		t.Fatalf("bin %s", bin)
	}
}
