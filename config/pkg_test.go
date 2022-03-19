// Copyright © 2017, 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package config

import (
	"os"
	"fmt"
	"path"
	"errors"
	"strings"
	"io/ioutil"
	"potano.layercake/fns"
	"potano.layercake/defaults"

	"testing"
)


type Tmpdir struct {
	rootdir string
	err error
}

func NewTmpdir(patt string) (*Tmpdir, error) {
	name, err := ioutil.TempDir("", patt)
	return &Tmpdir{name, nil}, err
}

func (t *Tmpdir) Cleanup() {
	os.RemoveAll(t.rootdir)
}

func (t *Tmpdir) Err() error {
	return t.err
}

func (t *Tmpdir) Path(name string) string {
	return path.Join(t.rootdir, name)
}

func (t *Tmpdir) Mkdir(dirname string) string {
	pathname := t.Path(dirname)
	if t.err == nil {
		t.err = os.MkdirAll(pathname, 0755)
	}
	return pathname
}

func (t *Tmpdir) WriteFile(filename, contents string) string {
	pathname := t.Path(filename)
	if t.err == nil {
		t.err = ioutil.WriteFile(pathname, []byte(contents), 0644)
	}
	return pathname
}






type resulter struct {
	level int32
	errmsg string
}

const (
	expect_readFailure = iota
	expect_readSuccess
	expect_readMapFailure
	expect_readMapSuccess
	expect_patchPathFailure
	expect_patchPathSuccess
	expect_loadFailure
	expect_loadSuccess
	expect_configFailure
	expect_configSuccess
	expect_configPathFailure
	expect_configPathSuccess
	expect_baseTreeFailure
	expect_baseTreeSuccess
	expect_completeSuccess
)


type ssmap map[int]string

type tstcase struct {
	name, filename string
	want resulter
	cfgmap ssmap
	cfg *ConfigType
}

type tstcases struct {
	cases []*tstcase
	namemap map[string]*tstcase
}

var settingSetupConfigKeys map[int]string

func (tc *tstcases) AddFile(td *Tmpdir, name, filename, contents string, r resulter, cfgmap ssmap,
		cfg *ConfigType) {
	tmplmap := make(map[string]string, len(cfgmap))
	for key, val := range cfgmap {
		tmplmap[settingSetupConfigKeys[key]] = val
	}
	pathname := td.WriteFile(filename, fns.Template(contents, tmplmap))
	item := &tstcase{name, pathname, r, cfgmap, cfg}
	tc.cases = append(tc.cases, item)
	tc.namemap[name] = item
}

func (tc *tstcases) Add(name string, r resulter, cfg *ConfigType) {
	p_testcase := tc.namemap[name]
	if p_testcase == nil {
		return
	}
	if cfg == nil {
		cfg = p_testcase.cfg
	}
	item := &tstcase{name, p_testcase.filename, r, p_testcase.cfgmap, cfg}
	tc.cases = append(tc.cases, item)
}



func samemap(map1, map2 ssmap) error {
	var extra, diff []string
	for key, val := range map1 {
		if map2[key] != val {
			diff = append(diff, settingSetupConfigKeys[key] + "=" + val)
		}
	}
	for key, _ := range map2 {
		if _, have := map1[key]; !have {
			extra = append(extra, settingSetupConfigKeys[key])
		}
	}
	if len(extra) > 0 || len(diff) > 0 {
		out := make([]string, 0, 2)
		if len(diff) > 0 {
			out = append(out, "unexpected values: " + strings.Join(diff, ", "))
		}
		if len(extra) > 0 {
			out = append(out, "unexpected keys: " + strings.Join(extra, ", "))
		}
		return errors.New(strings.Join(out, "; "))
	}
	return nil
}


