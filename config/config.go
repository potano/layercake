/* Processing for layercake's configuration values */

package config

import (
	"os"
	"fmt"
	"path"
	"bufio"
	"strings"
	"path/filepath"

	"layercake/fs"
)


const (
	default_layersfile = "layers"
	default_build_root = "build"
	default_overlayfs_workroot = "overlay_workdirs"
	default_overlayfs_upperdirs = "overlay_filedirs"
	default_html_root = "htdocs"
	default_symlink_target = "/var/lib/packages"
	default_base_bind_mount = "/usr/portage /usr/portage"
)


const (
	ss_value = iota
	ss_file
	ss_dir
	ss_layerdir
)

type settingsetup struct {
	name string
	s_type int
	desc string
}

var settingSetup = []settingsetup {
	settingsetup{"basepath", ss_dir, "base directory for layercake system"},
	settingsetup{"layersfile", ss_file, "name of file containing layer definitions"},
	settingsetup{"buildroot", ss_layerdir, "directory containing build roots for each layer"},
	settingsetup{"workroot", ss_layerdir, "directory containing overlayfs work directories"},
	settingsetup{"upperroot", ss_layerdir, "directory containing overlayfs uppper directories"},
	settingsetup{"htmlroot", ss_layerdir, "directory containing symlinks to packages"},
	settingsetup{"htmllink", ss_value, "target of HTML symlink in build directory"},
	settingsetup{"bindmount", ss_value, "bind mount into base-level layers"},
	settingsetup{"chrootexec", ss_value, "path of chroot executable"},
}

type ConfigType struct {
	cfg map[string]string
	paths map[string]string
	haveAbspath bool
}

func NewDefaultConfig() *ConfigType {
	return &ConfigType{
		cfg: map[string]string{
			"layersfile": default_layersfile,
			"buildroot": default_build_root,
			"workroot": default_overlayfs_workroot,
			"upperroot": default_overlayfs_upperdirs,
			"htmlroot": default_html_root,
			"htmllink": default_symlink_target,
			"bindmount": default_base_bind_mount,
			"chrootexec": "",
		},
		paths: map[string]string{},
	}
}

func (cfg *ConfigType) ReadConfigFile(filename string) error {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0666)
	if nil != err {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	lineno := 0
	for scanner.Scan() {
		lineno++
		line := scanner.Text()
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
		switch strings.Trim(parts[0], " \t") {
		case "BASEPATH":
			key = "basepath"
		case "LAYERSFILE", "LAYERS":
			key = "layersfile"
		case "BUILDROOT":
			key = "buildroot"
		case "OVERFS_WORKROOT", "WORKROOT", "WORKDIRS":
			key = "workroot"
		case "OVERFS_UPPERDIRS", "UPPERROOT", "UPPERDIRS":
			key = "upperroot"
		case "HTML_DIR", "HTMLROOT", "HTMLDIRS":
			key = "htmlroot"
		case "HTML_SYMLINK_TARGET":
			key = "htmllink"
		case "BASE_BIND_MOUNT":
			key = "bindmount"
		case "CHROOT_EXEC":
			key = "chrootexec"
		default:
			return fmt.Errorf("Unrecognized setting '%s' in %s at line %d", parts[0],
				filename, lineno)
		}
		cfg.cfg[key] = val
	}
	err = scanner.Err()
	if nil != err {
		return fmt.Errorf("%s reading %s line %s", err, filename, lineno)
	}
	return nil
}

func (cfg *ConfigType) Set(key, value string) {
	cfg.cfg[key] = value
}

func (cfg *ConfigType) GetCfg(key string) string {
	return cfg.cfg[key]
}

func (cfg *ConfigType) GetPath(key string) string {
	return cfg.paths[key]
}

func (cfg *ConfigType) GetDirPaths() []string {
	out := make([]string, 0, 4)
	for _, item := range settingSetup {
		if item.s_type == ss_layerdir || item.s_type == ss_dir {
			out = append(out, cfg.paths[item.name])
		}
	}
	return out
}

func (cfg *ConfigType) CheckConfigPaths() (missing []string, haveAbsPath bool) {
	basepath := cfg.cfg["basepath"]
	for _, item := range settingSetup {
		if ss_value == item.s_type {
			continue
		}
		name := item.name
		val := cfg.cfg[name]
		if len(val) < 0 {
			missing = append(missing, name)
			continue
		}
		if path.IsAbs(val) {
			if ss_dir != item.s_type {
				haveAbsPath = true
			}
		} else {
			val = filepath.Clean(basepath + "/" + val)
		}
		if !fs.IsFileOrDir(val, ss_file == item.s_type) {
			missing = append(missing, name)
		}
		cfg.paths[name] = val
	}
	cfg.haveAbspath = haveAbsPath
	return
}

