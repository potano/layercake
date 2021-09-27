package manage

import (
	"fmt"
	"path"

	"potano.layercake/fs"
	"potano.layercake/config"
	"potano.layercake/defaults"
)


func FindLayers(cfg *config.ConfigType, opts *config.Opts) (*Layerdefs, error) {
	layers := &Layerdefs{
		layermap: map[string]*Layerinfo{},
		cfg: cfg,
		opts: opts,
	}
	if err := layers.readLayerFiles(); err != nil {
		return nil, err
	}
	if err := layers.checkInheritance(); err != nil {
		return nil, err
	}
	layers.normalizeOrder()
	return layers, nil
}


func (layers *Layerdefs) readLayerFiles() error {
	errlist := []string{}
	lst, err := fs.Readdirnames(layers.cfg.Layerdirs)
	if err != nil {
		return err
	}
	for _, layername := range lst {
		if !isLegalLayerName(layername) {
			continue
		}
		layerpath := layers.layerPath(layername)
		layerconfig := path.Join(layerpath, defaults.LayerconfigFile)
		if !fs.IsFile(layerconfig) {
			errlist = append(errlist, "Layer " + layername + " has no definition file")
		} else {
			layer, err := ReadLayerFile(layerconfig, false)
			if err != nil {
				errlist = append(errlist, err.Error())
			} else {
				layer.Name = layername
				layer.LayerPath = layerpath
				layers.layermap[layername] = layer
			}
		}
	}
	return nil
}


func (layers *Layerdefs) checkInheritance() error {
	for layername, layer := range layers.layermap {
		visited := map[string]bool{layername: true}
		base := layer.Base
		for len(base) > 0 {
			layer = layers.layermap[base]
			if layer == nil {
				return fmt.Errorf("Layer %s refers to non-existent base %s",
					layername, base)
			}
			layername = layer.Name
			if visited[layername] {
				return fmt.Errorf("Layer %s is in cycle of inheritance", layername)
			}
			visited[layername] = true
			base = layer.Base
		}
	}
	return nil
}


func (layers *Layerdefs) layerPath(layername string) string {
	return path.Join(layers.cfg.Layerdirs, layername)
}

func (layers *Layerdefs) layerconfigFilePath(layer *Layerinfo) string {
	return path.Join(layer.LayerPath, defaults.LayerconfigFile)
}

func (layers *Layerdefs) buildPath(layer *Layerinfo) string {
	return path.Join(layer.LayerPath, layers.cfg.LayerBuildRoot)
}

func (layers *Layerdefs) ovfsWorkPath(layer *Layerinfo) string {
	return path.Join(layer.LayerPath, layers.cfg.LayerOvfsWorkdir)
}

func (layers *Layerdefs) ovfsUpperPath(layer *Layerinfo) string {
	return path.Join(layer.LayerPath, layers.cfg.LayerOvfsUpperdir)
}

func (layers *Layerdefs) layerExportDir(layer *Layerinfo) string {
	return path.Join(layers.cfg.Exportdirs, layer.Name)
}

func (layers *Layerdefs) expandLayerExportEndpoints(layer *Layerinfo) []NeededMountType {
	out := []NeededMountType{}
	for _, mount := range layer.ConfigExports {
		source := mount.Source
		target := mount.Mount
		if !path.IsAbs(source) {
			source, ok := defaults.ExportDirEntries[target]
			if !ok {
				continue
			}
			source = path.Join(layers.cfg.Exportdirs, layer.Name, source)
		}
		if path.IsAbs(target) {
			source = path.Join(layer.LayerPath, layers.cfg.LayerBuildRoot, target)
		} else if target == defaults.Generateddir {
			target = path.Join(layer.LayerPath, target)
		} else {
			continue
		}
		mount.Source = source
		mount.Mount = target
		out = append(out, mount)
	}
	return out
}



func (layer *Layerinfo) addMessage(msg string) {
	layer.Messages = append(layer.Messages, msg)
}

func (layer *Layerinfo) addMessagef(base string, params...interface{}) {
	layer.addMessage(fmt.Sprintf(base, params...))
}



