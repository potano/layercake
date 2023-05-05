// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos



type namespaceType struct {
	readApi Vos
	devices map[uint64]filesystemInstance
	mounts []*mountType
	processes map[int]*MemOS
	nextPid int
}




func newRootNamespace(process *MemOS) error {
	ns := &namespaceType{
//		readApi: spec.ReadOS,
		devices: map[uint64]filesystemInstance{},
		processes: map[int]*MemOS{},
		nextPid: defaultInitialPid,
	}
	st_dev, err := ns.makeFilesystem(-1, -1, "rootfs", "none")
	if err != nil {
		return err
	}
	mnt, err := newMount(ns, st_dev, 0, nil, 0)
	if err != nil {
		return err
	}
	err = ns.addProcess(process, mnt, ns.devices[st_dev].rootInode(), "/")
	if err != nil {
		return err
	}
	return nil
}


func (ns *namespaceType) makeFilesystem(major, minor int, fstype, source string) (uint64, error) {
	sourceIsPath := len(source) > 0 && source[0] == '/'
	var st_dev uint64
	if major >= 0 && minor >= 0 {
		st_dev, _ = MajorMinorToStDev(major, minor)
	} else 	if sourceIsPath {
		st_dev = stDevForDeviceType(source)
	}
	if st_dev == 0 {
		st_dev = ns.stDevForFsType(fstype)
	}
	var makeFS filesystemMaker
	switch fstype {
	case "devtmpfs":
		makeFS = newDeviceFilesystem
	case "overlay":
		makeFS = newOverlayFilesystem
	default:
		makeFS = newStorageFilesystem
	}
	if ns.devices[st_dev] != nil {
		return 0, EEXIST
	}
	// select filesystem type
	fs, err := makeFS(st_dev, fstype, source, ns)
	if err != nil {
		return 0, err
	}
	ns.devices[st_dev] = fs
	return st_dev, nil
}

func (ns *namespaceType) removeFilesystem(st_dev uint64) {
	delete(ns.devices, st_dev)
}


func (ns *namespaceType) addProcess(mos *MemOS, mnt *mountType, rootInode inodeType,
		rootPath string) error {
	pid := mos.pid
	if pid < 1 {
		pid = ns.nextPid
		ns.nextPid++
	}
	if _, exists := ns.processes[pid]; exists {
		return EINVAL
	}
	mos.ns = ns
	mos.pid = pid
	mos.openFiles = map[int]*mfsOpenFile{}
	mos.environment = map[string]string{}

	dir, err := mos.openInode(mnt, rootInode, rootPath, O_RDONLY)
	if err != nil {
		return err
	}
	dir.abspath = rootPath
	mos.hideFD(dir)
	mos.root = dir
	dir, err = mos.openInode(mnt, rootInode, "/", O_RDONLY)
	if err != nil {
		return err
	}
	dir.abspath = "/"
	mos.hideFD(dir)
	mos.cwd = dir
	mos.exitCode = -1
	ns.processes[pid] = mos
	return nil
}


func (ns *namespaceType) processIsRegistered(mos *MemOS) bool {
	for pid, p := range ns.processes {
		if p == mos && mos.pid == pid {
			return true
		}
	}
	return false
}

func (ns *namespaceType) removeProcess(mos *MemOS) error {
	delete(ns.processes, mos.pid)
	return nil
}

func (ns *namespaceType) process1() *MemOS {
	return ns.processes[1]
}


func stDevForDeviceType(name string) uint64 {
	if len(name) > 6 && name[:5] == "/dev/" {
		alphaStart, digitStart := -1, -1
		numValue := 0
		ok := true
		for i := 5; i < len(name); i++ {
			c := name[i]
			if c >= '0' && c <= '9' {
				if alphaStart < 0 {
					ok = false
				}
				if digitStart < 0 {
					digitStart = i
				}
				numValue = numValue * 10 + int(c - '0')
			} else if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				if digitStart >= 0 {
					ok = false
				}
				if alphaStart < 0 {
					alphaStart = i
				}
			} else {
				ok = false
			}
		}
		if ok && alphaStart >= 0 {
			var basename string
			if digitStart >= 0 {
				basename = name[alphaStart:digitStart]
			} else {
				basename = name[alphaStart:]
			}
			for _, grp := range []struct{maj int; min int; base string}{
				{1, 0, "ram"},
				{3, 0, "hda"}, {3, 64, "hdb"}, {22, 0, "hdc"}, {22, 64, "hdd"},
				{33, 0, "hde"}, {33, 64, "hdf"}, {34, 0, "hdg"}, {34, 64, "hdh"},
				{56, 0, "hdi"}, {56, 64, "hdj"}, {57, 0, "hdk"}, {57, 64, "hdl"},
				{88, 0, "hdm"}, {88, 64, "hdn"}, {89, 0, "hdo"}, {89, 64, "hdp"},
				{90, 0, "hdq"}, {90, 64, "hdr"}, {91, 0, "hds"}, {91, 64, "hdt"},
				{7, 0, "loop"},
				{8, 0, "sda"}, {8, 16, "sdb"}, {8, 32, "sdc"}, {8, 48, "sdd"},
				{8, 64, "sde"}, {8, 80, "sdf"}, {8, 96, "sdg"}, {8, 112, "sdh"},
				{8, 128, "sdi"}, {8, 144, "sdj"}, {8, 160, "sdk"}, {8, 176, "sdl"},
				{8, 192, "sdm"}, {8, 208, "sdn"}, {8, 224, "sdo"}, {8, 240, "sdp"},
				{11, 0, "scd"},
			} {
				if basename == grp.base {
					st_dev, _ := MajorMinorToStDev(grp.maj, grp.min+numValue)
					return st_dev
				}
			}
		}
	}
	return 0;
}


