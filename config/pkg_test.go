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


type ssmap map[string]string

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

func (tc *tstcases) AddFile(td *Tmpdir, name, filename, contents string, r resulter, cfgmap ssmap,
		cfg *ConfigType) {
	pathname := td.WriteFile(filename, fns.Template(contents, cfgmap))
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
			diff = append(diff, key + "=" + val)
		}
	}
	for key, _ := range map2 {
		if _, have := map1[key]; !have {
			extra = append(extra, key)
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
	if expt.ChrootExec != have.ChrootExec {
		return fmt.Errorf("expected ChrootExec=%s, got %s", expt.ChrootExec,
			have.ChrootExec)
	}
	err := samemap(expt.LayerExportDirs, have.LayerExportDirs)
	if err != nil {
		return errors.New("LayerExportDirs: " + err.Error())
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




func fixUpPathMap(cfg map[string]string) map[string]string {
	out := map[string]string{}
	basepath := cfg["basepath"]
	for key, val := range cfg {
		switch key {
		case "configfile", "layerdirs", "exportroot":
			if !path.IsAbs(val) && len(basepath) > 0 {
				val = path.Join(basepath, val)
			}
		}
		out[key] = val
	}
	return out
}


type confTypeMap map[string]interface{}

func makeConfigTypeObj(m confTypeMap) *ConfigType {
	obj := ConfigType{
		Basepath: defaults.BasePath,
		Layerdirs: path.Join(defaults.BasePath, defaults.Layerdirs),
		LayerBuildRoot: defaults.Builddir,
		LayerOvfsWorkdir: defaults.Workdir,
		LayerOvfsUpperdir: defaults.Upperdir,
		Exportdirs: path.Join(defaults.BasePath, defaults.Exportdirs),
		LayerExportDirs: map[string]string{"builds": "builds", "generated": "generated",
			"packages": "packages"},
		ChrootExec: defaults.ChrootExec,
	}
	if m != nil {
		for key, value := range m {
			switch key {
			case "basepath":
				obj.Basepath = value.(string)
			case "layerdirs":
				obj.Layerdirs = value.(string)
			case "buildroot":
				obj.LayerBuildRoot = value.(string)
			case "workdir":
				obj.LayerOvfsWorkdir = value.(string)
			case "upperdir":
				obj.LayerOvfsUpperdir = value.(string)
			case "exportroot":
				obj.Exportdirs = value.(string)
			case "exportdirs":
				obj.LayerExportDirs = value.(map[string]string)
			case "chrootexec":
				obj.ChrootExec = value.(string)
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
		"basepath = {basepath}",
		normResult,
		ssmap{"basepath": "/var/lib/there"},
		makeConfigTypeObj(confTypeMap{"basepath": "/var/lib/there",
			"layerdirs": "/var/lib/there/layers",
			"exportroot": "/var/lib/there/export"}),
		)

	tc.AddFile(td, "basepath_whitesp", "basepath_and_blank.conf",
		"\nbasepath={basepath}\n",
		normResult,
		ssmap{"basepath": "/var/lib/elsewhere"},
		makeConfigTypeObj(confTypeMap{"basepath": "/var/lib/elsewhere",
			"layerdirs": "/var/lib/elsewhere/layers",
			"exportroot": "/var/lib/elsewhere/export"}),
		)

	td.Mkdir("var/lib/binpackager/layer_root")
	td.Mkdir("var/lib/binpackager/export_root")
	tc.AddFile(td, "all_keys", "setall.conf",
		`# Set all values
		basepath={basepath}
		ConfigFile={configfile}
		LAYERS={layerdirs}
		buildroot = {buildroot} 
		workdir = {workdir}
		upperdir= {upperdir}
		exportroot ={exportroot}
		chrootExec={chrootexec}`,
		normResult,
		ssmap{
			"basepath": td.Path("/var/lib/binpackager"),
			"configfile": td.Path("emptyfile.conf"),
			"layerdirs": "layer_root",
			"buildroot": "bld",
			"workdir": "overlayfs/work",
			"upperdir": "overlayfs/upper",
			"exportroot": "export_root",
			"chrootexec": "/usr/bin/chroot"},
		makeConfigTypeObj(confTypeMap{
			"basepath": td.Path("/var/lib/binpackager"),
			"layerdirs": td.Path("/var/lib/binpackager/layer_root"),
			"buildroot": "bld",
			"workdir": "overlayfs/work",
			"upperdir": "overlayfs/upper",
			"exportroot": td.Path("/var/lib/binpackager/export_root"),
		}))

	tc.AddFile(td, "parse_failure", "fail_parse.conf",
		"unknown=val",
		resulter{expect_readFailure, "Unrecognized setting 'unknown'"},
		ssmap{},
		nil)

	tc.AddFile(td, "value_error", "wrong_val.conf",
		"layers=/var/lib/here",
		resulter{expect_readMapFailure, "unexpected values: layerdirs=/var/lib/elsewhere"},
		ssmap{"layerdirs": "/var/lib/elsewhere"},
		nil)

	circ1Path := td.Path("circular1.conf")
	circ2Path := td.Path("circular2.conf")
	tc.AddFile(td, "circular1", "circular1.conf",
		"CONFIGFILE = {configfile}",
		resulter{expect_loadFailure, "Config-file loop: have seen " + circ1Path},
		ssmap{"configfile": circ2Path},
		nil)

	tc.AddFile(td, "circular2", "circular2.conf",
		"CONFIGFILE = {configfile}",
		resulter{expect_loadFailure, "Config-file loop: have seen " + circ2Path},
		ssmap{"configfile": circ1Path},
		nil)

	td.Mkdir("home/builder/layercake/layer_dirs")
	tc.AddFile(td, "intermediate1", "intermediate1.conf",
		`# Set intermediate values for which setall.conf will set remainder
		BASEPATH = {basepath}
		CONFIGFILE = {configfile}
		buildroot = {buildroot}
		chroot_exec = {chrootexec}`,
		normResult,
		ssmap{
			"basepath": td.Path("/home/builder/layercake"),
			"configfile": td.Path("setall.conf"),
			"buildroot": "build_root",
			"chrootexec": "/sbin/chroot"},
		makeConfigTypeObj(confTypeMap{
			"basepath": td.Path("/home/builder/layercake"),
			"layerdirs": td.Path("/home/builder/layercake/layer_root"),
			"buildroot": "build_root",
			"workdir": "overlayfs/work",
			"upperdir": "overlayfs/upper",
			"exportroot": td.Path("/home/builder/layercake/export_root"),
			"chrootexec": "/sbin/chroot",
		}))

	tc.AddFile(td, "intermediate2", "intermediate2.conf",
		`# Set values where intermediate1.conf and setall.conf set defaults
		layers = {layerdirs}
		basepath = {basepath}
		CONFIGFILE = {configfile}`,
		normResult,
		ssmap{
			"basepath": td.Path("/home/builder/layercake"),
			"configfile": td.Path("intermediate1.conf"),
			"layerdirs": "layer_dirs"},
		makeConfigTypeObj(confTypeMap{
			"basepath": td.Path("/home/builder/layercake"),
			"layerdirs": td.Path("/home/builder/layercake/layer_dirs"),
			"buildroot": "build_root",
			"workdir": "overlayfs/work",
			"upperdir": "overlayfs/upper",
			"exportroot": td.Path("/home/builder/layercake/export_root"),
			"chrootexec": "/sbin/chroot",
		}))

	tc.AddFile(td, "intermediate3", "intermediate3.conf",
		`# As with intermediate2.conf but with a directory outside of base path
		layers = {layerdirs}
		basepath = {basepath}
		export_root = {exportroot}
		CONFIGFILE = {configfile}`,
		normResult,
		ssmap{
			"basepath": td.Path("/home/builder/layercake"),
			"configfile": td.Path("intermediate2.conf"),
			"layerdirs": "layer_dirs",
			"exportroot": td.Path("/var/lib/binpackager/export_root")},
		makeConfigTypeObj(confTypeMap{
			"basepath": td.Path("/home/builder/layercake"),
			"layerdirs": td.Path("/home/builder/layercake/layer_dirs"),
			"buildroot": "build_root",
			"workdir": "overlayfs/work",
			"upperdir": "overlayfs/upper",
			"exportroot": td.Path("/var/lib/binpackager/export_root"),
			"chrootexec": "/sbin/chroot",
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
		"basepath": envbase,
		"layerdirs": path.Join(envbase, "layers"),
		"exportroot": path.Join(envbase, "export"),
	}))
	tc.Add("just_basepath", normResult, makeConfigTypeObj(confTypeMap{
		"basepath": envbase,
		"layerdirs": path.Join(envbase, "layers"),
		"exportroot": path.Join(envbase, "export"),
	}))
	tc.Add("basepath_whitesp", normResult, makeConfigTypeObj(confTypeMap{
		"basepath": envbase,
		"layerdirs": path.Join(envbase, "layers"),
		"exportroot": path.Join(envbase, "export"),
	}))
	tc.Add("all_keys", normResult, makeConfigTypeObj(confTypeMap{
		"basepath": envbase,
		"layerdirs": path.Join(envbase, "layer_root"),
		"buildroot": "bld",
		"workdir": "overlayfs/work",
		"upperdir": "overlayfs/upper",
		"exportroot": path.Join(envbase, "export_root"),
	}))
	tc.Add("parse_failure", resulter{expect_loadFailure, "Unrecognized setting 'unknown'"}, nil)
	tc.Add("circular1", resulter{expect_loadFailure, "Config-file loop: have seen " +
		td.Path("circular1.conf")}, nil)
	tc.Add("intermediate2", normResult, makeConfigTypeObj(confTypeMap{
		"basepath": envbase,
		"layerdirs": path.Join(envbase, "layer_dirs"),
		"buildroot": "build_root",
		"workdir": "overlayfs/work",
		"upperdir": "overlayfs/upper",
		"exportroot": path.Join(envbase, "export_root"),
		"chrootexec": "/sbin/chroot",
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

