package fns

import (
	"os"
	"fmt"
	"path"
)

func TemplatedExitMessage(baseMessage string, exitCode int, subst map[string]string) {
	subst["myself"] = path.Base(os.Args[0])
	fmt.Fprintln(os.Stderr, Template(baseMessage, subst))
	os.Exit(exitCode)
}

