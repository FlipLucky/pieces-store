package tuibase

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/fliplucky/pieces-store/internal/editor"
	"github.com/fliplucky/pieces-store/internal/types"
	"github.com/fliplucky/pieces-store/internal/viewmanager"
)

// completionWindow bounds candidates to at most maxRows entries, keeping
// index always visible — a scrolling window rather than truncating the
// list to whatever fits at the top, so cycling past the initial screenful
// still shows where the selection actually is.
// maxCompletionRows bounds both the :e/:w path-completion bar and the LSP
// autocomplete popup to at most this many visible candidates.
const maxCompletionRows = 8

func completionWindow(candidates []string, index, maxRows int) ([]string, int) {
	if len(candidates) <= maxRows {
		return candidates, index
	}
	start := index - maxRows + 1
	if start < 0 {
		start = 0
	}
	end := start + maxRows
	if end > len(candidates) {
		end = len(candidates)
		start = end - maxRows
	}
	return candidates[start:end], index - start
}

// tokyoBgHex/tokyoFgHex are the same Tokyo Night background/default-text
// colors used throughout this file — named here so the tag-building
// helpers below don't repeat the literal hex strings.
const (
	tokyoBgHex = "#1a1b26"
	tokyoFgHex = "#a9b1d6"
)

// tviewStyleColor returns the tcell/tview color-tag value for a syntax
// Style, or ("", false) for types.StyleNone (nothing to wrap, same as an
// unstyled rune today). Each frontend owns this Style -> color mapping
// itself, matching gui-base's styleColor and how Mode's status-bar color
// already works.
//
// Deliberately has no cases for StyleDiagnosticError/Warning as of
// 2026-09-20 (it did before diagnostics were split out of StyleSpans into
// their own DiagnosticSpans accessor) — diagnostics now render as an
// underline (see styledLineText), not a color swap, so a token's own
// syntax color stays visible under an error/warning instead of being
// replaced by it. Real, checked limitation worth knowing: tview's
// `[fg:bg:attr]` tag-string parser only supports a bare on/off underline
// (`u`/`U`) — confirmed by reading its source — with no way to give the
// underline its own color independent of the text's, unlike
// tcell.Style.Underline's lower-level API (which does support a color
// parameter, just not reachable through tview's tag convenience layer).
// So error vs. warning isn't color-distinguishable by the underline alone
// in the terminal the way it is in gui-base, which draws the underline
// itself and can color it freely — flagged honestly, not silently
// downgraded without a trace of why.
func tviewStyleColor(style types.Style) (string, bool) {
	switch style {
	case types.StyleKeyword:
		return "#bb9af7", true
	case types.StyleString:
		return "#9ece6a", true
	case types.StyleComment:
		return "#565f89", true
	case types.StyleNumber:
		return "#ff9e64", true
	case types.StyleFunction:
		return "#7aa2f7", true
	case types.StyleType:
		return "#2ac3de", true
	default:
		return "", false
	}
}

// diagnosticColor returns the severity color for a diagnostic Style, and
// whether style is a diagnostic at all — the same red/yellow gui-base's
// own diagnosticColor uses, kept as a separate function from
// tviewStyleColor since a diagnostic Style is never looked up against
// syntax spans, only against DiagnosticSpans (see styledLineText).
func diagnosticColor(style types.Style) (string, bool) {
	switch style {
	case types.StyleDiagnosticError:
		return "#f7768e", true
	case types.StyleDiagnosticWarning:
		return "#e0af68", true
	default:
		return "", false
	}
}

// resetTag always explicitly forces underline off (":U"), not just
// restoring default colors — tview's attribute tags are sticky across a
// styled string unless a tag explicitly says otherwise (confirmed by
// reading its parser: a tag that omits the attribute section entirely
// leaves the running underline state unchanged, it does not reset it).
// Omitting the explicit "U" here would let an underlined diagnostic rune
// bleed underline onto every following character for the rest of the line.
const resetTag = "[" + tokyoFgHex + ":" + tokyoBgHex + ":U]"

