// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vdb

import "potano.layercake/portage/atom"


func GetDirectories(atoms atom.AtomSlice) []string {
	out := make([]string, len(atoms))
	for i, atm := range atoms {
		out[i] = atm.(*AvailableVersion).Directory
	}
	return out
}

