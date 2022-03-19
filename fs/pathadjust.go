// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package fs

import (
	"os"
	"fmt"
	"path"
	"os/user"
)


type PathPrefixCallback func (symbol, tail string) (string, error)


func AdjustPrefixedPath(pathname, relativeTo string, callback PathPrefixCallback) (string, error) {
	if len(pathname) < 1 {
		return "", nil
	}
	newpath := pathname
	sigil, name, tail := decomposePrefix(pathname)
	if sigil == "~" {
		var usr *user.User
		var err error
		if len(name) == 0 {
			usr, err = user.Current()
		} else {
			usr, err = user.Lookup(name)
		}
		if err != nil {
			return "", fmt.Errorf("%s resolving path %s", err, pathname)
		}
		if len(usr.HomeDir) == 0 {
			return "", fmt.Errorf("no home directory found for %s", pathname)
		}
		newpath = path.Join(usr.HomeDir, tail)
	} else if sigil == "$$" {
		pre, err := callback(name, tail)
		if err != nil {
			return "", fmt.Errorf("%s in resolving $$%s prefix of %s", err, name,
				pathname)
		}
		newpath = path.Join(pre, tail)
	} else if len(sigil) > 0 {
		return "", fmt.Errorf("illegal prefix characters %s in %s", sigil, pathname)
	}

	if len(newpath) > 0 && newpath[0] != '/' {
		if len(relativeTo) == 0 {
			return "", fmt.Errorf("relative path %s is not allowed", newpath)
		}
		if relativeTo[0] == '.' {
			cwd, err := os.Getwd()
			if err != nil {
				return "", err
			}
			relativeTo = path.Join(cwd, relativeTo)
		}
		return path.Join(relativeTo, newpath), nil
	}
	return newpath, nil
}


func decomposePrefix(pathname string) (sigil, name, tail string) {
	p := 0
	for p < len(pathname) {
		c := pathname[p]
		if c != '~' && c != '$' {
			break
		}
		p++
	}
	if p > 0 {
		sigil = pathname[:p]
		pathname = pathname[p:]
		p = 0
	}
	for p < len(pathname) {
		if pathname[p] == '/' {
			break
		}
		p++
	}
	if p > 0 {
		name = pathname[:p]
		pathname = pathname[p:]
		p = 0
	}
	tail = pathname
	return
}

