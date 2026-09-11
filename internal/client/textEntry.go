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

func (t *textEntry) moveWordBackward() {
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

	t.Cursor = i
}

func (t *textEntry) moveWordForward() {
	i := t.Cursor + 1

	for ; i < len(t.Runes); i++ {
		if !unicode.IsSpace(t.Runes[i]) {
			break
		}
	}

	for ; i < len(t.Runes); i++ {
		if unicode.IsSpace(t.Runes[i]) {
			break
		}
	}

	i--

	t.Cursor = i
}

func (t *textEntry) clear() {
	t.Runes = []rune{}
	t.Cursor = 0
}

func (t *textEntry) handleInput(input terminal.Input) {
	switch input := input.(type) {
	case terminal.InputKeyboard:
		if input.Str != "" {
			t.Runes = slices.Insert(t.Runes, t.Cursor, []rune(input.Str)...)
			t.Cursor++
		} else {
			switch {
			case input.Mod == terminal.Ctrl && input.Code == terminal.KeyArrowLeft ||
				input.Mod == terminal.Alt && input.Code == 'b':
				t.moveWordBackward()
			case input.Mod == terminal.Ctrl && input.Code == terminal.KeyArrowRight ||
				input.Mod == terminal.Alt && input.Code == 'f':
				t.moveWordForward()
			case input.Code == terminal.KeyArrowLeft ||
				input.Mod == terminal.Ctrl && input.Code == 'b':
				t.moveCursorTo(t.Cursor - 1)
			case input.Code == terminal.KeyArrowRight ||
				input.Mod == terminal.Ctrl && input.Code == 'f':
				t.moveCursorTo(t.Cursor + 1)
			case input.Code == terminal.KeyHome ||
				input.Mod == terminal.Ctrl && input.Code == 'a':
				t.moveCursorTo(0)
			case input.Code == terminal.KeyEnd ||
				input.Mod == terminal.Ctrl && input.Code == 'e':
				t.moveCursorTo(len(t.Runes))
			case input.Code == 'u' && input.Mod == terminal.Ctrl:
				t.deleteToTheLeft()
			case input.Code == 'w' && input.Mod == terminal.Ctrl:
				t.deleteWord()
			case input.Code == terminal.KeyBackspace:
				t.backspace()
			case input.Code == terminal.KeyDelete:
				t.deleteCharacter()
			}
		}
	}
}
