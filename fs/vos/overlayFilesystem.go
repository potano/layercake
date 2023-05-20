// Copyright © 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import (
	"io"
	"math"
	"path"
	"time"
	"strings"
	"syscall"
)

// Mount and inode operations for overlay file system


const (
	overlayFS_root_ino = 1
	whiteoutRdev = (0 << 8) + 0
	whiteoutPerms = 0
	opaqueXattr = "trusted.overlay.opaque"
	opaqueXattrValue = "y"
)

type overlayFilesystem struct {
	baseFilesystemData
	ns *namespaceType
	upperBaseDir, lowerBaseDir, workBaseDir *mfsOpenFile	// open directories at mount roots
	upperFS, lowerFS filesystemInstance
}


type ovfsInodeInterface interface {
	inodeType
	toWritableInode() inodeType
	getParent() dirInodeType
	setParent(dirInodeType)
	getActiveInode() (inodeType, bool)
	setActiveInode(inodeType, bool)
}

type ovfsInodeBase struct {
	st_type uint
	st_dev, st_ino, st_nlink uint64
	parent dirInodeType
	activeInode inodeType
	fs *overlayFilesystem
	pointsToUpper bool
}

type ovfsFileInode struct {
	ovfsInodeBase
}
type ovfsDirInode struct {
	ovfsInodeBase
	entries map[string]inodeType
	lowerCache map[string]inodeType
}
type ovfsLinkInode struct {
	ovfsInodeBase
}
type ovfsFifoInode struct {
	ovfsInodeBase
}
type ovfsSockInode struct {
	ovfsInodeBase
}
type ovfsChardevInode struct {
	ovfsInodeBase
}
type ovfsBlockdevInode struct {
	ovfsInodeBase
}


func newOverlayFilesystem(st_dev uint64, fstype, source string, ns *namespaceType,
		) (filesystemInstance, error) {
	fs := &overlayFilesystem{
		baseFilesystemData: baseFilesystemData{
			fstype: fstype,
			source: source,
			st_dev: st_dev,
			rootIno: overlayFS_root_ino,
			inodes: []inodeType{nil},
		},
		ns: ns,
	}
	return fs, nil
}


func (fs *overlayFilesystem) init(ns *namespaceType, mount *mountType) error {
	upperBaseDir := mount.sourceDir
	upperBaseDirInode := upperBaseDir.inode.(dirInodeType)
	fs.upperBaseDir = upperBaseDir
	fs.upperFS = ns.devices[upperBaseDirInode.dev()]
	lowerBaseDir := mount.source2Dir
	lowerBaseDirInode := lowerBaseDir.inode.(dirInodeType)
	fs.lowerBaseDir = lowerBaseDir
	fs.lowerFS = ns.devices[lowerBaseDirInode.dev()]
	fs.workBaseDir = mount.workDir
	rootInode := fs.newDirInode().(*ovfsDirInode)
	fs.baseFilesystemData.addInode(nil, "", rootInode)
	rootInode.setActiveInode(upperBaseDirInode, true)
	rootInode.setReadonlyInode(lowerBaseDirInode)
	return nil
}




func (fs *overlayFilesystem) resolveReadonlyPathIncrement(dirInode dirInodeType,
		pathname, name string) (inodeType, error) {
	roDir := dirInode.getReadonlyInode()
	if roDir == nil {
		return nil, nil
	}
	inode := roDir.(dirInodeType).direntByName(name)
	return inode, nil
}


func (fs *overlayFilesystem) inodeByInum(ino uint64) inodeType {
	if ino >= uint64(len(fs.inodes)) {
		return nil
	}
	return fs.inodes[ino]
}


