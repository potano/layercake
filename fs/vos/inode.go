// Copyright © 2022, 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos


import (
	"io"
	"math"
	"time"
	"syscall"
)


const (
	// st_mode permission-bit masks declared in kernel sources but not in <sys/stat.h>
	s_IRWXUGO = 0777	// bits which umask affects: normal permissions
	s_IALLUGO = 07777	// all permissions including setUID, setGID, and stick bit
)


type inodeType interface {
	nodeType() uint
	ino() uint64
	dev() uint64
	mode() uint64
	uid() uint64
	gid() uint64
	nlink() uint64
	setDevIno(uint64, uint64)
	sameDevIno(uint64, uint64) bool
	setUidGid(uint64, uint64)
	decrementNlinks() bool
	incrementNlinks() error
	isDir() bool
	isWritable() bool
	isSeekable() bool
	setPerms(uint64)
	applyUmask(uint64)
	hasReadPermission(bool, bool) bool
	hasWritePermission(bool, bool) bool
	hasExecutePermission(bool, bool) bool
	setMtime(time.Time)
	setAtime(time.Time)
	size() int64
	Stat(*Stat_t) error
	init(uint) error
	setReadonlyInode(inodeType)
	getReadonlyInode() inodeType
	copyUp() error
	open(*mfsOpenFile) error
	close(*mfsOpenFile)
	getMetadata() *inodeMetadataTransferRecord
	setMetadata(*inodeMetadataTransferRecord) error
}

type mfsInodeBase struct {
	st_type uint
	st_dev, st_ino, st_mode, st_nlink, st_uid, st_gid, st_parent uint64
	st_atim, st_mtim, st_ctim syscall.Timespec
	readonlyInode inodeType
}

type mfsFileInodeBase struct {
	mfsInodeBase
	contents []byte
}

type mfsDirInodeBase struct {
	mfsInodeBase
	parent dirInodeType
	entries map[string]inodeType
}

type mfsLinkInodeBase struct {
	mfsInodeBase
	target string
}

type mfsDeviceInodeBase struct {
	mfsInodeBase
	st_rdev uint64
}

type mfsSockInodeBase struct {
	mfsInodeBase
}

type mfsFifoInodeBase struct {
	mfsInodeBase
	contents []byte
}

type mfsChardevInodeBase struct {
	mfsDeviceInodeBase
	reader io.Reader
	writer io.Writer
}

type mfsBlockdevInodeBase struct {
	mfsDeviceInodeBase
}


type inodeMetadataTransferRecord struct {
	st_type uint
	st_mode, st_uid, st_gid, st_rdev uint64
	st_atim, st_mtim, st_ctim syscall.Timespec
	linkTarget string
	reader io.Reader
	writer io.Writer
}


const (
	nodeTypeNone = iota
	nodeTypeFile
	nodeTypeDir
	nodeTypeLink
	nodeTypeFifo
	nodeTypeSock
	nodeTypeCharDev
	nodeTypeBlockDev
)


type fileInodeType interface {
	inodeType
	readFile(buf []byte, start int64) (int, error)
	writeFile(buf[]byte, start int64) (int, error)
	truncateFile() error
}

type dirInodeType interface {
	inodeType
	direntByName(name string) inodeType
	direntMap() map[string]inodeType
	rawDirentMap() map[string]inodeType
	setDirent(name string, entry inodeType) error
	getParent() dirInodeType
	setParent(dirInodeType)
}

type linkInodeType interface {
	inodeType
	getLinkTarget() string
	setLinkTarget(target string) error
}

type fifoInodeType interface {
	inodeType
	peekFifo() []byte
	readFifo(buf []byte) (int, error)
	writeFifo(buf []byte) (int, error)
}

type sockInodeType interface {
	inodeType
}

type deviceInodeType interface {
	inodeType
	getRdev() uint64
}

