// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package fs

import (
	"os"
	"fmt"
	"path"
	"os/user"
	"testing"
)


const (
	tstBasePath = "/var/lib/where/base"
	tstSelfPath = "/var/lib/where/self"
)


func adjCallback(symbol, tail string) (string, error) {
	switch symbol {
	case "base":
		return tstBasePath, nil
	case "self":
		return tstSelfPath, nil
	}
	return "", fmt.Errorf("unknown symbol")
}


func TestPathPrefixAdjustments(t *testing.T) {
	curUser, err := user.Current()
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, tst := range []struct{before, after, errmsg string} {
		{"/a/b/c", "/a/b/c", ""},
		{"~/got/it", path.Join(curUser.HomeDir, "/got/it"), ""},
		{"~" + curUser.Username + "/a", path.Join(curUser.HomeDir, "/a"), ""},
		{"~youcantfindme/j", "",
			"user: unknown user youcantfindme resolving path ~youcantfindme/j"},
		{"$$base/one/file", path.Join(tstBasePath, "/one/file"), ""},
		{"$$unk/nown", "", "unknown symbol in resolving $$unk prefix of $$unk/nown"},
		{"abc", "", "relative path abc is not allowed"},
	} {
		got, err := AdjustPrefixedPath(tst.before, "", adjCallback)
		if err != nil {
			if err.Error() != tst.errmsg {
				t.Errorf("unexpected error message %s for path %s", err, tst.before)
			}
		} else if len(tst.errmsg) > 0 {
			t.Errorf("unexpected success for path %s", tst.before)
		} else if got != tst.after {
			t.Errorf("expected interpretation '%s' for path '%s', got '%s'",
				tst.after, tst.before, got)
		}
	}
}


func TestRelativePathAdjustments(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, tst := range []struct{before, relto, after, errmsg string} {
		{"/d/e/f", "", "/d/e/f", ""},
		{"g/h", "/i", "/i/g/h", ""},
		{"j/k", ".", path.Join(cwd, "/j/k"), ""},
		{"l/m", "..", path.Join(path.Dir(cwd), "/l/m"), ""},
		{"n/o", "../p", path.Join(path.Dir(cwd), "/p/n/o"), ""},
	} {
		got, err := AdjustPrefixedPath(tst.before, tst.relto, adjCallback)
		if err != nil {
			if err.Error() != tst.errmsg {
				t.Errorf("unexpected error message %s for path %s", err, tst.before)
			}
		} else if len(tst.errmsg) > 0 {
			t.Errorf("unexpected success for path %s", tst.before)
		} else if got != tst.after {
			t.Errorf("expected interpretation '%s' for path '%s', got '%s'",
				tst.after, tst.before, got)
		}
	}
}

