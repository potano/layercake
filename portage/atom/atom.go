// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package atom

import "sort"


type Atom interface {
	String() string
	PackageName() string
	ComparisonString() string
	GetSlot() string
	SetSlotAndSubslot(string, string)
	GetUseFlagMap() UseFlagMap
	GetUseFlagSet() UseFlagSet
	GetGroupingKey(int) string
}


type AtomSlice []Atom


type AtomSet struct {
	Atoms map[string]*AtomSlice
	Grouping int
}


const (
	GroupByVersion = iota
	GroupBySlot
)


func NewAtomSet(grouping int) *AtomSet {
	return &AtomSet{map[string]*AtomSlice{}, grouping}
}


func (as *AtomSet) Add(entry Atom) {
	pkgname := entry.PackageName()
	slice := as.Atoms[pkgname]
	if slice == nil {
		slice = &AtomSlice{}
		as.Atoms[pkgname] = slice
	}
	key := entry.GetGroupingKey(as.Grouping)
	insertPos := -1
	for i, at := range *slice {
		itemkey := at.GetGroupingKey(as.Grouping)
		if itemkey == key {
			return
		}
		if itemkey < key {
			insertPos = i
		}
	}
	*slice = append(*slice, entry)
	if insertPos >= 0 {
		copy((*slice)[insertPos + 1:], (*slice)[insertPos:])
		(*slice)[insertPos] = entry
	}
}


func (as *AtomSet) Remove(entry Atom) {
	pkgname := entry.PackageName()
	slice := as.Atoms[pkgname]
	if slice == nil {
		return
	}
	key := entry.GetGroupingKey(as.Grouping)
	for i, item := range *slice {
		if item.GetGroupingKey(as.Grouping) == key {
			if len(*slice) == 1 {
				delete(as.Atoms, pkgname)
			} else {
				*slice = append((*slice)[:i], (*slice)[i+1:]...)
			}
			break
		}
	}
}


func (as *AtomSet) Get(entry Atom) Atom {
	pkgname := entry.PackageName()
	slice := as.Atoms[pkgname]
	if slice == nil {
		return nil
	}
	key := entry.GetGroupingKey(as.Grouping)
	for _, item := range *slice {
		if item.GetGroupingKey(as.Grouping) == key {
			return item
		}
	}
	return nil
}


func (as *AtomSet) GetByName(pkgname string) AtomSlice {
	pslice := as.Atoms[pkgname]
	if pslice == nil {
		return nil
	}
	return *pslice
}


func (as *AtomSet) SortedAtoms() AtomSlice {
	names := make([]string, 0, len(as.Atoms))
	numVersions := 0
	for name, grp := range as.Atoms {
		names = append(names, name)
		numVersions += len(*grp)
	}
	sort.Strings(names)
	atoms := make(AtomSlice, 0, numVersions)
	for _, name := range names {
		grp := *as.Atoms[name]
		for i := len(grp) - 1; i >= 0; i-- {
			atoms = append(atoms, grp[i])
		}
	}
	return atoms
}