type charDeviceInodeType interface {
	deviceInodeType
	readChardev(buf []byte) (int, error)
	writeChardev(buf []byte) (int, error)
}

type blockDeviceInodeType interface {
	deviceInodeType
}

func (ino *mfsInodeBase) init(iType uint) error {
	var mode uint64
	switch iType {
	case nodeTypeFile:
		mode = syscall.S_IFREG
	case nodeTypeDir:
		mode = syscall.S_IFDIR | syscall.S_IXUSR | syscall.S_IXGRP | syscall.S_IXOTH
	case nodeTypeLink:
		mode = syscall.S_IFLNK | syscall.S_IXUSR | syscall.S_IXGRP | syscall.S_IXOTH
	case nodeTypeFifo:
		mode = syscall.S_IFIFO
	case nodeTypeSock:
		mode = syscall.S_IFSOCK
	case nodeTypeCharDev:
		mode = syscall.S_IFCHR
	case nodeTypeBlockDev:
		mode = syscall.S_IFBLK
	default:
		return syscall.EINVAL
	}
	ino.st_type = iType
	ino.st_mode = mode | syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP | syscall.S_IWGRP |
		syscall.S_IROTH | syscall.S_IWOTH
	ino.setAllTimesNow()
	return nil
}

func (ino *mfsInodeBase) setDevIno(st_dev, st_ino uint64) {
	ino.st_dev = st_dev
	ino.st_ino = st_ino
}

func (ino *mfsInodeBase) sameDevIno(st_dev, st_ino uint64) bool {
	return st_dev == ino.st_dev && st_ino == ino.st_ino
}

func (ino *mfsInodeBase) setReadonlyInode(inode inodeType) {
	ino.readonlyInode = inode
}

func (ino *mfsInodeBase) getReadonlyInode() inodeType {
	return ino.readonlyInode
}

func (ino *mfsInodeBase) copyUp() error {
	return nil
}

func (ino *mfsInodeBase) getMetadata() *inodeMetadataTransferRecord {
	return &inodeMetadataTransferRecord{
		st_type: ino.st_type,
		st_mode: ino.st_mode,
		st_uid: ino.st_uid,
		st_gid: ino.st_gid,
		st_atim: ino.st_atim,
		st_mtim: ino.st_mtim,
		st_ctim: ino.st_ctim,
	}
}

func (ino *mfsInodeBase) setMetadata(rec *inodeMetadataTransferRecord) error {
	if rec.st_type != ino.st_type {
		return EINVAL
	}
	ino.st_mode = rec.st_mode
	ino.st_uid = rec.st_uid
	ino.st_gid = rec.st_gid
	ino.st_atim = rec.st_atim
	ino.st_mtim = rec.st_mtim
	ino.st_ctim = rec.st_ctim
	return nil
}

//Will migrate to foreign-filesystem handler
func (ino *mfsInodeBase) copyUpMetadata(keepReadonlyInode bool) (inodeType, error) {
	if readonlyInode := ino.readonlyInode; readonlyInode != nil {
		var stat_buf Stat_t
		if !keepReadonlyInode {
			ino.readonlyInode = nil
		}
		err := readonlyInode.Stat(&stat_buf)
		if err != nil {
			return readonlyInode, err
		}
		ino.setPerms(uint64(stat_buf.Mode))
		ino.st_uid = uint64(stat_buf.Uid)
		ino.st_gid = uint64(stat_buf.Gid)
		ino.st_atim = stat_buf.Atim
		ino.st_mtim = stat_buf.Mtim
		ino.st_ctim = stat_buf.Ctim
		return readonlyInode, nil
	}
	return nil, nil
}

