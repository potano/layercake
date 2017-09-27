package fs

import (
	"os"
	"bufio"
	"strings"
)

type MountType struct {
	Source, Mountpoint string
	Source2, Workdir string
	Fstype string
	Options string
}

type Mounts []MountType


//ProbeMounts discovers the mounts in the current system using information at /proc/mounts.
//Format of these lines is described in the man page for getmntent(3)
func ProbeMounts() (Mounts, error) {
	var mounts Mounts
	fh, err := os.Open("/proc/mounts")
	if nil != err {
		return nil, err
	}
	defer fh.Close()
	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		line := scanner.Text()
		segments := strings.Split(line, " ")
		for segX, segment := range segments {
			p := strings.IndexByte(segment, '\\')
			if p >= 0 {
				segments[segX] = unescapeMntent(segment)
			}
		}
		if len(segments) < 4 {
			continue
		}
		sourcept := segments[0]
		sourcept2 := ""
		mountpt := segments[1]
		workpt := ""
		fstype := segments[2]
		opts := segments[3]

		if "overlay" == fstype {
			var newopts []string
			for _, part := range strings.Split(opts, ",") {
				consumed := false
				kv := strings.SplitN(part, "=", 2)
				if len(kv) > 1 {
					consumed = true
					switch kv[0] {
					case "lowerdir":
						sourcept = kv[1]
					case "upperdir":
						sourcept2 = kv[1]
					case "workdir":
						workpt = kv[1]
					default:
						consumed = false
					}
				}
				if !consumed {
					newopts = append(newopts, part)
				}
			}
			opts = strings.Join(newopts, ",")
		}
		mounts = append(mounts,
			MountType{sourcept, mountpt, sourcept2, workpt, fstype, opts})
	}
	return mounts, scanner.Err()
}

func (m Mounts) GetAll() Mounts {
	return m
}

func (m Mounts) GetMount(path string) *MountType {
	for mtX, mt := range m {
		if mt.Mountpoint == path {
			return &((m)[mtX])
		}
	}
	return nil
}

func (m Mounts) GetSubmounts(path string) Mounts {
	list := make([]MountType, 0, len(m))
	path += "/"
	for _, mt := range m {
		if strings.HasPrefix(mt.Mountpoint, path) {
			list = append(list, mt)
		}
	}
	return list
}

func (m Mounts) GetMountAndSubmounts(path string) Mounts {
	list := make([]MountType, 0, len(m))
	topmount := m.GetMount(path)
	if nil != topmount {
		list = append(list, *topmount)
	}
	path += "/"
	for _, mt := range m {
		if strings.HasPrefix(mt.Mountpoint, path) {
			list = append(list, mt)
		}
	}
	return list
}

func unescapeMntent(in string) string {
	out := []byte(in)
	outp := 0
	octal := -1
	for _, c := range out {
		if octal > -1 {
			if c >= '0' && c < '8' && octal < 32 {
				octal = octal * 8 + int(c - '0')
				continue
			}
			out[outp] = byte(octal)
			outp++
			octal = -1
		}
		if '\\' == c {
			octal = 0
			continue
		}
		out[outp] = c
		outp++
	}
	return string(out[:outp])
}

