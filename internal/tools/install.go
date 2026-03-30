// Package tools manages downloading and installing external tool binaries.
package tools

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Tool describes an installable tool.
type Tool struct {
	Name       string // binary name (e.g., "aid-gen-go")
	Repo       string // GitHub repo (e.g., "dan-strohschein/aid-gen-go")
	BinaryName string // name of the binary in the archive (may differ from Name)
}

// KnownTools maps tool names to their metadata.
var KnownTools = map[string]Tool{
	"aid-gen-go": {
		Name:       "aid-gen-go",
		Repo:       "dan-strohschein/aid-gen-go",
		BinaryName: "aid-gen-go",
	},
	"aid-gen-ts": {
		Name:       "aid-gen-ts",
		Repo:       "dan-strohschein/aid-gen-ts",
		BinaryName: "aid-gen-ts",
	},
	"aid-gen-cs": {
		Name:       "aid-gen-cs",
		Repo:       "dan-strohschein/aid-gen-cs",
		BinaryName: "aid-gen-cs",
	},
	"aid-gen": {
		Name:       "aid-gen",
		Repo:       "dan-strohschein/AID-Docs",
		BinaryName: "aid-gen",
	},
}

// InstallDir returns the directory where tools are installed.
func InstallDir() string {
	// Prefer ~/go/bin for Go users, fall back to /usr/local/bin
	gobin := os.Getenv("GOBIN")
	if gobin != "" {
		return gobin
	}
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		return filepath.Join(gopath, "bin")
	}
	home, err := os.UserHomeDir()
	if err == nil {
		goDir := filepath.Join(home, "go", "bin")
		if _, err := os.Stat(goDir); err == nil {
			return goDir
		}
	}
	return "/usr/local/bin"
}

// Install downloads and installs a tool from GitHub releases.
func Install(toolName string) error {
	tool, ok := KnownTools[toolName]
	if !ok {
		available := make([]string, 0, len(KnownTools))
		for k := range KnownTools {
			available = append(available, k)
		}
		return fmt.Errorf("unknown tool: %s\nAvailable: %s", toolName, strings.Join(available, ", "))
	}

	tag, err := getLatestRelease(tool.Repo)
	if err != nil {
		return fmt.Errorf("checking latest release: %w", err)
	}

	osName := runtime.GOOS
	arch := runtime.GOARCH

	fmt.Printf("Installing %s %s for %s/%s...\n", tool.Name, tag, osName, arch)

	suffix := fmt.Sprintf("%s-%s", osName, arch)
	archiveName := fmt.Sprintf("%s-%s.tar.gz", tool.BinaryName, suffix)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", tool.Repo, tag, archiveName)

	fmt.Printf("Downloading %s...\n", url)

	tmpDir, err := os.MkdirTemp("", "squire-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, archiveName)
	if err := downloadFile(url, archivePath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Extract
	binaryPath, err := extractTarGz(archivePath, tmpDir, tool.BinaryName)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Install
	installDir := InstallDir()
	destPath := filepath.Join(installDir, tool.Name)

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("creating install directory: %w", err)
	}

	// Copy binary to install location
	if err := copyFile(binaryPath, destPath); err != nil {
		// Try with sudo message
		return fmt.Errorf("installing to %s: %w\nTry: sudo mv %s %s", destPath, err, binaryPath, destPath)
	}

	if err := os.Chmod(destPath, 0755); err != nil {
		return err
	}

	fmt.Printf("✓ Installed %s to %s\n", tool.Name, destPath)

	// Save version info
	saveVersion(tool.Name, tag)

	return nil
}

// Upgrade updates squire and all installed tools.
func Upgrade() error {
	installDir := InstallDir()

	upgraded := 0
	for name, tool := range KnownTools {
		binPath := filepath.Join(installDir, name)
		if _, err := os.Stat(binPath); err != nil {
			continue // not installed
		}

		currentVersion := getInstalledVersion(name)
		latestTag, err := getLatestRelease(tool.Repo)
		if err != nil {
			fmt.Printf("  ⚠ %s: could not check for updates: %v\n", name, err)
			continue
		}

		if currentVersion == latestTag {
			fmt.Printf("  ✓ %s %s (up to date)\n", name, currentVersion)
			continue
		}

		fmt.Printf("  ↑ %s %s → %s\n", name, currentVersion, latestTag)
		if err := Install(name); err != nil {
			fmt.Printf("  ⚠ %s: upgrade failed: %v\n", name, err)
			continue
		}
		upgraded++
	}

	if upgraded == 0 {
		fmt.Println("All tools up to date.")
	} else {
		fmt.Printf("%d tool(s) upgraded.\n", upgraded)
	}
	return nil
}

// ListInstalled returns all installed tools with their versions.
func ListInstalled() map[string]string {
	installed := map[string]string{}
	installDir := InstallDir()

	for name := range KnownTools {
		binPath := filepath.Join(installDir, name)
		if _, err := os.Stat(binPath); err == nil {
			ver := getInstalledVersion(name)
			if ver == "" {
				ver = "unknown"
			}
			installed[name] = ver
		}
	}
	return installed
}

// --- GitHub API ---

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func getLatestRelease(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, repo)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

// --- File operations ---

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func extractTarGz(archivePath, destDir, binaryName string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		name := filepath.Base(header.Name)
		if strings.HasPrefix(name, binaryName) && header.Typeflag == tar.TypeReg {
			outPath := filepath.Join(destDir, name)
			outFile, err := os.Create(outPath)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return "", err
			}
			outFile.Close()
			os.Chmod(outPath, 0755)
			return outPath, nil
		}
	}

	return "", fmt.Errorf("binary %q not found in archive", binaryName)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// --- Version tracking ---

func versionFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".squire", "versions.json")
}

func saveVersion(tool, version string) {
	versions := loadVersions()
	versions[tool] = version

	dir := filepath.Dir(versionFile())
	os.MkdirAll(dir, 0755)

	data, _ := json.MarshalIndent(versions, "", "  ")
	os.WriteFile(versionFile(), data, 0644)
}

func getInstalledVersion(tool string) string {
	versions := loadVersions()
	return versions[tool]
}

func loadVersions() map[string]string {
	data, err := os.ReadFile(versionFile())
	if err != nil {
		return map[string]string{}
	}
	var versions map[string]string
	json.Unmarshal(data, &versions)
	if versions == nil {
		return map[string]string{}
	}
	return versions
}
