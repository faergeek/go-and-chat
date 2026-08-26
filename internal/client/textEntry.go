package client

import (
	"slices"
	"unicode"

	"github.com/faergeek/go-and-chat/internal/terminal"
)

type textEntry struct {
	Runes  []rune
	Cursor int
}

func newTextEntry() textEntry {
	return textEntry{Runes: []rune{}}
}

func (t *textEntry) backspace() {
	if t.Cursor > 0 {
		t.Runes = slices.Delete(t.Runes, t.Cursor-1, t.Cursor)
		t.Cursor--
	}
}

func (t *textEntry) deleteCharacter() {
	if t.Cursor < len(t.Runes) {
		t.Runes = slices.Delete(t.Runes, t.Cursor, t.Cursor+1)
	}
}

func (t *textEntry) deleteWord() {
	i := t.Cursor - 1

	for ; i > -1; i-- {
		if !unicode.IsSpace(t.Runes[i]) {
			break
		}
	}

	for ; i > -1; i-- {
		if unicode.IsSpace(t.Runes[i]) {
			break
		}
	}

	i++

	t.Runes = slices.Delete(t.Runes, i, t.Cursor)
	t.Cursor = i
}

func (t *textEntry) deleteToTheLeft() {
	t.Runes = slices.Delete(t.Runes, 0, t.Cursor)
	t.Cursor = 0
}

func (t *textEntry) moveCursorTo(n int) {
	t.Cursor = max(0, min(n, len(t.Runes)))
}

func (t *textEntry) clear() {
	t.Runes = []rune{}
	t.Cursor = 0
}

func (t *textEntry) handleInput(input *terminal.Input) {
	switch input.Kind {
	case terminal.InputKindPrint:
		for _, r := range input.Str {
			t.Runes = slices.Insert(t.Runes, t.Cursor, r)
			t.Cursor++
		}
	case terminal.InputKindControl:
		switch input.Str {
		case "\x01", "\x1b[H":
			t.moveCursorTo(t.Cursor - 1)
		case "\x06", "\x1b[C":
			t.moveCursorTo(t.Cursor + 1)
		case "\x05", "\x1b[F":
			t.moveCursorTo(0)
		case "\x02", "\x1b[D":
			t.moveCursorTo(len(t.Runes))
		case "\x15":
			t.deleteToTheLeft()
		case "\x17":
			t.deleteWord()
		case "\x7f":
			t.backspace()
		case "\x1b[3~":
			t.deleteCharacter()
		}
	}
}
