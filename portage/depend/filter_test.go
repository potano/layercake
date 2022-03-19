// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package depend

import (
	"testing"
	"potano.layercake/portage/atom"
	"potano.layercake/portage/parse"
)

func mkDepAtom(str string) (*DependAtom, error) {
	cursor := parse.NewAtomCursor([]byte(str))
	return newDependencyAtomAtCursor(cursor, false)
}


func TestVersionAndSlotComparer(t *testing.T) {
	//Note that these tests consider version and slot only; they have no checking for
	// USE flags or blocker status
	for _, grp := range []struct {depAtom, tstAtom string; shouldSucceed bool} {
		{"dev-lang/php:7.4", "dev-lang/php-7.4.2:7.4", true},
		{"dev-lang/php:7.4", "dev-lang/php-7.3.1:7.3", false},
		{"dev-lang/php:7.3", "dev-lang/php-7.4.2:7.4", false},
		{"dev-lang/php-7.4.2", "dev-lang/php-7.4.2:7.4", true},
		{">=sys-devel/patch-2.7", "sys-devel/patch-2.6", false},
		{">=sys-devel/patch-2.7", "sys-devel/patch-2.7", true},
		{">=sys-devel/patch-2.7", "sys-devel/patch-2.8", true},
		{"!<=gui-wm/sway-1.0_beta2[swaylock]", "gui-wm/sway-1.0", false},
		{"<=gui-wm/sway-1.0_beta2[swaylock]", "gui-wm/sway-1.0", false},
		{"<=gui-wm/sway-1.0_beta2[swaylock]", "gui-wm/sway-1.0_beta2", true},
		{"<=gui-wm/sway-1.0_beta2[swaylock]", "gui-wm/sway-1.0_beta3", false},
		{"!<=gui-wm/sway-1.0[swaylock]", "gui-wm/sway-1.0", true},
		{"<dev-python/click-8", "dev-python/click-7", true},
		{"<dev-python/click-8", "dev-python/click-8", false},
		{"=dev-lang/perl-5.30*", "dev-lang/perl-5.30.3", true},
		{"=dev-lang/perl-5.30*", "dev-lang/perl-5.30.3-r2", true},
		{"=dev-lang/perl-5.30*", "dev-lang/perl-5.31.4", false},
		{"~dev-lang/perl-5.30a", "dev-lang/perl-5.30a-r1", true},
		{"~dev-lang/perl-5.30a", "dev-lang/perl-5.30a_alpha4", true},
		{"~dev-lang/perl-5.30a", "dev-lang/perl-5.30", false},
		{"~dev-lang/perl-5.30a", "dev-lang/perl-5.30b", false},
		{">=sys-boot/gnu-efi-3.0u", "sys-boot/gnu-efi-3.0.12", true},
	} {
		depAtom, err := mkDepAtom(grp.depAtom)
		if err != nil {
			t.Errorf("%s creating dependency atom %s", err, grp.depAtom)
			continue
		}
		tstAtom, err := atom.NewUnprefixedConcreteAtom(grp.tstAtom)
		if err != nil {
			t.Errorf("%s creating dependency atom %s", err, grp.tstAtom)
			continue
		}
		got := depAtom.VersionAndSlotMatch(tstAtom)
		if got != grp.shouldSucceed {
			if grp.shouldSucceed {
				t.Errorf("%s should have matched %s", grp.tstAtom, grp.depAtom)
			} else {
				t.Errorf("%s should not have matched %s", grp.tstAtom, grp.depAtom)
			}
		}
	}
}


