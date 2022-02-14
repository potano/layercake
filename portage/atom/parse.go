package atom

import (
	"fmt"
	"regexp"
	"strings"
	"potano.layercake/portage/parse"
)


type ParsedAtom struct {
	Atom string	// Original package atom
	Category string	// Package category (e.g. dev-util)
	Name string	// Package base name (e.g. layercake)
	BaseVer string	// Comparable package base version (e.g. 00002.00000.00000 for 2.0.0)
	Suffix string	// Package-version suffix (e.g. _rc00001 for _rc1)
	Revision string	// Ebuild revision (e.g. r00001 for r1)
	CompVer string	// Concatenation of BaseVer, Suffix, and Revision as appropriate
	Slot string	// Comparable form of Gentoo slot (e.g. 00002 for 2)
	Subslot string	// Comparable form of Gentoo subslot (e.g. 00002 for 2)
	Repo string	// Repo containing package (e.g. gentoo)
	VerRelop int	// Comparison relation for version
	SlotRelop int	// Comparison relation for slot
	AnySlot bool	// Allows installation of any slot (* as slot specifier)
	SameSlot bool	// Depends on installed slot (= as slot specifier or suffix)
	Blocker bool	// Blocks specified package (! prefix)
	HardBlock bool	// Hard-blocks package (!! prefix)
	UseDependencies []UseDependency
}


const (
	Relop_none = iota
	Relop_lt
	Relop_le
	Relop_eq
	Relop_ge
	Relop_gt
	Relop_range
)


type UseDependency struct {
	Type int
	FlagDefault int
	UseFlag useFlagIndexType
}


const (
	Use_dep_enabled = iota	// [opt]	USE flag must be enabled
	Use_dep_same		// [opt=]	USE flag setting must be same as referrer's
	Use_dep_opposite	// [!opt=]	USE flag setting must be opposite to referer's
	Use_dep_set_only_if	// [opt?]	USE flag must be set if referrer has it set
	Use_dep_unset_only_if	// [!opt?]	USE flag must be unset if referrer has it unset
	Use_dep_disabled	// [-opt]	USE flag must be disabled
)

const (
	Use_default_none = iota	// no default setting given
	Use_default_enabled	// act as if flag is enabled: (+)
	Use_default_disabled	// act as if flag is disabled: (-)
)


func RawParseAtom(atom string, versionNeedsRelop, asDependencyAtom bool) (ParsedAtom, error) {
	cursor := parse.NewAtomCursor([]byte(atom))
	return RawParseAtomAtCursor(cursor, versionNeedsRelop, asDependencyAtom)
}


func NewDependencyAtomUnprefixed(atom string) (ParsedAtom, error) {
	cursor := parse.NewAtomCursor([]byte(atom))
	return RawParseAtomAtCursor(cursor, false, false)
}





const (
	maxAlphaVersion = "zzzzz"
	numericVersionSegmentWidth = 5
	releaseSuffixAlpha = "_a"
	releaseSuffixBeta = "_b"
	releaseSuffixPre = "_c"
	releaseSuffixRc = "_d"
	releaseSuffixNormal = "_n"
	releaseSuffixPatch = "_p"
	defaultRevision = "r00000"
)






