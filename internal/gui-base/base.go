package guibase

import (
	"fmt"
	"image/color"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget/material"

	"github.com/fliplucky/pieces-store/internal/editor"
	"github.com/fliplucky/pieces-store/internal/types"
	"github.com/fliplucky/pieces-store/internal/viewmanager"
)

// viewportMargin is how many extra lines above/below the visible window get
// fetched, so a fast scroll doesn't have to wait on a fresh Viewport call.
const viewportMargin = 10

// maxCompletionRows caps how many :e/:w completion candidates the popup
// shows at once — see completionWindow.
const maxCompletionRows = 8

type GuiApp interface {
	Run()
}

type GioApp struct {
	editor *editor.Editor
	theme  *material.Theme
	tag    *int
	// topRow is the first document row currently rendered — scroll state
	// lives here (per-window), not on Editor, same reasoning as
	// viewmanager owning view state generally.
	topRow int
}

func CreateApp(ed *editor.Editor) GuiApp {
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	// Tokyo Night Theme Colors
	theme.Palette.Bg = color.NRGBA{R: 0x1a, G: 0x1b, B: 0x26, A: 0xff}         // Deep dark blue-gray
	theme.Palette.Fg = color.NRGBA{R: 0xa9, G: 0xb1, B: 0xd6, A: 0xff}         // Light silver-blue text
	theme.Palette.ContrastBg = color.NRGBA{R: 0x7a, G: 0xa2, B: 0xf7, A: 0xff} // Tokyo Night bright blue (cursor)
	theme.Palette.ContrastFg = color.NRGBA{R: 0x1a, G: 0x1b, B: 0x26, A: 0xff} // Deep dark text for cursor overlay

	return &GioApp{
		editor: ed,
		theme:  theme,
		tag:    new(int),
	}
}

func (g *GioApp) Run() {
	go func() {
		window := new(app.Window)
		window.Option(app.Title("GUI Text Editor"))

		// Monitor editor updates and trigger frame invalidations
		go func() {
			for range g.editor.ChangeChan() {
				window.Invalidate()
			}
		}()
		var ops op.Ops
		for {
			switch e := window.Event().(type) {
			case app.DestroyEvent:
				return
			case app.FrameEvent:
				ops.Reset()
				gtx := app.NewContext(&ops, e)

				// Declare the input area over the entire window and paint
				// the background.
				clipStack := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
				event.Op(gtx.Ops, g.tag)

				// The focus request must come *after* event.Op has
				// registered the tag in this frame's ops — requesting it
				// any earlier gets silently dropped, since the tag isn't
				// yet a valid focus target for this frame.
				if !gtx.Focused(g.tag) {
					gtx.Execute(key.FocusCmd{Tag: g.tag})
				}

				paint.ColorOp{Color: g.theme.Palette.Bg}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)

				g.handleEvents(gtx)

				if g.editor.IsQuitRequested() {
					window.Perform(system.ActionClose)
				}

				cursor := g.editor.GetCursor()
				lineHeight := lineHeightPx(gtx, g.theme)
				visibleRows := visibleRowCount(gtx, lineHeight)
				g.topRow = followCursor(g.topRow, cursor.Row, visibleRows)
				slice := g.editor.Viewport(g.topRow, visibleRows, viewportMargin)
				spans := g.editor.StyleSpans(slice.StartOffset, slice.EndOffset)

				layout.Flex{
					Axis:      layout.Vertical,
					Alignment: layout.Start,
				}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return renderBody(gtx, g.theme, slice, spans, cursor)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return renderCompletionPopup(gtx, g.theme, cursor)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return renderStatusBar(gtx, g.theme, g.editor, cursor)
					}),
				)

				clipStack.Pop()
				e.Frame(gtx.Ops)
			}
		}
	}()

	app.Main()
}

