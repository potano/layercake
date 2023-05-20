package vos

import (
	"fmt"
	"sort"
	"strings"
	"syscall"
	"testing"
)


// Types and functions used only in unit tests



func getFSInodes(T *testing.T, ns *namespaceType, st_dev uint64) (string, uint64, []inodeType) {
	dev := ns.devices[st_dev]
	if fs, is := dev.(*storageFilesystem); is {
		return fs.fstype, fs.st_dev, fs.inodes
	} else if fs, is := dev.(*deviceFilesystem); is {
		return fs.fstype, fs.st_dev, fs.inodes
	} else if fs, is := dev.(*overlayFilesystem); is {
		return fs.fstype, fs.st_dev, fs.inodes
	} else {
		T.Fatalf("unknown filesystem type %v", dev)
	}
	return "", 0, nil
}


func sortDirentMapKeysAndInums(entries map[string]inodeType) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for i, key := range keys {
		keys[i] = fmt.Sprintf("%s=%d", key, entries[key].ino())
	}
	return keys
}


func inodeToDebugString(inode inodeType) string {
	if inode == nil {
		return "nil inode"
	}
	if ovfsInode, is := inode.(ovfsInodeInterface); is {
		baseInfo := fmt.Sprintf(" %s #%d ", StDevToString(inode.dev()), inode.ino())
		activeInode, pointsToUpper := ovfsInode.getActiveInode()
		if pointsToUpper {
			baseInfo += "upper: "
		} else {
			baseInfo += "lower: "
		}
		if activeInode == nil {
			baseInfo += "NO ACTIVE INODE"
		} else {
			baseInfo += fmt.Sprintf("-> %s #%d",
				StDevToString(activeInode.dev()), activeInode.ino())
		}
		switch inode.nodeType() {
		case nodeTypeFile:
			return "ovfs file" + baseInfo
		case nodeTypeDir:
			return "ovfs dir" + baseInfo + "; " + strings.Join(
				sortDirentMapKeysAndInums(inode.(*ovfsDirInode).entries), "; ")
		case nodeTypeLink:
			return "ovfs link" + baseInfo
		case nodeTypeSock:
			return "ovfs socket" + baseInfo
		case nodeTypeFifo:
			return "ovfs FIFO" + baseInfo
		case nodeTypeCharDev:
			var more string
			if pointsToUpper && inodeIsOvfsWhiteout(activeInode) {
				more += "; whiteout"
			}
			return "ovfs char dev" + baseInfo + "; " +
				StDevToString(inode.(deviceInodeType).getRdev()) + more
		case nodeTypeBlockDev:
			return "base block dev" + baseInfo + "; " +
				StDevToString(inode.(deviceInodeType).getRdev())
		default:
			return fmt.Sprintf("unknown base device type %d", inode.nodeType())
		}
	} else {
		baseInfo := fmt.Sprintf(" %s #%d %o %d:%d", StDevToString(inode.dev()), inode.ino(),
			inode.mode(), inode.uid(), inode.gid())
		switch inode.nodeType() {
		case nodeTypeFile:
			content := string(inode.(*mfsFileInodeBase).contents)
			if len(content) > 5 {
				content = content[:5] + "..."
			}
			return "base file" + baseInfo + "; \"" + content + "\""
		case nodeTypeDir:
			dirInode := inode.(*mfsDirInodeBase)
			return "base dir" + baseInfo + "; " + strings.Join(
				sortDirentMapKeysAndInums(dirInode.entries), ", ")
		case nodeTypeLink:
			return "base link" + baseInfo + "; -> " + inode.(*mfsLinkInodeBase).target
		case nodeTypeSock:
			return "base socket" + baseInfo
		case nodeTypeFifo:
			content := string(inode.(*mfsFifoInodeBase).contents)
			if len(content) > 5 {
				content = content[:5] + "..."
			}
			return "base FIFO" + baseInfo + "; " + content
		case nodeTypeCharDev:
			return "base char dev" + baseInfo + "; " +
				StDevToString(inode.(deviceInodeType).getRdev())
		case nodeTypeBlockDev:
			return "base block dev" + baseInfo + "; " +
				StDevToString(inode.(deviceInodeType).getRdev())
		default:
			return fmt.Sprintf("unknown base device type %d", inode.nodeType())
		}
	}
	return "unknown inode"
}


func dumpFilesystemInodes(fs filesystemInstance) {
	for i, ino := range fs.inodeList() {
		fmt.Printf(" %d: %s\n", i, inodeToDebugString(ino))
	}
}



