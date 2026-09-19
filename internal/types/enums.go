package types

import (
	"path/filepath"
	"strings"
)

type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeVisual
	ModeCommand
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeVisual:
		return "VISUAL"
	case ModeCommand:
		return "COMMAND"
	default:
		return "UNKNOWN"
	}
}

// Style is the semantic category of a styled range of text — what KIND of
// thing it is (a keyword, a string, an error), not what color it should
// render as. Each frontend owns its own Style -> color mapping, the same
// way each already maps Mode to its own status-bar color — this stays
// purely semantic and themeable, never a raw color itself.
//
// Deliberately a small starter set, not an attempt to enumerate every
// treesitter capture name up front — same "add a table entry" extension
// philosophy as the rest of this codebase. StyleNone (the zero value)
// means "no style applies here," i.e. plain text.
type Style int

const (
	StyleNone Style = iota
	StyleKeyword
	StyleString
	StyleComment
	StyleNumber
	StyleFunction
	StyleType
	StyleDiagnosticError
	StyleDiagnosticWarning
)

// Language identifies which language a buffer's content is written in —
// used to pick a treesitter grammar and, eventually, an LSP languageId.
// Detected from the file extension only (see DetectLanguage); no content
// sniffing, shebang detection, or special-cased filenames (Makefile,
// Dockerfile, etc.) yet — deliberately a small starter set, same
// "add a table entry" extension philosophy as Style. LanguagePlainText
// (the zero value) is both the fallback for an unrecognized extension and
// the correct answer for an unsaved buffer with no file path at all.
type Language int

const (
	LanguagePlainText Language = iota
	LanguageMarkdown
	LanguageHTML
	LanguageGo
	LanguageJSON
	LanguageJavaScript
	LanguageCSS
)

func (l Language) String() string {
	switch l {
	case LanguageMarkdown:
		return "Markdown"
	case LanguageHTML:
		return "HTML"
	case LanguageGo:
		return "Go"
	case LanguageJSON:
		return "JSON"
	case LanguageJavaScript:
		return "JavaScript"
	case LanguageCSS:
		return "CSS"
	default:
		return "Plain Text"
	}
}

// DetectLanguage maps a file path's extension to a Language —
// case-insensitive, and the only signal used today (see Language's doc
// comment for what's deliberately not attempted yet). An empty path or
// an unrecognized extension both correctly fall through to
// LanguagePlainText.
func DetectLanguage(filePath string) Language {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".md", ".markdown":
		return LanguageMarkdown
	case ".html", ".htm":
		return LanguageHTML
	case ".go":
		return LanguageGo
	case ".json":
		return LanguageJSON
	case ".js":
		return LanguageJavaScript
	case ".css":
		return LanguageCSS
	default:
		return LanguagePlainText
	}
}
