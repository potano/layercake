package fs

import (
	"os"
	"fmt"
	"syscall"
)


func Symlink(from, to string) error {
	if WriteOK("symlink source=%s target=%s", from, to) {
		err := os.Symlink(to, from)
		if nil != err {
			return fmt.Errorf("%s making symlink %s", err, from)
		}
	}
	return nil
}

func Rename(source, target string) error {
	if WriteOK("rename %s to %s", source, target) {
		return os.Rename(source, target)
	}
	return nil
}

func Remove(target string) error {
	if WriteOK("remove %s", target) {
		return os.RemoveAll(target)
	}
	return nil
}

func Mount(source, target, fstype, options string) error {
	if WriteOK("mount type=%s source=%s target=%s", fstype, source, target) {
		var flags uintptr
		switch fstype {
		case "bind":
			flags = syscall.MS_BIND
		case "rbind":
			flags = syscall.MS_BIND | syscall.MS_REC
		case "remount":
			flags = syscall.MS_REMOUNT
		}
		err := SyscallMount(source, target, fstype, flags, options)
		if nil != err {
			return fmt.Errorf("Cannot mount %s: %s", target, err)
		}
	}
	return nil
}

func Unmount(mounted string, force bool) error {
	if WriteOK("umount directory=%s force=%v", mounted, force) {
		var flags int
		if force {
			flags |= syscall.MNT_FORCE
		}
		err := SyscallUnmount(mounted, flags)
		if err != nil {
			return fmt.Errorf("Cannot unmount %s: %s", mounted, err)
		}
	}
	return nil
}

var SyscallMount func (string, string, string, uintptr, string) error
var SyscallUnmount func (string, int) error

func init() {
	SyscallMount = syscall.Mount
	SyscallUnmount = syscall.Unmount
}

