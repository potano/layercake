// Copyright © 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import (
	"sort"
	"time"
        "strings"
	"syscall"

        "testing"
)

func TestParseStDev(T *testing.T) {
	for _, tst := range []struct{s string; ma int; mn int; e string} {
		{"3:4", 3, 4, ""},
		{"0:1", 0, 1, ""},
		{"0:12", 0, 12, ""},
		{"80:5", 80, 5, ""},
		{"80:", 0, 0, "bad major:minor format: 80:"},
		{"102:255", 102, 255, ""},
		{":256", 0, 0, "bad major:minor format: :256"},
		{"42", 0, 0, "bad major:minor format: 42"},
		{":", 0, 0, "bad major:minor format: :"},
		{"", 0, 0, "bad major:minor format: "},
		{"1:2:3", 0, 0, "bad major:minor format: 1:2:3"},
	} {
		major, minor, err := ParseMajorMinorString(tst.s)
		if err != nil {
			if len(tst.e) > 0 && err.Error() != tst.e {
				T.Fatalf("parsing %s: expected error %s, got %s", tst.s, tst.e, err)
			} else if len(tst.e) == 0 {
				T.Fatalf("parsing %s: got unexpected error %s", tst.s, err)
			}
		} else if len(tst.e) > 0 {
			T.Fatalf("parsing %s: expected to fail with message %s", tst.s, tst.e)
		} else if major != tst.ma || minor != tst.mn {
			T.Fatalf("parsing %s: wanted major=%d, minor=%d, got %d : %d", tst.s,
				tst.ma, tst.mn, major, minor)
		}
	}
}


func TestMajorMinorToString(T *testing.T) {
	for _, tst := range []struct{major, minor int; s string} {
		{1, 3, "1:3"},
	} {
		str := MajorMinorToString(tst.major, tst.minor)
		if str != tst.s {
			T.Fatalf("composing major=%d, minor=%d: expected '%s', got '%s'",
				tst.major, tst.minor, tst.s, str)
		}
	}
}


func TestMajorMinorToStDev(T *testing.T) {
	for _, tst := range []struct{major, minor int; dev uint64; err string} {
		{1, 3, 259, ""},
		{-1, 3, 0, "major device number -1 is negative"},
		{5, -1, 0, "minor device number -1 is not between 0 and 255"},
		{10, 255, 2560 + 255, ""},
		{10, 256, 0, "minor device number 256 is not between 0 and 255"},
	} {
		st_dev, err := MajorMinorToStDev(tst.major, tst.minor)
		if err != nil {
			if len(tst.err) > 0 && err.Error() != tst.err {
				T.Fatalf("converting major=%d, minor=%d: expected '%s', got '%s'",
					tst.major, tst.minor, tst.err, err)
			} else if len(tst.err) == 0 {
				T.Fatalf("converting major=%d, minor=%d: unexpected error '%s'",
					tst.major, tst.minor, err)
			}
		} else if len(tst.err) > 0 {
			T.Fatalf("converting major=%d, minor=%d: expected error '%s'",
				tst.major, tst.minor, tst.err)
		} else if st_dev != tst.dev {
			T.Fatalf("converting major=%d, minor=%d: expected 0x%X, got 0x%X",
				tst.major, tst.minor, tst.dev, st_dev)
		}
	}
}


func TestStDevToMajorMinor(T *testing.T) {
	for _, tst := range []struct{st_dev uint64; major, minor int} {
		{2560 + 255, 10, 255},
		{2, 0, 2},
	} {
		major, minor := StDevToMajorMinor(tst.st_dev)
		if major != tst.major || minor != tst.minor {
			T.Fatalf("decomposing 0x%X: expected major=%d, minor=%d, got %d : %d",
				tst.st_dev, tst.major, tst.minor, major, minor)
		}
	}
}


func TestParseMountOptions(T *testing.T) {
	for _, tst := range []struct{s string; want map[string]string} {
		{"", map[string]string{}},
		{"ro", map[string]string{"ro": ""}},
		{"ro,", map[string]string{"ro": ""}},
		{"rw,nodev", map[string]string{"rw": "", "nodev": ""}},
		{"rw,,nodev", map[string]string{"rw": "", "nodev": ""}},
		{"uid=100,gid=50", map[string]string{"uid": "100", "gid": "50"}},
		{"lowerdir=/abc,upperdir=/def,workdir=/ghi", map[string]string{
			"lowerdir": "/abc",
			"upperdir": "/def",
			"workdir": "/ghi"}},
	} {
		got := ParseMountOptions(tst.s)
		if len(got) != len(tst.want) {
			T.Fatalf("expected %d-element map when parsing \"%s\", got %d elements",
				len(tst.want), tst.s, len(got))
		}
		for key, val := range tst.want {
			gotval, have := got[key]
			if !have {
				T.Fatalf("expected map of \"%s\" to have key '%s'", tst.s, key)
			}
			if gotval != val {
				T.Fatalf("expected value of '%s' key in \"%s\" to be '%s'" +
					", got %s", key, tst.s, val, gotval)
			}
		}
	}
}


func TestBasicFileInodeRead(T *testing.T) {
	for _, tst := range []struct{
		content string; start int64; bufsiz int; want, err string} {
		{"", 0, 0, "", ""},
		{"", -3, 0, "", "invalid argument"},
		{"abc", 0, 3, "abc", ""},
		{"abcd", 1, 2, "bc", ""},
		{"abcd", 1, 3, "bcd", ""},
		{"abcd", 1, 4, "bcd", ""},
		{"abcd", 2, 4, "cd", ""},
		{"abcd", 3, 4, "d", ""},
		{"abcd", 4, 4, "", ""},
		{"abcd", 5, 4, "", ""},
	} {
		inode := &mfsFileInodeBase{}
		inode.init(nodeTypeFile)
		inode.contents = []byte(tst.content)
		buf := make([]byte, tst.bufsiz)
		ct, err := inode.readFile(buf, tst.start)
		if err != nil {
			if len(tst.err) > 0 && tst.err != err.Error() {
				T.Fatalf("read buf from %d: expected error '%s', got '%s'",
					tst.start, tst.err, err)
			} else if len(tst.err) == 0 {
				T.Fatalf("read buf from %d: unexpected error '%s'", tst.start, err)
			}
		} else if len(tst.err) > 0 {
			T.Fatalf("read buf from %d: expected error '%s'", tst.start, tst.err)
		} else if ct != len(tst.want) {
			T.Fatalf("read buf from %d: expected %d characters, got %d", tst.start,
				len(tst.want), ct)
		} else {
			got := string(buf[:ct])
			if got != tst.want {
				T.Fatalf("read buf from %d: expected to read '%s', got '%s'",
					tst.start, tst.want, got)
			}
		}
	}
}


func TestBasicFileInodeWrite(T *testing.T) {
	for _, tst := range []struct{
		existing, scribenda string; start int64; written int; after, err string} {
		{"", "", 0, 0, "", ""},
		{"", "", -3, 0, "", "invalid argument"},
		{"", "a", 0, 1, "a", ""},
		{"abc", "", 0, 0, "abc", ""},
		{"abc", "", 1, 0, "abc", ""},
		{"abc", "", 2, 0, "abc", ""},
		{"abc", "", 3, 0, "abc", ""},
		{"abc", "A", 0, 1, "Abc", ""},
		{"abc", "AB", 0, 2, "ABc", ""},
		{"abc", "ABC", 0, 3, "ABC", ""},
		{"abc", "ABCD", 0, 4, "ABCD", ""},
		{"abc", "B", 1, 1, "aBc", ""},
		{"abc", "BC", 1, 2, "aBC", ""},
		{"abc", "BCD", 1, 3, "aBCD", ""},
		{"abc", "C", 2, 1, "abC", ""},
		{"abc", "CD", 2, 2, "abCD", ""},
		{"abc", "D", 3, 1, "abcD", ""},
		{"abc", "DE", 3, 2, "abcDE", ""},
		{"abc", "", 4, 0, "abc\000", ""},
		{"abc", "E", 4, 1, "abc\000E", ""},
		{"abc", "GH", 6, 2, "abc\000\000\000GH", ""},
	} {
		inode := &mfsFileInodeBase{}
		inode.init(nodeTypeFile)
		inode.contents = []byte(tst.existing)
		ct, err := inode.writeFile([]byte(tst.scribenda), tst.start)
		if err != nil {
			if len(tst.err) > 0 && tst.err != err.Error() {
				T.Fatalf("write buf at %d: expected error '%s', got '%s'",
					tst.start, tst.err, err)
			} else if len(tst.err) == 0 {
				T.Fatalf("write buf at %d: unexpected error '%s'", tst.start, err)
			}
		} else if len(tst.err) > 0 {
			T.Fatalf("write buf at %d: expected error '%s'", tst.start, tst.err)
		} else if ct != tst.written {
			T.Fatalf("write buf at %d: expected %d characters, got %d", tst.start,
				tst.written, ct)
		} else {
			newContent := string(inode.contents)
			if newContent != tst.after {
				T.Fatalf("write buf at %d: expected to read '%s', got '%s'",
					tst.start, tst.after, newContent)
			}
		}
	}
}


func TestParseAbspath(T *testing.T) {
	for _, test := range []struct{path string; want []string; pth string} {
		{"", []string{}, "/"},
		{"/", []string{}, "/"},
		{"a", []string{"a"}, "/a"},
		{"a/b", []string{"a", "b"}, "/a/b"},
		{"/a", []string{"a"}, "/a"},
		{"/a/", []string{"a"}, "/a"},
		{"/a/b", []string{"a", "b"}, "/a/b"},
		{"//", []string{}, "/"},
		{"//a", []string{"a"}, "/a"},
		{"a//b", []string{"a", "b"}, "/a/b"},
		{".", []string{}, "/"},
		{"./a", []string{"a"}, "/a"},
		{"a/.", []string{"a"}, "/a"},
		{"a/./b", []string{"a", "b"}, "/a/b"},
		{"a/./b/.", []string{"a", "b"}, "/a/b"},
		{"..", []string{}, "/"},
		{"../..", []string{}, "/"},
		{"../a/..", []string{}, "/"},
		{"../a/b/..", []string{"a"}, "/a"},
		{"../a/../..", []string{}, "/"},
		{"../a/../../", []string{}, "/"},
		{"../a/./..", []string{}, "/"},
		{"../a/b/./..", []string{"a"}, "/a"},
		{"../a/./../..", []string{}, "/"},
		{"../a/./../../", []string{}, "/"},
		{"a/..", []string{}, "/"},
		{"a/b/../c", []string{"a", "c"}, "/a/c"},
		{"a/b/c/../d/e/../../..", []string{"a"}, "/a"},
		{"a/b/c/../d/e/../../../..", []string{}, "/"},
		{"./a/b/c/../d/e/../../../..", []string{}, "/"},
		{"/..", []string{}, "/"},
		{"/a", []string{"a"}, "/a"},
		{"/a/b", []string{"a", "b"}, "/a/b"},
		{"/../..", []string{}, "/"},
		{"/../a/..", []string{}, "/"},
		{"/../a/b/..", []string{"a"}, "/a"},
		{"/../a/../..", []string{}, "/"},
		{"/../a/../../", []string{}, "/"},
		{"/../a/./..", []string{}, "/"},
		{"/../a/b/./..", []string{"a"}, "/a"},
		{"/../a/./../..", []string{}, "/"},
		{"/../a/./../../", []string{}, "/"},
	} {
		parsed := newAbspath(test.path)
		same := len(parsed) == len(test.want)
		if same {
			for i, val := range test.want {
				if parsed[i] != val {
					same = false
					break
				}
			}
		}
		if !same {
			T.Fatalf("parsed path '%s': wanted %#v, got %#v", test.path,
				test.want, parsed)
		}
		constructed := parsed.toString()
		if constructed != test.pth {
			T.Fatalf("parsed path '%s': wanted absolute path '%s', got '%s'",
				test.path, test.pth, constructed)
		}
	}
}


func TestStDevForDeviceName(T *testing.T) {
	for _, tst := range []struct{name string; major, minor int} {
		{"/dev/sda1", 8, 1},
		{"/dev/hda3", 3, 3},
		{"/dev/sdb2", 8, 18},
		{"/dev/loop3", 7, 3},
		{"/dev/sdp4", 8, 244},
	} {
		st_dev := stDevForDeviceType(tst.name)
		major, minor := StDevToMajorMinor(st_dev)
		if major != tst.major || minor != tst.minor {
			T.Fatalf("interprtation of %s: expected %d:%d, got %d:%d", tst.name,
				tst.major, tst.minor, major, minor)
		}
	}
}


func TestCreateEmptyNamespace(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	if mos.ns == nil {
		T.Fatal("did not instantiate namespace")
	}
	if mos.pid != 1 {
		T.Fatalf("expected pid=1, got %d", mos.pid)
	}
	if mos.euid != 0 || mos.gid != 0 {
		T.Fatalf("expected euid=0 and gid=0, got euid=%d, gid=%d", mos.euid, mos.gid)
	}
	if mos.root == nil || mos.cwd == nil {
		T.Fatal("failed to set either the process' root or current directories")
	}
	ns := mos.ns
	if len(ns.devices) != 1 {
		T.Fatalf("Expected 1 device, got %d", len(ns.devices))
	}
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{ {nodeTypeDir, 0755, 0, 0, 1, ""} } } } )
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestResolveRootPath(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	if mos.ns == nil {
		T.Fatal("did not instantiate namespace")
	}
	mount, dirInode, inode, name, path, err := mos.resolvePath(mos.cwd, "/", true)
	if err != nil {
		T.Fatalf("mos.resolvePath('/'): %s", err)
	}
	if inode == nil {
		T.Fatal("mos.resolvePath('/'): no inode found")
	}
	if mount == nil {
		T.Fatal("mos.resolvePath('/'): no mount found")
	}
	if mount != mos.ns.mounts[0] {
		T.Fatal("mos.resolvePath('/'): incorrect mount found")
	}
	if dirInode != nil {
		T.Fatal("mos.resolvePath('/'): dirInode is not nil")
	}
	if len(name) != 0 {
		T.Fatalf("mos.resolvePath('/'): expected empty name, got '%s'", name)
	}
	if len(path) != 0 {
		T.Fatalf("mos.resolvePath('/'): expected empty remaining path, got %#v", path)
	}
	if inode.ino() != 1 || inode.dev() != mount.st_dev {
		T.Fatal("mos.resolvePath('/'): incorrect inode found")
	}
}


