package manage

import (
	"os"
	"fmt"
	"path"
	"errors"
	"strings"
	"unicode"
	"path/filepath"

	"potano.layercake/fs"
	"potano.layercake/config"
	"potano.layercake/defaults"
)

type Layerdefs struct {
	layermap map[string]*Layerinfo
	normalizedOrder []string
	baselayers []string
	cfg *config.ConfigType
	opts *config.Opts
	mounts fs.Mounts
}

const (
	Layerstate_empty = iota
	Layerstate_error
	Layerstate_incomplete
	Layerstate_complete
	Layerstate_inhabited
	Layerstate_mountable
	Layerstate_partialmount
	Layerstate_mounted
)

var layerstateDescriptions []string = []string{
	"defined but empty",
	"error",
	"incomplete setup",
	"not yet populated",
	"build directories set up",
	"mountable",
	"partially mounted",
	"mounted and ready",
}


type NeededMountType struct {
	Mount, Source, Fstype string
}


type Layerinfo struct {
	Name, Base string
	ConfigMounts []NeededMountType
	ConfigExports []NeededMountType
	LayerPath string
	State int
	Messages []string
	Busy, Chroot bool
	Mounts []*fs.MountType
}



func (ld *Layerdefs) Layers() []*Layerinfo {
	layers := make([]*Layerinfo, len(ld.normalizedOrder))
	for i, name := range ld.normalizedOrder {
		layers[i] = ld.layermap[name]
	}
	return layers
}


func (ld *Layerdefs) Layer(name string) *Layerinfo {
	return ld.layermap[name]
}


func (ld *Layerdefs) DescribeState(li *Layerinfo, detailed bool) []string {
	out := []string{layerstateDescriptions[li.State]}
	if detailed {
		out = append(out, li.Messages...)
		mnts := ld.describeMounts(li, "  ")
		if len(mnts) > 0 {
			out = append(out, "Current mounts:")
			out = append(out, mnts...)
		}
		if li.Busy && !li.Chroot {
			out = append(out, "layer is in use")
		}
	}
	return out
}


func (ld *Layerdefs) describeMounts(li *Layerinfo, leftpad string) (out []string) {
	if len(li.Mounts) == 0 {
		return
	}
	mounts := map[string]*fs.MountType{}
	for _, mnt := range li.Mounts {
		mounts[mnt.Mountpoint] = mnt
	}
	buildpath := ld.buildPath(li)
	base := []string{}
	for _, cm := range li.ConfigMounts {
		mt := path.Join(buildpath, cm.Mount)
		if _, exists := mounts[mt]; exists {
			base = append(base, cm.Mount)
			delete(mounts, mt)
		}
	}
	if len(base) > 0 {
		out = append(out, leftpad + "required: " + strings.Join(base, ", "))
	}
	if _, exists := mounts[buildpath]; exists {
		out = append(out, leftpad + "overlayfs")
		delete(mounts, buildpath)
	}
	for _, mnt := range mounts {
		out = append(out, leftpad + mnt.Mountpoint)
	}
	return
}


func (ld *Layerdefs) getDefaultLayerinfo(filename string) (*Layerinfo, error) {
	if len(filename) == 0 {
		filename = path.Join(ld.cfg.Basepath, defaults.SkeletonLayerconfigFile)
	}
	origFilename := filename
	if !fs.IsFile(filename) {
		filename = path.Join(ld.cfg.Layerdirs, filename)
		if !fs.IsFile(filename) && len(filepath.Ext(filename)) == 0 {
			filename = filename + defaults.SkeletonLayerconfigFileExt
		}
	}
	if !fs.IsFile(filename) {
		return nil, fmt.Errorf("Cannot locate layer-configuration file %s",
			origFilename)
	}
	return ReadLayerFile(filename, true)
}


