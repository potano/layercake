// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package atom

import (
	"fmt"
	"strings"
	"testing"
)


func tryAdd(t *testing.T, as *AtomSet, atomString string) {
	ca, err := NewConcreteAtom(atomString)
	if err != nil {
		t.Errorf("%s adding %s", err, atomString)
	}
	as.Add(ca)
}


func tryRemove(t *testing.T, as *AtomSet, atomString string) {
	ca, err := NewConcreteAtom(atomString)
	if err != nil {
		t.Errorf("%s removing %s", err, atomString)
	}
	as.Remove(ca)
}


func checkAtomCount(t *testing.T, as *AtomSet, expected int) {
	if len(as.Atoms) != expected {
		t.Errorf("Expected %d package names in set, got %d", expected, len(as.Atoms))
	}
}


func checkPackageSlices(t *testing.T, as *AtomSet, name, slotSpec string) {
	slice := as.GetByName(name)
	slots := strings.Split(slotSpec, " ")
	if len(slice) != len(slots) {
		t.Errorf("Expected %d slot(s) for %s, got %d", len(slots), name, len(slice))
	}
	wrong := []string{}
	for i, want := range slots {
		if want != slice[i].GetSlot() {
			wrong = append(wrong, fmt.Sprintf("have %s, want %s", slice[i].GetSlot(),
				want))
		}
	}
	if len(wrong) > 0 {
		t.Errorf("unexpected slot(s) for %s: %s", name, strings.Join(wrong, "; "))
	}
}


func Test_insertion(t *testing.T) {
	as := NewAtomSet(GroupBySlot)
	tryAdd(t, as, "virtual/ada")
	tryAdd(t, as, "dev-lang/php:7.2")
	tryAdd(t, as, "dev-lang/php:7.4")

	checkAtomCount(t, as, 2)

	checkPackageSlices(t, as, "virtual/ada", "00000")
	checkPackageSlices(t, as, "dev-lang/php", "00007.00004 00007.00002")

	tryAdd(t, as, "dev-lang/php:7.3")
	checkPackageSlices(t, as, "dev-lang/php", "00007.00004 00007.00003 00007.00002")

	tryRemove(t, as, "dev-lang/php:7.2")
	checkPackageSlices(t, as, "dev-lang/php", "00007.00004 00007.00003")

	tryRemove(t, as, "dev-lang/php:7.4")
	checkPackageSlices(t, as, "dev-lang/php", "00007.00003")

	tryRemove(t, as, "dev-lang/php:7.3")
	checkAtomCount(t, as, 1)
	checkPackageSlices(t, as, "virtual/ada", "00000")
}

