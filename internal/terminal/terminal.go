package terminal

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
	"unicode"
	"unsafe"
)

type termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Cc     [20]byte
	Ispeed uint32
	Ospeed uint32
}

func getTermios(file *os.File) (*termios, error) {
	var result termios

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(syscall.TCGETS),
		uintptr(unsafe.Pointer(&result)),
	)
	if errno != 0 {
		return nil, fmt.Errorf("TCGETS: %v", errno)
	}

	return &result, nil
}

func setTermios(file *os.File, t *termios) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(syscall.TCSETS),
		uintptr(unsafe.Pointer(t)),
	)
	if errno != 0 {
		return fmt.Errorf("TCSETS: %v", errno)
	}

	return nil
}

func IsTerminal(file *os.File) bool {
	_, err := getTermios(file)

	return err == nil
}

type Terminal struct {
	file        *os.File
	reader      *bufio.Reader
	termiosPrev *termios
}

func AcquireTerminal(file *os.File) (*Terminal, error) {
	termiosPrev, err := getTermios(file)
	if err != nil {
		return nil, err
	}

	termios := *termiosPrev
	termios.Lflag &^= (syscall.ECHO | syscall.ICANON | syscall.ISIG | syscall.IEXTEN)
	termios.Iflag &^= (syscall.IXON | syscall.ICRNL | syscall.BRKINT | syscall.INPCK | syscall.ISTRIP)
	termios.Oflag &^= (syscall.OPOST)
	termios.Cflag |= syscall.CS8

	err = setTermios(file, &termios)
	if err != nil {
		return nil, err
	}

	t := Terminal{
		file:        file,
		reader:      bufio.NewReader(file),
		termiosPrev: termiosPrev,
	}

	return &t, nil
}

func (t *Terminal) Release() error {
	return setTermios(t.file, t.termiosPrev)
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
