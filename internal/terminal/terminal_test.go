package terminal

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"
)

func TestReadInput(t *testing.T) {
	tests := map[string]struct {
		input string
		want  []Input
	}{
		"empty input": {
			input: "",
			want:  []Input{},
		},
		"printable ascii": {
			input: "hello, world!",
			want: []Input{
				{Kind: InputKindPrint, Str: "h"},
				{Kind: InputKindPrint, Str: "e"},
				{Kind: InputKindPrint, Str: "l"},
				{Kind: InputKindPrint, Str: "l"},
				{Kind: InputKindPrint, Str: "o"},
				{Kind: InputKindPrint, Str: ","},
				{Kind: InputKindPrint, Str: " "},
				{Kind: InputKindPrint, Str: "w"},
				{Kind: InputKindPrint, Str: "o"},
				{Kind: InputKindPrint, Str: "r"},
				{Kind: InputKindPrint, Str: "l"},
				{Kind: InputKindPrint, Str: "d"},
				{Kind: InputKindPrint, Str: "!"},
			},
		},
		"utf-8": {
			input: "🫪 wow, wood 🪵",
			want: []Input{
				{Kind: InputKindPrint, Str: "🫪"},
				{Kind: InputKindPrint, Str: " "},
				{Kind: InputKindPrint, Str: "w"},
				{Kind: InputKindPrint, Str: "o"},
				{Kind: InputKindPrint, Str: "w"},
				{Kind: InputKindPrint, Str: ","},
				{Kind: InputKindPrint, Str: " "},
				{Kind: InputKindPrint, Str: "w"},
				{Kind: InputKindPrint, Str: "o"},
				{Kind: InputKindPrint, Str: "o"},
				{Kind: InputKindPrint, Str: "d"},
				{Kind: InputKindPrint, Str: " "},
				{Kind: InputKindPrint, Str: "🪵"},
			},
		},
		"control characters": {
			input: "\x03\x04",
			want: []Input{
				{Kind: InputKindControl, Str: "\x03"},
				{Kind: InputKindControl, Str: "\x04"},
			},
		},
		"Cursor Report + extra chars": {
			input: "\x1b[2;1R foo, bar",
			want: []Input{
				{Kind: InputKindControl, Str: "\x1b[2;1R"},
				{Kind: InputKindPrint, Str: " "},
				{Kind: InputKindPrint, Str: "f"},
				{Kind: InputKindPrint, Str: "o"},
				{Kind: InputKindPrint, Str: "o"},
				{Kind: InputKindPrint, Str: ","},
				{Kind: InputKindPrint, Str: " "},
				{Kind: InputKindPrint, Str: "b"},
				{Kind: InputKindPrint, Str: "a"},
				{Kind: InputKindPrint, Str: "r"},
			},
		},
		"Cursor Up + extra chars": {
			input: "\x1b[A abc",
			want: []Input{
				{Kind: InputKindControl, Str: "\x1b[A"},
				{Kind: InputKindPrint, Str: " "},
				{Kind: InputKindPrint, Str: "a"},
				{Kind: InputKindPrint, Str: "b"},
				{Kind: InputKindPrint, Str: "c"},
			},
		},
	}

	formatInput := func(input Input) string {
		switch input.Kind {
		case InputKindUnknown:
			return "unknown input"
		case InputKindPrint:
			return fmt.Sprintf("printable input: '%s'", input.Str)
		case InputKindControl:
			return fmt.Sprintf("control input: %v", []byte(input.Str))
		}

		t.Fatalf("Unexpected want.Kind: %v", input.Kind)
		return ""
	}

	for name, tc := range tests {
		term := Terminal{
			reader: bufio.NewReader(bytes.NewBuffer([]byte(tc.input))),
		}

		for i, want := range tc.want {
			got, err := term.ReadInput()
			if err != nil && errors.Is(err, io.EOF) {
				break
			}

			if err == nil && reflect.DeepEqual(want, got) {
				continue
			}

			var gotStr string
			if err != nil {
				gotStr = fmt.Sprintf("an error \"%s\"", err)
			} else {
				gotStr = formatInput(got)
			}

			t.Fatalf("%s: at index %d expected: %s, got: %s", name, i, formatInput(want), gotStr)
		}
	}
}
