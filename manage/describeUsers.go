// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package manage

import (
	"fmt"
	"sort"
	"potano.layercake/fs"
	"potano.layercake/fns"
)


type procDataType struct {
	pid uint
	inChroot, inLayer bool
	command, cwd string
	files []string
}


func (ld *Layerdefs) DescribeUsers(procs []fs.InUseProc, tbl *fns.AdaptiveTable) {
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].Pid < procs[j].Pid ||
			(procs[i].Pid == procs[j].Pid && procs[i].UsedAs < procs[j].UsedAs)
	})
	procData := procDataType{}
	for _, proc := range procs {
		if proc.Pid != procData.pid {
			procData.flush(tbl)
			procData = procDataType{pid: proc.Pid, command: proc.ProgName}
		}
		switch proc.UsedAs {
		case fs.UsedAs_root:
			procData.inChroot = true
		case fs.UsedAs_cwd:
			procData.inLayer = true
			procData.cwd = proc.File
		case fs.UsedAs_open:
			procData.files = append(procData.files, proc.File)
		}
	}
	procData.flush(tbl)
}


func (p procDataType) flush(tbl *fns.AdaptiveTable) {
	if p.pid < 1 {
		return
	}
	command := fmt.Sprintf("%s(%d)", p.command, p.pid)
	var details string
	if p.inChroot {
		details = "running in chroot; cwd=" + p.cwd
	} else if p.inLayer {
		details = "running in layer directory " + p.cwd
	} else {
		details = "opened files/directories in layer"
	}
	tbl.Print(command, details)
	for _, f := range p.files {
		tbl.Print("", "open file: " + f)
	}
}

