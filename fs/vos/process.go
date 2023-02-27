// Copyright © 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos


import (
	"io"
	"time"
	"strings"
	"syscall"
)


/* Low-level operations relating to a process, including filesystem operations */


const initPid = 1
const defaultInitialPid = 100
const initUmask = 0022           // as hardcoded in fs/fs_struct.c in the kernel sources
const maxSymlinks = 40           // as declared in include/linux/namei.h in the kernel sources
const defaultCreateMode = 0666   //from Golang standard library documentation

const (
	maskWantReadAccess = 1
	maskWantWriteAccess = 2
)

func (mos *MemOS) hasUserAccess(uid uint64) bool {
	return uid == mos.euid || uid == 0
}


func (mos *MemOS) hasGroupAccess(gid uint64) bool {
	if gid == mos.gid || mos.euid == 0 {
		return true
	}
	return mos.processHasGroup(gid)
}


func (mos *MemOS) processHasGroup(gid uint64) bool {
	for _, grp := range mos.groups {
		if grp == gid {
			return true
		}
	}
	return false
}


func (mos *MemOS) hasReadPermission(inode inodeType) bool {
	return inode.hasReadPermission(
		mos.hasUserAccess(inode.uid()),
		mos.hasGroupAccess(inode.gid()))
}


func (mos *MemOS) hasWritePermission(inode inodeType) bool {
	return inode.hasWritePermission(
		mos.hasUserAccess(inode.uid()),
		mos.hasGroupAccess(inode.gid()))
}


func (mos *MemOS) hasExecutePermission(inode inodeType) bool {
	return inode.hasExecutePermission(
		mos.hasUserAccess(inode.uid()),
		mos.hasGroupAccess(inode.gid()))
}


func (mos *MemOS) openInode(mt *mountType, inode inodeType, name string, flags int,
		) (*mfsOpenFile, error) {
	fd := 0
	for i := range mos.openFiles {
		if fd <= i {
			fd = i + 1
		}
	}

	userOK := mos.hasUserAccess(inode.uid())
	groupOK := mos.hasGroupAccess(inode.gid())
	readable := inode.hasReadPermission(userOK, groupOK)
	writable := inode.hasWritePermission(userOK, groupOK)
	executable := inode.hasExecutePermission(userOK, groupOK)

	access := (flags & O_ACCMODE) + 1	// Map the silly 0, 1, 2 access-mode values into
						// a useful bitmask
	wantReadAccess := (access & maskWantReadAccess) > 0
	wantWriteAccess := (access & maskWantWriteAccess) > 0
	if (wantReadAccess && !readable) || (wantWriteAccess && !writable) {
		   return nil, EACCES
	}
	if inode.isDir() && wantWriteAccess {
		return nil, EISDIR
	}
	of := &mfsOpenFile{
		mount: mt,
		inode: inode,
		mos: mos,
		fd: fd,
		flags: flags,
		name: name,
		readable: readable && wantReadAccess,
		writable: writable && wantWriteAccess,
		executable: executable,
	}
	inode.open(of)
	if mt != nil {
		mtKey := pidFdType{mos.pid, fd}
		if _, exists := mt.openFiles[mtKey]; exists {
			return nil, EINVAL
		}
		mt.openFiles[mtKey] = of
	}
	mos.openFiles[fd] = of
	return of, nil
}


func (mos *MemOS) hideFD(of *mfsOpenFile) error {
	return mos.changeOpenFileFD(of.fd, mos.nextNegativeFD())
}


func (mos *MemOS) nextNegativeFD() int {
	fd := -1
	for i := range mos.openFiles {
		if i <= fd {
			fd = i - 1
		}
	}
	return fd
}


func (mos *MemOS) changeOpenFileFD(oldFD, newFD int) error {
	of := mos.openFiles[oldFD]
	if of == nil {
		return ENOENT
	}
	if _, have := mos.openFiles[newFD]; have {
		return EINVAL
	}
	of.fd = newFD
	delete(mos.openFiles, oldFD)
	mos.openFiles[newFD] = of
	delete(of.mount.openFiles, pidFdType{mos.pid, oldFD})
	of.mount.openFiles[pidFdType{mos.pid, newFD}] = of
	return nil
}


