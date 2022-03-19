// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package depend


import (
	"fmt"
	"potano.layercake/portage/parse"
)


type PackageDependency interface {
	DependencyType() int
	UseFlag() string
	Dependencies() []PackageDependency
	String() string
}


func DecodeDependencies(buf []byte) ([]PackageDependency, error) {
	var out []PackageDependency
	cursor := parse.NewAtomCursor(buf)
	for {
		dep, err := decodeDependency(cursor)
		if err != nil {
			return nil, err
		}
		if dep == nil {
			break
		}
		out = append(out, dep)
	}
	return out, nil
}


const (
	Pkg_dep_atom = iota
	Pkg_dep_all
	Pkg_dep_any_of
	Pkg_dep_exactly_one_of
	Pkg_dep_at_most_one_of
	Pkg_dep_when_use_set
	Pkg_dep_when_use_unset
)


type ConditionalPackageDependency struct {
	Type int
	useFlag string
	Deps []PackageDependency
}


func (d *ConditionalPackageDependency) DependencyType() int {
	return d.Type
}

func (d *ConditionalPackageDependency) UseFlag() string {
	return d.useFlag
}

func (d *ConditionalPackageDependency) Dependencies() []PackageDependency {
	return d.Deps
}

func (d *ConditionalPackageDependency) String() string {
	var str string
	if d.Type == Pkg_dep_when_use_set {
		str = fmt.Sprintf("%s (", d.useFlag)
	} else if d.Type == Pkg_dep_when_use_unset {
		str = fmt.Sprintf("!%s (", d.useFlag)
	} else {
		str = []string{"", "(", "|| (", "^^ (", "?? ("}[d.Type]
	}
	for _, dep := range d.Deps {
		str += " " + dep.String()
	}
	return str + " )"
}



var toktype_to_pkg_dep map[int]int = map[int]int{
	toktype_when_use_set: Pkg_dep_when_use_set,
	toktype_when_use_unset: Pkg_dep_when_use_unset,
	toktype_any_of:Pkg_dep_any_of,
	toktype_exactly_one_of: Pkg_dep_exactly_one_of,
	toktype_at_most_one_of: Pkg_dep_at_most_one_of,
}

func decodeDependency(ac *parse.AtomCursor) (PackageDependency, error ) {
	var dep PackageDependency
	var newType int
	var err error
	start, toktype, useFlag := getToken(ac)
	switch toktype {
	case toktype_eof, toktype_close:
		return nil, nil
	case toktype_error:
		return nil, fmt.Errorf("unrecognized dependency token %s", ac.RemainingToken())
	case toktype_open:
		var list []PackageDependency
		for {
			dp, err := decodeDependency(ac)
			if err != nil {
				return nil, err
			}
			if dp == nil {
				break
			}
			list = append(list, dp)
		}
		dep = &ConditionalPackageDependency{Type: Pkg_dep_all, Deps: list}
		return dep, nil
	case toktype_when_use_set, toktype_when_use_unset:
		dep, err = decodeDependency(ac)
		if err != nil {
			return nil, err
		}
		newType = toktype_to_pkg_dep[toktype]
		switch dep.DependencyType() {
		case Pkg_dep_atom:
			atom := dep
			dep = &ConditionalPackageDependency{Type: newType, useFlag: useFlag,
				Deps: []PackageDependency{atom}}
		case Pkg_dep_all:
			dep.(*ConditionalPackageDependency).Type = newType
			dep.(*ConditionalPackageDependency).useFlag = useFlag
		default:
			return nil, fmt.Errorf("invalid pattern after USE flag %s: %s", useFlag,
				ac.RemainingTokenAtPos(start))
		}
		return dep, nil
	case toktype_any_of, toktype_exactly_one_of, toktype_at_most_one_of:
		dep, err = decodeDependency(ac)
		if err != nil {
			return nil, err
		}
		newType = toktype_to_pkg_dep[toktype]
		if dep.(*ConditionalPackageDependency).Type != Pkg_dep_all {
			return nil, fmt.Errorf("invalid pattern after %s: %s",
			ac.RemainingTokenAtPos(start), ac.SampleAfterPos(start+3))
		}
		dep.(*ConditionalPackageDependency).Type = newType
		return dep, nil
	case toktype_test_for_atom:
		dep, err = newDependencyAtomAtCursor(ac, true)
		if err != nil {
			return nil, err
		}
		return dep, nil
	}
	return nil, fmt.Errorf("internal error: unexpected token type %d", toktype)
}

