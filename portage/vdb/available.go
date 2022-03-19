// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vdb

import (
	"potano.layercake/portage/atom"
	"potano.layercake/portage/depend"
)


type AvailableVersion struct {
	atom.ConcreteAtom
	Directory string
	Deps []depend.PackageDependency
	Blocked bool
	Added bool
}


