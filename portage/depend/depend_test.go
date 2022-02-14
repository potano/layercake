package depend

import (
	"fmt"
	"testing"
)

var pkgdepType []string = []string{
	"Pkg_dep_atom",
	"Pkg_dep_all",
	"Pkg_dep_any_of",
	"Pkg_dep_exactly_one_of",
	"Pkg_dep_at_most_one_of",
	"Pkg_dep_when_use_set",
	"Pkg_dep_when_use_unset",
}


func printDependencies(deps []PackageDependency, indent string) {
	for _, dep := range deps {
		printDependency(dep, indent)
	}
}

const addIndent = "   "

var depGroupPrefix map[int]string = map[int]string{
	Pkg_dep_any_of: "|| ",
	Pkg_dep_exactly_one_of: "^^ ",
	Pkg_dep_at_most_one_of: "?? ",
}

func printDependency(dep PackageDependency, indent string) {
	switch dep.DependencyType() {
	case Pkg_dep_atom:
		fmt.Printf("%s%s\n", indent, dep.(*DependAtom).Atom)
	case Pkg_dep_all, Pkg_dep_any_of, Pkg_dep_exactly_one_of, Pkg_dep_at_most_one_of:
		fmt.Printf("%s%s(\n", indent, depGroupPrefix[dep.DependencyType()])
		printDependencies(dep.Dependencies(), indent + addIndent)
		fmt.Printf("%s)\n", indent)
	case Pkg_dep_when_use_set, Pkg_dep_when_use_unset:
		flag := dep.UseFlag()
		if dep.DependencyType() == Pkg_dep_when_use_unset {
			flag = "!" + flag
		}
		fmt.Printf("%s%s (\n", indent, flag)
		printDependencies(dep.Dependencies(), indent + addIndent)
		fmt.Printf("%s)\n", indent)
	}
}



type deptest struct {
	dependString string
	depPattern []patternItem
}

type patternItem struct {
	tp int
	text string
	patts []patternItem
}

type traceCheckRecord struct {
	t *testing.T
	deps []PackageDependency
	tstnum, linenum int
	printed bool
}

func newTracer(t *testing.T, tstnum int, deps []PackageDependency) *traceCheckRecord {
	return &traceCheckRecord{t: t, tstnum: tstnum, deps: deps}
}

func (tc *traceCheckRecord) nextLine() {
	tc.linenum++
}

func (tc *traceCheckRecord) errorf(str string, parms...interface{}) {
	if !tc.printed {
		tst := deptests[tc.tstnum]
		fmt.Printf("test %d: %s...\n", tc.tstnum, tst.dependString[:20])
		printDependencies(tc.deps, "")
		tc.printed = true
	}
	parms = append([]interface{}{tc.tstnum, tc.linenum}, parms...)
	tc.t.Errorf("test %d, line %d: " + str, parms...)
}

func checkDependencies(trace *traceCheckRecord, deps []PackageDependency, patt []patternItem) {
	if len(deps) != len(patt) {
		trace.errorf("expected %d dependencies, got %d", len(patt), len(deps))
		return
	}
	for i, item := range patt {
		trace.nextLine()
		dep := deps[i]
		deptype := dep.DependencyType()
		if item.tp == Pkg_dep_atom {
			if deptype != Pkg_dep_atom {
				trace.errorf("expected atom %s, got %s", item.text,
					pkgdepType[deptype])
			} else {
				atom := dep.(*DependAtom).Atom
				if atom != item.text {
					trace.errorf("expected atom %s, got %s", item.text,
						atom)
				}
			}
			return
		}
		if deptype != item.tp {
			trace.errorf("expected %s, got %s", pkgdepType[item.tp],
				pkgdepType[deptype])
		} else if dep.UseFlag() != item.text {
			trace.errorf("expected %s USE flag %s, got %s", pkgdepType[deptype],
				item.text, dep.UseFlag())
		}
		checkDependencies(trace, dep.Dependencies(), item.patts)
	}
}


var deptests []deptest = []deptest{
	{"!<dev-lang/python-exec-2.4.6-r4",
		[]patternItem{{Pkg_dep_atom, "!<dev-lang/python-exec-2.4.6-r4", nil}}},
	{"sys-libs/db:5.3/5.3= >=sys-libs/gdbm-1.8.3:0/6= app-arch/bzip2 sys-libs/zlib",
		[]patternItem{
			{Pkg_dep_atom, "sys-libs/db:5.3/5.3=", nil},
			{Pkg_dep_atom, ">=sys-libs/gdbm-1.8.3:0/6=", nil},
			{Pkg_dep_atom, "app-arch/bzip2", nil},
			{Pkg_dep_atom, "app-arch/bzip2 sys-libs/zlib", nil}}},
	{">=kde-frameworks/kpty-5.74.0:5 || ( kde-frameworks/breeze-icons:*" +
		" kde-frameworks/oxygen-icons:* ) >=kde-frameworks/kf-env-4",
		[]patternItem{
			{Pkg_dep_atom, ">=kde-frameworks/kpty-5.74.0:5", nil},
			{Pkg_dep_any_of, "", []patternItem{
				{Pkg_dep_atom, "kde-frameworks/breeze-icons:*", nil},
				{Pkg_dep_atom, "kde-frameworks/oxygen-icons:*", nil}}},
			{Pkg_dep_atom, ">=kde-frameworks/kf-env-4", nil},
		}},
	{"app-shells/bash dev-lang/perl || ( ( sys-apps/portage app-portage/portage-utils ) " +
		"sys-apps/pkgcore )",
		[]patternItem{
			{Pkg_dep_atom, "app-shells/bash", nil},
			{Pkg_dep_atom, "dev-lang/perl", nil},
			{Pkg_dep_any_of, "", []patternItem{
				{Pkg_dep_all, "", []patternItem{
					{Pkg_dep_atom, "sys-apps/portage", nil},
					{Pkg_dep_atom, "app-portage/portage-utils", nil}}},
				{Pkg_dep_atom, "sys-apps/pkgcore", nil}}},
		}},
}


func TestDependencies(t *testing.T) {
	for i, tst := range deptests {
		deps, err := DecodeDependencies([]byte(tst.dependString))
		if err != nil {
			t.Errorf("%s", err)
		} else {
			checkDependencies(newTracer(t, i, deps), deps, tst.depPattern)
		}
	}
}

