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

// tviewStyleTag returns the opening color tag for style, and false for
// types.StyleNone — nothing to wrap, same as an unstyled rune today.
// Each frontend owns this Style -> color mapping itself, matching
// gui-base's styleColor and how Mode's status-bar color already works.
func tviewStyleTag(style types.Style) (string, bool) {
	switch style {
	case types.StyleKeyword:
		return "[#bb9af7:#1a1b26]", true
	case types.StyleString:
		return "[#9ece6a:#1a1b26]", true
	case types.StyleComment:
		return "[#565f89:#1a1b26]", true
	case types.StyleNumber:
		return "[#ff9e64:#1a1b26]", true
	case types.StyleFunction:
		return "[#7aa2f7:#1a1b26]", true
	case types.StyleType:
		return "[#2ac3de:#1a1b26]", true
	case types.StyleDiagnosticError:
		return "[#f7768e:#1a1b26]", true
	case types.StyleDiagnosticWarning:
		return "[#e0af68:#1a1b26]", true
	default:
		return "", false
	}
}

// styledLineText renders one line as a tview color-tagged string: the
// cursor cell (if this is the cursor's line) takes priority over any
// styled span at the same position, otherwise each rune gets its span's
// color tag, or none at all — identical output to before StyledSpans
// existed when spans is empty, since tviewStyleTag(StyleNone) is (_, false).
func styledLineText(line string, lineStart int, spans []viewmanager.StyledSpan, isCursorLine bool, cursorCol int) string {
	runes := []rune(line)
	var b strings.Builder
	byteOffset := 0
	for i, r := range runes {
		switch {
		case isCursorLine && i == cursorCol:
			// Tokyo Night cursor block styling: dark text on bright blue
			// background, then reset to normal fg on the dark background.
			b.WriteString("[#1a1b26:#7aa2f7]")
			b.WriteRune(r)
			b.WriteString("[#a9b1d6:#1a1b26]")
		default:
			if open, ok := tviewStyleTag(viewmanager.StyleAt(spans, lineStart+byteOffset)); ok {
				b.WriteString(open)
				b.WriteRune(r)
				b.WriteString("[#a9b1d6:#1a1b26]")
			} else {
				b.WriteRune(r)
			}
		}
		byteOffset += len(string(r))
	}
	if isCursorLine && cursorCol >= len(runes) {
		b.WriteString("[#1a1b26:#7aa2f7] [#a9b1d6:#1a1b26]")
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

	const maxCompletionRows = 8
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

		var formattedLines []string
		for i, line := range slice.Lines {
			absRow := slice.StartRow + i
			lineStart := slice.LineOffsets[i]
			isCursorLine := absRow == cursor.Row
			lineSpans := viewmanager.SpansForRange(spans, lineStart, lineStart+len(line))
			if !isCursorLine && len(lineSpans) == 0 {
				formattedLines = append(formattedLines, line)
				continue
			}
			formattedLines = append(formattedLines, styledLineText(line, lineStart, lineSpans, isCursorLine, cursor.Col))
		}

		editorView.SetText(strings.Join(formattedLines, "\n"))

		cursorRow := cursor.Row - slice.StartRow
		scrollRow := cursorRow - cursorTopPadding
		if scrollRow < 0 {
			scrollRow = 0
		}
		editorView.ScrollTo(scrollRow, 0)

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

			statusBar.SetText(fmt.Sprintf(" %s | %s | Row: [#9ece6a]%d[white] Col: [#9ece6a]%d[white] | Offset: [#bb9af7]%d[white]",
				modeStr, filePath, cursor.Row, cursor.Col, cursor.ByteOffset))
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

	return app.SetRoot(flex, true).Run()
}
