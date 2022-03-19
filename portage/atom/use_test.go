// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package atom

import "testing"


func TestUseAllocation(t *testing.T) {
	origLen := useFlagIndexType(len(useFlagIndexToNames))
	if (origLen & 1) > 0 {
		t.Errorf("Expected even value for useFlagIndexToNames length; got %d", origLen)
	}
	if origLen != useFlagIndexType(len(useFlagNameToIndexMap) * 2) {
		t.Errorf("Before addition: mismatch between USE-cache lengths: %d and %d",
			origLen, len(useFlagNameToIndexMap) * 2)
	}
	newNameFinder := func () string {
		newName := "unused_flag"
		findName: for {
			for _, name := range useFlagIndexToNames {
				if name == newName {
					newName += "1"
					continue findName
				}
			}
			return newName
		}
	}

	newName := newNameFinder()
	newIndex := useFlagIndex(newName)
	if newIndex != origLen {
		t.Errorf("Expected index %d for new USE variable %s, got %d", origLen, newName,
			newIndex)
	}
	if useFlagIndexType(len(useFlagIndexToNames)) != origLen + 2 {
		t.Errorf("Expected new useFlagIndexToNames length to be %d, got %d", origLen + 2,
			len(useFlagIndexToNames))
	}
	newIndexA := useFlagIndex(newName)
	if newIndexA != newIndex {
		t.Errorf("Expected index %d for new USE variable %s not to vary; got %d", newIndex,
			newName, newIndexA)
	}
	if useFlagIndexToNames[newIndex] != newName {
		t.Errorf("Expected to find name %s at index %d, got %s", newName, newIndex,
			useFlagIndexToNames[newIndex])
	}
	if useFlagIndexToNames[newIndex + 1] != newName {
		t.Errorf("Expected to find name %s at index %d, got %s", newName, newIndex + 1,
			useFlagIndexToNames[newIndex + 1])
	}

	newName2 := newNameFinder()
	newIndex2 := useFlagIndex(newName2)
	if newIndex2 != origLen + 2 {
		t.Errorf("Expected index %d for new USE variable %s, got %d", origLen + 2, newName2,
			newIndex2)
	}

	newIndexA = useFlagIndex(newName)
	newIndex2A := useFlagIndex(newName2)
	if newIndexA != newIndex || newIndex2A != newIndex2 {
		t.Errorf("Expected name indices %d and %d, got %d and %d", newIndex, newIndex2,
			newIndexA, newIndex2A)
	}
}


type expectedUse []struct {
	flag string
	isSet bool
}

func checkExpectedUse(t *testing.T, uf UseFlagSet, setup string, expected expectedUse) {
	if len(uf) != len(expected) {
		t.Errorf("Expected [%s] to have %d USE flags, got %d", setup, len(expected),
			len(uf))
	}
	for i, et := range expected {
		index, ok := useFlagNameToIndexMap[et.flag]
		if !ok {
			t.Errorf("Can't find setting for flag '%s' in [%s]", et.flag, setup)
		}
		val := uf[i]
		if val != index && val != index + 1 {
			t.Errorf("Expected to find index %d for '%s' in [%s], got %d", index,
				et.flag, setup, val)
		}
		isSet := val & 1 > 0
		if isSet != et.isSet {
			t.Errorf("Expected %t value for '%s' in [%s], got %t", et.isSet, et.flag,
				setup, isSet)
		}
	}
}


func TestNewUseFromIUSE(t *testing.T) {
	for _, tst := range []struct {setup string; expected expectedUse} {
		{"", expectedUse{}},
		{"doc", expectedUse{{"doc", false}}},
		{"+doc", expectedUse{{"doc", false}}},
		{"-doc", expectedUse{{"doc", false}}},
		{"doc gcc", expectedUse{{"doc", false}, {"gcc", false}}},
		{"doc gcc doc", expectedUse{{"doc", false}, {"gcc", false}}},
	} {
		uf := NewUseFlagSetFromIUSE(tst.setup)
		checkExpectedUse(t, uf, tst.setup, tst.expected)
	}
}


func TestNewUseFromPrefixesPlus(t *testing.T) {
	for _, tst := range []struct {setup string; expected expectedUse} {
		{"", expectedUse{}},
		{"doc", expectedUse{{"doc", true}}},
		{"+doc", expectedUse{{"doc", true}}},
		{"-doc", expectedUse{{"doc", false}}},
		{"doc gcc", expectedUse{{"doc", true}, {"gcc", true}}},
		{"+doc -gcc", expectedUse{{"doc", true}, {"gcc", false}}},
		{"+doc -gcc -doc", expectedUse{{"doc", false}, {"gcc", false}}},
	} {
		uf := NewUseFlagSetFromPrefixes(tst.setup, true)
		checkExpectedUse(t, uf, tst.setup, tst.expected)
	}
}


func checkSameMap(t *testing.T, setup string, expected, have UseFlagMap) {
	for key, val := range expected {
		got, ok := have[key]
		if !ok {
			t.Errorf("In [%s], expected to find flag %s", setup, key)
		} else if got != val {
			t.Errorf("In [%s], expected %t for '%s', got %t", setup, val, key, got)
		}
	}
	for key := range have {
		_, ok := expected[key]
		if !ok {
			t.Errorf("In [%s], got unexpected flag %s", setup, key)
		}
	}
}


func TestGetMap(t *testing.T) {
	for _, tst := range []struct {setup string; expected UseFlagMap} {
		{"", UseFlagMap{}},
		{"doc", UseFlagMap{"doc": true}},
		{"-doc gcc", UseFlagMap{"doc": false, "gcc": true}},
		{"+doc -gcc -doc", UseFlagMap{"doc": false, "gcc": false}},
	} {
		uf := NewUseFlagSetFromPrefixes(tst.setup, true)
		checkSameMap(t, tst.setup, tst.expected, uf.GetMap())
	}
}

