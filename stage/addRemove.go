// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package stage

import (
	"fmt"
	"path"
	"path/filepath"
	"potano.layercake/fs"
	"potano.layercake/defaults"
	"potano.layercake/portage/vdb"
)


func globFiles(pth string, recursive bool) ([]string, error) {
	matches, err := filepath.Glob(pth)
	if err != nil {
		return nil, err
	}
	if !recursive {
		return matches, nil
	}
	var out []string
	err = expandSubdirs(&out, matches)
	if err != nil {
		return nil, err
	}
	return out, nil
}


func expandSubdirs(out *[]string, input []string) error {
	for _, m := range input {
		*out = append(*out, m)
		if fs.IsDirNotSymlink(m) {
			names, err := fs.Readdirnames(m)
			if err != nil {
				return err
			}
			for i, name := range names {
				names[i] = path.Join(m, name)
			}
			err = expandSubdirs(out, names)
			if err != nil {
				return err
			}
		}
	}
	return nil
}


func (fl *FileList) removeFiles(entry lineInfo) error {
	treeroot := fl.rootDir
	leadLength := len(treeroot)
	name := entry.name
	if entry.hasWildcard {
		names, err := globFiles(path.Join(treeroot, name), false)
		if err != nil {
			return err
		}
		for _, m := range names {
			if _, have := fl.entryMap[m]; have {
				m = m[leadLength:]
				delete(fl.entryMap, m)
			}
		}
	} else if _, have := fl.entryMap[name]; !have {
		return fmt.Errorf("%s does not exist", name)
	} else {
		delete(fl.entryMap, name)
	}
	return nil
}


func (fl *FileList) addFiles(entry lineInfo) (err error) {
	if len(entry.source) > 0 {
		src, err := fl.resolveSourceLocation(entry.source)
		if err != nil {
			return err
		}
		entry.source = src
	}
	if entry.hasWildcard {
		err = fl.addFromWildcard(entry)
	} else {
		err = fl.addSingleFile(entry)
	}
	return
}


func (fl *FileList) resolveSourceLocation(name string) (string, error) {
	return fs.AdjustPrefixedPath(name, ".", func (symbol, tail string) (string, error) {
		if symbol == defaults.TreeRootDirPrefixName {
			return fl.rootDir, nil
		}
		return "", fmt.Errorf("unknown prefix name %s", symbol)
	})
}


func (fl *FileList) addFromWildcard(entry lineInfo) error {
	name, source := entry.name, entry.source
	var globname, prefix string
	var choplen int
	useSource := len(source) > 0
	if useSource {
		globname = source
		choplen = len(path.Dir(globname))
		prefix = name
	} else {
		globname = path.Join(fl.rootDir, name)
		choplen = len(fl.rootDir)
	}
	names, err := globFiles(globname, entry.ltype == vdb.FileType_dir)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return fmt.Errorf("no matching files for %s", globname)
	}
	for _, m := range names {
		e := entry
		e.ltype = vdb.FileType_none
		if useSource {
			e.name = path.Join(prefix, m[choplen:])
			e.source = m
		} else {
			e.name = m[choplen:]
		}
		err := fl.addSingleFile(e)
		if err != nil {
			return err
		}
	}
	return nil
}

