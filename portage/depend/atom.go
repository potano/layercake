// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package depend

import (
	"potano.layercake/portage/atom"
	"potano.layercake/portage/parse"
)


type DependAtom struct {
	atom.BaseAtom	// Core atom
	Blocker bool    // Blocks specified package (! prefix)
	HardBlock bool  // Hard-blocks package (!! prefix)
	versionComparer, slotComparer verComparerFn
	useDependencies atom.UseDependencies
}



func NewDependencyAtom(str string) (*DependAtom, error) {
	return makeDA(atom.RawParseAtom(str, true, false))
}


func NewDependencyAtomUnprefixed(str string) (*DependAtom, error) {
	return makeDA(atom.RawParseAtom(str, false, false))
}


func newDependencyAtomAtCursor(cur *parse.AtomCursor, versionNeedsRelop bool) (*DependAtom, error) {
	return makeDA(atom.RawParseAtomAtCursor(cur, versionNeedsRelop, true))
}


func makeDA(pa atom.ParsedAtom, err error) (*DependAtom, error) {
	if err != nil {
		return nil, err
	}
	da := &DependAtom{
		BaseAtom: atom.NewBaseAtom(pa),
		Blocker: pa.Blocker,
		HardBlock: pa.HardBlock,
		useDependencies: atom.NewUseDependencies(pa.UseDependencies)}
	da.versionComparer = makeVersionComparer(pa.VerRelop, da.ComparisonString())
	if pa.AnySlot {
		da.slotComparer = makeVersionComparer(atom.Relop_none, "")
	} else {
		da.slotComparer = makeVersionComparer(pa.SlotRelop, da.Slot)
	}
	return da, nil
}


//Implementation of PackageDependency interface
func (da *DependAtom) DependencyType() int {
	return Pkg_dep_atom
}

func (da *DependAtom) UseFlag() string {
	return ""
}

func (da *DependAtom) Dependencies() []PackageDependency {
	return nil
}


//Comparison methods

func (da *DependAtom) VersionAndSlotMatch(tstAtom atom.Atom) bool {
	return da.slotComparer(tstAtom.GetSlot()) && da.versionComparer(tstAtom.ComparisonString())
}

func (da *DependAtom) FilterAtoms(tstAtoms []atom.Atom, contextUse atom.UseFlagMap) []atom.Atom {
	out := make([]atom.Atom, 0, len(tstAtoms))
	for _, tstAtom := range tstAtoms {
		if !da.VersionAndSlotMatch(tstAtom) {
			continue
		}
		if ok, _ := da.useDependencies.FlagsMatch(tstAtom, contextUse); !ok {
			continue
		}
		out = append(out, tstAtom)
	}
	return out
}


type verComparerFn func (string) bool

func makeVersionComparer(relop int, comparison string) (fn verComparerFn) {
	switch relop {
	case atom.Relop_lt:
		fn = func (tstval string) bool {
			return tstval < comparison
		}
	case atom.Relop_le:
		fn = func (tstval string) bool {
			return tstval <= comparison
		}
	case atom.Relop_eq:
		fn = func (tstval string) bool {
			return tstval == comparison
		}
	case atom.Relop_ge:
		fn = func (tstval string) bool {
			return tstval >= comparison
		}
	case atom.Relop_gt:
		fn = func (tstval string) bool {
			return tstval > comparison
		}
	case atom.Relop_range:
		asymptote := atom.MakeNextVer(comparison)
		fn = func (tstval string) bool {
			return tstval >= comparison && tstval < asymptote
		}
	default:
		fn = func (tstval string) bool {
			return true
		}
	}
	return
}

