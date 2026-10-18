package clipboardregister

import (
	"slices"
	"testing"
)

// entriesEqual compares two Entry values by content — Entry.Values is now
// a slice (T54, 2026-10-18, for block mode's multi-row entries), so Entry
// is no longer comparable via == / slices.Equal directly.
func entriesEqual(a, b Entry) bool {
	return a.Linewise == b.Linewise && slices.Equal(a.Values, b.Values)
}

func TestAppendToRegisterPrependsNewest(t *testing.T) {
	reg := Register{{Values: []string{"old"}}}
	appendToRegister(Entry{Values: []string{"new"}}, &reg)

	want := Register{{Values: []string{"new"}}, {Values: []string{"old"}}}
	if !slices.EqualFunc(reg, want, entriesEqual) {
		t.Errorf("appendToRegister result = %v, want %v", reg, want)
	}
}

func TestAppendToVisualMode(t *testing.T) {
	cases := []struct {
		name         string
		initial      ClipboardRegister
		registerType RegisterType
		input        []string
		want         Register
	}{
		{
			name:         "test visual register",
			initial:      ClipboardRegister{},
			registerType: Visual,
			input:        []string{"first", "second", "third", "fourth"},
			want:         Register{{Values: []string{"fourth"}}, {Values: []string{"third"}}, {Values: []string{"second"}}, {Values: []string{"first"}}},
		},
		{
			name:         "test yank register",
			initial:      ClipboardRegister{},
			registerType: Yank,
			input:        []string{"first", "second", "third", "fourth"},
			want:         Register{{Values: []string{"fourth"}}, {Values: []string{"third"}}, {Values: []string{"second"}}, {Values: []string{"first"}}},
		},
		{
			name:         "test delete register",
			initial:      ClipboardRegister{},
			registerType: Delete,
			input:        []string{"first", "second", "third", "fourth"},
			want:         Register{{Values: []string{"fourth"}}, {Values: []string{"third"}}, {Values: []string{"second"}}, {Values: []string{"first"}}},
		},
	}
	for _, c := range cases {
		for _, value := range c.input {
			c.initial.StoreRegisterValue(value, c.registerType, false, 0)
		}
		register := c.initial.getRegister(c.registerType)
		if !slices.EqualFunc(c.want, *register, entriesEqual) {
			t.Errorf("appendTo%v result = %v, want %v", c.registerType, register, c.want)
		}
	}

}