type nsTestDevices []struct{majmin string; inodes []nsTestInode}
type nsTestInode struct {tp uint; perms int; uid, gid, nlink uint64; contents string}

func checkNSDevices(ns *namespaceType, T *testing.T, testDevices nsTestDevices) {
	T.Helper()
	existingDevices := make(map[uint64]bool, len(ns.devices))
	for st_dev := range ns.devices {
		existingDevices[st_dev] = true
	}
	for _, spec := range testDevices {
		major, minor, err := ParseMajorMinorString(spec.majmin)
		if err != nil {
			T.Fatalf("trying to parse %s: %s", spec.majmin, err)
		}
		st_dev, err := MajorMinorToStDev(major, minor)
		if err != nil {
			T.Fatalf("%s composing major=%d, minor=%d", err, major, minor)
		}
		if !existingDevices[st_dev] {
			T.Fatalf("expected to find device %s", spec.majmin)
		}
		delete(existingDevices, st_dev)
		_, fs_st_dev, fs_inodes := getFSInodes(T, ns, st_dev)
		if fs_st_dev != st_dev {
			T.Fatalf("expected filesystem %s to have st_dev %d, got %d", spec.majmin,
				st_dev, fs_st_dev)
		}
		if len(fs_inodes) != len(spec.inodes) + 1 {
			T.Fatalf("expected filesystem %s to have %d inodes, got %d", spec.majmin,
				len(spec.inodes), len(fs_inodes) - 1)
		}
		for i, inode := range fs_inodes {
			if i == 0 {
				continue
			}
			if inode.ino() != uint64(i) {
				T.Fatalf("device %s: inode %d does not report the correct inum",
					spec.majmin, i)
			}
			if inode.dev() != st_dev {
				T.Fatalf("device %s: inode %d should have st_dev %d, not %d",
					spec.majmin, i, st_dev, inode.dev())
			}
			want := spec.inodes[i - 1]
			if inode.nodeType() != want.tp {
				T.Fatalf("device %s: expected inode %d to have type %d, got %d",
					spec.majmin, i, want.tp, inode.nodeType())
			}
			if int(inode.mode() & 07777) != want.perms {
				T.Fatalf("device %s: expected inode %d permission %04o, got %04o",
					spec.majmin, i, want.perms, inode.mode() & 07777)
			}
			if inode.uid() != want.uid {
				T.Fatalf("device %s: expected inode %d UID %d, got %d",
					spec.majmin, i, want.uid, inode.uid())
			}
			if inode.gid() != want.gid {
				T.Fatalf("device %s: expected inode %d GID %d, got %d",
					spec.majmin, i, want.gid, inode.gid())
			}
			if inode.nlink() != want.nlink {
				T.Fatalf("device %s: expected inode %d to have %d links, got %d",
					spec.majmin, i, want.nlink, inode.nlink())
			}
			var contents string
			switch want.tp {
			case nodeTypeFile:
				file := inode.(fileInodeType)
				buf := make([]byte, file.size())
				file.readFile(buf, 0)
				contents = string(buf)
			case nodeTypeDir:
				dir := inode.(dirInodeType)
				entries := dir.rawDirentMap()
				contents = strings.Join(sortDirentMapKeysAndInums(entries), "\n")
			case nodeTypeLink:
				contents = inode.(linkInodeType).getLinkTarget()
			case nodeTypeFifo:
				fifo := inode.(fifoInodeType)
				contents = string(fifo.peekFifo())
			case nodeTypeSock:
				contents = ""
			case nodeTypeCharDev, nodeTypeBlockDev:
				maj, min := StDevToMajorMinor(inode.(deviceInodeType).getRdev())
				contents = MajorMinorToString(maj, min)
			default:
				T.Fatalf("device %s, inode %d: unknown entry type %d", spec.majmin,
					i, want.tp)
			}
			if contents != want.contents {
				T.Fatalf("device %s, inode %d: contents not '%s', but '%s'",
					spec.majmin, i, want.contents, contents)
			}
		}
	}
	if len(existingDevices) > 0 {
		T.Fatalf("found %d unexpected devices", len(existingDevices))
	}
}



type ovTestDevices []struct{majmin string; inodes []ovTestInode}
type ovTestInode struct {tp uint; upper bool; inum, nlink uint64; contents string}

