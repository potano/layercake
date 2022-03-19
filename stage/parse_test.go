// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package stage

import (
	"testing"

	"potano.layercake/fs"
	"potano.layercake/portage/vdb"
)




func mkli(tp, name string) *lineInfo {
	return &lineInfo{
		ltype: map[string]uint8{
			"dev": vdb.FileType_device,
			"dir": vdb.FileType_dir,
			"file": vdb.FileType_file,
			"link": vdb.FileType_symlink,
			"tbd": vdb.FileType_none}[tp],
		name: name}
}


func (li *lineInfo) src(s string) *lineInfo {
	li.source = s
	return li
}


func (li *lineInfo) wild() *lineInfo {
	li.hasWildcard = true
	return li
}


func (li *lineInfo) targ(t string) *lineInfo {
	li.target = t
	li.hasTarget = true
	return li
}


func (li *lineInfo) g_(id uint32) *lineInfo {
	li.gid = id
	li.hasGid = true
	return li
}


func (li *lineInfo) u_(id uint32) *lineInfo {
	li.uid = id
	li.hasUid = true
	return li
}


func (li *lineInfo) gu_(g, u uint32) *lineInfo {
	li.gid = g
	li.uid = u
	li.hasGid, li.hasUid = true, true
	return li
}


func (li *lineInfo) dev(tp byte, major, minor uint32) *lineInfo {
	li.devtype = tp
	li.major, li.minor = major, minor
	li.hasDev = true
	return li
}


func (li *lineInfo) perm(andMask, orMask uint32) *lineInfo {
	li.andMask = andMask
	li.orMask = orMask
	li.hasPerm = true
	return li
}


func (li *lineInfo) skipAbsent() *lineInfo {
	li.skipIfAbsent = true
	return li
}


func testEntry(t *testing.T, line string, expected *lineInfo, got lineInfo) {
	if got.ltype != expected.ltype {
		t.Errorf("expected type %d for [%s], got %d", expected.ltype, line, got.ltype)
	}
	if got.name != expected.name {
		t.Errorf("expected name '%s' for [%s], got '%s'", expected.name, line, got.name)
	}
	if got.source != expected.source {
		t.Errorf("expected source '%s' for [%s], got '%s'", expected.source, line,
			got.source)
	}
	if got.hasWildcard != expected.hasWildcard {
		t.Errorf("expected wildcard=%v for [%s], got %v", expected.hasWildcard, line,
			got.hasWildcard)
	}
	if (got.hasTarget || expected.hasTarget) && got.target != expected.target {
		t.Errorf("expected target '%s' for [%s], got '%s'", expected.target, line,
			got.target)
	}
	if (got.hasGid || expected.hasGid) && got.gid != expected.gid {
		t.Errorf("expected gid %d for [%s], got %d", expected.gid, line, got.gid)
	}
	if (got.hasUid || expected.hasUid) && got.uid != expected.uid {
		t.Errorf("expected uid %d for [%s], got %d", expected.uid, line, got.uid)
	}
	if (got.hasDev || expected.hasDev) && (got.devtype != expected.devtype ||
		got.major != expected.major || got.minor != expected.minor) {
		t.Errorf("expected device %c%d:%d for [%s], got %c%d:%d", expected.devtype,
			expected.major, expected.minor, line, got.devtype, got.major, got.minor)
	}
	if (got.hasPerm || expected.hasPerm) && (got.andMask != expected.andMask ||
		got.orMask != expected.orMask) {
		t.Errorf("expected permissions 0%o | (0%o & v) for [%s], got 0%o | (0%o & v)",
			expected.orMask, expected.andMask, line, got.orMask, got.andMask)
	}
	if got.skipIfAbsent != expected.skipIfAbsent {
		t.Errorf("expected %t skip-absent for [%s], got %t", expected.skipIfAbsent,
			line, got.skipIfAbsent)
	}
}


