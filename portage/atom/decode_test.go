package atom

import (
	"fmt"
	"strings"
	"testing"
)

var relopSym []string = []string{
	"Relop_none",
	"Relop_lt",
	"Relop_le",
	"Relop_eq",
	"Relop_ge",
	"Relop_gt",
	"Relop_range",
}

var usedepType []string = []string{
	"Use_dep_enabled",
	"Use_dep_same",
	"Use_dep_opposite",
	"Use_dep_set_only_if",
	"Use_dep_unset_only_if",
	"Use_dep_disabled",
}

var usedefaultType []string = []string{
	"Use_default_none",
	"Use_default_enabled",
	"Use_default_disabled",
}

type atomProto struct {
	Atom string	// Original package atom
	Category string	// Package category (e.g. dev-util)
	Name string	// Package base name (e.g. layercake)
	BaseVer string	// Comparable package base version (e.g. 00002.00000.00000 for 2.0.0)
	Suffix string	// Package-version suffix (e.g. _rc00001 for _rc1)
	Revision string	// Ebuild revision (e.g. r00001 for r1)
	CompVer string	// Version for comparison--adjusted according to VerRelop
	Slot string	// Comparable form of Gentoo slot (e.g. 00002 for 2)
	Subslot string	// Comparable form of Gentoo subslot (e.g. 00002 for 2)
	Repo string	// Repo containing package (e.g. gentoo)
	VerRelop int	// Comparison relation for version
	SlotRelop int	// Comparison relation for slot
	AnySlot bool	// Allows installation of any slot (* as slot specifier)
	SameSlot bool	// Depends on installed slot (= as slot specifier or suffix)
	Blocker bool	// Blocks specified package (! prefix)
	HardBlock bool	// Hard-blocks package (!! prefix)
	UseDependencies []useDependencyProto
}

type useDependencyProto struct {
	Type int
	FlagDefault int
	UseFlag string
}


func tryAtom(t *testing.T, atomString, errmsg string, expected *atomProto, dec ParsedAtom,
	err error) {
	if err != nil {
		if len(errmsg) == 0 {
			t.Errorf("%s: unexpected error %s", atomString, err)
		} else if errmsg != err.Error() {
			t.Errorf("%s: expected error '%s', got '%s'", atomString, errmsg, err)
		}
		return
	} else if len(errmsg) > 0 {
		t.Errorf("%s: expected error %s", atomString, errmsg)
		return
	}
	if dec.Atom != atomString {
		t.Errorf("%s: unexpected value for Atom: %s", atomString, dec.Atom)
	}
	if dec.Category != expected.Category {
		t.Errorf("%s: expected Category=%s, got %s", atomString, expected.Category,
			dec.Category)
	}
	if dec.Name != expected.Name {
		t.Errorf("%s: expected Name=%s, got %s", atomString, expected.Name, dec.Name)
	}
	if dec.BaseVer != expected.BaseVer {
		t.Errorf("%s: expected BaseVer=%s, got %s", atomString, expected.BaseVer,
			dec.BaseVer)
	}
	if dec.Suffix != expected.Suffix {
		t.Errorf("%s: expected Suffix=%s, got %s", atomString, expected.Suffix, dec.Suffix)
	}
	if dec.Revision != expected.Revision {
		t.Errorf("%s: expected Revision=%s, got %s", atomString, expected.Revision,
			dec.Revision)
	}
	if dec.CompVer != expected.CompVer {
		t.Errorf("%s: expected CompVer=%s, got %s", atomString, expected.CompVer,
			dec.CompVer)
	}
	if dec.Slot != expected.Slot {
		t.Errorf("%s: expected Slot=%s, got %s", atomString, expected.Slot, dec.Slot)
	}
	if dec.Subslot != expected.Subslot {
		t.Errorf("%s: expected Subslot=%s, got %s", atomString, expected.Subslot,
			dec.Subslot)
	}
	if dec.Repo != expected.Repo {
		t.Errorf("%s: expected Repo=%s, got %s", atomString, expected.Repo, dec.Repo)
	}
	if dec.VerRelop != expected.VerRelop {
		t.Errorf("%s: expected VerRelop=%s, got %s", atomString,
			relopSym[expected.VerRelop], relopSym[dec.VerRelop])
	}
	if dec.SlotRelop != expected.SlotRelop {
		t.Errorf("%s: expected SlotRelop=%s, got %s", atomString,
			relopSym[expected.SlotRelop], relopSym[dec.SlotRelop])
	}
	if dec.AnySlot != expected.AnySlot {
		t.Errorf("%s: expected AnySlot=%v, got %v", atomString, expected.AnySlot,
			dec.AnySlot)
	}
	if dec.SameSlot != expected.SameSlot {
		t.Errorf("%s: expected SameSlot=%v, got %v", atomString, expected.SameSlot,
			dec.SameSlot)
	}
	if dec.Blocker != expected.Blocker {
		t.Errorf("%s: expected Blocker=%v, got %v", atomString, expected.Blocker,
			dec.Blocker)
	}
	if dec.HardBlock != expected.HardBlock {
		t.Errorf("%s: expected HardBlock=%v, got %v", atomString, expected.HardBlock,
			dec.HardBlock)
	}
	if len(dec.UseDependencies) != len(expected.UseDependencies) {
		t.Errorf("%s: expected %d USE dependencies, got %d", atomString,
			len(expected.UseDependencies), len(dec.UseDependencies))
	} else {
		for i, dep := range dec.UseDependencies {
			edep := expected.UseDependencies[i]
			if dep.Type != edep.Type {
				t.Errorf("%s USE dependency %d: expected type %s, got %s",
					atomString, i, usedepType[edep.Type], usedepType[dep.Type])
			}
			if dep.FlagDefault != edep.FlagDefault {
				t.Errorf("%s USE dependency %d: expected default %s, got %s",
					atomString, i, usedefaultType[edep.FlagDefault],
					usedefaultType[dep.FlagDefault])
			}
			useFlagString := useFlagIndexToNames[dep.UseFlag]
			if useFlagString != edep.UseFlag {
				t.Errorf("%s USE dependency %d: expected flag %s, got %s",
					atomString, i, edep.UseFlag, useFlagString)
			}
		}
	}
}

