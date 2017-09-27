package fs

import (
	"os"
	"fmt"
	"os/exec"
)

func Shell(dirname string) error {
	exe := os.Getenv("SHELL")
	if len(exe) == 0 {
		return fmt.Errorf("Cannot dereference the SHELL environment variable")
	}
	err := os.Chdir(dirname)
	if nil != err {
		return fmt.Errorf("%s attempting to switch to directory %s", err, dirname)
	}
	cmd := exec.Command(exe)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dirname
	return cmd.Run()
}

func Chroot(dirname, exe string, env []string, fds []*os.File) error {
	if len(exe) < 1 {
		var err error
		exe, err = exec.LookPath("chroot")
		if nil != err {
			return fmt.Errorf("%s looking up chroot executable", err)
		}
	}
	cmd := exec.Command(exe, dirname)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if len(env) > 0 {
		e := os.Environ()
		e = append(e, env...)
		cmd.Env = e
	}
	if len(fds) > 0 {
		cmd.ExtraFiles = fds
	}
	return cmd.Run()
}

