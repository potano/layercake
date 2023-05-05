// Copyright © 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import (
	"time"
	"syscall"
        "testing"
)

func TestMountOverlayFS(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0,
			[]nsTestOpen{
				{1, -5, O_RDONLY, 6, "/var/layer/work", 0, true, false, true,
					"/var/layer/work"},
				{1, -4, O_RDONLY, 7, "/var/layer/lower", 0, true, false, true,
					"/var/layer/lower"},
				{1, -3, O_RDONLY, 5, "/var/layer/upper", 0, true, false, true,
					"/var/layer/upper"},
				{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			},
			[]nsTestMount{
				{4, 1},
			},
		},
		{0, 6, 1, 0, 4, []nsTestOpen{}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
		}},
	})
}


func TestMountOverlayFSThenUmount(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	err = mos.SyscallUnmount("var/layer/build", 0)
	if err != nil {
		T.Fatalf("error unmounting /var/layer/build: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0,
			[]nsTestOpen{
				{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			},
			nil,
		},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-2, "0:2", 1},
			{-1, "0:2", 1},
		}},
	})
}


func TestOverlayOpenFileFromLower(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "written to lower dir"
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopFile{Name: "/var/layer/lower/file1", Perms: 0644, Contents: inFile},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file, err := mos.Open("/var/layer/build/file1")
	if err != nil {
		T.Fatalf("error when opening file: %s", err)
	}
	buf := make([]byte, 30)
	n, err := file.Read(buf)
	if err != nil {
		T.Fatalf("error reading file: %s", err)
	}
	if string(buf[:n]) != inFile {
		T.Fatalf("expected to read '%s'; got '%s'", inFile, buf[:n])
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=8"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "file1=2"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0,
			[]nsTestOpen{
				{1, -5, O_RDONLY, 6, "/var/layer/work", 0, true, false, true,
					"/var/layer/work"},
				{1, -4, O_RDONLY, 7, "/var/layer/lower", 0, true, false, true,
					"/var/layer/lower"},
				{1, -3, O_RDONLY, 5, "/var/layer/upper", 0, true, false, true,
					"/var/layer/upper"},
				{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			},
			[]nsTestMount{
				{4, 1},
			},
		},
		{0, 6, 1, 0, 4,
			[]nsTestOpen{
				{1, 0, O_RDONLY, 2, "/var/layer/build/file1", int64(len(inFile)),
					true, false, false, "/var/layer/build/file1"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
			{0, "0:6", 2},
		}},
	})
}


func TestOverlayWriteFileFromLower(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "written to lower dir"
	topFile := "written to mounted overlay"
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopFile{Name: "/var/layer/lower/file1", Perms: 0644, Contents: inFile},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file, err := mos.OpenFile("/var/layer/build/file1", O_WRONLY, 0)
	if err != nil {
		T.Fatalf("error when opening file: %s", err)
	}
	n, err := file.Write([]byte(topFile))
	if err != nil {
		T.Fatalf("error writing file: %s", err)
	}
	if n != len(topFile) {
		T.Fatalf("expected to write %d bytes; wrote %d", len(topFile), n)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=9"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=8"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
			{nodeTypeFile, 0644, 0, 0, 1, topFile},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "file1=2"},
			{nodeTypeFile, 0644, 0, 0, 1, topFile},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0,
			[]nsTestOpen{
				{1, -5, O_RDONLY, 6, "/var/layer/work", 0, true, false, true,
					"/var/layer/work"},
				{1, -4, O_RDONLY, 7, "/var/layer/lower", 0, true, false, true,
					"/var/layer/lower"},
				{1, -3, O_RDONLY, 5, "/var/layer/upper", 0, true, false, true,
					"/var/layer/upper"},
				{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			},
			[]nsTestMount{
				{4, 1},
			},
		},
		{0, 6, 1, 0, 4,
			[]nsTestOpen{
				{1, 0, O_WRONLY, 2, "/var/layer/build/file1", int64(len(topFile)),
					false, true, false, "/var/layer/build/file1"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
			{0, "0:6", 2},
		}},
	})
}