func TestStatRootPath(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	var stat syscall.Stat_t
	err = mos.SyscallStat("/", &stat)
	if err != nil {
		T.Fatalf("%s statting root directory", err)
	}
	checkStat(T, "", stat, syscall.Stat_t{
		Dev: mos.ns.mounts[0].st_dev,
		Ino: 1,
		Nlink: 1,
		Mode: syscall.S_IFDIR | syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR |
			syscall.S_IRGRP | syscall.S_IXGRP | syscall.S_IROTH | syscall.S_IXOTH,
		Uid: 0,
		Gid: 0,
		Rdev: 0,
		Size: 0,
	})
}


func TestStatCWD(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	var stat syscall.Stat_t
	err = mos.SyscallStat(".", &stat)
	if err != nil {
		T.Fatalf("%s statting current directory", err)
	}
	checkStat(T, "", stat, syscall.Stat_t{
		Dev: mos.ns.mounts[0].st_dev,
		Ino: 1,
		Nlink: 1,
		Mode: syscall.S_IFDIR | syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR |
			syscall.S_IRGRP | syscall.S_IXGRP | syscall.S_IROTH | syscall.S_IXOTH,
		Uid: 0,
		Gid: 0,
		Rdev: 0,
		Size: 0,
	})
}


func TestCreateFileAtRoot(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.Create("testfile")
	if err != nil {
		T.Fatal(err.Error())
	}
	if file == nil {
		T.Fatal("no open-file object returned")
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfile=2"},
			{nodeTypeFile, 0644, 0, 0, 1, ""}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, 0x241, 2, "testfile", 0, false, true, false, "/testfile"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 2 },
			},
		},
	})
}


func TestWriteToFileAtRoot(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.Create("testfile")
	if err != nil {
		T.Fatalf("error on creating file: %s", err)
	}
	buf := []byte("write this")
	ct, err := file.Write(buf)
	if err != nil {
		T.Fatalf("error on writing file: %s", err)
	}
	if ct != len(buf) {
		T.Fatalf("expected to write %d bytes; wrote %d\n", len(buf), ct)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfile=2"},
			{nodeTypeFile, 0644, 0, 0, 1, "write this"}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, 0x241, 2, "testfile", 10, false, true, false, "/testfile"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 2 },
			},
		},
	})
}


func TestReadFromUnclosedJustCreatedFile(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.Create("testfile")
	if err != nil {
		T.Fatalf("error on creating file: %s", err)
	}
	buf := make([]byte, 10)
	_, err = file.Read(buf)
	if err == nil {
		T.Fatalf("expected error on reading file")
	}
	want := "read testfile: permission denied"
	if err.Error() != want {
		T.Fatalf("expected error '%s' on reading file; got '%s'", want, err)
	}
}


func TestWriteToFileAtRootAndClose(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.Create("testfile")
	if err != nil {
		T.Fatalf("error on creating file: %s", err)
	}
	buf := []byte("write this")
	ct, err := file.Write(buf)
	if err != nil {
		T.Fatalf("error on writing file: %s", err)
	}
	if ct != len(buf) {
		T.Fatalf("expected to write %d bytes; wrote %d\n", len(buf), ct)
	}
	err = file.Close()
	if err != nil {
		T.Fatalf("error closing file: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfile=2"},
			{nodeTypeFile, 0644, 0, 0, 1, "write this"}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestCreateWriteCloseThenRead(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.Create("testfile")
	if err != nil {
		T.Fatalf("error on creating file: %s", err)
	}
	buf := []byte("write this")
	ct, err := file.Write(buf)
	if err != nil {
		T.Fatalf("error on writing file: %s", err)
	}
	if ct != len(buf) {
		T.Fatalf("expected to write %d bytes; wrote %d\n", len(buf), ct)
	}
	err = file.Close()
	if err != nil {
		T.Fatalf("error closing file: %s", err)
	}
	file, err = mos.Open("testfile")
	if err != nil {
		T.Fatalf("error on opening file: %s", err)
	}
	buf = make([]byte, 5)
	ct, err = file.Read(buf)
	if err != nil {
		T.Fatalf("error reading file: %s", err)
	}
	if ct != len(buf) {
		T.Fatalf("expected to read %d bytes, got %d", len(buf), ct)
	}
	want := "write"
	if string(buf) != want {
		T.Fatalf("expected to read '%s', got '%s'", want, string(buf))
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfile=2"},
			{nodeTypeFile, 0644, 0, 0, 1, "write this"}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_RDONLY, 2, "testfile", 5, true, false, false, "/testfile"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 2 },
			},
		},
	})
}


func TestCreateNoTrunc(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.Create("testfile")
	if err != nil {
		T.Fatalf("error on creating file: %s", err)
	}
	expected := "content to use"
	buf := []byte(expected)
	ct, err := file.Write(buf)
	if err != nil {
		T.Fatalf("error on writing file: %s", err)
	}
	err = file.Close()
	if err != nil {
		T.Fatalf("error closing file: %s", err)
	}
	file, err = mos.OpenFile("testfile", O_CREATE | O_RDONLY, 0644)
	if err != nil {
		T.Fatalf("error on opening file: %s", err)
	}
	statBuf, err := file.Stat()
	if err != nil {
		T.Fatalf("error statting file: %s", err)
	}
	if statBuf.Size() != int64(ct) {
		T.Fatalf("expected file length %d, got %d", ct, statBuf.Size())
	}
	buf = make([]byte, 20)
	n, err := file.Read(buf)
	if err != nil {
		T.Fatalf("error on reading from file: %s", err)
	}
	buf = buf[:n]
	if string(buf) != expected {
		T.Fatalf("expected to read '%s', got '%s'", expected, string(buf))
	}
}


func TestAppend(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.OpenFile("testfile", O_CREATE | O_APPEND | O_RDWR, 0755)
	if err != nil {
		T.Fatalf("error on creating file: %s", err)
	}
	part1 := "First part"
	part2 := " second part"
	expected := part1 + part2
	_, err = file.Write([]byte(part1))
	if err != nil {
		T.Fatalf("error on writing file: %s", err)
	}
	pos, err := file.Seek(3, SEEK_SET)
	if err != nil {
		T.Fatalf("error on first seek: %s", err)
	}
	if pos != 3 {
		T.Fatalf("expected position 3 on seek, got %d", pos)
	}
	buf := make([]byte, 2)
	n, err := file.Read(buf)
	if err != nil {
		T.Fatalf("error on first read: %s", err)
	}
	buf = buf[:n]
	if string(buf) != "st" {
		T.Fatalf("expected to read 't ', got '%s'", string(buf))
	}
	pos, err = file.Seek(0, SEEK_CUR)
	if err != nil {
		T.Fatalf("error on second seek: %s", err)
	}
	if pos != 5 {
		T.Fatalf("expected new position to be 5, got %d", pos)
	}
	_, err = file.Write([]byte(part2))
	if err != nil {
		T.Fatalf("error on second write: %s", err)
	}
	l, err := file.Seek(0, SEEK_CUR)
	if err != nil {
		T.Fatalf("error on third seek: %s", err)
	}
	if l != int64(len(expected)) {
		T.Fatalf("expected to have position %d, got %d", len(expected), l)
	}
	_, err = file.Seek(0, SEEK_SET)
	if err != nil {
		T.Fatalf("error on fourth seek: %s", err)
	}
	buf = make([]byte, 30)
	n, err = file.Read(buf)
	if err != nil {
		T.Fatalf("error on second read: %s", err)
	}
	buf = buf[:n]
	if string(buf) != expected {
		T.Fatalf("expected to read '%s' on final read, got '%s", expected, string(buf))
	}
	pos, err = file.Seek(0, SEEK_END)
	if err != nil {
		T.Fatalf("error on fifth seek: %s", err)
	}
	if pos != int64(len(expected)) {
		T.Fatalf("expected final file size to be %d, got %d", len(expected), pos)
	}
}


func TestCreateExclusive(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.OpenFile("testfile", O_CREATE | O_EXCL | O_WRONLY, 0644)
	if err != nil {
		T.Fatalf("error on creating file: %s", err)
	}
	file.Close()
	file, err = mos.OpenFile("testfile", O_CREATE | O_EXCL | O_WRONLY, 0644)
	if err == nil {
		T.Fatal("expected error when creating existing file with O_EXCL")
	}
	want := "open testfile: file exists"
	if err.Error() != want {
		T.Fatalf("expected error '%s', got '%s'", want, err)
	}
}


func TestMkdirAtRoot(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("newdir", 0755)
	if err != nil {
		T.Fatalf("error on creating directory: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "newdir=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestFileCreationInSubdirectory(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("newdir", 0755)
	if err != nil {
		T.Fatalf("error on creating directory: %s", err)
	}
	file, err := mos.Create("/newdir/afile")
	if err != nil {
		T.Fatalf("error in creating file: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "newdir=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "afile=3"},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_CREATE | O_WRONLY | O_TRUNC, 3, "/newdir/afile", 0,
				false, true, false, "/newdir/afile"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 3 },
			},
		},
	})
	err = file.Close()
	if err != nil {
		T.Fatalf("error on closing file %s", err)
	}
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "newdir=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "afile=3"},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestFileOpenFileInSubdirectory(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("newdir", 0755)
	if err != nil {
		T.Fatalf("error on creating directory: %s", err)
	}
	file, err := mos.Create("/newdir/afile")
	if err != nil {
		T.Fatalf("error in creating file: %s", err)
	}
	file.Close()
	if err != nil {
		T.Fatalf("error closing newly created file: %s", err)
	}
	file, err = mos.Open("/newdir/afile")
	if err != nil {
		T.Fatalf("error opening file: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "newdir=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "afile=3"},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_RDONLY, 3, "/newdir/afile", 0,
				true, false, false, "/newdir/afile"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 3 },
			},
		},
	})
	err = file.Close()
	if err != nil {
		T.Fatalf("error on closing file opened for reading %s", err)
	}
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "newdir=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "afile=3"},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestCreateSymlinkInRootDirectory(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Symlink("nowhere", "newlink")
	if err != nil {
		T.Fatalf("error on creating symlink: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "newlink=2"},
			{nodeTypeLink, 0777, 0, 0, 1, "nowhere"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestOpenSymlinkInRootDirectory(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Symlink("to_here", "newlink")
	if err != nil {
		T.Fatalf("error on creating symlink: %s", err)
	}
	file, err := mos.Create("to_here")
	if err != nil {
		T.Fatalf("error creating target file: %s", err)
	}
	file.Close()
	file, err = mos.Open("newlink")
	if err != nil {
		T.Fatalf("error opening file via symlink: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "newlink=2\nto_here=3"},
			{nodeTypeLink, 0777, 0, 0, 1, "to_here"},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_RDONLY, 3, "newlink", 0, true, false, false, "/to_here"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 3 },
			},
		},
	})
}


func TestSymlinkErrors(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.Create("blocker")
	if err != nil {
		T.Fatalf("error creating file: %s", err)
	}
	file.Close()
	want := "symlink to_here blocker: file exists"
	err = mos.Symlink("to_here", "blocker")
	if err == nil {
		T.Fatal("expected error upon creating symlink 'blocker'")
	}
	if err.Error() != want {
		T.Fatalf("expected error '%s', on creating symlink 'blocker', got '%s'", want, err)
	}
	want = "symlink to_here nosuchdir/name: no such file or directory"
	err = mos.Symlink("to_here", "nosuchdir/name")
	if err == nil {
		T.Fatal("expected error upon creating symlink 'nosuchdir/name'")
	}
	if err.Error() != want {
		T.Fatalf("expected error '%s', on creating symlink 'nosuchdir/name', got '%s'",
			want, err)
	}
}


func TestSimpleChdir(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("newdir", 0755)
	if err != nil {
		T.Fatalf("error on creating directory: %s", err)
	}
	err = mos.Chdir("newdir")
	if err != nil {
		T.Fatalf("error changing directory: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "newdir=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 2, "newdir", 0, true, false, true, "/newdir"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 2 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestChdirAndCreate(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("newdir", 0755)
	if err != nil {
		T.Fatalf("error on creating directory: %s", err)
	}
	err = mos.Chdir("newdir")
	if err != nil {
		T.Fatalf("error changing directory: %s", err)
	}
	_, err = mos.Create("newfile")
	if err != nil {
		T.Fatalf("error creating file in new directory: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "newdir=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "newfile=3"},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 2, "newdir", 0, true, false, true, "/newdir"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_CREATE | O_WRONLY | O_TRUNC, 3, "newfile", 0, false, true, false,
				"/newdir/newfile"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 2 },
				{-1, "0:2", 1 },
				{0, "0:2", 3 },
			},
		},
	})
}


func TestMkdirAll(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.MkdirAll("one/two/three", 0711)
	if err != nil {
		T.Fatalf("error on MkdirAll: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "one=2"},
			{nodeTypeDir, 0711, 0, 0, 1, "two=3"},
			{nodeTypeDir, 0711, 0, 0, 1, "three=4"},
			{nodeTypeDir, 0711, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestMkdirAllThenOpenFileThere(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.MkdirAll("one/two/three", 0711)
	if err != nil {
		T.Fatalf("error on MkdirAll: %s", err)
	}
	file, err := mos.Create("one/two/three/file")
	if err != nil {
		T.Fatalf("error on creating file: %s", err)
	}
	if file.inode.ino() != 5 {
		T.Fatalf("expected file to be created with inum=5, got %d", file.inode.ino())
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "one=2"},
			{nodeTypeDir, 0711, 0, 0, 1, "two=3"},
			{nodeTypeDir, 0711, 0, 0, 1, "three=4"},
			{nodeTypeDir, 0711, 0, 0, 1, "file=5"},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_CREATE | O_WRONLY | O_TRUNC, 5, "one/two/three/file", 0,
				false, true, false, "/one/two/three/file"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 5 },
			},
		},
	})
}


