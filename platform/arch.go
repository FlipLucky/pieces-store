package platform

import (
	"path/filepath"
	"runtime"
)

// isMuslLinux is a best-effort check for a musl libc system (e.g. Alpine)
// vs. the far more common glibc — mason-registry's Linux assets are often
// published separately per libc, and running a glibc binary on musl (or
// vice versa) fails outright rather than just running slower. Not
// authoritative (a real answer needs parsing the binary's own dynamic
// section), but checking for musl's own loader is the same heuristic
// several real installers use and is good enough to order candidates
// correctly in the common case.
var isMuslLinux = func() bool {
	matches, _ := filepath.Glob("/lib/ld-musl-*.so*")
	return len(matches) > 0
}

// MasonTargetCandidates returns the mason-registry target strings worth
// trying for the given goos/goarch, most specific first. A package's own
// asset list rarely has every one of these — callers try each in order and
// use the first one the package actually publishes, so returning a short
// ordered list here (rather than trying to guess the single "right" one) is
// deliberate: it tolerates a package publishing only the bare linux_<arch>
// form, or only one of gnu/musl, without needing per-package special-casing
// in the caller.
func MasonTargetCandidates(goos, goarch, archSuffix string) []string {
	switch goos {
	case "darwin":
		return []string{"darwin_" + archSuffix}
	case "windows":
		return []string{"win_" + archSuffix}
	case "linux":
		gnu := "linux_" + archSuffix + "_gnu"
		musl := "linux_" + archSuffix + "_musl"
		bare := "linux_" + archSuffix
		if isMuslLinux() {
			return []string{musl, gnu, bare}
		}
		return []string{gnu, musl, bare}
	default:
		return nil
	}
}

// archSuffix maps Go's GOARCH values to mason-registry's own vocabulary —
// confirmed empirically against a real registry.json dump (amd64 -> x64,
// arm64 -> arm64; no other GOARCH value appears in the curated language
// server set's actual published assets).
func archSuffix(goarch string) (suffix string, ok bool) {
	switch goarch {
	case "amd64":
		return "x64", true
	case "arm64":
		return "arm64", true
	default:
		return "", false
	}
}

// CurrentTargets returns MasonTargetCandidates for the running binary's
// actual GOOS/GOARCH, or nil if this platform/arch combination has no known
// mason-registry mapping (e.g. a 32-bit build — not a target this project
// supports).
func CurrentTargets() []string {
	suffix, ok := archSuffix(runtime.GOARCH)
	if !ok {
		return nil
	}
	return MasonTargetCandidates(runtime.GOOS, runtime.GOARCH, suffix)
}