//Will migrate to foreign-filesystem handler
func (ino *mfsInodeBase) setMetadataFromInode(source inodeType) {
	var stat_buf Stat_t
	err := source.Stat(&stat_buf)
	if err == nil {
		ino.setPerms(uint64(stat_buf.Mode))
		ino.st_uid = uint64(stat_buf.Uid)
		ino.st_gid = uint64(stat_buf.Gid)
		ino.st_atim = stat_buf.Atim
		ino.st_mtim = stat_buf.Mtim
		ino.st_ctim = stat_buf.Ctim
	}
}

func (ino *mfsInodeBase) open(of *mfsOpenFile) error {
	return nil
}

func (ino *mfsInodeBase) close(of *mfsOpenFile) {
}

func (ino *mfsInodeBase) nodeType() uint {
	return ino.st_type
}

func (ino *mfsInodeBase) ino() uint64 {
	return ino.st_ino
}

func (ino *mfsInodeBase) dev() uint64 {
	return ino.st_dev
}

func (ino *mfsInodeBase) mode() uint64 {
	return ino.st_mode
}

func (ino *mfsInodeBase) uid() uint64 {
	return ino.st_uid
}

func (ino *mfsInodeBase) gid() uint64 {
	return ino.st_gid
}

func (ino *mfsInodeBase) setUidGid(uid, gid uint64) {
	ino.copyUp()
	ino.st_uid = uid
	ino.st_gid = gid
}

func (ino *mfsInodeBase) getNlinks() uint64 {
	return ino.st_nlink
}

func timeToTimespec(tm time.Time) syscall.Timespec {
	return syscall.NsecToTimespec(tm.UnixNano())
}

func timespecToTime(t syscall.Timespec) time.Time {
	return time.Unix(0, syscall.TimespecToNsec(t))
}

func (ino *mfsInodeBase) getMtime() time.Time {
	return timespecToTime(ino.st_mtim)
}

func (ino *mfsInodeBase) setMtimeNow() {
	ino.copyUp()
	ino.st_mtim = timeToTimespec(time.Now())
}

func (ino *mfsInodeBase) setAllTimesNow() {
	ino.copyUp()
	timespec := timeToTimespec(time.Now())
	ino.st_atim = timespec
	ino.st_ctim = timespec
	ino.st_mtim = timespec
}

func (ino *mfsInodeBase) setAtime(tm time.Time) {
	ino.copyUp()
	ino.st_atim = timeToTimespec(tm)
}

func (ino *mfsInodeBase) setMtime(tm time.Time) {
	ino.copyUp()
	ino.st_mtim = timeToTimespec(tm)
}

func (ino *mfsInodeBase) setPerms(perms uint64) {
	ino.copyUp()
	ino.st_mode = uint64(int64(ino.st_mode) & ^s_IALLUGO) | (perms & s_IALLUGO)
}

func (ino *mfsInodeBase) applyUmask(umask uint64) {
	// Normal RWX bits only; see src/linux/sys.c in the kernel sources
	ino.st_mode &= ^(s_IRWXUGO & umask)
}

func (ino *mfsInodeBase) nlink() uint64 {
	return ino.st_nlink
}

func (ino *mfsInodeBase) decrementNlinks() bool {
	return ino.decNlink()
}

func (ino *mfsInodeBase) incrementNlinks() error {
	return ino.incNlink()
}

func (ino *mfsInodeBase) decNlink() bool {
	if ino.st_nlink > 0 {
		ino.st_nlink--
	}
	return ino.st_nlink == 0
}

func (ino *mfsInodeBase) incNlink() error {
	if ino.st_nlink >= math.MaxInt64 {
		return syscall.EMLINK
	}
	ino.st_nlink++
	return nil
}

func (ino *mfsInodeBase) baseStat(stat_buf *Stat_t) error {
	stat_buf.Dev = uint64(ino.st_dev)
	stat_buf.Ino = uint64(ino.st_ino)
	stat_buf.Mode = uint32(ino.st_mode)
	stat_buf.Nlink = uint64(ino.st_nlink)
	stat_buf.Uid = uint32(ino.st_uid)
	stat_buf.Gid = uint32(ino.st_gid)
	stat_buf.Rdev, stat_buf.Size, stat_buf.Blksize, stat_buf.Blocks = 0, 0, 0, 0
	stat_buf.Atim = ino.st_atim
	stat_buf.Mtim = ino.st_mtim
	stat_buf.Ctim = ino.st_ctim
	return nil
}

