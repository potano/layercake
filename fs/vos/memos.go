// Copyright © 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos


import (
	"io"
	"os"
	"time"
	"strings"
	"syscall"
)

/*
 * The MemOS structure represents not simply the OS API plus related Go library routines
 * but also locates them in the context of a process.  The semantics of the public Linux
 * API, after all, must always be understood in the context of a process.
 */

type MemOS struct {
	ns *namespaceType
	pid int
	euid, gid, umask uint64
	groups []uint64
	root, cwd *mfsOpenFile
	openFiles map[int]*mfsOpenFile
	environment map[string]string
	Args []string
	Stdin io.Reader
	Stdout io.Writer
	Stderr io.Writer
	exitCode int
}

type mfsOpenFile struct {
	mount *mountType
	inode inodeType
	mos *MemOS
	fd, flags int
	pos int64
	name, abspath string
	readable, writable, executable bool
}

type mosCmd struct {
	Path string
	Args []string
	Env []string
	Dir string
	Stdin io.Reader
	Stdout io.Writer
	Stderr io.Writer
	ExtraFiles []*mfsOpenFile
	mos, newProcess *MemOS
}


func NewMemOS() (*MemOS, error) {
	mos := &MemOS{pid: initPid, umask: initUmask}
	err := newRootNamespace(mos)
	if err != nil {
		return nil, err
	}
	return mos, nil
}



func (mos *MemOS) Chdir(name string) error {
	return toPathError("chdir", name, mos.chdir(name))
}


func (mos *MemOS) Chmod(name string, perm FileMode) error {
	return toPathError("chmod", name, mos.chmod(name, perm))
}


func (mos *MemOS) Chown(name string, uid, gid int) error {
	return toPathError("chown", name, mos.chown(name, uid, gid))
}


func (mos *MemOS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return toPathError("chtimes", name, mos.chtimes(name, atime, mtime))
}


func (mos *MemOS) Clearenv() {
	mos.environment = map[string]string{}
}


func (mos *MemOS) Command(name string, arg ...string) *mosCmd {
	return &mosCmd{
		Path: name,
		Args: append([]string{name}, arg...),
		mos: mos,
	}
}


func (mos *MemOS) Create(filename string) (*mfsOpenFile, error) {
	file, err := mos.open(filename, O_CREATE | O_WRONLY | O_TRUNC, defaultCreateMode)
	if err != nil {
		return nil, toPathError("create", filename, err)
	}
	return file, nil
}


// Note: Environ is NOT part of the Vos interface
func (mos *MemOS) Environ() []string {
	return EnvironmentMapToStrings(mos.environment)
}


func (mos *MemOS) Exited() bool {
	return mos.exitCode >= 0
}


func (mos *MemOS) Exit(code int) {
	mos.closeAll()
	mos.exitCode = 0xFF & code
}


func (mos *MemOS) ExitCode() int {
	return mos.exitCode
}


func (mos *MemOS) Expand(s string, mapping func (string) string) string {
	return os.Expand(s, mapping);
}


func (mos *MemOS) ExpandEnv(s string) string {
	return os.Expand(s, mos.GetEnv)
}


func (mos *MemOS) GetEnv(key string) string {
	return mos.environment[key]
}


func (mos *MemOS) Getegid() int {
	return int(mos.gid)
}


func (mos *MemOS) Geteuid() int {
	return int(mos.euid)
}


func (mos *MemOS) Getpid() int {
	return mos.pid
}


func (mos *MemOS) Getwd() (string, error) {
	if mos.cwd == nil {
		return "", EACCES
	}
	return mos.cwd.abspath, nil
}


// Note: InsertStorageDevice is NOT part of the Vos interface and does not correspond to an actual
// syscall
func (mos *MemOS) InsertStorageDevice(major, minor int, fstype, source string) error {
	_, err := mos.ns.makeFilesystem(major, minor, fstype, source)
	return err
}


