// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package stage

import (
	"os"
	"path"
	"time"
	"strconv"
	"os/user"
	"io/ioutil"

	"testing"
)


type Tmpdir struct {
	rootdir string
}


func NewTmpdir() (*Tmpdir, error) {
	name, err := ioutil.TempDir("", "layercake_stage")
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


func (t *Tmpdir) Symlink(target, linkname string) error {
	linkpath := t.Path(linkname)
	return os.Symlink(target, linkpath)
}


func (t *Tmpdir) Link(target, linkname string) error {
	linkpath := t.Path(linkname)
	return os.Link(target, linkpath)
}


func (t *Tmpdir) ReadFile(filename string) (string, error) {
	pathname := t.Path(filename)
	buf, err := ioutil.ReadFile(pathname)
	return string(buf), err
}



const (
	entryType_file = iota
	entryType_dir
	entryType_symlink
	entryType_hardlink
)


type entryEntry struct {
	etype int
	name, value string
}


var entrySet1 []entryEntry = []entryEntry {
	entryEntry{entryType_dir, "/etc/xml", ""},
	entryEntry{entryType_file, "/etc/xml/catalog", "<xml/>"},
	entryEntry{entryType_file, "/etc/xml/docbook", "<xml/>"},
	entryEntry{entryType_dir, "/usr/share/sddm/conf.d", ""},
	entryEntry{entryType_file, "/usr/share/sddm/conf.d/default.conf", ""},
}


func (td *Tmpdir) populateTestTree(entries []entryEntry) error {
	var err error
	for _, entry := range entries {
		switch entry.etype {
		case entryType_file:
			err = td.WriteFile(entry.name, entry.value)
		case entryType_dir:
			err = td.Mkdir(entry.name)
		case entryType_symlink:
			err = td.Symlink(td.Path(entry.value), entry.name)
		case entryType_hardlink:
			err = td.Link(td.Path(entry.value), entry.name)
		}
	}
	return err
}


type curEnvironment struct {
	gid, uid uint32
	unixTime int64
}


func getCurEnvironment() (curEnvironment, error) {
	usr, err := user.Current()
	if err != nil {
		return curEnvironment{}, err
	}
	gid, _ := strconv.Atoi(usr.Gid)
	uid, _ := strconv.Atoi(usr.Uid)
	return curEnvironment{
		gid: uint32(gid),
		uid: uint32(uid),
	}, nil
}


func TestManage(t *testing.T) {
	curEnv, err := getCurEnvironment()
	if err != nil {
		t.Fatal(err)
	}

	td, err := NewTmpdir()
	if err != nil {
		t.Fatal(err)
	}
	defer td.Cleanup()

	curEnv.unixTime = time.Now().Unix()
	err = td.populateTestTree(entrySet1)
	if err != nil {
		t.Fatal(err)
	}
	_ = t.Run("set1", func (t *testing.T) {
		testSet1(t, curEnv, td, entrySet1)
	})
}


func (li *lineInfo) g_env(env curEnvironment) *lineInfo {
	return li.g_(env.gid)
}

func (li *lineInfo) u_env(env curEnvironment) *lineInfo {
	return li.u_(env.uid)
}

func (li *lineInfo) gu_env(env curEnvironment) *lineInfo {
	return li.gu_(env.gid, env.uid)
}


func liName(tp uint8) string {
	names := []string{"tbd", "dir", "file", "link", "", "dev"}
	if int(tp) < len(names) {
		return names[tp]
	}
	return "???"
}


func testSet1(t *testing.T, curEnv curEnvironment, td *Tmpdir, entries []entryEntry) {
	fileList := &FileList{rootDir: td.rootdir, inodes: map[devIno]int32{}}
	for _, tst := range []struct {before, expect *lineInfo; errmsg string} {
		{mkli("dir", "/etc/xml"),
		 mkli("dir", "/etc/xml").src(td.Path("/etc/xml")).gu_env(curEnv),
			""},
		{mkli("file", "/etc/xml/catalog"),
		 mkli("file", "/etc/xml/catalog").src(td.Path("/etc/xml/catalog")).gu_env(curEnv),
			""},
		{mkli("dir", "/var/lib/new"),
		 mkli("dir", "/var/lib/new").src(td.Path("/var/lib/new")), ""},
		{mkli("file", "/etc/nogo"), nil,
			"file " + td.Path("/etc/nogo") + " does not exist (source of /etc/nogo)"},
		{mkli("file", "/etc/nogo").skipAbsent(), nil, ""},
		{mkli("file", "/etc/sddm.conf").
			src("$$stageroot/usr/share/sddm/conf.d/default.conf"),
			mkli("file", "/etc/sddm.conf").
			src(td.rootdir + "/usr/share/sddm/conf.d/default.conf").gu_env(curEnv), ""},
	 } {
		 fileList.entryMap = map[string]lineInfo{}
		 desc := liName(tst.before.ltype) + " " + tst.before.name
		 err := fileList.addFiles(*tst.before)
		 testResult, exists := fileList.entryMap[tst.before.name]
		 if err != nil {
			 if err.Error() != tst.errmsg {
				 t.Errorf("got unexpected error %s for [%s]", err, desc)
			 }
		 } else if len(tst.errmsg) > 0 {
			 t.Errorf("unexpected success for [%s], expected error %s", desc, err)
		 }
		 if exists {
			 if tst.expect == nil {
				 t.Errorf("unexpectedly accepted [%s]", desc)
			 } else {
				 testEntry(t, desc, tst.expect, testResult)
			 }
		 } else if tst.expect != nil {
			 t.Errorf("unexpectedly rejected [%s]", desc)
		 }
	 }
}

