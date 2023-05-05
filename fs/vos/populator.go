// Copyright © 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import (
	"fmt"
	"time"
	"syscall"
)


type PopulatorType []PopulatorAction

type PopulatorAction interface {
	describe() string
	populate(pop *PopulatorData) error
}


type PopulatorData struct {
	Mos *MemOS
	Uid, Gid uint64
	CmdMap map[string]*mosCmd
	OpenMap map[string]*mfsOpenFile
}


func (actions PopulatorType) Populate(mos *MemOS) (*PopulatorData, error) {
	pop := &PopulatorData{
		Mos: mos,
		Uid: uint64(mos.Geteuid()),
		Gid: uint64(mos.Getegid()),
		CmdMap: map[string]*mosCmd{},
		OpenMap: map[string]*mfsOpenFile{},
	}
	for i, act := range actions {
		err := act.populate(pop)
		if err != nil {
			return pop, fmt.Errorf("populate %s (action %d): %s", act.describe(), i,				 err)
		}
	}
	return pop, nil
}



type PopUidGid struct {
	Uid, Gid uint64
}

func (a PopUidGid) describe() string {
	return fmt.Sprintf("PopUidGid %d/%d", a.Uid, a.Gid)
}

func (a PopUidGid) populate(pop *PopulatorData) error {
	pop.Uid = a.Uid
	pop.Gid = a.Gid
	return nil
}


type PopFile struct {
	Name string
	Perms uint64
	Contents string
}

func (a PopFile) describe() string {
	return "PopFile"
}

func (a PopFile) populate(pop *PopulatorData) error {
	needPerms := a.Perms | syscall.S_IWUSR
	file, err := pop.Mos.OpenFile(a.Name, O_CREATE | O_EXCL | O_WRONLY, FileMode(needPerms))
	if err != nil {
		return err
	}
	if len(a.Contents) > 0 {
		n, err := file.Write([]byte(a.Contents))
		if err != nil {
			return err
		}
		if n != len(a.Contents) {
			return fmt.Errorf("%s: wrote %d bytes; expected to write %d", a.Name, n,
				len(a.Contents))
		}
	}
	err = file.Close()
	if needPerms != a.Perms && err == nil {
		err = pop.Mos.Chmod(a.Name, FileMode(a.Perms))
	}
	return err
}


type PopOpenFile struct {
	Name string
	Flags int
	Mode FileMode
	Symbol string
}

func (a PopOpenFile) describe() string {
	return "PopOpenFile"
}

func (a PopOpenFile) populate(pop *PopulatorData) error {
	file, err := pop.Mos.OpenFile(a.Name, a.Flags, a.Mode)
	if err != nil {
		return err
	}
	symbol := a.Name
	if len(a.Symbol) > 0 {
		symbol = a.Symbol
	}
	pop.OpenMap[symbol] = file
	return nil
}


type PopDir struct {
	Name string
	Perms uint64
}

func (a PopDir) describe() string {
	return "PopDir"
}

func (a PopDir) populate(pop *PopulatorData) error {
	return pop.Mos.MkdirAll(a.Name, FileMode(a.Perms))
}


type PopLink struct {
	Target, LinkName string
}

func (a PopLink) describe() string {
	return "PopLink"
}

func (a PopLink) populate(pop *PopulatorData) error {
	return pop.Mos.Link(a.Target, a.LinkName)
}


type PopSymlink struct {
	Target, LinkName string
}

func (a PopSymlink) describe() string {
	return "PopSymlink"
}

func (a PopSymlink) populate(pop *PopulatorData) error {
	return pop.Mos.Symlink(a.Target, a.LinkName)
}


type PopFifo struct {
	Name string
	Perms uint64
}

func (a PopFifo) describe() string {
	return "PopFifo"
}

func (a PopFifo) populate(pop *PopulatorData) error {
	return pop.Mos.Mkfifo(a.Name, FileMode(a.Perms))
}


type PopChtimes struct {
	Name string
	Atime time.Time
	Mtime time.Time
}

func (a PopChtimes) describe() string {
	return "PopChtimes"
}

func (a PopChtimes) populate(pop *PopulatorData) error {
	return pop.Mos.Chtimes(a.Name, a.Atime, a.Mtime)
}


type PopSetEnv struct {
	Var, Value string
}

func (a PopSetEnv) describe() string {
	return "PopSetEnv " + a.Var
}

func (a PopSetEnv) populate(pop *PopulatorData) error {
	return pop.Mos.SetEnv(a.Var, a.Value)
}