func (ld *Layerdefs) AddLayer(name, base, configFile string) error {
	err := ld.testName(nametest{name, name_free, "Layer"},
		nametest{base, name_optional | name_need, "Parent layer"})
	if nil != err {
		return err
	}
	var layer *Layerinfo
	if len(base) > 0 {
		if base == name {
			return errors.New("Layer cannot be its own base")
		}
		layer = ld.layermap[base]
	}
	if len(configFile) > 0 || len(base) == 0 {
		layer, err = ld.getDefaultLayerinfo(configFile)
		if err != nil {
			return err
		}
	}
	if layer == nil {
		return fmt.Errorf("Specify a layer-config file")
	}
	layer.Name = name
	layer.Base = base
	layer.LayerPath = ld.layerPath(name)
	layer.Mounts = []*fs.MountType{}

	err = fs.Mkdir(layer.LayerPath)
	if err != nil {
		return err
	}
	err = ld.writeLayerFile(layer)
	if err != nil {
		return err
	}
	err = fs.Mkdir(ld.buildPath(layer))
	if err != nil {
		return err
	}
	if len(base) > 0 {
		err = fs.Mkdir(ld.ovfsWorkPath(layer))
		if err != nil {
			return err
		}
		err = fs.Mkdir(ld.ovfsUpperPath(layer))
		if err != nil {
			return err
		}
	}
	ld.normalizeOrder()
	return nil
}


func (layer *Layerinfo) errorIfError() error {
	if layer.State == Layerstate_error {
		return fmt.Errorf("Layer %s is in error state", layer.Name)
	}
	return nil
}


func (layer *Layerinfo) errorIfBusy(operation string) error {
	if layer.Busy || len(layer.Mounts) > 0 {
		var msg string
		if len(layer.Mounts) > 0 {
			msg = "is mounted"
		}
		if layer.Busy {
			if len(msg) > 0 {
				msg += " and "
			}
			msg += "has active users"
		}
		return fmt.Errorf("Cannot %s layer %s because it %s", operation, layer.Name, msg)
	}
	return nil
}


func (ld *Layerdefs) errorIfParent(layer *Layerinfo) error {
	name := layer.Name
	for _, layer = range ld.layermap {
		if name == layer.Base {
			return fmt.Errorf("Layer %s has at least one child layer", name)
		}
	}
	return nil
}


func (ld *Layerdefs) RemoveLayer(name string, removeFiles bool) error {
	err := ld.testName(nametest{name, name_need, "Layer"})
	if nil != err {
		return err
	}
	layer := ld.layermap[name]
	err = layer.errorIfError()
	if err != nil {
		return err
	}
	err = ld.errorIfParent(layer)
	if err != nil {
		return err
	}
	err = layer.errorIfBusy("remove")
	if err != nil {
		return err
	}

	err = ld.removeLayerExportLinks(layer)
	if err != nil {
		return err
	}

	if removeFiles {
		err = fs.Remove(layer.LayerPath)
		if err != nil {
			return err
		}
	} else {
		newname := layer.LayerPath + defaults.RemovedLayerSuffix
		if fs.Exists(newname) {
			return fmt.Errorf("Cannot pseudo-delete layer %s", name)
		}
		err = fs.Rename(layer.LayerPath, newname)
	}

	delete(ld.layermap, name)
	ld.normalizeOrder()
	return nil
}


func (ld *Layerdefs) RenameLayer(oldname, newname string) error {
	err := ld.testName(nametest{oldname, name_need, "Layer"},
		nametest{newname, name_free, "New name"})
	if nil != err {
		return err
	}
	layer := ld.layermap[oldname]
	err = layer.errorIfError()
	if err != nil {
		return err
	}
	err = layer.errorIfBusy("remove")
	if err != nil {
		return err
	}

	children := []*Layerinfo{}
	for _, child := range ld.layermap {
		if child.Base != oldname {
			continue
		}
		err = child.errorIfBusy("rename parent")
		if err != nil {
			return err
		}
		children = append(children, child)
	}

	err = ld.removeLayerExportLinks(layer)
	if err != nil {
		return err
	}

	newLayerPath := ld.layerPath(newname)
	err = fs.Rename(layer.LayerPath, newLayerPath)
	if err != nil {
		return err
	}

	for _, child := range children {
		child.Base = newname
		err = ld.writeLayerFile(child)
		if err != nil {
			return nil
		}
	}

	layer.Name = newname
	layer.LayerPath = newLayerPath
	ld.normalizeOrder()
	err = ld.writeLayerFile(layer)
	if err != nil {
		return err
	}
	return nil
}


func (ld *Layerdefs) RebaseLayer(name, newbase string) error {
	err := ld.testName(nametest{name, name_need, "Layer"},
		nametest{newbase, name_need | name_optional, "Parent layer"})
	if nil != err {
		return err
	}
	layer := ld.layermap[name]
	err = layer.errorIfError()
	if err != nil {
		return err
	}
	err = layer.errorIfBusy("rebase")
	if err != nil {
		return err
	}
	oldbase := layer.Base
	err = ld.checkInheritance()
	if err != nil {
		layer.Base = oldbase
		return errors.New("Rebasing would orphan one or more layers")
	}

	for _, child := range ld.layermap {
		if child.Base != name {
			continue
		}
		err = child.errorIfBusy("rebase parent")
		if err != nil {
			return err
		}
	}

	layer.Base = newbase
	ld.normalizeOrder()
	err = ld.writeLayerFile(layer)
	if err != nil {
		return err
	}
	return nil
}


