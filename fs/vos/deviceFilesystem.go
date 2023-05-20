// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import "strings"


// Mount and inode operations for devtmpfs


const (
	deviceFS_root_ino = 1
	devNull_st_rdev = (1 << 8) + 3
)

type deviceFilesystem struct {
	baseFilesystemData
}


type devDirInode struct {
	mfsDirInodeBase
	ns *namespaceType
	devfs *deviceFilesystem
	prefix string
}


func newDeviceFilesystem(st_dev uint64, fstype, source string, ns *namespaceType,
		) (filesystemInstance, error) {
	fs := &deviceFilesystem{
		baseFilesystemData{
			fstype: fstype,
			source: source,
			st_dev: st_dev,
			rootIno: deviceFS_root_ino,
			inodes: []inodeType{nil},
		},
	}
	inode, _ := fs.addInode(nil, "", newDevDirInode(ns, fs, "/dev/"))
	inode.applyUmask(initUmask)
	nullStream := nullStreamer(0)
	fs.addInode(inode.(dirInodeType), "null",
		newBaseChardevInode(devNull_st_rdev, nullStream, nullStream))
	return fs, nil
}


func newDevDirInode(ns *namespaceType, devfs *deviceFilesystem, prefix string) *devDirInode {
	inode := &devDirInode{ns: ns, devfs: devfs, prefix: prefix}
	inode.entries = map[string]inodeType{}
	inode.init(nodeTypeDir)
	return inode
}


func (ino *devDirInode) direntByName(name string) inodeType {
	if len(name) == 0 {
		return nil
	}
	if target, exists := ino.entries[name]; exists {
		return target
	}
	blkdevs := ino.getBlockDeviceMap()
	st_dev, exists := blkdevs[name]
	if !exists {
		return nil
	}
	if st_dev > 0 {
		return newBaseBlockdevInode(st_dev)
	}
	dirInode := newDevDirInode(ino.ns, ino.devfs, ino.prefix + name + "/")
	dirInode.applyUmask(0022)
	ino.devfs.addInode(ino, name, dirInode)
	return dirInode
}


func (ino *devDirInode) getBlockDeviceMap() map[string]uint64 {
	devicemap := make(map[string]uint64, len(ino.ns.devices))
	for _, instance := range ino.ns.devices {
		st_dev := instance.getStDev()
		name := instance.getSource()
		if strings.HasPrefix(name, ino.prefix) {
			name = name[len(ino.prefix):]
			pos := strings.IndexByte(name, '/')
			if pos < 0 {
				devicemap[name] = st_dev
			} else {
				devicemap[name[:pos]] = 0
			}
		}
	}
	return devicemap
}


type nullStreamer int

func (n nullStreamer) Read(buf []byte) (int, error) {
	return 0, nil
}

func (n nullStreamer) Write(buf []byte) (int, error) {
	return 0, nil
}