func TestParseLine(t *testing.T) {
	for _, tst := range []struct {line string; fi *lineInfo; errmsg string} {
		{"file /abc", mkli("file", "/abc"), ""},
		{"file '/abc def'", mkli("file", "/abc def"), ""},
		{"file \"/etc/set up'this'\"", mkli("file", "/etc/set up'this'"), ""},
		{"file /etc/set\\ up'this'", mkli("file", "/etc/set up'this'"), ""},
		{"dir /etc/portage", mkli("dir", "/etc/portage"), ""},
		{"dir /etc/portage/*", mkli("dir", "/etc/portage/*").wild(), ""},
		{"dir /etc/portage/\\*", mkli("dir", "/etc/portage/\\*"), ""},
		{"dir '/etc/portage/\\*'", mkli("dir", "/etc/portage/\\*"), ""},
		{"tbd /etc/rule", mkli("tbd", "/etc/rule"), ""},
		{"node /dev/tty1 dev=c4:1", mkli("dev", "/dev/tty1").dev('c', 4, 1), ""},
		{"file abc", &lineInfo{}, "name 'abc' is not absolute in test line 0"},
		{"unk /a/b", &lineInfo{}, "unknown file type unk in test line 0"},
		{"''", &lineInfo{}, "no file type in test line 0\nno file name in test line 0"},
		{"file ''", &lineInfo{}, "no file name in test line 0"},
		{"file '' ''", &lineInfo{}, "no file name in test line 0"},
		{"symlink /var/lib/a targ=/var/lib/b", mkli("link", "/var/lib/a").
			targ("/var/lib/b"), ""},
		{"file /etc/portage/make.conf \"src=~user/my make conf\"",
			mkli("file", "/etc/portage/make.conf").
			src("~user/my make conf"), ""},
		{"file /etc/a 'src=some where' uid=0", mkli("file", "/etc/a").src("some where").
			u_(0), ""},
		{"file /etc/a src='some where' uid=0", &lineInfo{},
			"could not parse option where' in test line 0"},
		{"file /a/b mod=644", mkli("file", "/a/b").perm(0, 0644), ""},
		{"file /a/b mod=+w", mkli("file", "/a/b").perm(07777, 0222), ""},
		{"file /a/b mod=u+w", mkli("file", "/a/b").perm(07777, 0200), ""},
		{"dir /a/b mod=u+x,o-r", mkli("dir", "/a/b").perm(07773, 0100), ""},
		{"file /a/b mod=+r mod=-w", mkli("file", "/a/b"),
			"multiple settings of file permissions in test line 0"},
		{"file /bin/blah mod=u+s", mkli("file", "/bin/blah").perm(07777, 04000), ""},
		{"file /bin/blah mod=+s", mkli("file", "/bin/blah").perm(07777, 06000), ""},
		{"file /bin/blah mod=g+s,u-s", mkli("file", "/bin/blah").perm(03777, 02000), ""},
		{"file /bin/blah mod=t", mkli("file", "/bin/blah").perm(07777, 01000), ""},
		{"file /etc/blah mod=-w", mkli("file", "/etc/blah").perm(07555, 00000), ""},
		{"file /a mod=08", mkli("file", "/a"), "bad mode setting 08 in test line 0"},
		{"file /etc/touch absent=skip", mkli("file", "/etc/touch").skipAbsent(), ""},
		{"file /etc/touch absent=jump", mkli("file", "/etc/touch"),
			"illegal value for absent= option in test line 0"},
	} {
		cursor := fs.NewTextInputCursor("test", nil)
		_, entry, ok := parseLine(tst.line, cursor)
		err := cursor.Err()
		if err == nil {
			if !ok {
				t.Errorf("has error indication but no error result")
			}
			if len(tst.errmsg) > 0 {
				t.Errorf("expected error '%s' for [%s]", tst.errmsg, tst.line)
			} else {
				testEntry(t, tst.line, tst.fi, entry)
			}
		} else if len(tst.errmsg) == 0 {
			t.Errorf("unexpected error '%s' for [%s]", err, tst.line)
		} else if err.Error() != tst.errmsg {
			t.Errorf("expected error '%s' for [%s], got '%s'", tst.errmsg, tst.line, err)
		}
	}
}