func TestOverlayExistingUpperAndLowerFiles(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "written to lower dir"
	topFile := "written to overlay"
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopFile{Name: "/var/layer/lower/file1", Perms: 0644, Contents: inFile},
		PopFile{Name: "/var/layer/upper/file1", Perms: 0644, Contents: topFile},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	_, err = mos.Open("/var/layer/build/file1")
	if err != nil {
		T.Fatalf("error when opening file: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=9"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=8"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
			{nodeTypeFile, 0644, 0, 0, 1, topFile},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "file1=2"},
			{nodeTypeFile, 0644, 0, 0, 1, topFile},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0,
			[]nsTestOpen{
				{1, -5, O_RDONLY, 6, "/var/layer/work", 0, true, false, true,
					"/var/layer/work"},
				{1, -4, O_RDONLY, 7, "/var/layer/lower", 0, true, false, true,
					"/var/layer/lower"},
				{1, -3, O_RDONLY, 5, "/var/layer/upper", 0, true, false, true,
					"/var/layer/upper"},
				{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			},
			[]nsTestMount{
				{4, 1},
			},
		},
		{0, 6, 1, 0, 4,
			[]nsTestOpen{
				{1, 0, O_RDONLY, 2, "/var/layer/build/file1", 0,
					true, false, false, "/var/layer/build/file1"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
			{0, "0:6", 2},
		}},
	})
}


func TestOverlayFileCreationFromLowerWithStatAndLstat(T *testing.T) {
	inFile := "written to lower dir"
	setup := func () *MemOS {
		mos, err := NewMemOS()
		if err != nil {
			T.Fatal(err.Error())
		}
		populator := PopulatorType{
			PopDir{Name: "/var/layer/build", Perms: 0755},
			PopDir{Name: "/var/layer/upper", Perms: 0755},
			PopDir{Name: "/var/layer/work", Perms: 0755},
			PopDir{Name: "/var/layer/lower", Perms: 0755},
			PopFile{Name: "/var/layer/lower/file1", Perms: 0644, Contents: inFile},
			PopMount{Source: "overlay", Mountpoint: "/var/layer/build",
				Fstype: "overlay",
				Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
					"workdir=/var/layer/work"},
		}
		_, err = populator.Populate(mos)
		if err != nil {
			T.Fatalf("populator failure: %s", err)
		}
		return mos
	}
	testit  := func (mos *MemOS) {
		ns := mos.ns
		checkNSDevices(ns, T, nsTestDevices{
			{"0:2", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
				{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
				{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "file1=8"},
				{nodeTypeFile, 0644, 0, 0, 1, inFile},
			}},
			{"0:6", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "file1=2"},
				{nodeTypeFile, 0644, 0, 0, 1, inFile},
			}},
		})
		checkNSMounts(ns, T, nsTestMounts{
			{0, 2, 1, -1, 0,
				[]nsTestOpen{
					{1, -5, O_RDONLY, 6, "/var/layer/work", 0,
						true, false, true, "/var/layer/work"},
					{1, -4, O_RDONLY, 7, "/var/layer/lower", 0,
						true, false, true, "/var/layer/lower"},
					{1, -3, O_RDONLY, 5, "/var/layer/upper", 0,
						true, false, true, "/var/layer/upper"},
					{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
					{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				},
				[]nsTestMount{
					{4, 1},
				},
			},
			{0, 6, 1, 0, 4,
				[]nsTestOpen{
				}, nil},
		})
		checkNSProcesses(ns, T, nsTestProcesses{
			{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-5, "0:2", 6},
				{-4, "0:2", 7},
				{-3, "0:2", 5},
				{-2, "0:2", 1},
				{-1, "0:2", 1},
			}},
		})
	}
	var stat_buf Stat_t
	mos := setup()
	err := mos.SyscallStat("/var/layer/build/file1", &stat_buf)
	if err != nil {
		T.Fatalf("error when statting file: %s", err)
	}
	testit(mos)

	mos = setup()
	err = mos.SyscallLstat("/var/layer/build/file1", &stat_buf)
	if err != nil {
		T.Fatalf("error when statting file: %s", err)
	}
	testit(mos)
}


