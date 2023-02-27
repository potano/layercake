// Copyright © 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import "path"


type filesystemMaker func (st_dev uint64, fstype, source string, ns *namespaceType,
	) (filesystemInstance, error)


type filesystemInstance interface {
	rootInode() dirInodeType
	addInode(dirInodeType, string, inodeType) (inodeType, error)
	inodeByInum(uint64) inodeType
	getSource() string
	getStDev() uint64
	populateAsRequired(string, map[string]inodeType)
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
			inode.(dirInodeType).setParent(dirInode.ino())
		}
	}
	inode.incrementNlinks()
	return inode, nil
}


func (fs *baseFilesystemData) inodeByInum(ino uint64) inodeType {
	return nil
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

