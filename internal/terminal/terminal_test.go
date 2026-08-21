package terminal

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestReadInput(t *testing.T) {
	tests := map[string]struct {
		input   []byte
		want    Input
		wantErr error
	}{
		"empty input": {
			input:   []byte{},
			wantErr: io.EOF,
		},
		"basic printable ascii": {
			input: []byte("hello"),
			want:  Input{Kind: InputKindPrint, Str: string([]byte{104})},
		},
		"utf-8": {
			input: []byte{226, 156, 140, 240, 159, 140, 143},
			want:  Input{Kind: InputKindPrint, Str: string([]byte{226, 156, 140})},
		},
		"control characters": {
			input: []byte{3},
			want:  Input{Kind: InputKindControl, Str: string([]byte{3})},
		},
		"CSI": {
			input: []byte("\x1b[2;1R foo, bar"),
			want:  Input{Kind: InputKindCSI, Str: string([]byte{50, 59, 49, 82})},
		},
		"Cursor Up": {
			input: []byte("\x1b[A abc"),
			want:  Input{Kind: InputKindCSI, Str: string([]byte{65})},
		},
	}

	for name, tc := range tests {
		term := Terminal{
			reader: bytes.NewBuffer([]byte(tc.input)),
		}

		got, err := term.ReadInput()
		if tc.wantErr != nil {
			if tc.wantErr.Error() != err.Error() {
				t.Fatalf("%s: expected an error, got: %v", name, got)
			}
		}

		if !reflect.DeepEqual(tc.want, got) {
			t.Fatalf("%s: expected: %v, got: %v", name, tc.want, got)
		}
	}
}
