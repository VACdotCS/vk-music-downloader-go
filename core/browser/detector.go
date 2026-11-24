package browser

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type BrowserType string

const (
	TypeChromium BrowserType = "chromium"
	TypeFirefox  BrowserType = "firefox"
	TypeSafari   BrowserType = "safari"
)

type BrowserInfo struct {
	Path string
	Type BrowserType
	Name string
}

// FindSupportedBrowser searches for any supported browser on the system.
// Priority: Chromium-based (Chrome, Edge, Brave, Chromium) -> Firefox -> Safari.
func FindSupportedBrowser() (*BrowserInfo, error) {
	browsers := DetectAllBrowsers()
	if len(browsers) == 0 {
		return nil, os.ErrNotExist
	}

	// Prefer Chromium browsers first for standard CDP support
	for _, b := range browsers {
		if b.Type == TypeChromium {
			return &b, nil
		}
	}

	// Fallback to Firefox
	for _, b := range browsers {
		if b.Type == TypeFirefox {
			return &b, nil
		}
	}

	return &browsers[0], nil
}

// GetBrowserByPath returns a BrowserInfo for a specific executable path.
func GetBrowserByPath(path string) (*BrowserInfo, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	lower := strings.ToLower(path)
	bType := TypeChromium
	name := "Chromium"
	if strings.Contains(lower, "firefox") {
		bType = TypeFirefox
		name = "Mozilla Firefox"
	} else if strings.Contains(lower, "edge") || strings.Contains(lower, "msedge") {
		name = "Microsoft Edge"
	} else if strings.Contains(lower, "chrome") {
		name = "Google Chrome"
	} else if strings.Contains(lower, "brave") {
		name = "Brave Browser"
	} else if strings.Contains(lower, "yandex") {
		name = "Yandex Browser"
	}
	return &BrowserInfo{
		Path: path,
		Type: bType,
		Name: name,
	}, nil
}

// DetectAllBrowsers scans the system for installed browsers.
func DetectAllBrowsers() []BrowserInfo {
	var found []BrowserInfo

	switch runtime.GOOS {
	case "windows":
		found = append(found, detectWindowsBrowsers()...)
	case "darwin":
		found = append(found, detectDarwinBrowsers()...)
	default: // linux, bsd, etc.
		found = append(found, detectLinuxBrowsers()...)
	}

	return found
}

func detectWindowsBrowsers() []BrowserInfo {
	var list []BrowserInfo

	type candidate struct {
		name    string
		bType   BrowserType
		envBase string
		subPath string
	}

	progFiles := os.Getenv("ProgramFiles")
	progFilesX86 := os.Getenv("ProgramFiles(x86)")
	localAppData := os.Getenv("LOCALAPPDATA")

	candidates := []candidate{
		// Google Chrome
		{"Google Chrome", TypeChromium, progFiles, filepath.Join("Google", "Chrome", "Application", "chrome.exe")},
		{"Google Chrome", TypeChromium, progFilesX86, filepath.Join("Google", "Chrome", "Application", "chrome.exe")},
		{"Google Chrome", TypeChromium, localAppData, filepath.Join("Google", "Chrome", "Application", "chrome.exe")},

		// Microsoft Edge
		{"Microsoft Edge", TypeChromium, progFilesX86, filepath.Join("Microsoft", "Edge", "Application", "msedge.exe")},
		{"Microsoft Edge", TypeChromium, progFiles, filepath.Join("Microsoft", "Edge", "Application", "msedge.exe")},
		{"Microsoft Edge", TypeChromium, localAppData, filepath.Join("Microsoft", "Edge", "Application", "msedge.exe")},

		// Brave Browser
		{"Brave Browser", TypeChromium, progFiles, filepath.Join("BraveSoftware", "Brave-Browser", "Application", "brave.exe")},
		{"Brave Browser", TypeChromium, localAppData, filepath.Join("BraveSoftware", "Brave-Browser", "Application", "brave.exe")},

		// Yandex Browser
		{"Yandex Browser", TypeChromium, localAppData, filepath.Join("Yandex", "YandexBrowser", "Application", "browser.exe")},

		// Mozilla Firefox
		{"Mozilla Firefox", TypeFirefox, progFiles, filepath.Join("Mozilla Firefox", "firefox.exe")},
		{"Mozilla Firefox", TypeFirefox, progFilesX86, filepath.Join("Mozilla Firefox", "firefox.exe")},
	}

	seen := make(map[string]bool)
	for _, c := range candidates {
		if c.envBase == "" {
			continue
		}
		fullPath := filepath.Join(c.envBase, c.subPath)
		if seen[fullPath] {
			continue
		}
		if _, err := os.Stat(fullPath); err == nil {
			seen[fullPath] = true
			list = append(list, BrowserInfo{
				Path: fullPath,
				Type: c.bType,
				Name: c.name,
			})
		}
	}

	return list
}

func detectDarwinBrowsers() []BrowserInfo {
	var list []BrowserInfo

	macApps := []struct {
		name  string
		bType BrowserType
		path  string
	}{
		{"Google Chrome", TypeChromium, "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"},
		{"Microsoft Edge", TypeChromium, "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"},
		{"Brave Browser", TypeChromium, "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"},
		{"Chromium", TypeChromium, "/Applications/Chromium.app/Contents/MacOS/Chromium"},
		{"Mozilla Firefox", TypeFirefox, "/Applications/Firefox.app/Contents/MacOS/firefox"},
		{"Safari", TypeSafari, "/Applications/Safari.app/Contents/MacOS/Safari"},
	}

	for _, app := range macApps {
		if _, err := os.Stat(app.path); err == nil {
			list = append(list, BrowserInfo{
				Path: app.path,
				Type: app.bType,
				Name: app.name,
			})
		}
	}

	return list
}

func detectLinuxBrowsers() []BrowserInfo {
	var list []BrowserInfo

	linuxBins := []struct {
		name  string
		bType BrowserType
		bin   string
	}{
		{"Google Chrome", TypeChromium, "google-chrome"},
		{"Google Chrome", TypeChromium, "google-chrome-stable"},
		{"Chromium", TypeChromium, "chromium"},
		{"Chromium", TypeChromium, "chromium-browser"},
		{"Microsoft Edge", TypeChromium, "microsoft-edge"},
		{"Microsoft Edge", TypeChromium, "microsoft-edge-stable"},
		{"Brave Browser", TypeChromium, "brave-browser"},
		{"Mozilla Firefox", TypeFirefox, "firefox"},
	}

	seen := make(map[string]bool)
	for _, b := range linuxBins {
		if p, err := exec.LookPath(b.bin); err == nil {
			if !seen[p] {
				seen[p] = true
				list = append(list, BrowserInfo{
					Path: p,
					Type: b.bType,
					Name: b.name,
				})
			}
		}
	}

	return list
}
