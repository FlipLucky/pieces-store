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
		{"app.mjs", LanguageJavaScript},
		{"app.cjs", LanguageJavaScript},
		{"app.ts", LanguageTypeScript},
		{"component.tsx", LanguageTSX},
		{"style.css", LanguageCSS},
		{"style.scss", LanguageSCSS},
		{"index.php", LanguagePHP},
		{"main.dart", LanguageDart},
		{"lib.c", LanguageC},
		{"lib.h", LanguageC},
		{"lib.cpp", LanguageCPP},
		{"lib.hpp", LanguageCPP},
		{"config.yaml", LanguageYAML},
		{"config.yml", LanguageYAML},
		{"Dockerfile", LanguageDockerfile},        // no extension at all — matched by base name
		{"dockerfile", LanguageDockerfile},        // case-insensitive
		{"Dockerfile.dev", LanguageDockerfile},    // multi-stage/variant naming convention
		{"", LanguagePlainText},                   // unsaved buffer, no path at all
		{"Makefile", LanguagePlainText},           // no extension, not a special-cased name
		{"archive.tar.gz", LanguagePlainText},     // unrecognized extension
		{"/a/b/c/README.md", LanguageMarkdown},    // extension detection ignores directory components
		{"/a/b/c/Dockerfile", LanguageDockerfile}, // base-name detection also ignores directory components
	}
	for _, c := range cases {
		if got := DetectLanguage(c.path); got != c.want {
			t.Errorf("DetectLanguage(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}
