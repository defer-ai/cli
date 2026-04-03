package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	releasesURL = "https://api.github.com/repos/defer-ai/cli/releases/latest"
	timeout     = 3 * time.Second
)

// githubRelease represents the relevant fields from the GitHub releases API.
type githubRelease struct {
	TagName string        `json:"tag_name"`
	HTMLURL string        `json:"html_url"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckForUpdate checks the GitHub releases API for a newer version.
// Returns the latest version string and release HTML URL, or empty strings if
// already up to date. The "dev" version always returns empty (skip check).
// Errors are returned but callers should treat them as non-fatal.
func CheckForUpdate(currentVersion string) (latestVersion string, downloadURL string, err error) {
	if currentVersion == "dev" || currentVersion == "" {
		return "", "", nil
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(releasesURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", err
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	if CompareVersions(latest, current) > 0 {
		return latest, release.HTMLURL, nil
	}

	return "", "", nil
}

// CompareVersions compares two semver strings (without "v" prefix).
// Returns >0 if a > b, 0 if equal, <0 if a < b.
func CompareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		aVal := 0
		bVal := 0
		if i < len(aParts) {
			aVal, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bVal, _ = strconv.Atoi(bParts[i])
		}
		if aVal > bVal {
			return 1
		}
		if aVal < bVal {
			return -1
		}
	}
	return 0
}

// AssetName returns the expected goreleaser asset filename for the current
// OS and architecture.
func AssetName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map GOARCH to goreleaser naming conventions
	archMap := map[string]string{
		"amd64": "amd64",
		"arm64": "arm64",
		"386":   "386",
	}
	arch, ok := archMap[goarch]
	if !ok {
		arch = goarch
	}

	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}

	return fmt.Sprintf("defer_%s_%s.%s", goos, arch, ext)
}

// FindAssetURL searches a release's assets for the one matching the current platform.
// Goreleaser names assets as defer_VERSION_os_arch.ext, so we match the suffix.
func FindAssetURL(assets []githubAsset) string {
	suffix := AssetName() // e.g. "defer_linux_amd64.tar.gz"
	// Match: asset name contains the os_arch.ext part
	// e.g. "defer_2.0.2_linux_amd64.tar.gz" contains "linux_amd64.tar.gz"
	parts := strings.SplitN(suffix, "defer_", 2)
	if len(parts) < 2 {
		return ""
	}
	platformSuffix := parts[1] // "linux_amd64.tar.gz"
	for _, a := range assets {
		if strings.HasSuffix(a.Name, platformSuffix) {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

// FetchLatestRelease fetches the full release info (including assets) from GitHub.
func FetchLatestRelease() (*githubRelease, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(releasesURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

// DownloadAndReplace downloads the release asset, extracts the "defer" binary,
// and replaces the currently running executable.
func DownloadAndReplace(assetURL string) error {
	// Find current executable path, resolving symlinks
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find current executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("cannot resolve symlinks: %w", err)
	}

	// Download asset to temp file
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(assetURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Save to a temp file
	tmpArchive, err := os.CreateTemp("", "defer-update-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpArchivePath := tmpArchive.Name()
	defer os.Remove(tmpArchivePath)

	if _, err := io.Copy(tmpArchive, resp.Body); err != nil {
		tmpArchive.Close()
		return fmt.Errorf("download write failed: %w", err)
	}
	tmpArchive.Close()

	// Extract the binary
	binaryName := "defer"
	if runtime.GOOS == "windows" {
		binaryName = "defer.exe"
	}

	var binaryData []byte
	if strings.HasSuffix(assetURL, ".zip") {
		binaryData, err = extractFromZip(tmpArchivePath, binaryName)
	} else {
		binaryData, err = extractFromTarGz(tmpArchivePath, binaryName)
	}
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Write new binary to a temp file next to the target
	dir := filepath.Dir(execPath)
	tmpBin, err := os.CreateTemp(dir, "defer-new-*")
	if err != nil {
		return fmt.Errorf("cannot create temp binary: %w", err)
	}
	tmpBinPath := tmpBin.Name()

	if _, err := tmpBin.Write(binaryData); err != nil {
		tmpBin.Close()
		os.Remove(tmpBinPath)
		return fmt.Errorf("cannot write new binary: %w", err)
	}
	tmpBin.Close()

	// Make executable
	if err := os.Chmod(tmpBinPath, 0o755); err != nil {
		os.Remove(tmpBinPath)
		return fmt.Errorf("cannot set permissions: %w", err)
	}

	// Atomic-ish replacement: rename old, move new, remove old
	oldPath := execPath + ".old"
	os.Remove(oldPath) // clean up any previous .old

	if err := os.Rename(execPath, oldPath); err != nil {
		os.Remove(tmpBinPath)
		return fmt.Errorf("cannot move old binary: %w", err)
	}

	if err := os.Rename(tmpBinPath, execPath); err != nil {
		// Try to restore old binary
		os.Rename(oldPath, execPath)
		return fmt.Errorf("cannot install new binary: %w", err)
	}

	os.Remove(oldPath)
	return nil
}

// extractFromTarGz extracts a named file from a .tar.gz archive.
func extractFromTarGz(archivePath, targetName string) ([]byte, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Match the binary name (might be nested in a directory)
		name := filepath.Base(hdr.Name)
		if name == targetName && hdr.Typeflag == tar.TypeReg {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("%s not found in archive", targetName)
}

// extractFromZip extracts a named file from a .zip archive.
func extractFromZip(archivePath, targetName string) ([]byte, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name == targetName && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("%s not found in archive", targetName)
}