func compareConfigType(expt, have *ConfigType) error {
	if expt.Basepath != have.Basepath {
		return fmt.Errorf("expected Basepath=%s, got %s", expt.Basepath, have.Basepath)
	}
	if expt.Layerdirs != have.Layerdirs {
		return fmt.Errorf("expected Layerdirs=%s, got %s", expt.Layerdirs, have.Layerdirs)
	}
	if expt.LayerBuildRoot != have.LayerBuildRoot {
		return fmt.Errorf("expected Layerdirs=%s, got %s", expt.LayerBuildRoot,
			have.LayerBuildRoot)
	}
	if expt.LayerBinPkgdir != have.LayerBinPkgdir {
		return fmt.Errorf("expected LayerBinPkgdir=%s, got %s", expt.LayerBinPkgdir,
			have.LayerBinPkgdir)
	}
	if expt.LayerGeneratedir != have.LayerGeneratedir {
		return fmt.Errorf("expected LayerGeneratedir=%s, got %s", expt.LayerGeneratedir,
			have.LayerGeneratedir)
	}
	if expt.LayerOvfsWorkdir != have.LayerOvfsWorkdir {
		return fmt.Errorf("expected LayerOvfsWorkdir=%s, got %s", expt.LayerOvfsWorkdir,
			have.LayerOvfsWorkdir)
	}
	if expt.LayerOvfsUpperdir != have.LayerOvfsUpperdir {
		return fmt.Errorf("expected LayerOvfsUpperdir=%s, got %s", expt.LayerOvfsUpperdir,
			have.LayerOvfsUpperdir)
	}
	if expt.Exportdirs != have.Exportdirs {
		return fmt.Errorf("expected Exportdirs=%s, got %s", expt.Exportdirs,
			have.Exportdirs)
	}
	if expt.ExportBinPkgdir != have.ExportBinPkgdir {
		return fmt.Errorf("expected ExportBinPkgdir=%s, got %s", expt.ExportBinPkgdir,
			have.ExportBinPkgdir)
	}
	if expt.ExportGeneratedir != have.ExportGeneratedir {
		return fmt.Errorf("expected ExportGeneratedir=%s, got %s", expt.ExportGeneratedir,
			have.ExportGeneratedir)
	}
	if expt.ChrootExec != have.ChrootExec {
		return fmt.Errorf("expected ChrootExec=%s, got %s", expt.ChrootExec,
			have.ChrootExec)
	}
	return nil
}


func errIfMissingPaths(missing []string) error {
	if len(missing) > 0 {
		return errors.New(strings.Join(missing, ", "))
	}
	return nil
}


func errIfNonBase(haveNonBasePaths bool) error {
	if haveNonBasePaths {
		return errors.New("have non-base paths")
	}
	return nil
}




func fixUpPathMap(cfg map[int]string) map[int]string {
	out := map[int]string{}
	basepath := cfg[cfKey_basepath]
	for key, val := range cfg {
		switch key {
		case cfKey_configfile, cfKey_layerdirs, cfKey_exportroot:
			if !path.IsAbs(val) && len(basepath) > 0 {
				val = path.Join(basepath, val)
			}
		}
		out[key] = val
	}
	return out
}


type confTypeMap map[int]interface{}

func makeConfigTypeObj(m confTypeMap) *ConfigType {
	obj := ConfigType{
		Basepath: defaults.BasePath,
		Layerdirs: path.Join(defaults.BasePath, defaults.Layerdirs),
		LayerBuildRoot: defaults.Builddir,
		LayerBinPkgdir: defaults.Pkgdir,
		LayerGeneratedir: defaults.Generateddir,
		LayerOvfsWorkdir: defaults.Workdir,
		LayerOvfsUpperdir: defaults.Upperdir,
		Exportdirs: path.Join(defaults.BasePath, defaults.Exportdirs),
		ExportBinPkgdir: defaults.Pkgdir,
		ExportGeneratedir: defaults.Generateddir,
		LayerExportDirs: map[string]string{"builds": "builds", "generated": "generated",
			"packages": "packages"},
		ChrootExec: defaults.ChrootExec,
	}
	if m != nil {
		for key, value := range m {
			stringVal := value.(string)
			switch key {
			case cfKey_basepath:
				obj.Basepath = stringVal
			case cfKey_layerdirs:
				obj.Layerdirs = stringVal
			case cfKey_buildroot:
				obj.LayerBuildRoot = stringVal
			case cfKey_binpkgdir:
				obj.LayerBinPkgdir = stringVal
			case cfKey_gendir:
				obj.LayerGeneratedir = stringVal
			case cfKey_workdir:
				obj.LayerOvfsWorkdir = stringVal
			case cfKey_upperdir:
				obj.LayerOvfsUpperdir = stringVal
			case cfKey_exportroot:
				obj.Exportdirs = stringVal
			case cfKey_exportpkgdir:
				obj.ExportBinPkgdir = stringVal
			case cfKey_exportgendir:
				obj.ExportGeneratedir = stringVal
			case cfKey_chrootexec:
				obj.ChrootExec = stringVal
			}
		}
	}
	return &obj
}