func (fs *overlayFilesystem) makeUpperInode(dirInode *ovfsDirInode, mergeInode ovfsInodeInterface,
		) (inodeType, error) {
	abspath := fs.upperBaseDir.abspath
	fsPath := fs.findPathToInode(dirInode, mergeInode)
	mergeDirInode := fs.rootInode()
	upperDirInode := fs.upperBaseDir.inode.(dirInodeType)
	var upperInode inodeType
	var err error
	finalPathX := len(fsPath) - 1
	for pathX, name := range fsPath {
		abspath = path.Join(abspath, name)
		mergeInode = mergeDirInode.direntByName(name).(ovfsInodeInterface)
		if mergeInode == nil {
			return nil, ENOENT
		}
		if activeInode, isUpper := mergeInode.getActiveInode(); isUpper {
			upperInode = activeInode
		} else if activeInode := upperDirInode.direntByName(name); activeInode != nil {
			upperInode = activeInode
		} else {
			upperInode, err = fs.upperFS.resolveFromReadonlyFS(upperDirInode, abspath)
			if err != nil {
				return nil, err
			}
			if upperInode == nil {
				upperInode = fs.upperFS.duplicateInodeForFilesystem(mergeInode)
			}
			fs.upperFS.addInode(upperDirInode, name, upperInode)
		}
		mergeInode.setActiveInode(upperInode, true)
		if mergeInode.isDir() {
			mergeDirInode = mergeInode.(dirInodeType)
			upperDirInode = upperInode.(dirInodeType)
		} else if pathX < finalPathX {
			return nil, ENOTDIR
		}
	}
	return upperInode, nil
}


func (fs *overlayFilesystem) addInode(dirInode dirInodeType, name string, inode inodeType,
		) (inodeType, error) {
	_, err := fs.baseFilesystemData.addInode(dirInode, name, inode)
	if err != nil {
		return nil, err
	}
	mergeInode, inodeIsOvfs := inode.(ovfsInodeInterface)
	if !inodeIsOvfs {
		return inode, nil
	}
	mergeInode.setParent(dirInode)
	activeInode, _ := mergeInode.getActiveInode()
	if activeInode != nil {
		return inode, nil
	}
	mergeDir := dirInode.(*ovfsDirInode)
	lowerInode := mergeDir.lowerCache[name]
	upperDir, haveUpperDir := mergeDir.getActiveInode()
	makeOpaque := false
	if haveUpperDir && lowerInode != nil {
		upperInode := upperDir.(dirInodeType).direntByName(name)
		if upperInode != nil && inodeIsOvfsWhiteout(upperInode) {
			upperInode.decrementNlinks()
			upperDir.(dirInodeType).setDirent(name, nil)
			makeOpaque = inode.isDir()
		}
	}
	if lowerInode != nil && !makeOpaque {
		mergeInode.setActiveInode(lowerInode, false)
	} else {
		activeInode, err = fs.makeUpperInode(mergeDir, inode.(ovfsInodeInterface))
		if err != nil {
			return nil, err
		}
		if makeOpaque {
			activeInode.setXattr(opaqueXattr, opaqueXattrValue)
		}
		mergeInode.setActiveInode(activeInode, true)
	}
	return inode, nil
}


func (fs *overlayFilesystem) resolveFromReadonlyFS(
		dirInode dirInodeType, pathname string) (inodeType, error) {
	name := pathname
	if ipos := strings.LastIndexByte(name, '/'); ipos >= 0 {
		name = name[ipos + 1 :]
	}
	pid1 := fs.ns.process1()
	fsPath := path.Join(strings.Join(fs.findPathToDirectory(dirInode), "/"), name)
	_, _, upperInode, _, _, err := pid1.inodeAtPathWithParent(
		fs.upperBaseDir, fsPath)
	if err != nil && err != ENOENT {
		return nil, err
	}
	lowerMount, _, lowerInode, _, _, err := pid1.inodeAtPathWithParent(
		fs.lowerBaseDir, fsPath)
	if err != nil {
		return nil, err
	}
	if upperInode != nil && inodeIsOvfsWhiteout(upperInode) {
		return nil, nil
	}
	if lowerMount != fs.lowerBaseDir.mount {
		lowerInode = nil
	}
	if upperInode == nil && lowerInode == nil {
		return nil, nil
	}
	activeInode := upperInode
	usingUpper := true
	if activeInode == nil {
		activeInode = lowerInode
		usingUpper = false
	} else if activeInode.isDir() && lowerInode != nil && !inodeIsOpaque(activeInode) {
		activeInode.setReadonlyInode(lowerInode)
	}
	inode := fs.newMergeInodeLikeOriginal(activeInode, activeInode, usingUpper)
	return fs.addInode(dirInode, name, inode)
}