func TestSimpleFilterComparer(t *testing.T) {
	//Note that these tests do not check blocker status
	for _, grp := range []struct {depAtom, tstAtom, tstUseFlags string; shouldSucceed bool} {
		{"dev-lang/php:7.4", "dev-lang/php-7.4.2:7.4", "", true},
		{"dev-lang/php:7.4", "dev-lang/php-7.3.1:7.3", "", false},
		{"dev-lang/php:7.3", "dev-lang/php-7.4.2:7.4", "", false},
		{"dev-lang/php-7.4.2", "dev-lang/php-7.4.2:7.4", "", true},
		{">=sys-devel/patch-2.7", "sys-devel/patch-2.6", "", false},
		{">=sys-devel/patch-2.7", "sys-devel/patch-2.7", "", true},
		{">=sys-devel/patch-2.7", "sys-devel/patch-2.8", "", true},
		{"sys-apps/kbd[nls]", "sys-apps/kbd", "", false},
		{"sys-apps/kbd[nls]", "sys-apps/kbd", "nls", true},
		{"!<=gui-wm/sway-1.0_beta2[swaylock]", "gui-wm/sway-1.0", "", false},
		{"<=gui-wm/sway-1.0_beta2[swaylock]", "gui-wm/sway-1.0", "", false},
		{"<=gui-wm/sway-1.0_beta2[swaylock]", "gui-wm/sway-1.0_beta2", "", false},
		{"<=gui-wm/sway-1.0_beta2[swaylock]", "gui-wm/sway-1.0_beta3", "", false},
		{"!<=gui-wm/sway-1.0[swaylock]", "gui-wm/sway-1.0", "", false},
		{"!<=gui-wm/sway-1.0[swaylock]", "gui-wm/sway-1.0", "swaylock", true},
		{"<dev-python/click-8", "dev-python/click-7", "", true},
		{"<dev-python/click-8", "dev-python/click-8", "", false},
		{"!>=app-admin/sysklogd-2.0[logger]", "app-admin/sysklogd-2.1.2", "", false},
		{"!>=app-admin/sysklogd-2.0[logger]", "app-admin/sysklogd-2.1.2", "logger", true},
		{"dev-lang-python:3.7[threads]", "dev-lang/python-3.7.9:3.7", "", false},
		{"dev-lang-python:3.7[threads]", "dev-lang/python-3.7.9:3.7", "threads", true},
		{"dev-lang-python:3.7[-threads]", "dev-lang/python-3.7.9:3.7", "-threads", true},
		{"dev-lang-python:3.7[-threads]", "dev-lang/python-3.7.9:3.7", "threads", false},
		{"dev-lang-python:3.7[threads(+)]", "dev-lang/python-3.7.10:3.7", "", true},
		{"dev-lang-python:3.7[threads(+)]", "dev-lang/python-3.7.10:3.7", "threads", true},
		{"dev-lang-python:3.7[-threads(-)]", "dev-lang/python-3.7.10:3.7", "", true},
		{"dev-lang-python:3.7[-threads(-)]", "dev-lang/python-3.7.10:3.7", "-threads",
			true},
		{"dev-lang-python:3.7[-threads(-)]", "dev-lang/python-3.7.10:3.7", "threads",
			false},
	} {
		depAtom, err := mkDepAtom(grp.depAtom)
		if err != nil {
			t.Errorf("%s creating dependency atom %s", err, grp.depAtom)
			continue
		}
		tstAtom, err := atom.NewUnprefixedConcreteAtom(grp.tstAtom)
		if err != nil {
			t.Errorf("%s creating dependency atom %s", err, grp.tstAtom)
			continue
		}
		tstAtom.UseFlags = atom.NewUseFlagSetFromPrefixes(grp.tstUseFlags, true)
		tstAtoms := []atom.Atom{tstAtom}
		candidates := depAtom.FilterAtoms(tstAtoms, atom.EmptyUseFlagMap)
		got := len(candidates) > 0
		if got != grp.shouldSucceed {
			if grp.shouldSucceed {
				t.Errorf("%s should have matched %s", grp.tstAtom, grp.depAtom)
			} else {
				t.Errorf("%s should not have matched %s", grp.tstAtom, grp.depAtom)
			}
		}
	}
}


