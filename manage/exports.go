package manage

import (
	"fmt"
	"path"
	"strings"

	"potano.layercake/fs"
	"potano.layercake/defaults"
)


type expandedNeededMountType struct {
	Mount, Source, Fstype string
	UnexpandedMount, UnexpandedSource string
}


func (ld *Layerdefs) expandConfigExports(layer *Layerinfo) ([]expandedNeededMountType, error) {
	out := make([]expandedNeededMountType, len(layer.ConfigExports))
	unknowns := make([]string, 0, len(layer.ConfigExports))
	for i, mount := range layer.ConfigExports {
		source := mount.Source
		target := mount.Mount
		if !path.IsAbs(target) {
			dirname, ok := ld.cfg.LayerExportDirs[target]
			if !ok {
				unknowns = append(unknowns, target)
			} else {
				target = path.Join(ld.cfg.Exportdirs, dirname, layer.Name)
			}
		}
		source = path.Join(layer.LayerPath, ld.cfg.LayerBuildRoot, source)
		mount.Source = source
		mount.Mount = target
		out[i] = expandedNeededMountType{
			Mount: target,
			Source: source,
			Fstype: mount.Fstype,
			UnexpandedMount: mount.Mount,
			UnexpandedSource: mount.Source}
	}
	if len(unknowns) > 0 {
		return out, fmt.Errorf("unknown export-directory key(s) %s",
			strings.Join(unknowns, ", "))
	}
	return out, nil
}


func (ld *Layerdefs) expandConfigMounts(layer *Layerinfo) ([]expandedNeededMountType, error) {
	callback := func (symbol, tail string) (string, error) {
		switch symbol {
		case "base":
			return ld.findLayerBase(layer).LayerPath, nil
		}
		return "", fmt.Errorf("unknown key %s", symbol)
	}
	out := make([]expandedNeededMountType, len(layer.ConfigMounts))
	for i, mount := range layer.ConfigMounts {
		mountpoint := path.Join(ld.buildPath(layer), mount.Mount)
		source, err := fs.AdjustPrefixedPath(mount.Source, "", callback)
		if err != nil {
			return nil, err
		}
		out[i] = expandedNeededMountType{
			Mount: mountpoint,
			Source: source,
			Fstype: mount.Fstype,
			UnexpandedMount: mount.Mount,
			UnexpandedSource: mount.Source}
	}
	return out, nil
}


func makeSymlinkInDirectory(source, target string) error {
	if !fs.IsSymlink(target) {
		targetDir := path.Dir(target)
		if !fs.IsDir(targetDir) {
			err := fs.Mkdir(targetDir)
			if err != nil {
				return err
			}
		}
		if err := fs.Symlink(target, source); err != nil {
			return err
		}
	}
	return nil
}


func (ld *Layerdefs) makeExportSymlinks(layer *Layerinfo) error {
	exports, err := ld.expandConfigExports(layer)
	if err != nil {
		return err
	}
	for _, pair := range exports {
		err = makeSymlinkInDirectory(pair.Source, pair.Mount)
		if err != nil {
			return err
		}
	}
	source := path.Join(layer.LayerPath, defaults.Generateddir)
	if fs.Exists(source) {
		err = makeSymlinkInDirectory(source, path.Join(ld.cfg.Exportdirs,
			ld.cfg.LayerExportDirs[defaults.Generateddir], layer.Name))
		if err != nil {
			return err
		}
	}
	return nil
}


func (ld *Layerdefs) removeLayerExportLinks(li *Layerinfo) error {
	exportdir := ld.cfg.Exportdirs
	for _, dirbase := range ld.cfg.LayerExportDirs {
		linkname := path.Join(exportdir, dirbase, li.Name)
		if !fs.Exists(linkname) {
			continue
		}
		if !fs.IsSymlink(linkname) {
			return fmt.Errorf("Export %s is not a symlink; cannot remove", linkname)
		}
		fs.Remove(linkname)
	}
	return nil
}