func makeCases(td *Tmpdir) (*tstcases, error) {
	normResult := resulter{expect_completeSuccess, ""}

	tc := &tstcases{namemap: map[string]*tstcase{}}

	tc.AddFile(td, "emptyfile", "emptyfile.conf",
		"",
		normResult,
		ssmap{},
		makeConfigTypeObj(nil))


	tc.AddFile(td, "just_basepath", "just_basepath.conf",
		"basepath = {BASEPATH}",
		normResult,
		ssmap{cfKey_basepath: "/var/lib/there"},
		makeConfigTypeObj(confTypeMap{cfKey_basepath: "/var/lib/there",
			cfKey_layerdirs: "/var/lib/there/layers",
			cfKey_exportroot: "/var/lib/there/export"}),
		)

	tc.AddFile(td, "basepath_whitesp", "basepath_and_blank.conf",
		"\nbasepath={BASEPATH}\n",
		normResult,
		ssmap{cfKey_basepath: "/var/lib/elsewhere"},
		makeConfigTypeObj(confTypeMap{cfKey_basepath: "/var/lib/elsewhere",
			cfKey_layerdirs: "/var/lib/elsewhere/layers",
			cfKey_exportroot: "/var/lib/elsewhere/export"}),
		)

	td.Mkdir("var/lib/binpackager/layer_root")
	td.Mkdir("var/lib/binpackager/export_root")
	tc.AddFile(td, "all_keys", "setall.conf",
		`# Set all values
		basepath={BASEPATH}
		ConfigFile={CONFIGFILE}
		LAYERS={LAYERS}
		buildroot = {BUILDROOT} 
		binpkgs = {BINPKGS}
		generated_files = {GENERATED_FILES}
		OverFS_workdir = {OVERFS_WORKDIR}
		overfs_upperdir= {OVERFS_UPPERDIR}
		exports ={EXPORTS}
		export_binpkgs={EXPORT_BINPKGS}
		export_generated_files = {EXPORT_GENERATED_FILES}
		chroot_Exec={CHROOT_EXEC}`,
		normResult,
		ssmap{
			cfKey_basepath: td.Path("/var/lib/binpackager"),
			cfKey_configfile: td.Path("emptyfile.conf"),
			cfKey_layerdirs: "layer_root",
			cfKey_buildroot: "bld",
			cfKey_binpkgdir: "packages",
			cfKey_gendir: "gengen",
			cfKey_workdir: "overlayfs/work",
			cfKey_upperdir: "overlayfs/upper",
			cfKey_exportroot: "export_root",
			cfKey_exportpkgdir: "pkgs",
			cfKey_exportgendir: "gen",
			cfKey_chrootexec: "/usr/bin/chroot"},
		makeConfigTypeObj(confTypeMap{
			cfKey_basepath: td.Path("/var/lib/binpackager"),
			cfKey_layerdirs: td.Path("/var/lib/binpackager/layer_root"),
			cfKey_buildroot: "bld",
			cfKey_binpkgdir: "packages",
			cfKey_gendir: "gengen",
			cfKey_workdir: "overlayfs/work",
			cfKey_upperdir: "overlayfs/upper",
			cfKey_exportroot: td.Path("/var/lib/binpackager/export_root"),
			cfKey_exportpkgdir: "pkgs",
			cfKey_exportgendir: "gen",
		}))

	tc.AddFile(td, "parse_failure", "fail_parse.conf",
		"unknown=val",
		resulter{expect_readFailure, "Unrecognized setting 'unknown'"},
		ssmap{},
		nil)

/*
	tc.AddFile(td, "value_error", "wrong_val.conf",
		"layers=/var/lib/here",
		resulter{expect_readMapFailure, "unexpected values: layers=/var/lib/elsewhere"},
		ssmap{cfKey_layerdirs: "/var/lib/elsewhere"},
		nil)
*/

	circ1Path := td.Path("circular1.conf")
	circ2Path := td.Path("circular2.conf")
	tc.AddFile(td, "circular1", "circular1.conf",
		"CONFIGFILE = {CONFIGFILE}",
		resulter{expect_loadFailure, "Config-file loop: have seen " + circ1Path},
		ssmap{cfKey_configfile: circ2Path},
		nil)

	tc.AddFile(td, "circular2", "circular2.conf",
		"CONFIGFILE = {CONFIGFILE}",
		resulter{expect_loadFailure, "Config-file loop: have seen " + circ2Path},
		ssmap{cfKey_configfile: circ1Path},
		nil)

	td.Mkdir("home/builder/layercake/layer_dirs")
	tc.AddFile(td, "intermediate1", "intermediate1.conf",
		`# Set intermediate values for which setall.conf will set remainder
		BASEPATH = {BASEPATH}
		CONFIGFILE = {CONFIGFILE}
		buildroot = {BUILDROOT}
		chroot_exec = {CHROOT_EXEC}`,
		normResult,
		ssmap{
			cfKey_basepath: td.Path("/home/builder/layercake"),
			cfKey_configfile: td.Path("setall.conf"),
			cfKey_buildroot: "build_root",
			cfKey_chrootexec: "/sbin/chroot"},
		makeConfigTypeObj(confTypeMap{
			cfKey_basepath: td.Path("/home/builder/layercake"),
			cfKey_layerdirs: td.Path("/home/builder/layercake/layer_root"),
			cfKey_buildroot: "build_root",
			cfKey_gendir: "gengen",
			cfKey_workdir: "overlayfs/work",
			cfKey_upperdir: "overlayfs/upper",
			cfKey_exportroot: td.Path("/home/builder/layercake/export_root"),
			cfKey_chrootexec: "/sbin/chroot",
			cfKey_exportpkgdir: "pkgs",
			cfKey_exportgendir: "gen",
		}))

	tc.AddFile(td, "intermediate2", "intermediate2.conf",
		`# Set values where intermediate1.conf and setall.conf set defaults
		layers = {LAYERS}
		basepath = {BASEPATH}
		CONFIGFILE = {CONFIGFILE}`,
		normResult,
		ssmap{
			cfKey_basepath: td.Path("/home/builder/layercake"),
			cfKey_configfile: td.Path("intermediate1.conf"),
			cfKey_layerdirs: "layer_dirs"},
		makeConfigTypeObj(confTypeMap{
			cfKey_basepath: td.Path("/home/builder/layercake"),
			cfKey_layerdirs: td.Path("/home/builder/layercake/layer_dirs"),
			cfKey_buildroot: "build_root",
			cfKey_gendir: "gengen",
			cfKey_workdir: "overlayfs/work",
			cfKey_upperdir: "overlayfs/upper",
			cfKey_exportroot: td.Path("/home/builder/layercake/export_root"),
			cfKey_exportpkgdir: "pkgs",
			cfKey_exportgendir: "gen",
			cfKey_chrootexec: "/sbin/chroot",
		}))

	tc.AddFile(td, "intermediate3", "intermediate3.conf",
		`# As with intermediate2.conf but with a directory outside of base path
		layers = {LAYERS}
		basepath = {BASEPATH}
		exports = {EXPORTS}
		CONFIGFILE = {CONFIGFILE}`,
		normResult,
		ssmap{
			cfKey_basepath: td.Path("/home/builder/layercake"),
			cfKey_configfile: td.Path("intermediate2.conf"),
			cfKey_layerdirs: "layer_dirs",
			cfKey_exportroot: td.Path("/var/lib/binpackager/export_root")},
		makeConfigTypeObj(confTypeMap{
			cfKey_basepath: td.Path("/home/builder/layercake"),
			cfKey_layerdirs: td.Path("/home/builder/layercake/layer_dirs"),
			cfKey_buildroot: "build_root",
			cfKey_gendir: "gengen",
			cfKey_workdir: "overlayfs/work",
			cfKey_upperdir: "overlayfs/upper",
			cfKey_exportroot: td.Path("/var/lib/binpackager/export_root"),
			cfKey_exportpkgdir: "pkgs",
			cfKey_exportgendir: "gen",
			cfKey_chrootexec: "/sbin/chroot",
		}))

	return tc, td.Err()
}