func (mos *MemOS) Link(oldname, newname string) error {
	return toPathError("link", oldname + " " + newname, mos.link(oldname, newname))
}


func (mos *MemOS) LookPath(file string) (string, error) {
	if len(file) == 0 {
		return "", ENOENT
	}
	if file[0] == '/' || file[0] == '.' {
		mount, dirInode, inode, _, err := mos.inodeAtPathWithParent(mos.cwd, file)
		if err != nil {
			return "", err
		}
		if inode == nil {
			return "", ENOENT
		}
		if !mos.hasExecutePermission(inode) {
			return "", EACCES
		}
		return mos.findAbsolutePath(mount, dirInode, inode)
	}
	pathString := mos.environment["PATH"]
	if len(pathString) == 0 {
		return "", ENOENT
	}
	for _, part := range strings.Split(pathString, ":") {
		mount, dirInode, inode, _, err :=
			mos.inodeAtPathWithParent(mos.cwd, part + "/" + file)
		if err != nil {
			return "", err
		}
		if inode == nil || !mos.hasExecutePermission(inode) {
			continue
		}
		return mos.findAbsolutePath(mount, dirInode, inode)
	}
	return "", ENOENT
}


func (mos *MemOS) Mkdir(dirname string, perm FileMode) error {
	return toPathError("mkdir", dirname, mos.mkdir(dirname, perm))
}


func (mos *MemOS) MkdirAll(dirname string, perm FileMode) error {
	return toPathError("mkdirall", dirname, mos.mkdirAll(dirname, perm))
}


// Note: Mkfifo is NOT part of the Vos interface
func (mos *MemOS) Mkfifo(pathname string, perm FileMode) error {
	return toPathError("mkfifo", pathname, mos.mkfifo(pathname, perm))
}


func (mos *MemOS) Open(name string) (*mfsOpenFile, error) {
	file, err := mos.open(name, O_RDONLY, 0)
	if err != nil {
		return nil, toPathError("open", name, err)
	}
	return file, nil
}


func (mos *MemOS) OpenFile(name string, flag int, perm FileMode) (*mfsOpenFile, error) {
	file, err := mos.open(name, flag, uint64(perm))
	if err != nil {
		return nil, toPathError("open", name, err)
	}
	return file, nil
}


func (mos *MemOS) Remove(name string) error {
	return toPathError("remove", name, mos.remove(name))
}


func (mos *MemOS) RemoveAll(name string) error {
	return toPathError("removeall", name, mos.removeAll(name))
}


func (mos *MemOS) SetEnv(key, value string) error {
	mos.environment[key] = value
	return nil
}


func (mos *MemOS) SyscallLstat(filename string, stat *syscall.Stat_t) error {
	_, _, inode, _, err := mos.inodeAtPathWithParent(mos.cwd, filename)
	if err != nil {
		return err
	}
	if inode == nil {
		return ENOENT
	}
	return inode.Stat(stat)
}


func (mos *MemOS) SyscallMount(source, mtpoint, fstype string, flgs uintptr, options string) error {
	return toPathError("mount", source + " " + mtpoint,
		mos.mount(source, mtpoint, fstype, flgs, options))
}


func (mos *MemOS) SyscallStat(filename string, stat *syscall.Stat_t) error {
	_, inode, err := mos.inodeAtPath(mos.cwd, filename)
	if err != nil {
		return err
	}
	return inode.Stat(stat)
}


func (mos *MemOS) SyscallUnmount(mountpoint string, flags int) error {
	return toPathError("unmount", mountpoint, mos.umount(mountpoint, flags))
}


func (mos *MemOS) Symlink(oldname, newname string) error {
	return toPathError("symlink", oldname + " " + newname,  mos.symlink(oldname, newname))
}




func toPathError(op, pathname string, err error) error {
	if err != nil {
		return &PathError{op, pathname, err}
	}
	return nil
}

