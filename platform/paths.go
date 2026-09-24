// Package platform holds OS/arch-specific concerns that don't belong in any
// single feature package — where pieces-store keeps its own on-disk state,
// and how to map a Go GOOS/GOARCH pair onto the target vocabulary external
// package registries (mason-registry, for language servers) use. Nothing in
// this module had a config/cache directory convention before this package;
// it's the one place that decision gets made.
package platform

import (
	"os"
	"path/filepath"
)

// appDirName is the subdirectory pieces-store creates inside the OS's
// standard cache/config roots — never write directly into UserCacheDir()/
// UserConfigDir() itself, which are shared across every app on the machine.
const appDirName = "pieces-store"

// CacheDir returns (creating if necessary) the directory pieces-store uses
// for anything that can be safely deleted and re-fetched — installed
// language server binaries and their version metadata, primarily. Backed by
// os.UserCacheDir() (XDG_CACHE_HOME on Linux, ~/Library/Caches on macOS,
// %LocalAppData% on Windows).
func CacheDir() (string, error) {
	return ensureSubdir(os.UserCacheDir)
}

// ConfigDir returns (creating if necessary) the directory pieces-store uses
// for anything a user might actually want to hand-edit or back up — not
// used by the LSP installer today (installed servers are cache, not
// config), but established now so a future settings/plugin-state feature
// doesn't need to invent this convention from scratch.
func ConfigDir() (string, error) {
	return ensureSubdir(os.UserConfigDir)
}

// LSPServersDir returns (creating if necessary) the directory under
// CacheDir where internal/lspmanager installs language servers, one
// subdirectory per package.
func LSPServersDir() (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "lsp-servers")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func ensureSubdir(base func() (string, error)) (string, error) {
	root, err := base()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, appDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