func makeCases_LoadEnv(orig_tc *tstcases, td *Tmpdir) *tstcases {
	normResult := resulter{expect_completeSuccess, ""}
	configSuccess := resulter{expect_configSuccess, ""}
	tc := &tstcases{namemap: orig_tc.namemap}

	tc.Add("emptyfile", configSuccess, nil)
	tc.Add("just_basepath", configSuccess, nil)
	tc.Add("basepath_whitesp", configSuccess, nil)
	tc.Add("all_keys", normResult, nil)
	tc.Add("parse_failure", resulter{expect_loadFailure, "Unrecognized setting 'unknown'"}, nil)
	tc.Add("circular1", resulter{expect_loadFailure, "Config-file loop: have seen " +
		td.Path("circular1.conf")}, nil)
	tc.Add("intermediate2", resulter{expect_configPathFailure,
		td.Path("/home/builder/layercake/export_root")}, nil)
	tc.Add("intermediate3", resulter{expect_baseTreeFailure, "have non-base paths"}, nil)

	return tc
}


func makeCases_LoadPathEnv(orig_tc *tstcases, td *Tmpdir) *tstcases {
	envbase := td.Path("/var/lib/mkpkg")
	normResult := resulter{expect_completeSuccess, ""}
	tc := &tstcases{namemap: orig_tc.namemap}

	tc.Add("emptyfile", normResult, makeConfigTypeObj(confTypeMap{
		cfKey_basepath: envbase,
		cfKey_layerdirs: path.Join(envbase, "layers"),
		cfKey_exportroot: path.Join(envbase, "export"),
	}))
	tc.Add("just_basepath", normResult, makeConfigTypeObj(confTypeMap{
		cfKey_basepath: envbase,
		cfKey_layerdirs: path.Join(envbase, "layers"),
		cfKey_exportroot: path.Join(envbase, "export"),
	}))
	tc.Add("basepath_whitesp", normResult, makeConfigTypeObj(confTypeMap{
		cfKey_basepath: envbase,
		cfKey_layerdirs: path.Join(envbase, "layers"),
		cfKey_exportroot: path.Join(envbase, "export"),
	}))
	tc.Add("all_keys", normResult, makeConfigTypeObj(confTypeMap{
		cfKey_basepath: envbase,
		cfKey_layerdirs: path.Join(envbase, "layer_root"),
		cfKey_buildroot: "bld",
		cfKey_gendir: "gengen",
		cfKey_workdir: "overlayfs/work",
		cfKey_upperdir: "overlayfs/upper",
		cfKey_exportpkgdir: "pkgs",
		cfKey_exportgendir: "gen",
		cfKey_exportroot: path.Join(envbase, "export_root"),
	}))
	tc.Add("parse_failure", resulter{expect_loadFailure, "Unrecognized setting 'unknown'"}, nil)
	tc.Add("circular1", resulter{expect_loadFailure, "Config-file loop: have seen " +
		td.Path("circular1.conf")}, nil)
	tc.Add("intermediate2", normResult, makeConfigTypeObj(confTypeMap{
		cfKey_basepath: envbase,
		cfKey_layerdirs: path.Join(envbase, "layer_dirs"),
		cfKey_buildroot: "build_root",
		cfKey_gendir: "gengen",
		cfKey_workdir: "overlayfs/work",
		cfKey_upperdir: "overlayfs/upper",
		cfKey_exportroot: path.Join(envbase, "export_root"),
		cfKey_exportpkgdir: "pkgs",
		cfKey_exportgendir: "gen",
		cfKey_chrootexec: "/sbin/chroot",
	}))

	return tc
}