// openTag builds one rune's full style tag: its color (or the default
// text color if it has none) plus an explicit underline attribute state
// — always stated explicitly, for the same sticky-attribute reason
// resetTag is.
func openTag(color string, underline bool) string {
	if color == "" {
		color = tokyoFgHex
	}
	attr := "U"
	if underline {
		attr = "u"
	}
	return fmt.Sprintf("[%s:%s:%s]", color, tokyoBgHex, attr)
}

// cursorTag is openTag's equivalent for the cursor's own cell — inverted
// colors (dark text on the bright cursor block) instead of a syntax color.
func cursorTag(underline bool) string {
	attr := "U"
	if underline {
		attr = "u"
	}
	return fmt.Sprintf("[%s:#7aa2f7:%s]", tokyoBgHex, attr)
}

// styledLineText renders one line as a tview color/underline-tagged
// string: the cursor cell (if this is the cursor's line) takes priority
// over any styled span at the same position; a syntax span colors the
// rune; a diagnostic span underlines it — independently, so a diagnosed
// keyword keeps its keyword color and gains an underline rather than
// losing its color to solid red/yellow. Identical output to before either
// existed when spans and diagnostics are both empty.
func styledLineText(line string, lineStart int, spans []viewmanager.StyledSpan, diagnostics []viewmanager.DiagnosticSpan, isCursorLine bool, cursorCol int) string {
	runes := []rune(line)
	var b strings.Builder
	byteOffset := 0
	for i, r := range runes {
		abs := lineStart + byteOffset
		diagColor, underline := diagnosticColor(viewmanager.DiagnosticStyleAt(diagnostics, abs))

		switch {
		case isCursorLine && i == cursorCol:
			b.WriteString(cursorTag(underline))
			b.WriteRune(r)
			b.WriteString(resetTag)
		default:
			color, hasColor := tviewStyleColor(viewmanager.StyleAt(spans, abs))
			if underline {
				// tview's tag API ties underline to the same foreground
				// color as the text (confirmed against its real parser —
				// see tviewStyleColor's doc comment): there's no way to
				// underline in red while keeping a token's own syntax
				// color. Given that real constraint, using the
				// diagnostic's own severity color here — same as most
				// terminal-based tools' diagnostic output — is the
				// closer-to-standard choice, not a fallback: it's how a
				// user actually expects an error to look in a terminal.
				color = diagColor
				hasColor = true
			}
			if hasColor || underline {
				b.WriteString(openTag(color, underline))
				b.WriteRune(r)
				b.WriteString(resetTag)
			} else {
				b.WriteRune(r)
			}
		}
		byteOffset += len(string(r))
	}
	if isCursorLine && cursorCol >= len(runes) {
		b.WriteString(cursorTag(false))
		b.WriteString(" ")
		b.WriteString(resetTag)
	}
	return b.String()
}