func TestMkdirAllThenCdToParentThenOpenFile(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.MkdirAll("one/two/three", 0711)
	if err != nil {
		T.Fatalf("error on MkdirAll: %s", err)
	}
	err = mos.Chdir("one/two")
	if err != nil {
		T.Fatalf("error changing directory: %s", err)
	}
	file, err := mos.Create("three/file")
	if err != nil {
		T.Fatalf("error on creating file: %s", err)
	}
	if file.inode.ino() != 5 {
		T.Fatalf("expected file to be created with inum=5, got %d", file.inode.ino())
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "one=2"},
			{nodeTypeDir, 0711, 0, 0, 1, "two=3"},
			{nodeTypeDir, 0711, 0, 0, 1, "three=4"},
			{nodeTypeDir, 0711, 0, 0, 1, "file=5"},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 3, "one/two", 0, true, false, true, "/one/two"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_CREATE | O_WRONLY | O_TRUNC, 5, "three/file", 0,
				false, true, false, "/one/two/three/file"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 3 },
				{-1, "0:2", 1 },
				{0, "0:2", 5 },
			},
		},
	})
}


func TestMkdirAllOverSymlink(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.MkdirAll("one/two/three", 0751)
	if err != nil {
		T.Fatalf("error on MkdirAll(\"one/two/three\"): %s", err)
	}
	err = mos.Chdir("one")
	if err != nil {
		T.Fatalf("error in Chdir(\"one\"): %s", err)
	}
	err = mos.Symlink("two/three", "tothree")
	if err != nil {
		T.Fatalf("error creating symlink: %s", err)
	}
	err = mos.MkdirAll("tothree/four", 0755)
	if err != nil {
		T.Fatalf("error on MkdirAll(\"tothree/four\"): %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "one=2"},
			{nodeTypeDir, 0751, 0, 0, 1, "tothree=5\ntwo=3"},
			{nodeTypeDir, 0751, 0, 0, 1, "three=4"},
			{nodeTypeDir, 0751, 0, 0, 1, "four=6"},
			{nodeTypeLink, 0777, 0, 0, 1, "two/three"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 2, "one", 0, true, false, true, "/one"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 2 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestBasicLinks(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	_, err = mos.Create("testfile")
	if err != nil {
		T.Fatalf("error creating file: %s", err)
	}
	err = mos.Link("testfile", "second")
	if err != nil {
		T.Fatalf("error creating link to file: %s", err)
	}
	err = mos.Mkdir("dir", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	want := "link dir todir: operation not permitted"
	err = mos.Link("dir", "todir")
	if err == nil || err.Error() != want {
		T.Fatalf("expected error '%s' in attempt to link to directory, got '%s'", want, err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dir=3\nsecond=2\ntestfile=2"},
			{nodeTypeFile, 0644, 0, 0, 2, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_CREATE | O_WRONLY | O_TRUNC, 2, "testfile", 0,
				false, true, false, "/testfile"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 2 },
			},
		},
	})
}


func TestLstatScenarios(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("abc", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	_, err = mos.Create("abc/def")
	if err != nil {
		T.Fatalf("error creating file: %s", err)
	}
	err = mos.Symlink("/abc/def", "tofile")
	if err != nil {
		T.Fatalf("error creating symlink: %s", err)
	}
	err = mos.Link("/abc/def", "hardlinked")
	if err != nil {
		T.Fatalf("error creating link: %s", err)
	}
	testPairs := func (T *testing.T, path string, wantedStat, wantedLstat syscall.Stat_t) {
		var stat syscall.Stat_t
		err := mos.SyscallStat(path, &stat)
		if err != nil {
			T.Fatalf("error statting %s: %s", path, err)
		}
		checkStat(T, "stat " + path, stat, wantedStat)
		err = mos.SyscallLstat(path, &stat)
		if err != nil {
			T.Fatalf("error lstatting %s: %s", path, err)
		}
		checkStat(T, "lstat " + path, stat, wantedLstat)
	}
	wantedRootStat := syscall.Stat_t{
		Dev: mos.ns.mounts[0].st_dev,
		Ino: 1,
		Nlink: 1,
		Mode: syscall.S_IFDIR | syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR |
			syscall.S_IRGRP | syscall.S_IXGRP | syscall.S_IROTH | syscall.S_IXOTH,
		Uid: 0,
		Gid: 0,
		Rdev: 0,
		Size: 0,
	}
	wantedDirStat := syscall.Stat_t{
		Dev: mos.ns.mounts[0].st_dev,
		Ino: 2,
		Nlink: 1,
		Mode: syscall.S_IFDIR | syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR |
			syscall.S_IRGRP | syscall.S_IXGRP | syscall.S_IROTH | syscall.S_IXOTH,
		Uid: 0,
		Gid: 0,
		Rdev: 0,
		Size: 0,
	}
	wantedFileStat := syscall.Stat_t{
		Dev: mos.ns.mounts[0].st_dev,
		Ino: 3,
		Nlink: 2,
		Mode: syscall.S_IFREG | syscall.S_IRUSR | syscall.S_IWUSR |
			syscall.S_IRGRP | syscall.S_IROTH,
		Uid: 0,
		Gid: 0,
		Rdev: 0,
		Size: 0,
	}
	wantedSymlinkStat := syscall.Stat_t{
		Dev: mos.ns.mounts[0].st_dev,
		Ino: 4,
		Nlink: 1,
		Mode: syscall.S_IFLNK | syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR |
			syscall.S_IRGRP | syscall.S_IWGRP | syscall.S_IXGRP |
			syscall.S_IROTH | syscall.S_IWOTH | syscall.S_IXOTH,
		Uid: 0,
		Gid: 0,
		Rdev: 0,
		Size: 0,
	}
	testPairs(T, "/", wantedRootStat, wantedRootStat)
	testPairs(T, ".", wantedRootStat, wantedRootStat)
	testPairs(T, "abc", wantedDirStat, wantedDirStat)
	testPairs(T, "abc/def", wantedFileStat, wantedFileStat)
	testPairs(T, "tofile", wantedFileStat, wantedSymlinkStat)
	testPairs(T, "hardlinked", wantedFileStat, wantedFileStat)
}


func TestReaddirnamesAtOnce(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("dir", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	wantFiles := []string{"red", "green", "blue", "purple", "azure", "magenta", "orange"}
	for _, nm := range wantFiles {
		file, err := mos.Create("dir/" + nm)
		if err != nil {
			T.Fatalf("error creating dir/%s: %s", nm, err)
		}
		file.Close()
	}
	sort.Strings(wantFiles)
	dir, err := mos.Open("dir")
	if err != nil {
		T.Fatalf("error opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory names: %s", err)
	}
	wantString := strings.Join(wantFiles, " ")
	gotString := strings.Join(got, " ")
	if gotString != wantString {
		T.Fatalf("expected to read file list '%s', got '%s'", wantString, gotString)
	}
}


func TestReaddirnamesByParts(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("dir", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	wantFiles := []string{"red", "green", "blue", "purple", "azure", "magenta", "orange"}
	for _, nm := range wantFiles {
		file, err := mos.Create("dir/" + nm)
		if err != nil {
			T.Fatalf("error creating dir/%s: %s", nm, err)
		}
		file.Close()
	}
	sort.Strings(wantFiles)
	dir, err := mos.Open("dir")
	if err != nil {
		T.Fatalf("error opening directory: %s", err)
	}
	groupSize := 3
	groupCount := 0
	for groupStart := 0; groupStart < len(wantFiles); groupStart += groupSize {
		groupCount++
		got, err := dir.Readdirnames(groupSize)
		if err != nil {
			T.Fatalf("error reading group %d of directory names: %s", groupCount, err)
		}
		sliceEnd := groupStart + groupSize
		if sliceEnd > len(wantFiles) {
			sliceEnd = len(wantFiles)
		}
		wantString := strings.Join(wantFiles[groupStart:sliceEnd], " ")
		gotString := strings.Join(got, " ")
		if gotString != wantString {
			T.Fatalf("group %d: expected to read file list '%s', got '%s'",
				groupCount, wantString, gotString)
		}
	}
}


func TestCreateFifo(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkfifo("testfifo", 0644)
	if err != nil {
		T.Fatal(err.Error())
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfifo=2"},
			{nodeTypeFifo, 0644, 0, 0, 1, ""}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestFifoWrite(T *testing.T) {
	//Note that these test-framework FIFO's are non-blocking
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkfifo("testfifo", 0644)
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.OpenFile("testfifo", O_WRONLY, 0)
	if err != nil {
		T.Fatalf("error opening FIFO for writing: %s", err)
	}
	want := "transfer"
	n, err := file.Write([]byte(want))
	if err != nil {
		T.Fatalf("error writing to FIFO: %s", err)
	}
	if n != len(want) {
		T.Fatalf("expected to write %d bytes to FIFO; wrote %d", len(want), n)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfifo=2"},
			{nodeTypeFifo, 0644, 0, 0, 1, want}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_WRONLY, 2, "testfifo", 0, false, true, false, "/testfifo"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 2 },
			},
		},
	})
}


func TestFifoWriteAndRead(T *testing.T) {
	//Note that these test-framework FIFO's are non-blocking
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkfifo("testfifo", 0644)
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.OpenFile("testfifo", O_WRONLY, 0)
	if err != nil {
		T.Fatalf("error opening FIFO for writing: %s", err)
	}
	want := "transfer"
	n, err := file.Write([]byte(want))
	if err != nil {
		T.Fatalf("error writing to FIFO: %s", err)
	}
	err = file.Close()
	file, err = mos.Open("testfifo")
	if err != nil {
		T.Fatalf("error reopening FIFO: %s", err)
	}
	buf := make([]byte, 15)
	n, err = file.Read(buf)
	if err != nil {
		T.Fatalf("error reading from FIFO: %s", err)
	}
	got := string(buf[:n])
	if want != got {
		T.Fatalf("expected to read '%s' from FIFO, got '%s'", want, got)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfifo=2"},
			{nodeTypeFifo, 0644, 0, 0, 1, ""}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_RDONLY, 2, "testfifo", 0, true, false, false, "/testfifo"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 2 },
			},
		},
	})
}


func TestFifoWriteAndPartialRead(T *testing.T) {
	//Note that these test-framework FIFO's are non-blocking
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkfifo("testfifo", 0644)
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.OpenFile("testfifo", O_WRONLY, 0)
	if err != nil {
		T.Fatalf("error opening FIFO for writing: %s", err)
	}
	all := "porkchop"
	want := all[:4]
	n, err := file.Write([]byte(all))
	if err != nil {
		T.Fatalf("error writing to FIFO: %s", err)
	}
	err = file.Close()
	file, err = mos.Open("testfifo")
	if err != nil {
		T.Fatalf("error reopening FIFO: %s", err)
	}
	buf := make([]byte, len(want))
	n, err = file.Read(buf)
	if err != nil {
		T.Fatalf("error reading from FIFO: %s", err)
	}
	got := string(buf[:n])
	if want != got {
		T.Fatalf("expected to read '%s' from FIFO, got '%s'", want, got)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfifo=2"},
			{nodeTypeFifo, 0644, 0, 0, 1, "chop"}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_RDONLY, 2, "testfifo", 0, true, false, false, "/testfifo"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 2 },
			},
		},
	})
}


func TestReadEmptyFifo(T *testing.T) {
	//Note that these test-framework FIFO's are non-blocking
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkfifo("testfifo", 0644)
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.Open("testfifo")
	if err != nil {
		T.Fatalf("error reopening FIFO: %s", err)
	}
	buf := make([]byte, 20)
	n, err := file.Read(buf)
	if err != nil {
		T.Fatalf("error reading from FIFO: %s", err)
	}
	if n != 0 {
		T.Fatalf("expected to read 0 bytes from empty FIFO, read %d", n)
	}
}


func TestRemoveFile(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.Create("testfile")
	if err != nil {
		T.Fatal(err.Error())
	}
	err = file.Close()
	if err != nil {
		T.Fatalf("error closing file: %s", err)
	}
	err = mos.Remove("testfile")
	if err != nil {
		T.Fatalf("error removing file: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeFile, 0644, 0, 0, 0, ""}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestRemoveOpenFile(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.OpenFile("testfile", O_CREATE | O_RDWR, 0644)
	if err != nil {
		T.Fatalf("error opening file: %s", err)
	}
	want := "write this to the file"
	_, err = file.Write([]byte(want))
	if err != nil {
		T.Fatalf("error writing to file: %s", err)
	}
	_, err = file.Seek(0, SEEK_SET)
	if err != nil {
		T.Fatalf("error seeking in file: %s", err)
	}
	buf := make([]byte, 5)
	n, err := file.Read(buf)
	if err != nil {
		T.Fatalf("error reading from file: %s", err)
	}
	if string(buf[:n]) != "write" {
		T.Fatalf("expected to read 'write' from file, got '%s'", string(buf[:n]))
	}
	err = mos.Remove("testfile")
	if err != nil {
		T.Fatalf("error removing file: %s", err)
	}
	n, err = file.Read(buf)
	if err != nil {
		T.Fatalf("error reading from file: %s", err)
	}
	if string(buf[:n]) != " this" {
		T.Fatalf("expected to read ' this' from file, got '%s'", string(buf[:n]))
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeFile, 0644, 0, 0, 0, want}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_CREATE | O_RDWR, 2, "testfile", 10, true, true, false,
				"/testfile"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 2 },
			},
		},
	})
}


func TestRemoveLinkToMultiplyLinkedFile(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("workdir", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	file, err := mos.Create("workdir/testfile")
	if err != nil {
		T.Fatalf("error creating file: %s", err)
	}
	want := "contents"
	_, err = file.Write([]byte(want))
	if err != nil {
		T.Fatalf("error writing file: %s", err)
	}
	err = file.Close()
	if err != nil {
		T.Fatalf("error closing file: %s", err)
	}
	err = mos.Link("workdir/testfile", "/file")
	if err != nil {
		T.Fatalf("error linking to file: %s", err)
	}
	err = mos.Remove("workdir/testfile")
	if err != nil {
		T.Fatalf("error removing file: %s", err)
	}
	file, err = mos.Open("file")
	if err != nil {
		T.Fatalf("error opening file by new link: %s", err)
	}
	buf := make([]byte, 30)
	n, err := file.Read(buf)
	if err != nil {
		T.Fatalf("error reading file: %s", err)
	}
	got := string(buf[:n])
	if got != want {
		T.Fatalf("expected to read '%s' from file, got '%s'", want, got)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "file=3\nworkdir=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeFile, 0644, 0, 0, 1, want},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_RDONLY, 3, "file", int64(len(want)), true, false, false, "/file"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 3 },
			},
		},
	})
}