func checkOverlayFS(ns *namespaceType, T *testing.T, testDevices ovTestDevices) {
	T.Helper()
	existingDevices := make(map[uint64]bool, len(ns.devices))
	for st_dev, fs := range ns.devices {
		if _, is := fs.(*overlayFilesystem); is {
			existingDevices[st_dev] = true
		}
	}
	if len(testDevices) != len(existingDevices) {
		T.Fatalf("expected to find %d overlay filesystems, found %d", len(testDevices),
			len(existingDevices))
	}
	for _, spec := range testDevices {
		major, minor, err := ParseMajorMinorString(spec.majmin)
		if err != nil {
			T.Fatalf("trying to parse %s: %s", spec.majmin, err)
		}
		st_dev, err := MajorMinorToStDev(major, minor)
		if err != nil {
			T.Fatalf("%s composing major=%d, minor=%d", err, major, minor)
		}
		if !existingDevices[st_dev] {
			T.Fatalf("expected to find device %s", spec.majmin)
		}
		delete(existingDevices, st_dev)
		fs := ns.devices[st_dev].(*overlayFilesystem)
		_, upperStDev, upperInodes := getFSInodes(T, ns, fs.upperFS.getStDev())
		_, lowerStDev, lowerInodes := getFSInodes(T, ns, fs.upperFS.getStDev())
		if len(fs.inodes) != len(spec.inodes) + 1 {
			T.Fatalf("expected filesystem %s to have %d inodes, got %d", spec.majmin,
				len(spec.inodes), len(fs.inodes) - 1)
		}
		checkSideInodeConsistency := func (T *testing.T, i int, fs_st_dev uint64,
				inode inodeType, isUpper bool, inodes []inodeType, where string,
				want ovTestInode) {
			var tail string
			if inode == nil {
				if want.inum != 0 {
					tail = fmt.Sprintf("is missing %s inode", where)
				}
			} else if want.inum == 0 {
				tail = fmt.Sprintf("has unexpected %s inode", where)
			} else if inode.dev() != fs_st_dev {
				tail = fmt.Sprintf("has %s inode of filesystem %s, not %s", where,
					StDevToString(inode.dev()), StDevToString(fs_st_dev))
			} else if inode.ino() != want.inum {
				tail = fmt.Sprintf("has %s inode with inum %d, not %d", where,
					inode.ino(), want.inum)
			} else if isUpper != want.upper {
				tail = fmt.Sprintf("inum %d unexpectedly is %s inode", want.inum,
					where)
			} else if inode.nodeType() != want.tp {
				tail = fmt.Sprintf("has %s inode of type %d, not %d", where,
					inode.nodeType(), want.tp)
			}
			if len(tail) > 0 {
				T.Fatalf("device %s: merge inode %d %s", spec.majmin, i, tail)
			}
		}
		for i, genericInode := range fs.inodes {
			if i == 0 {
				continue
			}
			inode, ok := genericInode.(ovfsInodeInterface)
			if !ok {
				T.Fatalf("device %s: inode %d is not an overlayfs inode",
					spec.majmin, i)
			}
			if inode.ino() != uint64(i) {
				T.Fatalf("device %s: inode %d does not report the correct inum",
					spec.majmin, i)
			}
			if inode.dev() != st_dev {
				T.Fatalf("device %s: inode %d should have st_dev %d, not %d",
					spec.majmin, i, st_dev, inode.dev())
			}
			want := spec.inodes[i - 1]
			if inode.nodeType() != want.tp {
				T.Fatalf("device %s: expected inode %d to have type %d, got %d",
					spec.majmin, i, want.tp, inode.nodeType())
			}
			activeInode, isUpper := inode.getActiveInode()
			if isUpper {
				checkSideInodeConsistency(T, i, upperStDev, activeInode, isUpper,
					upperInodes, "upper", want)
			} else {
				checkSideInodeConsistency(T, i, lowerStDev, activeInode, isUpper,
					lowerInodes, "lower", want)
			}
			if inode.nlink() != want.nlink {
				T.Fatalf("device %s: expected inode %d to have %d links, got %d",
					spec.majmin, i, want.nlink, inode.nlink())
			}
			var contents string
			switch want.tp {
			case nodeTypeFile:
				file := inode.(fileInodeType)
				buf := make([]byte, file.size())
				file.readFile(buf, 0)
				contents = string(buf)
			case nodeTypeDir:
				entries := inode.(dirInodeType).direntMap()
				contents = strings.Join(sortDirentMapKeysAndInums(entries), "\n")
			}
			if contents != want.contents {
				T.Fatalf("device %s, inode %d: contents not '%s', but '%s'",
					spec.majmin, i, want.contents, contents)
			}
		}
	}
	if len(existingDevices) > 0 {
		T.Fatalf("failed to consider %d overlay filesystems", len(existingDevices))
	}
}



