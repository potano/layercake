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

func FindUses(prefix string, aggregate int) (map[string]int, error) {
	chop := 0
	if aggregate < 0 {
		aggregate = -aggregate
		chop = len(prefix)
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
		if !entry.IsDir() {
			continue
		}
		proc := entry.Name()
		ok := true
		for _, c := range proc {
			if c < '0' || c > '9' {
				ok = false
			}
		}
		if !ok {
			continue
		}
		proc = "/proc/" + proc + "/"
		buf := make([]byte, 256)
		for _, item := range []struct{path string; mask int}{
			{"cwd", UseMask_cwd}, {"root", UseMask_root}, {"exe", UseMask_exec}} {
			path := proc + item.path
			if !IsSymlink(path) {
				continue
			}
			n, err := syscall.Readlink(path, buf)
			if nil != err {
				continue
			}
			target := string(buf[:n])
			if strings.HasPrefix(target, prefix) {
				if chop > 0 {
					target = target[chop:]
				}
				out[target] |= item.mask
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
			if !IsSymlink(name) {
				continue
			}
			n, err := syscall.Readlink(name, buf)
			if nil != err {
				return nil, err
			}
			target := string(buf[:n])
			if strings.HasPrefix(target, prefix) {
				if chop > 0 {
					target = target[chop:]
				}
				out[target] |= UseMask_open
			}
		}
	}

	if aggregate > 0 {
		out2 := map[string]int{}
		for k, v := range out {
			parts := strings.Split(k, "/")
			target := strings.Join(parts[:aggregate], "/")
			out2[target] |= v
		}
		return out2, nil
	}
	return out, nil
}

