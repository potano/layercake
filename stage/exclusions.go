// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package stage


import (
	"potano.layercake/portage/atom"
	"potano.layercake/portage/vdb"
)


func (fl *FileList) UnstagedFileMap(installedSet *atom.AtomSet) (map[string]bool, error) {
	ssfm := map[string]bool{}
	for _, grp := range installedSet.Atoms {
		for _, atm := range *grp {
			files, err := vdb.GetAtomFileInfo(atm)
			if err != nil {
				return nil, err
			}
			for _, fe := range files {
				if _, staged := fl.entryMap[fe.Name]; !staged {
					ssfm[fe.Name] = true
				}
			}
		}
	}
	return ssfm, nil
}


func (fl *FileList) ExcludeFiles(excludable map[string]bool) {
	for name := range excludable {
		delete(fl.entryMap, name)
	}
}

