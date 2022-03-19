// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package manage

import (
	"fmt"
	"path"
	"errors"
	"strings"
	"syscall"
)

type mountNode struct {
	root, mountpoint, st_dev, fstype, options string
	index, mounted_in int
}

type mountNinja struct {
	nextIndex, nextMinor int
	nodes []mountNode
	extras map[string]string
}

func newMountNinja() *mountNinja {
	mn := &mountNinja{
		nextMinor: 30,
		nodes: []mountNode{
			{"/", "/", "8:1", "ext4", "rw", 1, 0},
			{"/", "/proc", "0:4", "proc", "rw", 2, 1},
			{"/", "/sys", "0:16", "sysfs", "rw", 3, 1},
			{"/", "/dev", "0:6", "devtmpfs", "rw", 4, 1},
			{"/", "/dev/pts", "0:17", "devpts", "rw", 5, 4},
			{"/", "/dev/shm", "0:18", "tmpfs", "rw", 6, 4},
			{"/", "/run", "0:19", "tmpfs", "rw", 7, 1},
			{"/", "/sys/kernel/security", "0:20", "securityfs", "rw", 8, 3},
		},
		extras: map[string]string{
			"0:6": "pts shm",
			"0:16": "kernel/security",
		},
	}
	mn.nextIndex = len(mn.nodes) + 1
	return mn
}

func (mn *mountNinja) mount(source, mtpoint, fstype string, flgs uintptr, options string) error {
	var st_dev, root string
	var parentID int
	onlyModifying := (flgs & (syscall.MS_REMOUNT | syscall.MS_SHARED | syscall.MS_PRIVATE |
		syscall.MS_SLAVE | syscall.MS_UNBINDABLE)) > 0
	if fstype == "overlay" {
		st_dev = fmt.Sprintf("0:%d", mn.nextMinor)
		mn.nextMinor++
		root = "/"
	} else if !onlyModifying {
		if fstype == "proc" {
			source = "/proc"
		}
		max_match := 0
		srclen := len(source)
		for _, mnt := range mn.nodes {
			mntlen := len(mnt.mountpoint)
			if srclen >= mntlen && mnt.mountpoint == source[:mntlen] &&
				(srclen == mntlen || mntlen < 2 || source[mntlen] == '/') {
				if mntlen > max_match {
					st_dev = mnt.st_dev
					fstype = mnt.fstype
					if srclen == mntlen {
						root = "/"
					} else {
						root = source[mntlen:]
					}
					max_match = mntlen
				}
			}
		}
		if max_match == 0 {
			return errors.New("no such device")
		}
	}
	max_match := 0
	newlen := len(mtpoint)
	for _, mnt := range mn.nodes {
		mntlen := len(mnt.mountpoint)
		if newlen >= mntlen && mnt.mountpoint == mtpoint[:mntlen] &&
			(newlen == mntlen || mntlen < 2 || mtpoint[mntlen] == '/') {
			if mntlen > max_match {
				parentID = mnt.index
				max_match = mntlen
			}
		}
	}
	if max_match == 0 {
		return errors.New("no such file or directory")
	}
	if onlyModifying {
		return nil
	}
	mn.nodes = append(mn.nodes, mountNode{root, mtpoint, st_dev, fstype, options,
		mn.nextIndex, parentID})
	mn.nextIndex++
	extras := mn.extras[st_dev]
	if len(extras) > 0 {
		for _, pt := range strings.Split(extras, " ") {
			mn.mount(path.Join(source, pt), path.Join(mtpoint, pt), fstype, 0, options)
		}
	}
	return nil
}

func (mn *mountNinja) mountFromDeviceNode(st_dev, mtpoint, fstype, options string) error {
	nodeX := len(mn.nodes)
	if err := mn.mount("/", mtpoint, fstype, 0, options); err != nil {
		return err
	}
	mn.nodes[nodeX].st_dev = st_dev
	mn.nodes[nodeX].fstype = fstype
	return nil
}

func (mn *mountNinja) unmount(mtpoint string, flags int) error {
	var mnt mountNode
	for _, mnt = range mn.nodes {
		if mnt.mountpoint == mtpoint {
			break
		}
	}
	if len(mnt.mountpoint) == 0 {
		return errors.New("invalid argument")
	}
	var submounts []string
	for _, mt := range mn.nodes {
		if mt.mounted_in == mnt.index {
			submounts = append(submounts, mt.mountpoint)
		}
	}
	if len(submounts) > 0 {
		if flags == 0 {
			return errors.New("device or resource busy")
		}
		for _, mp := range submounts {
			mn.unmount(mp, flags)
		}
	}
	for i, mt := range mn.nodes {
		if mt.mountpoint == mtpoint {
			mn.nodes = append(mn.nodes[:i], mn.nodes[i+1:]...)
			break
		}
	}
	return nil
}

func (mn *mountNinja) mountinfo() string {
	var lines []string
	for _, mnt := range mn.nodes {
		lines = append(lines, fmt.Sprintf("%d %d %s %s %s default - %s %s %s",
			mnt.index, mnt.mounted_in, mnt.st_dev, mnt.root, mnt.mountpoint,
			mnt.fstype, mnt.fstype, mnt.options))
	}
	return strings.Join(lines, "\n")
}

func (mn *mountNinja) is_mountpoint(mtpoint string) bool {
	for _, mnt := range mn.nodes {
		if mnt.mountpoint == mtpoint {
			return true
		}
	}
	return false
}

