// Package tui provides terminal UI helpers for interactive input.
package tui

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
	"unicode/utf8"
	"unsafe"
)

// ---------------------------------------------------------------------------
// Linux termios constants (x86_64, from asm-generic/termbits.h)
// ---------------------------------------------------------------------------

const (
	tcgets  = 0x5401
	tcsets  = 0x5402

	// c_lflag bits
	isig   = 0x0001
	icanon = 0x0002
	echo   = 0x0008
	echonl = 0x0040
	iexten = 0x8000

	// c_iflag bits — only the ones we need to unset
	ixon   = 0x0400
	icrnl  = 0x0100

	// c_cc indices (Linux NCCS = 19)
	vmin  = 6
	vtime = 5
)

// termios matches the Linux kernel struct termios layout (36 bytes on x86_64).
type termios struct {
	Iflag uint32
	Oflag uint32
	Cflag uint32
	Lflag uint32
	Line  uint8
	Cc    [19]uint8
}

func getTermios(fd int) (*termios, error) {
	t := &termios{}
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), tcgets, uintptr(unsafe.Pointer(t)), 0, 0, 0)
	if errno != 0 {
		return nil, errno
	}
	return t, nil
}

func setTermios(fd int, t *termios) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), tcsets, uintptr(unsafe.Pointer(t)), 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func makeRaw(fd int) (*termios, error) {
	old, err := getTermios(fd)
	if err != nil {
		return nil, fmt.Errorf("tcgetattr: %w", err)
	}
	raw := *old
	raw.Iflag &^= icrnl | ixon
	raw.Lflag &^= echo | echonl | icanon | isig | iexten
	raw.Cc[vmin] = 1
	raw.Cc[vtime] = 0
	if err := setTermios(fd, &raw); err != nil {
		return nil, fmt.Errorf("tcsetattr: %w", err)
	}
	return old, nil
}

func restoreTermios(fd int, state *termios) error {
	return setTermios(fd, state)
}

// ---------------------------------------------------------------------------
// TTY detection
// ---------------------------------------------------------------------------

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// ---------------------------------------------------------------------------
// Unicode display width (simplified; handles CJK and common wide chars)
// ---------------------------------------------------------------------------

func runeDisplayWidth(r rune) int {
	if r < 32 || r == 0x7f {
		return 0
	}
	// zero-width joiners and combining marks
	if r == 0x200d ||
		(r >= 0x0300 && r <= 0x036f) ||
		(r >= 0x0483 && r <= 0x0489) ||
		(r >= 0x0591 && r <= 0x05bd) {
		return 0
	}
	// CJK and wide characters
	if (r >= 0x1100 && r <= 0x115f) ||
		(r >= 0x2e80 && r <= 0xa4cf) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xff01 && r <= 0xff60) ||
		(r >= 0x1f300 && r <= 0x1f64f) ||
		(r >= 0x20000 && r <= 0x2fffd) ||
		(r >= 0x30000 && r <= 0x3fffd) {
		return 2
	}
	return 1
}

func displayWidthRunes(rs []rune) int {
	w := 0
	for _, r := range rs {
		w += runeDisplayWidth(r)
	}
	return w
}

// ---------------------------------------------------------------------------
// Escape sequence reader
// ---------------------------------------------------------------------------

type escResult int

const (
	escNone   escResult = iota
	escUp
	escDown
	escRight
	escLeft
	escHome
	escEnd
	escDelete
)

func readEscSeq() (escResult, error) {
	b := make([]byte, 1)
	n, err := os.Stdin.Read(b)
	if err != nil {
		return escNone, err
	}
	if n == 0 || b[0] != '[' {
		return escNone, nil
	}

	n, err = os.Stdin.Read(b)
	if err != nil {
		return escNone, err
	}
	if n == 0 {
		return escNone, nil
	}

	switch b[0] {
	case 'A':
		return escUp, nil
	case 'B':
		return escDown, nil
	case 'C':
		return escRight, nil
	case 'D':
		return escLeft, nil
	case 'H':
		return escHome, nil
	case 'F':
		return escEnd, nil
	case '1':
		n, err = os.Stdin.Read(b)
		if err != nil || n == 0 || b[0] != '~' {
			return escNone, nil
		}
		return escHome, nil
	case '3':
		n, err = os.Stdin.Read(b)
		if err != nil || n == 0 || b[0] != '~' {
			return escNone, nil
		}
		return escDelete, nil
	case '4':
		n, err = os.Stdin.Read(b)
		if err != nil || n == 0 || b[0] != '~' {
			return escNone, nil
		}
		return escEnd, nil
	default:
		return escNone, nil
	}
}