func (fs *overlayFilesystem) newMergeInodeLikeOriginal(orig,
		activeInode inodeType, usingUpper bool) inodeType {
	nodeType := orig.nodeType()
	base := ovfsInodeBase{
		st_type: nodeType,
		st_dev: orig.dev(),
		st_ino: orig.ino(),
		activeInode: activeInode,
		pointsToUpper: usingUpper,
		fs: fs}
	switch nodeType {
	case nodeTypeFile:
		return &ovfsFileInode{base}
	case nodeTypeDir:
		return &ovfsDirInode{base, map[string]inodeType{}, nil}
	case nodeTypeLink:
		return &ovfsLinkInode{base}
	case nodeTypeFifo:
		return &ovfsFifoInode{base}
	case nodeTypeSock:
		return &ovfsSockInode{base}
	case nodeTypeCharDev:
		return &ovfsChardevInode{base}
	case nodeTypeBlockDev:
		return &ovfsBlockdevInode{base}
	}
	return nil
}


func (fs *overlayFilesystem) findPathToInode(dirInode *ovfsDirInode, inode inodeType) []string {
	var parts []string
	for {
		name, found := dirInode.findInodeNameInDirectory(inode)
		if !found {
			return nil
		}
		parts = append(parts, name)
		parent := dirInode.getParent()
		if parent == nil {
			break
		}
		inode = dirInode
		dirInode = parent.(*ovfsDirInode)
	}
	for i, j := 0, len(parts) - 1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return parts
}


func (fs *overlayFilesystem) findPathToDirectory(dirInode dirInodeType) []string {
	parent := dirInode.getParent()
	if parent == nil {
		return nil
	}
	return fs.findPathToInode(parent.(*ovfsDirInode), dirInode)
}


func (fs *overlayFilesystem) makeOvfsWhiteout(dirInode dirInodeType, name string) error {
	whiteout := fs.upperFS.newChardevInode(whiteoutRdev, nil, nil)
	whiteout.setPerms(whiteoutPerms)
	_, err := fs.upperFS.addInode(dirInode, name, whiteout)
	return err
}


func inodeIsOvfsWhiteout(inode inodeType) bool {
	if chardev, is := inode.(deviceInodeType); is {
		return chardev.getRdev() == whiteoutRdev
	}
	return false
}


func inodeIsOpaque(inode inodeType) bool {
	value, _ := inode.xattrByName(opaqueXattr)
	return value == opaqueXattrValue
}



func (ino *ovfsInodeBase) nodeType() uint {
	return ino.st_type
}

func (ino *ovfsInodeBase) ino() uint64 {
	return ino.st_ino
}

func (ino *ovfsInodeBase) dev() uint64 {
	return ino.st_dev
}

func (ino *ovfsInodeBase) mode() uint64 {
	return ino.activeInode.mode()
}

func (ino *ovfsInodeBase) uid() uint64 {
	return ino.activeInode.uid()
}

func (ino *ovfsInodeBase) gid() uint64 {
	return ino.activeInode.gid()
}

func (ino *ovfsInodeBase) setDevIno(dev, inum uint64) {
	ino.st_dev = dev
	ino.st_ino = inum
}

func (ino *ovfsInodeBase) sameDevIno(dev, inum uint64) bool {
	return dev == ino.st_dev && inum == ino.st_ino
}

