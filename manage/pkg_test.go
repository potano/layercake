// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package manage


import (
	"os"
	"fmt"
	"path"
	"bytes"
	"strings"
	"io/ioutil"
	"potano.layercake/fs"
	"potano.layercake/fns"
	"potano.layercake/config"
	"potano.layercake/defaults"

	"testing"
)


const (
	wantfile_file = iota
	wantfile_dir
	wantfile_symlink
)

var wantfile_desc []string = []string{"file", "directory", "symlink"}

type Tmpdir struct {
	rootdir string
	want *wantFile
}

func NewTmpdir(patt string) (*Tmpdir, error) {
	name, err := ioutil.TempDir("", patt)
	return &Tmpdir{name, newWantFile("", wantfile_dir, "")}, err
}

func (t *Tmpdir) Cleanup() {
	os.RemoveAll(t.rootdir)
}

func (t *Tmpdir) Path(name string) string {
	return path.Join(t.rootdir, name)
}

func (t *Tmpdir) Mkdir(dirname string) error {
	pathname := t.Path(dirname)
	err := os.MkdirAll(pathname, 0755)
	if err == nil {
		t.want.MkdirAll(dirname)
	}
	return err
}

func (t *Tmpdir) Mkdirs(dirname, namelist string) error {
	for _, name := range strings.Split(namelist, " ") {
		err := t.Mkdir(path.Join(dirname, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Tmpdir) WriteFile(filename, contents string) error {
	pathname := t.Path(filename)
	err := ioutil.WriteFile(pathname, []byte(contents), 0644)
	if err == nil {
		t.want.WriteFile(filename, contents)
	}
	return err
}

func (t *Tmpdir) ReadFile(filename string) (string, error) {
	pathname := t.Path(filename)
	buf, err := ioutil.ReadFile(pathname)
	return string(buf), err
}

func (t *Tmpdir) Remove(filename string) error {
	pathname := t.Path(filename)
	err := os.Remove(pathname)
	if err == nil {
		t.want.Remove(filename)
	}
	return err
}

func (t *Tmpdir) IsFile(filename string) bool {
	pathname := t.Path(filename)
	return fs.IsFile(pathname)
}

func (td *Tmpdir) checkIfPathError(t *testing.T, err error, op, path, msg, desc string) {
	if err == nil {
		t.Fatal("test for " + desc + " returned no error")
	}
	if e, ok := err.(*os.PathError); !ok || e.Op != op || e.Path != td.Path(path) ||
		e.Err.Error() != msg {
			t.Fatalf("test for %s returned error %s", desc, err)
	}
}

func (td *Tmpdir) checkIfPathNotFoundError(t *testing.T, err error, op, path, desc string) {
	td.checkIfPathError(t, err, op, path, "no such file or directory", desc)
}

func (td *Tmpdir) checkExpectedFileContents(t *testing.T, err error, name, contents, desc string) {
	if err != nil {
		t.Fatalf("%s: got error %s", desc, err)
	}
	blob, err := td.ReadFile(name)
	if err != nil {
		t.Fatalf("%s: got error %s", desc, err)
	}
	if blob != contents {
		t.Fatalf("%s: read from file %s:\n%s", desc, name, blob)
	}
}

type wantFile struct {
	name string
	tp int
	contents string
	entries map[string]*wantFile
}

func newWantFile(name string, tp int, contents string) *wantFile {
	return &wantFile{name, tp, contents, map[string]*wantFile{}}
}

func (wf *wantFile) findParent(pathname string) (*wantFile, []string) {
	segs := strings.Split(pathname, "/")
	for len(segs) > 1 {
		var entry *wantFile
		name := segs[0]
		if len(name) == 0 {
			segs = segs[1:]
			continue
		}
		if entry = wf.entries[name]; entry == nil || entry.tp != wantfile_dir {
			break
		}
		wf = entry
		segs = segs[1:]
	}
	return wf, segs
}

func (wf *wantFile) WriteFile(filename, contents string) bool {
	dir, list := wf.findParent(filename)
	if dir.tp != wantfile_dir || len(list) != 1 {
		return false
	}
	dir.entries[list[0]] = newWantFile(list[0], wantfile_file, contents)
	return true
}

func (wf *wantFile) MkdirAll(dirname string) bool {
	dir, list := wf.findParent(dirname)
	if dir.tp != wantfile_dir {
		return false
	}
	for len(list) > 0 {
		name := list[0]
		list = list[1:]
		entry := newWantFile(name, wantfile_dir, "")
		dir.entries[name] = entry
		dir = entry
	}
	return true
}

func (wf *wantFile) Remove(filename string) bool {
	dir, list := wf.findParent(filename)
	if dir.tp != wantfile_dir || len(list) != 1 {
		return false
	}
	name := list[0]
	entry := dir.entries[name]
	if entry == nil || len(entry.entries) > 0 {
		return false
	}
	delete(dir.entries, name)
	return true
}

func (wf *wantFile) RemoveAll(filename string) bool {
	dir, list := wf.findParent(filename)
	if dir.tp != wantfile_dir || len(list) != 1 {
		return false
	}
	name := list[0]
	entry := dir.entries[name]
	if entry == nil {
		return false
	}
	delete(dir.entries, name)
	return true
}

func (wf *wantFile) RenameDirentry(pathname, newname string) bool {
	dir, list := wf.findParent(pathname)
	if dir.tp != wantfile_dir || len(list) != 1 {
		return false
	}
	name := list[0]
	entry := dir.entries[name]
	if entry == nil {
		return false
	}
	delete(dir.entries, name)
	dir.entries[newname] = entry
	return true
}

func fillWantFilesFromPath(pathname string) (*wantFile, error) {
	wf := newWantFile("", wantfile_dir, "")
	files, err := (func (pathname string) ([]os.FileInfo, error) {
		fh, err := os.Open(pathname)
		if err != nil {
			return nil, err
		}
		defer fh.Close()
		return fh.Readdir(-1)
	})(pathname)
	if err != nil {
		return wf, err
	}
	for _, fi := range files {
		name := fi.Name()
		mode := fi.Mode()
		newpath := pathname + "/" + name
		if (mode & (os.ModeNamedPipe|os.ModeSocket|os.ModeDevice|os.ModeIrregular)) != 0 {
			return wf, fmt.Errorf("Unexpected file mode %d found for %s", mode, newpath)
		}
		if fi.IsDir() {
			dir, err := fillWantFilesFromPath(newpath)
			if err != nil {
				return wf, err
			}
			dir.name = name
			wf.entries[name] = dir
		} else if (mode & os.ModeSymlink) != 0 {
			target, err := os.Readlink(newpath)
			if err != nil {
				return wf, err
			}
			wf.entries[name] = newWantFile(name, wantfile_symlink, target)
		} else {
			buf, err := ioutil.ReadFile(newpath)
			if err != nil {
				return wf, err
			}
			wf.entries[name] = newWantFile(name, wantfile_file, string(buf))
		}
	}
	return wf, nil
}

func (t *Tmpdir) UpdateWantFiles() error {
	wf, err := fillWantFilesFromPath(t.rootdir)
	if err != nil {
		return err
	}
	t.want = wf
	return nil
}

func (td *Tmpdir) CheckAgainstWantedTree(t *testing.T, phase string) {
	var messages messageSlice
	var compareTwo func (want, have *wantFile, basepath string)
	compareTwo = func (want, have *wantFile, basepath string) {
		for name, w := range want.entries {
			h := have.entries[name]
			newpath := basepath + "/" + name
			if h == nil {
				messages.addf("missing expected %s %s",
					wantfile_desc[w.tp], newpath)
			} else if h.tp != w.tp {
				messages.addf("expected %s %s, got %s", wantfile_desc[w.tp],
					newpath, wantfile_desc[h.tp])
			} else if h.contents != w.contents {
				if w.tp == wantfile_symlink {
					messages.addf("unexpected target of symlink %s: %s",
						newpath, h.contents)
				} else {
					messages.addf("unexpected contents of file %s: %s",
						newpath, h.contents)
				}
			} else if w.tp == wantfile_dir {
				compareTwo(w, h, newpath)
			}
		}
		for name, h := range have.entries {
			if want.entries[name] == nil {
				messages.addf("unexpected %s %s", wantfile_desc[h.tp],
					basepath + "/" + name)
			}
		}
	}

	have, err := fillWantFilesFromPath(td.rootdir)
	if err != nil {
		messages.add(err.Error())
	} else {
		compareTwo(td.want, have, "")
	}
	if len(messages) > 0 {
		t.Fatalf("%s: differences between wanted and expected files\n  %s", phase,
			strings.Join(messages, "\n  "))
	}
}

func (t *Tmpdir) MakeConfigTypeObj() (*config.ConfigType, error) {
	return config.Load("/dev/null", t.Path("/var/lib/layercake"))
}




type messageSlice []string

func (m *messageSlice) add(msg string) {
	*m = append(*m, msg)
}

func (m *messageSlice) addf(format string, p...interface{}) {
	*m = append(*m, fmt.Sprintf(format, p...))
}



type capturingMessageWriterType struct {
	b *bytes.Buffer
}

func (cmr capturingMessageWriterType) Write(msg []byte) (int, error) {
	return cmr.b.Write(append(msg,  '\n'))
}

var capturingMessageWriter capturingMessageWriterType = capturingMessageWriterType{&bytes.Buffer{}}

func readMessage() string {
	return capturingMessageWriter.b.String()
}



const (
	layercake_path = "var/lib/layercake"
	layercake_layers_path = layercake_path + "/layers"
	layercake_export_path = layercake_path + "/export"
	layercake_skel_path = layercake_path + "/default_layerconfig.skel"
	layercake_export_html_path = layercake_path + "/export/index.html"
)

func checkErrorByMessage(t *testing.T, err error, msg, phase string) {
	if err != nil {
		if err.Error() == msg {
			return
		}
		t.Fatalf("%s: got unexpected error %s", phase, err)
	}
	if len(msg) > 0 {
		t.Fatalf("%s: expected error", phase)
	}
}

func checkMia(t *testing.T, td *Tmpdir, mia []string, indices []int) {
	ok := len(mia) == len(indices)
	if ok {
		for i, idx := range indices {
			var msg string
			switch idx {
			case 1:
				msg = "base directory " + td.Path(layercake_path)
			case 2:
				msg = "base directory " + td.Path(layercake_layers_path)
			case 3:
				msg = "base directory " + td.Path(layercake_export_path)
			case 4:
				msg = "default layer configuration " + td.Path(layercake_skel_path)
			}
			if mia[i] != msg {
				ok = false
			}
		}
	}
	if !ok {
		if len(mia) > 0 {
			t.Fatalf("CheckBaseSetUp returned\n   " + strings.Join(mia, "\n   "))
		} else {
			t.Fatalf("CheckBaseSetup should have returned missing files")
		}
	}
}

func checkWillNotOverwrite(t *testing.T, td *Tmpdir, err error, indices []int, phase string) {
	var errmsg string
	if err == nil {
		if len(indices) > 0 {
			errmsg = "expected to have error"
		}
	} else {
		errmsg = err.Error()
		var parts []string
		for _, idx := range indices {
			var part string
			switch idx {
			case 1:
				part = "default layer configuration " +
					td.Path(layercake_skel_path)
			case 2:
				part = "export-directory file " +
					td.Path(layercake_export_html_path)
			}
			parts = append(parts, part)
		}
		if errmsg == "Will not overwrite " + strings.Join(parts, " or ") +
			"; delete manually to create default files" {
				errmsg = ""
		}
	}
	if len(errmsg) > 0 {
		t.Fatalf("InitLayercakeBase %s: %s", phase, errmsg)
	}
}

func compareNeededMountTypes(want, have []NeededMountType) []string {
	var problems messageSlice
	havemap := map[string]NeededMountType{}
	for _, h := range have {
		havemap[h.Mount] = h
	}
	for _, w := range want {
		name := w.Mount
		desc := "mount " + name
		if h, ok := havemap[name]; !ok {
			problems.add("lacks " + desc)
		} else {
			if h.Source != w.Source {
				problems.add(desc + " has Source " + h.Source)
			}
			if h.Fstype != w.Fstype {
				problems.add(desc + " has Fstype " + h.Fstype)
			}
			delete(havemap, name)
		}
	}
	for mp := range havemap {
		problems.add("has extra mount " + mp)
	}
	return problems
}

func compareMounts(want, have []*fs.MountType) []string {
	var problems messageSlice
	havemap := map[string]*fs.MountType{}
	for _, h := range have {
		havemap[h.Mountpoint] = h
	}
	for _, w := range want {
		name := w.Mountpoint
		desc := "mountpoint " + name
		if h, ok := havemap[name]; !ok {
			problems.add("lacks " + desc)
		} else {
			if h.Source != w.Source {
				problems.add(desc + " has Source = " + h.Source)
			}
			if h.Mountpoint != w.Mountpoint {
				problems.add(desc + " has Mountpoint = " + h.Mountpoint)
			}
			if h.Source != w.Source2 {
				problems.add(desc + " has Source2 = " + h.Source2)
			}
			if h.Fstype != w.Fstype {
				problems.add(desc + " has Fstype = " + h.Fstype)
			}
			if h.Options != w.Options {
				problems.add(desc + " has Options = " + h.Options)
			}
		}
	}
	for mp := range havemap {
		problems.add("has extra mountpoint " + mp)
	}
	return problems
}


type wantedLayerData struct {
	Name, Base string
	Busy, Overlain, Chroot bool
	Messages []string
}

func checkLayerDescriptions(t *testing.T, layers *Layerdefs, wants []wantedLayerData,
phase string) {
	llist := layers.Layers()
	if len(llist) != len(wants) {
		t.Fatalf("%s: expected %d layer(s), got %d", phase, len(wants), len(llist))
	}
	for i, l := range llist {
		want := wants[i]
		if l.Name != want.Name {
			t.Fatalf("%s: got layer name %s", phase, l.Name)
		}
		if l.Base != want.Base {
			t.Fatalf("%s: layer %s has unexpected base %s", phase, l.Name, l.Base)
		}
		if l.Busy != want.Busy {
			t.Fatalf("%s, layer %s has unexpected Busy=%t", phase, l.Name, l.Busy)
		}
		if l.Overlain != want.Overlain {
			t.Fatalf("%s, layer %s has unexpected Overlain=%t", phase, l.Name,
				l.Overlain)
		}
		if l.Chroot != want.Chroot {
			t.Fatalf("%s, layer %s has unexpected Chroot=%t", phase, l.Name, l.Chroot)
		}
		state := layers.DescribeState(l, true)
		if !stringSlicesEqual(want.Messages, state) {
			t.Fatalf("%s, layer %s has unexpected message(s)\n  %s", phase, l.Name,
				strings.Join(state, "\n  "))
		}
	}
}


func stringSlicesEqual(want, have []string) bool {
	ok := len(want) == len(have)
	if ok {
		for i, w := range want {
			if w != have[i] {
				ok = false
				break
			}
		}
	}
	return ok
}

func checkSameLayerinfo(t *testing.T, want Layerinfo, have *Layerinfo, phase string) {
	var problems messageSlice
	if have.Name != want.Name {
		problems.add("Name = " + have.Name)
	}
	if have.Base != want.Base {
		problems.add("Base = " + have.Base)
	}
	p2 := compareNeededMountTypes(want.ConfigMounts, have.ConfigMounts)
	if len(p2) > 0 {
		problems.add("ConfigMounts:\n  " + strings.Join(p2, "\n  "))
	}
	p2 = compareNeededMountTypes(want.ConfigExports, have.ConfigExports)
	if len(p2) > 0 {
		problems.add("ConfigExports:\n  " + strings.Join(p2, "\n  "))
	}
	if have.LayerPath != want.LayerPath {
		problems.add("LayerPath = " + have.LayerPath)
	}
	if have.State != want.State {
		problems.addf("State = %d", have.State)
	}
	if !stringSlicesEqual(want.Messages, have.Messages) {
		problems.add("Messages:\n  " + strings.Join(have.Messages, "\n  "))
	}
	if have.Busy != want.Busy {
		problems.addf("Busy = %t", have.Busy)
	}
	if have.Overlain != want.Overlain {
		problems.addf("Overlain = %t", have.Overlain)
	}
	if have.Chroot != want.Chroot {
		problems.addf("Chroot = %t", have.Chroot)
	}
	p2 = compareMounts(want.Mounts, have.Mounts)
	if len(p2) > 0 {
		problems.add("Mounts:\n  " + strings.Join(p2, "\n  "))
	}
	if len(problems) > 0 {
		t.Fatalf("%s: have Layerinfo\n%s", phase, strings.Join(problems, "\n"))
	}
}

func getLayers(t *testing.T, cfg *config.ConfigType, opts *config.Opts, inuse map[string]int,
phase string) *Layerdefs {
	layers, err := FindLayers(cfg, opts)
	if err != nil {
		t.Fatalf("%s: %s", phase, err)
	}
	err = layers.ProbeAllLayerstate(inuse)
	if err != nil {
		t.Fatalf("%s: %s", phase, err)
	}
	return layers
}



func TestManage(t *testing.T) {
	fs.MessageWriter = capturingMessageWriter
	var emptyFsMounts []*fs.MountType

	td, err := NewTmpdir("layercake_manage")
	if err != nil {
		t.Fatal(err)
	}
	defer td.Cleanup()

	savedSyscallMount := fs.SyscallMount
	savedSyscallUnmount := fs.SyscallUnmount

	cfg, err := td.MakeConfigTypeObj()
	if err != nil {
		t.Fatalf("%s loading default configuration", err)
	}

	if !t.Run("init", func (t *testing.T) {
		mia := CheckBaseSetUp(cfg)
		checkMia(t, td, mia, []int{1, 2, 3, 4})

		err = InitLayercakeBase(cfg)
		if err != nil {
			t.Fatalf("InitLayercakeBase: %s", err)
		}
		mia = CheckBaseSetUp(cfg)
		checkMia(t, td, mia, []int{})

		err = InitLayercakeBase(cfg)
		checkWillNotOverwrite(t, td, err, []int{1, 2}, "re-initialization")

		td.Remove(layercake_export_html_path)
		mia = CheckBaseSetUp(cfg)
		checkMia(t, td, mia, []int{})
		err = InitLayercakeBase(cfg)
		checkWillNotOverwrite(t, td, err, []int{1}, "replacing index.html")

		td.Remove(layercake_skel_path)
		mia = CheckBaseSetUp(cfg)
		checkMia(t, td, mia, []int{4})
		err = InitLayercakeBase(cfg)
		checkWillNotOverwrite(t, td, err, []int{2}, "replacing skeleton file")

		opts := &config.Opts{}
		layers, err := FindLayers(cfg, opts)
		if err != nil {
			t.Fatalf("FindLayers: %s", err)
		}
		if len(layers.layermap) > 0 {
			t.Fatalf("FindLayers: found %d layers", len(layers.layermap))
		}
	}) {
		return
	}


	if !t.Run("read_layerfile", func (t *testing.T) {
		typicalConfigMounts := []NeededMountType{
			{Mount: "/dev", Source: "/dev", Fstype: "rbind"},
			{Mount: "/proc", Source: "/proc", Fstype: "proc"},
			{Mount: "/sys", Source: "/sys", Fstype: "rbind"},
			{Mount: "/var/db/repos", Source: "/var/db/repos", Fstype: "rbind"},
			{Mount: "/var/cache/distfiles", Source: "/var/cache/distfiles",
				Fstype: "rbind"},
			{Mount: "/var/cache/binpkgs", Source: "$$base/packages", Fstype: "rbind"},
		}
		typicalConfigExports := []NeededMountType{}


		testName := "skeleton layout"
		li, err := ReadLayerFile(td.Path(layercake_skel_path), false)
		if err != nil {
			t.Fatalf("%s: got %s", testName, err)
		}
		checkSameLayerinfo(t, Layerinfo{
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports,
			Mounts: emptyFsMounts,
		}, li, testName)

		layerfileName := "test_layerfile"
		layerfilePath := td.Path(layerfileName)
		reducedLayerfileName := "test_reduced_layerfile"
		reducedLayerfilePath := td.Path(reducedLayerfileName)

		testName = "missing layerfile"
		li, err = ReadLayerFile(layerfilePath, false)
		td.checkIfPathNotFoundError(t, err, "open", layerfileName, testName)


		// Tests suppport for explicit export of a non-bind-mounted binpkg directory
		testName = "reduced layout, default config"
		reducedConfigMounts := []NeededMountType{
			{Mount: "/dev", Source: "/dev", Fstype: "rbind"},
			{Mount: "/proc", Source: "/proc", Fstype: "proc"},
			{Mount: "/sys", Source: "/sys", Fstype: "rbind"},
			{Mount: "/var/db/repos", Source: "/var/db/repos", Fstype: "rbind"},
			{Mount: "/var/cache/distfiles", Source: "/var/cache/distfiles",
				Fstype: "rbind"},
		}
		reducedConfigExports := []NeededMountType{
			{Mount: "$$package_export", Source: "/var/cache/binpkgs",
				Fstype: "symlink"},
		}
		td.WriteFile(reducedLayerfileName,
			`import rbind /dev /dev
			import proc /proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos /var/db/repos
			import rbind /var/cache/distfiles /var/cache/distfiles
			export symlink /var/cache/binpkgs $$package_export`)
		li, err = ReadLayerFile(reducedLayerfilePath, true)
		if err != nil {
			t.Fatalf("%s: got %s", testName, err)
		}
		checkSameLayerinfo(t, Layerinfo{
			ConfigMounts: reducedConfigMounts,
			ConfigExports: reducedConfigExports,
			Mounts: emptyFsMounts,
		}, li, testName)


		testName = "paths with extraneous slashes"
		td.WriteFile(layerfileName,
			`import rbind /dev /dev
			import proc /proc /proc/
			import rbind /sys /sys
			import rbind /var/db/repos/ /var/db/repos/
			import rbind /var/cache//distfiles/ /var/cache/distfiles
			import rbind $$base/packages /var/cache/binpkgs/`)
		li, err = ReadLayerFile(layerfilePath, true)
		if err != nil {
			t.Fatalf("%s: got %s", testName, err)
		}
		checkSameLayerinfo(t, Layerinfo{
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports,
			Mounts: emptyFsMounts,
		}, li, testName)


		testName = "base specified"
		td.WriteFile(layerfileName,
			`# Comment
			base basic
			import rbind /dev /dev
			import proc /proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos /var/db/repos
			import rbind /var/cache/distfiles /var/cache/distfiles
			import rbind $$base/packages /var/cache/binpkgs`)
		li, err = ReadLayerFile(layerfilePath, true)
		if err != nil {
			t.Fatalf("%s: got %s", testName, err)
		}
		checkSameLayerinfo(t, Layerinfo{
			Base: "basic",
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports,
			Mounts: emptyFsMounts,
		}, li, testName)


		testName = "unknown keyword"
		td.WriteFile(layerfileName,
			`import rbind /dev /dev
			input rbind /here /there
			import proc /proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos /var/db/repos
			import rbind /var/cache/distfiles /var/cache/distfiles
			import rbind $$base/packages /var/cache/binpkgs`)
		li, err = ReadLayerFile(layerfilePath, true)
		checkErrorByMessage(t, err, "Unknown layerconf keyword 'input' in " +
			layerfilePath + " line 2", testName)


		testName = "missing base name"
		td.WriteFile(layerfileName,
			`base
			import rbind /dev /dev
			import proc /proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos /var/db/repos
			import rbind /var/cache/distfiles /var/cache/distfiles
			import rbind $$base/packages /var/cache/binpkgs`)
		li, err = ReadLayerFile(layerfilePath, true)
		checkErrorByMessage(t, err, "No base specified in " +
			layerfilePath + " line 1", testName)


		testName = "invalid layer name"
		td.WriteFile(layerfileName,
			`base b.dir
			import rbind /dev /dev
			import proc /proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos /var/db/repos
			import rbind /var/cache/distfiles /var/cache/distfiles
			import rbind $$base/packages /var/cache/binpkgs`)
		li, err = ReadLayerFile(layerfilePath, true)  // validity checked at later stage
		checkErrorByMessage(t, err, "", "invalid layer name")
		checkSameLayerinfo(t, Layerinfo{
			Base: "b.dir",
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports,
			Mounts: emptyFsMounts,
		}, li, testName)


		testName = "2-argument import"
		td.WriteFile(layerfileName,
			`import rbind /dev
			import proc /proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos /var/db/repos
			import rbind /var/cache/distfiles /var/cache/distfiles
			import rbind $$base/packages /var/cache/binpkgs`)
		li, err = ReadLayerFile(layerfilePath, true)
		checkErrorByMessage(t, err, "Incomplete import specification in " + layerfilePath +
			" line 1", testName)


		testName = "2-argument export"
		td.WriteFile(layerfileName,
			`import rbind /dev /dev
			import proc /proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos /var/db/repos
			import rbind /var/cache/distfiles /var/cache/distfiles
			import rbind $$base/packages /var/cache/binpkgs
			export symlink /var/cache/binpkgs`)
		li, err = ReadLayerFile(layerfilePath, true)
		checkErrorByMessage(t, err, "Incomplete export specification in " +
			layerfilePath + " line 7", testName)


		testName = "2-argument export; soft error"
		li, err = ReadLayerFile(layerfilePath, false)
		checkErrorByMessage(t, err, "", testName)
		checkSameLayerinfo(t, Layerinfo{
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports[:0],
			Messages: []string{"Incomplete export specification in " +
				layerfilePath + " line 7"},
			Mounts: emptyFsMounts,
			State: 1,
		}, li, testName)


		testName = "no-base + 2-argument export; soft error"
		td.WriteFile(layerfileName,
			`base
			import rbind /dev /dev
			import proc /proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos /var/db/repos
			import rbind /var/cache/distfiles /var/cache/distfiles
			export symlink /var/cache/binpkgs
			import rbind $$base/packages /var/cache/binpkgs`)
		li, err = ReadLayerFile(layerfilePath, false)
		checkErrorByMessage(t, err, "", testName)
		checkSameLayerinfo(t, Layerinfo{
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports[:0],
			Messages: []string{
				"No base specified in " + layerfilePath + " line 1",
				"Incomplete export specification in " +
					layerfilePath + " line 7"},
			Mounts: emptyFsMounts,
			State: 1,
		}, li, testName)


		testName = "multiple bases, same value"
		td.WriteFile(layerfileName,
			`# Comment
			base fundamento
			import rbind /dev /dev
			import proc /proc /proc
			import rbind /sys /sys
			base fundamento extra items ignored
			import rbind /var/db/repos /var/db/repos
			import rbind /var/cache/distfiles /var/cache/distfiles
			import rbind $$base/packages /var/cache/binpkgs`)
		li, err = ReadLayerFile(layerfilePath, true)
		checkErrorByMessage(t, err, "", testName)
		checkSameLayerinfo(t, Layerinfo{
			Base: "fundamento",
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports,
			Mounts: emptyFsMounts,
		}, li, testName)


		testName = "multiple bases, different values"
		td.WriteFile(layerfileName,
			`# Comment
			base fundament
			import rbind /dev /dev
			import proc /proc /proc
			import rbind /sys /sys
			base fundamento
			import rbind /var/db/repos /var/db/repos
			import rbind /var/cache/distfiles /var/cache/distfiles
			export symlink /var/cache/binpkgs $$package_exports`)
		li, err = ReadLayerFile(layerfilePath, true)
		checkErrorByMessage(t, err, "New conflicting setting of base property in " +
			layerfilePath + " line 6", testName)
	}) {
		return
	}


	if !t.Run("write_layerfile", func (t *testing.T) {
		typicalConfigMounts := []NeededMountType{
			{Mount: "/dev", Source: "/dev", Fstype: "rbind"},
			{Mount: "/proc", Source: "/proc", Fstype: "proc"},
			{Mount: "/sys", Source: "/sys", Fstype: "rbind"},
			{Mount: "/var/db/repos", Source: "/var/db/repos", Fstype: "rbind"},
			{Mount: "/var/cache/distfiles", Source: "/var/cache/distfiles",
				Fstype: "rbind"},
			{Mount: "/var/cache/binpkgs", Source: "$$base/packages", Fstype: "rbind"},
		}
		typicalConfigExports := []NeededMountType{}


		testName := "basic file"
		layerfileName := "test_layerfile"
		layerfilePath := td.Path(layerfileName)
		li := Layerinfo{
			Name: "test",
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports,
			Mounts: emptyFsMounts}
		err := WriteLayerfile(layerfilePath, &li)
		td.checkExpectedFileContents(t, err, layerfileName,
`import rbind /dev /dev
import proc /proc /proc
import rbind /sys /sys
import rbind /var/db/repos /var/db/repos
import rbind /var/cache/distfiles /var/cache/distfiles
import rbind $$base/packages /var/cache/binpkgs
`, testName)


		testName = "extra import"
		li.ConfigMounts = append(typicalConfigMounts, NeededMountType{
			Mount: "/mnt/common", Source: "/root/common", Fstype: "rbind"})
		err = WriteLayerfile(layerfilePath, &li)
		td.checkExpectedFileContents(t, err, layerfileName,
`import rbind /dev /dev
import proc /proc /proc
import rbind /sys /sys
import rbind /var/db/repos /var/db/repos
import rbind /var/cache/distfiles /var/cache/distfiles
import rbind $$base/packages /var/cache/binpkgs
import rbind /root/common /mnt/common
`, testName)


		testName = "no binpkgs mount"
		reducedConfigMounts := []NeededMountType{
			{Mount: "/dev", Source: "/dev", Fstype: "rbind"},
			{Mount: "/proc", Source: "/proc", Fstype: "proc"},
			{Mount: "/sys", Source: "/sys", Fstype: "rbind"},
			{Mount: "/var/db/repos", Source: "/var/db/repos", Fstype: "rbind"},
			{Mount: "/var/cache/distfiles", Source: "/var/cache/distfiles",
				Fstype: "rbind"},
		}
		reducedConfigExports := []NeededMountType{
			{Mount: "$$package_export", Source: "/var/cache/binpkgs",
				Fstype: "symlink"},
		}
		li = Layerinfo{
			Name: "test",
			ConfigMounts: reducedConfigMounts,
			ConfigExports: reducedConfigExports,
			Mounts: emptyFsMounts}
		err = WriteLayerfile(layerfilePath, &li)
		td.checkExpectedFileContents(t, err, layerfileName,
`import rbind /dev /dev
import proc /proc /proc
import rbind /sys /sys
import rbind /var/db/repos /var/db/repos
import rbind /var/cache/distfiles /var/cache/distfiles

export symlink /var/cache/binpkgs $$package_export
`, testName)
	}) {
		return
	}


	if !t.Run("add_layer", func (t *testing.T) {
		basic_layerdef := fns.Template(
`import rbind /dev /dev
import proc /proc /proc
import rbind /sys /sys
import rbind {tmpdir}/var/db/repos /var/db/repos
import rbind {tmpdir}/var/cache/distfiles /var/cache/distfiles
import rbind $$base/packages /var/cache/binpkgs
`, map[string]string{"tmpdir": td.rootdir})

		reduced_layerdef := fns.Template(
`import rbind /dev /dev
import proc /proc /proc
import rbind /sys /sys
import rbind {tmpdir}/var/db/repos /var/db/repos
import rbind {tmpdir}/var/cache/distfiles /var/cache/distfiles

export symlink /var/cache/binpkgs $$package_export
`, map[string]string{"tmpdir": td.rootdir})
_ = reduced_layerdef

		alt_layerdef := fns.Template(
`import rbind /dev /dev
import proc /proc /proc
import rbind /sys /sys
import rbind {tmpdir}/var/db/repos/gentoo /usr/portage
import rbind {tmpdir}/var/cache/distfiles /var/cache/distfiles

export symlink /var/cache/binpkgs $$package_export
`, map[string]string{"tmpdir": td.rootdir})

		err := td.UpdateWantFiles()
		if err != nil {
			t.Fatal(err.Error())
		}
		opts := &config.Opts{}
		inuse := map[string]int{}

		td.WriteFile("/var/lib/layercake/default_layerconfig.skel", basic_layerdef)


		testName := "add base0"
		layers := getLayers(t, cfg, opts, inuse, testName)
		err = layers.AddLayer("base0", "", "")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		td.want.MkdirAll("/var/lib/layercake/layers/base0/build")
		td.want.WriteFile("/var/lib/layercake/layers/base0/layerconfig", basic_layerdef)
		td.want.MkdirAll("/var/lib/layercake/layers/base0/build/root")
		td.want.WriteFile("/var/lib/layercake/layers/base0/build/root/.bashrc",
			defaults.BaseLayerRootBashrc)
		td.CheckAgainstWantedTree(t, "after adding base0")
		layers = getLayers(t, cfg, opts, inuse, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, false, false, []string{"not yet populated"}},
		}, testName)

		testName = "remove just-added base0"
		err = layers.RemoveLayer("base0", false)
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		td.want.RemoveAll("/var/lib/layercake/layers/base0")
		td.CheckAgainstWantedTree(t, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{}, testName)


		testName = "add baseonly"
		layers = getLayers(t, cfg, opts, inuse, testName)
		err = layers.AddLayer("baseonly", "", "")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		td.want.MkdirAll("/var/lib/layercake/layers/baseonly/build")
		td.want.WriteFile("/var/lib/layercake/layers/baseonly/layerconfig", basic_layerdef)
		td.want.MkdirAll("/var/lib/layercake/layers/baseonly/build/root")
		td.want.WriteFile("/var/lib/layercake/layers/baseonly/build/root/.bashrc",
			defaults.BaseLayerRootBashrc)
		td.CheckAgainstWantedTree(t, "after adding baseonly")
		layers = getLayers(t, cfg, opts, inuse, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"baseonly", "", false, false, false, []string{"not yet populated"}},
		}, testName)

		testName = "remove just-added baseonly"
		err = layers.RemoveLayer("baseonly", false)
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		td.want.RemoveAll("/var/lib/layercake/layers/baseonly")
		td.CheckAgainstWantedTree(t, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{}, testName)


		testName = "re-add base0"
		layers = getLayers(t, cfg, opts, inuse, testName)
		err = layers.AddLayer("base0", "", "")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		td.want.MkdirAll("/var/lib/layercake/layers/base0/build")
		td.want.WriteFile("/var/lib/layercake/layers/base0/layerconfig", basic_layerdef)
		td.want.MkdirAll("/var/lib/layercake/layers/base0/build/root")
		td.want.WriteFile("/var/lib/layercake/layers/base0/build/root/.bashrc",
			defaults.BaseLayerRootBashrc)
		td.CheckAgainstWantedTree(t, testName)
		layers = getLayers(t, cfg, opts, inuse, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "add base1"
		td.WriteFile("/var/lib/layercake/usrportage.skel", alt_layerdef)
		layers = getLayers(t, cfg, opts, inuse, testName)
		err = layers.AddLayer("base1", "", "usrportage.skel")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		td.want.MkdirAll("/var/lib/layercake/layers/base1/build")
		td.want.WriteFile("/var/lib/layercake/layers/base1/layerconfig", alt_layerdef)
		td.want.MkdirAll("/var/lib/layercake/layers/base1/build/root")
		td.want.WriteFile("/var/lib/layercake/layers/base1/build/root/.bashrc",
			defaults.BaseLayerRootBashrc)
		td.CheckAgainstWantedTree(t, testName)
		layers = getLayers(t, cfg, opts, inuse, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, false, false, []string{"not yet populated"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "add derived1"
		layers = getLayers(t, cfg, opts, inuse, testName)
		err = layers.AddLayer("derived1", "base1", "")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		td.want.MkdirAll("/var/lib/layercake/layers/derived1/build")
		td.want.WriteFile("/var/lib/layercake/layers/derived1/layerconfig",
			"base base1\n\n" + alt_layerdef)
		td.want.MkdirAll("/var/lib/layercake/layers/derived1/overlayfs/workdir")
		td.want.MkdirAll("/var/lib/layercake/layers/derived1/overlayfs/upperdir")
		td.CheckAgainstWantedTree(t, testName)
		layers = getLayers(t, cfg, opts, inuse, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, false, false, []string{"not yet populated"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
			{"derived1", "base1", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "attempt to re-add derived1"
		layers = getLayers(t, cfg, opts, inuse, testName)
		err = layers.AddLayer("derived1", "base1", "")
		checkErrorByMessage(t, err, "Layer name 'derived1' already exists", testName)


		testName = "attempt base cycle"
		layers = getLayers(t, cfg, opts, inuse, testName)
		err = layers.AddLayer("derived1", "derived1", "")
		checkErrorByMessage(t, err, "Layer name 'derived1' already exists", testName)


		testName = "add with non-existent base"
		layers = getLayers(t, cfg, opts, inuse, testName)
		err = layers.AddLayer("derived2", "something1", "")
		checkErrorByMessage(t, err, "Parent layer name 'something1' does not exist",
			testName)


		testName = "add derived0"
		layers = getLayers(t, cfg, opts, inuse, testName)
		err = layers.AddLayer("derived0", "base0", "")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		td.want.MkdirAll("/var/lib/layercake/layers/derived0/build")
		td.want.WriteFile("/var/lib/layercake/layers/derived0/layerconfig",
			"base base0\n\n" + basic_layerdef)
		td.want.MkdirAll("/var/lib/layercake/layers/derived0/overlayfs/workdir")
		td.want.MkdirAll("/var/lib/layercake/layers/derived0/overlayfs/upperdir")
		td.CheckAgainstWantedTree(t, testName)
		layers = getLayers(t, cfg, opts, inuse, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, false, false, []string{"not yet populated"}},
			{"derived0", "base0", false, false, false, []string{"not yet populated"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
			{"derived1", "base1", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "added minimal entries to base0"
		minimalBaseDirectories := "bin etc lib opt root sbin usr"
		err = td.Mkdirs("/var/lib/layercake/layers/base0/build", minimalBaseDirectories)
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		layers = getLayers(t, cfg, opts, inuse, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, false, false, []string{"build directories set up",
				"missing mountpoint: /dev", "missing mountpoint: /proc",
				"missing mountpoint: /sys",
				"missing mountpoint: /var/db/repos",
				"missing mountpoint: /var/cache/distfiles",
				"missing mountpoint: /var/cache/binpkgs"}},
			{"derived0", "base0", false, false, false, []string{"not yet populated"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
			{"derived1", "base1", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "add some mountpoint directories to base0"
		err = td.Mkdirs("/var/lib/layercake/layers/base0/build", "dev proc sys")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		layers = getLayers(t, cfg, opts, inuse, "added derived0")
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, false, false, []string{"build directories set up",
				"missing mountpoint: /var/db/repos",
				"missing mountpoint: /var/cache/distfiles",
				"missing mountpoint: /var/cache/binpkgs"}},
			{"derived0", "base0", false, false, false, []string{"not yet populated"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
			{"derived1", "base1", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "add remaining mountpoint directories to base0"
		err = td.Mkdirs("/var/lib/layercake/layers/base0/build", "var/db/repos/gentoo " +
			"var/cache/distfiles var/cache/binpkgs")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		layers = getLayers(t, cfg, opts, inuse, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, false, false, []string{"build directories set up",
				"missing host directory: " + td.Path("/var/db/repos"),
				"missing host directory: " + td.Path("/var/cache/distfiles")}},
			{"derived0", "base0", false, false, false, []string{"not yet populated"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
			{"derived1", "base1", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "add host mount sources"
		err = td.Mkdirs("/", "var/db/repos/gentoo var/cache/distfiles")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		layers = getLayers(t, cfg, opts, inuse, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, false, false, []string{"mountable"}},
			{"derived0", "base0", false, false, false, []string{"mountable"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
			{"derived1", "base1", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "attempt to remove base0"
		err = layers.RemoveLayer("base0", true)
		checkErrorByMessage(t, err, "Layer base0 has at least one child layer", testName)

	}) {
		return
	}


	if !t.Run("mount_layer", func (t *testing.T) {
		m_ninja := newMountNinja()
		fs.GetAlternateProbeMountsCursor = func () fs.LineReader {
			mountinfo := m_ninja.mountinfo()
			reader := strings.NewReader(mountinfo)
			return fs.NewTextInputCursor("mountNinja", reader)
		}
		fs.SyscallMount = func (src, targ, fstype string, flgs uintptr, o string) error {
			return m_ninja.mount(src, targ, fstype, flgs, o)
		}
		fs.SyscallUnmount = func (mtpoint string, flags int) error {
			return m_ninja.unmount(mtpoint, flags)
		}
		defer func () {
			fs.GetAlternateProbeMountsCursor = nil
			fs.SyscallMount = savedSyscallMount
			fs.SyscallUnmount = savedSyscallUnmount
		}()

		opts := &config.Opts{}
		inuse_all_idle := map[string]int{}


		testName := "mount base0"
		layers := getLayers(t, cfg, opts, inuse_all_idle, testName)
		err := layers.Mount("base0")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, false, false, []string{"mounted and ready",
				"Current mounts:",
				"  required: /dev, /proc, /sys, /var/cache/binpkgs, /var/cache/distfiles, /var/db/repos"}},
			{"derived0", "base0", false, false, false, []string{"mountable"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
			{"derived1", "base1", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "mount derived0"
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		err = layers.Mount("derived0")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		td.Mkdirs("/var/lib/layercake/layers/derived0/build", "bin dev etc lib opt proc " +
			"root sbin sys usr /var/db/repos /var/cach/distfiles " +
			"var/cache/binpkgs var/cache/distfiles")
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, true, false, []string{"mounted and ready",
				"Current mounts:",
				"  required: /dev, /proc, /sys, /var/cache/binpkgs, /var/cache/distfiles, /var/db/repos"}},
			{"derived0", "base0", false, false, false, []string{"mounted and ready",
				"Current mounts:",
				"  required: /dev, /proc, /sys, /var/cache/binpkgs, /var/cache/distfiles, /var/db/repos",
				"  overlayfs"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
			{"derived1", "base1", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "attempt to mount derived1"
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		err = layers.Mount("derived1")
		checkErrorByMessage(t, err, "Layer base1 is not yet mountable", testName)


		testName = "attempt to rename derived0"
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		err = layers.RenameLayer("derived0", "othername")
		checkErrorByMessage(t, err, "Layer derived0 is mounted; cannot rename",
			testName)


		testName = "rebase derived1 to use derived0"
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		err = layers.RebaseLayer("derived1", "derived0")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, true, false, []string{"mounted and ready",
				"Current mounts:",
				"  required: /dev, /proc, /sys, /var/cache/binpkgs, /var/cache/distfiles, /var/db/repos"}},
			{"derived0", "base0", false, false, false, []string{"mounted and ready",
				"Current mounts:",
				"  required: /dev, /proc, /sys, /var/cache/binpkgs, /var/cache/distfiles, /var/db/repos",
				"  overlayfs"}},
			{"derived1", "derived0", false, false, false, []string{"mountable"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "mount derived1"
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		err = layers.Mount("derived1")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		td.Mkdirs("/var/lib/layercake/layers/derived1/build", "bin dev etc lib opt proc " +
			"root sbin sys usr/portage /var/cach/distfiles " +
			"var/cache/binpkgs var/cache/distfiles")
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, true, false, []string{"mounted and ready",
				"Current mounts:",
				"  required: /dev, /proc, /sys, /var/cache/binpkgs, /var/cache/distfiles, /var/db/repos"}},
			{"derived0", "base0", false, true, false, []string{"mounted and ready",
				"Current mounts:",
				"  required: /dev, /proc, /sys, /var/cache/binpkgs, /var/cache/distfiles, /var/db/repos",
				"  overlayfs"}},
			{"derived1", "derived0", false, false, false, []string{"mounted and ready",
				"Current mounts:",
				"  required: /dev, /proc, /sys, /usr/portage, /var/cache/distfiles",
				"  overlayfs"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "second attempt to rename derived0"
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		err = layers.RenameLayer("derived0", "othername")
		checkErrorByMessage(t, err,
			"Layer derived0 is mounted and is in use by overlay; cannot rename",
			testName)


		testName = "attempt to unmount derived0"
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		err = layers.Unmount("derived0", false)
		checkErrorByMessage(t, err, "Layer derived0 is in use by overlay; cannot unmount",
			testName)


		testName = "unmount all layers"
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		err = layers.Unmount("", true)
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, false, false, []string{"mountable"}},
			{"derived0", "base0", false, false, false, []string{"mountable"}},
			{"derived1", "derived0", false, false, false, []string{"mountable"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "mount derived1 with automatic mounting of ancestors"
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		err = layers.Mount("derived1")
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		td.Mkdirs("/var/lib/layercake/layers/derived1/build", "bin dev etc lib opt proc " +
			"root sbin sys usr/portage /var/cach/distfiles " +
			"var/cache/binpkgs var/cache/distfiles")
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, true, false, []string{"mounted and ready",
				"Current mounts:",
				"  required: /dev, /proc, /sys, /var/cache/binpkgs, /var/cache/distfiles, /var/db/repos"}},
			{"derived0", "base0", false, true, false, []string{"mounted and ready",
				"Current mounts:",
				"  required: /dev, /proc, /sys, /var/cache/binpkgs, /var/cache/distfiles, /var/db/repos",
				"  overlayfs"}},
			{"derived1", "derived0", false, false, false, []string{"mounted and ready",
				"Current mounts:",
				"  required: /dev, /proc, /sys, /usr/portage, /var/cache/distfiles",
				"  overlayfs"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
		}, testName)


		testName = "attempt to rename derived0"
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		err = layers.RenameLayer("derived0", "othername")
		checkErrorByMessage(t, err,
			"Layer derived0 is mounted and is in use by overlay; cannot rename",
			testName)


		testName = "attempt to unmount all with layer name specified"
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		err = layers.Unmount("someName", true)
		checkErrorByMessage(t, err,
			"Cannot specify unmount of a specific layer and also all layers",
			testName)


		testName = "verbosely unmount all layers"
		opts.Verbose = true
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		err = layers.Unmount("", true)
		if err != nil {
			t.Fatalf("%s: %s", testName, err)
		}
		layers = getLayers(t, cfg, opts, inuse_all_idle, testName)
		checkLayerDescriptions(t, layers, []wantedLayerData{
			{"base0", "", false, false, false, []string{"mountable"}},
			{"derived0", "base0", false, false, false, []string{"mountable"}},
			{"derived1", "derived0", false, false, false, []string{"mountable"}},
			{"base1", "", false, false, false, []string{"not yet populated"}},
		}, testName)
		msg := readMessage()
		if msg != "Layer base1 was not mounted\n" +
			"Unmounted layer derived1\n" +
			"Unmounted layer derived0\n" +
			"Unmounted layer base0\n" {
			t.Fatalf("%s: got verbose message %s", testName, msg)
		}
	}) {
		return
	}
}