func (ino *mfsInodeBase) Stat(stat_buf *Stat_t) error {
	return ino.baseStat(stat_buf)
}

func (ino *mfsInodeBase) isWritable() bool {
	return (ino.st_mode & (syscall.S_IWUSR | syscall.S_IWGRP | syscall.S_IWOTH)) > 0
}

func (ino *mfsInodeBase) hasReadPermission(userOK, groupOK bool) bool {
	return (ino.st_mode & syscall.S_IROTH) > 0 ||
		(groupOK && (ino.st_mode & syscall.S_IRGRP) > 0) ||
		(userOK && (ino.st_mode & syscall.S_IRUSR) > 0)
}

func (ino *mfsInodeBase) hasWritePermission(userOK, groupOK bool) bool {
	return (ino.st_mode & syscall.S_IWOTH) > 0 ||
		(groupOK && (ino.st_mode & syscall.S_IWGRP) > 0) ||
		(userOK && (ino.st_mode & syscall.S_IWUSR) > 0)
}

func (ino *mfsInodeBase) hasExecutePermission(userOK, groupOK bool) bool {
	return (ino.st_mode & syscall.S_IXOTH) > 0 ||
		(groupOK && (ino.st_mode & syscall.S_IXGRP) > 0) ||
		(userOK && (ino.st_mode & syscall.S_IXUSR) > 0)
}

func (ino *mfsInodeBase) size() int64 {
	return 0
}

func (ino *mfsInodeBase) isDir() bool {
	return false
}

func (ino *mfsInodeBase) isSeekable() bool {
	return true
}


func newBaseFileInode() *mfsFileInodeBase {
	inode := &mfsFileInodeBase{}
	inode.init(nodeTypeFile)
	return inode
}

func (ino *mfsFileInodeBase) Stat(stat_buf *Stat_t) error {
	err := ino.baseStat(stat_buf)
	stat_buf.Size = int64(len(ino.contents))
	return err
}

func (ino *mfsFileInodeBase) size() int64 {
	return int64(len(ino.contents))
}

func (ino *mfsFileInodeBase) readFile(buf []byte, start int64) (int, error) {
	if start < 0 {
		return 0, EINVAL
	}
	if ino.readonlyInode != nil {
		return ino.readonlyInode.(fileInodeType).readFile(buf, start)
	}
	readLen := int64(len(ino.contents)) - start
	if readLen < 0 {
		readLen = 0
	} else if readLen > int64(len(buf)) {
		readLen = int64(len(buf))
	}
	if readLen > 0 {
		copy(buf, ino.contents[start:start+readLen])
	}
	return int(readLen), nil
}

func (ino *mfsFileInodeBase) writeFile(buf []byte, start int64) (int, error) {
	if start < 0 {
		return 0, syscall.EINVAL
	}
	lenToWrite := len(buf)
	contentLen := len(ino.contents)
	if start == 0 && lenToWrite >= contentLen {
		ino.contents = buf
		buf = buf[:0]
	} else if int(start) < contentLen {
		toModify := contentLen - int(start)
		if toModify > lenToWrite {
			toModify = lenToWrite
		}
		copy(ino.contents[int(start):int(start)+toModify], buf[:toModify])
		buf = buf[toModify:]
	} else if int(start) > contentLen {
		ino.contents = append(ino.contents, make([]byte, int(start) - contentLen)...)
	}
	if len(buf) > 0 {
		ino.contents = append(ino.contents, buf...)
	}
	ino.setMtimeNow()
	return lenToWrite, nil
}