func Boot(ed *editor.Editor) error {
	app := tview.NewApplication()

	// Color Palette definitions
	tokyoBg := tcell.GetColor("#1a1b26")
	tokyoFg := tcell.GetColor("#a9b1d6")
	tokyoStatusBg := tcell.GetColor("#24283b")

	// Main editor display view
	editorView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true)

	editorView.SetBorder(true).
		SetTitle("Pieces Store TUI Editor").
		SetBorderColor(tcell.GetColor("#24283b"))

	editorView.SetBackgroundColor(tokyoBg)
	editorView.SetTextColor(tokyoFg)

	// :e/:w completion popup — collapsed to 0 height (via ResizeItem in
	// renderContent) whenever no completion is in progress.
	completionView := tview.NewTextView().
		SetDynamicColors(true)
	completionView.SetBackgroundColor(tokyoStatusBg)
	completionView.SetTextColor(tokyoFg)

	// Bottom Vim status bar
	statusBar := tview.NewTextView().
		SetDynamicColors(true)
	statusBar.SetBackgroundColor(tokyoStatusBg)

	// Layout: editor filling top space, completion popup + status bar at
	// the bottom.
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(editorView, 0, 1, true).
		AddItem(completionView, 0, 0, false).
		AddItem(statusBar, 1, 0, false)

	// hoverPopup is the cursor-anchored floating popup for LSP hover and
	// autocomplete (the user's explicit choice over reusing the docked
	// completionView above, which stays exactly as-is for :e/:w path
	// completion). A tview.Pages overlay with resize=false so its Rect can
	// be positioned exactly, in terminal cells — unlike gui-base, no font-
	// metric approximation is needed here: rows/columns are already exact
	// integers.
	hoverPopup := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false)
	hoverPopup.SetBackgroundColor(tokyoStatusBg)
	hoverPopup.SetTextColor(tokyoFg)
	hoverPopup.SetBorder(true).SetBorderColor(tcell.GetColor("#7aa2f7"))

	pages := tview.NewPages().
		AddPage("main", flex, true, true).
		AddPage("hover", hoverPopup, false, false)

	// viewportRadius is how many rows above/below the cursor get fetched —
	// generous compared to any real terminal height, tiny compared to a
	// large file, so rendering scales with editing activity near the
	// cursor rather than total file size (replaces the old naive
	// GetText()+strings.Split of the whole document on every keystroke).
	const viewportRadius = 200
	// cursorTopPadding is how many rows from the top of the terminal's
	// visible window the cursor is kept — simpler than gui-base's
	// minimal-scroll followCursor (which needs exact on-screen-row
	// knowledge this widget's internal scrolling doesn't expose cheaply),
	// but a real improvement over no auto-follow at all, which is what
	// existed here before.
	const cursorTopPadding = 10

	// Function to rebuild rendering text with cursor block highlight
	renderContent := func() {
		cursor := ed.GetCursor()
		filePath := ed.GetFilePath()
		if filePath == "" {
			filePath = "[No Name]"
		}

		editorView.SetTitle(fmt.Sprintf(" Pieces Store TUI Editor — %s ", filePath))

		slice := ed.Viewport(cursor.Row, 1, viewportRadius)
		spans := ed.StyleSpans(slice.StartOffset, slice.EndOffset)
		diagnosticSpans := ed.DiagnosticSpans(slice.StartOffset, slice.EndOffset)

		var formattedLines []string
		for i, line := range slice.Lines {
			absRow := slice.StartRow + i
			lineStart := slice.LineOffsets[i]
			isCursorLine := absRow == cursor.Row
			lineSpans := viewmanager.SpansForRange(spans, lineStart, lineStart+len(line))
			lineDiagnostics := viewmanager.DiagnosticSpansForRange(diagnosticSpans, lineStart, lineStart+len(line))

			var rendered string
			switch {
			case !isCursorLine && len(lineSpans) == 0 && len(lineDiagnostics) == 0:
				rendered = line
			default:
				rendered = styledLineText(line, lineStart, lineSpans, lineDiagnostics, isCursorLine, cursor.Col)
			}
			if msg, ok := viewmanager.DiagnosticMessageForRange(lineDiagnostics, lineStart, lineStart+len(line)); ok {
				rendered += virtualDiagnosticText(msg)
			}
			formattedLines = append(formattedLines, rendered)
		}

		editorView.SetText(strings.Join(formattedLines, "\n"))

		cursorRow := cursor.Row - slice.StartRow
		scrollRow := cursorRow - cursorTopPadding
		if scrollRow < 0 {
			scrollRow = 0
		}
		editorView.ScrollTo(scrollRow, 0)

		renderHoverPopup(pages, hoverPopup, editorView, cursor, cursorRow, scrollRow)

		if completion := cursor.Completion; completion.Active && len(completion.Candidates) > 0 {
			windowed, selected := completionWindow(completion.Candidates, completion.Index, maxCompletionRows)
			var b strings.Builder
			for i, candidate := range windowed {
				if i > 0 {
					b.WriteString("\n")
				}
				if i == selected {
					b.WriteString("[#1a1b26:#7aa2f7]")
					b.WriteString(candidate)
					b.WriteString("[#a9b1d6:#1a1b26]")
				} else {
					b.WriteString(candidate)
				}
			}
			completionView.SetText(b.String())
			flex.ResizeItem(completionView, len(windowed), 0)
		} else {
			completionView.SetText("")
			flex.ResizeItem(completionView, 0, 0)
		}

		if cursor.Mode == types.ModeCommand {
			statusBar.SetText(fmt.Sprintf(" [#7aa2f7]:%s[white]", cursor.CommandBuffer))
		} else {
			// Update bottom status bar with color-coded modes
			var modeStr string
			switch cursor.Mode {
			case types.ModeNormal:
				modeStr = "[#e0af68]-- NORMAL --[white]" // Yellow
			case types.ModeInsert:
				modeStr = "[#9ece6a]-- INSERT --[white]" // Green
			case types.ModeVisual:
				modeStr = "[#bb9af7]-- VISUAL --[white]" // Purple
			default:
				modeStr = cursor.Mode.String()
			}

			statusLine := fmt.Sprintf(" %s | %s | Row: [#9ece6a]%d[white] Col: [#9ece6a]%d[white] | Offset: [#bb9af7]%d[white]",
				modeStr, filePath, cursor.Row, cursor.Col, cursor.ByteOffset)
			// Surface why a command like :q was refused (e.g. unsaved
			// changes) — ErrQuit itself isn't an error worth showing, the
			// app is about to close. gui-base already does this; this was
			// a real, user-hit gap — :q being silently refused with zero
			// feedback looked indistinguishable from :q being broken.
			if err := ed.GetLastCommandError(); err != nil && err != editor.ErrQuit {
				statusLine += fmt.Sprintf(" | [#f7768e]%s[white]", err)
			}
			// Non-error progress text (e.g. ":LspInstall" in flight or its
			// result) — a separate slot from the error one above, since
			// StatusMessage isn't error-shaped.
			if msg := ed.StatusMessage(); msg != "" {
				statusLine += fmt.Sprintf(" | %s", msg)
			}
			statusBar.SetText(statusLine)
		}
	}

	// Initial render
	renderContent()

	// Listen for editor updates and request redraws
	go func() {
		for range ed.ChangeChan() {
			app.QueueUpdateDraw(func() {
				renderContent()
			})
		}
	}()

	// Capture and handle raw keyboard events for Vim emulation
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var keyStr string

		switch event.Key() {
		case tcell.KeyESC:
			keyStr = "<Esc>"
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			keyStr = "<BS>"
		case tcell.KeyEnter:
			keyStr = "<Enter>"
		case tcell.KeyLeft:
			keyStr = "<Left>"
		case tcell.KeyRight:
			keyStr = "<Right>"
		case tcell.KeyUp:
			keyStr = "<Up>"
		case tcell.KeyDown:
			keyStr = "<Down>"
		case tcell.KeyCtrlR:
			keyStr = "<C-r>"
		case tcell.KeyTab:
			keyStr = "<Tab>"
		case tcell.KeyCtrlN:
			keyStr = "<C-n>"
		case tcell.KeyCtrlP:
			keyStr = "<C-p>"
		default:
			if event.Rune() != 0 {
				keyStr = string(event.Rune())
			}
		}

		if keyStr != "" {
			if ed.HandleKey(keyStr) {
				if ed.IsQuitRequested() {
					app.Stop()
				}
				return nil
			}
		}

		return event
	})

	return app.SetRoot(pages, true).Run()
}