func TestOverlayFileCreationFromUpperWithStatAndLstat(T *testing.T) {
	inFile := "written to upper dir"
	setup := func () *MemOS {
		mos, err := NewMemOS()
		if err != nil {
			T.Fatal(err.Error())
		}
		populator := PopulatorType{
			PopDir{Name: "/var/layer/build", Perms: 0755},
			PopDir{Name: "/var/layer/upper", Perms: 0755},
			PopDir{Name: "/var/layer/work", Perms: 0755},
			PopDir{Name: "/var/layer/lower", Perms: 0755},
			PopFile{Name: "/var/layer/upper/file1", Perms: 0644, Contents: inFile},
			PopMount{Source: "overlay", Mountpoint: "/var/layer/build",
				Fstype: "overlay",
				Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
					"workdir=/var/layer/work"},
		}
		_, err = populator.Populate(mos)
		if err != nil {
			T.Fatalf("populator failure: %s", err)
		}
		return mos
	}
	testit  := func (mos *MemOS) {
		ns := mos.ns
		checkNSDevices(ns, T, nsTestDevices{
			{"0:2", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
				{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
				{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "file1=8"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeFile, 0644, 0, 0, 1, inFile},
			}},
			{"0:6", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "file1=2"},
				{nodeTypeFile, 0644, 0, 0, 1, inFile},
			}},
		})
		checkNSMounts(ns, T, nsTestMounts{
			{0, 2, 1, -1, 0,
				[]nsTestOpen{
					{1, -5, O_RDONLY, 6, "/var/layer/work", 0,
						true, false, true, "/var/layer/work"},
					{1, -4, O_RDONLY, 7, "/var/layer/lower", 0,
						true, false, true, "/var/layer/lower"},
					{1, -3, O_RDONLY, 5, "/var/layer/upper", 0,
						true, false, true, "/var/layer/upper"},
					{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
					{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				},
				[]nsTestMount{
					{4, 1},
				},
			},
			{0, 6, 1, 0, 4,
				[]nsTestOpen{
				}, nil},
		})
		checkNSProcesses(ns, T, nsTestProcesses{
			{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-5, "0:2", 6},
				{-4, "0:2", 7},
				{-3, "0:2", 5},
				{-2, "0:2", 1},
				{-1, "0:2", 1},
			}},
		})
	}
	var stat_buf Stat_t
	mos := setup()
	err := mos.SyscallStat("/var/layer/build/file1", &stat_buf)
	if err != nil {
		T.Fatalf("error when statting file: %s", err)
	}
	testit(mos)

	mos = setup()
	err = mos.SyscallLstat("/var/layer/build/file1", &stat_buf)
	if err != nil {
		T.Fatalf("error when statting file: %s", err)
	}
	testit(mos)
}


func TestOverlayFileCreationFromUpperAndLowerWithStatAndLstat(T *testing.T) {
	botFile := "written to lower dir"
	topFile := "written to upper dir"
	setup := func () *MemOS {
		mos, err := NewMemOS()
		if err != nil {
			T.Fatal(err.Error())
		}
		populator := PopulatorType{
			PopDir{Name: "/var/layer/build", Perms: 0755},
			PopDir{Name: "/var/layer/upper", Perms: 0755},
			PopDir{Name: "/var/layer/work", Perms: 0755},
			PopDir{Name: "/var/layer/lower", Perms: 0755},
			PopFile{Name: "/var/layer/lower/file1", Perms: 0644, Contents: botFile},
			PopFile{Name: "/var/layer/upper/file1", Perms: 0644, Contents: topFile},
			PopMount{Source: "overlay", Mountpoint: "/var/layer/build",
				Fstype: "overlay",
				Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
					"workdir=/var/layer/work"},
		}
		_, err = populator.Populate(mos)
		if err != nil {
			T.Fatalf("populator failure: %s", err)
		}
		return mos
	}
	testit  := func (mos *MemOS) {
		ns := mos.ns
		checkNSDevices(ns, T, nsTestDevices{
			{"0:2", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
				{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
				{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "file1=9"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "file1=8"},
				{nodeTypeFile, 0644, 0, 0, 1, botFile},
				{nodeTypeFile, 0644, 0, 0, 1, topFile},
			}},
			{"0:6", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "file1=2"},
				{nodeTypeFile, 0644, 0, 0, 1, topFile},
			}},
		})
		checkNSMounts(ns, T, nsTestMounts{
			{0, 2, 1, -1, 0,
				[]nsTestOpen{
					{1, -5, O_RDONLY, 6, "/var/layer/work", 0,
						true, false, true, "/var/layer/work"},
					{1, -4, O_RDONLY, 7, "/var/layer/lower", 0,
						true, false, true, "/var/layer/lower"},
					{1, -3, O_RDONLY, 5, "/var/layer/upper", 0,
						true, false, true, "/var/layer/upper"},
					{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
					{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				},
				[]nsTestMount{
					{4, 1},
				},
			},
			{0, 6, 1, 0, 4,
				[]nsTestOpen{
				}, nil},
		})
		checkNSProcesses(ns, T, nsTestProcesses{
			{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-5, "0:2", 6},
				{-4, "0:2", 7},
				{-3, "0:2", 5},
				{-2, "0:2", 1},
				{-1, "0:2", 1},
			}},
		})
	}
	var stat_buf Stat_t
	mos := setup()
	err := mos.SyscallStat("/var/layer/build/file1", &stat_buf)
	if err != nil {
		T.Fatalf("error when statting file: %s", err)
	}
	testit(mos)

	mos = setup()
	err = mos.SyscallLstat("/var/layer/build/file1", &stat_buf)
	if err != nil {
		T.Fatalf("error when statting file: %s", err)
	}
	testit(mos)
}