func (ld *Layerdefs) Shell(name string) error {
	err := ld.testName(nametest{name, name_need, "Layer"})
	if nil != err {
		return err
	}
	layer := ld.layermap[name]
	builddir := ld.buildPath(layer)
	if !fs.IsDir(builddir) {
		return fmt.Errorf("Build directory for layer %s does not exist", name)
	}
	fs.Println("Exit to return to layercake.\nCaution: this is *not* a chroot.")
	return fs.Shell(builddir)
}


func (ld *Layerdefs) Makedirs(name string) error {
	err := ld.testName(nametest{name, name_need, "Layer"})
	if nil != err {
		return err
	}
	layer := ld.layermap[name]
	err = layer.errorIfError()
	if err != nil {
		return err
	}
	if layer.State < Layerstate_complete {
		needDirs := []string{}
		dir := ld.buildPath(layer)
		if !fs.IsDir(dir) {
			needDirs = append(needDirs, dir)
		}
		if len(layer.Base) > 0 {
			dir = ld.ovfsWorkPath(layer)
			if !fs.IsDir(dir) {
				needDirs = append(needDirs, dir)
			}
			dir = ld.ovfsUpperPath(layer)
			if !fs.IsDir(dir) {
				needDirs = append(needDirs, dir)
			}
		}
		for _, dir = range needDirs {
			err = fs.Mkdir(dir)
			if err != nil {
				return err
			}
		}
		ld.findLayerstate(layer)
	}

	err = ld.makeExportSymlinks(layer)
	if err != nil {
		return err
	}
	return nil
}


func (ld *Layerdefs) Mount(name string) error {
	err := ld.testName(nametest{name, name_need, "Layer"})
	if nil != err {
		return err
	}
	layer := ld.layermap[name]
	err = layer.errorIfError()
	if err != nil {
		return err
	}
	ancestors, err := ld.getAncestorsAndSelf(name)
	if nil != err {
		return err
	}
	for _, layer := range ancestors {
		err = ld.Makedirs(layer.Name)
		if nil != err {
			return err
		}
	}
	for _, layer := range ancestors {
		err = ld.mountOne(layer)
		if err != nil {
			return err
		}
	}
	return nil
}


func (ld *Layerdefs) mountOne(layer *Layerinfo) error {
	name := layer.Name
	if layer.State < Layerstate_mountable {
		return fmt.Errorf("Layer %s is not yet mountable", name)
	}
	builddir := ld.buildPath(layer)
	if len(layer.Base) > 0 {
		if nil == ld.mounts.GetMount(builddir) {
			basedir := ld.buildPath(ld.layermap[layer.Base])
			workdir := ld.ovfsWorkPath(layer)
			upperdir := ld.ovfsUpperPath(layer)
			err := fs.Mount("overlay", builddir, "overlay",
				"lowerdir=" + basedir + ",upperdir=" + upperdir +
				",workdir=" + workdir)
			if nil != err {
				return err
			}
		}
	}
	for _, m := range layer.ConfigMounts {
		mountpoint := path.Join(builddir, m.Mount)
		if nil == ld.mounts.GetMount(mountpoint) {
			err := fs.Mount(m.Source, mountpoint, m.Fstype, "")
			if nil != err {
				return err
			}
		}
	}
	err := ld.refreshMountInfo()
	if err != nil {
		return err
	}
	ld.findLayerstate(layer)
	return nil
}


const (
	Unmount_status_ok = iota
	Unmount_status_was_not_mounted
	Unmount_status_busy
	Unmount_status_error
)


