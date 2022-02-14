package atom

import "strings"


type UseFlagSet []useFlagIndexType

type UseFlagMap map[string]bool

type useFlagIndexType uint16

// Low bit of index indicates flag state: 0 = not set, 1 = set
var useFlagNameToIndexMap map[string]useFlagIndexType = map[string]useFlagIndexType{}
var useFlagIndexToNames []string

var EmptyUseFlagMap UseFlagMap = UseFlagMap{}


func NewUseFlagSetFromIUSE(group string) UseFlagSet {
	fields := strings.Fields(group)
	slice := make(UseFlagSet, 0, len(fields))
	addValueLoop:
	for _, name := range strings.Split(group, " ") {
		if len(name) == 0 {
			continue
		}
		if name[0] == '+' || name[0] == '-' {
			name = name[1:]
		}
		index := useFlagIndex(name)
		for _, val := range slice {
			if (val & ^useFlagIndexType(1)) == index {
				continue addValueLoop
			}
		}
		slice = append(slice, index)
	}
	return slice
}


func NewUseFlagSetFromPrefixes(group string, defaultSetting bool) UseFlagSet {
	fields := strings.Fields(group)
	slice := make(UseFlagSet, 0, len(fields))
	addValueLoop:
	for _, name := range strings.Split(group, " ") {
		if len(name) == 0 {
			continue
		}
		setFlag := defaultSetting
		if name[0] == '+' {
			setFlag = true
			name = name[1:]
		} else if name[0] == '-' {
			setFlag = false
			name = name[1:]
		}
		index := useFlagIndex(name)
		setting := index
		if setFlag {
			setting++
		}
		for i, val := range slice {
			if (val & ^useFlagIndexType(1)) == index {
				slice[i] = setting
				continue addValueLoop
			}
		}
		slice = append(slice, setting)
	}
	return slice
}


func (f UseFlagSet) SetFlagsFromUSE(group string) {
	for _, name := range strings.Fields(group) {
		if name[0] == '+' || name[0] == '-' {
			name = name[1:]
		}
		index, ok := useFlagNameToIndexMap[name]
		if ok {
			for i, val := range f {
				if val & ^useFlagIndexType(1) == index {
					f[i] = index | 1
					break
				}
			}
		}
	}
}


func (f UseFlagSet) SetFlagsFromPrefixes(group string, defaultSetting bool) {
	for _, name := range strings.Fields(group) {
		var mask useFlagIndexType
		if name[0] == '+' {
			mask = 1
			name = name[1:]
		} else if name[0] == '-' {
			mask = 0
			name = name[1:]
		} else if defaultSetting {
			mask = 1
		}
		index, ok := useFlagNameToIndexMap[name]
		if ok {
			for i, val := range f {
				if val & ^useFlagIndexType(1) == index {
					f[i] = index | mask
					break
				}
			}
		}
	}
}


func (f UseFlagSet) GetMap() UseFlagMap {
	ufm := make(UseFlagMap, len(f))
	for _, ind := range f {
		name := useFlagIndexToNames[ind]
		state := false
		if ind & 1 > 0 {
			state = true
		}
		ufm[name] = state
	}
	return ufm
}


func useFlagIndex(name string) useFlagIndexType {
	index, have := useFlagNameToIndexMap[name]
	if !have {
		index = useFlagIndexType(len(useFlagIndexToNames))
		useFlagNameToIndexMap[name] = index
		useFlagIndexToNames = append(useFlagIndexToNames, name, name)
	}
	return index
}


func (f UseFlagSet) flagStateByIndex(index useFlagIndexType) (bool, bool) {
	index = index & ^useFlagIndexType(1)
	for _, val := range f {
		if val == index || val == index + 1 {
			if val & 1 > 0 {
				return true, true
			}
			return false, true
		}
	}
	return false, false
}

