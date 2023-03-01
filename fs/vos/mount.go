// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos


type mountType struct {
//	root, mountpoint, options string
//	source, source2, workdir string
	ns *namespaceType
	st_dev, root_ino uint64
//	index, mounted_in int
	mounted_in *mountType
	mounted_in_ino uint64
	sourceDir, source2Dir, workDir *mfsOpenFile
	openFiles map[pidFdType]*mfsOpenFile
	mountpoints map[uint64]*mountType
}


type pidFdType struct {
	pid, fd int
}


func newMount(ns *namespaceType, st_dev, root_ino uint64, mounted_in *mountType,
		mounted_in_ino uint64) (*mountType, error) {
	device := ns.devices[st_dev]
	if device == nil {
		return nil, ENOTBLK
	}
	if root_ino == 0 {
		root_ino = device.rootInode().ino()
	}
	mount := &mountType{
		ns: ns,
		st_dev: st_dev,
		root_ino: root_ino,
		mounted_in: mounted_in,
		mounted_in_ino: mounted_in_ino,
		openFiles: map[pidFdType]*mfsOpenFile{},
		mountpoints: map[uint64]*mountType{},
	}
	ns.mounts = append(ns.mounts, mount)
	if mounted_in != nil {
		mounted_in.mountpoints[mounted_in_ino] = mount
	}
	return mount, nil
}


func (mount *mountType) inodeByInum(inum uint64) inodeType {
	return mount.ns.devices[mount.st_dev].inodeByInum(inum)
}


func (mount *mountType) rootInode() dirInodeType {
	return mount.inodeByInum(mount.root_ino).(dirInodeType)
}


func (mount *mountType) addInode(dirInode dirInodeType, name string, inode inodeType,
		) (inodeType, error) {
	return mount.ns.devices[mount.st_dev].addInode(dirInode, name, inode)
}


func (mount *mountType) umount(flags int) error {
	if len(mount.openFiles) > 0 || len(mount.mountpoints) > 0 {
		return EBUSY
	}
	delete(mount.mounted_in.mountpoints, mount.mounted_in_ino)
	index := -1
	for i, mt := range mount.ns.mounts {
		if mt == mount {
			index = i
			break
		}
	}
	if index >= 0 {
		mount.ns.mounts = append(mount.ns.mounts[:index], mount.ns.mounts[index+1:]...)
	}
	if mount.sourceDir != nil {
		mount.sourceDir.Close()
	}
	if mount.source2Dir != nil {
		mount.source2Dir.Close()
	}
	if mount.workDir != nil {
		mount.workDir.Close()
	}
	return nil
}



