// Copyright © 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import (
	"sort"
	"strings"
)


func (cmd *mosCmd) CallingProcess() *MemOS {
	return cmd.mos
}


func (cmd *mosCmd) ChildProcess() *MemOS {
	return cmd.newProcess
}


func (cmd *mosCmd) Run() error {
	err := cmd.Start()
	if err == nil {
		err = cmd.Wait()
	}
	return err
}


func (cmd *mosCmd) Start() error {
	mos := cmd.mos
	abs, err := mos.LookPath(cmd.Path)
	if err != nil {
		return err
	}
	args := cmd.Args
	if args == nil {
		args = []string{abs}
	} else {
		args[0] = abs
	}
	newProcess := &MemOS{umask: initUmask}
	err = mos.ns.addProcess(newProcess, mos.root.mount, mos.root.inode, mos.root.abspath)
	if err != nil {
		return err
	}
	newProcess.groups = mos.groups
	newProcess.environment = cmd.environMap()
	newProcess.Args = args
	file, err := newProcess.OpenFile(abs, O_RDONLY, 0)
	if err != nil {
		newProcess.ns.removeProcess(newProcess)
		return err
	}
	newProcess.hideFD(file)
	if len(cmd.Dir) > 0 {
		newProcess.Chdir(cmd.Dir)
	}
	newProcess.Stdin = newProcess.prepReaderStream(cmd.Stdin)
	newProcess.Stdout = newProcess.prepWriterStream(cmd.Stdout)
	newProcess.Stderr = newProcess.prepWriterStream(cmd.Stderr)
	cmd.newProcess = newProcess
	return nil
}


func (cmd *mosCmd) Wait() error {
	newProcess := cmd.newProcess
	if newProcess == nil || newProcess.ns == nil &&
			!newProcess.ns.processIsRegistered(newProcess) {
		return ECHILD
	}
	newProcess.closeAll()
	return newProcess.ns.removeProcess(newProcess)
}



func (cmd *mosCmd) environMap() map[string]string {
	if cmd.Env == nil {
		return cmd.mos.environment
	}
	return EnvironmentStringsToMap(cmd.Env)
}


func EnvironmentStringsToMap(env []string) map[string]string {
	emap := make(map[string]string, len(env))
	for _, item := range env {
		pos := strings.IndexByte(item, '=')
		if pos < 1 {
			continue
		}
		emap[item[:pos]] = item[pos+1:]
	}
	return emap
}


func EnvironmentMapToStrings(em map[string]string) []string {
	env := make([]string, len(em))
	i := 0
	for key, value := range em {
		env[i] = key + "=" + value
		i++
	}
	sort.Strings(env)
	return env
}

