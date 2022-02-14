package fs

import (
	"io"
	"os"
	"fmt"
	"bufio"
	"errors"
	"strings"
)


type LineReader interface {
	ReadLine(*string) bool
	Close()
	LogError(string)
	Err() error
	GetMessages() []string
}


type TextInputCursor struct {
	filename string
	lineno int
	scanner *bufio.Scanner
	messages []string
}


type TextInputFileCursor struct {
	*TextInputCursor
	fh *os.File
}


func NewTextInputCursor(filename string, reader io.Reader) (*TextInputCursor) {
	return &TextInputCursor{filename: filename, scanner: bufio.NewScanner(reader)}
}


func NewTextInputFileCursor(filename string) (*TextInputFileCursor, error) {
	fh, err := os.OpenFile(filename, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}
	return &TextInputFileCursor{
		TextInputCursor: NewTextInputCursor(filename, fh),
		fh: fh,
	}, nil
}


func (tifc *TextInputFileCursor) Close() {
	tifc.fh.Close()
}


func (tic *TextInputCursor) ReadLine(s *string) bool {
	status := tic.scanner.Scan()
	err :=  tic.scanner.Err()
	tic.lineno++
	if err != nil {
		tic.LogError(err.Error())
	}
	*s = tic.scanner.Text()
	return status
}


func (tic *TextInputCursor) ReadNonBlankNonCommentLine(s *string) bool {
	for {
		rc := tic.ReadLine(s)
		if rc == false {
			return false
		}
		ll := strings.TrimSpace(*s)
		if len(ll) > 0 && ll[0] != '#' && (len(ll) < 2 || ll[:2] != "//") {
			break
		}
	}
	return true
}


func (tic *TextInputCursor) Close() {}


func (tic *TextInputCursor) LogError(message string) {
	tic.messages = append(tic.messages, fmt.Sprintf(message + " in %s line %d", tic.filename,
	tic.lineno))
}


func (tic *TextInputCursor) GetMessages() []string {
	return tic.messages
}

func (tic *TextInputCursor) Err() error {
	if len(tic.messages) > 0 {
		return errors.New(strings.Join(tic.messages, "\n"))
	}
	return nil
}