var relopName []string = []string{
	"none",
	"less than",
	"less than or equal to",
	"equal to",
	"greater than or equal to",
	"greater than",
	"range",
}

var useDepTypeToProto []string = []string{
	"%s",	// Use_dep_enabled
	"%s=",	// Use_dep_same
	"!%s=",	// Use_dep_opposite
	"%s?",	// Use_dep_set_only_if
	"!%s?",	// Use_dep_unset_only_if
	"-%s",	// Use_dep_disabled,
}

func printAtom(atom *ParsedAtom) {
	fmt.Printf("Atom: %s\n", atom.Atom)
	fmt.Printf("   Category: %s, Package name: %s, Version: %s / %s / %s\n", atom.Category,
		atom.Name, atom.BaseVer, atom.Suffix, atom.Revision)
	fmt.Printf("   Slot: %s, Subslot: %s, repo: %s\n", atom.Slot, atom.Subslot, atom.Repo)
	fmt.Printf("   Version comparson: %s; slot comparison: %s\n", relopName[atom.VerRelop],
		relopName[atom.SlotRelop])
	fmt.Printf("   Any slot: %v, replace w/ same slot: %v, Blocker: %v, Hard Blocker: %v\n",
		atom.AnySlot, atom.SameSlot, atom.Blocker, atom.HardBlock)
	if len(atom.UseDependencies) > 0 {
		uses := make([]string, len(atom.UseDependencies))
		for i, dep := range atom.UseDependencies {
			var use string
			useFlag := useFlagIndexToNames[dep.UseFlag]
			if dep.Type >= 0 && dep.Type < len(useDepTypeToProto) {
				use = fmt.Sprintf(useDepTypeToProto[dep.Type], useFlag)
			} else {
				use = fmt.Sprintf("'%s'_type=%d", useFlag, dep.Type)
			}
			if dep.FlagDefault == Use_default_enabled {
				use += "(+)"
			} else if dep.FlagDefault == Use_default_disabled {
				use += "(-)"
			}
			uses[i] = use
		}
		fmt.Printf("   Use Dependencies: %s\n", strings.Join(uses, " "))
	}
}


