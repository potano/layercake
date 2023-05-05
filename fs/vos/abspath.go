// Copyright © 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos


import (
	"path"
	"strings"
)


type abspathType []string


func newAbspath(pth string) abspathType {
	out := strings.Split(path.Clean(pth), "/")
	for len(out) > 0 && (len(out[0]) == 0 || out[0][0] == '.') {
		out = out[1:]
	}
	return abspathType(out)
}


func (ap abspathType) toString() string {
	return "/" + strings.Join(ap, "/")
}


func (ap abspathType) partial(n int) string {
	return "/" + strings.Join(ap[:n], "/")
}

