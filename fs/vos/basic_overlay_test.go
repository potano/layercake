// Copyright © 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import (
	"time"
	"strings"
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
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, ""},
		}},
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
	checkOverlayFS(ns, T, ovTestDevices{})
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
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "file1=8"},
			{nodeTypeFile, false, 8, 1, inFile},
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
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "file1=9"},
			{nodeTypeFile, true, 9, 1, topFile},
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
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "file1=9"},
			{nodeTypeFile, true, 9, 1, topFile},
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
		checkOverlayFS(ns, T, ovTestDevices{
			{"0:6", []ovTestInode{
				{nodeTypeDir, true, 5, 1, "file1=8"},
				{nodeTypeFile, false, 8, 1, inFile},
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
		checkOverlayFS(ns, T, ovTestDevices{
			{"0:6", []ovTestInode{
				{nodeTypeDir, true, 5, 1, "file1=8"},
				{nodeTypeFile, true, 8, 1, inFile},
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
		checkOverlayFS(ns, T, ovTestDevices{
			{"0:6", []ovTestInode{
				{nodeTypeDir, true, 5, 1, "file1=9"},
				{nodeTypeFile, true, 9, 1, topFile},
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
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "file1=9"},
			{nodeTypeFile, true, 9, 1, topFile},
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
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "file1=9"},
			{nodeTypeFile, true, 9, 1, topFile},
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
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "file1=8"},
			{nodeTypeFile, true, 8, 1, ""},
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
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "etc=8"},
			{nodeTypeDir, false, 8, 1, "portage=9"},
			{nodeTypeDir, false, 9, 1, "package.use=10"},
			{nodeTypeFile, false, 10, 1, inFile},
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
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "etc=11"},
			{nodeTypeDir, true, 11, 1, "portage=12"},
			{nodeTypeDir, true, 12, 1, "package.use=13"},
			{nodeTypeFile, true, 13, 1, inFile},
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
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "etc=11"},
			{nodeTypeDir, true, 11, 1, "portage=12"},
			{nodeTypeDir, true, 12, 1, "package.use=13"},
			{nodeTypeFile, true, 13, 1, inFile},
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
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "etc=11"},
			{nodeTypeDir, true, 11, 1, "portage=12"},
			{nodeTypeDir, true, 12, 1, "package.use=13"},
			{nodeTypeFile, true, 13, 1, inFile},
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


func TestOverlayReaddirnamesFromLowerRoot(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "written to lower dir"
	wantFiles := []string{"file1", "file2"}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopFile{Name: "/var/layer/lower/file1", Perms: 0644, Contents: inFile},
		PopFile{Name: "/var/layer/lower/file2", Perms: 0644, Contents: ""},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	dir, err := mos.Open("/var/layer/build")
	if err != nil {
		T.Fatalf("error when opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory: %s", err)
	}
	wantString := strings.Join(wantFiles, " ")
	gotString := strings.Join(got, " ")
	if gotString != wantString {
		T.Fatalf("expected directory to have entries  '%s'; got '%s'",
			wantString, gotString)
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
			{nodeTypeDir, 0755, 0, 0, 1, "file1=8\nfile2=9"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "file1=8\nfile2=9"},
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
				{1, 0, O_RDONLY, 1, "/var/layer/build", 0, true, false, true,
					"/var/layer/build"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
			{0, "0:6", 1},
		}},
	})
}


func TestOverlayReaddirnamesFromUpperRoot(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "written to lower dir"
	wantFiles := []string{"file1", "file2"}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopFile{Name: "/var/layer/upper/file1", Perms: 0644, Contents: inFile},
		PopFile{Name: "/var/layer/upper/file2", Perms: 0644, Contents: ""},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	dir, err := mos.Open("/var/layer/build")
	if err != nil {
		T.Fatalf("error when opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory: %s", err)
	}
	wantString := strings.Join(wantFiles, " ")
	gotString := strings.Join(got, " ")
	if gotString != wantString {
		T.Fatalf("expected directory to have entries  '%s'; got '%s'",
			wantString, gotString)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=8\nfile2=9"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "file1=8\nfile2=9"},
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
				{1, 0, O_RDONLY, 1, "/var/layer/build", 0, true, false, true,
					"/var/layer/build"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
			{0, "0:6", 1},
		}},
	})
}


func TestOverlayReaddirnamesFromMergedRoot(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "written to lower dir"
	wantFiles := []string{"file1", "file2", "file3"}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopFile{Name: "/var/layer/lower/file1", Perms: 0644, Contents: inFile},
		PopFile{Name: "/var/layer/lower/file2", Perms: 0644, Contents: ""},
		PopFile{Name: "/var/layer/upper/file3", Perms: 0644, Contents: ""},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	dir, err := mos.Open("/var/layer/build")
	if err != nil {
		T.Fatalf("error when opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory: %s", err)
	}
	wantString := strings.Join(wantFiles, " ")
	gotString := strings.Join(got, " ")
	if gotString != wantString {
		T.Fatalf("expected directory to have entries  '%s'; got '%s'",
			wantString, gotString)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file3=10"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=8\nfile2=9"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "file1=8\nfile2=9\nfile3=10"},
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
				{1, 0, O_RDONLY, 1, "/var/layer/build", 0, true, false, true,
					"/var/layer/build"},
			}, nil},
	})
	checkNSProcesses(ns, T, nsTestProcesses{
		{1, 0, 0, -1, -2, []nsTestProcOpen{
			{-5, "0:2", 6},
			{-4, "0:2", 7},
			{-3, "0:2", 5},
			{-2, "0:2", 1},
			{-1, "0:2", 1},
			{0, "0:6", 1},
		}},
	})
}