func TestRemoveSymlink(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.Create("targfile")
	if err != nil {
		T.Fatal(err.Error())
	}
	err = file.Close()
	if err != nil {
		T.Fatalf("error closing file: %s", err)
	}
	err = mos.Symlink("targfile", "newlink")
	if err != nil {
		T.Fatalf("error creating symlink: %s", err)
	}
	err = mos.Remove("newlink")
	if err != nil {
		T.Fatalf("error removing file: %s", err)
	}
	var stat syscall.Stat_t
	err = mos.SyscallStat("targfile", &stat)
	if err != nil {
		T.Fatalf("%s statting /targfile: ", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "targfile=2"},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
			{nodeTypeLink, 0777, 0, 0, 0, "targfile"}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestRemoveEmptyDir(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("newdir", 0755)
	if err != nil {
		T.Fatalf("error on creating directory: %s", err)
	}
	err = mos.Remove("newdir")
	if err != nil {
		T.Fatalf("error on removing directory: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 0, ""}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestRemoveNonemptyDir(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("newdir", 0755)
	if err != nil {
		T.Fatalf("error on creating directory: %s", err)
	}
	_, err = mos.Create("/newdir/file")
	if err != nil {
		T.Fatalf("error on creating file: %s", err)
	}
	err = mos.Remove("newdir")
	if err == nil || err.Error() != "remove newdir: directory not empty" {
		T.Fatalf("expected error on removing directory, got: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "newdir=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "file=3"},
			{nodeTypeFile, 0644, 0, 0, 1, ""}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_CREATE | O_WRONLY | O_TRUNC, 3, "/newdir/file", 0,
				false, true, false, "/newdir/file"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 3 },
			},
		},
	})
}


func TestRemoveAll(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	file, err := mos.Create("newfile")
	if err != nil {
		T.Fatalf("error on creating file: %s", err)
	}
	err = mos.RemoveAll("newfile")
	if err != nil {
		T.Fatalf("error on removing file: %s", err)
	}
	_, err = file.Write([]byte("written"))
	if err != nil {
		T.Fatalf("error on writing to removed file: %s", err)
	}
	err = mos.Mkdir("newdir", 0755)
	if err != nil {
		T.Fatalf("error on creating directory: %s", err)
	}
	_, err = mos.Create("/newdir/file")
	if err != nil {
		T.Fatalf("error on creating file: %s", err)
	}
	err = mos.RemoveAll("newdir")
	if err != nil {
		T.Fatalf("error on recursive directory removal %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeFile, 0644, 0, 0, 0, "written"},
			{nodeTypeDir, 0755, 0, 0, 0, ""},
			{nodeTypeFile, 0644, 0, 0, 0, ""}} },
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_CREATE | O_WRONLY | O_TRUNC, 2, "newfile", 7,
				false, true, false, "/newfile"},
			{1, 1, O_CREATE | O_WRONLY | O_TRUNC, 4, "/newdir/file", 0,
				false, true, false, "/newdir/file"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 2 },
				{1, "0:2", 4 },
			},
		},
	})
}


func TestRemoveScenarios(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.MkdirAll("one/two/three", 0751)
	if err != nil {
		T.Fatalf("error on MkdirAll(\"one/two/three\"): %s", err)
	}
	_, err = mos.Create("file")
	if err != nil {
		T.Fatalf("error creating file: %s", err)
	}
	err = mos.Symlink("file", "link")
	if err != nil {
		T.Fatalf("error creating symlink: %s", err)
	}
	want := "remove one/two: directory not empty"
	err = mos.Remove("one/two")
	if err == nil {
		T.Fatal("expected error removing non-empty directory")
	}
	if err.Error() != want {
		T.Fatalf("in attempt to remove non-empty directory, expected error '%s', got '%s'",
			want, err)
	}
	err = mos.Remove("file")
	if err != nil {
		T.Fatalf("error removing file: %s", err)
	}
	err = mos.Remove("link")
	if err != nil {
		T.Fatalf("error removing link: %s", err)
	}
}


func TestIncompletePathFileCreation(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	_, err = mos.Create("/missing/thisfile")
	if err == nil {
		T.Fatal("expected error on attempt to create file")
	}
}


func TestIncompletePath_O_CREATE_File(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	_, err = mos.OpenFile("/missing/thisfile", O_CREATE | O_RDWR, 0644)
	if err == nil {
		T.Fatal("expected error on attempt to create file")
	}
}


func TestIncompletePathMkdir(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("/missing/thisdir", 0755)
	if err == nil {
		T.Fatal("expected error on attempt to create directory")
	}
}


func TestIncompletePathSymlink(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("targdir", 0755)
	if err != nil {
		T.Fatalf("error creating link target: %s", err)
	}
	err = mos.Symlink("/targdir", "/missing/thislink")
	if err == nil {
		T.Fatal("expected error on attempt to create symlink")
	}
}


func TestIncompletePathLink(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkfifo("targfifo", 0700)
	if err != nil {
		T.Fatalf("error creating link target: %s", err)
	}
	err = mos.Symlink("/targfifo", "/missing/thislink")
	if err == nil {
		T.Fatal("expected error on attempt to create link")
	}
}


func TestIncompletePathFifo(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkfifo("/missing/fifo", 0700)
	if err == nil {
		T.Fatal("expected error on attempt to create fifo")
	}
}


func TestIncompletePathSock(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkfifo("/missing/fifo", 0700)
	if err == nil {
		T.Fatal("expected error on attempt to create fifo")
	}
}