func (ino *mfsFileInodeBase) truncateFile() error {
	ino.contents = ino.contents[:0]
	ino.setMtimeNow()
	return nil
}

func (ino *mfsFileInodeBase) open(of *mfsOpenFile) error {
	if of.writable && ino.readonlyInode != nil {
		ino.copyUp()
	}
	return nil
}

func (ino *mfsFileInodeBase) copyUp() error {
	if ino.readonlyInode != nil {
		buf := make([]byte, ino.readonlyInode.(fileInodeType).size())
		_, err := ino.readonlyInode.(fileInodeType).readFile(buf, 0)
		if err != nil {
			return err
		}
		_, err = ino.writeFile(buf, 0)
		if err != nil {
			return err
		}
		ino.setMetadata(ino.readonlyInode.getMetadata())
		ino.setMtimeNow()
		ino.readonlyInode = nil
	}
	return nil
}



func newBaseDirInode() *mfsDirInodeBase {
	inode := &mfsDirInodeBase{entries: map[string]inodeType{}}
	inode.init(nodeTypeDir)
	return inode
}

func (ino *mfsDirInodeBase) copyUp() error {
	if ino.readonlyInode != nil {
		ino.setMetadata(ino.readonlyInode.getMetadata())
		ino.setMtimeNow()
	}
	return nil
}

func (ino *mfsDirInodeBase) isDir() bool {
	return true
}

func (ino *mfsDirInodeBase) direntByName(name string) inodeType {
	if len(name) == 0 {
		return nil
	}
	return ino.entries[name]
}

func (ino *mfsDirInodeBase) direntMap() map[string]inodeType {
	return ino.entries
}

func (ino *mfsDirInodeBase) rawDirentMap() map[string]inodeType {
	return ino.entries
}

func (ino *mfsDirInodeBase) setDirent(name string, entry inodeType) error {
	if entry == nil {
		delete(ino.entries, name)
	} else {
		ino.entries[name] = entry
	}
	ino.setMtimeNow()
	return nil
}

func (ino *mfsDirInodeBase) getParent() dirInodeType {
	return ino.parent
}

func (ino *mfsDirInodeBase) setParent(p dirInodeType) {
	ino.parent = p
}


func newBaseLinkInode() *mfsLinkInodeBase {
	inode := &mfsLinkInodeBase{}
	inode.init(nodeTypeLink)
	return inode
}

func (ino *mfsLinkInodeBase) copyUp() error {
	if ino.readonlyInode != nil {
		ino.setMetadata(ino.readonlyInode.getMetadata())
		ino.readonlyInode = nil
	}
	return nil
}

func (ino *mfsLinkInodeBase) getMetadata() *inodeMetadataTransferRecord {
	rec := ino.mfsInodeBase.getMetadata()
	rec.linkTarget = ino.target
	return rec
}

func (ino *mfsLinkInodeBase) setMetadata(rec *inodeMetadataTransferRecord) error {
	err := ino.mfsInodeBase.setMetadata(rec)
	ino.target = rec.linkTarget
	return err
}

func (ino *mfsLinkInodeBase) getLinkTarget() string {
	return ino.target
}

func (ino *mfsLinkInodeBase) setLinkTarget(target string) error {
	ino.target = target
	ino.setMtimeNow()
	return nil
}


func newBaseFifoInode() *mfsFifoInodeBase {
	inode := &mfsFifoInodeBase{}
	inode.init(nodeTypeFifo)
	return inode
}

func (ino *mfsFifoInodeBase) isSeekable() bool {
	return false
}

func (ino *mfsFifoInodeBase) peekFifo() []byte {
	return ino.contents
}

func (ino *mfsFifoInodeBase) readFifo(buf []byte) (int, error) {
	n := len(buf)
	if n > len(ino.contents) {
		n = len(ino.contents)
	}
	copy(buf, ino.contents[:n])
	ino.contents = ino.contents[n:]
	return n, nil
}

