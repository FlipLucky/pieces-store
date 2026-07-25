package editor

import "unicode/utf8"

type Document interface {
	GetRuneAt(offset int) (rune, int)
	Len() int
}

type ByteDocument []byte

func (b ByteDocument) Len() int {
	return len(b)
}

func (b ByteDocument) GetRuneAt(offset int) (rune, int) {
	if offset >= len(b) || offset < 0 {
		return utf8.RuneError, 0
	}
	return utf8.DecodeRune(b[offset:])
}