func TestBasicChmod(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.MkdirAll("one/two/three", 0711)
	if err != nil {
		T.Fatalf("error on MkdirAll: %s", err)
	}
	err = mos.Chmod("one/two/three", 0751)
	if err != nil {
		T.Fatalf("error on Chmod(\"one/two/three\"): %s", err)
	}
	err = mos.Chmod("one", 0771)
	if err != nil {
		T.Fatalf("error on Chmod(\"one\"): %s", err)
	}
	err = mos.Chmod("/", 0711)
	if err != nil {
		T.Fatalf("error on Chmod(\"/\"): %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0711, 0, 0, 1, "one=2"},
			{nodeTypeDir, 0771, 0, 0, 1, "two=3"},
			{nodeTypeDir, 0711, 0, 0, 1, "three=4"},
			{nodeTypeDir, 0751, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestBasicChown(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.MkdirAll("one/two", 0755)
	if err != nil {
		T.Fatalf("error on MkdirAll: %s", err)
	}
	err = mos.Chown("one/two", 1000, 50)
	if err != nil {
		T.Fatalf("error on Chown(\"one/two\"): %s", err)
	}
	err = mos.Chown("one", 1000, 75)
	if err != nil {
		T.Fatalf("error on Chown(\"one\"): %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "one=2"},
			{nodeTypeDir, 0755, 1000, 75, 1, "two=3"},
			{nodeTypeDir, 0755, 1000, 50, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestBasicChtimes(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	_, err = mos.Create("thisone")
	if err != nil {
		T.Fatalf("error on creating file: %s", err)
	}
	fmt := "2006-01-02 15:04:05"
	utc, _ := time.LoadLocation("UTC")
	mTimeString := "2023-01-02 15:20:02"
	mTime, err := time.ParseInLocation(fmt, mTimeString, utc)
	if err != nil {
		T.Fatalf("error formatting time %s: %s", mTimeString, err)
	}
	aTimeString := "2023-01-15 04:01:00"
	aTime, err := time.ParseInLocation(fmt, aTimeString, utc)
	if err != nil {
		T.Fatalf("error formatting time %s: %s", aTimeString, err)
	}
	err = mos.Chtimes("thisone", aTime, mTime)
	if err != nil {
		T.Fatal("error setting file times")
	}
	var stat syscall.Stat_t
	err = mos.SyscallStat("thisone", &stat)
	if err != nil {
		T.Fatalf("%s statting /thisone: ", err)
	}
	mTimeResult := timespecToTime(stat.Mtim).In(utc).Format(fmt)
	aTimeResult := timespecToTime(stat.Atim).In(utc).Format(fmt)
	if mTimeResult != mTimeString {
		T.Fatalf("expected time '%s', got '%s'", mTimeString, mTimeResult)
	}
	if aTimeResult != aTimeString {
		T.Fatalf("expected time '%s', got '%s'", aTimeString, aTimeResult)
	}
}


func TestGetwd(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.MkdirAll("one/two/three/four", 0755)
	if err != nil {
		T.Fatalf("error creating directories: %s", err)
	}
	err = mos.Chdir("one/two")
	if err != nil {
		T.Fatalf("error changing directory: %s", err)
	}
	want := "/one/two"
	got, err := mos.Getwd()
	if err != nil {
		T.Fatalf("error getting working directory: %s", err)
	}
	if got != want {
		T.Fatalf("expected current directory to be %s, got %s", want, got)
	}
	err = mos.Chdir("three")
	if err != nil {
		T.Fatalf("error changing directory: %s", err)
	}
	want = "/one/two/three"
	got, err = mos.Getwd()
	if err != nil {
		T.Fatalf("error getting working directory: %s", err)
	}
	if got != want {
		T.Fatalf("expected current directory to be %s, got %s", want, got)
	}
}


func TestEnvironmentInPid1(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.SetEnv("first", "one")
	if err != nil {
		T.Fatalf("error in setting environment variable: %s", err)
	}
	if len(mos.environment) != 1 {
		T.Fatalf("expected 1 value in environment, got %d", len(mos.environment))
	}
	val := mos.environment["first"]
	if val != "one" {
		T.Fatalf("unexpected return value from environment: %s", val)
	}
	val2 := mos.GetEnv("first")
	if val2 != val {
		T.Fatalf("unexpected return value from environment: wanted '%s', got '%s'",
			val, val2)
	}
	err = mos.SetEnv("second", "two")
	if mos.GetEnv("first") != "one" {
		T.Fatalf("expected value 'one' from key 'first', got '%s'", mos.GetEnv("first"))
	}
	if mos.GetEnv("second") != "two" {
		T.Fatalf("expected value 'two' from key 'second', got '%s'", mos.GetEnv("second"))
	}
	if len(mos.environment) != 2 {
		T.Fatalf("expected 2 values in environment, got %d", len(mos.environment))
	}
	mos.Clearenv()
	if len(mos.environment) != 0 {
		T.Fatalf("expected empty environment, got %d variable(s)", len(mos.environment))
	}
}


func TestExpand(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	mapper := func (key string) string {
		mp := map[string]string{"who": "quis"}
		return mp[key]
	}
	want := "quis scit?"
	got := mos.Expand("$who scit?", mapper)
	if got != want {
		T.Fatalf("expected expansion '%s', got '%s'", want, got)
	}
	mos.SetEnv("HOME", "/home/user1000")
	mos.SetEnv("PATH", "/bin:/usr/bin")
	want = "/home/user1000/bin:/home/user1000/.local/bin:/bin:/usr/bin"
	got = mos.ExpandEnv("${HOME}/bin:$HOME/.local/bin:$PATH")
	if got != want {
		T.Fatalf("expected expandion '%s', got '%s'", want, got)
	}
}


func TestLoadPath(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("bin", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	err = mos.MkdirAll("usr/bin", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	file, err := mos.OpenFile("bin/nonprog", O_CREATE, 0644)
	if err != nil {
		T.Fatalf("error creating nonprog: %s", err)
	}
	file.Close()
	file, err = mos.OpenFile("bin/prog", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating prog: %s", err)
	}
	file.Close()
	file, err = mos.OpenFile("usr/bin/prog", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating prog: %s", err)
	}
	file.Close()
	file, err = mos.OpenFile("usr/bin/domagic", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating domagic: %s", err)
	}
	file.Close()
	file, err = mos.OpenFile("littleScript", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating littleScript: %s", err)
	}
	file.Close()
	lptest := func (arg, result, errmsg string) {
		pth, err := mos.LookPath(arg)
		if err != nil {
			if len(errmsg) == 0 {
				T.Fatalf("unexpected error in LookPath(\"%s\"): %s", arg, err)
			} else if errmsg != err.Error() {
				T.Fatalf("expected error '%s' in LookPath(\"%s\"), got '%s'",
					errmsg, arg, err)
			}
		} else if len(errmsg) > 0 {
			T.Fatalf("expected to get error '%s' in LookPath(\"%s\")", errmsg, arg)
		}
		if pth != result {
			T.Fatalf("expected LookPath(\"%s\") to return '%s', got '%s'", arg,
				result, pth)
		}
	}
	lptest("./littleScript", "/littleScript", "")
	lptest("littleScript", "", "no such file or directory")
	mos.SetEnv("PATH", "/bin:/usr/bin")
	lptest("littleScript", "", "no such file or directory")
	lptest("prog", "/bin/prog", "")
	lptest("nonprog", "", "no such file or directory")
	lptest("domagic", "/usr/bin/domagic", "")
}


func TestBasicStart(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("bin", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	file, err := mos.OpenFile("bin/prog", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating program: %s", err)
	}
	file.Close()
	cmd := mos.Command("prog")
	mos.SetEnv("PATH", "/bin")
	err = cmd.Start()
	if err != nil {
		T.Fatalf("error starting program: %s", err)
	}
	if cmd.newProcess.exitCode != -1 {
		T.Fatalf("expected exit code to be seeded with -1, got %d", cmd.newProcess.exitCode)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "bin=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "prog=3"},
			{nodeTypeFile, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{100, -3, O_RDONLY, 3, "/bin/prog", 0, true, false, true, "/bin/prog"},
			{100, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{100, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
		{100, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{-3, "0:2", 3 },
				{0, "0:0", 0 },
				{1, "0:0", 0 },
				{2, "0:0", 0 },
			},
		},
	})
}


func TestBasicStartAndReadNullStdin(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("bin", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	file, err := mos.OpenFile("bin/prog", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating program: %s", err)
	}
	file.Close()
	cmd := mos.Command("prog")
	mos.SetEnv("PATH", "/bin")
	err = cmd.Start()
	if err != nil {
		T.Fatalf("error starting program: %s", err)
	}
	newproc := cmd.newProcess
	if newproc.exitCode != -1 {
		T.Fatalf("expected exit code to be seeded with -1, got %d", newproc.exitCode)
	}
	buf := make([]byte, 10)
	n, err := newproc.Stdin.Read(buf)
	if err != nil {
		T.Fatalf("error reading from null stdin: %s", err)
	}
	if n != 0 {
		T.Fatalf("expected to read 0 bytes from null stdin, got %d", n)
	}
}


func TestBasicStartAndWriteNullStdout(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("bin", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	file, err := mos.OpenFile("bin/prog", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating program: %s", err)
	}
	file.Close()
	cmd := mos.Command("prog")
	mos.SetEnv("PATH", "/bin")
	err = cmd.Start()
	if err != nil {
		T.Fatalf("error starting program: %s", err)
	}
	newproc := cmd.newProcess
	if newproc.exitCode != -1 {
		T.Fatalf("expected exit code to be seeded with -1, got %d", newproc.exitCode)
	}
	n, err := newproc.Stdout.Write([]byte("test"))
	if err != nil {
		T.Fatalf("error writing to null stdin: %s", err)
	}
	if n != 0 {
		T.Fatalf("expected to write 0 bytes to null stdin, got %d", n)
	}
}


func TestBasicStartAndReadArgStdin(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("bin", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	file, err := mos.OpenFile("bin/prog", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating program: %s", err)
	}
	file.Close()
	cmd := mos.Command("prog")
	mos.SetEnv("PATH", "/bin")
	want := "get this"
	cmd.Stdin = strings.NewReader(want)
	err = cmd.Start()
	if err != nil {
		T.Fatalf("error starting program: %s", err)
	}
	newproc := cmd.newProcess
	if newproc.exitCode != -1 {
		T.Fatalf("expected exit code to be seeded with -1, got %d", newproc.exitCode)
	}
	buf := make([]byte, 10)
	n, err := newproc.Stdin.Read(buf)
	if err != nil {
		T.Fatalf("error reading from null stdin: %s", err)
	}
	got := string(buf[:n])
	if got != want {
		T.Fatalf("expected to read '%s' from stdin, got '%s'", want, got)
	}
}


func TestBasicStartAndWriteArgStdout(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("bin", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	file, err := mos.OpenFile("bin/prog", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating program: %s", err)
	}
	file.Close()
	file, err = mos.OpenFile("capture", O_CREATE | O_RDWR, 0755)
	if err != nil {
		T.Fatalf("error creating capture file: %s", err)
	}
	cmd := mos.Command("prog")
	mos.SetEnv("PATH", "/bin")
	cmd.Stdout = file
	err = cmd.Start()
	if err != nil {
		T.Fatalf("error starting program: %s", err)
	}
	newproc := cmd.newProcess
	if newproc.exitCode != -1 {
		T.Fatalf("expected exit code to be seeded with -1, got %d", newproc.exitCode)
	}
	want := "sent"
	n, err := newproc.Stdout.Write([]byte(want))
	if err != nil {
		T.Fatalf("error writing to null stdin: %s", err)
	}
	_, err = file.Seek(0, SEEK_SET)
	if err != nil {
		T.Fatalf("error seeking in capture file: %s", err)
	}
	buf := make([]byte, 10)
	n, err = file.Read(buf)
	if err != nil {
		T.Fatalf("error reading from capture file: %s", err)
	}
	got := string(buf[:n])
	if got != want {
		T.Fatalf("expected to read '%s' from stdout, got '%s'", want, got)
	}
}


func TestStartAndOpenFile(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("bin", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	file, err := mos.OpenFile("bin/prog", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating program: %s", err)
	}
	file.Close()
	cmd := mos.Command("prog")
	mos.SetEnv("PATH", "/bin")
	err = cmd.Start()
	if err != nil {
		T.Fatalf("error starting program: %s", err)
	}
	newproc := cmd.ChildProcess()
	file, err = newproc.Create("newfile")
	if err != nil {
		T.Fatalf("error opening file in new process: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "bin=2\nnewfile=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "prog=3"},
			{nodeTypeFile, 0755, 0, 0, 1, ""},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{100, -3, O_RDONLY, 3, "/bin/prog", 0, true, false, true, "/bin/prog"},
			{100, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{100, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{100, 3, O_CREATE | O_WRONLY | O_TRUNC, 4, "newfile", 0,
				false, true, false, "/newfile"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
		{100, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{-3, "0:2", 3 },
				{0, "0:0", 0 },
				{1, "0:0", 0 },
				{2, "0:0", 0 },
				{3, "0:2", 4 },
			},
		},
	})
}


func TestBasicStartThenExit(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("bin", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	file, err := mos.OpenFile("bin/prog", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating program: %s", err)
	}
	file.Close()
	cmd := mos.Command("prog")
	mos.SetEnv("PATH", "/bin")
	err = cmd.Start()
	if err != nil {
		T.Fatalf("error starting program: %s", err)
	}
	newproc := cmd.ChildProcess()
	if newproc.Exited() {
		T.Fatal("unexepected indication that process exited")
	}
	newproc.Exit(3)
	if !newproc.Exited() {
		T.Fatal("unexpected indication that process has not ended")
	}
	if newproc.ExitCode() != 3 {
		T.Fatalf("expected exit code %d, got %d", 3, newproc.ExitCode())
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "bin=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "prog=3"},
			{nodeTypeFile, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
		{100, 0, 0, -1, -2, []nsTestProcOpen{
			},
		},
	})
}


func TestBasicStartWithDirectory(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("bin", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	file, err := mos.OpenFile("bin/prog", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating program: %s", err)
	}
	file.Close()
	cmd := mos.Command("prog")
	mos.SetEnv("PATH", "/bin")
	err = mos.Mkdir("workdir", 0755)
	if err != nil {
		T.Fatalf("error creating workdir: %s", err)
	}
	cmd.Dir = "workdir"
	err = cmd.Start()
	if err != nil {
		T.Fatalf("error starting program: %s", err)
	}
	newproc := cmd.ChildProcess()
	if newproc.Exited() {
		T.Fatal("unexepected indication that process exited")
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "bin=2\nworkdir=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "prog=3"},
			{nodeTypeFile, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{100, -4, O_RDONLY, 4, "workdir", 0, true, false, true, "/workdir"},
			{100, -3, O_RDONLY, 3, "/bin/prog", 0, true, false, true, "/bin/prog"},
			{100, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
		{100, 0, 0, -1, -4, []nsTestProcOpen{
				{-1, "0:2", 1 },
				{-3, "0:2", 3 },
				{-4, "0:2", 4 },
				{0, "0:0", 0 },
				{1, "0:0", 0 },
				{2, "0:0", 0 },
			},
		},
	})
}


func TestBasicStartWithDirectoryThenExit(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("bin", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	file, err := mos.OpenFile("bin/prog", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating program: %s", err)
	}
	file.Close()
	cmd := mos.Command("prog")
	mos.SetEnv("PATH", "/bin")
	err = mos.Mkdir("workdir", 0755)
	if err != nil {
		T.Fatalf("error creating workdir: %s", err)
	}
	cmd.Dir = "workdir"
	err = cmd.Start()
	if err != nil {
		T.Fatalf("error starting program: %s", err)
	}
	newproc := cmd.ChildProcess()
	if newproc.Exited() {
		T.Fatal("unexepected indication that process exited")
	}
	newproc.Exit(0)
	if !newproc.Exited() {
		T.Fatal("unexpected indication that process has not ended")
	}
	if newproc.ExitCode() != 0 {
		T.Fatalf("expected exit code %d, got %d", 0, newproc.ExitCode())
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "bin=2\nworkdir=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "prog=3"},
			{nodeTypeFile, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
		{100, 0, 0, -1, -4, []nsTestProcOpen{
			},
		},
	})
}


func TestBasicRun(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("bin", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	file, err := mos.OpenFile("bin/prog", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating program: %s", err)
	}
	file.Close()
	cmd := mos.Command("prog")
	mos.SetEnv("PATH", "/bin")
	err = cmd.Run()
	if err != nil {
		T.Fatalf("error starting program: %s", err)
	}
	if cmd.newProcess.exitCode != -1 {
		T.Fatalf("expected exit code to be seeded with -1, got %d", cmd.newProcess.exitCode)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "bin=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "prog=3"},
			{nodeTypeFile, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestBasicRunWithDirectory(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("bin", 0755)
	if err != nil {
		T.Fatalf("error creating directory: %s", err)
	}
	file, err := mos.OpenFile("bin/prog", O_CREATE, 0755)
	if err != nil {
		T.Fatalf("error creating program: %s", err)
	}
	file.Close()
	cmd := mos.Command("prog")
	mos.SetEnv("PATH", "/bin")
	err = mos.Mkdir("work", 0700)
	if err != nil {
		T.Fatalf("error creating work directory: %s", err)
	}
	cmd.Dir = "work"
	err = cmd.Run()
	if err != nil {
		T.Fatalf("error starting program: %s", err)
	}
	if cmd.newProcess.exitCode != -1 {
		T.Fatalf("expected exit code to be seeded with -1, got %d", cmd.newProcess.exitCode)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "bin=2\nwork=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "prog=3"},
			{nodeTypeFile, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0700, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestBasicPopulatorFile(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopFile{Name: "testfile", Perms: 0644, Contents: "stuffing"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfile=2"},
			{nodeTypeFile, 0644, 0, 0, 1, "stuffing"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestBasicPopulatorOpenFile(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopFile{Name: "testfile", Perms: 0644, Contents: "stuffing"},
		PopOpenFile{Name: "testfile", Flags: O_RDONLY},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfile=2"},
			{nodeTypeFile, 0644, 0, 0, 1, "stuffing"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_RDONLY, 2, "testfile", 0, true, false, false, "/testfile"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 2 },
			},
		},
	})
}


func TestBasicPopulatorOpenFileThenRead(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopFile{Name: "testfile", Perms: 0644, Contents: "stuffing"},
		PopOpenFile{Name: "testfile", Flags: O_RDONLY},
	}
	pop, err := populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file := pop.OpenMap["testfile"]
	if file == nil {
		T.Fatal("populator failed to return expected file")
	}
	buf := make([]byte, 5)
	_, err = file.Read(buf)
	if err != nil {
		T.Fatalf("error reading file: %s", err)
	}
	want := "stuff"
	if string(buf) != want {
		T.Fatalf("wanted to read '%s', got '%s'", want, buf)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfile=2"},
			{nodeTypeFile, 0644, 0, 0, 1, "stuffing"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_RDONLY, 2, "testfile", 5, true, false, false, "/testfile"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 2 },
			},
		},
	})
}


func TestBasicPopulatorOpenFileThenReadAlternateSymbol(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopFile{Name: "testfile", Perms: 0644, Contents: "stuffing"},
		PopOpenFile{Name: "testfile", Flags: O_RDONLY, Symbol: "prueba"},
	}
	pop, err := populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file := pop.OpenMap["prueba"]
	if file == nil {
		T.Fatal("populator failed to return expected file")
	}
	buf := make([]byte, 5)
	_, err = file.Read(buf)
	if err != nil {
		T.Fatalf("error reading file: %s", err)
	}
	want := "stuff"
	if string(buf) != want {
		T.Fatalf("wanted to read '%s', got '%s'", want, buf)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfile=2"},
			{nodeTypeFile, 0644, 0, 0, 1, "stuffing"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_RDONLY, 2, "testfile", 5, true, false, false, "/testfile"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:2", 2 },
			},
		},
	})
}


func TestBasicPopulatorDir(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "testfile", Perms: 0777},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfile=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestBasicPopulatorLink(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopFile{Name: "testfile", Perms: 0444, Contents: "filled"},
		PopLink{Target: "testfile", LinkName: "second"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "second=2\ntestfile=2"},
			{nodeTypeFile, 0444, 0, 0, 2, "filled"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestBasicPopulatorSymlink(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopFile{Name: "testfile", Perms: 0400, Contents: "stuffing"},
		PopSymlink{Target: "testfile", LinkName: "newlink"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "newlink=3\ntestfile=2"},
			{nodeTypeFile, 0400, 0, 0, 1, "stuffing"},
			{nodeTypeLink, 0777, 0, 0, 1, "testfile"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestBasicPopulatorFifo(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopFifo{Name: "testfifo", Perms: 0644},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "testfifo=2"},
			{nodeTypeFifo, 0644, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestSetEnv(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	want := "/bin:/usr/bin"
	populator := PopulatorType{
		PopSetEnv{Var: "PATH", Value: want},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	got := mos.GetEnv("PATH")
	if got != want {
		T.Fatalf("expected to get PATH='%s', got '%s'", want, got)
	}
}


func TestBasicPopulatorRunProcess(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/bin", Perms: 0755},
		PopSetEnv{Var: "PATH", Value: "/bin"},
		PopFile{Name: "bin/testprog", Perms: 0755, Contents: "#!/bin/run"},
		PopDir{Name: "workdir", Perms: 0755},
		PopRunProcess{Executable: "testprog", Dir: "workdir", ExpectedPid: 100},
	}
	data, err := populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	if data.Mos.Getpid() != 1 {
		T.Fatalf("unexpected switch from PID 1 to %d", data.Mos.Getpid())
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "bin=2\nworkdir=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "testprog=3"},
			{nodeTypeFile, 0755, 0, 0, 1, "#!/bin/run"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestBasicPopulatorStartProcess(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/bin", Perms: 0755},
		PopSetEnv{Var: "PATH", Value: "/bin"},
		PopFile{Name: "bin/testprog", Perms: 0755, Contents: "#!/bin/run"},
		PopDir{Name: "workdir", Perms: 0755},
		PopStartProcess{Executable: "testprog", Dir: "workdir", ExpectedPid: 100, Symbol: "new"},
	}
	data, err := populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	if data.Mos.Getpid() != 1 {
		T.Fatalf("unexpected switch from PID 1 to %d", data.Mos.Getpid())
	}
	newProcess := data.CmdMap["new"].ChildProcess()
	env := newProcess.Environ()
	if len(env) != 1 {
		T.Fatalf("expected only 1 variable in path, found %d", len(env))
	}
	if env[0] != "PATH=/bin" {
		T.Fatalf("expected 'PATH=/bin' in environment, got '%s'", env[0])
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "bin=2\nworkdir=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "testprog=3"},
			{nodeTypeFile, 0755, 0, 0, 1, "#!/bin/run"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{100, -4, O_RDONLY, 4, "workdir", 0, true, false, true, "/workdir"},
			{100, -3, O_RDONLY, 3, "/bin/testprog", 0, true, false, true, "/bin/testprog"},
			{100, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
		{100, 0, 0, -1, -4, []nsTestProcOpen{
				{-1, "0:2", 1 },
				{-3, "0:2", 3 },
				{-4, "0:2", 4 },
				{0, "0:0", 0 },
				{1, "0:0", 0 },
				{2, "0:0", 0 },
			},
		},
	})
}


func TestBasicPopulatorStartAndSwitchToProcess(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/bin", Perms: 0755},
		PopSetEnv{Var: "PATH", Value: "/bin"},
		PopFile{Name: "bin/testprog", Perms: 0755, Contents: "#!/bin/run"},
		PopDir{Name: "workdir", Perms: 0755},
		PopStartProcess{Executable: "testprog", Dir: "workdir", ExpectedPid: 100, Symbol: "new"},
		PopSwitchContext{Symbol: "new"},
		PopFile{Name: "info", Perms: 0664, Contents: "filled"},
	}
	data, err := populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	wantPid := 100
	if data.Mos.Getpid() != wantPid {
		T.Fatalf("unexpected switch from PID %d to %d", wantPid, data.Mos.Getpid())
	}
	newProcess := data.CmdMap["new"].ChildProcess()
	if newProcess.Getpid() != wantPid {
		T.Fatalf("new process does not report PID %d but %d", wantPid, newProcess.Getpid())
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "bin=2\nworkdir=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "testprog=3"},
			{nodeTypeFile, 0755, 0, 0, 1, "#!/bin/run"},
			{nodeTypeDir, 0755, 0, 0, 1, "info=5"},
			{nodeTypeFile, 0644, 0, 0, 1, "filled"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{100, -4, O_RDONLY, 4, "workdir", 0, true, false, true, "/workdir"},
			{100, -3, O_RDONLY, 3, "/bin/testprog", 0, true, false, true, "/bin/testprog"},
			{100, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
		{100, 0, 0, -1, -4, []nsTestProcOpen{
				{-1, "0:2", 1 },
				{-3, "0:2", 3 },
				{-4, "0:2", 4 },
				{0, "0:0", 0 },
				{1, "0:0", 0 },
				{2, "0:0", 0 },
			},
		},
	})
}


func TestBasicPopulatorStartAndSwitchToProcessThenExit(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/bin", Perms: 0755},
		PopSetEnv{Var: "PATH", Value: "/bin"},
		PopFile{Name: "bin/testprog", Perms: 0755, Contents: "#!/bin/run"},
		PopDir{Name: "workdir", Perms: 0755},
		PopStartProcess{Executable: "testprog", Dir: "workdir", ExpectedPid: 100, Symbol: "new"},
		PopSwitchContext{Symbol: "new"},
		PopFile{Name: "info", Perms: 0664, Contents: "filled"},
		PopExit{Symbol: "new", Code: 4},
	}
	data, err := populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	wantPid := 1
	if data.Mos.Getpid() != wantPid {
		T.Fatalf("unexpected switch from PID %d to %d", wantPid, data.Mos.Getpid())
	}
	cmd := data.CmdMap["new"]
	if cmd == nil {
		T.Fatalf("unexpected removal of child process from CmdMap")
	}
	newProcess := cmd.ChildProcess()
	if newProcess.Getpid() != 100 {
		T.Fatalf("child process does not report PID %d but %d", 100, newProcess.Getpid())
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "bin=2\nworkdir=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "testprog=3"},
			{nodeTypeFile, 0755, 0, 0, 1, "#!/bin/run"},
			{nodeTypeDir, 0755, 0, 0, 1, "info=5"},
			{nodeTypeFile, 0644, 0, 0, 1, "filled"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
		{100, 0, 0, -1, -4, []nsTestProcOpen{
			},
		},
	})
}


func TestBasicPopulatorStartAndSwitchToProcessThenExitAndWait(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	wantRc := 4
	populator := PopulatorType{
		PopDir{Name: "/bin", Perms: 0755},
		PopSetEnv{Var: "PATH", Value: "/bin"},
		PopFile{Name: "bin/testprog", Perms: 0755, Contents: "#!/bin/run"},
		PopDir{Name: "workdir", Perms: 0755},
		PopStartProcess{Executable: "testprog", Dir: "workdir", ExpectedPid: 100,
			Symbol: "new"},
		PopSwitchContext{Symbol: "new"},
		PopFile{Name: "info", Perms: 0664, Contents: "filled"},
		PopExit{Symbol: "new", Code: wantRc},
		PopWait{Symbol: "new"},
	}
	data, err := populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	wantPid := 1
	if data.Mos.Getpid() != wantPid {
		T.Fatalf("unexpected switch from PID %d to %d", wantPid, data.Mos.Getpid())
	}
	cmd := data.CmdMap["new"]
	if cmd == nil {
		T.Fatalf("unexpected removal of child process from CmdMap")
	}
	newProcess := cmd.ChildProcess()
	rc := newProcess.ExitCode()
	if rc != wantRc {
		T.Fatalf("expected child to exit with code %d, got %d", wantRc, rc)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "bin=2\nworkdir=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "testprog=3"},
			{nodeTypeFile, 0755, 0, 0, 1, "#!/bin/run"},
			{nodeTypeDir, 0755, 0, 0, 1, "info=5"},
			{nodeTypeFile, 0644, 0, 0, 1, "filled"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestMountDevTmpfs(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("dev", 0755)
	if err != nil {
		T.Fatalf("error on creating /dev: %s", err)
	}
	err = mos.SyscallMount("dev", "/dev", "devtmpfs", 0, "")
	if err != nil {
		T.Fatalf("error mounting /dev: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestOpenMountpointDirOnDevTmpfs(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("dev", 0755)
	if err != nil {
		T.Fatalf("error on creating /dev: %s", err)
	}
	err = mos.SyscallMount("dev", "/dev", "devtmpfs", 0, "")
	if err != nil {
		T.Fatalf("error mounting /dev: %s", err)
	}
	dir, err := mos.Open("dev")
	if err != nil {
		T.Fatalf("error opening /dev: %s", err)
	}
	if dir.abspath != "/dev" {
		T.Fatalf("expected absolute path /dev, got %s", dir.abspath)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
			{1, 0, O_RDONLY, 1, "dev", 0, true, false, true, "/dev"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:5", 1 },
			},
		},
	})
}


func TestReadDevNull(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("dev", 0755)
	if err != nil {
		T.Fatalf("error on creating /dev: %s", err)
	}
	err = mos.SyscallMount("dev", "/dev", "devtmpfs", 0, "")
	if err != nil {
		T.Fatalf("error mounting /dev: %s", err)
	}
	file, err := mos.Open("/dev/null")
	if err != nil {
		T.Fatalf("error opening /dev/null: %s", err)
	}
	buf := make([]byte, 10)
	n, err := file.Read(buf)
	if err != nil {
		T.Fatalf("error reading /dev/null: %s", err)
	}
	if n != 0 {
		T.Fatalf("expected to read 0 bytes from /dev/null; got %d", n)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
			{1, 0, O_RDONLY, 2, "/dev/null", 0, true, false, false, "/dev/null"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:5", 2 },
			},
		},
	})
}


func TestAddFilesystem(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.InsertStorageDevice(-1, -1, "ext4", "/dev/sda1")
	if err != nil {
		T.Fatalf("error creating device: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"8:1", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestMountDevTmpfsThenInsertSda1(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("dev", 0755)
	if err != nil {
		T.Fatalf("error on creating /dev: %s", err)
	}
	err = mos.SyscallMount("dev", "/dev", "devtmpfs", 0, "")
	if err != nil {
		T.Fatalf("error mounting /dev: %s", err)
	}
	err = mos.InsertStorageDevice(-1, -1, "ext4", "/dev/sda1")
	if err != nil {
		T.Fatalf("error creating device: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
		{"8:1", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestMountDevTmpfsThenInsertSda1ThenStatDevice(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("dev", 0755)
	if err != nil {
		T.Fatalf("error on creating /dev: %s", err)
	}
	err = mos.SyscallMount("dev", "/dev", "devtmpfs", 0, "")
	if err != nil {
		T.Fatalf("error mounting /dev: %s", err)
	}
	err = mos.InsertStorageDevice(-1, -1, "ext4", "/dev/sda1")
	if err != nil {
		T.Fatalf("error creating device: %s", err)
	}
	var stat syscall.Stat_t
	err = mos.SyscallStat("/dev/sda1", &stat)
	if err != nil {
		T.Fatalf("error statting /dev/sda1: %s", err)
	}
	want, _ := MajorMinorToStDev(8, 1)
	if stat.Rdev != want {
		T.Fatalf("expected to find st_dev %d, got %d", want, stat.Rdev)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
		{"8:1", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestInsertSda1ThenMount(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("dev", 0755)
	if err != nil {
		T.Fatalf("error on creating /dev: %s", err)
	}
	err = mos.SyscallMount("dev", "/dev", "devtmpfs", 0, "")
	if err != nil {
		T.Fatalf("error mounting /dev: %s", err)
	}
	err = mos.InsertStorageDevice(-1, -1, "ext4", "/dev/sda1")
	if err != nil {
		T.Fatalf("error creating device: %s", err)
	}
	err = mos.Mkdir("var", 0755)
	if err != nil {
		T.Fatalf("error creating /var: %s", err)
	}
	err = mos.SyscallMount("/dev/sda1", "/var", "ext4", 0, "")
	if err != nil {
		T.Fatalf("error mounting /var: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
		{"8:1", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
			{3, 2},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
		{8, 1, 1, 0, 3, []nsTestOpen{
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestInsertAndMountSda1ThenMkdir(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("dev", 0755)
	if err != nil {
		T.Fatalf("error on creating /dev: %s", err)
	}
	err = mos.SyscallMount("dev", "/dev", "devtmpfs", 0, "")
	if err != nil {
		T.Fatalf("error mounting /dev: %s", err)
	}
	err = mos.InsertStorageDevice(-1, -1, "ext4", "/dev/sda1")
	if err != nil {
		T.Fatalf("error creating device: %s", err)
	}
	err = mos.Mkdir("var", 0755)
	if err != nil {
		T.Fatalf("error creating /var: %s", err)
	}
	err = mos.SyscallMount("/dev/sda1", "/var", "ext4", 0, "")
	if err != nil {
		T.Fatalf("error mounting /var: %s", err)
	}
	err = mos.Mkdir("/var/log", 0755)
	if err != nil {
		T.Fatalf("error creating /var/log: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
		{"8:1", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "log=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
			{3, 2},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
		{8, 1, 1, 0, 3, []nsTestOpen{
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestInsertAndMountSda1ThenMkdirThenUnmount(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("dev", 0755)
	if err != nil {
		T.Fatalf("error on creating /dev: %s", err)
	}
	err = mos.SyscallMount("dev", "/dev", "devtmpfs", 0, "")
	if err != nil {
		T.Fatalf("error mounting /dev: %s", err)
	}
	err = mos.InsertStorageDevice(-1, -1, "ext4", "/dev/sda1")
	if err != nil {
		T.Fatalf("error creating device: %s", err)
	}
	err = mos.Mkdir("var", 0755)
	if err != nil {
		T.Fatalf("error creating /var: %s", err)
	}
	err = mos.SyscallMount("/dev/sda1", "/var", "ext4", 0, "")
	if err != nil {
		T.Fatalf("error mounting /var: %s", err)
	}
	err = mos.Mkdir("/var/log", 0755)
	if err != nil {
		T.Fatalf("error creating /var/log: %s", err)
	}
	err = mos.SyscallUnmount("var", 0)
	if err != nil {
		T.Fatalf("error unmounting /var: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
		{"8:1", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "log=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestInsertAndMountSda1ThenMkdirAndOpenFile(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("dev", 0755)
	if err != nil {
		T.Fatalf("error on creating /dev: %s", err)
	}
	err = mos.SyscallMount("dev", "/dev", "devtmpfs", 0, "")
	if err != nil {
		T.Fatalf("error mounting /dev: %s", err)
	}
	err = mos.InsertStorageDevice(-1, -1, "ext4", "/dev/sda1")
	if err != nil {
		T.Fatalf("error creating device: %s", err)
	}
	err = mos.Mkdir("var", 0755)
	if err != nil {
		T.Fatalf("error creating /var: %s", err)
	}
	err = mos.SyscallMount("/dev/sda1", "/var", "ext4", 0, "")
	if err != nil {
		T.Fatalf("error mounting /var: %s", err)
	}
	err = mos.Mkdir("/var/log", 0755)
	if err != nil {
		T.Fatalf("error creating /var/log: %s", err)
	}
	file, err := mos.OpenFile("/var/log/messages", O_CREATE | O_WRONLY, 0600)
	if err != nil {
		T.Fatalf("error creating /var/log/messages: %s", err)
	}
	_, err = file.Write([]byte("started\n"))
	if err != nil {
		T.Fatalf("error writing to /var/log/messages: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
		{"8:1", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "log=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "messages=3"},
			{nodeTypeFile, 0600, 0, 0, 1, "started\n"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
			{3, 2},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
		{8, 1, 1, 0, 3, []nsTestOpen{
			{1, 0, O_CREATE | O_WRONLY, 3, "/var/log/messages", int64(len("started\n")),
				false, true, false, "/var/log/messages"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "8:1", 3 },
			},
		},
	})
}


func TestInsertAndMountLvmVolumeThenMkdirAndOpenFile(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	err = mos.Mkdir("dev", 0755)
	if err != nil {
		T.Fatalf("error on creating /dev: %s", err)
	}
	err = mos.SyscallMount("dev", "/dev", "devtmpfs", 0, "")
	if err != nil {
		T.Fatalf("error mounting /dev: %s", err)
	}
	err = mos.InsertStorageDevice(-1, -1, "btrfs", "/dev/mapper/vg-var")
	if err != nil {
		T.Fatalf("error creating device: %s", err)
	}
	err = mos.Mkdir("var", 0755)
	if err != nil {
		T.Fatalf("error creating /var: %s", err)
	}
	err = mos.SyscallMount("/dev/mapper/vg-var", "/var", "btrfs", 0, "")
	if err != nil {
		T.Fatalf("error mounting /var: %s", err)
	}
	err = mos.Mkdir("/var/log", 0755)
	if err != nil {
		T.Fatalf("error creating /var/log: %s", err)
	}
	file, err := mos.OpenFile("/var/log/messages", O_CREATE | O_WRONLY, 0600)
	if err != nil {
		T.Fatalf("error creating /var/log/messages: %s", err)
	}
	_, err = file.Write([]byte("started\n"))
	if err != nil {
		T.Fatalf("error writing to /var/log/messages: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "mapper=3\nnull=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "log=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "messages=3"},
			{nodeTypeFile, 0600, 0, 0, 1, "started\n"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
			{3, 2},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
		{0, 6, 1, 0, 3, []nsTestOpen{
			{1, 0, O_CREATE | O_WRONLY, 3, "/var/log/messages", int64(len("started\n")),
				false, true, false, "/var/log/messages"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:6", 3 },
			},
		},
	})
}


func TestStorageMountsWithPopulator(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "ext4", Source: "/dev/sda1"},
		PopDir{Name: "boot", Perms: 0755},
		PopMount{Source: "/dev/sda1", Mountpoint: "/boot", Fstype: "ext4"},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "btrfs",
			Source: "/dev/mapper/vg-var"},
		PopDir{Name: "var", Perms: 0755},
		PopMount{Source: "/dev/mapper/vg-var", Mountpoint: "/var", Fstype: "btrfs"},
		PopDir{Name: "/var/log", Perms: 0755},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file, err := mos.OpenFile("/var/log/messages", O_CREATE | O_WRONLY, 0600)
	if err != nil {
		T.Fatalf("error creating /var/log/messages: %s", err)
	}
	_, err = file.Write([]byte("started\n"))
	if err != nil {
		T.Fatalf("error writing to /var/log/messages: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "boot=3\ndev=2\nvar=4"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "mapper=3\nnull=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "log=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "messages=3"},
			{nodeTypeFile, 0600, 0, 0, 1, "started\n"},
		}},
		{"8:1", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
			{3, 2},
			{4, 3},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
		{8, 1, 1, 0, 3, []nsTestOpen{
		},
		nil},
		{0, 6, 1, 0, 4, []nsTestOpen{
			{1, 0, O_CREATE | O_WRONLY, 3, "/var/log/messages", int64(len("started\n")),
				false, true, false, "/var/log/messages"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:6", 3 },
			},
		},
	})
}


func TestUnmountBusy(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "ext4", Source: "/dev/sda1"},
		PopDir{Name: "boot", Perms: 0755},
		PopMount{Source: "/dev/sda1", Mountpoint: "/boot", Fstype: "ext4"},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "btrfs",
			Source: "/dev/mapper/vg-var"},
		PopDir{Name: "var", Perms: 0755},
		PopMount{Source: "/dev/mapper/vg-var", Mountpoint: "/var", Fstype: "btrfs"},
		PopDir{Name: "/var/log", Perms: 0755},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	_, err = mos.OpenFile("/var/log/messages", O_CREATE | O_WRONLY, 0600)
	if err != nil {
		T.Fatalf("error creating /var/log/messages: %s", err)
	}
	want := "unmount /var: device or resource busy"
	err = mos.SyscallUnmount("/var", 0)
	if err == nil {
		T.Fatalf("expected error on attempt to unmount busy /var")
	} else if err.Error() != want {
		T.Fatalf("wanted error '%s' on unmount of busy /var, got '%s'", want, err)
	}
}


func TestUnmountNonMountpoint(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "ext4", Source: "/dev/sda1"},
		PopDir{Name: "boot", Perms: 0755},
		PopMount{Source: "/dev/sda1", Mountpoint: "/boot", Fstype: "ext4"},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "btrfs",
			Source: "/dev/mapper/vg-var"},
		PopDir{Name: "var", Perms: 0755},
		PopMount{Source: "/dev/mapper/vg-var", Mountpoint: "/var", Fstype: "btrfs"},
		PopDir{Name: "/var/log", Perms: 0755},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	_, err = mos.OpenFile("/var/log/messages", O_CREATE | O_WRONLY, 0600)
	if err != nil {
		T.Fatalf("error creating /var/log/messages: %s", err)
	}
	want := "unmount /var/log: invalid argument"
	err = mos.SyscallUnmount("/var/log", 0)
	if err == nil {
		T.Fatalf("expected error on attempt to unmount busy /var")
	} else if err.Error() != want {
		T.Fatalf("wanted error '%s' on unmount of busy /var, got '%s'", want, err)
	}
}


func TestUnmountMountNotAtEnd(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "ext4", Source: "/dev/sda1"},
		PopDir{Name: "boot", Perms: 0755},
		PopMount{Source: "/dev/sda1", Mountpoint: "/boot", Fstype: "ext4"},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "btrfs",
			Source: "/dev/mapper/vg-var"},
		PopDir{Name: "var", Perms: 0755},
		PopMount{Source: "/dev/mapper/vg-var", Mountpoint: "/var", Fstype: "btrfs"},
		PopDir{Name: "/var/log", Perms: 0755},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file, err := mos.OpenFile("/var/log/messages", O_CREATE | O_WRONLY, 0600)
	if err != nil {
		T.Fatalf("error creating /var/log/messages: %s", err)
	}
	_, err = file.Write([]byte("started\n"))
	if err != nil {
		T.Fatalf("error writing to /var/log/messages: %s", err)
	}
	err = mos.SyscallUnmount("boot", 0)
	if err != nil {
		T.Fatalf("error unmounting /boot: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "boot=3\ndev=2\nvar=4"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "mapper=3\nnull=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "log=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "messages=3"},
			{nodeTypeFile, 0600, 0, 0, 1, "started\n"},
		}},
		{"8:1", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
			{4, 2},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
		{0, 6, 1, 0, 4, []nsTestOpen{
			{1, 0, O_CREATE | O_WRONLY, 3, "/var/log/messages", int64(len("started\n")),
				false, true, false, "/var/log/messages"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:6", 3 },
			},
		},
	})
}


func TestBasicSubmount(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "ext4",
			Source: "/dev/mapper/vg-var"},
		PopDir{Name: "var", Perms: 0755},
		PopMount{Source: "/dev/mapper/vg-var", Mountpoint: "/var", Fstype: "ext4"},
		PopDir{Name: "/var/log", Perms: 0755},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "ext4",
			Source: "/dev/mapper/vg-portage"},
		PopDir{Name: "var/db/repos", Perms: 0755},
		PopMount{Source: "/dev/mapper/vg-portage", Mountpoint: "/var/db/repos"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file, err := mos.OpenFile("/var/log/messages", O_CREATE | O_WRONLY, 0600)
	if err != nil {
		T.Fatalf("error creating /var/log/messages: %s", err)
	}
	_, err = file.Write([]byte("started\n"))
	if err != nil {
		T.Fatalf("error writing to /var/log/messages: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "mapper=3\nnull=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "db=3\nlog=2"},	// /var
			{nodeTypeDir, 0755, 0, 0, 1, "messages=5"},	// /var/log
			{nodeTypeDir, 0755, 0, 0, 1, "repos=4"},	// /var/db
			{nodeTypeDir, 0755, 0, 0, 1, ""},		// /var/db/repos
			{nodeTypeFile, 0600, 0, 0, 1, "started\n"},	// /var/log/messages
		}},
		{"0:7", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
			{3, 2},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
		{0, 6, 1, 0, 3, []nsTestOpen{
			{1, 0, O_CREATE | O_WRONLY, 5, "/var/log/messages", int64(len("started\n")),
				false, true, false, "/var/log/messages"},
		},
		[]nsTestMount{
			{4, 3},
		}},
		{0, 7, 1, 2, 4, []nsTestOpen{},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:6", 5 },
			},
		},
	})
}


func TestUnmountWithSubmount(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "ext4",
			Source: "/dev/mapper/vg-var"},
		PopDir{Name: "var", Perms: 0755},
		PopMount{Source: "/dev/mapper/vg-var", Mountpoint: "/var", Fstype: "ext4"},
		PopDir{Name: "/var/log", Perms: 0755},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "ext4",
			Source: "/dev/mapper/vg-portage"},
		PopDir{Name: "var/db/repos", Perms: 0755},
		PopMount{Source: "/dev/mapper/vg-portage", Mountpoint: "/var/db/repos"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	want := "unmount /var: device or resource busy"
	err = mos.SyscallUnmount("/var", 0)
	if err == nil {
		T.Fatal("expected error in attempt to unmount /dev")
	} else if err.Error() != want {
		T.Fatalf("expected error '%s' attempting to unmount /var, got '%s'", want, err)
	}
}


func TestUnmountSubmount(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "ext4",
			Source: "/dev/mapper/vg-var"},
		PopDir{Name: "var", Perms: 0755},
		PopMount{Source: "/dev/mapper/vg-var", Mountpoint: "/var", Fstype: "ext4"},
		PopDir{Name: "/var/log", Perms: 0755},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "ext4",
			Source: "/dev/mapper/vg-portage"},
		PopDir{Name: "var/db/repos", Perms: 0755},
		PopMount{Source: "/dev/mapper/vg-portage", Mountpoint: "/var/db/repos"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file, err := mos.OpenFile("/var/log/messages", O_CREATE | O_WRONLY, 0600)
	if err != nil {
		T.Fatalf("error creating /var/log/messages: %s", err)
	}
	_, err = file.Write([]byte("started\n"))
	if err != nil {
		T.Fatalf("error writing to /var/log/messages: %s", err)
	}
	err = mos.SyscallUnmount("/var/db/repos", 0)
	if err != nil {
		T.Fatalf("error unmounting /var/db/repos: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "mapper=3\nnull=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "db=3\nlog=2"},	// /var
			{nodeTypeDir, 0755, 0, 0, 1, "messages=5"},	// /var/log
			{nodeTypeDir, 0755, 0, 0, 1, "repos=4"},	// /var/db
			{nodeTypeDir, 0755, 0, 0, 1, ""},		// /var/db/repos
			{nodeTypeFile, 0600, 0, 0, 1, "started\n"},	// /var/log/messages
		}},
		{"0:7", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
			{3, 2},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
		{0, 6, 1, 0, 3, []nsTestOpen{
			{1, 0, O_CREATE | O_WRONLY, 5, "/var/log/messages", int64(len("started\n")),
				false, true, false, "/var/log/messages"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
				{0, "0:6", 5 },
			},
		},
	})
}


func TestUnmountSubmountTheCloseFileThenUnmount(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "ext4",
			Source: "/dev/mapper/vg-var"},
		PopDir{Name: "var", Perms: 0755},
		PopMount{Source: "/dev/mapper/vg-var", Mountpoint: "/var", Fstype: "ext4"},
		PopDir{Name: "/var/log", Perms: 0755},
		PopInsertStorageDevice{Major: -1, Minor: -1, Fstype: "ext4",
			Source: "/dev/mapper/vg-portage"},
		PopDir{Name: "var/db/repos", Perms: 0755},
		PopMount{Source: "/dev/mapper/vg-portage", Mountpoint: "/var/db/repos"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file, err := mos.OpenFile("/var/log/messages", O_CREATE | O_WRONLY, 0600)
	if err != nil {
		T.Fatalf("error creating /var/log/messages: %s", err)
	}
	_, err = file.Write([]byte("started\n"))
	if err != nil {
		T.Fatalf("error writing to /var/log/messages: %s", err)
	}
	err = mos.SyscallUnmount("/var/db/repos", 0)
	if err != nil {
		T.Fatalf("error unmounting /var/db/repos: %s", err)
	}
	err = file.Close()
	if err != nil {
		T.Fatalf("error closing /var/log/message: %s", err)
	}
	err = mos.SyscallUnmount("/var", 0)
	if err != nil {
		T.Fatalf("error unmounting /var: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "mapper=3\nnull=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "db=3\nlog=2"},	// /var
			{nodeTypeDir, 0755, 0, 0, 1, "messages=5"},	// /var/log
			{nodeTypeDir, 0755, 0, 0, 1, "repos=4"},	// /var/db
			{nodeTypeDir, 0755, 0, 0, 1, ""},		// /var/db/repos
			{nodeTypeFile, 0600, 0, 0, 1, "started\n"},	// /var/log/messages
		}},
		{"0:7", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },
				{-1, "0:2", 1 },
			},
		},
	})
}


func TestBasicBindMount(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopDir{Name: "/var/db/repos/gentoo/", Perms: 0755},
		PopFile{Name: "/var/db/repos/gentoo/Manifest", Perms: 0644,
			Contents: "manifest goes here"},
		PopDir{Name: "/var/db/repos/local/profiles", Perms: 0755},
		PopDir{Name: "/usr/portage", Perms: 0755},
		PopMount{Source: "var/db/repos/gentoo", Mountpoint: "usr/portage", Flags: MS_BIND},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nusr=10\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "db=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "repos=5"},
			{nodeTypeDir, 0755, 0, 0, 1, "gentoo=6\nlocal=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "Manifest=7"},
			{nodeTypeFile, 0644, 0, 0, 1, "manifest goes here"},
			{nodeTypeDir, 0755, 0, 0, 1, "profiles=9"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=11"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -3, O_RDONLY, 6, "var/db/repos/gentoo", 0, true, false, true,
				"/var/db/repos/gentoo"},
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
			{11, 2},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
		{0, 2, 6, 0, 11, []nsTestOpen{
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-3, "0:2", 6 },	// /var/db/repos/gentoo
				{-2, "0:2", 1 },	// /
				{-1, "0:2", 1 },	// /
			},
		},
	})
}


func TestBasicBindMountWithOtherProcess(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopDir{Name: "/var/db/repos/gentoo/", Perms: 0755},
		PopFile{Name: "/var/db/repos/gentoo/Manifest", Perms: 0644,
			Contents: "manifest goes here"},
		PopDir{Name: "/var/db/repos/local/profiles", Perms: 0755},
		PopDir{Name: "/usr/portage", Perms: 0755},
		PopFile{Name: "/runner", Perms: 0755},
		PopStartProcess{Executable: "/runner", Dir: "/", ExpectedPid: 100, Symbol: "new"},
		PopSwitchContext{Symbol: "new"},
		PopMount{Source: "var/db/repos/gentoo", Mountpoint: "usr/portage", Flags: MS_BIND},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nrunner=12\nusr=10\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "db=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "repos=5"},
			{nodeTypeDir, 0755, 0, 0, 1, "gentoo=6\nlocal=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "Manifest=7"},
			{nodeTypeFile, 0644, 0, 0, 1, "manifest goes here"},
			{nodeTypeDir, 0755, 0, 0, 1, "profiles=9"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=11"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeFile, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -3, O_RDONLY, 6, "var/db/repos/gentoo", 0, true, false, true,
				"/var/db/repos/gentoo"},
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{100, -3, O_RDONLY, 12, "/runner", 0, true, false, true,
				"/runner"},
			{100, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{100, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
			{11, 2},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
		{0, 2, 6, 0, 11, []nsTestOpen{
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-3, "0:2", 6 },	// /var/db/repos/gentoo
				{-2, "0:2", 1 },	// /
				{-1, "0:2", 1 },	// /
			},
		},
		{100, 0, 0, -1, -2, []nsTestProcOpen{
				{-3, "0:2", 12 },	// /runner
				{-2, "0:2", 1 },	// /
				{-1, "0:2", 1 },	// /
				{ 0, "0:0", 0 },	// stdin
				{ 1, "0:0", 0 },	// stdout
				{ 2, "0:0", 0 },	// stderr
			},
		},
	})
}


func TestBasicBindMountWithOtherProcessThenUnmount(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopDir{Name: "/var/db/repos/gentoo/", Perms: 0755},
		PopFile{Name: "/var/db/repos/gentoo/Manifest", Perms: 0644,
			Contents: "manifest goes here"},
		PopDir{Name: "/var/db/repos/local/profiles", Perms: 0755},
		PopDir{Name: "/usr/portage", Perms: 0755},
		PopFile{Name: "/runner", Perms: 0755},
		PopStartProcess{Executable: "/runner", Dir: "/", ExpectedPid: 100, Symbol: "new"},
		PopSwitchContext{Symbol: "new"},
		PopMount{Source: "var/db/repos/gentoo", Mountpoint: "usr/portage", Flags: MS_BIND},
	}
	data, err := populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	newProcess := data.CmdMap["new"].ChildProcess()
	err = newProcess.SyscallUnmount("/usr/portage", 0)
	if err != nil {
		T.Fatalf("error unmounting bind mount: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nrunner=12\nusr=10\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "db=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "repos=5"},
			{nodeTypeDir, 0755, 0, 0, 1, "gentoo=6\nlocal=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "Manifest=7"},
			{nodeTypeFile, 0644, 0, 0, 1, "manifest goes here"},
			{nodeTypeDir, 0755, 0, 0, 1, "profiles=9"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=11"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeFile, 0755, 0, 0, 1, ""},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{100, -3, O_RDONLY, 12, "/runner", 0, true, false, true,
				"/runner"},
			{100, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{100, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
		},
		[]nsTestMount{
			{2, 1},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },	// /
				{-1, "0:2", 1 },	// /
			},
		},
		{100, 0, 0, -1, -2, []nsTestProcOpen{
				{-3, "0:2", 12 },	// /runner
				{-2, "0:2", 1 },	// /
				{-1, "0:2", 1 },	// /
				{ 0, "0:0", 0 },	// stdin
				{ 1, "0:0", 0 },	// stdout
				{ 2, "0:0", 0 },	// stderr
			},
		},
	})
}


func TestBindMountTwoOpens(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopDir{Name: "/var/db/repos/gentoo/", Perms: 0755},
		PopFile{Name: "/var/db/repos/gentoo/Manifest", Perms: 0644,
			Contents: "manifest goes here"},
		PopDir{Name: "/var/db/repos/local/profiles", Perms: 0755},
		PopDir{Name: "/usr/portage", Perms: 0755},
		PopMount{Source: "var/db/repos/gentoo", Mountpoint: "usr/portage", Flags: MS_BIND},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file1, err := mos.Create("/usr/portage/header.txt")
	if err != nil {
		T.Fatalf("error creating /usr/portage/header.txt")
	}
	want := "Goes into header"
	_, err = file1.Write([]byte(want))
	if err != nil {
		T.Fatalf("error writing to /usr/portage/header.txt")
	}
	file2, err := mos.Open("/var/db/repos/gentoo/header.txt")
	if err != nil {
		T.Fatalf("error opening /var/db/repos/gentoo/header.txt: %s", err)
	}
	buf := make([]byte, 30)
	n, err := file2.Read(buf)
	if err != nil {
		T.Fatalf("error reading from /var/db/repos/gentoo/header.txt: %s", err)
	}
	if string(buf[:n]) != want {
		T.Fatalf("expected to read '%s', got '%s'", want, buf[:n])
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nusr=10\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "db=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "repos=5"},
			{nodeTypeDir, 0755, 0, 0, 1, "gentoo=6\nlocal=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "Manifest=7\nheader.txt=12"},
			{nodeTypeFile, 0644, 0, 0, 1, "manifest goes here"},
			{nodeTypeDir, 0755, 0, 0, 1, "profiles=9"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=11"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeFile, 0644, 0, 0, 1, want},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -3, O_RDONLY, 6, "var/db/repos/gentoo", 0, true, false, true,
				"/var/db/repos/gentoo"},
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 1, O_RDONLY, 12, "/var/db/repos/gentoo/header.txt",
				int64(n), true, false, false, "/var/db/repos/gentoo/header.txt"},
		},
		[]nsTestMount{
			{2, 1},
			{11, 2},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
		{0, 2, 6, 0, 11, []nsTestOpen{
			{1, 0, O_CREATE | O_TRUNC | O_WRONLY, 12, "/usr/portage/header.txt",
				int64(n), false, true, false, "/usr/portage/header.txt"},
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-3, "0:2", 6 },	// /var/db/repos/gentoo
				{-2, "0:2", 1 },	// /
				{-1, "0:2", 1 },	// /
				{ 0, "0:2", 12 },	// /usr/portage/header.txt
				{ 1, "0:2", 12 },	// /var/db/repos/gentoo/header.txt
			},
		},
	})
}


func TestBindMountTwoOpensAttemptUnmount(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopDir{Name: "/var/db/repos/gentoo/", Perms: 0755},
		PopFile{Name: "/var/db/repos/gentoo/Manifest", Perms: 0644,
			Contents: "manifest goes here"},
		PopDir{Name: "/var/db/repos/local/profiles", Perms: 0755},
		PopDir{Name: "/usr/portage", Perms: 0755},
		PopMount{Source: "var/db/repos/gentoo", Mountpoint: "usr/portage", Flags: MS_BIND},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file1, err := mos.Create("/usr/portage/header.txt")
	if err != nil {
		T.Fatalf("error creating /usr/portage/header.txt")
	}
	want := "Goes into header"
	_, err = file1.Write([]byte(want))
	if err != nil {
		T.Fatalf("error writing to /usr/portage/header.txt")
	}
	file2, err := mos.Open("/var/db/repos/gentoo/header.txt")
	if err != nil {
		T.Fatalf("error opening /var/db/repos/gentoo/header.txt: %s", err)
	}
	buf := make([]byte, 30)
	n, err := file2.Read(buf)
	if err != nil {
		T.Fatalf("error reading from /var/db/repos/gentoo/header.txt: %s", err)
	}
	if string(buf[:n]) != want {
		T.Fatalf("expected to read '%s', got '%s'", want, buf[:n])
	}
	want = "unmount /usr/portage: device or resource busy"
	err = mos.SyscallUnmount("/usr/portage", 0)
	if err == nil {
		T.Fatalf("expected error on attempting to unmount /usr/portage: %s", err)
	} else if err.Error() != want {
		T.Fatalf("expected error '%s', got '%s'", want, err)
	}
}


func TestBindMountOpenTwoCloseOneThenUnmount(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopDir{Name: "/var/db/repos/gentoo/", Perms: 0755},
		PopFile{Name: "/var/db/repos/gentoo/Manifest", Perms: 0644,
			Contents: "manifest goes here"},
		PopDir{Name: "/var/db/repos/local/profiles", Perms: 0755},
		PopDir{Name: "/usr/portage", Perms: 0755},
		PopMount{Source: "var/db/repos/gentoo", Mountpoint: "usr/portage", Flags: MS_BIND},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file1, err := mos.Create("/usr/portage/header.txt")
	if err != nil {
		T.Fatalf("error creating /usr/portage/header.txt")
	}
	want := "Goes into header"
	_, err = file1.Write([]byte(want))
	if err != nil {
		T.Fatalf("error writing to /usr/portage/header.txt")
	}
	file2, err := mos.Open("/var/db/repos/gentoo/header.txt")
	if err != nil {
		T.Fatalf("error opening /var/db/repos/gentoo/header.txt: %s", err)
	}
	buf := make([]byte, 30)
	n, err := file2.Read(buf)
	if err != nil {
		T.Fatalf("error reading from /var/db/repos/gentoo/header.txt: %s", err)
	}
	if string(buf[:n]) != want {
		T.Fatalf("expected to read '%s', got '%s'", want, buf[:n])
	}
	err = file1.Close()
	if err != nil {
		T.Fatalf("error closing /usr/portage/header.txt: %s", err)
	}
	err = mos.SyscallUnmount("/usr/portage", 0)
	if err != nil {
		T.Fatalf("error unmounting /usr/portage: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nusr=10\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "db=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "repos=5"},
			{nodeTypeDir, 0755, 0, 0, 1, "gentoo=6\nlocal=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "Manifest=7\nheader.txt=12"},
			{nodeTypeFile, 0644, 0, 0, 1, "manifest goes here"},
			{nodeTypeDir, 0755, 0, 0, 1, "profiles=9"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=11"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeFile, 0644, 0, 0, 1, want},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 1, O_RDONLY, 12, "/var/db/repos/gentoo/header.txt",
				int64(n), true, false, false, "/var/db/repos/gentoo/header.txt"},
		},
		[]nsTestMount{
			{2, 1},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-2, "0:2", 1 },	// /
				{-1, "0:2", 1 },	// /
				{ 1, "0:2", 12 },	// /var/db/repos/gentoo/header.txt
			},
		},
	})
}


func TestBindMountOpenExistingFile(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/dev", Perms: 0755},
		PopMount{Source: "dev", Mountpoint: "/dev", Fstype: "devtmpfs"},
		PopDir{Name: "/var/db/repos/gentoo/", Perms: 0755},
		PopFile{Name: "/var/db/repos/gentoo/Manifest", Perms: 0644,
			Contents: "manifest goes here"},
		PopDir{Name: "/var/db/repos/local/profiles", Perms: 0755},
		PopDir{Name: "/usr/portage", Perms: 0755},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file1, err := mos.Create("/var/db/repos/gentoo/header.txt")
	if err != nil {
		T.Fatalf("error creating /var/db/repos/gentoo/header.txt: %s", err)
	}
	want := "This is the header"
	_, err = file1.Write([]byte(want))
	if err != nil {
		T.Fatalf("error writing to /var/db/repos/gentoo/header.txt: %s", err)
	}
	err = mos.SyscallMount("/var/db/repos/gentoo", "/usr/portage", "", MS_BIND, "")
	if err != nil {
		T.Fatalf("error mounting /usr/portage: %s", err)
	}
	file2, err := mos.Open("usr/portage/header.txt")
	if err != nil {
		T.Fatalf("error opening /usr/portage/header.txt: %s", err)
	}
	buf := make([]byte, 40)
	n, err := file2.Read(buf)
	if err != nil {
		T.Fatalf("error reading /usr/portage/header.txt: %s", err)
	}
	got := string(buf[:n])
	if got != want {
		T.Fatalf("expected to read '%s', got '%s'", want, got)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "dev=2\nusr=10\nvar=3"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "db=4"},
			{nodeTypeDir, 0755, 0, 0, 1, "repos=5"},
			{nodeTypeDir, 0755, 0, 0, 1, "gentoo=6\nlocal=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "Manifest=7\nheader.txt=12"},
			{nodeTypeFile, 0644, 0, 0, 1, "manifest goes here"},
			{nodeTypeDir, 0755, 0, 0, 1, "profiles=9"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=11"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeFile, 0644, 0, 0, 1, want},
		}},
		{"0:5", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "null=2"},
			{nodeTypeCharDev, 0666, 0, 0, 1, "1:3"},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0, []nsTestOpen{
			{1, -3, O_RDONLY, 6, "/var/db/repos/gentoo", 0,
				true, false, true, "/var/db/repos/gentoo"},
			{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			{1, 0, O_CREATE | O_TRUNC | O_WRONLY, 12, "/var/db/repos/gentoo/header.txt",
				int64(n), false, true, false, "/var/db/repos/gentoo/header.txt"},
		},
		[]nsTestMount{
			{2, 1},
			{11, 2},
		}},
		{0, 5, 1, 0, 2, []nsTestOpen{
		},
		nil},
		{0, 2, 6, 0, 11, []nsTestOpen{
			{1, 1, O_RDONLY, 12, "usr/portage/header.txt", int64(n),
				true, false, false, "/usr/portage/header.txt"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-3, "0:2", 6 },	// /var/db/repos/gentoo/
				{-2, "0:2", 1 },	// /
				{-1, "0:2", 1 },	// /
				{ 0, "0:2", 12 },	// /var/db/repos/gentoo/header.txt
				{ 1, "0:2", 12 },	// /usr/portage/header.txt
			},
		},
	})
}

