// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package stage

import (
	"fmt"
	"path"
	"time"
	"bytes"
	unix "syscall"
	"potano.layercake/fs"
	"potano.layercake/defaults"
	"potano.layercake/portage/vdb"
)


type devIno struct {
	dev, ino uint64
}


func (fl *FileList) addSingleFile(info lineInfo) error {
	source := info.source
	nameIsSource := len(source) == 0
	if nameIsSource {
		source = path.Join(fl.rootDir, info.name)
		info.source = source
	}

	needLtypeCheck := nameIsSource
	switch info.ltype {
	case vdb.FileType_dir:
		needLtypeCheck = false
	case vdb.FileType_symlink:
		needLtypeCheck = len(info.target) == 0
	case vdb.FileType_device:
		needLtypeCheck = ! nameIsSource || ! info.hasDev
	}

	info.devino = -1
	var statbuf unix.Stat_t
	exists := true
	if err := unix.Lstat(source, &statbuf); err != nil {
		if err.Error() == "no such file or directory" {
			exists = false
		} else {
			return err
		}
	}

	var actualLtype uint8
	var err error
	if exists {
		switch statbuf.Mode & unix.S_IFMT {
		case unix.S_IFSOCK:
			err = fmt.Errorf("cannot tar socket %s", info.name)
		case unix.S_IFLNK:
			actualLtype = vdb.FileType_symlink
		case unix.S_IFREG:
			actualLtype = vdb.FileType_file
		case unix.S_IFBLK, unix.S_IFCHR:
			actualLtype = vdb.FileType_device
		case unix.S_IFDIR:
			actualLtype = vdb.FileType_dir
		case unix.S_IFIFO:
			err = fmt.Errorf("cannot tar FIFO %s", info.name)
		default:
			return fmt.Errorf("%s has unknown type bits 0%o", info.name,
				statbuf.Mode & unix.S_IFMT)
		}
	} else if info.skipIfAbsent {
		return nil
	}

	if info.ltype == vdb.FileType_none {
		if !exists {
			return fmt.Errorf("no file found to determine type of %s", info.name)
		} else if err != nil {
			return err
		}
		info.ltype = actualLtype
	} else if needLtypeCheck {
		if err != nil {
			return err
		} else if exists && info.ltype != actualLtype {
			return fmt.Errorf("%s has file type %s; expected %s", info.name,
				ltypeName(actualLtype), ltypeName(info.ltype))
		}
	}

	perms := statbuf.Mode
	if !exists {
		perms = defaults.Umask
	}
	if info.hasPerm {
		if info.andMask > 0 {
			perms = (perms & info.andMask) | info.orMask
		} else {
			perms = info.orMask
		}
	}
	info.orMask = perms

	if !info.hasGid {
		if !exists {
			info.gid = defaults.StageFileGID
		} else {
			info.gid = statbuf.Gid
		}
	}

	if !info.hasUid {
		if !exists {
			info.uid = defaults.StageFileUID
		} else {
			info.uid = statbuf.Uid
		}
	}

	if exists {
		info.unixTime = statbuf.Mtim.Sec
		info.xattrs = getXattrs(source)
	} else {
		info.unixTime = time.Now().Unix()
	}

	switch info.ltype {
	case vdb.FileType_dir:
	case vdb.FileType_file:
		if !exists {
			return fmt.Errorf("file %s does not exist (source of %s)", source,
				info.name)
		}
		if nameIsSource && statbuf.Nlink > 1 {
			id := devIno{statbuf.Dev, statbuf.Ino}
			if index, exists := fl.inodes[id]; exists {
				info.devino = index
			} else {
				index = int32(len(fl.inodes))
				fl.inodes[id] = index
				info.devino = index
			}
		}
		info.fsize = statbuf.Size
	case vdb.FileType_symlink:
		if len(info.target) == 0 {
			info.target, err = fs.Readlink(source)
			if err != nil {
				return err
			}
		}
	case vdb.FileType_device:
		if !info.hasDev {
			if !exists {
				return fmt.Errorf("device %s does not exist (source of %s)", source,
					info.name)
			}
			if statbuf.Mode & unix.S_IFCHR > 0 {
				info.devtype = 'c'
			} else if statbuf.Mode & unix.S_IFBLK > 0 {
				info.devtype = 'b'
			} else {
				return fmt.Errorf("expected %s to be a device node", source)
			}
			info.major = uint32(statbuf.Rdev >> 8)
			info.minor = uint32(statbuf.Rdev & 0xFF)
		}
	default:
		return fmt.Errorf("assertion error: unknown file type %d for %s", info.ltype,
			info.name)
	}

	fl.entryMap[info.name] = info
	return nil
}


func getXattrs(filename string) map[string]string {
	namebuf := make([]byte, 256)
	sz, err := unix.Listxattr(filename, namebuf)
	if sz > cap(namebuf) {
		namebuf = make([]byte, sz+1)
		sz, err = unix.Listxattr(filename, namebuf)
	}
	if err != nil {
		return nil
	}
	xattrs := map[string]string{}
	value := make([]byte, 1024)
	for _, nm := range bytes.Split(namebuf[:sz], []byte{0}) {
		name := string(nm)
		sz, err := unix.Getxattr(filename, name, value)
		if sz > cap(value) {
			value = make([]byte, sz+1)
			sz, err = unix.Getxattr(filename, name, value)
		}
		if err != nil {
			continue
		}
		xattrs[name] = string(value[:sz])
	}
	return xattrs
}


var ftNames []string = []string{
	"undetermined",	// FileType_none
	"directory",	// FileType_dir
	"regular file",	// FileType_file
	"symlink",	// FileType_symlink
	"hard link",	// FileType_hardlink
	"device node",	// FileType_device
}


func ltypeName(ltype uint8) string {
	if ltype < uint8(len(ftNames)) {
		return ftNames[ltype]
	}
	return fmt.Sprintf("unknown type %d", ltype)
}


func (fl *FileList) fixHardlinks() {
	inodeMap := map[int32]string{}
	for i, file := range fl.Files {
		if file.devino < 0 {
			continue
		}
		targ, exists := inodeMap[file.devino]
		if exists {
			file.ltype = vdb.FileType_hardlink
			file.target = targ
			fl.Files[i] = file
		} else {
			inodeMap[file.devino] = file.name
		}
	}
}

