package tuibase

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/fliplucky/pieces-store/internal/editor"
	"github.com/fliplucky/pieces-store/internal/types"
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

	// Function to rebuild rendering text with cursor block highlight
	renderContent := func() {
		text := ed.GetText()
		cursor := ed.GetCursor()
		filePath := ed.GetFilePath()
		if filePath == "" {
			filePath = "[No Name]"
		}

		editorView.SetTitle(fmt.Sprintf(" Pieces Store TUI Editor — %s ", filePath))

		lines := strings.Split(text, "\n")
		var formattedLines []string

		for i, line := range lines {
			if i == cursor.Row {
				runes := []rune(line)
				var builder strings.Builder
				for j, r := range runes {
					if j == cursor.Col {
						// Apply Tokyo Night cursor block styling: dark text (#1a1b26) on bright blue background (#7aa2f7)
						// Then reset text style back to normal (#a9b1d6) on dark background (#1a1b26)
						builder.WriteString("[#1a1b26:#7aa2f7]")
						builder.WriteRune(r)
						builder.WriteString("[#a9b1d6:#1a1b26]")
					} else {
						builder.WriteRune(r)
					}
				}
				// If cursor is at the end of the line (append position)
				if cursor.Col >= len(runes) {
					builder.WriteString("[#1a1b26:#7aa2f7] [#a9b1d6:#1a1b26]")
				}
				formattedLines = append(formattedLines, builder.String())
			} else {
				formattedLines = append(formattedLines, line)
			}
		}

		editorView.SetText(strings.Join(formattedLines, "\n"))

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