func TestContextualFilterComparer(t *testing.T) {
	//Note that these tests do not check blocker status
	//Note that these tests do not check blocker status
	for _, grp := range []struct {
		depAtom, tstAtom, tstUseFlags string
		contextFlags atom.UseFlagMap
		shouldSucceed bool
	} {
		{"sys-apps/kbd[nls]", "sys-apps/kbd", "-nls", atom.UseFlagMap{}, false},
		{"sys-apps/kbd[nls]", "sys-apps/kbd", "nls", atom.UseFlagMap{}, true},
		{"sys-apps/kbd[nls]", "sys-apps/kbd", "nls", atom.UseFlagMap{"nls": false}, true},

		{"!>=app-admin/sysklogd-2.0[logger]",
			"app-admin/sysklogd-2.1.1", "-logger",
			atom.UseFlagMap{}, false},
		{"!>=app-admin/sysklogd-2.0[logger]",
			"app-admin/sysklogd-2.1.2", "logger",
			atom.UseFlagMap{}, true},

		{"!>=app-admin/sysklogd-2.0[logger=]",
			"app-admin/sysklogd-2.2.1", "-logger",
			atom.UseFlagMap{}, true},
		{"!>=app-admin/sysklogd-2.0[logger=]",
			"app-admin/sysklogd-2.2.1", "logger",
			atom.UseFlagMap{}, false},
		{"!>=app-admin/sysklogd-2.0[logger=]",
			"app-admin/sysklogd-2.2.2", "logger",
			atom.UseFlagMap{"logger": true}, true},
		{"!>=app-admin/sysklogd-2.0[logger=]",
			"app-admin/sysklogd-2.2.2", "-logger",
			atom.UseFlagMap{"logger": true}, false},

		{"!>=app-admin/sysklogd-2.0[!logger=]",
			"app-admin/sysklogd-2.3.1", "logger",
			atom.UseFlagMap{"logger": true}, false},
		{"!>=app-admin/sysklogd-2.0[!logger=]",
			"app-admin/sysklogd-2.3.2", "-logger",
			atom.UseFlagMap{"logger": true}, true},
		{"!>=app-admin/sysklogd-2.0[!logger=]",
			"app-admin/sysklogd-2.3.3", "logger",
			atom.UseFlagMap{"logger": false}, true},
		{"!>=app-admin/sysklogd-2.0[!logger=]",
			"app-admin/sysklogd-2.3.4", "-logger",
			atom.UseFlagMap{"logger": false}, false},

		{"net-misc/ntp[ipv6?]",
			"net-misc/ntp-4.2.8", "-ipv6",
			atom.UseFlagMap{}, true},
		{"net-misc/ntp[ipv6?]",
			"net-misc/ntp-4.2.8", "ipv6",
			atom.UseFlagMap{}, true},
		{"net-misc/ntp[ipv6?]",
			"net-misc/ntp-4.2.8", "ipv6",
			atom.UseFlagMap{"ipv6": true}, true},
		{"net-misc/ntp[ipv6?]",
			"net-misc/ntp-4.2.8", "-ipv6",
			atom.UseFlagMap{"ipv6": true}, false},

		{"dev-libs/apr-util[!nss?]",
			"dev-libs/apr-util-1.5", "-nss",
			atom.UseFlagMap{}, true},
		{"dev-libs/apr-util[!nss?]",
			"dev-libs/apr-util-1.5", "nss",
			atom.UseFlagMap{}, true},
		{"dev-libs/apr-util[!nss?]",
			"dev-libs/apr-util-1.5", "nss",
			atom.UseFlagMap{"nss": true}, false},
		{"dev-libs/apr-util[!nss?]",
			"dev-libs/apr-util-1.5", "-nss",
			atom.UseFlagMap{"nss": true}, true},

		{"dev-vcs/git[-tk]",
			"dev-libs/apr-util-2.4.10", "-tk",
			atom.UseFlagMap{}, true},
		{"dev-vcs/git[-tk]",
			"dev-libs/apr-util-2.4.10", "tk",
			atom.UseFlagMap{}, false},
		{"dev-vcs/git[-tk]",
			"dev-libs/apr-util-2.4.10", "tk",
			atom.UseFlagMap{"tk": true}, false},
		{"dev-vcs/git[-tk]",
			"dev-libs/apr-util-2.4.10", "-tk",
			atom.UseFlagMap{"tk": true}, true},

		{"dev-lang-python:3.7[threads=(+)]",
			"dev-lang/python-3.7.10:3.7", "",
			atom.UseFlagMap{}, false},
		{"dev-lang-python:3.7[threads=(+)]",
			"dev-lang/python-3.7.10:3.7", "",
			atom.UseFlagMap{"threads": true}, true},
		{"dev-lang-python:3.7[!threads=(+)]",
			"dev-lang/python-3.7.10:3.7", "",
			atom.UseFlagMap{}, true},
		{"dev-lang-python:3.7[!threads=(+)]",
			"dev-lang/python-3.7.10:3.7", "",
			atom.UseFlagMap{"threads": true}, false},

		{"dev-lang-python:3.8[threads=(-)]",
			"dev-lang/python-3.8.10:3.8", "",
			atom.UseFlagMap{}, true},
		{"dev-lang-python:3.8[threads=(-)]",
			"dev-lang/python-3.8.10:3.8", "",
			atom.UseFlagMap{"threads": true}, false},
		{"dev-lang-python:3.8[!threads=(-)]",
			"dev-lang/python-3.8.10:3.8", "",
			atom.UseFlagMap{}, false},
		{"dev-lang-python:3.8[!threads=(-)]",
			"dev-lang/python-3.8.10:3.8", "",
			atom.UseFlagMap{"threads": true}, true},
	} {
		depAtom, err := mkDepAtom(grp.depAtom)
		if err != nil {
			t.Errorf("%s creating dependency atom %s", err, grp.depAtom)
			continue
		}
		tstAtom, err := atom.NewUnprefixedConcreteAtom(grp.tstAtom)
		if err != nil {
			t.Errorf("%s creating dependency atom %s", err, grp.tstAtom)
			continue
		}
		tstAtom.UseFlags = atom.NewUseFlagSetFromPrefixes(grp.tstUseFlags, true)
		tstAtoms := []atom.Atom{tstAtom}
		candidates := depAtom.FilterAtoms(tstAtoms, grp.contextFlags)
		got := len(candidates) > 0
		if got != grp.shouldSucceed {
			if grp.shouldSucceed {
				t.Errorf("%s should have matched %s", grp.tstAtom, grp.depAtom)
			} else {
				t.Errorf("%s should not have matched %s", grp.tstAtom, grp.depAtom)
			}
		}
	}
}