type PopRunProcess struct {
	Executable string
	Args []string
	Env []string
	Dir string
	ExpectedPid int
}

func (a PopRunProcess) describe() string {
	return "PopRunProcess " + a.Executable
}

func (a PopRunProcess) populate(pop *PopulatorData) error {
	cmd := pop.Mos.Command(a.Executable)
	cmd.Args = a.Args
	cmd.Env = a.Env
	cmd.Dir = a.Dir
	err := cmd.Run()
	if err != nil {
		return err
	}
	newProcess := cmd.ChildProcess()
	pid := newProcess.Getpid()
	if pid != a.ExpectedPid {
		return fmt.Errorf("expected %s to run as PID %d, got %d", a.Executable,
			a.ExpectedPid, pid)
	}
	return nil
}


type PopStartProcess struct {
	Executable, Symbol string
	Args []string
	Env []string
	Dir string
	ExpectedPid int
}

func (a PopStartProcess) describe() string {
	return "PopStartProcess " + a.Executable
}

func (a PopStartProcess) populate(pop *PopulatorData) error {
	cmd := pop.Mos.Command(a.Executable)
	cmd.Args = a.Args
	cmd.Env = a.Env
	cmd.Dir = a.Dir
	err := cmd.Start()
	if err != nil {
		return err
	}
	newProcess := cmd.ChildProcess()
	pid := newProcess.Getpid()
	if pid != a.ExpectedPid {
		return fmt.Errorf("expected %s to run as PID %d, got %d", a.Executable,
			a.ExpectedPid, pid)
	}
	if len(a.Symbol) > 0 {
		pop.CmdMap[a.Symbol] = cmd
	}
	return nil
}


type PopSwitchContext struct {
	Pid int
	Symbol string
}

func (a PopSwitchContext) describe() string {
	return fmt.Sprintf("PopSwitchContext %d / '%s'", a.Pid, a.Symbol)
}

func (a PopSwitchContext) populate(pop *PopulatorData) error {
	pid := a.Pid
	if len(a.Symbol) > 0 {
		if cmd, have := pop.CmdMap[a.Symbol]; !have {
			return fmt.Errorf("unknown process symbol %s", a.Symbol)
		} else {
			pid = cmd.ChildProcess().Getpid()
		}
	}
	if proc, have := pop.Mos.ns.processes[pid]; have {
		pop.Mos = proc
		return nil
	}
	return fmt.Errorf("unknown PID %d", pid)
}


type PopExit struct {
	Symbol string
	Code int
}

func (a PopExit) describe() string {
	return fmt.Sprintf("PopExit %s (rc=%d)", a.Symbol, a.Code)
}

func (a PopExit) populate(pop *PopulatorData) error {
	if len(a.Symbol) == 0 {
		return fmt.Errorf("no symbol given to exit process")
	}
	if cmd, have := pop.CmdMap[a.Symbol]; have {
		cmd.ChildProcess().Exit(a.Code)
		pop.Mos = cmd.CallingProcess()
		return nil
	}
	return fmt.Errorf("unknown process symbol %s", a.Symbol)
}


type PopWait struct {
	Symbol string
}

func (a PopWait) describe() string {
	return "PopWait " + a.Symbol
}

func (a PopWait) populate(pop *PopulatorData) error {
	if len(a.Symbol) == 0 {
		return fmt.Errorf("no symbol given to wait for process")
	}
	if cmd, have := pop.CmdMap[a.Symbol]; have {
		err := cmd.Wait()
		if err != nil {
			return fmt.Errorf("error waiting for process %s to exit: %s", a.Symbol, err)
		}
		pop.Mos = cmd.CallingProcess()
		return nil
	}
	return fmt.Errorf("unknown process symbol %s", a.Symbol)
}


type PopMount struct {
	Source, Mountpoint, Fstype, Options string
	Flags uintptr
}

func (a PopMount) describe() string {
	return "PopMount " + a.Source + " " + a.Mountpoint
}

func (a PopMount) populate(pop *PopulatorData) error {
	return pop.Mos.SyscallMount(a.Source, a.Mountpoint, a.Fstype, a.Flags, a.Options)
}


type PopInsertStorageDevice struct {
	Major, Minor int
	Fstype, Source string
}

func (a PopInsertStorageDevice) describe() string {
	return fmt.Sprintf("PopInsertStorage %d:%d", a.Major, a.Minor)
}

func (a PopInsertStorageDevice) populate(pop *PopulatorData) error {
	return pop.Mos.InsertStorageDevice(a.Major, a.Minor, a.Fstype, a.Source)
}

