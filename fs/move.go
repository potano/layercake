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
	if WriteOK("mkdir(%s)", dir) {
		err := os.Mkdir(dir, 0755)
		if nil != err {
			return fmt.Errorf("%s attempting mkdir of %s", err, dir)
		}
	}
	return nil
}

func Symlink(from, to string) error {
	if WriteOK("symlink(%s, %s)", from, to) {
		err := os.Symlink(to, from)
		if nil != err {
			return fmt.Errorf("%s making symlink %s", err, from)
		}
	}
	return nil
}

func Rename(source, target string) error {
	if WriteOK("mv(%s, %s)", source, target) {
		return os.Rename(source, target)
	}
	return nil
}

func Remove(target string) error {
	if WriteOK("rm(%s)", target) {
		return os.RemoveAll(target)
	}
	return nil
}

func Mount(source, target, fstype, options string) error {
	if WriteOK("mount --type %s %s %s", fstype, source, target) {
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
	}
	return nil
}

func Unmount(mounted string, force bool) error {
	if WriteOK("umount %s (force=%v)", mounted, force) {
		var flags int
		if force {
			flags |= syscall.MNT_FORCE
		}
		return syscall.Unmount(mounted, flags)
	}
	return nil
}