func (ino *ovfsInodeBase) setUidGid(uid, gid uint64) {
	ino.activeInode.setUidGid(uid, gid)
}

func (ino *ovfsInodeBase) nlink() uint64 {
	return ino.st_nlink
}

func (ino *ovfsInodeBase) decrementNlinks() bool {
	if ino.st_nlink > 1 {
		ino.st_nlink--
		return false
	}
	ino.st_nlink = 0
	return true
}

func (ino *ovfsInodeBase) incrementNlinks() error {
	if ino.st_nlink >= math.MaxInt64 {
		return syscall.EMLINK
	}
	ino.st_nlink++
	return nil
}

func (ino *ovfsInodeBase) isDir() bool {
	return ino.st_type == nodeTypeDir
}

func (ino *ovfsInodeBase) isWritable() bool {
	return ino.activeInode.isWritable()
}

func (ino *ovfsInodeBase) isSeekable() bool {
	return ino.activeInode.isSeekable()
}

func (ino *ovfsInodeBase) setPerms(perms uint64) {
	ino.toWritableInode().setPerms(perms)
}

func (ino *ovfsInodeBase) applyUmask(umask uint64) {
	ino.activeInode.applyUmask(umask)
}

func (ino *ovfsInodeBase) hasReadPermission(userOK, groupOK bool) bool {
	return ino.activeInode.hasReadPermission(userOK, groupOK)
}

func (ino *ovfsInodeBase) hasWritePermission(userOK, groupOK bool) bool {
	return ino.activeInode.hasWritePermission(userOK, groupOK)
}

func (ino *ovfsInodeBase) hasExecutePermission(userOK, groupOK bool) bool {
	return ino.activeInode.hasExecutePermission(userOK, groupOK)
}

func (ino *ovfsInodeBase) setMtime(tm time.Time) {
	ino.toWritableInode().setMtime(tm)
}

func (ino *ovfsInodeBase) setAtime(tm time.Time) {
	ino.toWritableInode().setAtime(tm)
}

func (ino *ovfsInodeBase) size() int64 {
	return ino.activeInode.size()
}

func (ino *ovfsInodeBase) Stat(stat_buf *Stat_t) error {
	return ino.activeInode.Stat(stat_buf)
}

func (ino *ovfsInodeBase) xattrByName(name string) (string, bool) {
	return ino.activeInode.xattrByName(name)
}

func (ino *ovfsInodeBase) xattrMap() map[string]string {
	return ino.activeInode.xattrMap()
}

func (ino *ovfsInodeBase) setXattr(name, value string) {
	ino.toWritableInode().setXattr(name, value)
}

func (ino *ovfsInodeBase) init(tp uint) error {
	return ino.activeInode.init(tp)
}

func (ino *ovfsInodeBase) setReadonlyInode(inode inodeType) {
	if ino.activeInode == nil {
		ino.activeInode = inode
		ino.pointsToUpper = false
		return
	}
	activeInode := ino.activeInode
	newDev, newIno := inode.dev(), inode.ino()
	for activeInode.dev() != newDev || activeInode.ino() != newIno {
		if upperRO := activeInode.getReadonlyInode(); upperRO != nil {
			activeInode = upperRO
		} else {
			activeInode.setReadonlyInode(inode)
			break
		}
	}
}

func (ino *ovfsInodeBase) getReadonlyInode() inodeType {
	if ino.pointsToUpper {
		return ino.activeInode.getReadonlyInode()
	}
	return ino.activeInode
}

func (ino *ovfsInodeBase) copyUp() error {
	if ino.pointsToUpper || ino.activeInode == nil {
		return nil
	}
	activeInode := ino.activeInode
	dirInode := ino.getParent().(*ovfsDirInode)
	activeInode.copyUp()
	if !ino.pointsToUpper {
		upperInode, err := ino.fs.makeUpperInode(dirInode, ino)
		if err != nil {
			return err
		}
		upperInode.setReadonlyInode(activeInode)
		upperInode.copyUp()
		ino.activeInode = upperInode
		ino.pointsToUpper = true
	}
	return nil
}

