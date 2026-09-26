package clipboardregister

import (
	"slices"
	"testing"
)

func TestAppendToRegisterPrependsNewest(t *testing.T) {
	reg := Register{"old"}
	appendToRegister("new", &reg)

	want := Register{"new", "old"}
	if !slices.Equal(reg, want) {
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
			want:         Register{"fourth", "third", "second", "first"},
		},
		{
			name:         "test yank register",
			initial:      ClipboardRegister{},
			registerType: Yank,
			input:        []string{"first", "second", "third", "fourth"},
			want:         Register{"fourth", "third", "second", "first"},
		},
		{
			name:         "test delete register",
			initial:      ClipboardRegister{},
			registerType: Delete,
			input:        []string{"first", "second", "third", "fourth"},
			want:         Register{"fourth", "third", "second", "first"},
		},
	}
	for _, c := range cases {
		for _, value := range c.input {
			c.initial.StoreRegisterValue(value, c.registerType)
		}
		register := c.initial.getRegister(c.registerType)
		if !(slices.Equal(c.want, *register)) {
			t.Errorf("appendTo%v result = %v, want %v", c.registerType, register, c.want)
		}
	}

}