func TestOverlayCopyUpFromLowerOnWrite(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	botFile := "written to lower dir"
	toAppend := " then modified"
	topFile := botFile + toAppend
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopFile{Name: "/var/layer/lower/file1", Perms: 0644, Contents: botFile},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file, err := mos.OpenFile("/var/layer/build/file1", O_APPEND | O_WRONLY, 0)
	if err != nil {
		T.Fatalf("error when opening file: %s", err)
	}
	n, err := file.Write([]byte(toAppend))
	if err != nil {
		T.Fatalf("error writing file: %s", err)
	}
	if n != len(toAppend) {
		T.Fatalf("expected to write %d bytes; wrote %d", len(toAppend), n)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=9"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=8"},
			{nodeTypeFile, 0644, 0, 0, 1, botFile},
			{nodeTypeFile, 0644, 0, 0, 1, topFile},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "file1=2"},
			{nodeTypeFile, 0644, 0, 0, 1, topFile},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0,
			[]nsTestOpen{
				{1, -5, O_RDONLY, 6, "/var/layer/work", 0, true, false, true,
					"/var/layer/work"},
				{1, -4, O_RDONLY, 7, "/var/layer/lower", 0, true, false, true,
					"/var/layer/lower"},
				{1, -3, O_RDONLY, 5, "/var/layer/upper", 0, true, false, true,
					"/var/layer/upper"},
				{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			},
			[]nsTestMount{
				{4, 1},
			},
		},
		{0, 6, 1, 0, 4,
			[]nsTestOpen{
				{1, 0, O_APPEND | O_WRONLY, 2, "/var/layer/build/file1",
					int64(len(topFile)),
					false, true, false, "/var/layer/build/file1"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
			{0, "0:6", 2},
		}},
	})
}


func TestOverlayCopyUpFromLowerOnWriteWithExistingUpper(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	botFile := "written to lower dir"
	origTopFile := "written to upper dir"
	toWrite := "visible on"
	topFile := "visible on upper dir"
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopFile{Name: "/var/layer/lower/file1", Perms: 0644, Contents: botFile},
		PopFile{Name: "/var/layer/upper/file1", Perms: 0664, Contents: origTopFile},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file, err := mos.OpenFile("/var/layer/build/file1", O_WRONLY, 0)
	if err != nil {
		T.Fatalf("error when opening file: %s", err)
	}
	n, err := file.Write([]byte(toWrite))
	if err != nil {
		T.Fatalf("error writing file: %s", err)
	}
	if n != len(toWrite) {
		T.Fatalf("expected to write %d bytes; wrote %d", len(toWrite), n)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=9"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=8"},
			{nodeTypeFile, 0644, 0, 0, 1, botFile},
			{nodeTypeFile, 0644, 0, 0, 1, topFile},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "file1=2"},
			{nodeTypeFile, 0644, 0, 0, 1, topFile},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0,
			[]nsTestOpen{
				{1, -5, O_RDONLY, 6, "/var/layer/work", 0, true, false, true,
					"/var/layer/work"},
				{1, -4, O_RDONLY, 7, "/var/layer/lower", 0, true, false, true,
					"/var/layer/lower"},
				{1, -3, O_RDONLY, 5, "/var/layer/upper", 0, true, false, true,
					"/var/layer/upper"},
				{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			},
			[]nsTestMount{
				{4, 1},
			},
		},
		{0, 6, 1, 0, 4,
			[]nsTestOpen{
				{1, 0, O_WRONLY, 2, "/var/layer/build/file1", int64(len(toWrite)),
					false, true, false, "/var/layer/build/file1"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
			{0, "0:6", 2},
		}},
	})
}


