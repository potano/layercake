package manage

import (
	"fmt"
	"path"
	"errors"
	"strings"

	"potano.layercake/fs"
	"potano.layercake/fns"
	"potano.layercake/config"
	"potano.layercake/defaults"
)

type baseFileSetup struct {pathname, contents, desc string; required bool}
type baseFiles struct {
	have []baseFileSetup
	need []baseFileSetup
}

func newBaseFiles(cfg *config.ConfigType) *baseFiles {
	bf := &baseFiles{}

	skeleton := fns.Template(defaults.SkeletonLayerconfig,
		map[string]string{"pkgdir": cfg.LayerBinPkgdir})
	bf.addBaseFile(cfg.Basepath, defaults.SkeletonLayerconfigFile, skeleton,
		"default layer configuration", true)
	bf.addBaseFile(cfg.Exportdirs, defaults.ExportIndexHtmlName, defaults.ExportIndexHtml,
		"export-directory file", false)
	return bf
}

func (bf *baseFiles) addBaseFile(basepath, filename, contents, desc string, required bool) {
	pth := path.Join(basepath, filename)
	bfs := baseFileSetup{pth, contents, desc, required}
	if fs.IsFile(pth) {
		bf.have = append(bf.have, bfs)
	} else {
		bf.need = append(bf.need, bfs)
	}
}


func InitLayercakeBase(cfg *config.ConfigType) error {
	missing, haveNonBasePaths := cfg.CheckConfigPaths()
	if len(missing) > 0 && haveNonBasePaths {
		return errors.New("LAYERS or EXPORT_DIRS has absolute path: need manual setup")
	}
	for _, pth := range missing {
		err := fs.Mkdir(pth)
		if err != nil {
			return fmt.Errorf("%s creating directory %s", err, pth)
		}
	}
	bf := newBaseFiles(cfg)
	for _, bfs := range bf.need {
		err := fs.WriteTextFile(bfs.pathname, bfs.contents)
		if err != nil {
			return fmt.Errorf("%s setting up %s %s", err, bfs.desc, bfs.pathname)
		}
	}
	if len(bf.have) > 0 {
		var existing []string
		for _, bfs := range bf.have {
			existing = append(existing, bfs.desc + " " + bfs.pathname)
		}
		return fmt.Errorf("Will not overwrite %s; delete manually to create default files",
			strings.Join(existing, " or "))
	}
	if len(missing) == 0 && len(bf.need) == 0 {
		return errors.New("Base directories already set up:  nothing to do")
	}
	return nil
}


func CheckBaseSetUp(cfg *config.ConfigType) []string {
	var mia []string
	missing, _ := cfg.CheckConfigPaths()
	for _, name := range missing {
		mia = append(mia, "base directory " + name)
	}
	bf := newBaseFiles(cfg)
	for _, bfs := range bf.need {
		if bfs.required {
			mia = append(mia, bfs.desc + " " + bfs.pathname)
		}
	}
	return mia
}