// handleEvents drains this frame's key events and forwards each one to
// Editor.HandleKey (or InsertLiteralText for multi-rune paste/IME chunks) —
// the frontend does no vim-grammar interpretation of its own, matching
// tui-base's single ed.HandleKey(keyStr) call site. A physical keystroke
// arrives as either a key.EditEvent (its produced text) or a key.Event (its
// name/modifiers), never meaningfully both: printable keys are handled via
// EditEvent's text, and key.Event is only translated for the non-printable
// keys EditEvent never carries text for (Escape, Enter, Backspace, Ctrl
// combos) — so nothing here double-dispatches a single keystroke.
func (g *GioApp) handleEvents(gtx layout.Context) {
	for {
		// key.FocusFilter is what makes g.tag "focusable" at all — without
		// it, the router strips focus from the tag every single frame (it
		// only keeps a focus target that some registered filter actually
		// claims), so key.FocusCmd never sticks and key.EditEvent (which
		// is only ever routed to the currently-focused tag) never arrives.
		// key.Filter{} separately picks up named/control key.Events, which
		// don't require focus at all — but only events with NO modifiers,
		// since a filter's Required/Optional default to zero and
		// keyFilterMatch rejects any modifier bit not covered by one of
		// them (`e.Modifiers &^ (Required|Optional) != 0`). Declaring
		// Optional: key.ModCtrl is what lets Ctrl-combos (<C-r>, <C-n>,
		// <C-p>) through at all — without it they're silently dropped
		// before handleEvents ever sees them, same failure shape as bare
		// Tab below. Bare Tab is a further special case on top of that:
		// Gio reserves it as a system-level "move focus to the next
		// widget" key and won't deliver it to a wildcard key.Filter{} at
		// all (see keyFilterMatch's "system" flag) — it has to be claimed
		// explicitly by name, or it's silently consumed for focus-cycling
		// instead of reaching us (as it was doing for :e/:w completion).
		ev, ok := gtx.Event(
			key.FocusFilter{Target: g.tag},
			key.Filter{Optional: key.ModCtrl},
			key.Filter{Name: key.NameTab},
		)
		if !ok {
			break
		}
		switch ev := ev.(type) {
		case key.EditEvent:
			runes := []rune(ev.Text)
			switch len(runes) {
			case 0:
			case 1:
				g.editor.HandleKey(string(runes[0]))
			default:
				g.editor.InsertLiteralText(ev.Text)
			}
		case key.Event:
			if keyStr, ok := translateKeyEvent(ev); ok {
				g.editor.HandleKey(keyStr)
			}
		}
	}
}

// translateKeyEvent maps a named/control key.Event to the engine's key
// vocabulary. Plain printable keys are deliberately not handled here — they
// arrive via key.EditEvent instead (see handleEvents).
func translateKeyEvent(e key.Event) (string, bool) {
	if e.State != key.Press {
		return "", false
	}
	switch e.Name {
	case key.NameEscape:
		return "<Esc>", true
	case key.NameReturn, key.NameEnter:
		return "<Enter>", true
	case key.NameDeleteBackward:
		return "<BS>", true
	case key.NameTab:
		return "<Tab>", true
	}
	if e.Modifiers.Contain(key.ModCtrl) {
		switch e.Name {
		case "R":
			return "<C-r>", true
		case "N":
			return "<C-n>", true
		case "P":
			return "<C-p>", true
		}
	}
	return "", false
}

// lineHeightPx estimates a line's pixel height from the theme's text size
// (size + a leading factor) — approximate, but good enough to size the
// viewport window; exact text-shaper metrics can replace this later.
func lineHeightPx(gtx layout.Context, theme *material.Theme) int {
	h := int(float32(gtx.Sp(theme.TextSize)) * 1.2)
	if h < 1 {
		h = 1
	}
	return h
}

// visibleRowCount is how many text rows fit above the (single-line) status
// bar, given the current window height.
func visibleRowCount(gtx layout.Context, lineHeight int) int {
	available := gtx.Constraints.Max.Y - lineHeight
	rows := available / lineHeight
	if rows < 1 {
		rows = 1
	}
	return rows
}

// followCursor keeps the cursor's row within the visible window, scrolling
// only the minimum amount necessary rather than re-centering every frame.
func followCursor(topRow, cursorRow, visibleRows int) int {
	if cursorRow < topRow {
		return cursorRow
	}
	if cursorRow >= topRow+visibleRows {
		return cursorRow - visibleRows + 1
	}
	return topRow
}