func (ino *ovfsInodeBase) open(of *mfsOpenFile) error {
	if of.writable && !ino.pointsToUpper {
		ino.copyUp()
	}
	return nil
}

func (ino *ovfsInodeBase) close(of *mfsOpenFile) {
}

func (ino *ovfsInodeBase) getMetadata() *inodeMetadataTransferRecord {
	if ino.activeInode != nil {
		return ino.activeInode.getMetadata()
	}
	baseInode := &mfsInodeBase{}
	baseInode.init(ino.st_type)
	return baseInode.getMetadata()
}

func (ino *ovfsInodeBase) setMetadata(rec *inodeMetadataTransferRecord) error {
	if !ino.pointsToUpper {
		return EINVAL
	}
	return ino.activeInode.setMetadata(rec)
}

func (ino *ovfsInodeBase) toWritableInode() inodeType {
	ino.copyUp()
	return ino.activeInode
}

func (ino *ovfsInodeBase) getParent() dirInodeType {
	return ino.parent
}

func (ino *ovfsInodeBase) setParent(p dirInodeType) {
	ino.parent = p
}

func (ino *ovfsInodeBase) getActiveInode() (inodeType, bool) {
	return ino.activeInode, ino.pointsToUpper
}

func (ino *ovfsInodeBase) setActiveInode(activeInode inodeType, isUpper bool) {
	ino.activeInode = activeInode
	ino.pointsToUpper = isUpper
}




func (fs *overlayFilesystem) newFileInode() fileInodeType {
	return &ovfsFileInode{ovfsInodeBase{st_type: nodeTypeFile, fs: fs}}
}

func (ino *ovfsFileInode) readFile(buf []byte, start int64) (int, error) {
	return ino.activeInode.(fileInodeType).readFile(buf, start)
}

func (ino *ovfsFileInode) writeFile(buf []byte, start int64) (int, error) {
	if !ino.pointsToUpper {
		ino.copyUp()
	}
	return ino.activeInode.(fileInodeType).writeFile(buf, start)
}

func (ino *ovfsFileInode) truncateFile() error {
	return ino.toWritableInode().(fileInodeType).truncateFile()
}


func (fs *overlayFilesystem) newDirInode() dirInodeType {
	return &ovfsDirInode{
		ovfsInodeBase{st_type: nodeTypeDir, fs: fs},
		map[string]inodeType{},
		nil,
	}
}

func (ino *ovfsDirInode) direntByName(name string) inodeType {
	return ino.entries[name]
}

func (ino *ovfsDirInode) direntMap() map[string]inodeType {
	if ino.lowerCache == nil {
		ino.setLowerCache()
	}
	if !ino.pointsToUpper {
		return ino.lowerCache
	}
	upperEntries := ino.activeInode.(dirInodeType).rawDirentMap()
	out := make(map[string]inodeType, len(upperEntries) + len(ino.lowerCache))
	for k, v := range ino.lowerCache {
		out[k] = v
	}
	for k, v := range upperEntries {
		if inodeIsOvfsWhiteout(v) {
			delete(out, k)
		} else {
			out[k] = v
		}
	}
	return out
}

func (ino *ovfsDirInode) setLowerCache() {
	var lowerSource map[string]inodeType
	if ino.pointsToUpper {
		roInode := ino.activeInode.getReadonlyInode()
		if roInode != nil {
			lowerSource = roInode.(dirInodeType).direntMap()
		}
	} else {
		lowerSource = ino.activeInode.(dirInodeType).direntMap()
	}
	out := make(map[string]inodeType, len(lowerSource))
	for k, v := range lowerSource {
		out[k] = v
	}
	ino.lowerCache = out
}

