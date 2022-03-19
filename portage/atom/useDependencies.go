// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package atom

import "fmt"


type UseDependencies []useDependencyIndexType

type useDependencyIndexType uint16

var useDependencyMap map[UseDependency]useDependencyIndexType =
	map[UseDependency]useDependencyIndexType{}
var useDependencyList []UseDependency


func useDependencyIndex(dep UseDependency) useDependencyIndexType {
	index, have := useDependencyMap[dep]
	if !have {
		index = useDependencyIndexType(len(useDependencyList))
		useDependencyMap[dep] = index
		useDependencyList = append(useDependencyList, dep)
	}
	return index
}


func NewUseDependencies(deps []UseDependency) UseDependencies {
	slice := make(UseDependencies, len(deps))
	for i, dep := range deps {
		slice[i] = useDependencyIndex(dep)
	}
	return slice
}


func (ud UseDependencies) FlagsMatch(tstAtom Atom, contextFlags UseFlagMap) (bool, error) {
	tstUseFlags := tstAtom.GetUseFlagSet()
	for _, udIndex := range ud {
		dep := useDependencyList[udIndex]
		state, ok := tstUseFlags.flagStateByIndex(dep.UseFlag)
		if !ok {
			if dep.FlagDefault == Use_default_enabled {
				state = true
			} else if dep.FlagDefault == Use_default_disabled {
				state = false
			} else {
				return false, fmt.Errorf("%s has no %s USE flag defined", tstAtom,
					useFlagIndexToNames[dep.UseFlag])
			}
		}
		switch dep.Type {
		case Use_dep_enabled:
			if !state {
				return false, nil
			}
		case Use_dep_same:
			if state != contextFlags[useFlagIndexToNames[dep.UseFlag]] {
				return false, nil
			}
		case Use_dep_opposite:
			if state == contextFlags[useFlagIndexToNames[dep.UseFlag]] {
				return false, nil
			}
		case Use_dep_set_only_if:
			if contextFlags[useFlagIndexToNames[dep.UseFlag]] && !state {
				return false, nil
			}
		case Use_dep_unset_only_if:
			if contextFlags[useFlagIndexToNames[dep.UseFlag]] && state {
				return false, nil
			}
		default:	// Use_dep_disabled
			if state {
				return false, nil
			}
		}
	}
	return true, nil
}

