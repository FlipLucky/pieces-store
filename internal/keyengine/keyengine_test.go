package keyengine

import (
	"errors"
	"testing"

	"github.com/fliplucky/pieces-store/internal/types"
)

func mustParse(t *testing.T, cc *CommandContext, keys ...string) *ExecutableCommand {
	t.Helper()
	var cmd *ExecutableCommand
	for _, k := range keys {
		c, err := cc.ProcessCommand(k)
		if err != nil {
			t.Fatalf("ProcessCommand(%q) returned unexpected error: %v", k, err)
		}
		cmd = c
	}
	if cmd == nil {
		t.Fatalf("sequence %v did not finalize into a command", keys)
	}
	return cmd
}

func wantCommand(t *testing.T, got *ExecutableCommand, verb, modifier, noun KeyActionName) {
	t.Helper()
	if got.Verb != verb || got.Modifier != modifier || got.Noun != noun {
		t.Errorf("got %+v, want Verb=%v Modifier=%v Noun=%v", got, verb, modifier, noun)
	}
}

func TestSequences(t *testing.T) {
	cases := []struct {
		name                 string
		keys                 []string
		verb, modifier, noun KeyActionName
	}{
		{"diw", []string{"d", "i", "w"}, Delete, Inside, Word},
		{"daw", []string{"d", "a", "w"}, Delete, Around, Word},
		{"dip", []string{"d", "i", "p"}, Delete, Inside, Paragraph},
		{"dd", []string{"d", "d"}, Delete, None, LineMotion},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cc := CreateCommandContext()
			cmd := mustParse(t, cc, tc.keys...)
			wantCommand(t, cmd, tc.verb, tc.modifier, tc.noun)
		})
	}
}

func TestDirects(t *testing.T) {
	cases := []struct {
		name                 string
		key                  string
		verb, modifier, noun KeyActionName
	}{
		{"w", "w", Move, None, Word},
		{"b", "b", Move, None, WordBackward},
		{"e", "e", Move, None, WordEnd},
		{"u", "u", Undo, None, None},
		{"redo", "<C-r>", Redo, None, None},
		{"r", "r", Replace, None, None},
		{"colon", ":", EnterCommand, None, None},
		{"i", "i", EnterInsert, None, None},
		{"o", "o", OpenBelow, None, None},
		{"O", "O", OpenAbove, None, None},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cc := CreateCommandContext()
			cmd, err := cc.ProcessCommand(tc.key)
			if err != nil {
				t.Fatalf("ProcessCommand(%q) returned unexpected error: %v", tc.key, err)
			}
			if cmd == nil {
				t.Fatalf("key %q did not finalize into a command", tc.key)
			}
			wantCommand(t, cmd, tc.verb, tc.modifier, tc.noun)
		})
	}
}

func TestCountPrefix(t *testing.T) {
	t.Run("count then verb", func(t *testing.T) {
		cc := CreateCommandContext()
		cmd := mustParse(t, cc, "3", "d", "i", "w")
		if cmd.Count != 3 {
			t.Errorf("Count = %d, want 3", cmd.Count)
		}
	})

	t.Run("multi-digit count", func(t *testing.T) {
		cc := CreateCommandContext()
		cmd := mustParse(t, cc, "1", "0", "w")
		if cmd.Count != 10 {
			t.Errorf("Count = %d, want 10", cmd.Count)
		}
	})

	t.Run("count mid-sequence", func(t *testing.T) {
		cc := CreateCommandContext()
		cmd := mustParse(t, cc, "d", "3", "i", "w")
		if cmd.Count != 3 {
			t.Errorf("Count = %d, want 3 (count typed after the verb)", cmd.Count)
		}
	})

	t.Run("default count is 1", func(t *testing.T) {
		cc := CreateCommandContext()
		cmd := mustParse(t, cc, "w")
		if cmd.Count != 1 {
			t.Errorf("Count = %d, want 1", cmd.Count)
		}
	})

	t.Run("leading zero is not swallowed into count", func(t *testing.T) {
		cc := CreateCommandContext()
		// "0" alone isn't registered as anything yet (no motion wired to
		// it) — it must fizzle cleanly, not silently become count=10 the
		// way it used to.
		cmd, err := cc.ProcessCommand("0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd != nil {
			t.Fatalf("bare \"0\" produced a command %+v, want nil (fizzle)", cmd)
		}
		if cc.count != 1 || cc.countStarted {
			t.Errorf("internal count state = %d (started=%v), want reset (1, false)", cc.count, cc.countStarted)
		}
	})
}

func TestFizzleIsSilent(t *testing.T) {
	cc := CreateCommandContext()
	cmd, err := cc.ProcessCommand("z")
	if err != nil {
		t.Fatalf("unregistered key returned an error, want a silent fizzle: %v", err)
	}
	if cmd != nil {
		t.Fatalf("unregistered key produced a command: %+v", cmd)
	}
}

func TestInconsistentKeymapIsAnError(t *testing.T) {
	// Deliberately construct a whitelist/table mismatch: "d" claims "x" is
	// an allowed continuation, but "x" is registered as neither a modifier
	// nor a noun. This must surface as an error, not a silent fizzle,
	// since it's a bug in the tables rather than invalid user input.
	cc := CreateCommandContext()
	cc.normalModeVerbs["d"] = KeyAction{
		name:              Delete,
		allowedKeyActions: map[string]KeyActionName{"x": Word},
	}

	if _, err := cc.ProcessCommand("d"); err != nil {
		t.Fatalf("unexpected error entering verb: %v", err)
	}
	cmd, err := cc.ProcessCommand("x")
	if cmd != nil {
		t.Fatalf("inconsistent keymap produced a command: %+v", cmd)
	}
	if !errors.Is(err, ErrInconsistentKeymap) {
		t.Fatalf("err = %v, want wrapping ErrInconsistentKeymap", err)
	}
}

func TestVisualModeHasNoDirects(t *testing.T) {
	cc := CreateCommandContext()
	cc.SetEditorMode(types.ModeNormal)
	cmd, err := cc.ProcessCommand("w")
	if err != nil || cmd == nil {
		t.Fatalf("bare w in Normal mode should resolve to a Move, got cmd=%+v err=%v", cmd, err)
	}

	cc2 := CreateCommandContext()
	cc2.SetEditorMode(types.ModeVisual)
	cmd2, err2 := cc2.ProcessCommand("w")
	if err2 != nil {
		t.Fatalf("unexpected error in visual mode: %v", err2)
	}
	if cmd2 != nil {
		t.Fatalf("visual mode has no direct table yet, \"w\" should fizzle, got %+v", cmd2)
	}
}