func TestOverlayReaddirnamesFromLowerInterior(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "written to lower dir"
	wantFiles := []string{"file1", "file2"}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopDir{Name: "/var/layer/lower/inner", Perms: 0755},
		PopFile{Name: "/var/layer/lower/inner/file1", Perms: 0644, Contents: inFile},
		PopFile{Name: "/var/layer/lower/inner/file2", Perms: 0644, Contents: ""},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	dir, err := mos.Open("/var/layer/build/inner")
	if err != nil {
		T.Fatalf("error when opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory: %s", err)
	}
	wantString := strings.Join(wantFiles, " ")
	gotString := strings.Join(got, " ")
	if gotString != wantString {
		T.Fatalf("expected directory to have entries  '%s'; got '%s'",
			wantString, gotString)
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
			{nodeTypeDir, 0755, 0, 0, 1, "inner=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=9\nfile2=10"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "inner=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "inner=8"},
			{nodeTypeDir, false, 8, 1, "file1=9\nfile2=10"},
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
				{1, 0, O_RDONLY, 2, "/var/layer/build/inner", 0, true, false, true,
					"/var/layer/build/inner"},
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


func TestOverlayReaddirnamesFromUpperInterior(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "written to lower dir"
	wantFiles := []string{"file1", "file2"}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopDir{Name: "/var/layer/upper/inner", Perms: 0755},
		PopFile{Name: "/var/layer/upper/inner/file1", Perms: 0644, Contents: inFile},
		PopFile{Name: "/var/layer/upper/inner/file2", Perms: 0644, Contents: ""},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	dir, err := mos.Open("/var/layer/build/inner")
	if err != nil {
		T.Fatalf("error when opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory: %s", err)
	}
	wantString := strings.Join(wantFiles, " ")
	gotString := strings.Join(got, " ")
	if gotString != wantString {
		T.Fatalf("expected directory to have entries  '%s'; got '%s'",
			wantString, gotString)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "inner=8"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=9\nfile2=10"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "inner=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "inner=8"},
			{nodeTypeDir, true, 8, 1, "file1=9\nfile2=10"},
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
				{1, 0, O_RDONLY, 2, "/var/layer/build/inner", 0, true, false, true,
					"/var/layer/build/inner"},
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


func TestOverlayReaddirnamesFromMergedInterior(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "written to lower dir"
	wantFiles := []string{"file0", "file1", "file2", "file3"}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopDir{Name: "/var/layer/lower/inner", Perms: 0755},
		PopFile{Name: "/var/layer/lower/inner/file1", Perms: 0644, Contents: inFile},
		PopFile{Name: "/var/layer/lower/inner/file2", Perms: 0644, Contents: ""},
		PopDir{Name: "/var/layer/upper/inner", Perms: 0755},
		PopFile{Name: "/var/layer/upper/inner/file0", Perms: 0644, Contents: ""},
		PopFile{Name: "/var/layer/upper/inner/file1", Perms: 0644, Contents: ""},
		PopFile{Name: "/var/layer/upper/inner/file3", Perms: 0644, Contents: ""},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	dir, err := mos.Open("/var/layer/build/inner")
	if err != nil {
		T.Fatalf("error when opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory: %s", err)
	}
	wantString := strings.Join(wantFiles, " ")
	gotString := strings.Join(got, " ")
	if gotString != wantString {
		T.Fatalf("expected directory to have entries  '%s'; got '%s'",
			wantString, gotString)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "inner=11"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "inner=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=9\nfile2=10"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file0=12\nfile1=13\nfile3=14"},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "inner=2"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "inner=11"},
			{nodeTypeDir, true, 11, 1, "file0=12\nfile1=13\nfile2=10\nfile3=14"},
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
				{1, 0, O_RDONLY, 2, "/var/layer/build/inner", 0, true, false, true,
					"/var/layer/build/inner"},
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


func TestOverlayCreateFileThenReaddirnames(T *testing.T) {
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	inFile := "written to lower dir"
	wantFiles := []string{"file1", "file2", "file3"}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopDir{Name: "/var/layer/lower/inner", Perms: 0755},
		PopFile{Name: "/var/layer/lower/inner/file1", Perms: 0644, Contents: inFile},
		PopFile{Name: "/var/layer/lower/inner/file2", Perms: 0644, Contents: ""},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build", Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	file, err := mos.Create("/var/layer/build/inner/file3")
	if err != nil {
		T.Fatalf("error creating file: %s\n", err)
	}
	file.Close()
	dir, err := mos.Open("/var/layer/build/inner")
	if err != nil {
		T.Fatalf("error when opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory: %s", err)
	}
	wantString := strings.Join(wantFiles, " ")
	gotString := strings.Join(got, " ")
	if gotString != wantString {
		T.Fatalf("expected directory to have entries  '%s'; got '%s'",
			wantString, gotString)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "inner=11"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "inner=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=9\nfile2=10"},
			{nodeTypeFile, 0644, 0, 0, 1, inFile},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "file3=12"},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "inner=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "file3=3"},
			{nodeTypeFile, 0644, 0, 0, 1, ""},
		}},
	})
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "inner=11"},
			{nodeTypeDir, true, 11, 1, "file1=9\nfile2=10\nfile3=12"},
			{nodeTypeFile, true, 12, 1, ""},
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
				{1, 0, O_RDONLY, 2, "/var/layer/build/inner", 0, true, false, true,
					"/var/layer/build/inner"},
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


func TestOverlayRemoveFileFromLower(T *testing.T) {
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
		err = mos.Remove("/var/layer/build/file1")
		if err != nil {
			T.Fatalf("error when removing file: %s", err)
		}
		return mos
	}
	testit := func (mos *MemOS) {
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
				{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
			}},
			{"0:6", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeFile, 0644, 0, 0, 0, inFile},
			}},
		})
		checkOverlayFS(ns, T, ovTestDevices{
			{"0:6", []ovTestInode{
				{nodeTypeDir, true, 5, 1, ""},
				{nodeTypeFile, false, 8, 0, inFile},
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
			{0, 6, 1, 0, 4, nil, nil},
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

	mos := setup()
	testit(mos)

	mos = setup()
	var stat_buf Stat_t
	err := mos.SyscallLstat("/var/layer/upper/file1", &stat_buf)
	if err != nil {
		T.Fatalf("unexpected error when statting deleted file; got %s", err)
	}
	checkStat(T, "", stat_buf, Stat_t{
		Dev: mos.ns.mounts[0].st_dev,
		Ino: 9,
		Nlink: 1,
		Mode: syscall.S_IFCHR,
	})
	err = mos.SyscallLstat("/var/layer/build/file1", &stat_buf)
	if err != ENOENT {
		T.Fatalf("expected ENOENT when statting deleted file; got %s", err)
	}
	testit(mos)

	mos = setup()
	dir, err := mos.Open("/var/layer/build")
	if err != nil {
		T.Fatalf("error opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory: %s", err)
	}
	if len(got) > 0 {
		T.Fatalf("expected to read empty directory; got %s", strings.Join(got, ", "))
	}
	dir.Close()
	testit(mos)
}


func TestOverlayRemoveFileFromUpperHidingLower(T *testing.T) {
	botFile := "written to lower dir"
	topFile := "now up upper dir"
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
		err = mos.Remove("/var/layer/build/file1")
		if err != nil {
			T.Fatalf("error when removing file: %s", err)
		}
		return mos
	}
	testit := func (mos *MemOS) {
		ns := mos.ns
		checkNSDevices(ns, T, nsTestDevices{
			{"0:2", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
				{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
				{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "file1=10"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "file1=8"},
				{nodeTypeFile, 0644, 0, 0, 1, botFile},
				{nodeTypeFile, 0644, 0, 0, 0, topFile},
				{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
			}},
			{"0:6", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeFile, 0644, 0, 0, 0, topFile},
			}},
		})
		checkOverlayFS(ns, T, ovTestDevices{
			{"0:6", []ovTestInode{
				{nodeTypeDir, true, 5, 1, ""},
				{nodeTypeFile, true, 9, 0, topFile},
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
			{0, 6, 1, 0, 4, nil, nil},
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

	mos := setup()
	testit(mos)

	mos = setup()
	var stat_buf Stat_t
	err := mos.SyscallLstat("/var/layer/upper/file1", &stat_buf)
	if err != nil {
		T.Fatalf("unexpected error when statting deleted file; got %s", err)
	}
	checkStat(T, "", stat_buf, Stat_t{
		Dev: mos.ns.mounts[0].st_dev,
		Ino: 10,
		Nlink: 1,
		Mode: syscall.S_IFCHR,
	})
	err = mos.SyscallLstat("/var/layer/build/file1", &stat_buf)
	if err != ENOENT {
		T.Fatalf("expected ENOENT when statting deleted file; got %s", err)
	}
	testit(mos)

	mos = setup()
	dir, err := mos.Open("/var/layer/build")
	if err != nil {
		T.Fatalf("error opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory: %s", err)
	}
	if len(got) > 0 {
		T.Fatalf("expected to read empty directory; got %s", strings.Join(got, ", "))
	}
	dir.Close()
	testit(mos)
}


func TestOverlayRemoveFileFromUpperNoLower(T *testing.T) {
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
		err = mos.Remove("/var/layer/build/file1")
		if err != nil {
			T.Fatalf("error when removing file: %s", err)
		}
		return mos
	}
	testit := func (mos *MemOS) {
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
				{nodeTypeFile, 0644, 0, 0, 0, inFile},
			}},
			{"0:6", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeFile, 0644, 0, 0, 0, inFile},
			}},
		})
		checkOverlayFS(ns, T, ovTestDevices{
			{"0:6", []ovTestInode{
				{nodeTypeDir, true, 5, 1, ""},
				{nodeTypeFile, true, 8, 0, inFile},
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
			{0, 6, 1, 0, 4, nil, nil},
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

	mos := setup()
	testit(mos)

	mos = setup()
	var stat_buf Stat_t
	err := mos.SyscallLstat("/var/layer/upper/file1", &stat_buf)
	if err != ENOENT {
		T.Fatalf("expected ENOENT when statting deleted upper file; got %s", err)
	}
	err = mos.SyscallLstat("/var/layer/build/file1", &stat_buf)
	if err != ENOENT {
		T.Fatalf("expected ENOENT when statting deleted file; got %s", err)
	}
	testit(mos)

	mos = setup()
	dir, err := mos.Open("/var/layer/build")
	if err != nil {
		T.Fatalf("error opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory: %s", err)
	}
	if len(got) > 0 {
		T.Fatalf("expected to read empty directory; got %s", strings.Join(got, ", "))
	}
	dir.Close()
	testit(mos)
}


func TestOverlayCreateFileOverWhiteout(T *testing.T) {
	botFile := "written to lower file"
	topFile := "now up upper file"
	newFile := "contents of new file"
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
		err = mos.Remove("/var/layer/build/file1")
		if err != nil {
			T.Fatalf("error when removing file: %s", err)
		}
		file, err := mos.Create("/var/layer/build/file1")
		if err != nil {
			T.Fatalf("error creating file: %s", err)
		}
		_, err = file.Write([]byte(newFile))
		if err != nil {
			T.Fatalf("error writing to new file: %s", err)
		}
		return mos
	}
	testit := func (mos *MemOS) {
		ns := mos.ns
		checkNSDevices(ns, T, nsTestDevices{
			{"0:2", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
				{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
				{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "file1=11"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "file1=8"},
				{nodeTypeFile, 0644, 0, 0, 1, botFile},
				{nodeTypeFile, 0644, 0, 0, 0, topFile},
				{nodeTypeCharDev, 0000, 0, 0, 0, "0:0"},
				{nodeTypeFile, 0644, 0, 0, 1, newFile},
			}},
			{"0:6", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "file1=3"},
				{nodeTypeFile, 0644, 0, 0, 0, topFile},
				{nodeTypeFile, 0644, 0, 0, 1, newFile},
			}},
		})
		checkOverlayFS(ns, T, ovTestDevices{
			{"0:6", []ovTestInode{
				{nodeTypeDir, true, 5, 1, "file1=11"},
				{nodeTypeFile, true, 9, 0, topFile},
				{nodeTypeFile, true, 11, 1, newFile},
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
					{1, 0, O_WRONLY | O_CREATE | O_TRUNC, 3,
						"/var/layer/build/file1", int64(len(newFile)),
						false, true, false, "/var/layer/build/file1"},
				},
				nil,
			},
		})
		checkNSProcesses(ns, T, nsTestProcesses{
			{1, 0, 0, -1, -2, []nsTestProcOpen{
				{-5, "0:2", 6},
				{-4, "0:2", 7},
				{-3, "0:2", 5},
				{-2, "0:2", 1},
				{-1, "0:2", 1},
				{0, "0:6", 3},
			}},
		})
	}

	mos := setup()
	testit(mos)

	mos = setup()
	dir, err := mos.Open("/var/layer/build")
	if err != nil {
		T.Fatalf("error opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory: %s", err)
	}
	gotString := strings.Join(got, " ")
	if gotString != "file1" {
		T.Fatalf("expected to read 'file1' from directory; got %s", gotString)
	}
	dir.Close()
	testit(mos)

	mos = setup()
	file, err := mos.Open("/var/layer/build/file1")
	if err != nil {
		T.Fatalf("error opening file for reading: %s", err)
	}
	buf := make([]byte, 30)
	n, err := file.Read(buf)
	if err != nil {
		T.Fatalf("error reading file: %s", err)
	}
	gotString = string(buf[:n])
	if gotString != newFile {
		T.Fatalf("read '%s' from file; expected '%s'", gotString, newFile)
	}
	file.Close()
	testit(mos)
}


func TestOverlayRemoveFileFromLowerOnLowerInterior(T *testing.T) {
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
			PopDir{Name: "/var/layer/lower/inner", Perms: 0755},
			PopFile{Name: "/var/layer/lower/inner/file1", Perms: 0644,
				Contents: inFile},
			PopMount{Source: "overlay", Mountpoint: "/var/layer/build",
				Fstype: "overlay",
				Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
					"workdir=/var/layer/work"},
		}
		_, err = populator.Populate(mos)
		if err != nil {
			T.Fatalf("populator failure: %s", err)
		}
		err = mos.Remove("/var/layer/build/inner/file1")
		if err != nil {
			T.Fatalf("error when removing file: %s", err)
		}
		return mos
	}
	testit := func (mos *MemOS) {
		ns := mos.ns
		checkNSDevices(ns, T, nsTestDevices{
			{"0:2", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
				{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
				{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "inner=10"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "inner=8"},
				{nodeTypeDir, 0755, 0, 0, 1, "file1=9"},
				{nodeTypeFile, 0644, 0, 0, 1, inFile},
				{nodeTypeDir, 0755, 0, 0, 1, "file1=11"},
				{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
			}},
			{"0:6", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "inner=2"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeFile, 0644, 0, 0, 0, inFile},
			}},
		})
		checkOverlayFS(ns, T, ovTestDevices{
			{"0:6", []ovTestInode{
				{nodeTypeDir, true, 5, 1, "inner=10"},
				{nodeTypeDir, true, 10, 1, ""},
				{nodeTypeFile, false, 9, 0, inFile},
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
			{0, 6, 1, 0, 4, nil, nil},
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

	mos := setup()
	testit(mos)

	mos = setup()
	var stat_buf Stat_t
	err := mos.SyscallLstat("/var/layer/upper/file1", &stat_buf)
	if err != ENOENT {
		T.Fatalf("unexpected ENOENT when statting deleted file; got %s", err)
	}
	err = mos.SyscallLstat("/var/layer/build/file1", &stat_buf)
	if err != ENOENT {
		T.Fatalf("expected ENOENT when statting deleted file; got %s", err)
	}
	testit(mos)

	mos = setup()
	dir, err := mos.Open("/var/layer/build/inner")
	if err != nil {
		T.Fatalf("error opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory: %s", err)
	}
	if len(got) > 0 {
		T.Fatalf("expected to read empty directory; got %s", strings.Join(got, ", "))
	}
	dir.Close()
	testit(mos)
}


func TestOverlayRemoveDirectoryFromLowerInterior(T *testing.T) {
	lowerFile1 := "first written to lower dir"
	lowerFile2 := "second written to lower dir"
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
			PopDir{Name: "/var/layer/lower/inner", Perms: 0755},
			PopFile{Name: "/var/layer/lower/inner/file1", Perms: 0644,
				Contents: lowerFile1},
			PopFile{Name: "/var/layer/lower/inner/file2", Perms: 0644,
				Contents: lowerFile2},
			PopMount{Source: "overlay", Mountpoint: "/var/layer/build",
				Fstype: "overlay",
				Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
					"workdir=/var/layer/work"},
		}
		_, err = populator.Populate(mos)
		if err != nil {
			T.Fatalf("populator failure: %s", err)
		}
		// Can't use RemoveAll since its deletion order is nondeterministic
		for _, name := range []string{"/var/layer/build/inner/file1",
			"/var/layer/build/inner/file2", "/var/layer/build/inner"} {
			err = mos.Remove(name)
			if err != nil {
				T.Fatalf("error when removing %s: %s", name, err)
			}
		}
		return mos
	}
	testit := func (mos *MemOS) {
		ns := mos.ns
		checkNSDevices(ns, T, nsTestDevices{
			{"0:2", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
				{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
				{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "inner=14"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "inner=8"},
				{nodeTypeDir, 0755, 0, 0, 1, "file1=9\nfile2=10"},
				{nodeTypeFile, 0644, 0, 0, 1, lowerFile1},
				{nodeTypeFile, 0644, 0, 0, 1, lowerFile2},
				{nodeTypeDir, 0755, 0, 0, 0, "file1=12\nfile2=13"},
				{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
				{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
				{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
			}},
			{"0:6", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 0, ""},
				{nodeTypeFile, 0644, 0, 0, 0, lowerFile1},
				{nodeTypeFile, 0644, 0, 0, 0, lowerFile2},
			}},
		})
		checkOverlayFS(ns, T, ovTestDevices{
			{"0:6", []ovTestInode{
				{nodeTypeDir, true, 5, 1, ""},
				{nodeTypeDir, true, 11, 0, ""},
				{nodeTypeFile, false, 9, 0, lowerFile1},
				{nodeTypeFile, false, 10, 0, lowerFile2},
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
			{0, 6, 1, 0, 4, nil, nil},
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

	mos := setup()
	testit(mos)

	mos = setup()
	var stat_buf Stat_t
	err := mos.SyscallLstat("/var/layer/upper/file1", &stat_buf)
	if err != ENOENT {
		T.Fatalf("unexpected ENOENT when statting deleted file; got %s", err)
	}
	err = mos.SyscallLstat("/var/layer/build/file1", &stat_buf)
	if err != ENOENT {
		T.Fatalf("expected ENOENT when statting deleted file; got %s", err)
	}
	testit(mos)
}


func TestOverlayRemoveDirectoryFromLowerInteriorThenRecreateIt(T *testing.T) {
	lowerFile1 := "first written to lower dir"
	lowerFile2 := "second written to lower dir"
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
			PopDir{Name: "/var/layer/lower/inner", Perms: 0755},
			PopFile{Name: "/var/layer/lower/inner/file1", Perms: 0644,
				Contents: lowerFile1},
			PopFile{Name: "/var/layer/lower/inner/file2", Perms: 0644,
				Contents: lowerFile2},
			PopMount{Source: "overlay", Mountpoint: "/var/layer/build",
				Fstype: "overlay",
				Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
					"workdir=/var/layer/work"},
		}
		_, err = populator.Populate(mos)
		if err != nil {
			T.Fatalf("populator failure: %s", err)
		}
		// Can't use RemoveAll since its deletion order is nondeterministic
		for _, name := range []string{"/var/layer/build/inner/file1",
			"/var/layer/build/inner/file2", "/var/layer/build/inner"} {
			err = mos.Remove(name)
			if err != nil {
				T.Fatalf("error when removing %s: %s", name, err)
			}
		}
		err = mos.Mkdir("/var/layer/build/inner", 0750)
		if err != nil {
			T.Fatalf("error recreating directory: %s", err)
		}
		return mos
	}
	testit := func (mos *MemOS) {
		ns := mos.ns
		checkNSDevices(ns, T, nsTestDevices{
			{"0:2", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
				{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
				{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "inner=15"},
				{nodeTypeDir, 0755, 0, 0, 1, ""},
				{nodeTypeDir, 0755, 0, 0, 1, "inner=8"},
				{nodeTypeDir, 0755, 0, 0, 1, "file1=9\nfile2=10"},
				{nodeTypeFile, 0644, 0, 0, 1, lowerFile1},
				{nodeTypeFile, 0644, 0, 0, 1, lowerFile2},
				{nodeTypeDir, 0755, 0, 0, 0, "file1=12\nfile2=13"},
				{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
				{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
				{nodeTypeCharDev, 0000, 0, 0, 0, "0:0"},
				{nodeTypeDir, 0750, 0, 0, 1, ""},
			}},
			{"0:6", []nsTestInode{
				{nodeTypeDir, 0755, 0, 0, 1, "inner=5"},
				{nodeTypeDir, 0755, 0, 0, 0, ""},
				{nodeTypeFile, 0644, 0, 0, 0, lowerFile1},
				{nodeTypeFile, 0644, 0, 0, 0, lowerFile2},
				{nodeTypeDir, 0750, 0, 0, 1, ""},
			}},
		})
		checkOverlayFS(ns, T, ovTestDevices{
			{"0:6", []ovTestInode{
				{nodeTypeDir, true, 5, 1, "inner=15"},
				{nodeTypeDir, true, 11, 0, ""},
				{nodeTypeFile, false, 9, 0, lowerFile1},
				{nodeTypeFile, false, 10, 0, lowerFile2},
				{nodeTypeDir, true, 15, 1, ""},
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
			{0, 6, 1, 0, 4, nil, nil},
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

	mos := setup()
	testit(mos)

	mos = setup()
	var stat_buf Stat_t
	err := mos.SyscallLstat("/var/layer/upper/file1", &stat_buf)
	if err != ENOENT {
		T.Fatalf("unexpected ENOENT when statting deleted file; got %s", err)
	}
	err = mos.SyscallLstat("/var/layer/build/file1", &stat_buf)
	if err != ENOENT {
		T.Fatalf("expected ENOENT when statting deleted file; got %s", err)
	}
	testit(mos)

	mos = setup()
	dir, err := mos.Open("/var/layer/build/inner")
	if err != nil {
		T.Fatalf("error opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading directory: %s", err)
	}
	if len(got) > 0 {
		T.Fatalf("expected to read empty directory; got %s", strings.Join(got, ", "))
	}
	dir.Close()
	testit(mos)
}


func TestOverlayMakeOpaqueDirectoryThenRemount(T *testing.T) {
	lowerFile1 := "first written to lower dir"
	lowerFile2 := "second written to lower dir"
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopDir{Name: "/var/layer/lower/inner", Perms: 0755},
		PopFile{Name: "/var/layer/lower/inner/file1", Perms: 0644,
			Contents: lowerFile1},
		PopFile{Name: "/var/layer/lower/inner/file2", Perms: 0644,
			Contents: lowerFile2},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build",
			Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	// Can't use RemoveAll since its deletion order is nondeterministic
	for _, name := range []string{"/var/layer/build/inner/file1",
		"/var/layer/build/inner/file2", "/var/layer/build/inner"} {
		err = mos.Remove(name)
		if err != nil {
			T.Fatalf("error when removing %s: %s", name, err)
		}
	}
	err = mos.Mkdir("/var/layer/build/inner", 0750)
	if err != nil {
		T.Fatalf("error recreating directory: %s", err)
	}
	err = mos.SyscallUnmount("/var/layer/build", 0)
	if err != nil {
		T.Fatalf("error unmounting overlay: %s", err)
	}
	err = mos.SyscallMount("overlay", "/var/layer/build", "overlay", 0,
		"upperdir=/var/layer/upper,lowerdir=/var/layer/lower,workdir=/var/layer/work")
	if err != nil {
		T.Fatalf("error remounting overlay: %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "inner=15"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "inner=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=9\nfile2=10"},
			{nodeTypeFile, 0644, 0, 0, 1, lowerFile1},
			{nodeTypeFile, 0644, 0, 0, 1, lowerFile2},
			{nodeTypeDir, 0755, 0, 0, 0, "file1=12\nfile2=13"},
			{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
			{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
			{nodeTypeCharDev, 0000, 0, 0, 0, "0:0"},
			{nodeTypeDir, 0750, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
		}},
	})
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "inner=15"},
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
		{0, 6, 1, 0, 4, nil, nil},
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


func TestOverlayMakeOpaqueDirectoryThenRemountAndList(T *testing.T) {
	lowerFile1 := "first written to lower dir"
	lowerFile2 := "second written to lower dir"
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopDir{Name: "/var/layer/lower/inner", Perms: 0755},
		PopFile{Name: "/var/layer/lower/inner/file1", Perms: 0644,
			Contents: lowerFile1},
		PopFile{Name: "/var/layer/lower/inner/file2", Perms: 0644,
			Contents: lowerFile2},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build",
			Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	// Can't use RemoveAll since its deletion order is nondeterministic
	for _, name := range []string{"/var/layer/build/inner/file1",
		"/var/layer/build/inner/file2", "/var/layer/build/inner"} {
		err = mos.Remove(name)
		if err != nil {
			T.Fatalf("error when removing %s: %s", name, err)
		}
	}
	err = mos.Mkdir("/var/layer/build/inner", 0750)
	if err != nil {
		T.Fatalf("error recreating directory: %s", err)
	}
	err = mos.SyscallUnmount("/var/layer/build", 0)
	if err != nil {
		T.Fatalf("error unmounting overlay: %s", err)
	}
	err = mos.SyscallMount("overlay", "/var/layer/build", "overlay", 0,
		"upperdir=/var/layer/upper,lowerdir=/var/layer/lower,workdir=/var/layer/work")
	if err != nil {
		T.Fatalf("error remounting overlay: %s", err)
	}
	dir, err := mos.Open("/var/layer/build/inner")
	if err != nil {
		T.Fatalf("error opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading names from directory: %s", err)
	}
	if len(got) > 0 {
		T.Fatalf("expected empty directory, got %s", strings.Join(got, ", "))
	}
	dir.Close()
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "inner=15"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "inner=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=9\nfile2=10"},
			{nodeTypeFile, 0644, 0, 0, 1, lowerFile1},
			{nodeTypeFile, 0644, 0, 0, 1, lowerFile2},
			{nodeTypeDir, 0755, 0, 0, 0, "file1=12\nfile2=13"},
			{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
			{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
			{nodeTypeCharDev, 0000, 0, 0, 0, "0:0"},
			{nodeTypeDir, 0750, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "inner=2"},
			{nodeTypeDir, 0750, 0, 0, 1, ""},
		}},
	})
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "inner=15"},
			{nodeTypeDir, true, 15, 1, ""},
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
		{0, 6, 1, 0, 4, nil, nil},
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


func TestOverlayMakeOpaqueDirectoryRemountThenRemoveAgain(T *testing.T) {
	lowerFile1 := "first written to lower dir"
	lowerFile2 := "second written to lower dir"
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopDir{Name: "/var/layer/lower/inner", Perms: 0755},
		PopFile{Name: "/var/layer/lower/inner/file1", Perms: 0644,
			Contents: lowerFile1},
		PopFile{Name: "/var/layer/lower/inner/file2", Perms: 0644,
			Contents: lowerFile2},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build",
			Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	// Can't use RemoveAll since its deletion order is nondeterministic
	for _, name := range []string{"/var/layer/build/inner/file1",
		"/var/layer/build/inner/file2", "/var/layer/build/inner"} {
		err = mos.Remove(name)
		if err != nil {
			T.Fatalf("error when removing %s: %s", name, err)
		}
	}
	err = mos.Mkdir("/var/layer/build/inner", 0750)
	if err != nil {
		T.Fatalf("error recreating directory: %s", err)
	}
	err = mos.SyscallUnmount("/var/layer/build", 0)
	if err != nil {
		T.Fatalf("error unmounting overlay: %s", err)
	}
	err = mos.SyscallMount("overlay", "/var/layer/build", "overlay", 0,
		"upperdir=/var/layer/upper,lowerdir=/var/layer/lower,workdir=/var/layer/work")
	if err != nil {
		T.Fatalf("error remounting overlay: %s", err)
	}
	dir, err := mos.Open("/var/layer/build/inner")
	if err != nil {
		T.Fatalf("error opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading names from directory: %s", err)
	}
	if len(got) > 0 {
		T.Fatalf("expected empty directory, got %s", strings.Join(got, ", "))
	}
	dir.Close()
	err = mos.Remove("/var/layer/build/inner")
	if err != nil {
		T.Fatalf("error on second removal of diretory %s", err)
	}
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "inner=16"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "inner=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=9\nfile2=10"},
			{nodeTypeFile, 0644, 0, 0, 1, lowerFile1},
			{nodeTypeFile, 0644, 0, 0, 1, lowerFile2},
			{nodeTypeDir, 0755, 0, 0, 0, "file1=12\nfile2=13"},
			{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
			{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
			{nodeTypeCharDev, 0000, 0, 0, 0, "0:0"},
			{nodeTypeDir, 0750, 0, 0, 0, ""},
			{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0750, 0, 0, 0, ""},
		}},
	})
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, ""},
			{nodeTypeDir, true, 15, 0, ""},
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
		{0, 6, 1, 0, 4, nil, nil},
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


func TestOverlayMakeOpaqueDirectoryRemountThenRemoveAgainAddBackRemount(T *testing.T) {
	lowerFile1 := "first written to lower dir"
	lowerFile2 := "second written to lower dir"
	mos, err := NewMemOS()
	if err != nil {
		T.Fatal(err.Error())
	}
	populator := PopulatorType{
		PopDir{Name: "/var/layer/build", Perms: 0755},
		PopDir{Name: "/var/layer/upper", Perms: 0755},
		PopDir{Name: "/var/layer/work", Perms: 0755},
		PopDir{Name: "/var/layer/lower", Perms: 0755},
		PopDir{Name: "/var/layer/lower/inner", Perms: 0755},
		PopFile{Name: "/var/layer/lower/inner/file1", Perms: 0644,
			Contents: lowerFile1},
		PopFile{Name: "/var/layer/lower/inner/file2", Perms: 0644,
			Contents: lowerFile2},
		PopMount{Source: "overlay", Mountpoint: "/var/layer/build",
			Fstype: "overlay",
			Options: "upperdir=/var/layer/upper,lowerdir=/var/layer/lower," +
				"workdir=/var/layer/work"},
	}
	_, err = populator.Populate(mos)
	if err != nil {
		T.Fatalf("populator failure: %s", err)
	}
	// Can't use RemoveAll since its deletion order is nondeterministic
	for _, name := range []string{"/var/layer/build/inner/file1",
		"/var/layer/build/inner/file2", "/var/layer/build/inner"} {
		err = mos.Remove(name)
		if err != nil {
			T.Fatalf("error when removing %s: %s", name, err)
		}
	}
	err = mos.Mkdir("/var/layer/build/inner", 0750)
	if err != nil {
		T.Fatalf("error recreating directory: %s", err)
	}
	err = mos.SyscallUnmount("/var/layer/build", 0)
	if err != nil {
		T.Fatalf("error unmounting overlay: %s", err)
	}
	err = mos.SyscallMount("overlay", "/var/layer/build", "overlay", 0,
		"upperdir=/var/layer/upper,lowerdir=/var/layer/lower,workdir=/var/layer/work")
	if err != nil {
		T.Fatalf("error remounting overlay: %s", err)
	}
	dir, err := mos.Open("/var/layer/build/inner")
	if err != nil {
		T.Fatalf("error opening directory: %s", err)
	}
	got, err := dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading names from directory: %s", err)
	}
	if len(got) > 0 {
		T.Fatalf("expected empty directory, got %s", strings.Join(got, ", "))
	}
	dir.Close()
	err = mos.Remove("/var/layer/build/inner")
	if err != nil {
		T.Fatalf("error on second removal of diretory %s", err)
	}
	err = mos.Mkdir("/var/layer/build/inner", 0751)
	if err != nil {
		T.Fatalf("error recreating directory: %s", err)
	}
	err = mos.SyscallUnmount("/var/layer/build", 0)
	if err != nil {
		T.Fatalf("error unmounting overlay: %s", err)
	}
	err = mos.SyscallMount("overlay", "/var/layer/build", "overlay", 0,
		"upperdir=/var/layer/upper,lowerdir=/var/layer/lower,workdir=/var/layer/work")
	if err != nil {
		T.Fatalf("error remounting overlay: %s", err)
	}
	dir, err = mos.Open("/var/layer/build/inner")
	if err != nil {
		T.Fatalf("error opening directory: %s", err)
	}
	got, err = dir.Readdirnames(-1)
	if err != nil {
		T.Fatalf("error reading names from directory: %s", err)
	}
	if len(got) > 0 {
		T.Fatalf("expected empty directory, got %s", strings.Join(got, ", "))
	}
	dir.Close()
	ns := mos.ns
	checkNSDevices(ns, T, nsTestDevices{
		{"0:2", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "var=2"},
			{nodeTypeDir, 0755, 0, 0, 1, "layer=3"},
			{nodeTypeDir, 0755, 0, 0, 1, "build=4\nlower=7\nupper=5\nwork=6"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "inner=17"},
			{nodeTypeDir, 0755, 0, 0, 1, ""},
			{nodeTypeDir, 0755, 0, 0, 1, "inner=8"},
			{nodeTypeDir, 0755, 0, 0, 1, "file1=9\nfile2=10"},
			{nodeTypeFile, 0644, 0, 0, 1, lowerFile1},
			{nodeTypeFile, 0644, 0, 0, 1, lowerFile2},
			{nodeTypeDir, 0755, 0, 0, 0, "file1=12\nfile2=13"},
			{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
			{nodeTypeCharDev, 0000, 0, 0, 1, "0:0"},
			{nodeTypeCharDev, 0000, 0, 0, 0, "0:0"},
			{nodeTypeDir, 0750, 0, 0, 0, ""},
			{nodeTypeCharDev, 0000, 0, 0, 0, "0:0"},
			{nodeTypeDir, 0751, 0, 0, 1, ""},
		}},
		{"0:6", []nsTestInode{
			{nodeTypeDir, 0755, 0, 0, 1, "inner=2"},
			{nodeTypeDir, 0751, 0, 0, 1, ""},
		}},
	})
	checkOverlayFS(ns, T, ovTestDevices{
		{"0:6", []ovTestInode{
			{nodeTypeDir, true, 5, 1, "inner=17"},
			{nodeTypeDir, true, 17, 1, ""},
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
		{0, 6, 1, 0, 4, nil, nil},
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

