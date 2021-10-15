package manage


import (
	"os"
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
	err error
}

func NewTmpdir(patt string) (*Tmpdir, error) {
	name, err := ioutil.TempDir("", patt)
	return &Tmpdir{name, nil}, err
}

func (t *Tmpdir) Cleanup() {
	os.RemoveAll(t.rootdir)
}

func (t *Tmpdir) Err() error {
	return t.err
}

func (t *Tmpdir) Path(name string) string {
	return path.Join(t.rootdir, name)
}

func (t *Tmpdir) Mkdir(dirname string) string {
	pathname := t.Path(dirname)
	if t.err == nil {
		t.err = os.MkdirAll(pathname, 0755)
	}
	return pathname
}

func (t *Tmpdir) WriteFile(filename, contents string) string {
	pathname := t.Path(filename)
	if t.err == nil {
		t.err = ioutil.WriteFile(pathname, []byte(contents), 0644)
	}
	return pathname
}

func (t *Tmpdir) ReadFile(filename string) string {
	pathname := t.Path(filename)
	var buf []byte
	if t.err == nil {
		buf, t.err = ioutil.ReadFile(pathname)
	}
	return string(buf)
}

func (t *Tmpdir) Remove(filename string) {
	pathname := t.Path(filename)
	os.Remove(pathname)
}

func (t *Tmpdir) IsFile(filename string) bool {
	pathname := t.Path(filename)
	return fs.IsFile(pathname)
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


func TestInit(t *testing.T) {
	fs.MessageWriter = capturingMessageWriter
	td, err := NewTmpdir("layercake_layer_init")
	if err != nil {
		t.Fatal(err)
	}
	defer td.Cleanup()

	cfg := td.MakeConfigTypeObj()

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
}

