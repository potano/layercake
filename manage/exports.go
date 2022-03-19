// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package manage

import (
	"fmt"
	"path"

	"potano.layercake/fs"
)


type expandedNeededMountType struct {
	Mount, Source, Fstype string
	UnexpandedMount, UnexpandedSource string
}


func (ld *Layerdefs) expandConfigExports(layer *Layerinfo) ([]expandedNeededMountType, error) {
	callback := func (symbol, tail string) (string, error) {
		switch symbol {
		case "package_export":
			return path.Join(ld.cfg.Exportdirs, ld.cfg.ExportBinPkgdir, layer.Name), nil
		case "file_export":
			return path.Join(ld.cfg.Exportdirs, ld.cfg.ExportGeneratedir, layer.Name),
				nil
		}
		return "", fmt.Errorf("unknown key %s", symbol)
	}
	out := make([]expandedNeededMountType, len(layer.ConfigExports))
	for i, mount := range layer.ConfigExports {
		source := path.Join(layer.LayerPath, ld.cfg.LayerBuildRoot, mount.Source)
		target, err := fs.AdjustPrefixedPath(mount.Mount, "", callback)
		if err != nil {
			return nil, err
		}
		out[i] = expandedNeededMountType{
			Mount: target,
			Source: source,
			Fstype: mount.Fstype,
			UnexpandedMount: mount.Mount,
			UnexpandedSource: mount.Source}
	}
	return out, nil
}


func (ld *Layerdefs) expandConfigMounts(layer *Layerinfo) ([]expandedNeededMountType, error) {
	callback := func (symbol, tail string) (string, error) {
		switch symbol {
		case "base":
			return ld.findLayerBase(layer).LayerPath, nil
		case "self":
			return layer.LayerPath, nil
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


func (ld *Layerdefs) automatedLayerExportPaths(layer *Layerinfo) []NeededMountType {
	binpkgMount := path.Join(ld.cfg.Exportdirs, ld.cfg.ExportBinPkgdir, layer.Name)
	gendirMount := path.Join(ld.cfg.Exportdirs, ld.cfg.ExportGeneratedir, layer.Name)
	binpkgSource := path.Join(layer.LayerPath, ld.cfg.LayerBinPkgdir)
	gendirSource := path.Join(layer.LayerPath, ld.cfg.LayerGeneratedir)
	return []NeededMountType{
		{binpkgMount, binpkgSource, "symlink"},
		{gendirMount, gendirSource, "symlink"},
	}
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
	for _, item := range ld.automatedLayerExportPaths(layer) {
		if fs.Exists(item.Source) {
			err = makeSymlinkInDirectory(item.Source, item.Mount)
			if err != nil {
				return err
			}
		}
	}
	return nil
}


func (ld *Layerdefs) removeLayerExportLinks(layer *Layerinfo) error {
	for _, item := range ld.automatedLayerExportPaths(layer) {
		if !fs.Exists(item.Mount) {
			continue
		}
		if !fs.IsSymlink(item.Mount) {
			return fmt.Errorf("Export %s is not a symlink; cannot remove", item.Mount)
		}
		fs.Remove(item.Mount)
	}
	return nil
}

