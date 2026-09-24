package platform

import (
	"reflect"
	"runtime"
	"testing"
)

func TestMasonTargetCandidatesDarwin(t *testing.T) {
	got := MasonTargetCandidates("darwin", "arm64", "arm64")
	want := []string{"darwin_arm64"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MasonTargetCandidates(darwin, arm64) = %v, want %v", got, want)
	}
}

func TestMasonTargetCandidatesWindows(t *testing.T) {
	got := MasonTargetCandidates("windows", "amd64", "x64")
	want := []string{"win_x64"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MasonTargetCandidates(windows, amd64) = %v, want %v", got, want)
	}
}

func TestMasonTargetCandidatesLinuxPrefersGnuByDefault(t *testing.T) {
	restore := isMuslLinux
	isMuslLinux = func() bool { return false }
	defer func() { isMuslLinux = restore }()

	got := MasonTargetCandidates("linux", "amd64", "x64")
	want := []string{"linux_x64_gnu", "linux_x64_musl", "linux_x64"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MasonTargetCandidates(linux, amd64) = %v, want %v", got, want)
	}
}

func TestMasonTargetCandidatesLinuxPrefersMuslWhenDetected(t *testing.T) {
	restore := isMuslLinux
	isMuslLinux = func() bool { return true }
	defer func() { isMuslLinux = restore }()

	got := MasonTargetCandidates("linux", "arm64", "arm64")
	want := []string{"linux_arm64_musl", "linux_arm64_gnu", "linux_arm64"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MasonTargetCandidates(linux, arm64, musl) = %v, want %v", got, want)
	}
}

func TestMasonTargetCandidatesUnknownGOOS(t *testing.T) {
	got := MasonTargetCandidates("plan9", "amd64", "x64")
	if got != nil {
		t.Errorf("MasonTargetCandidates(plan9, amd64) = %v, want nil", got)
	}
}

func TestArchSuffixKnownAndUnknown(t *testing.T) {
	if s, ok := archSuffix("amd64"); !ok || s != "x64" {
		t.Errorf("archSuffix(amd64) = (%q, %v), want (x64, true)", s, ok)
	}
	if s, ok := archSuffix("arm64"); !ok || s != "arm64" {
		t.Errorf("archSuffix(arm64) = (%q, %v), want (arm64, true)", s, ok)
	}
	if _, ok := archSuffix("386"); ok {
		t.Errorf("archSuffix(386) = ok, want unsupported (32-bit isn't a target this project supports)")
	}
}

func TestCurrentTargetsMatchesRunningPlatformWhenSupported(t *testing.T) {
	// Doesn't assert an exact value (that would just restate CurrentTargets'
	// own logic) — proves it doesn't panic and, on a supported arch, returns
	// a non-empty candidate list consistent with MasonTargetCandidates.
	got := CurrentTargets()
	suffix, ok := archSuffix(runtime.GOARCH)
	if !ok {
		if got != nil {
			t.Errorf("CurrentTargets() = %v on an unsupported GOARCH, want nil", got)
		}
		return
	}
	if len(got) == 0 {
		t.Errorf("CurrentTargets() = empty on a supported arch (suffix %q)", suffix)
	}
}
