package fs

import (
	"os"
	"strings"
	"syscall"
)

const (
	UseMask_cwd = 1 << iota
	UseMask_exec
	UseMask_open
	UseMask_root
)

func FindLayersInUse(prefix string) (map[string]int, error) {
	if len(prefix) > 1 && prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	fh, err := os.Open("/proc")
	if nil != err {
		return nil, err
	}
	defer fh.Close()
	lst, err := fh.Readdir(-1)
	if nil != err {
		return nil, err
	}
	if nil == lst {
		return nil, err
	}
	out := map[string]int{}
	for _, entry := range lst {
		proc := entry.Name()
		if !entry.IsDir() || !isNumeric(proc) {
			continue
		}
		proc = "/proc/" + proc + "/"
		buf := make([]byte, 256)
		for _, item := range []struct{path string; mask int}{
			{"cwd", UseMask_cwd}, {"root", UseMask_root}, {"exe", UseMask_exec}} {
			path := proc + item.path
			if lvlname, ok := isLinkToLevel(buf, prefix, path); ok {
				out[lvlname] |= item.mask
			}
		}
		if !IsDir(proc + "fd") {
			continue
		}
		fdh, err := os.Open(proc + "fd")
		if nil != err {
			continue
		}
		items, err := fdh.Readdir(-1)
		fdh.Close()
		if nil != err {
			return nil, err
		}
		if nil == items {
			continue
		}
		for _, entry := range items {
			name := proc + "fd/" + entry.Name()
			if lvlname, ok := isLinkToLevel(buf, prefix, name); ok {
				out[lvlname] |= UseMask_open
			}
		}
	}
	return out, nil
}


func isNumeric(str string) bool {
	for _, c := range str {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}


func isLinkToLevel(buf []byte, prefix, path string) (lvlname string, have bool) {
	if !IsSymlink(path) {
		return
	}
	n, err := syscall.Readlink(path, buf)
	if nil != err {
		return
	}
	target := string(buf[:n])
	if len(target) >= len(prefix) && target[:len(prefix)] == prefix {
		lvlname = target[len(prefix):]
		have = true
		p := strings.Index(lvlname, "/")
		if p > -1 {
			lvlname = lvlname[:p]
		}
	}
	return
}