/*
func (fs *MemFS) getDirent(rootfs *memDevice, name string) *memDirEnt {
	parts := strings.Split(name, "/")
	if rootfs == nil || len(parts) == 0 {
		return nil
	}
	dir := rootfs.rootDir
	for {
		if dir == nil || dir.tp != memEntDir {
			return nil
		}
		if len(dir.contents) > 0 {
			rootfs = fs.devices[dir.contents]
			if rootfs == nil {
				return nil
			}
			dir = rootfs.rootDir
			continue
		}
		name := parts[0]
		for len(name) == 0 {
			parts = parts[1:]
			name = parts[0]
		}
		dir = dir.entries[name]
		if len(parts) == 0 {
			break
		}
	}
	return dir
}


func (mfs *MemFS) getSourceDeviceFS(source string, fstype *string) (st_dev, root string, err error) {
	if *fstype == "overlay" {
		st_dev = fmt.Sprintf("0:%d", mfs.nextMinor)
		mfs.nextMinor++
		root = "/"
	} else {
		max_match := 0
		srclen := len(source)
		for _, mnt := range mfs.nodes {
			mntlen := len(mnt.mountpoint)
			if srclen >= mntlen && mnt.mountpoint == source[:mntlen] &&
				(srclen == mntlen || mntlen < 2 || source[mntlen] == '/') {
				if mntlen > max_match {
					st_dev = mnt.st_dev
					*fstype = mnt.fstype
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
			err = errors.New("no such device")
		}
	}
	return
}

func (mfs *MemFS) findMountpointParentID(mtpoint string) (parentID int, err error) {
	max_match := 0
	newlen := len(mtpoint)
	for _, mnt := range mfs.nodes {
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
		err = errors.New("no such file or directory")
	}
	return
}

func (fs *MemFS) mount(source, mtpoint, fstype string, flgs uintptr, options string) error {
	var st_dev, root string
	var parentID int
	onlyModifying := (flgs & (syscall.MS_REMOUNT | syscall.MS_SHARED | syscall.MS_PRIVATE |
		syscall.MS_SLAVE | syscall.MS_UNBINDABLE)) > 0
	if onlyModifying {
		_, err := mfs.findMountpointParentID(mtpoint)
		return err
	}
	st_dev, root, err := getSourceDeviceFS(source, &fstype)
	if err != nil {
		return err
	}
	parentID, err := mfs.findMountpointParentID(mtpoint)
	if err != nil {
		return err
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

func (mfs *MemFS) addMfsMountNode(root, mountpoint, st_dev, fstype, options string,
		parentID int) error {
	mountNode := mfsMount{
		root: root,
		mountpoint: mountpoint,
		st_dev: st_dev,
		fstype: fstype,
		options: options,
		index: len(mfs.nodes) + 1,
		mounted_in: parentID,
		rootDir: memDirEnt{"/", memEntDir, "", map[string]*memDirEnt{}},
	}




func (mfs *MemFS) toAbsolutePath(name string) string {
	if !path.IsAbs(dirname) {
		dirname := path.Join(mfs.pwd, dirname)
	}
	return dirname
}



func pathToParent(pth string) string {
	i := strings.LastIndexByte(pth)
	if i < 0 {
		return ".."
	}
	return pth[:i]
}


func (mfs *MemFS) stat_needDir(name) (inode mfsInode, err error) {
	inode, err = mfs.stat(name)
	if err == nil && !inode.isDir() {
		err = syscall.ENOTDIR
	}
	return
}


// Find the matching inode (dereference final symlink)
// Resolve realpath
// Find the matching inode (not dereferencing the final symlink)
// Generate inode in directory that is penultimate entry in the path
// Generate inode along with any directories need to reach it

func (mfs *MemFS) stat(name string) (mfsInode, error) {
	inode, _, err := mfs.pathToInode(name, pti_stat)
	return inode, err
}
func (mfs *MemFS) lstat(name string) (mfsInode, error) {
	inode, _, err := mfs.pathToInode(name, pti_lstat)
	return inode, err
}
func (mfs *MemFS) realpath(name string) (string, error) {
	_, pth, err := mfs.pathToInode(name, pti_realpath, nil)
	return pth, nil
}
func (mfs *MemFS) makeInodeInDir(name string, ftype int) (mfsInode, err) {
	dir, fname, err := mfs.pathToInode(name, pti_parent)
	if err != nil {
		return nil, err
	}
	inode := newInode(dir.getDev(), ftype)
	err := addDirent(dir, inode, fname)
	if err != nil {
		return err
	}
	return inode, nil
}
func (mfs *MemFS) makeLinkInDir(target, linkname string) error {
	inode, _, err := mfs.pathToInode(target, pti_stat)
	if err != nil {
		return err
	}
	dir, lname, err := mfs.pathToInode(linkname, pti_parent)
	if err != nil {
		return err
	}
	return mfs.addDirent(dir, inode, lname)
}
func (mfs *MemFS) mkdir(name string) (mfsInode, err) {
	dir, dirname, err := mfs.pathToInode(name, pti_parent)
	if err != nil {
		return nil, err
	}
	inode := newInode(dir.getDev(), mfsiDir)
	err = mfs.addDirent(dir, inode, dirname)
	if err != nil && err != syscall.EEXIST {
		return nil, err
	}
	inode.st_nlink++
	return inode, nil
}
func (mfs *MemFS) makeInodeAtPath(name string, ftype int) (mfsInode, err) {
	dir, segment, err := mfs.pathToInode(name, pti_makeParents)
	if err != nil {
		return nil, err
	}
	inode := newInode(dir.getDev(), ftype)
	err = mfs.addDirent(dir, inode, segment)
	if err != nil {
		return nil, err
	}
	return inode, nil
}

// Resolves a path to an array of inodes
// Arguments
//     name: path name
//     mode: operation mode; one of the following integer constant values
//        pti_stat: as stat(2) call: return inode item exists at path, error otherwise
//        pti_lstat: as lstat(2) call: like pti_stat but without dereferencing any final symlink
//        pti_realpath: return canonical path if item found, error if not
//        pti_parent: resolve path to containing directory and return name of last path element
//        pti_makeParents: resolve/generate path to containing directory, return last element name
// Results
//     inode of final element or containing directory
//     name/path string
//     error value
// Form of the result inode and string depend on the operation-mode argument:
//     pti_stat, pti_lstat: inode of final element / empty string
//     pti_realpath: nil / canonical pathname
//     pti_parent, pti_makeParents: inode of containing directory / name of final pathname element
func (mfs *MemFS) resolvePath(name string, mode int) (memFSInode, string, error) {
	if len(name) == 0 {
		return nil, "", syscall.ENOENT
	}
	if name[0] != '/' {
		name = mfs.pwd + "/" + name
	}
	scanToParentOnly := mode == pti_parent || mode == pti_makeParents
	parts := strings.Split(name, "/")
	dir := mfs.devices[mfs.mounts[0].st_dev].inodes[1]
	inodes := make([]*memFSInode, 1, len(parts))
	pathname := make([]string, 1, len(parts))
	inodes[0] = dir
	for len(parts) > 0 {
		part0 = parts[0]
		if len(part0) == 0 || part0 == "." {
			parts = parts[1:]
			continue
		}
		if part0 == ".." {
			parts = parts[1:]
			if len(path) > 1 {
				dir = inodes[len(path)-1]
				path = inodes[:len(inodes)-1]
				pathname = pathname[:len(pathname)-1]
			}
			continue
		}
		if inum, exists := dir.entries[part0]; !exists {
			if mode != pti_makeParent || len(parts) > 1 {
				break
			}
			dir2 := newInode(dir.st_dev, mfsiDir)
			_ = mfs.addDirent(dir, dir2, part0)
			inum = dir2.st_ino
		}
		parts = parts[1:]
		inode = mfs.devices[dev.st_dev].inodes[inum]
		inodes = append(inodes, inode)
		pathname = append(pathname, part0)
		switch typedInode := inode.(type) {
		case mfsInodeDir:
			dir = typedInode
			if dir.mount != nil {
				dir = mfs.devices[dir.mount.st_dev].inodes[dir.mount.root_ino]
				path[len(path)-1] = dir
			}
		case mfsInodeLink:
			if mode == pti_lstat && len(parts) == 0 {
				break
			}
			if len(typedInode.target) == 0 {
				return nil, "", syscall.ENOENT
			}
			target := strings.Split(typedInode.target, "/")
			if len(target[0]) == 0 {
				inodes = inodes[:0]
				pathname = pathname[:0]
				path = target
			} else {
				inodes = inodes[:len(inodes)-1]
				pathname = pathname[:len(pathname)-1]
				target = append(target, parts...)
				parts = target
			}
		default:
			if len(parts) > 0 {
				return nil, "", syscall.ENOTDIR
			}
		}
	}

	if len(inodes) == 0 {
		return nil, "", syscall.ENOENT
	}
	if mode == pti_parent || pti_makeParent {
		if len(parts) > 1 {
			return nil, "", syscall.ENOENT
		}
		if len(inodes) == 1 {
			return inodes[0], "", nil
		}
		if len(parts) == 0 {
			return inodes[len(inodes)-2], realpath[len(realpath)-1], nil
		}
		return inodes[len(inodes)-1], parts[0], nil
	}

	if len(parts) != 0 {
		return nil, "", syscall.ENOENT
	}
	inode := inodes[len(inodes)-1]
	if mode == pti_realpath {
		return inode, strings.Join(pathname, "/"), nil
	}
	return inode, "", nil
}


func (mfs *MemFS) registerNewInode(device uint64, inode *mfsInode) int {
	dev := &mfs.devices[device]
	ino := len(dev.inodes) + 1
	inode.st_ino := ino
	inode.st_dev = device
	inode.setAllTimesNow()
	dev.devices = append(dev.devices, inode)
	return ino
}


func (mfs *MemFS) newInode(device uint64, tp int) mfsInode {
	var inode mfsInode
	var mode uint64
	switch tp {
	case mfsiFile:
		inode = &mfsInodeFile{}
		mode = S_IFREG
	case mfsiDir:
		inode = &mfsInodeDir{}
		mode = S_IFDIR | syscall.S_IXUSR | syscall.S_IXGRP | syscall.S_IXOTH
	case mfsiLink:
		node := &mfsInodeLink{}
		mode = S_IFLNK
	case mfsiFifo:
		inode = &mfsInodeBase{}
		mode = S_IFIFO
	case mfsiSock:
		inode = &mfsInodeBase{}
		mode = S_IFSOCK
	case mfsiCharDev:
		inode = &mfsInodeDevice{}
		mode = S_IFCHR
	case mfsiBlockDev:
		inode = &mfsInodeDevice{}
		mode = S_IFBLK
	}
	node.st_mode = mode | syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP | syscall.S_IROTH
	_ := mfs.registerNewInode(device, inode)
	return inode
}


func (mfs *MemFS) addNewInode(dir *mfsInodeDir, tp int, name string) (mfsInode, error) {
	inode := mfs.newInode(dir.st_dev, tp)
	err := mfs.addDirent(dir, inode, name)
	return inode, err
}



func (mfs *MemFS) addDirent(dirInode, inode mfsInode, name string) error {
	dir, ok := dirInode.(*mfsInodeDir)
	if !ok {
		return syscall.ENOTDIR
	}
	if _, exists := dir.entries[name]; exists {
		return syscall.EEXIST
	}
	dir.entries[name] = inode.st_ino
	dir.incrementNlinks()
	return inode.incrementNlinks()
}


func (mfs *MemFS) rmDirent(dirInode, name string) error {
	dir, ok := dirInode.(*mfsInodeDir)
	if !ok {
		return syscall.ENOTDIR
	}
	inum, exists := dir.entries[name]
	if !exists {
		return syscall.ENOENT
	}
	inode := mfs.devices[dev.st_dev].inodes[inum]
	delete(dir.entries, name)
	dir.decrementNlinks()
	inode.decrementNlinks()
	return nil
}
*/

