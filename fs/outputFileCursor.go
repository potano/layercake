package fs

import (
	"os"
	"fmt"
)


type TextOutputFileCursor struct {
	filename string
	lineno int
	fh *os.File
	pretend bool
}


func NewTextOutputFileCursor(filename string) (*TextOutputFileCursor, error) {
	cursor := &TextOutputFileCursor{filename: filename}
	if WriteOK("writing text file %s", filename) {
		fh, err := os.OpenFile(filename, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0666)
		if nil != err {
			return nil, err
		}
		cursor.fh = fh
	} else {
		cursor.pretend = true
	}
	return cursor, nil
}


func (toc *TextOutputFileCursor) Println(line string) {
	toc.lineno++
	if !toc.pretend {
		fmt.Fprintln(toc.fh, line)
	}
}


func (toc *TextOutputFileCursor) Printf(msg string, parms...interface{}) {
	toc.lineno++
	if !toc.pretend {
		fmt.Fprintf(toc.fh, msg, parms...)
	}
}


func (toc *TextOutputFileCursor) Close() {
	if !toc.pretend {
		toc.fh.Close()
	}
}

