package terminal

import (
	"io"
	"os"

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
	reader              io.Reader
	restoreTerminalMode func() error
}

func AcquireTerminal() (Terminal, error) {
	restoreTerminalMode, err := rawMode()
	if err != nil {
		return Terminal{}, err
	}

	t := Terminal{
		reader:              os.Stdin,
		restoreTerminalMode: restoreTerminalMode,
	}

	return t, nil
}

func (t *Terminal) Release() error {
	return t.restoreTerminalMode()
}

type InputKind int

const (
	InputKindUnknown InputKind = iota
	InputKindPrint
	InputKindControl
	InputKindCSI
)

type Input struct {
	Kind InputKind
	Str  string
}

func (t *Terminal) ReadInput() (Input, error) {
	buf := make([]byte, 1)
	_, err := t.reader.Read(buf)
	if err != nil {
		return Input{}, err
	}

	// Printable ASCII
	if '\x20' <= buf[0] && buf[0] <= '\x7e' {
		return Input{Kind: InputKindPrint, Str: string(buf)}, nil
	}

	// UTF-8 multibyte sequence
	if buf[0] > '\x7f' {
		// High bits encodes the sequence length:
		// 110xxxxx - 2 bytes
		// 1110xxxx - 3 bytes
		// etc.
		l := 1
		for b := buf[0] << 1; b&(1<<7) != 0; b <<= 1 {
			l++
		}

		data := []byte{buf[0]}
		// Don't stop until the whole sequence is read
		for len(data) < l {
			_, err = t.reader.Read(buf)
			if err != nil {
				return Input{}, err
			}

			data = append(data, buf...)
		}

		return Input{Kind: InputKindPrint, Str: string(data)}, nil
	}

	// treat anything other than Escape as a single control character
	if buf[0] != '\x1b' {
		return Input{Kind: InputKindControl, Str: string(buf)}, nil
	}

	data := []byte{buf[0]}
	_, err = t.reader.Read(buf)
	if err != nil {
		return Input{}, err
	}

	data = append(data, buf[0])

	// CSI
	if buf[0] == '[' {
		for {
			_, err = t.reader.Read(buf)
			if err != nil {
				return Input{}, err
			}

			data = append(data, buf...)

			// Stop when final byte of CSI is read
			if '\x40' <= buf[0] && buf[0] <= '\x7e' {
				break
			}
		}

		return Input{Kind: InputKindCSI, Str: string(data[2:])}, nil
	}

	// ignore anything else
	return Input{}, nil
}