// ---------------------------------------------------------------------------
// UTF-8 decoder (byte-by-byte from stdin)
// ---------------------------------------------------------------------------

func readUTF8(lead byte) rune {
	extra := 0
	switch {
	case lead&0xE0 == 0xC0:
		extra = 1
	case lead&0xF0 == 0xE0:
		extra = 2
	case lead&0xF8 == 0xF0:
		extra = 3
	default:
		return utf8.RuneError
	}
	buf := make([]byte, 1+extra)
	buf[0] = lead
	for i := 0; i < extra; i++ {
		n, err := os.Stdin.Read(buf[1+i : 2+i])
		if err != nil || n == 0 {
			return utf8.RuneError
		}
	}
	r, _ := utf8.DecodeRune(buf)
	return r
}

// ---------------------------------------------------------------------------
// Terminal redraw helpers
// ---------------------------------------------------------------------------

func redrawLine(prompt string, runes []rune, pos int) {
	fmt.Print("\r\033[K")
	fmt.Print(prompt)
	fmt.Print(string(runes))
	back := displayWidthRunes(runes[pos:])
	if back > 0 {
		fmt.Printf("\033[%dD", back)
	}
}

// ---------------------------------------------------------------------------
// ReadLine — public API
// ---------------------------------------------------------------------------

// ReadLine reads a single line with full line-editing support (arrow keys,
// Ctrl+A/E for home/end, Ctrl+U to clear, Ctrl+D to delete, Ctrl+C to abort,
// Backspace, Delete). Falls back to bufio.Scanner when stdin is not a TTY.
func ReadLine(prompt string) (string, error) {
	if !isTTY(os.Stdin) {
		fmt.Print(prompt)
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			return scanner.Text(), nil
		}
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", nil
	}

	fd := int(os.Stdin.Fd())
	oldState, err := makeRaw(fd)
	if err != nil {
		fmt.Print(prompt)
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			return scanner.Text(), nil
		}
		return "", scanner.Err()
	}
	defer restoreTermios(fd, oldState)

	fmt.Print(prompt)

	runes := make([]rune, 0, 256)
	pos := 0

	var buf [1]byte
	for {
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			fmt.Println()
			return "", err
		}
		if n == 0 {
			continue
		}
		ch := buf[0]

		switch {
		case ch == '\n' || ch == '\r':
			fmt.Println()
			return string(runes), nil

		case ch == 0x03:
			fmt.Println()
			return "", fmt.Errorf("interrupted")

		case ch == 0x7f || ch == '\b':
			if pos > 0 {
				copy(runes[pos-1:], runes[pos:])
				runes = runes[:len(runes)-1]
				pos--
				redrawLine(prompt, runes, pos)
			}

		case ch == 0x1b:
			res, _ := readEscSeq()
			switch res {
			case escLeft:
				if pos > 0 {
					pos--
					redrawLine(prompt, runes, pos)
				}
			case escRight:
				if pos < len(runes) {
					pos++
					redrawLine(prompt, runes, pos)
				}
			case escHome:
				pos = 0
				redrawLine(prompt, runes, pos)
			case escEnd:
				pos = len(runes)
				redrawLine(prompt, runes, pos)
			case escDelete:
				if pos < len(runes) {
					copy(runes[pos:], runes[pos+1:])
					runes = runes[:len(runes)-1]
					redrawLine(prompt, runes, pos)
				}
			case escUp, escDown:
				// Ignore up/down.
			}

		case ch == 0x01:
			pos = 0
			redrawLine(prompt, runes, pos)

		case ch == 0x05:
			pos = len(runes)
			redrawLine(prompt, runes, pos)

		case ch == 0x04:
			if pos < len(runes) {
				copy(runes[pos:], runes[pos+1:])
				runes = runes[:len(runes)-1]
				redrawLine(prompt, runes, pos)
			} else if len(runes) == 0 {
				fmt.Println()
				return "", nil
			}

		case ch == 0x15:
			runes = runes[:0]
			pos = 0
			redrawLine(prompt, runes, pos)

		case ch >= 32 && ch <= 126:
			runes = append(runes, 0)
			copy(runes[pos+1:], runes[pos:])
			runes[pos] = rune(ch)
			pos++
			redrawLine(prompt, runes, pos)

		default:
			if ch >= 0x80 {
				r := readUTF8(ch)
				if r != utf8.RuneError {
					runes = append(runes, 0)
					copy(runes[pos+1:], runes[pos:])
					runes[pos] = r
					pos++
					redrawLine(prompt, runes, pos)
				}
			}
		}
	}
}
