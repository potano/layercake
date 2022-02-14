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


