# platform/

OS/arch-specific concerns nothing else owned. Not a placeholder — real,
used surface.

- `paths.go` — `CacheDir`/`ConfigDir` (`os.UserCacheDir`/`os.UserConfigDir`-
  backed), the first config/cache-directory convention this codebase has
  ever had. Used today by [lspmanager](lsp-stack.md) for installed
  language servers; reusable later for settings/plugin state.
- `arch.go` — `MasonTargetCandidates`/`CurrentTargets` map Go's
  `GOOS`/`GOARCH` onto mason-registry's own target vocabulary
  (`linux_x64_gnu`, `darwin_arm64`, etc.), most-specific-first, with a
  best-effort musl-vs-glibc detection on Linux.

## Depends on

Stdlib only (`os`, `runtime`).

## Depended on by

[internal/lspmanager](lsp-stack.md) — the only current consumer.
