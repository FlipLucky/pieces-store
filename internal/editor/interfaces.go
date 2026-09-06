package editor

type Document interface {
	GetRuneAt(offset int) (rune, int)
	Len() int
}