func RawParseAtomAtCursor(ac *parse.AtomCursor,
	versionNeedsRelop, asDependencyAtom bool) (ParsedAtom, error) {
	relop := Relop_none
	extendCompVer := false
	da := ParsedAtom{}

	start := ac.Pos
	c := ac.Peek()

	if c == '!' {
		da.Blocker = true
		c = ac.Take()
		if c == '!' {
			da.HardBlock = true
			c = ac.Take()
		}
	}

	if c == '~' {
		relop = Relop_range
		c = ac.Take()
	} else if c == '=' || c == '<' || c == '>' {
		c1 := c
		c := ac.Take()
		if c1 == '=' {
			relop = Relop_eq
		} else if c == '=' {
			if c1 == '<' {
				relop = Relop_le
			} else {
				relop = Relop_ge
			}
			c = ac.Take()
		} else if c1 == '<' {
			relop = Relop_lt
		} else {
			relop = Relop_gt
		}
	}

	nameStart := ac.Pos
	endpos := ac.TakeNameVerChars()
	slot, subslot, slotop := ac.TakeSlot()
	da.Repo = ac.TakeRepo()

	if asDependencyAtom {
		useDependencyString := ac.TakeUseDependencyString()
		if len(useDependencyString) > 0 {
			deps, err := parseUseDependencies(useDependencyString)
			if err != nil {
				return da, fmt.Errorf("atom %s: %s", ac.Slice[start:ac.Pos], err)
			}
			da.UseDependencies = deps
		}
	} else if ac.Pos <= ac.Last {
		return da, fmt.Errorf("atom %s followed by extraneous characters: %s",
			ac.Slice[start:ac.Pos], ac.RemainingToken())
	}

	atomEndpos := ac.Pos

	matches := pkgVerRE.FindSubmatch(ac.Slice[nameStart:endpos])
	basever := getMatchvalByOffset(matches, 2)
	suffix := getMatchvalByOffset(matches, 3)
	revision := getMatchvalByOffset(matches, 4)
	haveWildcard := len(getMatchvalByOffset(matches, 5)) > 0

	if len(matches) > 1 {
		if relop == Relop_none && versionNeedsRelop {
			return da, fmt.Errorf("atom %s has version but lacks prefix operator",
				ac.Slice[start:atomEndpos])
		}
		endpos = nameStart + len(matches[1])
		if haveWildcard {
			relop = Relop_range
		}
	} else if relop != Relop_none {
		return da, fmt.Errorf("cannot parse version of %s", ac.Slice[start:atomEndpos])
	}

	matches = pkgCatNameRE.FindSubmatch(ac.Slice[nameStart:endpos])
	if len(matches) == 0 {
		return da, fmt.Errorf("could not decode atom %s", ac.Slice[start:atomEndpos])
	}
	da.Category = getMatchvalByOffset(matches, 1)
	da.Name = getMatchvalByOffset(matches, 2)

	ac.Pos = atomEndpos
	atom := string(ac.Slice[start:atomEndpos])
	da.Atom = atom

	if len(basever) > 0 {
		basever = makeComparable(basever)
		c := basever[len(basever)-1]
		if !isDigit(c) {
			basever = basever[:len(basever)-1] + " " + string(c)
		}
		da.BaseVer = basever
		da.CompVer = basever
		extendCompVer = true
		if len(suffix) > 0 {
			suffix = strings.ReplaceAll(suffix, "_alpha", releaseSuffixAlpha)
			suffix = strings.ReplaceAll(suffix, "_beta", releaseSuffixBeta)
			suffix = strings.ReplaceAll(suffix, "_pre", releaseSuffixPre)
			suffix = strings.ReplaceAll(suffix, "_rc", releaseSuffixRc)
			da.Suffix = makeComparable(suffix)
		} else {
			da.Suffix = releaseSuffixNormal
			extendCompVer = extendCompVer && relop != Relop_range
		}
		if extendCompVer {
			da.CompVer += " " + da.Suffix
		}
		if len(revision) > 0 {
			da.Revision = makeComparable(revision)
		} else {
			da.Revision = defaultRevision
			extendCompVer = extendCompVer && relop != Relop_range
		}
		if extendCompVer {
			da.CompVer += " " + da.Revision
		}
		if relop == Relop_none {
			relop = Relop_eq
		}
		da.VerRelop = relop
	}

	if len(slot) > 0 {
		da.SlotRelop = Relop_eq
	}

	if len(slotop) > 0 {
		if slotop == "*" {
			if len(slot) > 0 {
				da.SlotRelop = Relop_range
			} else {
				da.AnySlot = true
			}
		} else if slotop == "=" {
			if len(slot) == 0 {
				da.AnySlot = true
			}
			da.SameSlot = true
		}
	}

	if !da.AnySlot {
		if len(slot) == 0 {
			slot = "0"
		}
		da.Slot = makeComparable(slot)
		if len(subslot) > 0 {
			da.Subslot= makeComparable(subslot)
		} else {
			da.Subslot = da.Slot
		}
	}

	return da, nil
}


var prefixSuffixMap map[byte]map[byte]int = map[byte]map[byte]int{
	0:   {	0: Use_dep_enabled,		// [opt]
		'=': Use_dep_same,		// [opt=]
		'?': Use_dep_set_only_if,	// [opt?]
	},
	'!': {	'=': Use_dep_opposite,		// [!opt=]
		'?': Use_dep_unset_only_if,	// [!opt?]
	},
	'-': {	0: Use_dep_disabled,		// [-opt]
	},
}


func parseUseDependencies(input []byte) ([]UseDependency, error) {
	var deps []UseDependency
	cur := parse.NewAtomCursor(input)
	for {
		useDefault := Use_default_none
		var prefix, suffix byte
		var start int
		c := cur.Peek()
		if c == '!' || c == '-' {
			prefix = c
			c = cur.Take()
		}
		if !parse.IsUseFlagChar[c] {
			return nil, fmt.Errorf("missing dependency USE flag")
		}
		start = cur.Pos
		for {
			c = cur.Take()
			if !parse.IsUseFlagChar[c] {
				break
			}
		}
		flag := string(cur.Slice[start:cur.Pos])
		if c == '=' || c == '?' {
			suffix = c
			c = cur.Take()
		}
		if c == '(' && cur.Peek2() == ')' {
			c = cur.Take()
			switch c {
			case '+':
				useDefault = Use_default_enabled
			case '-':
				useDefault = Use_default_disabled
			default:
				return nil, fmt.Errorf("unknown USE-default character %c", c)
			}
			cur.Pos++
			c = cur.Take()
		}
		tp, ok := prefixSuffixMap[prefix][suffix]
		if !ok {
			return nil, fmt.Errorf("unknown USE-dependency combination %c/%c", prefix,
				suffix)
		}
		deps = append(deps, UseDependency{
			Type: tp,
			FlagDefault: useDefault,
			UseFlag: useFlagIndex(flag),
		})
		if c == 0 {
			break
		}
		if c != ',' {
			return nil, fmt.Errorf("cannot parse USE dependencies")
		}
		cur.Pos++
	}
	return deps, nil
}




var pkgCatNameRE, pkgVerRE *regexp.Regexp

func init () {
//	pkgVerRE = regexp.MustCompile(`^(.*?)-(\d+(?:\.\d+)*[a-z]?\*?)(_\w+)?(?:-(r\d+))?$`)
	pkgVerRE = regexp.MustCompile(`^(.*?)-(\d+(?:\.\d+)*[a-z]?)(_\w+)?(?:-(r\d+))?(\*?)$`)
	pkgCatNameRE = regexp.MustCompile(`^(?:(\w[\w+.-]*)/)?(\w[\w+-]*)$`)
}

func getMatchvalByOffset(slice [][]byte, index int) string {
	if index > 0 && index < len(slice) {
		return string(slice[index])
	}
	return ""
}

