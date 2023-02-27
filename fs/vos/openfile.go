// Copyright © 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import (
	"sort"
	"time"
)


func (of *mfsOpenFile) Close() error {
	if of == nil || of.inode == nil {
		return ErrInvalid
	}
	mount := of.mount
	mos := of.mos
	pid := mos.pid
	fd := of.fd
	mtKey := pidFdType{pid, fd}
	if (mount != nil && mount.openFiles[mtKey] != of) || mos.openFiles[fd] != of {
		panic("corrupted open file")
	}
	of.inode.close(of)
	if mount != nil {
		delete(mount.openFiles, mtKey)
	}
	delete(mos.openFiles, fd)
	of.inode = nil
	return nil
}


func (of *mfsOpenFile) Read(buf []byte) (int, error) {
	if of == nil || of.inode == nil {
		return 0, ErrInvalid
	}
	if !of.readable {
		return 0, of.pathError("read", EACCES)
	}
	if fileInode, isFile := of.inode.(fileInodeType); isFile {
		pos := of.pos
		if pos < 0 {
			pos = 0
		}
		n, err := fileInode.readFile(buf, pos)
		if err != nil {
			return 0, of.pathError("read", err)
		}
		of.pos = pos + int64(n)
		return n, nil
	} else if of.inode.nodeType() == nodeTypeDir {
		return 0, of.pathError("read", EISDIR)
	} else if fifoInode, isFifo := of.inode.(fifoInodeType); isFifo {
		return fifoInode.readFifo(buf)
	}
	return 0, nil
}


func (of *mfsOpenFile) Readdirnames(n int) ([]string, error) {
	if of == nil || of.inode == nil {
		return nil, ErrInvalid
	}
	if !of.readable {
		return nil, of.pathError("readdirnames", EACCES)
	}
	if dirInode, isDir := of.inode.(dirInodeType); isDir {
		dirents := dirInode.direntMap()
		names := make([]string, len(dirents))
		i := 0
		for name := range dirents {
			names[i] = name
			i++
		}
		sort.Strings(names)
		if n < 0 {
			return names, nil
		}
		pos := of.pos
		lastOne := pos + int64(n)
		if lastOne > int64(len(names)) {
			lastOne = int64(len(names))
		}
		of.pos = lastOne
		return names[pos:lastOne], nil
	}
	return nil, of.pathError("readdirnames", ENOTDIR)
}


func (of *mfsOpenFile) Seek(pos int64, whence int) (int64, error) {
	if of == nil || of.inode == nil {
		return -1, ErrInvalid
	}
	if !of.inode.isSeekable() {
		return -1, of.pathError("seek", ESPIPE)
	}
	newpos := of.pos
	switch whence {
	case SEEK_SET:
		newpos = pos
	case SEEK_CUR:
		newpos += pos
	case SEEK_END:
		newpos = of.inode.size() + pos
	default:
		return -1, of.pathError("seek", EINVAL)
	}
	if newpos < 0 {
		return -1, of.pathError("seek", EINVAL)
	}
	of.pos = newpos
	return newpos, nil
}


func (of *mfsOpenFile) Stat() (FileInfo, error) {
	fileinfo := mfsFileInfo{}
	if of == nil || of.inode == nil {
		return fileinfo, ErrInvalid
	}
	err := of.inode.Stat(&fileinfo.stat)
	return fileinfo, err
}


func (of *mfsOpenFile) Write(buf []byte) (int, error) {
	if of == nil || of.inode == nil {
		return 0, ErrInvalid
	}
	if !of.writable {
		return 0, of.pathError("write", EACCES)
	}
	if fileInode, isFile := of.inode.(fileInodeType); isFile {
		pos := of.pos
		if (of.flags & O_APPEND) > 0 {
			pos = fileInode.size()
		} else if pos < 0 {
			pos = 0
		}
		n, err := fileInode.writeFile(buf, pos)
		if err != nil {
			return n, of.pathError("write", err)
		}
		of.pos = pos + int64(n)
		return n, nil
	} else if fifoInode, isFifo := of.inode.(fifoInodeType); isFifo {
		return fifoInode.writeFifo(buf)
	}
	return 0, nil
}



func (of *mfsOpenFile) pathError(op string, err error) error {
	return &PathError{op, of.name, err}
}



// Satisifies the FileInfo interface

type mfsFileInfo struct {
	mos *mfsOpenFile
	stat Stat_t
}

func (s mfsFileInfo) Name() string {
	return s.mos.name
}

func (s mfsFileInfo) Size() int64 {
	return s.stat.Size
}

func (s mfsFileInfo) Mode() FileMode {
	return FileMode(s.stat.Mode)
}

func (s mfsFileInfo) ModTime() time.Time {
	return timespecToTime(s.stat.Mtim)
}

func (s mfsFileInfo) IsDir() bool {
	return s.mos.inode.isDir()
}

func (s mfsFileInfo) Sys() interface{} {
	return s.stat
}

