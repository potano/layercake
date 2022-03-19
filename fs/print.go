// Copyright © 2021 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package fs

import (
	"io"
	"os"
	"fmt"
)


var MessageWriter io.Writer


func Print(a...interface{}) {
	fmt.Fprint(MessageWriter, a...)
}


func Printf(format string, a...interface{}) {
	fmt.Fprintf(MessageWriter, format, a...)
}


func Println(a...interface{}) {
	fmt.Fprintln(MessageWriter, a...)
}



func init() {
	MessageWriter = os.Stdout
}