func (ino *ovfsDirInode) rawDirentMap() map[string]inodeType {
	return ino.entries
}

func (ino *ovfsDirInode) setDirent(name string, entry inodeType) error {
	if ino.lowerCache == nil {
		ino.setLowerCache()
	}
	_, lowerInodeExists := ino.lowerCache[name]
	if entry == nil {
		oldEnt := ino.entries[name]
		delete(ino.entries, name)
		if _, is := oldEnt.(ovfsInodeInterface); !is {
			return nil
		}
		if ino.pointsToUpper {
			upperDir := ino.activeInode.(dirInodeType)
			if upperInode := upperDir.direntByName(name); upperInode != nil {
				upperInode.decrementNlinks()
				upperDir.setDirent(name, nil)
			}
		} else {
			ino.copyUp()
		}
		if lowerInodeExists {
			upperDir := ino.activeInode.(dirInodeType)
			err := ino.fs.makeOvfsWhiteout(upperDir, name)
			if err != nil {
				return err
			}
		}
	} else {
		ino.entries[name] = entry
		if !ino.pointsToUpper {
			if !lowerInodeExists {
				ino.copyUp()
			}
		}
	}
	return nil
}

func (dir *ovfsDirInode) findInodeNameInDirectory(inode inodeType) (string, bool) {
	st_dev := inode.dev()
	st_ino := inode.ino()
	for name, ptr := range dir.entries {
		if ptr.sameDevIno(st_dev, st_ino) {
			return name, true
		}
	}
	return "", false
}



func (fs *overlayFilesystem) newLinkInode() linkInodeType {
	return &ovfsLinkInode{ovfsInodeBase{st_type: nodeTypeLink, fs: fs}}
}

func (ino *ovfsLinkInode) getLinkTarget() string {
	return ino.activeInode.(linkInodeType).getLinkTarget()
}

func (ino *ovfsLinkInode) setLinkTarget(target string) error {
	return ino.toWritableInode().(linkInodeType).setLinkTarget(target)
}


func (fs *overlayFilesystem) newFifoInode() fifoInodeType {
	return &ovfsFifoInode{ovfsInodeBase{st_type: nodeTypeFifo, fs: fs}}
}

func (ino *ovfsFifoInode) peekFifo() []byte {
	return ino.activeInode.(fifoInodeType).peekFifo()
}

func (ino *ovfsFifoInode) readFifo(buf []byte) (int, error) {
	return ino.activeInode.(fifoInodeType).readFifo(buf)
}

func (ino *ovfsFifoInode) writeFifo(buf []byte) (int, error) {
	return ino.toWritableInode().(fifoInodeType).writeFifo(buf)
}


func (fs *overlayFilesystem) newSockInode() sockInodeType {
	return &ovfsSockInode{ovfsInodeBase{st_type: nodeTypeSock, fs: fs}}
}


func (fs *overlayFilesystem) newCharDevInode(a uint64, r io.Reader, w io.Writer,
		) charDeviceInodeType {
	return &ovfsChardevInode{ovfsInodeBase{st_type: nodeTypeCharDev, fs: fs}}
}

func (ino *ovfsChardevInode) getRdev() uint64 {
	return ino.activeInode.(charDeviceInodeType).getRdev()
}

func (ino *ovfsChardevInode) readChardev(buf []byte) (int, error) {
	return ino.activeInode.(charDeviceInodeType).readChardev(buf)
}

func (ino *ovfsChardevInode) writeChardev(buf []byte) (int, error) {
	return ino.toWritableInode().(charDeviceInodeType).writeChardev(buf)
}


func (fs *overlayFilesystem) newBlockDevInode(r uint64) blockDeviceInodeType {
	return &ovfsBlockdevInode{ovfsInodeBase{st_type: nodeTypeBlockDev, fs: fs}}
}

func (ino *ovfsBlockdevInode) getRdev() uint64 {
	return ino.activeInode.(blockDeviceInodeType).getRdev()
}

