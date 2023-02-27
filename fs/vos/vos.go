// Copyright © 2017, 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import (
        "os"
	"time"
	"os/user"
        "syscall"
)

// Types, constants, and variables from the os package

type FileMode = os.FileMode
type FileInfo = os.FileInfo
type PathError = os.PathError

const (
	ModeIrregular = os.ModeIrregular
	ModeSymlink = os.ModeSymlink
	O_APPEND = os.O_APPEND
	O_CREATE = os.O_CREATE
	O_EXCL = os.O_EXCL
	O_RDONLY = os.O_RDONLY
	O_WRONLY = os.O_WRONLY
	O_RDWR = os.O_RDWR
	O_ACCMODE = O_RDONLY | O_WRONLY | O_RDWR  // defined in /usr/include/asm-generic/fcntl.h
	O_TRUNC = os.O_TRUNC
)

const (
	SEEK_SET = os.SEEK_SET
	SEEK_CUR = os.SEEK_CUR
	SEEK_END = os.SEEK_END
)

var (
	Stderr = os.Stderr
	Stdin = os.Stdin
	Stdout = os.Stdout
)

var (
	ErrInvalid = os.ErrInvalid
	ErrPermission = os.ErrPermission
	ErrExist = os.ErrExist
	ErrNotExist = os.ErrNotExist
	ErrClosed = os.ErrClosed
)


// Types, constants, and variables from the os.exec package

type Cmd interface {
	Run() error
	Start() error
	Wait() error
}


// Types, constants, and variables from the os.user package

type User = user.User


// Types, constants, and variables from the syscall package

type Stat_t = syscall.Stat_t

const (
	EACCES = syscall.EACCES
	EBUSY = syscall.EBUSY
	ECHILD = syscall.ECHILD
	EEXIST = syscall.EEXIST
	EINVAL = syscall.EINVAL
	EISDIR = syscall.EISDIR
	ELOOP = syscall.ELOOP
	ENOENT = syscall.ENOENT
	ENOTBLK = syscall.ENOTBLK
	ENOTDIR = syscall.ENOTDIR
	ENOTEMPTY = syscall.ENOTEMPTY
	EPERM = syscall.EPERM
	ESPIPE = syscall.ESPIPE
	EXDEV = syscall.EXDEV

	MNT_FORCE = syscall.MNT_FORCE

	MS_BIND       = syscall.MS_BIND
	MS_PRIVATE    = syscall.MS_PRIVATE
	MS_REC        = syscall.MS_REC
	MS_REMOUNT    = syscall.MS_REMOUNT
	MS_UNBINDABLE = syscall.MS_UNBINDABLE

	S_IFDIR = syscall.S_IFDIR
	S_IFLNK = syscall.S_IFLNK
	S_IFMT  = syscall.S_IFMT
	S_IFREG = syscall.S_IFREG
)


type Vos interface {
	Chdir(dirname string) error
	Chmod(name string, mode FileMode) error
	Chown(name string, uid, gid int) error
	Chtimes(name string, atime time.Time, mtime time.Time) error
	Clearenv()
	Create(name string) (File, error)
	Command(name string, arg ...string) Cmd
	CurrentUser() (*User, error)
	Exit(code int)
	Expand(s string, mapping func (string) string) string
	ExpandEnv(s string) string
	Getenv(key string) string
	Geteuid() int
	Getpid() int
	Getwd() (string, error)
	Link(oldname, newname string) error
	LookPath(filename string) (string, error)
	LookupUser(username string) (*user.User, error)
	Mkdir(dir string, perm FileMode) error
	MkdirAll(path string, perm FileMode) error
	Mount(source, target, fstype string, flags uintptr, data string) error
	Open(name string) (File, error)
	Readlink(name string) (string, error)
	Remove(name string) error
	RemoveAll(path string) error
	Rename(oldpath, newpath string) error
	Setenv(key, value string) error
	SyscallLstat(filename string, stat *Stat_t) error
	SyscallStat(filename string, stat *Stat_t) error
	Symlink(target, linkname string) error
	Unmount(target string, flags int) error

	OpenFile(filename string, flag int, perm os.FileMode) (*mfsOpenFile, error)
	Readdirnames(directory string) ([]string, error)
	WriteTextFile(filename, contents string) error

	mount(source, target, fstype string, flags uintptr, data string) error
	removeAll(target string) error
	rename(source, target string) error
	symlink(to, from string) error
	unmount(target string, flags int) error
}

type File interface {
	Close() error
	Read(b []byte) (int, error)
	Seek(int64, int) (int64, error)
	Stat() (FileInfo, error)
	Write(b []byte) (int, error)
}




/*
type baseFS struct {}


func (vfs *baseFS) Exists(filename string) bool {
	return vfs.existsStat(filename, syscall.S_IFMT)
}


func (vfs *baseFS) IsDir(filename string) bool {
	return vfs.existsStat(filename, syscall.S_IFDIR)
}


func (vfs *baseFS) IsDirNotSymlink(filename string) bool {
	return vfs.existsLstat(filename, syscall.S_IFDIR)
}


func (vfs *baseFS) IsFile(filename string) bool {
	return vfs.existsStat(filename, syscall.S_IFREG)
}


func (vfs *baseFS) IsFileOrDir(filename string, wantFile bool) bool {
	mask := syscall.S_IFDIR
	if wantFile {
		mask |= syscall.S_IFREG
	}
	return vfs.existsStat(filename, mask)
}


func (vfs *baseFS) IsSymlink(filename string) bool {
	return vfs.existsStat(filename, syscall.S_IFLNK)
}

func (vfs *baseFS) existsLstat(filename string, ftMask uint32) bool {
	var stat syscall.Stat_t
	err := syscall.Lstat(filename, &stat)
	return nil == err && (stat.Mode & syscall.S_IFMT & ftMask > 0)
}


func (vfs *baseFS) existsStat(filename string, ftMask uint32) bool {
	var stat syscall.Stat_t
	err := syscall.Stat(filename, &stat)
	return nil == err && (stat.Mode & syscall.S_IFMT & ftMask > 0)
}
*/

