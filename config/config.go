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
	LayerOvfsWorkdir string
	LayerOvfsUpperdir string
	Exportdirs string
	LayerExportDirs map[string]string
	ChrootExec string
}


const (
	ss_value = iota
	ss_file
	ss_dir
)


type cfsetup struct {
	name string
	s_type int
	relative_to string
	default_value string
}


var settingSetup = []cfsetup {
	cfsetup{"basepath", ss_dir, "", defaults.BasePath},
	cfsetup{"configfile", ss_file, "basepath", defaults.MainConfigFile},
	cfsetup{"layerdirs", ss_dir, "basepath", defaults.Layerdirs},
	cfsetup{"buildroot", ss_value, "", defaults.Builddir},
	cfsetup{"workdir", ss_value, "", defaults.Workdir},
	cfsetup{"upperdir", ss_value, "", defaults.Upperdir},
	cfsetup{"exportroot", ss_dir, "basepath", defaults.Exportdirs},
	cfsetup{"chrootexec", ss_file, "", defaults.ChrootExec},
}


func defaultSettingSetup() map[string]string {
	setup := map[string]string{}
	for _, v := range settingSetup {
		setup[v.name] = v.default_value
	}
	for key, value := range defaults.ExportDirEntries {
		setup["export/" + key] = value
	}
	return setup
}


func mergeSettingSetup(target map[string]string, source map[string]string) {
	for key, value := range source {
		if len(target[key]) == 0 {
			target[key] = value
		}
	}
}


func patchPaths(cfg map[string]string) error {
	for _, setup := range settingSetup {
		key := setup.name
		value := cfg[key]
		if (setup.s_type != ss_dir && setup.s_type != ss_file) || len(value) < 1 {
			continue
		}
		value = path.Clean(value)
		if !path.IsAbs(value) {
			relTo := cfg[setup.relative_to]
			if len(relTo) < 1 || !path.IsAbs(relTo) {
				return fmt.Errorf("No absolute path for setup element %s", key)
			}
			value = path.Join(relTo, value)
		}
		cfg[key] = value
	}
	return nil
}


func Load(configfile string, basepath string) (*ConfigType, error) {
	setup := map[string]string{}

	if len(basepath) < 1 {
		basepath = os.Getenv("LAYERROOT")
	}
	setup["basepath"] = basepath

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
		configfile = fileSetup["configfile"]
	}

	if err := patchPaths(setup); err != nil {
		return nil, err
	}

	defaultSetup := defaultSettingSetup()
	mergeSettingSetup(setup, defaultSetup)
	if err := patchPaths(setup); err != nil {
		return nil, err
	}

	exportEntries := map[string]string{}
	for key, value := range setup {
		if key[0:7] == "export/" {
			exportEntries[key[7:]] = value
		}
	}

	cfg := &ConfigType{
		Basepath: setup["basepath"],
		Layerdirs: setup["layerdirs"],
		LayerBuildRoot: setup["buildroot"],
		LayerOvfsWorkdir: setup["workdir"],
		LayerOvfsUpperdir: setup["upperdir"],
		Exportdirs: setup["exportroot"],
		LayerExportDirs: exportEntries,
		ChrootExec: setup["chrootexec"],
	}
	return cfg, nil
}


func readConfigFile(filename string) (map[string]string, error) {
	cfg := map[string]string{}
	cursor, err := fs.NewTextInputFileCursor(filename)
	if nil != err {
		return cfg, err
	}
	defer cursor.Close()
	for cursor.Scan() {
		line := cursor.Text()
		if len(line) < 1 || line[0] == '#' || (len(line) > 1 && "//" == line[:2]) {
			continue
		}
		line = strings.Trim(line, " \t\r\n")
		parts := strings.SplitN(line, "=", 2)
		var val string
		if len(parts) > 1 {
			val = strings.Trim(parts[1], " \t")
		}
		if len(val) < 1 {
			continue
		}
		var key string
		switch strings.ToUpper(strings.Trim(parts[0], " \t")) {
		case "BASEPATH":
			key = "basepath"
		case "CONFIGFILE", "CONFIG_FILE":
			key = "configfile"
		case "LAYERS":
			key = "layerdirs"
		case "BUILDROOT":
			key = "buildroot"
		case "OVERFS_WORKDIR", "WORKDIR":
			key = "workdir"
		case "OVERFS_UPPERDIR", "UPPERDIR":
			key = "upperdir"
		case "EXPORT_DIR", "EXPORT_ROOT", "EXPORTDIR", "EXPORTROOT":
			key = "exportroot"
		case "CHROOTEXEC", "CHROOT_EXEC":
			key = "chrootexec"
		default:
			return cfg, fmt.Errorf("Unrecognized setting '%s'", parts[0])
		}
		cfg[key] = val
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

