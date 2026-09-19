package types

import "testing"

func TestDetectLanguage(t *testing.T) {
	cases := []struct {
		path string
		want Language
	}{
		{"README.md", LanguageMarkdown},
		{"notes.markdown", LanguageMarkdown},
		{"NOTES.MD", LanguageMarkdown}, // case-insensitive
		{"index.html", LanguageHTML},
		{"page.htm", LanguageHTML},
		{"main.go", LanguageGo},
		{"data.json", LanguageJSON},
		{"app.js", LanguageJavaScript},
		{"style.css", LanguageCSS},
		{"", LanguagePlainText},                // unsaved buffer, no path at all
		{"Makefile", LanguagePlainText},        // no extension
		{"archive.tar.gz", LanguagePlainText},  // unrecognized extension
		{"/a/b/c/README.md", LanguageMarkdown}, // extension detection ignores directory components
	}
	for _, c := range cases {
		if got := DetectLanguage(c.path); got != c.want {
			t.Errorf("DetectLanguage(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}