func (ino *mfsFifoInodeBase) writeFifo(buf []byte) (int, error) {
	ino.contents = append(ino.contents, buf...)
	return len(buf), nil
}


func newBaseSockInode() *mfsSockInodeBase {
	inode := &mfsSockInodeBase{}
	inode.init(nodeTypeSock)
	return inode
}


func (ino *mfsDeviceInodeBase) getRdev() uint64 {
	return ino.st_rdev
}

func (ino *mfsDeviceInodeBase) Stat(stat_buf *Stat_t) error {
	err := ino.baseStat(stat_buf)
	stat_buf.Rdev = ino.st_rdev
	return err
}


func newBaseChardevInode(st_rdev uint64, reader io.Reader, writer io.Writer) *mfsChardevInodeBase {
	inode := &mfsChardevInodeBase{reader: reader, writer: writer}
	inode.st_rdev = st_rdev
	inode.init(nodeTypeCharDev)
	return inode
}

func (ino *mfsChardevInodeBase) copyUp() error {
	if ino.readonlyInode != nil {
		ino.setMetadata(ino.readonlyInode.getMetadata())
		ino.readonlyInode = nil
	}
	return nil
}

func (ino *mfsChardevInodeBase) getMetadata() *inodeMetadataTransferRecord {
	rec := ino.mfsInodeBase.getMetadata()
	rec.st_rdev = ino.st_rdev
	rec.reader = ino.reader
	rec.writer = ino.writer
	return rec
}

func (ino *mfsChardevInodeBase) setMetadata(rec *inodeMetadataTransferRecord) error {
	err := ino.mfsInodeBase.setMetadata(rec)
	rec.st_rdev = ino.st_rdev
	rec.reader = ino.reader
	rec.writer = ino.writer
	return err
}

func (ino *mfsChardevInodeBase) isSeekable() bool {
	return false
}

func (ino *mfsChardevInodeBase) readChardev(buf []byte) (int, error) {
	if ino.reader == nil {
		return 0, nil
	}
	return ino.reader.Read(buf)
}

func (ino *mfsChardevInodeBase) writeChardev(buf []byte) (int, error) {
	if ino.writer == nil {
		return 0, nil
	}
	return ino.writer.Write(buf)
}


func newBaseBlockdevInode(st_rdev uint64) *mfsBlockdevInodeBase {
	inode := &mfsBlockdevInodeBase{}
	inode.st_rdev = st_rdev
	inode.init(nodeTypeBlockDev)
	return inode
}

func (ino *mfsBlockdevInodeBase) copyUp() error {
	if ino.readonlyInode != nil {
		ino.setMetadata(ino.readonlyInode.getMetadata())
		ino.readonlyInode = nil
	}
	return nil
}

func (ino *mfsBlockdevInodeBase) getMetadata() *inodeMetadataTransferRecord {
	rec := ino.mfsInodeBase.getMetadata()
	rec.st_rdev = ino.st_rdev
	return rec
}

func (ino *mfsBlockdevInodeBase) setMetadata(rec *inodeMetadataTransferRecord) error {
	err := ino.mfsInodeBase.setMetadata(rec)
	rec.st_rdev = ino.st_rdev
	return err
}



func applyMissingDirents(target, source map[string]inodeType) map[string]inodeType {
	if target == nil {
		target = map[string]inodeType{}
	}
	for key, val := range source {
		if _, exists := target[key]; !exists {
			target[key] = val
		}
	}
	return target
}


func findInodeNameInDirectory(dir dirInodeType, inode inodeType) (string, bool) {
	st_dev := inode.dev()
	st_ino := inode.ino()
	if dir != nil {
		for name, ptr := range dir.direntMap() {
			if ptr == inode || ptr.sameDevIno(st_dev, st_ino) {
				return name, true
			}
		}
	}
	return "", false
}

