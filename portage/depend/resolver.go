// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package depend

import "potano.layercake/portage/atom"


type Resolver struct {
	Data ResolverData
}

type ResolverData interface {
	ResolveAtom(*Resolver, *DependAtom) error
	ResolveEach(*Resolver, PackageDependency) error
	ResolveSomeOf(*Resolver, PackageDependency, int, int, int) error
	ParentUseFlags() atom.UseFlagMap
}


func NewResolver(data ResolverData) *Resolver {
	return &Resolver{data}
}


func (res *Resolver) NewSubresolver(data ResolverData) *Resolver {
	return &Resolver{data}
}


func (res *Resolver) ResolveDependencies(deps []PackageDependency) error {
	return res.Data.ResolveEach(res,
		&ConditionalPackageDependency{Type: Pkg_dep_all, Deps: deps})
}

func (res *Resolver) Resolve(dep PackageDependency) (err error) {
	for _, dep := range dep.Dependencies() {
		dt := dep.DependencyType()
		switch dt {
		case Pkg_dep_atom:
			err = res.Data.ResolveAtom(res, dep.(*DependAtom))
		case Pkg_dep_all:
			err = res.Data.ResolveEach(res, dep)
		case Pkg_dep_any_of, Pkg_dep_exactly_one_of, Pkg_dep_at_most_one_of:
			minNeeded, maxNeeded := 1, len(dep.Dependencies())
			if dt == Pkg_dep_exactly_one_of {
				maxNeeded = 1
			} else if dt == Pkg_dep_at_most_one_of {
				minNeeded, maxNeeded = 0, 1
			}
			err = res.Data.ResolveSomeOf(res, dep, dt, minNeeded, maxNeeded)
		case Pkg_dep_when_use_set:
			if res.Data.ParentUseFlags()[dep.UseFlag()] {
				err = res.Resolve(dep)
			}
		case Pkg_dep_when_use_unset:
			if !res.Data.ParentUseFlags()[dep.UseFlag()] {
				err = res.Resolve(dep)
			}
		}
		if err != nil {
			break
		}
	}
	return
}

