package atom


type BaseAtom struct {
	Atom string	// Original package atom
	Category string	// Package category (e.g. dev-util)
	Name string	// Package base name (e.g. layercake)
	compVer string	// Version-comparison string: space-separated normalized base version,
			// release suffix, e-build revision, and repo
	Slot string	// Comparable form of Gentoo slot (e.g. 00002 for 2)
	Subslot string	// Comparable form of Gentoo subslot (e.g. 00002 for 2)
	Repo string	// Repo containing package (e.g. gentoo)
}


func NewBaseAtom(pa ParsedAtom) BaseAtom {
	return BaseAtom{
		Atom: pa.Atom,
		Category: pa.Category,
		Name: pa.Name,
		compVer: pa.CompVer,
		Slot: pa.Slot,
		Subslot: pa.Subslot,
		Repo: pa.Repo}
}


func (ba *BaseAtom) String() string {
	return ba.Atom
}


func (ba *BaseAtom) PackageName() string {
	return ba.Category + "/" + ba.Name
}


func (ba *BaseAtom) ComparisonString() string {
	return ba.compVer
}


func (ba *BaseAtom) GetSlot() string {
	return ba.Slot
}


func (ba *BaseAtom) SetSlotAndSubslot(slot, subslot string) {
	if len(slot) == 0 {
		slot = "0"
	}
	ba.Slot = makeComparable(slot)
	if len(subslot) > 0 {
		ba.Subslot = makeComparable(subslot)
	} else {
		ba.Subslot = ba.Slot
	}
}


func (ba *BaseAtom) GetUseFlagMap() UseFlagMap {
	return EmptyUseFlagMap
}


func (ba *BaseAtom) GetUseFlagSet() UseFlagSet {
	return UseFlagSet{}
}


func (ba *BaseAtom) GetGroupingKey(ktype int) string {
	switch ktype {
	case GroupByVersion:
		return ba.ComparisonString()
	case GroupBySlot:
		return ba.GetSlot()
	}
	return ""
}