const (
	a_error_if_any = 1 << iota
	a_error_if_no_prefix
	a_error_if_input_atom
)

type a_error_if struct {
	when int
	msg string
}

type atomTest struct {
	atomString string
	error_if []a_error_if
	expected *atomProto
}

func mkAtomTest(atomString, errmsg string, error_if_cases int, expected *atomProto) atomTest {
	var error_if []a_error_if
	if (error_if_cases & a_error_if_input_atom) > 0 {
		at, tail := atomString, atomString
		mk := strings.IndexRune(at, '[')
		if mk >= 0 {
			at = atomString[:mk]
			tail = atomString[mk:]
		}
		error_if = append(error_if, a_error_if{a_error_if_input_atom,
			"atom " + at + " followed by extraneous characters: " + tail})
	}
	if (error_if_cases & a_error_if_no_prefix) > 0 {
		error_if = append(error_if, a_error_if{a_error_if_no_prefix,
			"atom " + atomString + " has version but lacks prefix operator"})
	}
	if len(errmsg) > 0 {
		error_if = append(error_if, a_error_if{a_error_if_any, errmsg})
	}
	return atomTest{atomString, error_if, expected}
}

func (at atomTest) pickErrmsg(when int) string {
	when |= a_error_if_any
	for _, item := range at.error_if {
		if (item.when & when) > 0 {
			return item.msg
		}
	}
	return ""
}

