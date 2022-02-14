package vdb

import "potano.layercake/portage/atom"


func GetDirectories(atoms atom.AtomSlice) []string {
	out := make([]string, len(atoms))
	for i, atm := range atoms {
		out[i] = atm.(*AvailableVersion).Directory
	}
	return out
}