type nsTestMounts []struct{
	major, minor, root_ino, mounted_in, mounted_at int
	opens []nsTestOpen
	mounts []nsTestMount
}
type nsTestOpen struct {pid, fd, flags int; ino uint64; name string; pos int64; r, w, x bool;
	abspath string}
type nsTestMount struct {st_ino uint64; mountnum int}

func checkNSMounts(ns *namespaceType, T *testing.T, testMounts nsTestMounts) {
	T.Helper()
	if len(ns.mounts) != len(testMounts) {
		T.Fatalf("expected %d mounts, have %d", len(testMounts), len(ns.mounts))
	}
	for mntno, spec := range testMounts {
		mnt := ns.mounts[mntno]
		st_dev, _ := MajorMinorToStDev(spec.major, spec.minor)
		if mnt.st_dev != st_dev {
			T.Fatalf("mount %d: expected st_dev=%d, got %d", mntno, st_dev, mnt.st_dev)
		}
		if mnt.ns != ns {
			T.Fatalf("mount %d not mounted to correct filesystem", mntno)
		}
		if mnt.root_ino != uint64(spec.root_ino) {
			T.Fatalf("mount %d: expected root ino=%d, got %d", mntno, spec.root_ino,
				mnt.root_ino)
		}
		if spec.mounted_in < 0 {
			if mnt.mounted_in != nil {
				T.Fatalf("mount %d: is not the root mount", mntno)
			}
		} else if mnt.mounted_in != ns.mounts[spec.mounted_in] {
			T.Fatalf("mount %d: is not mounted in mount %d", mntno, spec.mounted_in)
		}
		if spec.mounted_in >= 0 && mnt.mounted_in_ino != uint64(spec.mounted_at) {
			T.Fatalf("mount %d: expected mounted-in inum to be %d, got %d", mntno,
				spec.mounted_at, mnt.mounted_in_ino)
		}
		if len(mnt.openFiles) != len(spec.opens) {
			T.Fatalf("mount %d: expected %d open files, got %d", mntno, len(spec.opens),
				len(mnt.openFiles))
		}
		for ofc, open := range spec.opens {
			key := pidFdType{open.pid, open.fd}
			of := mnt.openFiles[key]
			if of == nil {
				T.Fatalf("mount %d: expected to find PID %d's open fd %d", mntno,
					open.pid, open.fd)
			}
			if of.mount != mnt {
				T.Fatalf("mount %d, open file %d: wrong mount", mntno, ofc)
			}
			if of.mos.pid != open.pid {
				T.Fatalf("mount %d, open file %d: want  pid %d, got %d", mntno, ofc,
					open.pid, of.mos.pid)
			}
			if of.fd != open.fd {
				T.Fatalf("mount %d, open file %d: want fd %d, got %d", mntno, ofc,
					open.fd, of.fd)
			}
			if of.flags != open.flags {
				T.Fatalf("mount %d, open file %d: want flags %X, got %X",
					mntno, ofc, open.flags, of.flags)
			}
			if of.inode.ino() != open.ino {
				T.Fatalf("mount %d, open file %d: want inum %d, got %d", mntno,
					ofc, open.ino, of.inode.ino())
			}
			if of.name != open.name {
				T.Fatalf("mount %d, open file %d: want name %s, got %s", mntno,
					ofc, open.name, of.name)
			}
			if of.pos != open.pos {
				T.Fatalf("mount %d, open file %d, want pos %d, got %d", mntno,
					ofc, open.pos, of.pos)
			}
			if of.readable != open.r || of.writable != open.w ||
					of.executable != open.x {
				T.Fatalf("mount %d, open file %d: want %s, got %s", mntno, ofc,
					rwx(open.r, open.w, open.x),
					rwx(of.readable, of.writable, of.executable))
			}
			if of.abspath != open.abspath {
				T.Fatalf("mount %d, open file %d: want path %s, got %s", mntno,
					ofc, open.abspath, of.abspath)
			}
		}
		if len(mnt.mountpoints) != len(spec.mounts) {
			T.Fatalf("mount %d: expected %d submounts, got %d", mntno, len(spec.mounts),
				len(mnt.mountpoints))
		}
		for _, mtinfo := range spec.mounts {
			submount := mnt.mountpoints[mtinfo.st_ino]
			if submount == nil {
				T.Fatalf("mount %d, expected submount at inode %d", mntno,
					mtinfo.st_ino)
			}
			if submount != ns.mounts[mtinfo.mountnum] {
				T.Fatalf("mount %d, wrong submount applied to inode %d", mntno,
					mtinfo.st_ino)
			}
		}
	}
}