// renderBody lays out exactly the lines in slice — never the whole
// document — one Flex row per line.
func renderBody(gtx layout.Context, theme *material.Theme, slice viewmanager.Slice, spans []viewmanager.StyledSpan, cursor editor.Cursor) layout.Dimensions {
	children := make([]layout.FlexChild, len(slice.Lines))
	for i, line := range slice.Lines {
		absRow := slice.StartRow + i
		line := line
		lineStart := slice.LineOffsets[i]
		children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return renderLine(gtx, theme, line, absRow, lineStart, spans, cursor)
		})
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func renderLine(gtx layout.Context, theme *material.Theme, line string, absRow, lineStart int, spans []viewmanager.StyledSpan, cursor editor.Cursor) layout.Dimensions {
	isCursorLine := absRow == cursor.Row
	lineSpans := viewmanager.SpansForRange(spans, lineStart, lineStart+len(line))

	if !isCursorLine && len(lineSpans) == 0 {
		lbl := material.Label(theme, theme.TextSize, line)
		lbl.Color = theme.Palette.Fg
		return lbl.Layout(gtx)
	}

	// This line needs per-rune treatment — either it's the cursor's line
	// (exactly one cell styled as the cursor) or it has styled spans (or
	// both). Lay out rune by rune either way.
	runes := []rune(line)
	count := len(runes)
	if isCursorLine && cursor.Col >= count {
		count++ // trailing cursor position past the last rune
	}

	// Precompute each rune's byte offset within the line up front rather
	// than accumulating inside the layout.List callback below — the
	// callback's invocation order isn't something to rely on.
	byteOffsets := make([]int, len(runes))
	acc := 0
	for i, r := range runes {
		byteOffsets[i] = acc
		acc += len(string(r))
	}

	var charList layout.List
	charList.Axis = layout.Horizontal
	return charList.Layout(gtx, count, func(gtx layout.Context, charIndex int) layout.Dimensions {
		var charStr string
		if charIndex < len(runes) {
			charStr = string(runes[charIndex])
		} else {
			charStr = " "
		}

		if isCursorLine && charIndex == cursor.Col {
			return renderCursorCell(gtx, theme, charStr, cursor.Mode)
		}

		style := types.StyleNone
		if charIndex < len(runes) {
			style = viewmanager.StyleAt(lineSpans, lineStart+byteOffsets[charIndex])
		}
		lbl := material.Label(theme, theme.TextSize, charStr)
		lbl.Color = styleColor(theme, style)
		return lbl.Layout(gtx)
	})
}

// styleColor maps a semantic Style to this theme's color — each frontend
// owns this mapping itself (the same pattern renderStatusBar already uses
// for Mode's color), keeping types.Style purely semantic and themeable.
func styleColor(theme *material.Theme, style types.Style) color.NRGBA {
	switch style {
	case types.StyleKeyword:
		return color.NRGBA{R: 0xbb, G: 0x9a, B: 0xf7, A: 0xff} // Purple
	case types.StyleString:
		return color.NRGBA{R: 0x9e, G: 0xce, B: 0x6a, A: 0xff} // Green
	case types.StyleComment:
		return color.NRGBA{R: 0x56, G: 0x5f, B: 0x89, A: 0xff} // Muted gray-blue
	case types.StyleNumber:
		return color.NRGBA{R: 0xff, G: 0x9e, B: 0x64, A: 0xff} // Orange
	case types.StyleFunction:
		return color.NRGBA{R: 0x7a, G: 0xa2, B: 0xf7, A: 0xff} // Blue
	case types.StyleType:
		return color.NRGBA{R: 0x2a, G: 0xc3, B: 0xde, A: 0xff} // Cyan
	case types.StyleDiagnosticError:
		return color.NRGBA{R: 0xf7, G: 0x76, B: 0x8e, A: 0xff} // Red
	case types.StyleDiagnosticWarning:
		return color.NRGBA{R: 0xe0, G: 0xaf, B: 0x68, A: 0xff} // Yellow
	default:
		return theme.Palette.Fg
	}
}

