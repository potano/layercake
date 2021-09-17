package manage

import (
	"os"
	"fmt"
	"strings"
	"syscall"
	"unicode"
	"path/filepath"

	"potano.layercake/fs"
	"potano.layercake/config"
)

type Layerdefs struct {
	layers []Layerinfo
	names map[string]int
	baselayers []string
	buildroot, upperroot, workroot, htmlroot string
	baseErrors map[string]string
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


type Layerinfo struct {
	Name, Base string
	State int
	Messages []string
	Defined, Busy, Chroot, BaseError bool
	Mounts fs.Mounts
}

func NewLayerdefs() (*Layerdefs) {
	return &Layerdefs{
		names: map[string]int{},
	}
}

func (ld *Layerdefs) Layers() []Layerinfo {
	return ld.layers
}

func (ld *Layerdefs) DescribeState(li Layerinfo, detailed bool) []string {
	desc := ([]string{
		"defined but empty",
		"error",
		"incomplete setup",
		"not yet populated",
		"all directories complete",
		"mountable",
		"partially mounted",
		"mounted and ready",
	})[li.State]
	if len(li.Messages) > 0 {
		if "error" == desc {
			desc = li.Messages[0]
			if len(li.Messages) > 1 && !detailed {
				desc += fmt.Sprintf(" plus %d other error(s)", len(li.Messages))
			}
		} else if len(li.Messages) > 0 {
			desc += "; " + li.Messages[0]
		}
	}
	if li.BaseError {
		desc += "; " + ld.baseErrors[li.Name]
	}
	out := []string{desc}
	if detailed {
		if len(li.Messages) > 1 {
			out = append(out, li.Messages[1:]...)
		}
		mnts := ld.describeMounts(li, "  ")
		if len(mnts) > 0 {
			out = append(out, "Mounts:")
			out = append(out, mnts...)
		}
		if li.Busy && !li.Chroot {
			out = append(out, "layer is in use")
		}
	}
	return out
}

func (ld *Layerdefs) describeMounts(li Layerinfo, leftpad string) (out []string) {
	if len(li.Mounts) == 0 {
		return
	}
	var haveOverlay bool
	var other []string
	prefixes := make(map[string]string)
	buildpath := ld.buildroot + "/" + li.Name
	lenBuildpath := len(buildpath)
	for _, mnt := range li.Mounts {
		if len(mnt.Source2) > 0 && mnt.Mountpoint == buildpath {
			haveOverlay = true
		} else {
			found := false
			for _, nbm := range neededBaseMounts {
				mp := buildpath + nbm.mount
				if mp == mnt.Mountpoint {
					found = true
					prefixes[nbm.source] = mnt.Mountpoint
					break
				}
			}
			if !found {
				for _, mp := range prefixes {
					if strings.HasPrefix(mnt.Mountpoint, mp) {
						found = true
						break
					}
				}
			}
			if !found {
				srcpath := mnt.Source
				if strings.HasPrefix(srcpath, buildpath) {
					srcpath = srcpath[:lenBuildpath]
				}
				other = append(other, srcpath)
			}
		}
	}
	if len(prefixes) > 0 {
		basemounts := make([]string, 0, len(prefixes))
		for _, nbm := range neededBaseMounts {
			basemounts = append(basemounts, nbm.source)
		}
		out = append(out, leftpad + "required: " +
			strings.Join(basemounts, ", "))
	}
	if haveOverlay {
		out = append(out, leftpad + "overlayfs")
	}
	for _, src := range other {
		out = append(out, leftpad + src)
	}
	return
}




const (
	layerPresent_build = 1 << iota
	layerPresent_work
	layerPresent_upper
)

const layersPresent__base = layerPresent_build
const layersPresent__derived = layersPresent__base + layerPresent_work + layerPresent_upper

type layercomponentType struct {
	desc string	// User-oriented description of component
	cfgkey string	// Configuration key to use to component value
	mask int	// layerPresent mask
}

var layercomponents = []layercomponentType{
	{
		"Build directory",
		"buildroot",
		layerPresent_build,
	},
	{
		"Overlayfs work directory",
		"workroot",
		layerPresent_work,
	},
	{
		"Overlayfs upper directory",
		"upperroot",
		layerPresent_upper,
	},
}


func (ld *Layerdefs) ProbeLayerstate(cfg *config.ConfigType, mounts fs.Mounts,
		inuse map[string]int) error {
	if err := setBaseDirInfoFromCfg(cfg); nil != err {
		return err
	}
	ld.buildroot = cfg.GetPath("buildroot")
	ld.upperroot = cfg.GetPath("upperroot")
	ld.workroot = cfg.GetPath("workroot")
	ld.htmlroot = cfg.GetPath("htmlroot")
	for _, lc := range(layercomponents) {
		ldir := cfg.GetPath(lc.cfgkey)
		fh, err := os.Open(ldir)
		if nil != err {
			return err
		}
		defer fh.Close()
		lst, err := fh.Readdir(-1)
		if nil != err {
			return err
		}
		if nil == lst {
			continue
		}
		for _, entry := range lst {
			name := filepath.Base(entry.Name())
			var layer *Layerinfo
			if nameX, have := ld.names[name]; have {
				layer = &ld.layers[nameX]
			} else {
				ld.addLayerinfo(Layerinfo{
					Name: name,
				})
				layer = &ld.layers[ld.names[name]]
			}
			mode := uint32(entry.Mode()) & uint32(os.ModeDir)
			if mode == 0 {
				layer.Messages = append(layer.Messages, name +
					" exists but is not a directory")
				continue
			}
			layer.State |= lc.mask
		}
	}

	for lX := range ld.layers {
		layer := &ld.layers[lX]
		if 0 == layer.State {
			continue
		}
		mask := inuse[layer.Name]
		if mask > 0 {
			layer.Busy = true
			if (mask & fs.UseMask_root) > 0 {
				layer.Chroot = true
			}
		}
		if len(layer.Messages) > 0 {
			layer.State = Layerstate_error
			continue
		}
		if layer.Defined {
			var mask, missing int
			if len(layer.Base) > 0 {
				mask = layersPresent__derived
			} else {
				mask =  layersPresent__base
			}
			missing = layer.State ^ mask
			if missing > 0 {
				var dirs []string
				layer.State = Layerstate_incomplete
				for lcX := range layercomponents {
					if mask & layercomponents[lcX].mask > 0 {
						dirs = append(dirs, layercomponents[lcX].desc)
					}
				}
				layer.Messages = append(layer.Messages, "Missing " + strings.Join(dirs, ", "))
				continue
			}
		} else if (layer.State ^ layersPresent__derived) == 0 {
			mnt := mounts.GetMount(ld.buildroot + "/" + layer.Name)
			if nil != mnt && "overlay" == mnt.Fstype {
				layer.Base = filepath.Base(mnt.Source)
			}
		} else if (layer.State & layersPresent__base) > 0 {
			layer.State = Layerstate_incomplete
			continue
		}
		layer.State = Layerstate_complete
	}

	ld.normalizeOrder()
	ld.checkBases()

	//Now that layers are identified and the layers are in normalized order, do more
	// intense checking of each layer
	for lX := range ld.layers {
		layer := &ld.layers[lX]
		ld.findLayerstate(layer, mounts)
	}
	return nil
}

func (ld *Layerdefs) Caution() string {
	if nil != ld.baseErrors {
		ct := len(ld.baseErrors)
		if ct == 1 {
			return "Caution: a layer has an incorrect base-layer setup.\n" +
				"Mounting or chrooting it is not permitted.\n"
		}
		return fmt.Sprintf("Caution: %d layers have an incorrect base-layer setup.\n" +
				"Mounting or chrooting them is not permitted.\n", ct)
	}
	return ""
}

func (ld *Layerdefs) CautionIfCaution(before, after bool) {
	msg := ld.Caution()
	if len(msg) > 0 {
		if before {
			fmt.Println()
		}
		fmt.Print(msg)
		if after {
			fmt.Println()
		}
	}
}

func (ld *Layerdefs) AddLayer(name, base string, opts *config.Opts) error {
	err := ld.testName(nametest{name, 0, "Layer"},
		nametest{base, name_opt | name_need, "Parent layer"})
	if nil != err {
		return err
	}
	var descMessage string
	lX, have := ld.names[name]
	var layer *Layerinfo
	if have {
		layer = &ld.layers[lX]
		if layer.Defined {
			return fmt.Errorf("Layer %s is already defined", name)
		}
		if layer.State >= Layerstate_partialmount && layer.Base != base {
			var where string
			if len(layer.Base) > 0 {
				where = "with base layer " + layer.Base
			} else {
				where = "as a base layer"
			}
			return fmt.Errorf("Undefined layer %s is mounted %s; layer must be " +
				" unmounted before attempting to change basis", name, where)
		}
		if len(base) > 0 && layer.Base != base {
			layer.Base = base
		}
		layer.Defined = true
		descMessage = "add existing undefined layer %s to defined-layer list"
	} else {
		descMessage = "add layer %s"
		err := ld.addLayerinfo(Layerinfo{
			Name: name,
			Base: base,
			Defined: true,
		})
		lX := ld.names[name]
		layer = &ld.layers[lX]
		if nil != err {
			return err
		}
	}
	ld.normalizeOrder()
	if layer.BaseError {
		return fmt.Errorf("Cannot add layer %s because that would cause a cycle",
			name)
	}
	opts.Describe(descMessage, name)
	return nil
}

func (ld *Layerdefs) RemoveLayer(name string, removeFiles bool, opts *config.Opts) error {
	err := ld.testName(nametest{name, name_need, "Layer"})
	if nil != err {
		return err
	}
	lX := ld.names[name]
	layer := &ld.layers[lX]
	busy := layer.Busy
	mounted := len(layer.Mounts) > 0
	if busy || mounted {
		var msg string
		if busy && mounted {
			msg = "is mounted and has active users"
		} else if busy {
			msg = "has have users"
		} else {
			msg = "is mounted"
		}
		return fmt.Errorf("Cannot remove layer %s because it %s", name, msg)
	}
	for _, test := range ld.layers {
		if test.Base == name && test.Name != name {
			return fmt.Errorf("Layer %s is in use as a parent to another layer",
				name)
		}
	}
	ld.layers = append(ld.layers[:lX], ld.layers[lX+1:]...)
	opts.Describe("remove layer %s", name)
	ld.normalizeOrder()
	if removeFiles {
		builddir := ld.buildroot + "/" + name
		workdir := ld.workroot + "/" + name
		upperdir := ld.upperroot + "/" + name
		htmllink := ld.htmlroot + "/" + name
		if fs.IsDir(builddir) {
			opts.Describe("remove build directory of %s: %s", name, builddir)
			if !opts.Pretend {
				if err := fs.Remove(builddir); nil != err {
					return err
				}
			}
		} else if opts.Verbose {
			fmt.Printf("Build directory %s does not exist; skipping\n", builddir)
		}
		if len(layer.Base) > 0 {
			if fs.IsDir(upperdir) {
				opts.Describe("remove upper directory of %s: %s", name, upperdir)
				if !opts.Pretend {
					if err := fs.Remove(upperdir); nil != err {
						return err
					}
				}
			} else if opts.Verbose {
				fmt.Printf("Upper directory %s does not exist; skipping\n",
					upperdir)
			}
			if fs.IsDir(workdir) {
				opts.Describe("remove work directory of %s: %s", name, workdir)
				if !opts.Pretend {
					if err := fs.Remove(workdir); nil != err {
						return err
					}
				}
			} else if opts.Verbose {
				fmt.Printf("Work directory %s does not exist; skipping\n", workdir)
			}
		}
		if fs.IsSymlink(htmllink) {
			opts.Describe("remove HTML symlink of %s: %s", name, htmllink)
			if !opts.Pretend {
				if err := fs.Remove(htmllink); nil != err {
					return err
				}
			}
		} else if opts.Verbose {
			fmt.Printf("HTML symlink %s does not exist; skipping\n", htmllink)
		}
	}
	return nil
}

func (ld *Layerdefs) RenameLayer(oldname, newname string, opts *config.Opts) error {
	err := ld.testName(nametest{oldname, name_need, "Layer"},
		nametest{newname, name_free, "New name"})
	if nil != err {
		return err
	}
	lX := ld.names[oldname]
	var names []string
	for _, layer := range ld.layers {
		if layer.Base == oldname {
			names = append(names, layer.Name)
		}
	}
	if len(names) > 0 {
		if len(names) == 1 {
			return fmt.Errorf("Cannot rename layer %s: it is the basis of layer %s",
				oldname, names[0])
		}
		return fmt.Errorf("Cannot rename layer %s: it is the basis of the following: %s",
			oldname, strings.Join(names, ", "))
	}
	layer := &ld.layers[lX]
	if len(layer.Mounts) > 0 {
		return fmt.Errorf("Cannot rename layer %s: it is currently mounted", oldname)
	}
	if layer.Busy {
		return fmt.Errorf("Cannot rename layer %s: it is currently busy", oldname)
	}
	layer.Name = newname
	ld.normalizeOrder()
	opts.Describe("rename layer %s to %s", oldname, newname)
	if layer.State >= Layerstate_incomplete {
		builddir := ld.buildroot + "/" + oldname
		workdir := ld.workroot + "/" + oldname
		upperdir := ld.upperroot + "/" + oldname
		htmllink := ld.htmlroot + "/" + oldname
		newbuilddir := ld.buildroot + "/" + newname
		newworkdir := ld.workroot + "/" + newname
		newupperdir := ld.upperroot + "/" + newname
		newhtmllink := ld.htmlroot + "/" + newname
		if fs.IsDir(builddir) {
			opts.Describe("rename build directory %s", builddir)
			if !opts.Pretend {
				if err := fs.Rename(builddir, newbuilddir); nil != err {
					return err
				}
			}
		} else if opts.Verbose {
			fmt.Printf("Build directory %s did not exist; not renaming\n", builddir)
		}
		if len(layer.Base) > 0 {
			if fs.IsDir(upperdir) {
				opts.Describe("rename upper directory %s", upperdir)
				if !opts.Pretend {
					if err := fs.Rename(upperdir, newupperdir); nil != err {
						return err
					}
				}
			} else if opts.Verbose {
				fmt.Printf("Upper directory %s did not exit; not renaming\n",
					upperdir)
			}
			if fs.IsDir(workdir) {
				opts.Describe("rename work directory %s", workdir)
				if !opts.Pretend {
					if err := fs.Rename(workdir, newworkdir); nil != err {
						return err
					}
				}
			} else if opts.Verbose {
				fmt.Printf("Work directory %s did not exist; not renaming\n",
					workdir)
			}
		}
		if fs.IsSymlink(htmllink) {
			opts.Describe("rename HTML symlink %s", htmllink)
			if !opts.Pretend {
				if err := fs.Rename(htmllink, newhtmllink); nil != err {
					return err
				}
			}
		} else if opts.Verbose {
			fmt.Printf("HTML symlink did not exist; not renaming\n", htmllink)
		}
	}
	return nil
}

func (ld *Layerdefs) RebaseLayer(name, newbase string, opts *config.Opts) error {
	err := ld.testName(nametest{name, name_need, "Layer"},
		nametest{newbase, name_need | name_opt, "Parent layer"})
	if nil != err {
		return err
	}
	lX := ld.names[name]
	layer := &ld.layers[lX]
	if len(layer.Mounts) > 0 {
		return fmt.Errorf("Cannot rebase layer %s: it is currently mounted", name)
	}
	layer.Base = newbase
	ld.normalizeOrder()
	if layer.BaseError {
		return fmt.Errorf("Cannot rebase layer %s because that would cause a cycle", name)
	}
	opts.Describe("rebase layer %s to '%s'", name, newbase)
	return nil
}

func (ld *Layerdefs) Shell(name string, opts *config.Opts) error {
	if len(name) == 0 {
		if len(ld.baselayers) == 0 {
			return fmt.Errorf("There are no base layers defined")
		}
		name = ld.baselayers[0]
	}
	err := ld.testName(nametest{name, name_need, "Layer"})
	if nil != err {
		return err
	}
	lX := ld.names[name]
	layer := &ld.layers[lX]
	if layer.State == Layerstate_empty {
		return fmt.Errorf("Layer %s is empty; cannot change to it", name)
	}
	builddir := ld.buildroot + "/" + name
	if !fs.IsDir(builddir) {
		return fmt.Errorf("Build directory for layer %s does not exist", name)
	}
	fmt.Println("Exit to return to layercake.\nCaution: this is *not* a chroot.")
	return fs.Shell(builddir)
}

func (ld *Layerdefs) Makedirs(name string, opts *config.Opts) error {
	err := ld.testName(nametest{name, name_opt, "Layer"})
	if nil != err {
		return err
	}
	var nameList []string
	if len(name) == 0 {
		nameList = make([]string, 0, len(ld.layers))
		for _, layer := range ld.layers {
			if layer.Defined {
				nameList = append(nameList, layer.Name)
			}
		}
	} else {
		lX, have := ld.names[name]
		if !have {
			return fmt.Errorf("Layer %s does not exist", name)
		}
		layer := &ld.layers[lX]
		if !layer.Defined {
			return fmt.Errorf("Layer %s is not defined", name)
		}
		nameList = []string{name}
	}
	for _, name := range nameList {
		builddir := ld.buildroot + "/" + name
		workdir := ld.workroot + "/" + name
		upperdir := ld.upperroot + "/" + name
		htmllink := ld.htmlroot + "/" + name
		builddirIsNew := false
		if !fs.IsDir(builddir) {
			builddirIsNew = true
			opts.Describe("make build directory for %s: %s", name, builddir)
			if !opts.Pretend {
				if err := fs.Mkdir(builddir); nil != err {
					return err
				}
			}
		} else if opts.Verbose {
			fmt.Printf("Build directory %s already existed; skipping\n", builddir)
		}
		if len(ld.layers[ld.names[name]].Base) > 0 {
			if !fs.IsDir(upperdir) {
				opts.Describe("make upper directory for %s: %s", name, upperdir)
				if !opts.Pretend {
					if err := fs.Mkdir(upperdir); nil != err {
						return err
					}
				}
			} else if opts.Verbose {
				fmt.Printf("Upper directory %s already existed; skipping\n",
					upperdir)
			}
			if !fs.IsDir(workdir) {
				opts.Describe("make work directory for %s: %s", name, workdir)
				if !opts.Pretend {
					if err := fs.Mkdir(workdir); nil != err {
						return err
					}
				}
			} else if opts.Verbose {
				fmt.Printf("Work directory %s already existed; skipping\n", workdir)
			}
		}
		if !fs.IsSymlink(htmllink) {
			linkTarget := builddir + htmlLinkTarget
			if !builddirIsNew && !fs.IsDir(linkTarget) {
				fmt.Printf(
					"Caution: directory for link target (%s) does not exist\n",
					linkTarget)
			}
			opts.Describe("make HTML symlink for %s: %s", name, htmllink)
			if !opts.Pretend {
				if err := fs.Symlink(htmllink, linkTarget); nil != err {
					return err
				}
			}
		} else if opts.Verbose {
			fmt.Printf("HTML symlink %s already existed; skipping\n", htmllink)
		}
	}
	return nil
}

func (ld *Layerdefs) Mount(name string, mounts fs.Mounts, opts *config.Opts) error {
	if len(name) > 0 {
		ancestors, err := ld.getAncestorsAndSelf(name)
		if nil != err {
			return err
		}
		for _, lX := range ancestors {
			layer := &ld.layers[lX]
			if !layer.Defined {
				return fmt.Errorf("Cannot mount undefined layer %s", layer.Name)
			}
			if layer.State < Layerstate_complete {
				err = ld.Makedirs(ld.layers[lX].Name, opts)
				if nil != err {
					return err
				}
				ld.findLayerstate(layer, mounts)
			}
		}
		for _, lX := range ancestors {
			layer := &ld.layers[lX]
			if layer.State > Layerstate_mountable {
				continue
			}
			if layer.State < Layerstate_mountable {
				return fmt.Errorf("Layer %s is not yet mountable", layer.Name)
			}
			err = ld.mountOne(layer, opts)
			if nil != err {
				return err
			}
		}
	} else {
		for lX := range ld.layers {
			layer := &ld.layers[lX]
			if !layer.Defined {
				continue
			}
			if layer.State < Layerstate_complete {
				err := ld.Makedirs(ld.layers[lX].Name, opts)
				if nil != err {
					return err
				}
				ld.findLayerstate(layer, mounts)
			}
		}
		for lX := range ld.layers {
			layer := &ld.layers[lX]
			if layer.State == Layerstate_mountable {
				err := ld.mountOne(layer, opts)
				if nil != err {
					return err
				}
			}
		}
	}
	return nil
}

func (ld *Layerdefs) mountOne(layer *Layerinfo, opts *config.Opts) error {
	name := layer.Name
	builddir := ld.buildroot + "/" + name
	if len(layer.Base) > 0 {
		basedir := ld.buildroot + "/" + layer.Base
		workdir := ld.workroot + "/" + name
		upperdir := ld.upperroot + "/" + name
		if nil == layer.Mounts.GetMount(builddir) {
			opts.Describe("mount %s on %s (overlay)", upperdir, builddir)
			if !opts.Pretend {
				err := fs.Mount("overlay", builddir, "overlay",
					"lowerdir=" + basedir + ",upperdir=" + upperdir +
					",workdir=" + workdir)
				if nil != err {
					return err
				}
			}
		} else if opts.Verbose {
			fmt.Printf("Build directory %s already mounted; skipping", builddir)
		}
	}
	for _, m := range neededBaseMounts {
		mountpoint := builddir + m.mount
		if nil == layer.Mounts.GetMount(mountpoint) {
			opts.Describe("mount %s on %s (%s)", m.source, mountpoint, m.fstype)
			if !opts.Pretend {
				err := fs.Mount(m.source, mountpoint, m.fstype, "")
				if nil != err {
					return err
				}
			}
		} else if opts.Verbose {
			fmt.Printf("Directory %s already mounted; skipping", mountpoint)
		}
	}
	layer.State = Layerstate_mounted
	return nil
}

func (ld *Layerdefs) Unmount(name string, opts *config.Opts) error {
	if len(name) > 0 {
		lX, have := ld.names[name]
		if !have {
			return fmt.Errorf("Layer %s does not exist\n", name)
		}
		layer := &ld.layers[lX]
		if layer.Busy {
			return fmt.Errorf("Cannot unmount %s because it is busy\n", name)
		}
		for _, ly := range ld.layers {
			if ly.Base == name {
				if len(ly.Mounts) > 0 {
					return fmt.Errorf(
						"Cannot unmount %s because %s is mounted\n",
						name, ly.Name)
				}
			}
		}
		err := ld.unmountTree(layer.Mounts, opts)
		if nil != err {
			return err
		}
	} else {
		inhibited := make([]bool, len(ld.layers))
		for _, layer := range ld.layers {
			if len(layer.Mounts) > 0 && len(layer.Base) > 0 {
				baseX := ld.names[layer.Base]
				inhibited[baseX] = true
			}
		}
		for lX, layer := range ld.layers {
			name := layer.Name
			if layer.Busy {
				fmt.Printf("%s is busy; cannot unmount it\n", name)
				continue
			}
			if len(layer.Mounts) == 0 {
				if opts.Verbose {
					fmt.Printf("%s was already not mounted\n", name)
				}
				continue
			}
			if inhibited[lX] {
				fmt.Printf("%s is a base layer to at least one mounted layer," +
					" cannot unmount\n", name)
				continue
			}
			err := ld.unmountTree(layer.Mounts, opts)
			if nil != err {
				return err
			}
		}
	}
	return nil
}

func (ld *Layerdefs) unmountTree(mountlist fs.Mounts, opts *config.Opts) error {
	var addDesc string
	if opts.Force {
		addDesc = "forced "
	}
	for uX := len(mountlist) - 1; uX >= 0; uX-- {
		path := mountlist[uX].Mountpoint
		opts.Describe("%sunmount %s", addDesc, path)
		if !opts.Pretend {
			err := fs.Unmount(path, opts.Force)
			if nil != err {
				return err
			}
		}
	}
	return nil
}

func (ld *Layerdefs) Chroot(name string, mounts fs.Mounts,
		cfg *config.ConfigType, opts *config.Opts) error {
	if len(name) == 0 {
		if len(ld.baselayers) == 0 {
			return fmt.Errorf("There are no base layers defined")
		}
		name = ld.baselayers[0]
	}
	err := ld.testName(nametest{name, name_need, "Layer"})
	if nil != err {
		return err
	}
	lX := ld.names[name]
	layer := &ld.layers[lX]
	if layer.State < Layerstate_mounted {
		if err = ld.Mount(name, mounts, opts); nil != err {
			return err
		}
		if layer.State < Layerstate_mounted {
			return fmt.Errorf("Layer %s is not mounted; cannot change to it", name)
		}
	}
	builddir := ld.buildroot + "/" + name
	if !fs.IsDir(builddir) {
		return fmt.Errorf("Build directory for layer %s does not exist", name)
	}
	env := []string{"LAYERCAKE_LAYER=" + name}
	fds := []*os.File{}
	return fs.Chroot(builddir, cfg.GetCfg("chrootexec"), env, fds)
}

func (ld *Layerdefs) Shake(opts *config.Opts) error {
	for _, layer := range ld.layers {
		target := ld.buildroot + "/" + layer.Name
		if len(layer.Base) > 0 && layer.State >= Layerstate_mounted {
			opts.Describe("remount %s", target)
			if !opts.Pretend {
				err := fs.Mount("", target, "remount", "")
				if nil != err {
					return err
				}
			}
		} else if opts.Verbose && layer.Defined {
			if len(layer.Base) > 0 {
				fmt.Printf("Directory %s not mounted; skipping", target)
			} else {
				fmt.Printf("Directory %s is a base layer; skipping", target)
			}
		}
	}
	return nil
}


type nametest struct {
	name string
	mask int
	desc string
}

const (
	name_need = 1 << iota
	name_free
	name_opt
)

func (ld *Layerdefs) testName(tests...nametest) error {
	for _, test := range tests {
		name := test.name
		mask := test.mask
		desc := test.desc
		if len(name) < 1 {
			if (mask & name_opt) > 0 {
				continue
			}
			return fmt.Errorf("%s name is not set", desc)
		}
		for pos, c := range name {
			if !unicode.In(c, unicode.L, unicode.Nd) &&
				c != '_' && c != '.' && (c != '-' || pos == 0) {
				return fmt.Errorf("%s name '%s' is not legal", desc, name)
			}
		}
		if (mask & name_free) > 0 {
			if _, have := ld.names[name]; have {
				return fmt.Errorf("%s name '%s' already exists", desc, name)
			}
		} else if (mask & name_need) > 0 {
			if _, have := ld.names[name]; !have {
				return fmt.Errorf("%s name '%s' does not exist", desc, name)
			}
		}
	}
	return nil
}

func (ld *Layerdefs) addLayerinfo(li Layerinfo) error {
	name := li.Name
	base := li.Base
	err := ld.testName(nametest{name, 0, "Layer"}, nametest{base, name_opt, "Parent layer"})
	if nil != err {
		return err
	}
	if _, have := ld.names[name]; have {
		return fmt.Errorf("Duplicate layer name %s", name)
	}
	ld.names[name] = len(ld.layers)
	if len(base) == 0 {
		ld.baselayers = append(ld.baselayers, name)
	}
	ld.layers = append(ld.layers, li)
	return nil
}

func (ld *Layerdefs) reindex() {
	names := map[string]int{}
	baselayers := []string{}
	for lX, layer := range ld.layers {
		name := layer.Name
		base := layer.Base
		names[name] = lX
		if len(base) < 1 {
			baselayers = append(baselayers, name)
		}
	}
	ld.names = names
	ld.baselayers = baselayers
}

// Precondition: indices in ld.names are correct
func (ld *Layerdefs) checkBases() {
	var errs map[string]string
	for lX := range ld.layers {
		var err string
		layer := &ld.layers[lX]
		layer.BaseError = false
		name := layer.Name
		test := layer.Base
		path := []string{name}
		for len(test) > 0 {
			testX := len(path)
			path = append(path, test)
			nextBase, have := ld.names[test]
			if !have {
				err = "missing base layer: " + test
			} else {
				for pX, s := range path {
					if pX != testX && s == test {
						err = "base-layer cycle: " +
							strings.Join(path, " -> ")
						break
					}
				}
				test = ld.layers[nextBase].Base
			}
			if len(err) > 0 {
				layer.BaseError = true
				if nil == errs {
					errs = make(map[string]string)
				}
				errs[name] = err
				break
			}
		}
	}
	ld.baseErrors = errs
}

func (ld *Layerdefs) normalizeOrder() {
	needStack := make([]string, len(ld.layers))
	var needSP int
	need := ""
	lX := 0
	for lX < len(ld.layers) {
		found := false
		for sX := lX + 1; sX < len(ld.layers); sX++ {
			layer := ld.layers[sX]
			if layer.Base == need {
				needStack[needSP] = need
				needSP++
				need = layer.Name
				if sX != lX {
					copy(ld.layers[lX+1:], ld.layers[lX:sX])
					ld.layers[lX] = layer
				}
				lX++
				found = true
				break
			}
		}
		if !found {
			if needSP > 0 {
				needSP--
				need = needStack[needSP]
			} else {
				break
			}
		}
	}
	ld.reindex()
	ld.checkBases()
}

func (ld *Layerdefs) getAncestorsAndSelf(name string) (list []int, err error) {
	err = ld.testName(nametest{name, name_need, "Layer"})
	if nil != err {
		return
	}
	for {
		lX := ld.names[name]
		list = append(list, lX)
		copy(list[1:], list[0:])
		list[0] = lX
		layer := ld.layers[lX]
		if layer.BaseError {
			err = fmt.Errorf("Layer %s has a base error", name)
			return
		}
		name = layer.Base
		if len(name) == 0 {
			break
		}
	}
	return
}



type neededMountType struct {
	mount, source, fstype string
}

var neededBaseMounts = []neededMountType{
	{"/dev", "/dev", "rbind"},
	{"/proc", "proc", "proc"},
	{"/sys", "/sys", "rbind"},
}

var neededBaseDirs = []string{"bin", "dev", "etc", "lib32", "lib64", "opt", "proc", "root",
	"run", "sbin", "sys", "tmp", "usr"}

var htmlLinkTarget string

func setBaseDirInfoFromCfg(cfg *config.ConfigType) error {
	htmlLinkTarget = cfg.GetCfg("htmllink")
	if len(htmlLinkTarget) < 1 {
		return fmt.Errorf("Invalid configuration setting for HTML_SYMLINK_TARGET")
	}
	if '/' != htmlLinkTarget[0] {
		htmlLinkTarget = "/" + htmlLinkTarget
	}
	for _, mnt := range strings.Split(cfg.GetCfg("bindmount"), ",") {
		parts := strings.Fields(mnt)
		if len(parts) != 2 {
			return fmt.Errorf("Invalid configuration setting for BASE_BIND_MOUNT")
		}
		source := parts[0]
		target := parts[1]
		neededBaseMounts = append(neededBaseMounts,
			neededMountType{target, source, "rbind"})
		if target[0] == '/' {
			target = target[1:]
		}
		neededBaseDirs = append(neededBaseDirs, target)
	}
	return nil
}


func (ld *Layerdefs) findLayerstate(layer *Layerinfo, mounts fs.Mounts) {
	name := layer.Name
	builddir := ld.buildroot + "/" + name
	workdir := ld.workroot + "/" + name
	upperdir := ld.upperroot + "/" + name
	htmllink := ld.htmlroot + "/" + name

	layer.Mounts = mounts.GetMountAndSubmounts(builddir)
	if layer.State <= Layerstate_error {
		return
	}
	layerComplete := true;
	if have := fs.IsDir(builddir); !have {
		layer.Messages = append(layer.Messages, "Missing build directory " + builddir)
		layerComplete = false
	}
	if have := fs.IsSymlink(htmllink); !have {
		layer.Messages = append(layer.Messages, "Missing HTML symbolic link " + htmllink)
		layerComplete = false
	}
	if len(layer.Base) > 0 {
		if have := fs.IsDir(upperdir); !have {
			layer.Messages = append(layer.Messages, "Missing upper directory " + upperdir)
			layerComplete = false
		}
		if have := fs.IsDir(workdir); !have {
			layer.Messages = append(layer.Messages, "Missing work directory " + workdir)
			layerComplete = false
		}
	}
	if !layerComplete {
		layer.State = Layerstate_incomplete
		return
	}
	buf := make([]byte, 256)
	n, err := syscall.Readlink(htmllink, buf)
	if nil != err {
		layer.State = Layerstate_error
		layer.Messages = append(layer.Messages, "Could not dereference HTML symlink: " +
		   err.Error())
		return
	}
	if !strings.HasPrefix(string(buf[:n]), builddir) {
		layer.State = Layerstate_error
		layer.Messages = append(layer.Messages, "HTML symlink has wrong target")
		return
	}

	var badMounts int

	// Test for base overlayfs mounts
	layer.State = Layerstate_complete
	if len(layer.Base) > 0 {
		// Derived layer:  is correct overlayfs mount in place?
		baseX := ld.names[layer.Base]
		baseLayer := ld.layers[baseX]
		if baseLayer.State < Layerstate_mounted {
			layer.State = Layerstate_inhabited
			badMounts++
		} else {
			layer.State = Layerstate_mountable
		}
		mnt := layer.Mounts.GetMount(builddir)
		if nil == mnt {
			return
		}
		if mnt.Fstype != "overlay" {
			layer.Messages = append(layer.Messages, "Mounted but not as overlayfs")
			layer.State = Layerstate_error
			return
		}
		messages := make([]string, 0, 3)
		if mnt.Source != ld.buildroot + "/" + layer.Base {
			messages = append(messages, "Wrong parent directory mounted")
		}
		if mnt.Source2 != upperdir {
			messages = append(messages, "Wrong upper directory mounted")
		}
		if mnt.Workdir != workdir {
			messages = append(messages, "Wrong work directory mounted")
		}
		if len(messages) > 0 {
			layer.State = Layerstate_error
			layer.Messages = append(layer.Messages, messages...)
			return
		}
		// Now that we know we have the overlayfs mount, are needed directories present?
		for _, d := range neededBaseDirs {
			if !fs.IsDir(builddir + "/" + d) {
				layer.State = Layerstate_partialmount
				return
			}
		}
	} else {
		// Base layer:  are needed base directories and mounts present?
		for _, d := range neededBaseDirs {
			if !fs.IsDir(builddir + "/" + d) {
				return
			}
		}
		layer.State = Layerstate_mountable
	}
	for _, nbm := range neededBaseMounts {
		mnt := layer.Mounts.GetMount(builddir + nbm.mount)
		if nil == mnt {
			badMounts++
			continue
		}
		if mnt.Source != nbm.source {
			mnt2 := mounts.GetMount(nbm.source)
			if nil != mnt2 && mnt2.Source != mnt.Source {
				badMounts++
				continue
			}
		}
	}
	if len(layer.Mounts) > 0 {
		if badMounts > 0 {
			layer.State = Layerstate_partialmount
		} else {
			layer.State = Layerstate_mounted
		}
	}
	return
}

