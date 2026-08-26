package terminal

import (
	"bufio"
	"os"
	"unicode"

	"golang.org/x/sys/unix"
)

func IsTerminal() bool {
	_, err := unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)

	return err == nil
}

func rawMode() (func() error, error) {
	termios, err := unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)
	if err != nil {
		return nil, err
	}

	termiosOrig := *termios
	termios.Lflag = termios.Lflag &^ (unix.ECHO | unix.ICANON | unix.ISIG | unix.IEXTEN)
	termios.Iflag = termios.Iflag &^ (unix.IXON | unix.ICRNL | unix.BRKINT | unix.INPCK | unix.ISTRIP)
	termios.Oflag = termios.Oflag &^ (unix.OPOST)
	termios.Cflag = termios.Cflag | unix.CS8

	err = unix.IoctlSetTermios(unix.Stdin, unix.TCSETSF, termios)
	if err != nil {
		return nil, err
	}

	return func() error {
		return unix.IoctlSetTermios(unix.Stdin, unix.TCSETSF, &termiosOrig)
	}, nil
}

type Terminal struct {
	reader  *bufio.Reader
	restore func() error
}

func AcquireTerminal() (Terminal, error) {
	restore, err := rawMode()
	if err != nil {
		return Terminal{}, err
	}

	t := Terminal{
		reader:  bufio.NewReader(os.Stdin),
		restore: restore,
	}

	return t, nil
}

func (t *Terminal) Release() error {
	return t.restore()
}

type InputKind int

const (
	InputKindUnknown InputKind = iota
	InputKindPrint
	InputKindControl
)

type Input struct {
	Kind InputKind
	Str  string
}

func (t *Terminal) ReadInput() (Input, error) {
	r, _, err := t.reader.ReadRune()
	if err != nil {
		return Input{}, err
	}

	if unicode.IsPrint(r) {
		return Input{Kind: InputKindPrint, Str: string(r)}, nil
	}

	// treat anything other than Escape as a single control character
	if r != '\x1b' {
		return Input{Kind: InputKindControl, Str: string(r)}, nil
	}

	data := []rune{r}
	r, _, err = t.reader.ReadRune()
	if err != nil {
		return Input{}, err
	}

	data = append(data, r)

	// CSI
	if r == '[' {
		for {
			r, _, err = t.reader.ReadRune()
			if err != nil {
				return Input{}, err
			}

			data = append(data, r)

			// Stop when final byte of CSI is read
			if '\x40' <= r && r <= '\x7e' {
				break
			}
		}

		return Input{Kind: InputKindControl, Str: string(data)}, nil
	}

	// ignore anything else
	return Input{}, nil
}
