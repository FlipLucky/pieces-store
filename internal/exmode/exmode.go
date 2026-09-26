// Package exmode parses command-mode input (":w somefile", ":LspInstall
// go") into a resolved Command — vim's own term for these is "Ex commands,"
// from the : prompt inherited from the ex editor, hence the package name.
//
// Deliberately not an extension of internal/keyengine, even though the
// idea is the same ("keystrokes/text in, a resolved command out, never
// acted on here"): keyengine's ExecutableCommand{Count,Verb,Modifier,Noun}
// shape is built specifically for vim's verb/modifier/noun grammar, and
// command-mode text isn't that grammar at all — it's flat, space-separated
// tokens, closer to a shell command line. Forcing it into keyengine's types
// would be a conceptual mismatch, not a simplification. See BACKLOG.md's
// 2026-09-24 architecture review for the reasoning.
//
// Like keyengine, this package only ever resolves — it has zero dependency
// on piecetable, lspclient, or Editor, and never mutates anything. Editor's
// own command dispatcher (executeCommandLocked) does exactly what
// dispatch.go's execute() already does for keyengine's output: take the
// resolved Command, call the right Editor method.
package exmode

import "strings"

// Command is one parsed command-mode invocation — Name is the first token
// (case-sensitive; ":LspInstall" and ":lspinstall" are different commands
// today, matching the existing convention), Args is everything after it.
// The zero value (Name == "") means "nothing was typed" — e.g. bare ":"
// followed immediately by Enter — which callers should treat as a no-op,
// not an unknown-command error.
type Command struct {
	Name string
	Args []string
}

// Parse splits raw command-mode text into a Command. Leading/trailing
// whitespace and a leading ":" (if present — callers may pass either
// "w file" or ":w file") are stripped before splitting on whitespace.
func Parse(raw string) Command {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, ":")
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return Command{}
	}
	return Command{Name: parts[0], Args: parts[1:]}
}