func rwx(r, w, x bool) string {
	var rs, ws, xs string
	if r { rs = "R" } else { rs = "r"}
	if w { ws = "W" } else { ws = "w"}
	if x { xs = "X" } else { xs = "x"}
	return rs + "/" + ws + "/" + xs
}



type nsTestProcesses []struct{pid int; euid, gid uint64; rootfd, cwdfd int; opens []nsTestProcOpen}
type nsTestProcOpen struct {fd int; majmin string; ino uint64}

func checkNSProcesses(ns *namespaceType, T *testing.T, testProcesses nsTestProcesses) {
	T.Helper()
	if len(ns.processes) != len(testProcesses) {
		T.Fatalf("expected %d processes, have %d", len(testProcesses), len(ns.processes))
	}
	for _, pspec := range testProcesses {
		pid := pspec.pid
		proc := ns.processes[pid]
		if proc == nil {
			T.Fatalf("no such process %d", pid)
		}
		if proc.pid != pid {
			T.Fatalf("process %d: identifies as process %d", pid, proc.pid)
		}
		if proc.euid != pspec.euid || proc.gid != pspec.gid {
			T.Fatalf("process %d: expected uid/gid %d/%d, got %d/%d", pid,
				pspec.euid, pspec.gid, proc.euid, proc.gid)
		}
		if proc.root == nil {
			T.Fatalf("process %d: no root directory set", pid)
		}
		if proc.root.fd != pspec.rootfd {
			T.Fatalf("process %d: expected root on FD %d, have %d", pid, pspec.rootfd,
				proc.root.fd)
		}
		if proc.cwd == nil {
			T.Fatalf("process %d: no current directory set", pid)
		}
		if proc.cwd.fd != pspec.cwdfd {
			T.Fatalf("process %d: expected cwd on FD %d, have %d", pid, pspec.cwdfd,
				proc.cwd.fd)
		}
		if len(proc.openFiles) != len(pspec.opens) {
			T.Fatalf("process %d: expected %d open files, got %d", pid,
				len(pspec.opens), len(proc.openFiles))
		}
		for _, want := range pspec.opens {
			fd := want.fd
			of := proc.openFiles[fd]
			if of == nil {
				T.Fatalf("process %d: expected FD %d to be open", pid, fd)
			}
			if of.mos.pid != pid {
				T.Fatalf("process %d, FD %d: expected PID %d, not %d",
					pid, fd, pid, of.mos.pid)
			}
			maj, min, _ := ParseMajorMinorString(want.majmin)
			st_dev, _ := MajorMinorToStDev(maj, min)
			if of.inode.dev() != st_dev {
				T.Fatalf("process %d, FD %d: expected st_dev %d, not %d",
					pid, fd, st_dev, of.inode.dev())
			}
			if of.inode.ino() != want.ino {
				T.Fatalf("process %d, FD %d: expected inum %d, not %d",
					pid, fd, want.ino, of.inode.ino())
			}
		}
	}
}



func checkStat(T *testing.T, desc string, stat, test syscall.Stat_t) {
	T.Helper()
	if stat.Dev != test.Dev {
		T.Fatalf("%s expected st_dev = %d, got %d", desc, test.Dev, stat.Dev)
	}
	if stat.Ino != test.Ino {
		T.Fatalf("%s expected st_ino = %d, got %d", desc, test.Ino, stat.Ino)
	}
	if stat.Nlink != test.Nlink {
		T.Fatalf("%s expected st_nlink = %d, got %d", desc, test.Nlink, stat.Nlink)
	}
	if stat.Mode != test.Mode {
		T.Fatalf("%s expected st_mode = %o, got %o", desc, test.Mode, stat.Mode)
	}
	if stat.Uid != test.Uid {
		T.Fatalf("%s expected st_uid = %d, got %d", desc, test.Uid, stat.Uid)
	}
	if stat.Gid != test.Gid {
		T.Fatalf("%s expected st_gid = %d, got %d", desc, test.Gid, stat.Gid)
	}
	if stat.Rdev != test.Rdev {
		T.Fatalf("%s expecting st_rdev = %d, got %d", desc, test.Rdev, stat.Rdev)
	}
	if stat.Size != test.Size {
		T.Fatalf("%s expecting st_size = %d, got %d", desc, test.Size, stat.Size)
	}
}

