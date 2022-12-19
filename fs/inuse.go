// Copyright © 2017, 2021 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package fs

import (
	"os"
	"path"
	"strings"
	"strconv"
	"syscall"
)

const (
	UsedAs_root = iota
	UsedAs_cwd
	UsedAs_exec
	UsedAs_open
)

type InUseProc struct {
	Pid, UsedAs uint
	ProgName, File string
}

type InUseLayerMap map[string][]InUseProc


func FindLayerUsers(prefix string) (InUseLayerMap, error) {
	if len(prefix) > 1 && prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	fh, err := os.Open("/proc")
	if nil != err {
		return nil, err
	}
	defer fh.Close()
	lst, err := fh.Readdir(-1)
	if nil != err {
		return nil, err
	}
	if nil == lst {
		return nil, err
	}
	out := InUseLayerMap{}
	for _, entry := range lst {
		pidString := entry.Name()
		if !entry.IsDir() || !isNumeric(pidString) {
			continue
		}
		pid, _ := strconv.ParseUint(pidString, 10, 64)
		proc := "/proc/" + pidString + "/"
		progName, err := Readlink(proc + "exe")
		if err != nil {
			if err != syscall.ENOENT {
				return nil, err
			}
			progName = "[anon]"
		} else {
			progName = path.Base(progName)
		}
		for _, item := range []struct{path string; mask uint}{
			{"cwd", UsedAs_cwd}, {"root", UsedAs_root}, {"exe", UsedAs_exec}} {
			path := proc + item.path
			if layername, tail, ok := isLinkToLayer(prefix, path); ok {
				entry := InUseProc{
					Pid: uint(pid),
					UsedAs: item.mask,
					ProgName: progName,
					File: tail}
				if _, exists := out[layername]; exists {
					out[layername] = append(out[layername], entry)
				} else {
					out[layername] = []InUseProc{entry}
				}
			}
		}
		if !IsDir(proc + "fd") {
			continue
		}
		fdh, err := os.Open(proc + "fd")
		if nil != err {
			continue
		}
		items, err := fdh.Readdir(-1)
		fdh.Close()
		if nil != err {
			return nil, err
		}
		if nil == items {
			continue
		}
		for _, entry := range items {
			name := proc + "fd/" + entry.Name()
			if layername, tail, ok := isLinkToLayer(prefix, name); ok {
				entry := InUseProc{
					Pid: uint(pid),
					UsedAs: UsedAs_open,
					ProgName: progName,
					File: tail}
				if _, exists := out[layername]; exists {
					out[layername] = append(out[layername], entry)
				} else {
					out[layername] = []InUseProc{entry}
				}
			}
		}
	}
	return out, nil
}


func isNumeric(str string) bool {
	for _, c := range str {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}


func isLinkToLayer(prefix, path string) (layername, tail string, have bool) {
	target, err := Readlink(path)
	if nil != err {
		return
	}
	if SameDirectoryOrDescendant(target, prefix) {
		layername = target[len(prefix):]
		have = true
		p := strings.Index(layername, "/")
		if p > -1 {
			tail = layername[p+1:]
			layername = layername[:p]
		}
	}
	return
}


func SameDirectoryOrDescendant(path, prefix string) bool {
	lenPrefix := len(prefix)
	return strings.HasPrefix(path, prefix) &&
		(prefix[lenPrefix-1] == '/' || len(path) == lenPrefix || path[lenPrefix] == '/')
}

