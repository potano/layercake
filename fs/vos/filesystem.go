// Copyright © 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import (
	"io"
	"path"
	"strings"
)


type filesystemMaker func (st_dev uint64, fstype, source string, ns *namespaceType,
	) (filesystemInstance, error)


type filesystemInstance interface {
	rootInode() dirInodeType
	addInode(dirInodeType, string, inodeType) (inodeType, error)
	inodeByInum(uint64) inodeType
	inodeList() []inodeType
	getSource() string
	getStDev() uint64
	populateAsRequired(string, map[string]inodeType)
	resolveFromReadonlyFS(dirInodeType, string) (inodeType, error)
	duplicateInodeForFilesystem(inodeType) inodeType
	newFileInode() fileInodeType
	newDirInode() dirInodeType
	newLinkInode() linkInodeType
	newFifoInode() fifoInodeType
	newSockInode() sockInodeType
	newChardevInode(uint64, io.Reader, io.Writer) charDeviceInodeType
	newBlockdevInode(uint64) blockDeviceInodeType
}


type baseFilesystemData struct {
	fstype, source string
	st_dev, rootIno uint64
	inodes []inodeType
}


func (fs *baseFilesystemData) rootInode() dirInodeType {
	return fs.inodes[fs.rootIno].(dirInodeType)
}


func (fs *baseFilesystemData) addInode(dirInode dirInodeType, name string, inode inodeType,
		) (inodeType, error) {
	inode.setDevIno(fs.st_dev, uint64(len(fs.inodes)))
	fs.inodes = append(fs.inodes, inode)
	if dirInode != nil {
		err := dirInode.setDirent(name, inode)
		if err != nil {
			return nil, err
		}
		if inode.isDir() {
			inode.(dirInodeType).setParent(dirInode)
		}
	}
	inode.incrementNlinks()
	return inode, nil
}


func (fs *baseFilesystemData) inodeByInum(ino uint64) inodeType {
	if ino >= uint64(len(fs.inodes)) {
		return nil
	}
	return fs.inodes[ino]
}


func (fs *baseFilesystemData) inodeList() []inodeType {
	return fs.inodes
}


func (fs *baseFilesystemData) getSource() string {
	return fs.source
}


func (fs *baseFilesystemData) getStDev() uint64 {
	return fs.st_dev
}


type inoRef struct {
	ino, parent uint64
}


func (fs *baseFilesystemData) populateAsRequired(prefix string, required map[string]inodeType) {
	linearized := make(map[string]inoRef, len(fs.inodes))
	fs.linearizeTree(prefix, fs.rootInode(), linearized)
	D.printf("linearized: %#v\n", linearized)
}


func (fs *baseFilesystemData) linearizeTree(prefix string, dir dirInodeType,
		entries map[string]inoRef) {
	parent_ino := dir.ino()
	for name, ent := range dir.rawDirentMap() {
		pathname := path.Join(prefix, name)
		entries[pathname] = inoRef{ent.ino(), parent_ino}
		if childDir, is := ent.(dirInodeType); is {
			fs.linearizeTree(pathname, childDir, entries)
		}
	}
}



func (fs *baseFilesystemData) newFileInode() fileInodeType {
	return newBaseFileInode()
}

func (fs *baseFilesystemData) newDirInode() dirInodeType {
	return newBaseDirInode()
}

func (fs *baseFilesystemData) newLinkInode() linkInodeType {
	return newBaseLinkInode()
}

func (fs *baseFilesystemData) newFifoInode() fifoInodeType {
	return newBaseFifoInode()
}

func (fs *baseFilesystemData) newSockInode() sockInodeType {
	return newBaseSockInode()
}

func (fs *baseFilesystemData) newChardevInode(
		st_rdev uint64, reader io.Reader, writer io.Writer) charDeviceInodeType {
	return newBaseChardevInode(st_rdev, reader, writer)
}

func (fs *baseFilesystemData) newBlockdevInode(	st_rdev uint64) blockDeviceInodeType {
	return newBaseBlockdevInode(st_rdev)
}



func (fs *baseFilesystemData) resolveReadonlyPathIncrement(dirInode dirInodeType,
		pathname, name string) (inodeType, error) {
	return nil, nil
}

func (fs *baseFilesystemData) resolveFromReadonlyFS(
		dirInode dirInodeType, pathname string) (inodeType, error) {
	name := pathname
	if ipos := strings.LastIndexByte(name, '/'); ipos >= 0 {
		name = name[ipos + 1 :]
	}
	roInode, err := fs.resolveReadonlyPathIncrement(dirInode, pathname, name)
	if roInode == nil || err != nil {
		return roInode, err
	}
	inode := fs.duplicateInodeForFilesystem(roInode)
	inode.setReadonlyInode(roInode)
	fs.addInode(dirInode, name, inode)
	return inode, nil
}

func (fs *baseFilesystemData) duplicateInodeForFilesystem(orig inodeType) (inode inodeType) {
	tfer := orig.getMetadata()
	switch orig.nodeType() {
	case nodeTypeFile:
		inode = fs.newFileInode()
	case nodeTypeDir:
		inode = fs.newDirInode()
	case nodeTypeLink:
		inode = fs.newLinkInode()
	case nodeTypeFifo:
		inode = fs.newFifoInode()
	case nodeTypeSock:
		inode = fs.newSockInode()
	case nodeTypeCharDev:
		inode = fs.newChardevInode(0, nil, nil)
	case nodeTypeBlockDev:
		inode = fs.newBlockdevInode(0)
	}
	inode.setMetadata(tfer)
	return inode
}

