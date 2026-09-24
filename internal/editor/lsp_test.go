package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fliplucky/pieces-store/internal/lspmanager"
	"github.com/fliplucky/pieces-store/internal/types"
	"github.com/fliplucky/pieces-store/platform"
)

// TestLanguageIDCoversEveryCuratedLanguage is a cheap, always-run guard
// against silently shipping a new curated Language without a matching
// LSP languageId.
func TestLanguageIDCoversEveryCuratedLanguage(t *testing.T) {
	all := []types.Language{
		types.LanguageMarkdown, types.LanguageHTML, types.LanguageGo, types.LanguageJSON,
		types.LanguageJavaScript, types.LanguageTypeScript, types.LanguageTSX, types.LanguageCSS,
		types.LanguageSCSS, types.LanguagePHP, types.LanguageDart, types.LanguageC,
		types.LanguageCPP, types.LanguageYAML, types.LanguageDockerfile,
	}
	for _, lang := range all {
		if got := languageID(lang); got == "" || got == "plaintext" {
			t.Errorf("languageID(%s) = %q, want a real LSP languageId", lang, got)
		}
	}
}

// TestRealDiagnosticsFlowIntoStyleSpans is the real end-to-end proof for
// the whole LSP-diagnostics wiring: install a real gopls, open a real .go
// file with an actual compile error through a live Editor, and confirm a
// real StyleDiagnosticError span shows up in DiagnosticSpans — not a mock
// diagnostic, an actual one gopls computed. Gated behind
// LSPMANAGER_INTEGRATION=1, matching this project's established
// network/toolchain-gated discipline (isolateLSPCacheDir keeps this off
// the real machine's cache dir regardless).
//
// Observed to take up to ~2 minutes under `go test` specifically (vs. ~5s
// for the identical `go install` command run standalone in a shell) —
// confirmed via repeated measurement, not a one-off fluke: real resource
// contention in a sandboxed environment between the parent `go test`
// process and the child `go install`/gopls processes, not a defect in this
// code. Pass a generous `-timeout` (150s+) when running this directly.
func TestRealDiagnosticsFlowIntoStyleSpans(t *testing.T) {
	if os.Getenv("LSPMANAGER_INTEGRATION") == "" {
		t.Skip("set LSPMANAGER_INTEGRATION=1 to run a real gopls end-to-end")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go toolchain on PATH")
	}
	isolateLSPCacheDir(t)

	serversDir, err := platform.LSPServersDir()
	if err != nil {
		t.Fatalf("platform.LSPServersDir() error = %v", err)
	}
	entry, _ := lspmanager.EntryFor(types.LanguageGo)
	specs, err := lspmanager.LoadPackages(t.TempDir(), map[string]bool{"gopls": true}, time.Hour, nil)
	if err != nil {
		t.Fatalf("LoadPackages() error = %v", err)
	}
	installed, err := lspmanager.Install(serversDir, specs["gopls"], entry, nil)
	if err != nil {
		t.Fatalf("Install(gopls) error = %v", err)
	}
	if err := lspmanager.RecordInstalled(serversDir, installed); err != nil {
		t.Fatalf("RecordInstalled() error = %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module probe\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A real, unambiguous compile error: calling an undefined function.
	source := "package main\n\nfunc main() {\n\tundefinedFunctionCall()\n}\n"
	mainGo := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainGo, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	e, err := NewEditorFromFile(mainGo)
	if err != nil {
		t.Fatalf("NewEditorFromFile() error = %v", err)
	}
	// Same-package white-box access (this project's established testing
	// convention) specifically to avoid leaving a real gopls process
	// orphaned after the test exits — Editor has no public Close/Stop
	// today (a known, accepted gap elsewhere in this codebase), so a real
	// running session would leak this too; not fixing that gap here, just
	// not compounding it by leaking a process on every test run.
	t.Cleanup(func() {
		e.mu.Lock()
		server := e.lspServer
		e.mu.Unlock()
		if server != nil {
			_ = server.Stop()
		}
	})

	deadline := time.Now().Add(30 * time.Second)
	var spans int
	for time.Now().Before(deadline) {
		s := e.DiagnosticSpans(0, len(source))
		for _, span := range s {
			if span.Style == types.StyleDiagnosticError {
				spans++
			}
		}
		if spans > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if spans == 0 {
		t.Fatalf("no StyleDiagnosticError span appeared within the deadline for a file with a real compile error; last status: %s", e.StatusMessage())
	}
	t.Logf("real diagnostic span(s) found: %d, status: %s", spans, e.StatusMessage())
}

// TestRealHoverAutocompleteAndFormatEndToEnd proves hover, autocomplete
// (trigger detection, applying a real TextEdit-based completion), and
// formatting all work through a live Editor against a real gopls — not
// just that lspclient can talk to gopls (already proven in its own
// package), but that Editor's own new logic (offset conversion, generation
// staleness, multi-edit ordering, the reentrant-lock-sensitive dispatch
// wiring) is actually correct. A locking mistake here would hang this test
// rather than fail it cleanly — a real, useful property given how many of
// these methods thread through the async bridge and executeCommandLocked.
func TestRealHoverAutocompleteAndFormatEndToEnd(t *testing.T) {
	if os.Getenv("LSPMANAGER_INTEGRATION") == "" {
		t.Skip("set LSPMANAGER_INTEGRATION=1 to run a real gopls end-to-end")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go toolchain on PATH")
	}
	isolateLSPCacheDir(t)

	serversDir, err := platform.LSPServersDir()
	if err != nil {
		t.Fatalf("platform.LSPServersDir() error = %v", err)
	}
	entry, _ := lspmanager.EntryFor(types.LanguageGo)
	specs, err := lspmanager.LoadPackages(t.TempDir(), map[string]bool{"gopls": true}, time.Hour, nil)
	if err != nil {
		t.Fatalf("LoadPackages() error = %v", err)
	}
	installed, err := lspmanager.Install(serversDir, specs["gopls"], entry, nil)
	if err != nil {
		t.Fatalf("Install(gopls) error = %v", err)
	}
	if err := lspmanager.RecordInstalled(serversDir, installed); err != nil {
		t.Fatalf("RecordInstalled() error = %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module probe\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Deliberately mis-indented (a real gofmt violation: a space instead
	// of a tab) so :Format has a genuine, verifiable change to make.
	source := "package main\n\nimport \"fmt\"\n\nfunc main() {\n fmt.Println(\"hi\")\n}\n"
	mainGo := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainGo, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	e, err := NewEditorFromFile(mainGo)
	if err != nil {
		t.Fatalf("NewEditorFromFile() error = %v", err)
	}
	t.Cleanup(func() {
		e.mu.Lock()
		server := e.lspServer
		e.mu.Unlock()
		if server != nil {
			_ = server.Stop()
		}
	})

	waitForLSPReady(t, e)

	// --- Hover: place the cursor inside "Println" and request hover ---
	printlnOffset := strings.Index(source, "Println") + 2
	e.mu.Lock()
	e.cursor.Update(printlnOffset, 0, 0)
	e.mu.Unlock()
	e.RequestHover()

	waitFor(t, 15*time.Second, func() bool {
		return e.GetCursor().Hover.Active
	}, "hover never became active")
	hoverText := e.GetCursor().Hover.Text
	if hoverText == "" || !strings.Contains(hoverText, "Println") {
		t.Errorf("hover text = %q, want it to mention Println", hoverText)
	}
	t.Logf("real hover text: %s", hoverText)

	// A subsequent cursor move must dismiss the hover popup.
	e.MoveCursorLeft()
	if e.GetCursor().Hover.Active {
		t.Error("hover still active after a cursor move — should have been dismissed")
	}

	// --- Autocomplete: type "fmt." in Insert mode and confirm the first candidate ---
	// Position right after the existing fmt.Println(...) call, still
	// inside main()'s body — not after the function's closing brace,
	// which would put the new statement at invalid package scope.
	afterCall := strings.Index(source, `fmt.Println("hi")`) + len(`fmt.Println("hi")`)
	e.mu.Lock()
	e.cursor.Update(afterCall, 0, 0)
	e.mu.Unlock()
	e.HandleKey("i")
	e.HandleKey("<Enter>")
	e.HandleKey("f")
	e.HandleKey("m")
	e.HandleKey("t")
	e.HandleKey(".") // "." is gopls's real advertised trigger character

	waitFor(t, 15*time.Second, func() bool {
		return e.GetCursor().LSPCompletion.Active
	}, "LSP completion never became active after typing a trigger character")
	items := e.GetCursor().LSPCompletion.Items
	if len(items) == 0 {
		t.Fatal("LSPCompletion.Active but Items is empty")
	}
	t.Logf("real completion items: %d, first label: %s", len(items), items[0].Label)

	beforeConfirm := e.GetText()
	e.HandleKey("<Enter>") // confirm the highlighted (first) candidate
	if e.GetCursor().LSPCompletion.Active {
		t.Error("LSPCompletion still active after confirming — should have cleared")
	}
	afterConfirm := e.GetText()
	if afterConfirm == beforeConfirm {
		t.Error("confirming a completion did not change the buffer at all")
	}
	if !strings.Contains(afterConfirm, items[0].Label) {
		t.Errorf("buffer after confirm = %q, want it to contain the confirmed label %q", afterConfirm, items[0].Label)
	}
	t.Logf("buffer after confirm: %q", afterConfirm)

	e.SetMode(types.ModeNormal)

	// --- Formatting: the original file has a real gofmt violation ---
	if err := e.ExecuteCommand(":Format"); err != nil {
		t.Fatalf("ExecuteCommand(:Format) error = %v", err)
	}
	waitFor(t, 15*time.Second, func() bool {
		msg := e.StatusMessage()
		return msg == "formatted" || strings.Contains(msg, "failed")
	}, "format never reached a terminal status")
	if msg := e.StatusMessage(); strings.Contains(msg, "failed") {
		t.Fatalf("format reported failure: %s", msg)
	}
	formatted := e.GetText()
	if strings.Contains(formatted, "\n fmt.Println") {
		t.Errorf("buffer still contains the mis-indented line after :Format: %q", formatted)
	}
	if !strings.Contains(formatted, "\tfmt.Println") {
		t.Errorf("formatted buffer = %q, want a tab-indented fmt.Println line", formatted)
	}
	t.Logf("formatted buffer: %q", formatted)
}

func waitForLSPReady(t *testing.T, e *Editor) {
	t.Helper()
	waitFor(t, 30*time.Second, func() bool {
		e.mu.RLock()
		defer e.mu.RUnlock()
		return e.lspServer != nil
	}, "LSP server never became ready; last status: "+e.StatusMessage())
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool, failMsg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	if !cond() {
		t.Fatal(failMsg)
	}
}