func (ns *namespaceType) stDevForFsType(fstype string) uint64 {
	var minor int
	if fstype == "rootfs" {
		minor = 2
	} else if fstype == "devtmpfs" {
		minor = 5
	} else {
		minor = 6
		for st_dev := range ns.devices {
			maj, min := StDevToMajorMinor(st_dev)
			if maj == 0 && minor <= min {
				minor = min + 1
			}
		}
	}
	st_dev, _ := MajorMinorToStDev(0, minor)
	return st_dev
}


func (ns *namespaceType) lookUpFilesystem(fstype, source string) (uint64, error) {
	if len(source) > 0 && source[0] == '/' {
		// look up device
	} else {
		for st_dev, fs := range ns.devices {
			pfs := fs.(*baseFilesystemData)
			if pfs.fstype == fstype && (fstype != "tmpfs" || pfs.source == source) {
				return st_dev, nil
			}
		}
	}
	return 0, ENOENT
}



func (ns *namespaceType) reconfigure_mount(mtpoint *mfsOpenFile, flgs uintptr) error {
	return EINVAL
}

func (ns *namespaceType) remount(mtpoint *mfsOpenFile, flgs uintptr) error {
	return EINVAL
}

func (ns *namespaceType) bind_mount(mtpoint *mfsOpenFile, source string, flgs uintptr) error {
	pid1 := ns.processes[1]
	sourceOpen, err := pid1.open(source, O_RDONLY, 0)
	if err != nil {
		return err
	}
	pid1.hideFD(sourceOpen)
	mnt, err := newMount(ns, sourceOpen.mount.st_dev, sourceOpen.inode.ino(),
		mtpoint.mount, mtpoint.inode.ino())
	if err != nil {
		return err
	}
	mnt.sourceDir = sourceOpen
	return nil
}

func (ns *namespaceType) overlay_mount(mtpoint *mfsOpenFile, optionMap map[string]string,
		flgs uintptr) error {
	st_dev, err := ns.makeFilesystem(-1, -1, "overlay", "overlay")
	if err != nil {
		return err
	}
	mnt, err := newMount(ns, st_dev, overlayFS_root_ino, mtpoint.mount, mtpoint.inode.ino())
	if err != nil {
		ns.removeFilesystem(st_dev)
		return err
	}
	mnt.ephemeralFilesystem = true
	pid1 := ns.processes[1]
	for _, dir := range []struct{key string; openFile **mfsOpenFile} {
		{"upperdir", &mnt.sourceDir},
		{"lowerdir", &mnt.source2Dir},
		{"workdir", &mnt.workDir},
	} {
		var openFile *mfsOpenFile
		openFile, err = pid1.open(optionMap[dir.key], O_RDONLY, 0)
		if err != nil {
			break
		}
		*dir.openFile = openFile
		pid1.hideFD(openFile)
	}
	if err != nil {
		mnt.umount(0)
		return err
	}
	return ns.devices[st_dev].(*overlayFilesystem).init(ns, mnt)
}

func (ns *namespaceType) change_mount_type(mtpoint *mfsOpenFile, flgs uintptr) error {
	return EINVAL
}

func (ns *namespaceType) move_mount(mtpoint *mfsOpenFile, mos *MemOS, source string) error {
	return EINVAL
}

func (ns *namespaceType) mount(mos *MemOS, source string, mtpoint *mfsOpenFile, fstype string,
		flgs uintptr, options string) error {
	var st_dev uint64
	var err error
	optionMap := ParseMountOptions(options)
	switch fstype {
	case "tmpfs", "proc", "sysfs", "devtmpfs", "devpts", "securityfs":
		st_dev, err = ns.makeFilesystem(-1, -1, fstype, source)
		if err != nil {
			return err
		}
	case "overlay":
		return ns.overlay_mount(mtpoint, optionMap, flgs)
	default:
		sourceOpen, err := mos.mfsOpenFileAtPath(source)
		if err != nil {
			return err
		}
		if blkdev, have := sourceOpen.inode.(blockDeviceInodeType); !have {
			return ENOTBLK
		} else {
			st_dev = blkdev.getRdev()
		}
	}
	if _, have := ns.devices[st_dev]; !have {
		return ENOENT
	}
	mnt, err := newMount(ns, st_dev, 0, mtpoint.mount, mtpoint.inode.ino())
	if err != nil {
		return err
	}
	_ = mnt
	return nil
}