func (mos *MemOS) inodeAtPath(relDir *mfsOpenFile, pathname string) (*mountType, inodeType, error) {
	mount, _, inode, _, err := mos.resolvePath(relDir, pathname, false)
	return mount, inode, err
}


func (mos *MemOS) inodeAtPathWithParent(relDir *mfsOpenFile, pathname string,
		) (*mountType, dirInodeType, inodeType, string, error) {
	return mos.resolvePath(relDir, pathname, true)
}


func (mos *MemOS) mfsOpenFileAtPath(pathname string) (*mfsOpenFile, error) {
	mount, _, inode, name, err := mos.inodeAtPathWithParent(mos.cwd, pathname)
	return &mfsOpenFile{
		mount: mount,
		inode: inode,
		mos: mos,
		name: name}, err
}


func (mos *MemOS) resolvePath(relDir *mfsOpenFile, pathname string, asLstat bool,
		) (mount *mountType, dirInode dirInodeType, inode inodeType, name string, err error,
		) {
	if len(pathname) == 0 {
		err = ENOENT
		return
	}
	path := parsePath(pathname)
	if path[0] == "/" {
		relDir = mos.root
		path = path[1:]
	}
	mount = relDir.mount
	inode = relDir.inode
	for len(path) > 0 {
		name = ""
		var isDir bool
		dirInode, isDir = inode.(dirInodeType)
		if !isDir {
			err = ENOTDIR
			return
		}
		part0 := path[0]
		path = path[1:]
		if part0 == "." {
			continue
		}
		if part0 == ".." {
			mount, inode, err = mos.toParentDirectory(mount, inode)
			if err != nil {
				return
			}
			continue
		}
		name = part0
		inode = dirInode.direntByName(part0)
		if inode == nil {
			if len(path) > 0 || !asLstat {
				err = ENOENT
			}
			return
		}
		if linkInode, is := inode.(linkInodeType); is {
			if asLstat && len(path) == 0 {
				return
			}
			mount, dirInode, inode, name, err = mos.resolvePath(
				&mfsOpenFile{mount: mount, inode: dirInode},
				linkInode.getLinkTarget(), false)
			if err != nil {
				return
			}
		}
		if _, is := inode.(dirInodeType); is {
			for mount.mountpoints[inode.ino()] != nil {
				mount = mount.mountpoints[inode.ino()]
				inode = mos.ns.devices[mount.st_dev].rootInode()
				dirInode = nil
			}
			if len(path) > 0 && !inode.hasExecutePermission(
					mos.hasUserAccess(inode.uid()),
					mos.hasGroupAccess(inode.gid())) {
				err = EACCES
				return
			}
		}
	}
	return
}


func (mos *MemOS) toParentDirectory(mount *mountType, inode inodeType,
		) (*mountType, inodeType, error) {
	origMount := mount
	origInode := inode
	root_st_dev := mos.root.mount.st_dev
	root_st_ino := mos.root.inode.(*mfsInodeBase).st_ino
	for {
		baseInode, isDir := inode.(*mfsDirInodeBase)
		if !isDir {
			return mount, inode, ENOTDIR
		}
		if baseInode.st_dev == root_st_dev && baseInode.st_ino == root_st_ino {
			return origMount, origInode, nil
		}
		if baseInode.st_ino != mount.root_ino && baseInode.parent_ino != 0 {
			inode = mos.ns.devices[mount.st_dev].inodeByInum(baseInode.parent_ino)
			break
		} else if mount.mounted_in == nil || mount.mounted_in == mount {
			return origMount, origInode, nil
		}
		mounted_in_ino := mount.mounted_in_ino
		mount = mount.mounted_in
		inode = mos.ns.devices[mount.st_dev].inodeByInum(mounted_in_ino)
	}
	if !inode.hasExecutePermission(
			mos.hasUserAccess(inode.uid()),
			mos.hasGroupAccess(inode.gid())) {
		return mount, inode, EACCES
	}
	return mount, inode, nil
}


