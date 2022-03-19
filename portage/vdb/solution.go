// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vdb

import (
	"fmt"
	"strings"
	"potano.layercake/portage/atom"
	"potano.layercake/portage/depend"
)

type Solution struct {
	Installed *atom.AtomSet
	Resolution *atom.AtomSet
	IncludeBdepend bool
}


func StartSolution(installedSet *atom.AtomSet, includeBdepend bool) (*Solution, error) {
	return &Solution{
		Installed: installedSet,
		Resolution: atom.NewAtomSet(atom.GroupBySlot),
		IncludeBdepend: includeBdepend,
	}, nil
}


func (sol *Solution) ResolveUserDeps(deps *depend.UserEnteredDependencies) error {
	var wanted, blockers []depend.PackageDependency
	for _, ue := range deps.Atoms {
		if len(ue.Name) == 0 {
			return fmt.Errorf("entered atom %s has no package name", ue)
		}
		if len(ue.Category) == 0 {
			cats := sol.findPackageCategory(ue.Name)
			if len(cats) == 1 {
				ue.Category = cats[0]
			} else if len(cats) > 1 {
				return fmt.Errorf("ambiguous atom name: %s exists in categories %s",
					ue.Name, strings.Join(cats, ", "))
			} else if !ue.Blocker {
				return fmt.Errorf("no package named %s", ue.Name)
			} else {
				continue
			}
		}
		if ue.Blocker {
			blockers = append(blockers, ue)
		} else {
			wanted = append(wanted, ue)
		}
	}
	dependencies := make([]depend.PackageDependency, len(wanted) + len(blockers))
	copy(dependencies[:len(blockers)], blockers)
	copy(dependencies[len(blockers):], wanted)

	rootAtom, _ := atom.NewConcreteAtom("requested")
	resolver := sol.newResolver(&AvailableVersion{ConcreteAtom: *rootAtom})
	err := resolver.ResolveDependencies(dependencies)
	if err != nil {
		return err
	}
	return nil
}


func (sol *Solution) findPackageCategory(name string) []string {
	candidates := []string{}
	for fqname := range sol.Installed.Atoms {
		parts := strings.Split(fqname, "/")
		if parts[1] == name {
			candidates = append(candidates, parts[0])
		}
	}
	return candidates
}


func (sol *Solution) findDependencies(ia *AvailableVersion) error {
	var names []string
	var dependencies []depend.PackageDependency
	if sol.IncludeBdepend {
		names = []string{"BDEPEND", "DEPEND", "RDEPEND", "PDEPEND"}
	} else {
		names = []string{"RDEPEND", "PDEPEND"}
	}
	for _, name := range names {
		line, found, err := ia.readFileIfExists(name)
		if err != nil {
			return err
		}
		if found {
			deps, err := depend.DecodeDependencies([]byte(line))
			if err != nil {
				return err
			}
			dependencies = append(dependencies, deps...)
		}
	}
	if len(dependencies) == 0 {
		return nil
	}
	resolver := sol.newResolver(ia)
	return resolver.ResolveDependencies(dependencies)
}


type installedResolverData struct {
	rundata *Solution
	contextAtom *AvailableVersion
	resolved []*AvailableVersion
	conditionalAtomMode bool
}


func (solution *Solution) newResolver(contextAtom *AvailableVersion) *depend.Resolver {
	return depend.NewResolver(&installedResolverData{
		rundata: solution,
		contextAtom: contextAtom})
}


func newSubresolver(res *depend.Resolver, conditionalAtomMode bool) *depend.Resolver {
	data := res.Data.(*installedResolverData)
	return res.NewSubresolver(&installedResolverData{
		rundata: data.rundata,
		contextAtom: data.contextAtom,
		conditionalAtomMode: conditionalAtomMode})
}


func (data *installedResolverData) ResolveAtom(res *depend.Resolver, dep *depend.DependAtom) error {
	candidates := data.rundata.Installed.GetByName(dep.PackageName())
	if len(candidates) > 0 {
		candidates = dep.FilterAtoms(candidates, data.ParentUseFlags())
	}
	if dep.Blocker {
		if len(candidates) == 0 {
			return nil
		}
		for _, cand := range candidates {
			if cand.(*AvailableVersion).Added {
				return fmt.Errorf("%s blocks package: %s", data.contextAtom, cand)
			}
			cand.(*AvailableVersion).Blocked = true
		}
		return nil
	}
	if len(candidates) == 0 {
		if !data.conditionalAtomMode {
			return fmt.Errorf("could not resolve %s dependency of %s", dep,
				data.contextAtom)
		}
	} else {
		for _, cand := range candidates {
			data.resolved = append(data.resolved, cand.(*AvailableVersion))
		}
	}
	return nil
}


func (data *installedResolverData) ResolveEach(res *depend.Resolver,
	dep depend.PackageDependency) error {
	err := res.Resolve(dep)
	if err != nil {
		return err
	}
	for _, ia := range data.resolved {
		if ia.Blocked {
			return fmt.Errorf("blocked package: %s", ia)
		}
		if ia.Added || data.conditionalAtomMode {
			continue
		}
		ia.Added = true
		data.rundata.Resolution.Add(ia)
		err = data.rundata.findDependencies(ia)
		if err != nil {
			return err
		}
	}
	return nil
}


func (data *installedResolverData) ResolveSomeOf(res *depend.Resolver,
	dep depend.PackageDependency, depType, minNeeded, maxNeeded int) error {

	subres := newSubresolver(res, true)
	err := subres.Resolve(dep)
	if err != nil {
		return err
	}
	newdata := subres.Data.(*installedResolverData)
	resolved := newdata.resolved
	if len(resolved) < minNeeded {
		return fmt.Errorf("cannot resolve dependency %s of %s", dep, data.contextAtom)
	}
	if len(resolved) > maxNeeded {
		resolved = resolved[:maxNeeded]
	}
	data.resolved = append(data.resolved, resolved...)
	return nil
}


func (data *installedResolverData) ParentUseFlags() atom.UseFlagMap {
	return data.contextAtom.GetUseFlagMap()
}

