// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

// Mount and inode operations for storage-oriented (non-special) filesystems


const (
	storageFS_root_ino = 1	// in contrast to ext{2,3,4}, where ROOT_INO is 2
	storageFS_first_ino = 2	// in contrast to ext{2,3,4}, where GOOD_OLD_FIRST_INO is 11
)

type storageFilesystem struct {
	baseFilesystemData
}




func newStorageFilesystem(st_dev uint64, fstype, source string, ns *namespaceType,
		) (filesystemInstance, error) {
	fs := &storageFilesystem{
		baseFilesystemData{
			fstype: fstype,
			source: source,
			st_dev: st_dev,
			rootIno: storageFS_root_ino,
			inodes: []inodeType{nil},
		},
	}
	inode, _ := fs.addInode(nil, "", newBaseDirInode())
	inode.applyUmask(initUmask)
	return fs, nil
}



func (fs *storageFilesystem) inodeByInum(ino uint64) inodeType {
	if ino >= uint64(len(fs.inodes)) {
		return nil
	}
	return fs.inodes[ino]
}