func TestCreateFileInEmptyOverlay(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	_, err = mos.Create("/var/layer/build/file1")
	if err != nil {
		T.Fatalf("error when creating file: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=8"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "file1=2"},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0,
			[]nsTestOpen{
				{1, -5, O_RDONLY, 6, "/var/layer/work", 0, true, false, true,
					"/var/layer/work"},
				{1, -4, O_RDONLY, 7, "/var/layer/lower", 0, true, false, true,
					"/var/layer/lower"},
				{1, -3, O_RDONLY, 5, "/var/layer/upper", 0, true, false, true,
					"/var/layer/upper"},
				{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			},
			[]nsTestMount{
				{4, 1},
			},
		},
		{0, 6, 1, 0, 4,
			[]nsTestOpen{
				{1, 0, O_WRONLY | O_CREATE | O_TRUNC, 2, "/var/layer/build/file1",
					0, false, true, false, "/var/layer/build/file1"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
			{0, "0:6", 2},
		}},
	})
}


func TestOverlayOpenFileInteriorNoCopyUp(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "app-dicts/latin-words tools"
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopDir{Name: "/var/layer/lower/etc/portage", Perms: 0755},
		PopFile{Name: "/var/layer/lower/etc/portage/package.use", Perms: 0644,
			Contents: inFile},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file, err := mos.Open("/var/layer/build/etc/portage/package.use")
	if err != nil {
		T.Fatalf("error when opening file: %s", err)
	}
	buf := make([]byte, 30)
	n, err := file.Read(buf)
	if err != nil {
		T.Fatalf("error reading file: %s", err)
	}
	if string(buf[:n]) != inFile {
		T.Fatalf("expected to read '%s'; got '%s'", inFile, buf[:n])
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "etc=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=9"},
			{nodeTypeDir, 0755, 0, 0, 1, "package.use=10"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "etc=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "package.use=4"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0,
			[]nsTestOpen{
				{1, -5, O_RDONLY, 6, "/var/layer/work", 0, true, false, true,
					"/var/layer/work"},
				{1, -4, O_RDONLY, 7, "/var/layer/lower", 0, true, false, true,
					"/var/layer/lower"},
				{1, -3, O_RDONLY, 5, "/var/layer/upper", 0, true, false, true,
					"/var/layer/upper"},
				{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			},
			[]nsTestMount{
				{4, 1},
			},
		},
		{0, 6, 1, 0, 4,
			[]nsTestOpen{
				{1, 0, O_RDONLY, 4, "/var/layer/build/etc/portage/package.use",
					int64(len(inFile)), true, false, false,
					"/var/layer/build/etc/portage/package.use"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
			{0, "0:6", 4},
		}},
	})
}