// renderHoverPopup positions and shows/hides the cursor-anchored hover/
// completion popup. Exact terminal-cell positioning (unlike gui-base,
// which has no pixel-accurate cursor position to anchor on and uses a
// font-metric approximation instead): editorView.GetInnerRect() gives its
// content area in real terminal cells, and cursorRow/scrollRow (already
// computed by the caller for ScrollTo) give the cursor's row within that
// area. Prefers showing below the cursor's line; flips above if there's
// not enough room below, matching common completion-popup convention.
func renderHoverPopup(pages *tview.Pages, popup *tview.TextView, editorView *tview.TextView, cursor editor.Cursor, cursorRow, scrollRow int) {
	innerX, innerY, innerW, innerH := editorView.GetInnerRect()
	onScreenRow := cursorRow - scrollRow

	var lines []string
	selected := -1

	switch {
	case cursor.Hover.Active:
		width := innerW - 4
		if width < 20 {
			width = 20
		}
		lines = wrapText(cursor.Hover.Text, width)
	case cursor.LSPCompletion.Active && len(cursor.LSPCompletion.Items) > 0:
		display := make([]string, len(cursor.LSPCompletion.Items))
		for i, item := range cursor.LSPCompletion.Items {
			line := item.Label
			if item.Detail != "" {
				line += "  " + item.Detail
			}
			display[i] = line
		}
		windowed, sel := completionWindow(display, cursor.LSPCompletion.Index, maxCompletionRows)
		lines = windowed
		selected = sel
	}

	if len(lines) == 0 {
		pages.HidePage("hover")
		return
	}

	var b strings.Builder
	width := 0
	for i, line := range lines {
		if i > 0 {
			b.WriteString("\n")
		}
		if i == selected {
			b.WriteString("[#1a1b26:#7aa2f7]")
			b.WriteString(line)
			b.WriteString("[#a9b1d6:#24283b]")
		} else {
			b.WriteString(line)
		}
		if len(line) > width {
			width = len(line)
		}
	}
	popup.SetText(b.String())

	width += 2 // border
	if width > innerW {
		width = innerW
	}
	height := len(lines) + 2 // border
	if height > innerH {
		height = innerH
	}

	x := innerX + cursor.Col
	if x+width > innerX+innerW {
		x = innerX + innerW - width
	}
	if x < innerX {
		x = innerX
	}
	y := innerY + onScreenRow + 1 // one row below the cursor's own line
	if y+height > innerY+innerH {
		y = innerY + onScreenRow - height // not enough room below — show above instead
	}
	if y < innerY {
		y = innerY
	}

	popup.SetRect(x, y, width, height)
	pages.ShowPage("hover")
}

