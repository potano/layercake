// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package stage


import (
	"fmt"
	"path"
	"strings"
	"potano.layercake/fs"
	"potano.layercake/defaults"
	"potano.layercake/portage/vdb"
)


type symlinkRecovery struct {
	nogoPaths, nogoNames map[string]bool
	fl *FileList
}


func (fl *FileList) RecoverMissingLinks() error {
	nogoPaths := map[string]bool{}
	nogoNames := map[string]bool{}
	for _, name := range strings.Fields(defaults.DoNotTraverse) {
		if name[0] == '/' {
			nogoPaths[name] = true
		} else {
			nogoNames[name] = true
		}
	}
	sr := symlinkRecovery{nogoPaths, nogoNames, fl}
	return sr.addMissingLinks("/")
}


func (sr symlinkRecovery) addMissingLinks(dir string) error {
	if sr.nogoPaths[dir] {
		return nil
	}
	fl := sr.fl
	dirpath := path.Join(fl.rootDir, dir)
	matches, err := fs.Readdirnames(dirpath)
	if err != nil {
		return err
	}
	for _, match := range matches {
		matchname := path.Join(dir, match)
		matchpath := path.Join(fl.rootDir, matchname)
		if fs.IsSymlink(matchpath) {
			// skip symlinks that were already added to the file set
			if _, exists := fl.entryMap[matchname]; exists {
				continue
			}
			target, err := fl.ultimateSymlinkTarget(matchname, matchpath)
			if err != nil {
				return err
			}
			if _, exists := fl.entryMap[target]; exists {
				err := fl.addFiles(lineInfo{
					ltype: vdb.FileType_symlink,
					name: matchname})
				if err != nil {
					return err
				}
			}
		} else if fs.IsDir(matchpath) {
			if !sr.nogoNames[match] {
				err := sr.addMissingLinks(matchname)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}


// Needed because eselect's indirection technique may result in symlinks to symlinks
func (fl *FileList) ultimateSymlinkTarget(source, absPath string) (string, error) {
	linkCount := 1
	relPath := source
	for {
		target, err := fs.Readlink(absPath)
		if err != nil {
			return "", err
		}
		if target[0] != '/' {
			target = path.Join(path.Dir(relPath), target)
		}
		absPath = path.Join(fl.rootDir, target)
		if ! fs.IsSymlink(absPath) {
			return target, nil
		}
		linkCount++
		if linkCount > defaults.MaxSymlinkChain {
			break
		}
		relPath = target
	}
	return "", fmt.Errorf("symlink chain from %s is too long", source)
}


func (fl *FileList) AddDirectoriesByName(dirNames []string) error {
	choplen := len(fl.rootDir)
	if choplen == 1 {
		choplen = 0
	}
	for _, name := range dirNames {
		entry := lineInfo{
			ltype: vdb.FileType_dir,
			name: name[choplen:] + "/*"}
		err := fl.addFromWildcard(entry)
		if err != nil {
			return err
		}
	}
	return nil
}


func (fl *FileList) AddMissingStageDirs() error {
	stageDirs := map[string]bool{}
	for _, entry := range fl.entryMap {
		dir := path.Dir(entry.name)
		if len(dir) > 0 {
			stageDirs[dir] = true
		}
	}
	for dir := range stageDirs {
		for {
			if _, exists := fl.entryMap[dir]; ! exists {
				err := fl.addFiles(lineInfo{
					ltype: vdb.FileType_dir,
					name: dir})
				if err != nil {
					return err
				}
			}
			pos := strings.LastIndexByte(dir, '/')
			if pos < 1 {
				break
			}
			dir = dir[:pos]
		}
	}
	return nil
}