func (mos *MemOS) findAbsolutePath(mount *mountType, parentInode dirInodeType,
		inode inodeType) (string, error) {
	root_st_dev := mos.root.mount.st_dev
	root_st_ino := mos.root.inode.ino()
	path := []string{}
	for parentInode != nil { 
		found := false
		for name, tst := range parentInode.direntMap() {
			if tst == inode {
				path = append(path, name)
				found = true
				break
			}
		}
		if !found {
			return "", ENOENT
		}
		inode = parentInode
		for parentInode != nil {
			if parentInode.dev() == root_st_dev && parentInode.ino() == root_st_ino {
				parentInode = nil
				break
			}
			if parentInode.ino() == mount.root_ino {
				if mount.mounted_in == nil || mount.mounted_in == mount {
					parentInode = nil
					break
				}
				mounted_in_ino := mount.mounted_in_ino
				mount = mount.mounted_in
				inode = mount.inodeByInum(mounted_in_ino)
				parentInode = inode.(dirInodeType)
			} else {
				parentInode = mount.inodeByInum(parentInode.getParent()).
					(dirInodeType)
				break
			}
		}
	}
	for i, j := 0, len(path) - 1; i < j; {
		path[i], path[j] = path[j], path[i]
		i++
		j--
	}
	pathString := "/" + strings.Join(path, "/")
	return pathString, nil
}


func parsePath(name string) []string {
	path := strings.Split(name, "/")
	out := 0
	movesForward := 0
	for in := 0; in < len(path); in++ {
		elem := path[in]
		if len(elem) == 0 {
			if in > 0 || len(path) == 1 {
				continue
			}
			elem = "/"
		} else if elem == "." && in > 0 && in < len(path) - 1 {
			continue
		} else if elem == ".." {
			if movesForward > 0 {
				out--
				movesForward--
				if out > 0 {
					continue
				}
				elem = "."
			}
		} else {
			movesForward++
		}
		path[out] = elem
		out++
	}
	return path[:out]
}


func (mos *MemOS) open(filename string, flags int, mode uint64) (*mfsOpenFile, error) {
	numSymlinks := 0
	mount, dirInode, inode, name, err := mos.inodeAtPathWithParent(mos.cwd, filename)
	symlinkLoop:
	if err != nil {
		return nil, err
	}
	if inode == nil {
		if (flags & O_CREATE) == 0 {
			return nil, ENOENT
		}
		if !mos.hasWritePermission(dirInode) {
			return nil, EACCES
		}
		inode, err = mount.addInode(dirInode, name, newBaseFileInode(nil))
		if err != nil {
			return nil, err
		}
		inode.setPerms(uint64(mode))
		inode.applyUmask(mos.umask)
	} else if (flags & (O_CREATE | O_EXCL)) == (O_CREATE | O_EXCL) {
		return nil, EEXIST
	} else if linkInode, isLink := inode.(linkInodeType); isLink {
		numSymlinks++
		if numSymlinks >= maxSymlinks {
			return nil, ELOOP
		}
		mount, dirInode, inode, name, err = mos.inodeAtPathWithParent(
			&mfsOpenFile{mount: mount, inode: dirInode},
			linkInode.getLinkTarget())
		goto symlinkLoop
	}
	of, err := mos.openInode(mount, inode, filename, flags)
	if err != nil {
		return nil, err
	}
	if of.writable && (flags & O_TRUNC) > 0 {
		if fileInode, isFile := inode.(fileInodeType); isFile {
			fileInode.truncateFile()
		}
	}
	pathname, err := mos.findAbsolutePath(mount, dirInode, inode)
	of.abspath = pathname
	return of, err
}


