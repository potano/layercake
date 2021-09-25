package fs

import (
	"io"
	"os"
	"fmt"
	"bufio"
	"errors"
	"strings"
)


type TextInputFileCursor struct {
	filename string
	lineno int
	fh *os.File
	scanner *bufio.Scanner
	messages []string
}


func NewTextInputFileCursor(filename string) (*TextInputFileCursor, error) {
	fh, err := os.OpenFile(filename, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(fh)
	return &TextInputFileCursor{
		filename: filename,
		lineno: 0,
		fh: fh,
		scanner: scanner,
		messages: []string{},
	}, nil
}


func (tic *TextInputFileCursor) GetLine() (string, error) {
	if !tic.scanner.Scan() {
		return "", io.EOF
	}
	tic.lineno++
	return tic.scanner.Text(), nil
}


func (tic *TextInputFileCursor) Close() {
	tic.fh.Close()
}


func (tic *TextInputFileCursor) LogError(message string) {
	tic.messages = append(tic.messages, fmt.Sprintf(message + " in %s line %d", tic.filename,
	tic.lineno))
}


func (tic *TextInputFileCursor) HaveError() bool {
	return len(tic.messages) > 0
}


func (tic *TextInputFileCursor) GetMessages() []string {
	return tic.messages
}

func (tic *TextInputFileCursor) GetError() error {
	return errors.New(strings.Join(tic.messages, "\n"))
}


