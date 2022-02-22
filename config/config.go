/* Processing for layercake's configuration values */

package config

import (
	"os"
	"fmt"
	"path"
	"strings"

	"potano.layercake/fs"
	"potano.layercake/defaults"
)


type ConfigType struct {
	Basepath string
	Layerdirs string
	LayerBuildRoot string
	LayerBinPkgdir string
	LayerGeneratedir string
	LayerOvfsWorkdir string
	LayerOvfsUpperdir string
	Exportdirs string
	ExportBinPkgdir string
	ExportGeneratedir string
	LayerExportDirs map[string]string
	ChrootExec string
}


const (
	ss_value = iota
	ss_file
	ss_dir
)


const (
	cfKey_none = iota
	cfKey_basepath
	cfKey_configfile
	cfKey_layerdirs
	cfKey_buildroot
	cfKey_binpkgdir
	cfKey_gendir
	cfKey_workdir
	cfKey_upperdir
	cfKey_exportroot
	cfKey_exportpkgdir
	cfKey_exportgendir
	cfKey_chrootexec
)


type cfsetup struct {
	key int
	s_type int
	relative_to int
	default_value string
	configKey string
}



var settingSetup = []cfsetup {
	cfsetup{cfKey_basepath, ss_dir, 0, defaults.BasePath, "BASEPATH"},
	cfsetup{cfKey_configfile, ss_file, 0, "", "CONFIGFILE"},
	cfsetup{cfKey_layerdirs, ss_dir, cfKey_basepath, defaults.Layerdirs, "LAYERS"},
	cfsetup{cfKey_buildroot, ss_value, 0, defaults.Builddir, "BUILDROOT"},
	cfsetup{cfKey_binpkgdir, ss_value, 0, defaults.Pkgdir, "BINPKGS"},
	cfsetup{cfKey_gendir, ss_value, 0, defaults.Generateddir, "GENERATED_FILES"},
	cfsetup{cfKey_workdir, ss_value, 0, defaults.Workdir, "OVERFS_WORKDIR"},
	cfsetup{cfKey_upperdir, ss_value, 0, defaults.Upperdir, "OVERFS_UPPERDIR"},
	cfsetup{cfKey_exportroot, ss_dir, cfKey_basepath, defaults.Exportdirs, "EXPORTS"},
	cfsetup{cfKey_exportpkgdir, ss_value, 0, defaults.Pkgdir, "EXPORT_BINPKGS"},
	cfsetup{cfKey_exportgendir, ss_value, 0, defaults.Generateddir, "EXPORT_GENERATED_FILES"},
	cfsetup{cfKey_chrootexec, ss_file, 0, defaults.ChrootExec, "CHROOT_EXEC"},
}


func defaultSettingSetup() map[int]string {
	setup := map[int]string{}
	for _, v := range settingSetup {
		setup[v.key] = v.default_value
	}
	return setup
}


func mergeSettingSetup(target map[int]string, source map[int]string) {
	for key, value := range source {
		if len(target[key]) == 0 {
			target[key] = value
		}
	}
}


func patchPaths(cfg map[int]string) error {
	for _, setup := range settingSetup {
		key := setup.key
		value := cfg[key]
		if (setup.s_type != ss_dir && setup.s_type != ss_file) || len(value) < 1 {
			continue
		}
		value = path.Clean(value)
		if !path.IsAbs(value) {
			relTo := cfg[setup.relative_to]
			if len(relTo) < 1 || !path.IsAbs(relTo) {
				return fmt.Errorf("No absolute path for setup element %s",
					setup.configKey)
			}
			value = path.Join(relTo, value)
		}
		cfg[key] = value
	}
	return nil
}


func Load(configfile string, basepath string) (*ConfigType, error) {
	setup := map[int]string{}

	if len(basepath) < 1 {
		basepath = os.Getenv("LAYERROOT")
	}
	setup[cfKey_basepath] = basepath

	if len(configfile) < 1 {
		var choices []string
		fromenv := os.Getenv("LAYERCONF")
		if len(fromenv) > 0 {
			choices = append(choices, fromenv)
		}
		home := os.Getenv("HOME")
		if len(home) > 0 {
			choices = append(choices, home + "/.layercake")
		}
		parentdir := path.Dir(path.Dir(os.Args[0]))
		if len(parentdir) > 0 {
			choices = append(choices, parentdir + "/etc/layercake.conf")
		}
		choices = append(choices, "/etc/layercake.conf")
		for _, filepath := range choices {
			if fs.IsFile(filepath) {
				configfile = filepath
				break
			}
		}
	}

	visited := map[string]bool{}
	for len(configfile) > 0 {
		if _, seen := visited[configfile]; seen {
			return nil, fmt.Errorf("Config-file loop: have seen %s", configfile)
		}
		visited[configfile] = true
		fileSetup, err := readConfigFile(configfile);
		if err != nil {
			return nil, err
		}
		mergeSettingSetup(setup, fileSetup)
		configfile = fileSetup[cfKey_configfile]
	}

	if err := patchPaths(setup); err != nil {
		return nil, err
	}

	defaultSetup := defaultSettingSetup()
	mergeSettingSetup(setup, defaultSetup)
	if err := patchPaths(setup); err != nil {
		return nil, err
	}

	cfg := &ConfigType{
		Basepath: setup[cfKey_basepath],
		Layerdirs: setup[cfKey_layerdirs],
		LayerBuildRoot: setup[cfKey_buildroot],
		LayerBinPkgdir: setup[cfKey_binpkgdir],
		LayerGeneratedir: setup[cfKey_gendir],
		LayerOvfsWorkdir: setup[cfKey_workdir],
		LayerOvfsUpperdir: setup[cfKey_upperdir],
		Exportdirs: setup[cfKey_exportroot],
		ExportBinPkgdir: setup[cfKey_exportpkgdir],
		ExportGeneratedir: setup[cfKey_exportgendir],
		ChrootExec: setup[cfKey_chrootexec],
	}
	return cfg, nil
}


func readConfigFile(filename string) (map[int]string, error) {
	cfg := map[int]string{}
	cursor, err := fs.NewTextInputFileCursor(filename)
	if nil != err {
		return cfg, err
	}
	defer cursor.Close()
	var line string
	for cursor.ReadNonBlankNonCommentLine(&line) {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, "=", 2)
		var val string
		if len(parts) > 1 {
			val = strings.TrimSpace(parts[1])
		}
		if len(val) < 1 {
			continue
		}
		uckey := strings.ToUpper(strings.TrimSpace(parts[0]))
		found := false
		for _, item := range settingSetup {
			if uckey == item.configKey {
				cfg[item.key] = val
				found = true
				break
			}
		}
		if !found {
			return cfg, fmt.Errorf("Unrecognized setting '%s'", parts[0])
		}
	}
	err = cursor.Err()
	if nil != err {
		return nil, err
	}
	return cfg, nil
}


func (cfg *ConfigType) CheckConfigPaths() (missing []string, haveNonBasePaths bool) {
	dirsToCheck := []string{cfg.Layerdirs, cfg.Exportdirs}
	missing = make([]string, 0, len(dirsToCheck) + 1)
	if !fs.IsDir(cfg.Basepath) {
		missing = append(missing, cfg.Basepath)
	}
	for _, name := range dirsToCheck {
		if !fs.IsDir(name) {
			missing = append(missing, name)
		} else {
			isDescendant, err := fs.IsDescendant(cfg.Basepath, name)
			if err != nil {
				missing = append(missing, name)
			} else if !isDescendant {
				haveNonBasePaths = true
			}
		}
	}
	return
}