func (mos *MemOS) mkdir(dirname string, perm FileMode) error {
	mount, dirInode, inode, name, err := mos.inodeAtPathWithParent(mos.cwd, dirname)
	if err != nil {
		return err
	}
	if inode != nil {
		return EEXIST
	}
	if !mos.hasWritePermission(dirInode) {
		return EACCES
	}
	inode, err = mount.addInode(dirInode, name, newBaseDirInode(""))
	if err != nil {
		return err
	}
	inode.setPerms(uint64(perm))
	inode.applyUmask(mos.umask)
	return nil
}


func (mos *MemOS) mkdirAll(path string, perm FileMode) error {
	_, inode, err := mos.inodeAtPath(mos.cwd, path)
	if err == nil {
		if inode.isDir() {
			return nil
		}
		return ENOTDIR
	}

	// Use algorithm in the Go library: recursively call mkdirall() then mkdir()
	i := len(path)
	for i > 0 && path[i-1] == '/' {
		i--
	}
	j := i
	for j > 0 && path[j-1] != '/' {
		j--
	}
	if j > 1 {
		err = mos.mkdirAll(path[:j-1], perm)
		if err != nil {
			return err
		}
	}
	err = mos.mkdir(path, perm)
	if err != nil {
		_, _, inode, _, err2 := mos.inodeAtPathWithParent(mos.cwd, path)
		if err2 != nil || !inode.isDir() {
			return err
		}
	}
	return nil
}


func (mos *MemOS) symlink(target, linkname string) error {
	if len(target) == 0 {
		return ENOENT
	}
	mount, dirInode, inode, name, err := mos.inodeAtPathWithParent(mos.cwd, linkname)
	if err != nil {
		return err
	}
	if inode != nil {
		return EEXIST
	}
	if !mos.hasWritePermission(dirInode) {
		return EACCES
	}
	inode, err = mount.addInode(dirInode, name, newBaseLinkInode())
	if err != nil {
		return err
	}
	inode.(linkInodeType).setLinkTarget(target)
	return nil
}


func (mos *MemOS) link(oldname, newname string) error {
	oldmount, _, oldInode, _, err := mos.inodeAtPathWithParent(mos.cwd, oldname)
	if err != nil {
		return err
	}
	if oldInode == nil {
		return ENOENT
	}
	if oldInode.isDir() {
		return EPERM
	}
	newmount, dirInode, newInode, name, err := mos.inodeAtPathWithParent(mos.cwd, newname)
	if err != nil {
		return err
	}
	if newmount != oldmount {
		return EXDEV
	}
	if newInode != nil {
		return EEXIST
	}
	if !mos.hasWritePermission(dirInode) {
		return EACCES
	}
	err = dirInode.setDirent(name, oldInode)
	if err != nil {
		return err
	}
	oldInode.incrementNlinks()
	return nil
}


func (mos *MemOS) mkfifo(pathname string, perm FileMode) error {
	mount, dirInode, inode, name, err := mos.inodeAtPathWithParent(mos.cwd, pathname)
	if err != nil {
		return err
	}
	if inode != nil {
		return EEXIST
	}
	if !mos.hasWritePermission(dirInode) {
		return EACCES
	}
	inode, err = mount.addInode(dirInode, name, newBaseFifoInode())
	if err != nil {
		return err
	}
	inode.setPerms(uint64(perm))
	inode.applyUmask(mos.umask)
	return nil
}


func (mos *MemOS) chdir(dirname string) error {
	file, err := mos.open(dirname, O_RDONLY, 0)
	if err != nil {
		return err
	}
	if file.inode == mos.cwd.inode {
		return file.Close()
	}
	err = mos.cwd.Close()
	mos.cwd = file
	if err == nil {
		err = mos.hideFD(file)
	}
	return err
}


func (mos *MemOS) chmod(name string, perm FileMode) error {
	_, inode, err := mos.inodeAtPath(mos.cwd, name)
	if err == nil {
		if !mos.hasUserAccess(inode.uid()) {
			return EPERM
		}
		inode.setPerms(uint64(perm))
	}
	return nil
}


