package manage


import (
	"os"
	"fmt"
	"path"
	"bytes"
	"strings"
	"io/ioutil"
	"potano.layercake/fs"
	"potano.layercake/config"
	"potano.layercake/defaults"

	"testing"
)


type Tmpdir struct {
	rootdir string
}

func NewTmpdir(patt string) (*Tmpdir, error) {
	name, err := ioutil.TempDir("", patt)
	return &Tmpdir{name}, err
}

func (t *Tmpdir) Cleanup() {
	os.RemoveAll(t.rootdir)
}

func (t *Tmpdir) Path(name string) string {
	return path.Join(t.rootdir, name)
}

func (t *Tmpdir) Mkdir(dirname string) error {
	pathname := t.Path(dirname)
	return os.MkdirAll(pathname, 0755)
}

func (t *Tmpdir) WriteFile(filename, contents string) error {
	pathname := t.Path(filename)
	return ioutil.WriteFile(pathname, []byte(contents), 0644)
}

func (t *Tmpdir) ReadFile(filename string) (string, error) {
	pathname := t.Path(filename)
	buf, err := ioutil.ReadFile(pathname)
	return string(buf), err
}

func (t *Tmpdir) Remove(filename string) error {
	pathname := t.Path(filename)
	return os.Remove(pathname)
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

func (t *Tmpdir) MakeConfigTypeObj() *config.ConfigType {
	basePath := t.Path(defaults.BasePath)
	exportMap := map[string]string{}
	for _, pair := range strings.Split(defaults.ExportDirEntries, "|") {
		slice := strings.Split(pair, ":")
		exportMap[slice[0]] = slice[1]
	}
	return &config.ConfigType{
		Basepath: basePath,
		Layerdirs: path.Join(basePath, defaults.Layerdirs),
		LayerBuildRoot: defaults.Builddir,
		LayerOvfsWorkdir: defaults.Workdir,
		LayerOvfsUpperdir: defaults.Upperdir,
		Exportdirs: path.Join(basePath, defaults.Exportdirs),
		LayerExportDirs: exportMap,
		ChrootExec: defaults.ChrootExec,
	}
}




type messageSlice []string

func (m *messageSlice) add(msg string) {
	*m = append(*m, msg)
}

func (m *messageSlice) addf(format string, p...interface{}) {
	*m = append(*m, fmt.Sprintf(format, p...))
}




var capturingMessageWriter *bytes.Buffer

func readMessage() string {
	return capturingMessageWriter.String()
}



const (
	layercake_path = "var/lib/layercake"
	layercake_layers_path = "var/lib/layercake/layers"
	layercake_export_path = "var/lib/layercake/export"
	layercake_skel_path = "var/lib/layercake/default_layerconfig.skel"
	layercake_export_html_path = "var/lib/layercake/export/index.html"
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


func TestManage(t *testing.T) {
	fs.MessageWriter = capturingMessageWriter
	var emptyFsMounts []*fs.MountType

	td, err := NewTmpdir("layercake_manage")
	if err != nil {
		t.Fatal(err)
	}
	defer td.Cleanup()

	cfg := td.MakeConfigTypeObj()

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
			{Mount: "/proc", Source: "proc", Fstype: "proc"},
			{Mount: "/sys", Source: "/sys", Fstype: "rbind"},
			{Mount: "/var/db/repos/gentoo", Source: "/var/db/repos/gentoo",
				Fstype: "rbind"},
			{Mount: "/var/cache/distfiles", Source: "/var/cache/distfiles",
				Fstype: "rbind"},
		}
		typicalConfigExports := []NeededMountType{
			{Mount: "packages", Source: "/var/cache/binpkgs", Fstype: "symlink"},
		}

		li, err := ReadLayerFile(td.Path(layercake_skel_path), false)
		if err != nil {
			t.Fatalf("skeleton layout: got %s", err)
		}
		checkSameLayerinfo(t, Layerinfo{
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports,
			Mounts: emptyFsMounts,
		}, li, "skeleton layout")

		layerfileName := "test_layerfile"
		layerfilePath := td.Path(layerfileName)
		li, err = ReadLayerFile(layerfilePath, false)
		td.checkIfPathNotFoundError(t, err, "open", layerfileName, "missing layerfile")

		td.WriteFile(layerfileName,
			`import rbind /dev /dev
			import proc proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos/gentoo /var/db/repos/gentoo
			import rbind /var/cache/distfiles /var/cache/distfiles
			export symlink /var/cache/binpkgs packages
			export symlink / build`)
		augmentedConfigExports := append(typicalConfigExports,
			NeededMountType{Mount: "build", Source: "/", Fstype: "symlink"})
		li, err = ReadLayerFile(layerfilePath, true)
		if err != nil {
			t.Fatalf("augmented layout: got %s", err)
		}
		checkSameLayerinfo(t, Layerinfo{
			ConfigMounts: typicalConfigMounts,
			ConfigExports: augmentedConfigExports,
			Mounts: emptyFsMounts,
		}, li, "augmented layout")

		td.WriteFile(layerfileName,
			`# Comment
			base basic
			import rbind /dev /dev
			import proc proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos/gentoo /var/db/repos/gentoo
			import rbind /var/cache/distfiles /var/cache/distfiles
			export symlink /var/cache/binpkgs packages`)
		li, err = ReadLayerFile(layerfilePath, true)
		if err != nil {
			t.Fatalf("base specified: got %s", err)
		}
		checkSameLayerinfo(t, Layerinfo{
			Base: "basic",
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports,
			Mounts: emptyFsMounts,
		}, li, "base specified")

		td.WriteFile(layerfileName,
			`import rbind /dev /dev
			input rbind /here /there
			import proc proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos/gentoo /var/db/repos/gentoo
			import rbind /var/cache/distfiles /var/cache/distfiles
			export symlink /var/cache/binpkgs packages`)
		li, err = ReadLayerFile(layerfilePath, true)
		checkErrorByMessage(t, err, "Unknown layerconf keyword 'input' in " +
			layerfilePath + " line 2", "unknown keyword")

		td.WriteFile(layerfileName,
			`base
			import rbind /dev /dev
			import proc proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos/gentoo /var/db/repos/gentoo
			import rbind /var/cache/distfiles /var/cache/distfiles
			export symlink /var/cache/binpkgs packages`)
		li, err = ReadLayerFile(layerfilePath, true)
		checkErrorByMessage(t, err, "No base specified in " +
			layerfilePath + " line 1", "missing base name")

		td.WriteFile(layerfileName,
			`base b.dir
			import rbind /dev /dev
			import proc proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos/gentoo /var/db/repos/gentoo
			import rbind /var/cache/distfiles /var/cache/distfiles
			export symlink /var/cache/binpkgs packages`)
		li, err = ReadLayerFile(layerfilePath, true)  // validity checked at later stage
		checkErrorByMessage(t, err, "", "invalid layer name")
		checkSameLayerinfo(t, Layerinfo{
			Base: "b.dir",
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports,
			Mounts: emptyFsMounts,
		}, li, "invalid layer name")

		td.WriteFile(layerfileName,
			`import rbind /dev
			import proc proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos/gentoo /var/db/repos/gentoo
			import rbind /var/cache/distfiles /var/cache/distfiles
			export symlink /var/cache/binpkgs packages`)
		li, err = ReadLayerFile(layerfilePath, true)
		checkErrorByMessage(t, err, "Incomplete import specification in " + layerfilePath +
			" line 1", "2-argument import")

		td.WriteFile(layerfileName,
			`import rbind /dev /dev
			import proc proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos/gentoo /var/db/repos/gentoo
			import rbind /var/cache/distfiles /var/cache/distfiles
			export symlink /var/cache/binpkgs`)
		li, err = ReadLayerFile(layerfilePath, true)
		checkErrorByMessage(t, err, "Incomplete export specification in " +
			layerfilePath + " line 6", "2-argument export")

		li, err = ReadLayerFile(layerfilePath, false)
		checkErrorByMessage(t, err, "", "2-argument export; soft error")
		checkSameLayerinfo(t, Layerinfo{
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports[:0],
			Messages: []string{"Incomplete export specification in " +
				layerfilePath + " line 6"},
			Mounts: emptyFsMounts,
			State: 1,
		}, li, "2-argument export; soft error")

		td.WriteFile(layerfileName,
			`base
			import rbind /dev /dev
			import proc proc /proc
			import rbind /sys /sys
			import rbind /var/db/repos/gentoo /var/db/repos/gentoo
			import rbind /var/cache/distfiles /var/cache/distfiles
			export symlink /var/cache/binpkgs`)
		li, err = ReadLayerFile(layerfilePath, false)
		checkErrorByMessage(t, err, "", "no base + 2-argument export; soft error")
		checkSameLayerinfo(t, Layerinfo{
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports[:0],
			Messages: []string{
				"No base specified in " + layerfilePath + " line 1",
				"Incomplete export specification in " +
					layerfilePath + " line 7"},
			Mounts: emptyFsMounts,
			State: 1,
		}, li, "no-base + 2-argument export; soft error")

		td.WriteFile(layerfileName,
			`# Comment
			base fundamento
			import rbind /dev /dev
			import proc proc /proc
			import rbind /sys /sys
			base fundamento extra items ignored
			import rbind /var/db/repos/gentoo /var/db/repos/gentoo
			import rbind /var/cache/distfiles /var/cache/distfiles
			export symlink /var/cache/binpkgs packages`)
		li, err = ReadLayerFile(layerfilePath, true)
		checkErrorByMessage(t, err, "", "multiple bases, same value")
		checkSameLayerinfo(t, Layerinfo{
			Base: "fundamento",
			ConfigMounts: typicalConfigMounts,
			ConfigExports: typicalConfigExports,
			Mounts: emptyFsMounts,
		}, li, "multiple bases, same value")

		td.WriteFile(layerfileName,
			`# Comment
			base fundament
			import rbind /dev /dev
			import proc proc /proc
			import rbind /sys /sys
			base fundamento
			import rbind /var/db/repos/gentoo /var/db/repos/gentoo
			import rbind /var/cache/distfiles /var/cache/distfiles
			export symlink /var/cache/binpkgs packages`)
		li, err = ReadLayerFile(layerfilePath, true)
		checkErrorByMessage(t, err, "New conflicting setting of base property in " +
			layerfilePath + " line 6", "multiple bases, same value")
	}) {
		return
	}

	if !t.Run("write_layerfile", func (t *testing.T) {
		typicalConfigMounts := []NeededMountType{
			{Mount: "/dev", Source: "/dev", Fstype: "rbind"},
			{Mount: "/proc", Source: "proc", Fstype: "proc"},
			{Mount: "/sys", Source: "/sys", Fstype: "rbind"},
			{Mount: "/var/db/repos/gentoo", Source: "/var/db/repos/gentoo",
				Fstype: "rbind"},
			{Mount: "/var/cache/distfiles", Source: "/var/cache/distfiles",
				Fstype: "rbind"},
		}
		typicalConfigExports := []NeededMountType{
			{Mount: "packages", Source: "/var/cache/binpkgs", Fstype: "symlink"},
		}

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
import proc proc /proc
import rbind /sys /sys
import rbind /var/db/repos/gentoo /var/db/repos/gentoo
import rbind /var/cache/distfiles /var/cache/distfiles

export symlink /var/cache/binpkgs packages
`, "basic file")

		li.ConfigMounts = append(typicalConfigMounts, NeededMountType{
			Mount: "/mnt/common", Source: "/root/common", Fstype: "rbind"})
		err = WriteLayerfile(layerfilePath, &li)
		td.checkExpectedFileContents(t, err, layerfileName,
`import rbind /dev /dev
import proc proc /proc
import rbind /sys /sys
import rbind /var/db/repos/gentoo /var/db/repos/gentoo
import rbind /var/cache/distfiles /var/cache/distfiles
import rbind /root/common /mnt/common

export symlink /var/cache/binpkgs packages
`, "extra import")
	}) {
		return
	}
}