func TestConfig(t *testing.T) {
	td, err := NewTmpdir("layercake_config")
	if err != nil {
		t.Fatal(err)
	}
	defer td.Cleanup()

	tc, err := makeCases(td)
	if err != nil {
		t.Fatal(err)
	}

	for _, tst := range tc.cases {
		t.Run("read " + tst.name, func (t *testing.T) {
			cfgmap, err := readConfigFile(tst.filename)
			if testFailed(t, tst, err, expect_readSuccess, "readConfigFile") {
				return
			}
			err = samemap(tst.cfgmap, cfgmap)
			if testFailed(t, tst, err, expect_readMapSuccess, "samemap") {
				return
			}
			err = patchPaths(cfgmap)
			if testFailed(t, tst, err, expect_patchPathSuccess, "patchPaths") {
				return
			}
			err = samemap(fixUpPathMap(tst.cfgmap), cfgmap)
			if testFailed(t, tst, err, expect_patchPathSuccess, "compare patchPaths") {
				return
			}
		})
	}

	for _, tst := range tc.cases {
		if tst.want.level < expect_loadFailure {
			continue
		}
		t.Run("Load " + tst.name, func (t *testing.T) {
//fmt.Printf("loading %s\n", tst.filename)
			cfg, err := Load(tst.filename, "")
			if testFailed(t, tst, err, expect_loadSuccess, "config.Load") {
				return
			}
			err = compareConfigType(tst.cfg, cfg)
//fmt.Printf("expected LayerGeneratedir: %s\n", tst.cfg.LayerGeneratedir)
//fmt.Printf("     got LayerGeneratedir: %s\n", cfg.LayerGeneratedir)
			if testFailed(t, tst, err, expect_configSuccess, "config.Load values") {
				return
			}
		})
	}

	tc = makeCases_LoadEnv(tc, td)
	for _, tst := range tc.cases {
		t.Run("Load config env " + tst.name, func (t *testing.T) {
			os.Setenv("LAYERCONF", tst.filename)
			cfg, err := Load("", "")
			if testFailed(t, tst, err, expect_loadSuccess, "config.Load") {
				return
			}
			err = compareConfigType(tst.cfg, cfg)
			if testFailed(t, tst, err, expect_configSuccess, "config.Load values") {
				return
			}
			missing, haveNonBasePaths := cfg.CheckConfigPaths()
			err = errIfMissingPaths(missing)
			if testFailed(t, tst, err, expect_configPathSuccess, "missing paths") {
				return
			}
			err = errIfNonBase(haveNonBasePaths)
			if testFailed(t, tst, err, expect_baseTreeSuccess, "non-base paths") {
				return
			}
		})
	}

	tc = makeCases_LoadPathEnv(tc, td)
	envbase := td.Path("/var/lib/mkpkg")
	for _, tst := range tc.cases {
		t.Run("Load basepath env " + tst.name, func (t *testing.T) {
			os.Setenv("LAYERROOT", envbase)
			cfg, err := Load(tst.filename, "")
			if testFailed(t, tst, err, expect_loadSuccess, "config.Load") {
				return
			}
			err = compareConfigType(tst.cfg, cfg)
			if testFailed(t, tst, err, expect_configSuccess, "config.Load values") {
				return
			}
		})
	}
}

func testFailed(t *testing.T, tst *tstcase, err error, level int32, desc string) bool {
	if level > tst.want.level {
		if err == nil {
			t.Errorf("%s should have failed", desc)
		} else if tst.want.errmsg != err.Error() {
			t.Errorf("%s failed with wrong error: %s", desc, err.Error())
		}
		return true
	}
	if err != nil {
		t.Errorf("%s failed: %s", desc, err.Error())
		return true
	}
	return level == tst.want.level
}


func init() {
	keys := make(map[int]string, len(settingSetup))
	for _, item := range settingSetup {
		keys[item.key] = item.configKey
	}
	settingSetupConfigKeys = keys
}

