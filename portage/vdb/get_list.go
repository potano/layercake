package vdb

import (
	"fmt"
	"path"
	"strings"
	"potano.layercake/fs"
	"potano.layercake/portage/atom"
)


func GetInstalledPackageList(rootdir string) (*atom.AtomSet, error) {
	ps := atom.NewAtomSet(atom.GroupBySlot)
	pkgDbPath := path.Join(rootdir, "/var/db/pkg")
	if !fs.IsDir(pkgDbPath) {
		return nil, fmt.Errorf("installed package database %s not found", pkgDbPath)
	}
	cats, err := fs.Readdirnames(pkgDbPath)
	if err != nil {
		return nil, err
	}
	for _, category := range cats {
		catDirName := path.Join(pkgDbPath, category)
		names, err := fs.Readdirnames(catDirName)
		if err != nil {
			return nil, fmt.Errorf("error listing %s: %s", catDirName, err)
		}
		for _, namever := range names {
			av := &AvailableVersion{Directory: path.Join(catDirName, namever)}
			err := av.setAtom(category, namever)
			if err != nil {
				return nil, err
			}
			ps.Add(av)
		}
	}
	return ps, nil
}


func (av *AvailableVersion) setAtom(category, namever string) error {
	ca, err := atom.NewUnprefixedConcreteAtom(category + "/" + namever)
	if err != nil {
		return err
	}
	line, _, err := av.readFirstFile("IUSE_EFFECTIVE", "IUSE")
	ca.UseFlags = atom.NewUseFlagSetFromIUSE(line)
	line, _, err = av.readFileIfExists("USE")
	if err != nil {
		return err
	}
	ca.UseFlags.SetFlagsFromUSE(line)
	slot, err := av.readFile("SLOT")
	if err != nil {
		return err
	}
	if ind := strings.Index(slot, "/"); ind >= 0 {
		slot = slot[:ind]
	}
	ca.SetSlotAndSubslot(slot, "")
	av.ConcreteAtom = *ca
	return nil
}


func (av *AvailableVersion) readFile(name string) (string, error) {
	line, err := fs.ReadFile(path.Join(av.Directory, name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}


func (av *AvailableVersion) readFileIfExists(name string) (string, bool, error) {
	line, exists, err := fs.ReadFileIfExists(path.Join(av.Directory, name))
	if err != nil {
		return "", false, err
	}
	if !exists {
		return "", false, nil
	}
	return strings.TrimSpace(line), true, nil
}


func (av *AvailableVersion) readFirstFile(names ...string) (string, bool, error) {
	for _, name := range names {
		line, exists, err := av.readFileIfExists(name)
		if err != nil {
			return "", false, err
		}
		if exists {
			return line, true, nil
		}
	}
	return "", false, nil
}

