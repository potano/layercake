// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package atom

import (
	"strings"
)


type ConcreteAtom struct {
	BaseAtom
	UseFlags UseFlagSet
}


func NewConcreteAtom(atom string) (*ConcreteAtom, error) {
	return makeCA(atom, true)
}


func NewUnprefixedConcreteAtom(atom string) (*ConcreteAtom, error) {
	return makeCA(atom, false)
}


func makeCA(atom string, requireVersionRelop bool) (*ConcreteAtom, error) {
	pa, err := RawParseAtom(atom, requireVersionRelop, false)
	if err != nil {
		return nil, err
	}
	return &ConcreteAtom{BaseAtom: NewBaseAtom(pa)}, nil
}


func (ca *ConcreteAtom) GetUseFlagMap() UseFlagMap {
	return ca.UseFlags.GetMap()
}


func (ca *ConcreteAtom) GetUseFlagSet() UseFlagSet {
	return ca.UseFlags
}


func NewUseFlagMap(str string) UseFlagMap {
	m := UseFlagMap{}
	if len(str) > 0 {
		for _, flag := range strings.Fields(str) {
			m[flag] = true
		}
	}
	return m
}

