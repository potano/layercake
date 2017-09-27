package fs

import (
	"os"
	"fmt"
	"syscall"
)

func Cd(dir string) error {
	err := os.Chdir(dir)
	if nil != err {
		return fmt.Errorf("%s attempting chdir to %s", err, dir)
	}
	return nil
}

func Mkdir(dir string) error {
	err := os.Mkdir(dir, 0755)
	if nil != err {
		return fmt.Errorf("%s attempting mkdir of %s", err, dir)
	}
	return nil
}

func Symlink(from, to string) error {
	err := os.Symlink(to, from)
	if nil != err {
		return fmt.Errorf("%s making symlink %s", err, from)
	}
	return nil
}

func Rename(source, target string) error {
	return os.Rename(source, target)
}

func Remove(target string) error {
	return os.RemoveAll(target)
}

func Mount(source, target, fstype, options string) error {
	var flags uintptr
	switch fstype {
	case "bind":
		flags = syscall.MS_BIND
	case "rbind":
		flags = syscall.MS_BIND | syscall.MS_REC
	case "remount":
		flags = syscall.MS_REMOUNT
	}
	err := syscall.Mount(source, target, fstype, flags, options)
	if nil != err {
		return fmt.Errorf("Cannot mount %s: %s", target, err)
	}
	return nil
}

func Unmount(mounted string, force bool) error {
	var flags int
	if force {
		flags |= syscall.MNT_FORCE
	}
	return syscall.Unmount(mounted, flags)
}