// renderCursorCell draws the cursor's own cell: a thin pipe/bar in Insert
// mode (matches vim's insertion-point caret), a solid block otherwise
// (Normal/Visual/Command).
func renderCursorCell(gtx layout.Context, theme *material.Theme, charStr string, mode types.Mode) layout.Dimensions {
	if mode == types.ModeInsert {
		return layout.Stack{}.Layout(gtx,
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(theme, theme.TextSize, charStr)
				lbl.Color = theme.Palette.Fg
				return lbl.Layout(gtx)
			}),
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				bar := gtx.Constraints.Min
				bar.X = gtx.Dp(2)
				defer clip.Rect{Max: bar}.Push(gtx.Ops).Pop()
				paint.ColorOp{Color: theme.Palette.ContrastBg}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
		)
	}

	return layout.Stack{Alignment: layout.Center}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: theme.Palette.ContrastBg}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(theme, theme.TextSize, charStr)
			lbl.Color = theme.Palette.ContrastFg
			return lbl.Layout(gtx)
		}),
	)
}

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

// renderCompletionPopup draws the :e/:w candidate list — vim's wildmenu,
// essentially — above the status bar while a completion cycle
// (Editor.TriggerCompletion) is active. Zero-size when it isn't, so it
// takes no layout space.
func renderCompletionPopup(gtx layout.Context, theme *material.Theme, cursor editor.Cursor) layout.Dimensions {
	completion := cursor.Completion
	if !completion.Active || len(completion.Candidates) == 0 {
		return layout.Dimensions{}
	}
	windowed, selected := completionWindow(completion.Candidates, completion.Index, maxCompletionRows)

	children := make([]layout.FlexChild, len(windowed))
	for i, candidate := range windowed {
		candidate := candidate
		isSelected := i == selected
		children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !isSelected {
				lbl := material.Label(theme, theme.TextSize, candidate)
				lbl.Color = theme.Palette.Fg
				return lbl.Layout(gtx)
			}
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
					paint.ColorOp{Color: theme.Palette.ContrastBg}.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
					return layout.Dimensions{Size: gtx.Constraints.Min}
				}),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(theme, theme.TextSize, candidate)
					lbl.Color = theme.Palette.ContrastFg
					return lbl.Layout(gtx)
				}),
			)
		})
	}

	popupBg := color.NRGBA{R: 0x24, G: 0x28, B: 0x3b, A: 0xff} // Tokyo Night darker background, matches the status bar

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: popupBg}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
}

func renderStatusBar(gtx layout.Context, theme *material.Theme, ed *editor.Editor, cursor editor.Cursor) layout.Dimensions {
	statusBarBg := color.NRGBA{R: 0x24, G: 0x28, B: 0x3b, A: 0xff} // Tokyo Night darker status background

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
			paint.ColorOp{Color: statusBarBg}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if cursor.Mode == types.ModeCommand {
				lbl := material.Label(theme, theme.TextSize, fmt.Sprintf(" :%s", cursor.CommandBuffer))
				lbl.Color = theme.Palette.ContrastBg
				return lbl.Layout(gtx)
			}

			var modeColor color.NRGBA
			switch cursor.Mode {
			case types.ModeNormal:
				modeColor = color.NRGBA{R: 0xe0, G: 0xaf, B: 0x68, A: 0xff} // Yellow
			case types.ModeInsert:
				modeColor = color.NRGBA{R: 0x9e, G: 0xce, B: 0x6a, A: 0xff} // Green
			case types.ModeVisual:
				modeColor = color.NRGBA{R: 0xbb, G: 0x9a, B: 0xf7, A: 0xff} // Purple
			default:
				modeColor = theme.Palette.Fg
			}

			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(theme, theme.TextSize, fmt.Sprintf(" -- %s -- ", cursor.Mode.String()))
					lbl.Color = modeColor
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(theme, theme.TextSize, fmt.Sprintf(" | Row: %d  Col: %d | Offset: %d ", cursor.Row, cursor.Col, cursor.ByteOffset))
					lbl.Color = theme.Palette.Fg
					return lbl.Layout(gtx)
				}),
			}
			// Surface why a command like :q was refused (e.g. unsaved
			// changes) — ErrQuit itself isn't an error worth showing, the
			// window is about to close.
			if err := ed.GetLastCommandError(); err != nil && err != editor.ErrQuit {
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(theme, theme.TextSize, fmt.Sprintf(" | %s", err))
					lbl.Color = color.NRGBA{R: 0xf7, G: 0x76, B: 0x8e, A: 0xff} // red
					return lbl.Layout(gtx)
				}))
			}

			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
		}),
	)
}
