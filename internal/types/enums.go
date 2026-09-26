package types

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
// used to pick a treesitter grammar and an LSP languageId. Detected from
// the file extension only (see internal/langdetect.Detect — deliberately
// not a method here, since types is meant to stay a logic-free enum
// package); no content sniffing, shebang detection, or special-cased
// filenames (Makefile, Dockerfile itself being the one deliberate
// exception, since it has no extension at all) yet. This is the curated,
// deliberately-bounded
// language set decided 2026-09-19 — the ones the project actually needs,
// not an attempt at broad coverage; extending it is still just "add a
// table entry" (here, and in internal/syntax's grammar-name mapping).
// LanguagePlainText (the zero value) is both the fallback for an
// unrecognized extension and the correct answer for an unsaved buffer
// with no file path at all.
type Language int

const (
	LanguagePlainText Language = iota
	LanguageMarkdown
	LanguageHTML
	LanguageGo
	LanguageJSON
	LanguageJavaScript
	LanguageTypeScript
	LanguageTSX
	LanguageCSS
	LanguageSCSS
	LanguagePHP
	LanguageDart
	LanguageC
	LanguageCPP
	LanguageYAML
	LanguageDockerfile
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
	case LanguageTypeScript:
		return "TypeScript"
	case LanguageTSX:
		return "TSX"
	case LanguageCSS:
		return "CSS"
	case LanguageSCSS:
		return "SCSS"
	case LanguagePHP:
		return "PHP"
	case LanguageDart:
		return "Dart"
	case LanguageC:
		return "C"
	case LanguageCPP:
		return "C++"
	case LanguageYAML:
		return "YAML"
	case LanguageDockerfile:
		return "Dockerfile"
	default:
		return "Plain Text"
	}
}