func TestOverlayOpenFileInteriorForceCopyUp(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "app-dicts/latin-words tools"
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopDir{Name: "/var/layer/lower/etc/portage", Perms: 0755},
		PopFile{Name: "/var/layer/lower/etc/portage/package.use", Perms: 0644,
			Contents: inFile},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file, err := mos.OpenFile("/var/layer/build/etc/portage/package.use", O_RDWR, 0)
	if err != nil {
		T.Fatalf("error when opening file: %s", err)
	}
	buf := make([]byte, 30)
	n, err := file.Read(buf)
	if err != nil {
		T.Fatalf("error reading file: %s", err)
	}
	if string(buf[:n]) != inFile {
		T.Fatalf("expected to read '%s'; got '%s'", inFile, buf[:n])
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "etc=11"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "etc=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=9"},
			{nodeTypeDir, 0755, 0, 0, 1, "package.use=10"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=12"},
			{nodeTypeDir, 0755, 0, 0, 1, "package.use=13"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "etc=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "package.use=4"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0,
			[]nsTestOpen{
				{1, -5, O_RDONLY, 6, "/var/layer/work", 0, true, false, true,
					"/var/layer/work"},
				{1, -4, O_RDONLY, 7, "/var/layer/lower", 0, true, false, true,
					"/var/layer/lower"},
				{1, -3, O_RDONLY, 5, "/var/layer/upper", 0, true, false, true,
					"/var/layer/upper"},
				{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			},
			[]nsTestMount{
				{4, 1},
			},
		},
		{0, 6, 1, 0, 4,
			[]nsTestOpen{
				{1, 0, O_RDWR, 4, "/var/layer/build/etc/portage/package.use",
					int64(len(inFile)), true, true, false,
					"/var/layer/build/etc/portage/package.use"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
			{0, "0:6", 4},
		}},
	})
}


func TestOverlayCopyUpTimestamp(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "app-dicts/latin-words tools"
	inAtime, _ := time.Parse(time.DateTime, "2023-01-15 15:42:10")
	inMtime, _ := time.Parse(time.DateTime, "2022-10-24 04:16:22")
	inMtimeNS := timeToTimespec(inMtime)
	referenceNowNS := time.Now().UnixNano()
	maxNowSlop := int64(time.Millisecond)
	nowEnough := func (ts syscall.Timespec) bool {
		diff := ts.Nano() - referenceNowNS
		return diff >= 0 && diff < maxNowSlop
	}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopDir{Name: "/var/layer/lower/etc/portage", Perms: 0755},
		PopFile{Name: "/var/layer/lower/etc/portage/package.use", Perms: 0644,
			Contents: inFile},
		PopChtimes{Name: "/var/layer/lower/etc/portage/package.use", Atime: inAtime,
			Mtime: inMtime},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	var stat_buf Stat_t
	err = mos.SyscallStat("/var/layer/build/etc/portage/package.use", &stat_buf)
	if err != nil {
		T.Fatalf("error statting file: %s", err)
	}
	if stat_buf.Mtim != inMtimeNS {
		T.Fatalf("expected to read lower Mtime of %s", inMtime)
	}
	file, err := mos.OpenFile("/var/layer/build/etc/portage/package.use", O_RDWR, 0)
	if err != nil {
		T.Fatalf("error when opening file: %s", err)
	}
	err = mos.SyscallStat("/var/layer/lower/etc/portage/package.use", &stat_buf)
	if err != nil {
		T.Fatalf("error statting file: %s", err)
	}
	if stat_buf.Mtim != inMtimeNS {
		T.Fatalf("expected to read lower Mtime of %s", inMtime)
	}
	err = mos.SyscallStat("/var/layer/build/etc/portage/package.use", &stat_buf)
	if err != nil {
		T.Fatalf("error statting file: %s", err)
	}
	if !nowEnough(stat_buf.Mtim) {
		T.Fatal("expected to read merge Mtime as Now")
	}
	err = mos.SyscallStat("/var/layer/upper/etc/portage/package.use", &stat_buf)
	if err != nil {
		T.Fatalf("error statting file: %s", err)
	}
	if !nowEnough(stat_buf.Mtim) {
		T.Fatal("expected to read upper Mtime as Now")
	}
	buf := make([]byte, 30)
	n, err := file.Read(buf)
	if err != nil {
		T.Fatalf("error reading file: %s", err)
	}
	if string(buf[:n]) != inFile {
		T.Fatalf("expected to read '%s'; got '%s'", inFile, buf[:n])
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "etc=11"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "etc=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=9"},
			{nodeTypeDir, 0755, 0, 0, 1, "package.use=10"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=12"},
			{nodeTypeDir, 0755, 0, 0, 1, "package.use=13"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "etc=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "package.use=4"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0,
			[]nsTestOpen{
				{1, -5, O_RDONLY, 6, "/var/layer/work", 0, true, false, true,
					"/var/layer/work"},
				{1, -4, O_RDONLY, 7, "/var/layer/lower", 0, true, false, true,
					"/var/layer/lower"},
				{1, -3, O_RDONLY, 5, "/var/layer/upper", 0, true, false, true,
					"/var/layer/upper"},
				{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			},
			[]nsTestMount{
				{4, 1},
			},
		},
		{0, 6, 1, 0, 4,
			[]nsTestOpen{
				{1, 0, O_RDWR, 4, "/var/layer/build/etc/portage/package.use",
					int64(len(inFile)), true, true, false,
					"/var/layer/build/etc/portage/package.use"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
			{0, "0:6", 4},
		}},
	})
}


func TestOverlaySetMergeTimestamp(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "app-dicts/latin-words tools"
	lowerMtime, _ := time.Parse(time.DateTime, "2023-01-15 15:42:10")
	lowerMtimeNS := timeToTimespec(lowerMtime)
	upperMtime, _ := time.Parse(time.DateTime, "2023-02-12 05:49:24")
	upperMtimeNS := timeToTimespec(upperMtime)
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopDir{Name: "/var/layer/lower/etc/portage", Perms: 0755},
		PopFile{Name: "/var/layer/lower/etc/portage/package.use", Perms: 0644,
			Contents: inFile},
		PopChtimes{Name: "/var/layer/lower/etc/portage/package.use", Atime: lowerMtime,
			Mtime: lowerMtime},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file, err := mos.OpenFile("/var/layer/build/etc/portage/package.use", O_RDWR, 0)
	if err != nil {
		T.Fatalf("error when opening file: %s", err)
	}
	var stat_buf Stat_t
	mos.Chtimes("/var/layer/build/etc/portage/package.use", upperMtime, upperMtime)
	err = mos.SyscallStat("/var/layer/lower/etc/portage/package.use", &stat_buf)
	if err != nil {
		T.Fatalf("error statting file: %s", err)
	}
	if stat_buf.Mtim != lowerMtimeNS {
		T.Fatalf("expected to read lower Mtime of %s", lowerMtime)
	}
	err = mos.SyscallStat("/var/layer/build/etc/portage/package.use", &stat_buf)
	if err != nil {
		T.Fatalf("error statting file: %s", err)
	}
	if stat_buf.Mtim != upperMtimeNS {
		T.Fatalf("expected to read merge Mtime of %s", upperMtime)
	}
	err = mos.SyscallStat("/var/layer/upper/etc/portage/package.use", &stat_buf)
	if err != nil {
		T.Fatalf("error statting file: %s", err)
	}
	if stat_buf.Mtim != upperMtimeNS {
		T.Fatalf("expected to upper merge Mtime of %s", upperMtime)
	}
	buf := make([]byte, 30)
	n, err := file.Read(buf)
	if err != nil {
		T.Fatalf("error reading file: %s", err)
	}
	if string(buf[:n]) != inFile {
		T.Fatalf("expected to read '%s'; got '%s'", inFile, buf[:n])
	}
	err = mos.SyscallStat("/var/layer/build/etc/portage/package.use", &stat_buf)
	if err != nil {
		T.Fatalf("error statting file: %s", err)
	}
	if stat_buf.Mtim != upperMtimeNS {
		T.Fatalf("expected to read merge Mtime of %s", upperMtime)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "etc=11"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "etc=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=9"},
			{nodeTypeDir, 0755, 0, 0, 1, "package.use=10"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=12"},
			{nodeTypeDir, 0755, 0, 0, 1, "package.use=13"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "etc=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "portage=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "package.use=4"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
		}},
	})
	checkNSMounts(ns, T, nsTestMounts{
		{0, 2, 1, -1, 0,
			[]nsTestOpen{
				{1, -5, O_RDONLY, 6, "/var/layer/work", 0, true, false, true,
					"/var/layer/work"},
				{1, -4, O_RDONLY, 7, "/var/layer/lower", 0, true, false, true,
					"/var/layer/lower"},
				{1, -3, O_RDONLY, 5, "/var/layer/upper", 0, true, false, true,
					"/var/layer/upper"},
				{1, -2, O_RDONLY, 1, "/", 0, true, false, true, "/"},
				{1, -1, O_RDONLY, 1, "/", 0, true, false, true, "/"},
			},
			[]nsTestMount{
				{4, 1},
			},
		},
		{0, 6, 1, 0, 4,
			[]nsTestOpen{
				{1, 0, O_RDWR, 4, "/var/layer/build/etc/portage/package.use",
					int64(len(inFile)), true, true, false,
					"/var/layer/build/etc/portage/package.use"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
			{0, "0:6", 4},
		}},
	})
}

