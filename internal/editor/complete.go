package editor

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// completionCandidates splits buffer at its last whitespace-separated
// token — the same "the last token might be a path" rule every
// command-mode command that takes one (:e, :w) shares — and returns
// everything before that token (lineHead, kept as-is) alongside the
// sorted list of filesystem entries matching it. No token to complete
// (no space in buffer at all, e.g. the command name itself) yields no
// candidates, so ":q<Tab>" can't turn into a file listing.
func completionCandidates(buffer string) (lineHead string, candidates []string) {
	idx := strings.LastIndexByte(buffer, ' ')
	if idx < 0 {
		return "", nil
	}
	lineHead, fragment := buffer[:idx+1], buffer[idx+1:]

	dir, namePrefix := filepath.Split(fragment)
	searchDir := dir
	if searchDir == "" {
		searchDir = "."
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return lineHead, nil
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), namePrefix) {
			continue
		}
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		candidates = append(candidates, dir+name)
	}
	sort.Strings(candidates)
	return lineHead, candidates
}
