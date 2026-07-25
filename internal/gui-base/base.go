package guibase

import (
	"fmt"
	"image/color"
	"log"
	"strings"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget/material"

	"github.com/fliplucky/pieces-store/internal/editor"
)

type GuiApp interface {
	Run()
}

type GioApp struct {
	editor  *editor.Editor
	theme   *material.Theme
	tag     *int
	focused bool
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
		editor:  ed,
		theme:   theme,
		tag:     new(int),
		focused: false,
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

				if !g.focused {
					fmt.Printf("focussed was false, focussing")
					gtx.Execute(key.FocusCmd{Tag: g.tag})
					g.focused = true
				}
				// 1. Declare target input area over the entire window constraints
				// Paint the entire window background and register input area inside the clip stack
				clipStack := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
				event.Op(gtx.Ops, g.tag)

				gtx.Execute(key.FocusCmd{Tag: g.tag})

				paint.ColorOp{Color: g.theme.Palette.Bg}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)

				focused := gtx.Focused(g.tag)
				log.Printf("[DEBUG] GUI Frame: tag=%p focused=%t", g.tag, focused)

				// 2. Query event queue using key.Filter
				for {
					// ev, ok := gtx.Event(key.Filter{Focus: g.tag})
					ev, ok := gtx.Event(key.Filter{})
					if !ok {
						break
					}
					log.Printf("[DEBUG] GUI Event retrieved: %#v", ev)
					log.Printf("[DEBUG] Raw Event received: %T, Value: %#v", ev, ev)
					switch e := ev.(type) {
					case key.EditEvent:
						cursor := g.editor.GetCursor()
						if cursor.Mode == editor.ModeInsert {
							g.editor.InsertText([]byte(e.Text))
						}
					case key.Event:
						if e.State == key.Press {
							cursor := g.editor.GetCursor()
							nameLower := strings.ToLower(string(e.Name))

							if cursor.Mode == editor.ModeNormal {
								switch nameLower {
								case "h":
									g.editor.MoveCursorLeft()
								case "l":
									g.editor.MoveCursorRight()
								case "k":
									g.editor.MoveCursorUp()
								case "j":
									g.editor.MoveCursorDown()
								case "i":
									g.editor.SetMode(editor.ModeInsert)
								case "x":
									g.editor.DeleteText()
								}
							} else if cursor.Mode == editor.ModeInsert {
								switch e.Name {
								case key.NameEscape:
									g.editor.SetMode(editor.ModeNormal)
								case key.NameEnter, key.NameReturn:
									g.editor.InsertText([]byte("\n"))
								case key.NameDeleteBackward, "Backspace":
									g.editor.DeleteText()
								}
							}
						}
					}
				}

				// 3. Render Content and Layout
				currentText := g.editor.GetText()
				cursor := g.editor.GetCursor()

				layout.Flex{
					Axis:      layout.Vertical,
					Alignment: layout.Start,
				}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						// Split editor text into lines
						lines := strings.Split(currentText, "\n")

						// Render lines inside a vertical list
						var list layout.List
						list.Axis = layout.Vertical

						return list.Layout(gtx, len(lines), func(gtx layout.Context, index int) layout.Dimensions {
							line := lines[index]

							if index != cursor.Row {
								// Regular line (Tokyo Night silver-blue foreground)
								lbl := material.Label(g.theme, g.theme.TextSize, line)
								lbl.Color = g.theme.Palette.Fg
								return lbl.Layout(gtx)
							}

							// Line with active cursor: layout rune by rune
							runes := []rune(line)

							var charList layout.List
							charList.Axis = layout.Horizontal

							count := len(runes)
							if cursor.Col >= len(runes) {
								count++
							}

							return charList.Layout(gtx, count, func(gtx layout.Context, charIndex int) layout.Dimensions {
								var charStr string
								isCursor := charIndex == cursor.Col

								if charIndex < len(runes) {
									charStr = string(runes[charIndex])
								} else {
									charStr = " " // cursor trailing position
								}

								if !isCursor {
									lbl := material.Label(g.theme, g.theme.TextSize, charStr)
									lbl.Color = g.theme.Palette.Fg
									return lbl.Layout(gtx)
								}

								// Draw solid cursor block beneath the character
								return layout.Stack{Alignment: layout.Center}.Layout(gtx,
									layout.Expanded(func(gtx layout.Context) layout.Dimensions {
										defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
										paint.ColorOp{Color: g.theme.Palette.ContrastBg}.Add(gtx.Ops)
										paint.PaintOp{}.Add(gtx.Ops)
										return layout.Dimensions{Size: gtx.Constraints.Min}
									}),
									layout.Stacked(func(gtx layout.Context) layout.Dimensions {
										lbl := material.Label(g.theme, g.theme.TextSize, charStr)
										lbl.Color = g.theme.Palette.ContrastFg
										return lbl.Layout(gtx)
									}),
								)
							})
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						// Render bottom Vim status bar
						statusBarBg := color.NRGBA{R: 0x24, G: 0x28, B: 0x3b, A: 0xff} // Tokyo Night darker status background

						var modeColor color.NRGBA
						switch cursor.Mode {
						case editor.ModeNormal:
							modeColor = color.NRGBA{R: 0xe0, G: 0xaf, B: 0x68, A: 0xff} // Yellow
						case editor.ModeInsert:
							modeColor = color.NRGBA{R: 0x9e, G: 0xce, B: 0x6a, A: 0xff} // Green
						case editor.ModeVisual:
							modeColor = color.NRGBA{R: 0xbb, G: 0x9a, B: 0xf7, A: 0xff} // Purple
						default:
							modeColor = g.theme.Palette.Fg
						}

						return layout.Stack{}.Layout(gtx,
							layout.Expanded(func(gtx layout.Context) layout.Dimensions {
								defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
								paint.ColorOp{Color: statusBarBg}.Add(gtx.Ops)
								paint.PaintOp{}.Add(gtx.Ops)
								return layout.Dimensions{Size: gtx.Constraints.Min}
							}),
							layout.Stacked(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										lbl := material.Label(g.theme, g.theme.TextSize, fmt.Sprintf(" -- %s -- ", cursor.Mode.String()))
										lbl.Color = modeColor
										return lbl.Layout(gtx)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										lbl := material.Label(g.theme, g.theme.TextSize, fmt.Sprintf(" | Row: %d  Col: %d | Offset: %d ", cursor.Row, cursor.Col, cursor.ByteOffset))
										lbl.Color = g.theme.Palette.Fg
										return lbl.Layout(gtx)
									}),
								)
							}),
						)
					}),
				)

				// Pop the main window clipStack
				clipStack.Pop()

				e.Frame(gtx.Ops)
			}
		}
	}()

	app.Main()
}