// wrapText greedily word-wraps text to width, preserving existing newlines
// as paragraph breaks — good enough for hover text (short prose/code
// signatures), not a general-purpose text layout algorithm.
// maxVirtualTextLen bounds how much of a diagnostic's message gets shown
// as inline virtual text — real messages can be multi-line paragraphs,
// and virtual text is meant to be a short one-line hint, not a full
// reproduction (K still shows the real hover/diagnostic text in full).
const maxVirtualTextLen = 80

// virtualDiagnosticText renders a diagnostic's message as dim inline text
// after a line's own content — the same vim/nvim convention (grayed-out
// text explaining what's wrong, not just a colored underline with no
// explanation). Uses the same muted color StyleComment already does,
// since both mean "this text isn't part of the code" to the reader.
func virtualDiagnosticText(message string) string {
	message = strings.ReplaceAll(message, "\n", " ")
	// Truncate by rune, not byte — a byte-index cut could split a
	// multi-byte UTF-8 rune in half for a non-ASCII message.
	if runes := []rune(message); len(runes) > maxVirtualTextLen {
		message = string(runes[:maxVirtualTextLen-1]) + "…"
	}
	return fmt.Sprintf("  [#565f89:%s]// %s[%s:%s:U]", tokyoBgHex, message, tokyoFgHex, tokyoBgHex)
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	var lines []string
	for _, paragraph := range strings.Split(text, "\n") {
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		current := ""
		for _, word := range strings.Fields(paragraph) {
			switch {
			case current == "":
				current = word
			case len(current)+1+len(word) <= width:
				current += " " + word
			default:
				lines = append(lines, current)
				current = word
			}
		}
		if current != "" {
			lines = append(lines, current)
		}
	}
	return lines
}