func (ld *Layerdefs) ProbeAllLayerstate(mounts fs.Mounts, inuse map[string]int) error {
	for _, layer := range ld.Layers() {
		if layer.State == Layerstate_error {
			continue
		}
		name := layer.Name
		buildroot := ld.buildPath(layer)
		if !fs.IsDir(buildroot) {
			layer.State = Layerstate_incomplete
			continue
		}
		workdir := ld.ovfsWorkPath(layer)
		upperdir := ld.ovfsUpperPath(layer)
		haveWorkdir := fs.IsDir(workdir)
		haveUpperdir := fs.IsDir(upperdir)
		if len(layer.Base) < 1 {
			if haveWorkdir || haveUpperdir {
				layer.addMessagef("Base layer %s has extraneous overlayfs dir(s)",
					name)
			}
		} else if !haveWorkdir && !haveUpperdir {
			if !haveWorkdir {
				layer.addMessagef("Layer %s lacks an overlayfs work dir", name)
			}
			if !haveUpperdir {
				layer.addMessagef("Layer %s lacks an overlayfs upper dir", name)
			}
			layer.State = Layerstate_incomplete
			continue
		}

		mask := inuse[name]
		if mask > 0 {
			layer.Busy = true
			if (mask & fs.UseMask_root) > 0 {
				layer.Chroot = true
			}
		}

		layer.State = Layerstate_complete

		ld.findLayerstate(layer, mounts)
	}
	return nil
}



func minimalBuildDirsPresent(buildroot string) bool {
	for _, name := range defaults.MinimalBuildDirs {
		if !fs.IsDir(path.Join(buildroot, name)) {
			return false
		}
	}
	return true
}


func (ld *Layerdefs) findLayerstate(layer *Layerinfo, mounts fs.Mounts) {
	builddir := ld.buildPath(layer)
	layer.Mounts = mounts.GetMountAndSubmounts(builddir)
	if layer.State < Layerstate_complete {
		return
	}
	layer.State = Layerstate_complete
	layer.Messages = []string{}

	base := layer.Base
	numMounted := 0

	if len(base) > 0 {
		// Derived layer:  is correct overlayfs mount in place?
		baseLayer := ld.layermap[base]
		if baseLayer.State < Layerstate_mountable {
			return
		}
		mnt := layer.Mounts.GetMount(builddir)
		if nil == mnt {
			return
		}
		if mnt.Fstype != "overlay" {
			layer.addMessage("mounted but not as overlayfs")
			layer.State = Layerstate_error
			return
		}
		ovfsError := false
		if mnt.Source != ld.buildPath(baseLayer) {
			layer.addMessage("wrong parent directory mounted")
			ovfsError = true
		}
		if mnt.Source2 != ld.ovfsUpperPath(layer) {
			layer.addMessage("wrong upper directory mounted")
			ovfsError = true
		}
		if mnt.Workdir != ld.ovfsWorkPath(layer) {
			layer.addMessage("wrong work directory mounted")
			ovfsError = true
		}
		if ovfsError {
			layer.State = Layerstate_error
			return
		}
		numMounted++
	}
	if !minimalBuildDirsPresent(builddir) {
		return
	}
	layer.State = Layerstate_inhabited

	missingMountpoints := []string{}
	incorrectMounts := []NeededMountType{}
	fsErrors := []string{}

	for _, pair := range layer.ConfigMounts {
		target := path.Join(builddir, pair.Mount)
		if !fs.Exists(target) {
			missingMountpoints = append(missingMountpoints, pair.Mount)
			continue
		}
		mnt := layer.Mounts.GetMount(target)
		if mnt == nil {
			continue
		}
		numMounted++
		if mnt.Source != pair.Source {
			incorrectMounts = append(incorrectMounts, pair)
		}
	}
	// Note that we don't count missing export symlinks to be errors.  Mounting creates them
	for _, pair := range ld.expandLayerExportEndpoints(layer) {
		isDescendant, err := fs.IsDescendant(builddir, pair.Mount)
		if err != nil {
			fsErrors = append(fsErrors, err.Error())
			continue
		}
		if !isDescendant {
			incorrectMounts = append(incorrectMounts, pair)
			continue
		}
		if !fs.Exists(pair.Mount) {
			missingMountpoints = append(missingMountpoints, pair.Mount)
			continue
		}
		if fs.IsSymlink(pair.Source) {
			linktarg, err := fs.Readlink(pair.Source)
			if err != nil {
				fsErrors = append(fsErrors, err.Error())
				continue
			}
			if pair.Mount != linktarg {
				incorrectMounts = append(incorrectMounts, pair)
			}
		}
	}

	for _, msg := range fsErrors {
		layer.addMessage("error probing symlink: " + msg)
	}
	for _, pair := range incorrectMounts {
		layer.addMessagef("%s has wrong mount source %s", pair.Mount, pair.Source)
	}
	for _, msg := range missingMountpoints {
		layer.addMessage("missing mountpoint: " + msg)
	}

	if len(incorrectMounts) > 0 || len(fsErrors) > 0 {
		layer.State = Layerstate_error
		return
	}

	if len(missingMountpoints) > 0 {
		return
	}

	if numMounted == 0 {
		layer.State = Layerstate_mountable
	} else if numMounted < len(layer.ConfigMounts) {
		layer.State = Layerstate_partialmount
	} else {
		layer.State = Layerstate_mounted
	}
}

