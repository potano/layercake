package fs

import (
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
	text string
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


func (tic *TextInputFileCursor) Scan() bool {
	status := tic.scanner.Scan()
	err :=  tic.scanner.Err()
	tic.lineno++
	if err != nil {
		tic.LogError(err.Error())
	}
	tic.text = tic.scanner.Text()
	return status
}


func (tic *TextInputFileCursor) Text() string {
	return tic.text
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

func (tic *TextInputFileCursor) Err() error {
	if len(tic.messages) > 0 {
		return errors.New(strings.Join(tic.messages, "\n"))
	}
	return nil
}


