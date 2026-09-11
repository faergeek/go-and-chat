package terminal

import (
	"fmt"
	"os"
	"syscall"
	"unicode"
	"unicode/utf8"
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
	buf         []byte
	pos         int
	file        *os.File
	termiosPrev *termios
}

const bufSize = 128

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

	return &Terminal{
		buf:         make([]byte, bufSize),
		file:        file,
		termiosPrev: termiosPrev,
	}, nil
}

func (t *Terminal) Release() error {
	return setTermios(t.file, t.termiosPrev)
}

const (
	KeyUnknown   = '\x00'
	KeyBackspace = '\b'
	KeyTab       = '\t'
	KeyEnter     = '\r'
	KeyEscape    = '\x1b'
	KeySpace     = '\x20'
)

const (
	KeyArrowLeft = unicode.MaxRune + iota + 1
	KeyArrowRight
	KeyHome
	KeyEnd
	KeyDelete
)

type Mod uint8

const (
	Ctrl Mod = 1 << iota
	Alt
)

type InputKeyboard struct {
	Code rune
	Mod  Mod
	Str  string
}

type Input any

func (t *Terminal) ReadInput() (Input, error) {
	n, err := t.file.Read(t.buf)
	if err != nil {
		return nil, err
	}

	t.buf = t.buf[:n]
	input := t.ground()

	t.buf = t.buf[:bufSize]
	t.pos = 0

	return input, nil
}

func (t *Terminal) ground() Input {
	b := t.buf[t.pos]

	// try to read an escape sequence if there's something else besides an escape
	// character in the buffer
	if b == '\x1b' && len(t.buf) > 1 {
		t.pos++
		return t.escape()
	}

	// control characters
	if b <= '\x1f' || b == '\x7f' {
		switch {
		case b == '\x00':
			return InputKeyboard{Code: KeySpace, Mod: Ctrl}
		case b == '\t':
			return InputKeyboard{Code: KeyTab}
		case b == '\r':
			return InputKeyboard{Code: KeyEnter}
		case b == '\x1b':
			return InputKeyboard{Code: KeyEscape}
		case b == '\x7f':
			return InputKeyboard{Code: KeyBackspace}
		case b >= '\x01' && b <= '\x1a':
			return InputKeyboard{Code: rune(b) + 0x60, Mod: Ctrl}
		case b >= '\x1c' && b <= '\x1f':
			return InputKeyboard{Code: rune(b) + 0x40, Mod: Ctrl}
		}
	}

	r, n := utf8.DecodeRune(t.buf[t.pos:])
	t.pos += n

	return InputKeyboard{Code: unicode.ToLower(r), Str: string(r)}
}

func keyModifiers(param int) Mod {
	var mod Mod

	param -= 1

	if param&0b0010 == 0b0010 {
		mod |= Alt
	}

	if param&0b0100 == 0b0100 {
		mod |= Ctrl
	}

	return mod
}

func (t *Terminal) escape() Input {
	b := t.buf[t.pos]

	if b == '[' { // CSI
		t.pos++

		const missingParam = -1
		curParam := missingParam
		params := []int{}
		for t.pos < len(t.buf) && t.buf[t.pos] >= '\x30' && t.buf[t.pos] <= '\x3f' {
			b := t.buf[t.pos]
			t.pos++

			switch {
			case b >= '0' && b <= '9':
				if curParam == missingParam {
					curParam = 0
				} else {
					curParam = curParam * 10
				}

				curParam += int(b - '0')
			case b == ';':
				params = append(params, curParam)
				curParam = missingParam
			}
		}
		params = append(params, curParam)

		switch t.buf[t.pos] {
		case 'C':
			var mod Mod
			if len(params) == 2 {
				mod = keyModifiers(params[1])
			}

			return InputKeyboard{Code: KeyArrowRight, Mod: mod}
		case 'D':
			var mod Mod
			if len(params) == 2 {
				mod = keyModifiers(params[1])
			}

			return InputKeyboard{Code: KeyArrowLeft, Mod: mod}
		case 'F':
			return InputKeyboard{Code: KeyEnd}
		case 'H':
			return InputKeyboard{Code: KeyHome}
		case '~':
			if len(params) != 0 {
				switch params[0] {
				case 3:
					return InputKeyboard{Code: KeyDelete}
				}
			}
		}
	} else { // Alt-<key>
		input := t.ground()
		if input, ok := input.(InputKeyboard); ok {
			input.Mod |= Alt
			input.Str = ""
			return input
		}
	}

	// ignore anything else
	return InputKeyboard{}
}
