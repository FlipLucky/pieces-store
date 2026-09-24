package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCacheDirAppendsAppNameAndCreatesIt(t *testing.T) {
	base := t.TempDir()
	setCacheEnv(t, base)

	dir, err := CacheDir()
	if err != nil {
		t.Fatalf("CacheDir() error = %v", err)
	}
	if filepath.Base(dir) != appDirName {
		t.Errorf("CacheDir() = %q, want a path ending in %q", dir, appDirName)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("CacheDir() = %q was not created as a directory (err=%v)", dir, err)
	}
}

func TestConfigDirAppendsAppNameAndCreatesIt(t *testing.T) {
	base := t.TempDir()
	setConfigEnv(t, base)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error = %v", err)
	}
	if filepath.Base(dir) != appDirName {
		t.Errorf("ConfigDir() = %q, want a path ending in %q", dir, appDirName)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("ConfigDir() = %q was not created as a directory (err=%v)", dir, err)
	}
}

func TestLSPServersDirIsUnderCacheDir(t *testing.T) {
	base := t.TempDir()
	setCacheEnv(t, base)

	cache, err := CacheDir()
	if err != nil {
		t.Fatalf("CacheDir() error = %v", err)
	}
	servers, err := LSPServersDir()
	if err != nil {
		t.Fatalf("LSPServersDir() error = %v", err)
	}
	want := filepath.Join(cache, "lsp-servers")
	if servers != want {
		t.Errorf("LSPServersDir() = %q, want %q", servers, want)
	}
	if info, err := os.Stat(servers); err != nil || !info.IsDir() {
		t.Errorf("LSPServersDir() = %q was not created as a directory (err=%v)", servers, err)
	}
}

// setCacheEnv/setConfigEnv point os.UserCacheDir/os.UserConfigDir at base
// for the duration of the test, without needing to know each platform's
// exact env var — t.Setenv covers the variables Go's stdlib actually reads.
func setCacheEnv(t *testing.T, base string) {
	t.Helper()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("LocalAppData", base)
	case "darwin":
		t.Setenv("HOME", base)
	default:
		t.Setenv("XDG_CACHE_HOME", base)
	}
}

func setConfigEnv(t *testing.T, base string) {
	t.Helper()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("AppData", base)
	case "darwin":
		t.Setenv("HOME", base)
	default:
		t.Setenv("XDG_CONFIG_HOME", base)
	}
}
