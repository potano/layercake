package fns

import "strings"

func Template(tpl string, vars map[string]string) string {
	segs := strings.Split(tpl, "{{")
	for segX, seg := range segs {
		i := strings.Index(seg, "}}")
		if i < 0 {
			if 0 == segX {
				continue
			}
			seg = "{{" + seg
		} else {
			seg = vars[seg[:i]] + seg[i+2:]
		}
		segs[segX] = seg
	}
	return strings.Join(segs, "")
}