func (ld *Layerdefs) Unmount(name string, unmountAll bool) error {
	if len(name) > 0 && unmountAll {
		fmt.Errorf("Cannot specify unmount of a specific layer and also all layers")
	}
	if len(name) > 0 {
		err := ld.testName(nametest{name, name_need, "Layer"})
		if nil != err {
			return err
		}
		_, err = ld.unmountLayer(name)
		return err
	}
	if !unmountAll {
		fmt.Errorf("Must specify a layer to unmount or -all switch")
	}
	busyLayers := make([]string, 0, len(ld.normalizedOrder))
	for i := len(ld.normalizedOrder) - 1; i >= 0; i-- {
		name = ld.normalizedOrder[i]
		status, err := ld.unmountLayer(name)
		switch status {
		case Unmount_status_ok:
			if ld.opts.Verbose {
				fs.Printf("Unmounted layer %s", name)
			}
		case Unmount_status_was_not_mounted:
			if ld.opts.Verbose {
				fs.Printf("Layer %s was not mounted", name)
			}
		case Unmount_status_busy:
			busyLayers = append(busyLayers, name)
		case Unmount_status_error:
			return err
		}
	}
	if len(busyLayers) > 0 {
		return fmt.Errorf("Could not unmount busy layer(s): %s",
			strings.Join(busyLayers, ", "))
	}
	return nil
}


func (ld *Layerdefs) unmountLayer(name string) (int, error) {
	layer := ld.layermap[name]
	err := layer.errorIfBusy("unmount")
	if err != nil {
		return Unmount_status_busy, err
	}
	if len(layer.Mounts) == 0 {
		return Unmount_status_was_not_mounted,
			fmt.Errorf("Layer %s was not mounted", name)
	}
	for uX := len(layer.Mounts) - 1; uX >= 0; uX-- {
		path := layer.Mounts[uX].Mountpoint
		err := fs.Unmount(path, ld.opts.Force)
		if nil != err {
			layer.State = Layerstate_error
			return Unmount_status_error, err
		}
	}
	err = ld.refreshMountInfo()
	if err != nil {
		return Unmount_status_error, err
	}
	ld.findLayerstate(layer)
	return Unmount_status_ok, nil
}


func (ld *Layerdefs) Chroot(name string) error {
	err := ld.testName(nametest{name, name_need, "Layer"})
	if nil != err {
		return err
	}
	layer := ld.layermap[name]
	if layer.State < Layerstate_mounted {
		if err = ld.Mount(name); nil != err {
			return err
		}
	}
	builddir := ld.buildPath(layer)
	if !fs.IsDir(builddir) {
		return fmt.Errorf("Build directory for layer %s does not exist", name)
	}
	env := []string{"LAYERCAKE_LAYER=" + name}
	fds := []*os.File{}
	return fs.Chroot(builddir, ld.cfg.ChrootExec, env, fds)
}


func (ld *Layerdefs) Shake() error {
	for _, layer := range ld.Layers() {
		target := ld.buildPath(layer)
		if len(layer.Base) > 0 && layer.State >= Layerstate_mounted {
			err := fs.Mount("", target, "remount", "")
			if nil != err {
				return err
			}
		}
	}
	return nil
}


func isLegalLayerName(name string) bool {
	for pos, c := range name {
		if !unicode.In(c, unicode.L, unicode.Nd) && c != '_' && (c != '-' || pos == 0) {
			return false
		}
	}
	return true
}


type nametest struct {
	name string
	mask int
	desc string
}

const (
	name_need = 1 << iota
	name_free
	name_optional
)

func (ld *Layerdefs) testName(tests...nametest) error {
	for _, test := range tests {
		name := test.name
		mask := test.mask
		desc := test.desc
		if len(name) < 1 {
			if (mask & name_optional) > 0 {
				continue
			}
			return fmt.Errorf("%s name is not set", desc)
		}
		if !isLegalLayerName(name) {
			return fmt.Errorf("%s name '%s' is not legal", desc, name)
		}
		if (mask & name_free) > 0 {
			if _, have := ld.layermap[name]; have {
				return fmt.Errorf("%s name '%s' already exists", desc, name)
			}
		} else if (mask & name_need) > 0 {
			if _, have := ld.layermap[name]; !have {
				return fmt.Errorf("%s name '%s' does not exist", desc, name)
			}
		}
	}
	return nil
}


func (ld *Layerdefs) getAncestorsAndSelf(name string) ([]*Layerinfo, error) {
	err := ld.testName(nametest{name, name_need, "Layer"})
	if nil != err {
		return nil, err
	}
	pos := len(ld.normalizedOrder)
	chain := make([]*Layerinfo, pos)
	for len(name) > 0 {
		layer := ld.layermap[name]
		pos--
		chain[pos] = layer
		name = layer.Base
	}
	return chain[pos:], nil
}

