package main

import (
	"io"
	"slices"
	"unicode"

	"github.com/faergeek/go-and-chat/internal/terminal"
)

type textInput struct {
	Runes  []rune
	Cursor int
}

func newTextInput() textInput {
	return textInput{Runes: []rune{}}
}

func (t *textInput) insert(r rune) {
	t.Runes = slices.Insert(t.Runes, t.Cursor, r)
	t.Cursor++
}

func (t *textInput) backspace() {
	if t.Cursor > 0 {
		t.Runes = slices.Delete(t.Runes, t.Cursor-1, t.Cursor)
		t.Cursor--
	}
}

func (t *textInput) deleteCharacter() {
	if t.Cursor < len(t.Runes) {
		t.Runes = slices.Delete(t.Runes, t.Cursor, t.Cursor+1)
	}
}

func (t *textInput) deleteWord() {
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

func (t *textInput) deleteToTheLeft() {
	t.Runes = slices.Delete(t.Runes, 0, t.Cursor)
	t.Cursor = 0
}

func (t *textInput) moveCursorTo(n int) {
	t.Cursor = max(min(n, len(t.Runes)), 0)
}

func (t *textInput) moveCursorBy(n int) {
	t.moveCursorTo(t.Cursor + n)
}

func (t *textInput) clear() {
	t.Runes = []rune{}
	t.Cursor = 0
}

func (t *textInput) handleInput(input *terminal.Input) (done bool, err error) {
	switch input.Kind {
	case terminal.InputKindPrint:
		for _, r := range input.Str {
			t.insert(r)
		}
	case terminal.InputKindControl:
		switch input.Str {
		case "\x01":
			t.moveCursorTo(0)
		case "\x02":
			t.moveCursorBy(-1)
		case "\x03":
			return true, io.EOF
		case "\x05":
			t.moveCursorTo(len(t.Runes))
		case "\x06":
			t.moveCursorBy(1)
		case "\x15":
			t.deleteToTheLeft()
		case "\x17":
			t.deleteWord()
		case "\r", "\n":
			return true, nil
		case "\x7f":
			t.backspace()
		}
	case terminal.InputKindCSI:
		switch input.Str {
		case "3~":
			t.deleteCharacter()
		case "C":
			t.moveCursorBy(1)
		case "D":
			t.moveCursorBy(-1)
		case "F":
			t.moveCursorTo(len(t.Runes))
		case "H":
			t.moveCursorTo(0)
		}
	}

	return false, nil
}