var atomTestSet []atomTest = []atomTest{
	mkAtomTest("dev-lang/go", "", 0, &atomProto{
		Atom: "dev-lang/go",
		Category: "dev-lang",
		Name: "go",
		VerRelop: Relop_none,
		BaseVer: "",
		Suffix: "",
		Revision: "",
		CompVer: "",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("dev-lang/go[gccgo]", "", a_error_if_input_atom, &atomProto{
		Atom: "dev-lang/go",
		Category: "dev-lang",
		Name: "go",
		VerRelop: Relop_none,
		BaseVer: "",
		Suffix: "",
		Revision: "",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		UseDependencies: []useDependencyProto{
			{Use_dep_enabled, Use_default_none, "gccgo"},
		}}),
	mkAtomTest("=dev-lang/go-10.0.1", "", 0, &atomProto{
		Atom: "dev-lang/go-10.0.1",
		Category: "dev-lang",
		Name: "go",
		VerRelop: Relop_eq,
		BaseVer: "00010.00000.00001",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00010.00000.00001 _n r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("dev-lang/go-10.0.1", "", a_error_if_no_prefix, &atomProto{
		Atom: "dev-lang/go-10.0.1",
		Category: "dev-lang",
		Name: "go",
		VerRelop: Relop_eq,
		BaseVer: "00010.00000.00001",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00010.00000.00001 _n r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("=dev-lang/go-10.0.1[gccgo]", "", a_error_if_input_atom, &atomProto{
		Atom: "dev-lang/go-10.0.1[gccgo]",
		Category: "dev-lang",
		Name: "go",
		VerRelop: Relop_eq,
		BaseVer: "00010.00000.00001",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00010.00000.00001 _n r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		UseDependencies: []useDependencyProto{
			{Use_dep_enabled, Use_default_none, "gccgo"},
		}}),
	mkAtomTest("dev-lang/go-10.0.1[gccgo]", "", a_error_if_input_atom|a_error_if_no_prefix,
		&atomProto{
		Atom: "dev-lang/go-10.0.1[gccgo]",
		Category: "dev-lang",
		Name: "go",
		VerRelop: Relop_eq,
		BaseVer: "00010.00000.00001",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00010.00000.00001 _n r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		UseDependencies: []useDependencyProto{
			{Use_dep_enabled, Use_default_none, "gccgo"},
		}}),
	mkAtomTest("<sys-apps/openrc-0.40", "", 0, &atomProto{
		Atom: ">sys-apps/openrc-0.17",
		Category: "sys-apps",
		Name: "openrc",
		VerRelop: Relop_lt,
		BaseVer: "00000.00040",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00000.00040 _n r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest(">=dev-libs/libffi-3.0.11", "", 0, &atomProto{
		Atom: ">=dev-libs/libffi-3.0.11",
		Category: "dev-libs",
		Name: "libffi",
		VerRelop: Relop_ge,
		BaseVer: "00003.00000.00011",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00003.00000.00011 _n r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest(">=dev-libs/libffi-3.0.11[-test]", "", a_error_if_input_atom, &atomProto{
		Atom: ">=dev-libs/libffi-3.0.11[-test]",
		Category: "dev-libs",
		Name: "libffi",
		VerRelop: Relop_ge,
		BaseVer: "00003.00000.00011",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00003.00000.00011 _n r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		UseDependencies: []useDependencyProto{
			{Use_dep_disabled, Use_default_none, "test"},
		}}),
	mkAtomTest(">=dev-libs/libffi-3.0.11[-test,static-libs?]", "", a_error_if_input_atom,
		&atomProto{
		Atom: ">=dev-libs/libffi-3.0.11[-test]",
		Category: "dev-libs",
		Name: "libffi",
		VerRelop: Relop_ge,
		BaseVer: "00003.00000.00011",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00003.00000.00011 _n r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		UseDependencies: []useDependencyProto{
			{Use_dep_disabled, Use_default_none, "test"},
			{Use_dep_set_only_if, Use_default_none, "static-libs"},
		}}),
	mkAtomTest(">=dev-libs/libffi-3.0.11[-test,static-libs=]", "", a_error_if_input_atom,
		&atomProto{
		Atom: ">=dev-libs/libffi-3.0.11[-test]",
		Category: "dev-libs",
		Name: "libffi",
		VerRelop: Relop_ge,
		BaseVer: "00003.00000.00011",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00003.00000.00011 _n r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		UseDependencies: []useDependencyProto{
			{Use_dep_disabled, Use_default_none, "test"},
			{Use_dep_same, Use_default_none, "static-libs"},
		}}),
	mkAtomTest("sys-devel/gcc[!cxx=]", "", a_error_if_input_atom, &atomProto{
		Atom: "sys-devel/gcc",
		Category: "sys-devel",
		Name: "gcc",
		VerRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		UseDependencies: []useDependencyProto{
			{Use_dep_opposite, Use_default_none, "cxx"},
		}}),
	mkAtomTest("sys-devel/gcc[!cxx=,!gtk?]", "", a_error_if_input_atom, &atomProto{
		Atom: "sys-devel/gcc",
		Category: "sys-devel",
		Name: "gcc",
		VerRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		UseDependencies: []useDependencyProto{
			{Use_dep_opposite, Use_default_none, "cxx"},
			{Use_dep_unset_only_if, Use_default_none, "gtk"},
		}}),
	mkAtomTest("<=app-admin/logrotate-3.8.0", "", 0, &atomProto{
		Atom: "<app-admin/logrotate-3.8.0",
		Category: "app-admin",
		Name: "logrotate",
		VerRelop: Relop_le,
		BaseVer: "00003.00008.00000",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00003.00008.00000 _n r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("=media-video/ffmpeg-4.3*", "", 0, &atomProto{
		Atom: "media-video/ffmpeg-4.3*",
		Category: "media-video",
		Name: "ffmpeg",
		VerRelop: Relop_range,
		BaseVer: "00004.00003",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00004.00003",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("media-video/ffmpeg-4.3*", "", a_error_if_no_prefix, &atomProto{
		Atom: "media-video/ffmpeg-4.3*",
		Category: "media-video",
		Name: "ffmpeg",
		VerRelop: Relop_range,
		BaseVer: "00004.00003",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00004.00003",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("=sys-apps/opentmpfiles-0.1.3-r1", "", 0, &atomProto{
		Atom: "sys-apps/opentmpfiles-0.1.3-r1",
		Category: "sys-apps",
		Name: "opentmpfiles",
		VerRelop: Relop_eq,
		BaseVer: "00000.00001.00003",
		Suffix: "_n",
		Revision: "r00001",
		CompVer: "00000.00001.00003 _n r00001",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("sys-apps/opentmpfiles-0.1.3-r1", "", a_error_if_no_prefix, &atomProto{
		Atom: "sys-apps/opentmpfiles-0.1.3-r1",
		Category: "sys-apps",
		Name: "opentmpfiles",
		VerRelop: Relop_eq,
		BaseVer: "00000.00001.00003",
		Suffix: "_n",
		Revision: "r00001",
		CompVer: "00000.00001.00003 _n r00001",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("~net-libs/libnet-1.0.2b", "", 0, &atomProto{
		Atom: "~net-libs/libnet-1.0.2b",
		Category: "net-libs",
		Name: "libnet",
		VerRelop: Relop_range,
		BaseVer: "00001.00000.00002 b",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00001.00000.00002 b",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("!app-text/dos2unix", "", 0, &atomProto{
		Atom: "!app-text/dos2unix",
		Category: "app-text",
		Name: "dos2unix",
		VerRelop: Relop_none,
		BaseVer: "",
		Suffix: "",
		Revision: "",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		Blocker: true}),
	mkAtomTest("!>sys-apps/openrc-0.17", "", 0, &atomProto{
		Atom: ">sys-apps/openrc-0.17",
		Category: "sys-apps",
		Name: "openrc",
		VerRelop: Relop_gt,
		BaseVer: "00000.00017",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00000.00017 _n r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		Blocker: true}),
	mkAtomTest("=dev-lang/python-3.10.0_alpha6-r2:3.10", "", 0, &atomProto{
		Atom: "=dev-lang/python/python-3.10.0_alpha6-r2:3.10",
		Category: "dev-lang",
		Name: "python",
		VerRelop: Relop_eq,
		BaseVer: "00003.00010.00000",
		Suffix: "_a00006",	// _alpha => _a
		Revision: "r00002",
		CompVer: "00003.00010.00000 _a00006 r00002",
		SlotRelop: Relop_eq,
		Slot: "00003.00010",
		Subslot: "00003.00010"}),
	mkAtomTest("=net-p2p/tvrss-1.8_beta", "", 0, &atomProto{
		Atom: "=net-p2p/tvrss-1.8_beta",
		Category: "net-p2p",
		Name: "tvrss",
		VerRelop: Relop_eq,
		BaseVer: "00001.00008",
		Suffix: "_b",		// _beta => _b
		Revision: "r00000",
		CompVer: "00001.00008 _b r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("!!<sys-apps/portage-2.1.4_rc1", "", 0, &atomProto{
		Atom: "!!<sys-apps/portage-2.1.4_rc1",
		Category: "sys-apps",
		Name: "portage",
		VerRelop: Relop_lt,
		BaseVer: "00002.00001.00004",
		Suffix: "_d00001",	// _rc => _d
		Revision: "r00000",
		CompVer: "00002.00001.00004 _d00001 r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		Blocker: true,
		HardBlock: true}),
	mkAtomTest("=sys-boot/aboot-1.0_pre20040408-r3", "", 0, &atomProto{
		Atom: "sys-boot/aboot-1.0_pre20040408-r3",
		Category: "sys-boot",
		Name: "aboot",
		VerRelop: Relop_eq,
		BaseVer: "00001.00000",
		Suffix: "_c20040408",	// _pre => _c
		Revision: "r00003",
		CompVer: "00001.00000 _c20040408 r00003",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest(">=net-misc/ipcalc-0.42_p2", "", 0, &atomProto{
		Atom: ">=net-misc/ipcalc-0.42_p2",
		Category: "net-misc",
		Name: "ipcalc",
		VerRelop: Relop_ge,
		BaseVer: "00000.00042",
		Suffix: "_p00002",	// _p => _p
		Revision: "r00000",
		CompVer: "00000.00042 _p00002 r00000",
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("sys-boot/aboot-1.0_pre20040408-r3", "", a_error_if_no_prefix, &atomProto{
		Atom: "sys-boot/aboot-1.0_pre20040408-r3",
		Category: "sys-boot",
		Name: "aboot",
		VerRelop: Relop_eq,
		BaseVer: "00001.00000",
		Suffix: "_c20040408",	// _pre => _c
		Revision: "r00003",
		CompVer: "00001.00000 _c20040408 r00003",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("dev-lang/python:3.8[threads(+)]", "", a_error_if_input_atom, &atomProto{
		Atom: "dev-lang/python:3.8[threads(+)]",
		Category: "dev-lang",
		Name: "python",
		VerRelop: Relop_none,
		SlotRelop: Relop_eq,
		Slot: "00003.00008",
		Subslot: "00003.00008",
		UseDependencies: []useDependencyProto{
			{Use_dep_enabled, Use_default_enabled, "threads"},
		}}),
	mkAtomTest("=virtual/package-manager-1::gentoo", "", 0, &atomProto{
		Atom: "virtual/package-manager-1::gentoo",
		Category: "virtual",
		Name: "package-manager", Repo: "gentoo",
		VerRelop: Relop_eq,
		BaseVer: "00001",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00001 _n r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("virtual/package-manager-1::gentoo", "", a_error_if_no_prefix, &atomProto{
		Atom: "virtual/package-manager-1::gentoo",
		Category: "virtual",
		Name: "package-manager", Repo: "gentoo",
		VerRelop: Relop_eq,
		BaseVer: "00001",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00001 _n r00000",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000"}),
	mkAtomTest("dev-lang/python-exec:2", "", 0, &atomProto{
		Atom: "dev-lang/python-exec:2",
		Category: "dev-lang",
		Name: "python-exec",
		VerRelop: Relop_none,
		BaseVer: "",
		Suffix: "",
		Revision: "",
		SlotRelop: Relop_eq,
		Slot: "00002",
		Subslot: "00002"}),
	mkAtomTest(">=kde-apps/kdegraphics-meta-20.12.3:5", "", 0, &atomProto{
		Atom: ">=kde-apps/kdegraphics-meta-20.12.3:5",
		Category: "kde-apps",
		Name: "kdegraphics-meta",
		VerRelop: Relop_ge,
		BaseVer: "00020.00012.00003",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00020.00012.00003 _n r00000",
		SlotRelop: Relop_eq,
		Slot: "00005",
		Subslot: "00005"}),
	mkAtomTest("media-libs/libpng:0=", "", 0, &atomProto{
		Atom: "media-libs/libpng:0=",
		Category: "media-libs",
		Name: "libpng",
		VerRelop: Relop_none,
		BaseVer: "",
		Suffix: "",
		Revision: "",
		SlotRelop: Relop_eq,
		Slot: "00000",
		Subslot: "00000",
		AnySlot: false,
		SameSlot: true,
		Blocker: false,
		HardBlock: false}),
	mkAtomTest("media-fonts/font-adobe-100dpi", "", 0, &atomProto{
		Atom: "media-fonts/font-adobe-100dpi",
		Category: "media-fonts",
		Name: "font-adobe-100dpi",
		VerRelop: Relop_none,
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		AnySlot: false,
		SameSlot: false,
		Blocker: false,
		HardBlock: false}),
	mkAtomTest("=media-fonts/font-adobe-100dpi-1.0.3-r1", "", 0, &atomProto{
		Atom: "media-fonts/font-adobe-100dpi",
		Category: "media-fonts",
		Name: "font-adobe-100dpi",
		VerRelop: Relop_eq,
		BaseVer: "00001.00000.00003",
		Suffix: "_n",
		Revision: "r00001",
		CompVer: "00001.00000.00003 _n r00001",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		AnySlot: false,
		SameSlot: false,
		Blocker: false,
		HardBlock: false}),
	mkAtomTest("media-fonts/font-adobe-100dpi-1.0.3-r1", "", a_error_if_no_prefix, &atomProto{
		Atom: "media-fonts/font-adobe-100dpi",
		Category: "media-fonts",
		Name: "font-adobe-100dpi",
		VerRelop: Relop_eq,
		BaseVer: "00001.00000.00003",
		Suffix: "_n",
		Revision: "r00001",
		CompVer: "00001.00000.00003 _n r00001",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		AnySlot: false,
		SameSlot: false,
		Blocker: false,
		HardBlock: false}),
	mkAtomTest("font-adobe-100dpi", "", 0, &atomProto{
		Atom: "font-adobe-100dpi",
		Category: "",
		Name: "font-adobe-100dpi",
		VerRelop: Relop_none,
		BaseVer: "",
		Suffix: "",
		Revision: "",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		AnySlot: false,
		SameSlot: false,
		Blocker: false,
		HardBlock: false}),
	mkAtomTest("font-adobe-100dpi-1.0.3-r1", "", a_error_if_no_prefix, &atomProto{
		Atom: "font-adobe-100dpi",
		Category: "",
		Name: "font-adobe-100dpi",
		VerRelop: Relop_eq,
		BaseVer: "00001.00000.00003",
		Suffix: "_n",
		Revision: "r00001",
		CompVer: "00001.00000.00003 _n r00001",
		SlotRelop: Relop_none,
		Slot: "00000",
		Subslot: "00000",
		AnySlot: false,
		SameSlot: false,
		Blocker: false,
		HardBlock: false}),
	mkAtomTest(">=kde-frameworks/kio-5.63.0:5", "", 0, &atomProto{
		Atom: ">=kde-frameworks/kio-5.63.0:5",
		Category: "kde-frameworks",
		Name: "kio",
		VerRelop: Relop_ge,
		BaseVer: "00005.00063.00000",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00005.00063.00000 _n r00000",
		SlotRelop: Relop_eq,
		Slot: "00005",
		Subslot: "00005",
		AnySlot: false,
		SameSlot: false,
		Blocker: false,
		HardBlock: false}),
	mkAtomTest(">=kde-frameworks/kio-5.63.0:5=", "", 0, &atomProto{
		Atom: ">=kde-frameworks/kio-5.63.0:5=",
		Category: "kde-frameworks",
		Name: "kio",
		VerRelop: Relop_ge,
		BaseVer: "00005.00063.00000",
		Suffix: "_n",
		Revision: "r00000",
		CompVer: "00005.00063.00000 _n r00000",
		SlotRelop: Relop_eq,
		Slot: "00005",
		Subslot: "00005",
		AnySlot: false,
		SameSlot: true,
		Blocker: false,
		HardBlock: false}),
	mkAtomTest("kde-frameworks/breeze-icons:*", "", 0, &atomProto{
		Atom: "kde-frameworks/breeze-icons:*",
		Category: "kde-frameworks",
		Name: "breeze-icons",
		VerRelop: Relop_none,
		SlotRelop: Relop_none,
		AnySlot: true,
		SameSlot: false,
		Blocker: false,
		HardBlock: false}),
}


func TestStandardInputAtoms(t *testing.T) {
	for _, tst := range atomTestSet {
		// Versioned atoms must have relop prefixes; USE dependencies not allowed
		decoded, err := RawParseAtom(tst.atomString, true, false)
		wantErrmsg := tst.pickErrmsg(a_error_if_no_prefix | a_error_if_input_atom)
		tryAtom(t, tst.atomString, wantErrmsg, tst.expected, decoded, err)
	}
}


func TestUnprefixedInputAtoms(t *testing.T) {
	for _, tst := range atomTestSet {
		// Versioned atoms may be unprefixed; USE dependencies not allowed
		decoded, err := RawParseAtom(tst.atomString, false, false)
		wantErrmsg := tst.pickErrmsg(a_error_if_input_atom)
		tryAtom(t, tst.atomString, wantErrmsg, tst.expected, decoded, err)
	}
}


func TestDependencyAtoms(t *testing.T) {
	for _, tst := range atomTestSet {
		// Versioned atoms must have relop prefixes; USE dependencies allowed
		decoded, err := RawParseAtom(tst.atomString, true, true)
		wantErrmsg := tst.pickErrmsg(a_error_if_no_prefix)
		tryAtom(t, tst.atomString, wantErrmsg, tst.expected, decoded, err)
	}
}