func (mos *MemOS) chown(name string, uid, gid int) error {
	if uid < 0 || gid < 0 {
		return EINVAL
	}
	uid64, gid64 := uint64(uid), uint64(gid)
	_, inode, err := mos.inodeAtPath(mos.cwd, name)
	if err == nil {
		if !mos.hasUserAccess(inode.uid()) ||
			(mos.euid != 0 && (uid64 != mos.euid || !mos.processHasGroup(gid64))) {
			return EPERM
		}
		inode.setUidGid(uid64, gid64)
	}
	return err
}


func (mos *MemOS) chtimes(name string, atime time.Time, mtime time.Time) error {
	_, inode, err := mos.inodeAtPath(mos.cwd, name)
	if err == nil {
		if !mos.hasUserAccess(inode.uid()) {
			return EPERM
		}
		inode.setAtime(atime)
		inode.setMtime(mtime)
	}
	return nil
}


func (mos *MemOS) remove(pathname string) error {
	mount, dirInode, inode, name, err := mos.inodeAtPathWithParent(mos.cwd, pathname)
	if err != nil {
		return err
	}
	if inode == nil {
		return ENOENT
	}
	if !mos.hasWritePermission(dirInode) {
		return EACCES
	}
	if asDirInode, isDir := inode.(dirInodeType); isDir {
		if mount.mountpoints[inode.ino()] != nil || inode == mos.root.inode {
			return EBUSY
		}
		if len(asDirInode.direntMap()) > 0 {
			return ENOTEMPTY
		}
	}
	inode.decrementNlinks()
	dirInode.setDirent(name, nil)
	return nil
}


func (mos *MemOS) removeAll(path string) error {
	_, _, inode, _, err := mos.inodeAtPathWithParent(mos.cwd, path)
	if err != nil {
		return err
	}
	if inode == nil {
		return nil
	}
	if asDirInode, isDir := inode.(dirInodeType); isDir {
		for childName := range asDirInode.direntMap() {
			err = mos.removeAll(path + "/" + childName)
			if err != nil {
				return err
			}
		}
	}
	return mos.remove(path)
}


func (mos *MemOS) closeAll() {
	for _, of := range mos.openFiles {
		of.Close()
	}
}


func (mos *MemOS) mount(source, mtpoint, fstype string, flgs uintptr, options string) error {
	mtpointOpen, err := mos.mfsOpenFileAtPath(mtpoint)
	if err != nil {
		return err
	}
	if !mtpointOpen.inode.isDir() {
		return ENOTDIR
	}
	ns := mos.ns
	if (flgs & syscall.MS_REMOUNT) > 0 {
		if (flgs & syscall.MS_BIND) > 0 {
			return ns.reconfigure_mount(mtpointOpen, flgs)
		} else {
			return ns.remount(mtpointOpen, flgs)
		}
	} else if (flgs & syscall.MS_BIND) > 0 {
		return ns.bind_mount(mtpointOpen, mos, source, flgs)
	} else if (flgs & (syscall.MS_SHARED | syscall.MS_PRIVATE | syscall.MS_SLAVE |
			syscall.MS_UNBINDABLE)) > 0 {
		return ns.change_mount_type(mtpointOpen, flgs)
	} else if (flgs & syscall.MS_MOVE) > 0 {
		return ns.move_mount(mtpointOpen, mos, source)
	}
	return ns.mount(mos, source, mtpointOpen, fstype, flgs)
}


func (mos *MemOS) prepReaderStream(instream io.Reader) io.Reader {
	if instream == nil {
		instream = nullStreamer(0)
	}
	mos.openInode(nil, newBaseChardevInode(devNull_st_rdev, instream, nil), "", O_RDONLY)
	return instream
}


func (mos *MemOS) prepWriterStream(ostream io.Writer) io.Writer {
	if ostream == nil {
		ostream = nullStreamer(0)
	}
	mos.openInode(nil, newBaseChardevInode(devNull_st_rdev, nil, ostream), "", O_WRONLY)
	return ostream
}

