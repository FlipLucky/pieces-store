package exmode

import (
	"reflect"
	"testing"
)

func TestParseWithLeadingColon(t *testing.T) {
	got := Parse(":w somefile.go")
	want := Command{Name: "w", Args: []string{"somefile.go"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse(:w somefile.go) = %+v, want %+v", got, want)
	}
}

func TestParseWithoutLeadingColon(t *testing.T) {
	got := Parse("w somefile.go")
	want := Command{Name: "w", Args: []string{"somefile.go"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse(w somefile.go) = %+v, want %+v", got, want)
	}
}

func TestParseNoArgs(t *testing.T) {
	got := Parse(":q")
	want := Command{Name: "q", Args: []string{}}
	if got.Name != want.Name || len(got.Args) != 0 {
		t.Errorf("Parse(:q) = %+v, want %+v", got, want)
	}
}

func TestParseMultipleArgs(t *testing.T) {
	got := Parse(":LspInstall go extra")
	want := Command{Name: "LspInstall", Args: []string{"go", "extra"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse(:LspInstall go extra) = %+v, want %+v", got, want)
	}
}

func TestParseEmptyIsZeroValue(t *testing.T) {
	cases := []string{"", ":", "   ", " : "}
	for _, c := range cases {
		got := Parse(c)
		if got.Name != "" || len(got.Args) != 0 {
			t.Errorf("Parse(%q) = %+v, want the zero Command", c, got)
		}
	}
}

func TestParseTrimsWhitespace(t *testing.T) {
	got := Parse("  :w   somefile.go  ")
	want := Command{Name: "w", Args: []string{"somefile.go"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse with extra whitespace = %+v, want %+v", got, want)
	}
}

func TestParsePreservesCaseAndBangSuffix(t *testing.T) {
	// "q!" is its own distinct command name today, not "q" plus a flag —
	// confirm parsing doesn't split or normalize it.
	got := Parse(":q!")
	if got.Name != "q!" {
		t.Errorf("Parse(:q!) Name = %q, want %q", got.Name, "q!")
	}
}
