// Copyright © 2017, 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import (
        "os"
	"time"
	"os/exec"
	"os/user"
        "syscall"
)

type RealOS struct {
	Args []string
	Chdir func (dirname string) error
	Chmod func (name string, mode os.FileMode) error
	Chown func (name string, uid, gid int) error
	Chtimes func (name string, atime time.Time, mtime time.Time) error
	Clearenv func ()
	// Create func (name string) (File, error)
	// Command func (name string, arg ...string) Cmd
	CurrentUser func () (*User, error)
	Exit func (code int)
	Expand func (s string, mapping func (string) string) string
	ExpandEnv func (s string) string
	Getenv func (key string) string
	Geteuid func () int
	Getpid func () int
	Getwd func () (string, error)
	Link func (oldname, newname string) error
	LookPath func (filename string) (string, error)
	LookupUser func (username string) (*User, error)
	Lstat func (filename string, stat *Stat_t) error
	Mkdir func (dir string, perm os.FileMode) error
	MkdirAll func (path string, perm os.FileMode) error
	Mount func (source, target, fstype string, flags uintptr, data string) error
	// Open func (name string) (File, error)
	Readlink func (name string) (string, error)
	Remove func (name string) error
	RemoveAll func (path string) error
	Rename func (oldpath, newpath string) error
	Setenv func (key, value string) error
	Stat func (filename string, stat *Stat_t) error
	Unmount func (target string, flags int) error
}

func NewRealOS() *RealOS {
	return &RealOS{
		Args: os.Args,
		Chdir: os.Chdir,
		Chmod: os.Chmod,
		Chown: os.Chown,
		Chtimes: os.Chtimes,
		Clearenv: os.Clearenv,
		CurrentUser: user.Current,
		Exit: os.Exit,
		Expand: os.Expand,
		ExpandEnv: os.ExpandEnv,
		Getenv: os.Getenv,
		Geteuid: os.Geteuid,
		Getpid: os.Getpid,
		Getwd: os.Getwd,
		Link: os.Link,
		LookPath: exec.LookPath,
		LookupUser: user.Lookup,
		Lstat: syscall.Lstat,
		Mkdir: os.Mkdir,
		MkdirAll: os.MkdirAll,
		Mount: syscall.Mount,
		Readlink: os.Readlink,
		Remove: os.Remove,
		RemoveAll: os.RemoveAll,
		Rename: os.Rename,
		Setenv: os.Setenv,
		Stat: syscall.Stat,
		Unmount: syscall.Unmount,
	}
}


func (os *RealOS) Create(name string) (File, error) {
	return os.Create(name)
}

func (os *RealOS) Command(name string, arg ...string) Cmd {
	return os.Command(name, arg...)
}

func (os *RealOS) Open(name string) (File, error) {
	return os.Open(name)
}

