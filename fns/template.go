package fns

func Template(tpl string, vars map[string]string) string {
	const (
		st_norm = iota
		st_eat
		st_key
	)
	var out []rune
	var start int
	state := st_norm
	for i, c := range tpl {
		switch state {
		case st_norm:
			if c == '{' {
				state = st_key
				start = i + 1
			} else if c == '\\' {
				state = st_eat
			} else {
				out = append(out, c)
			}
		case st_eat:
			out = append(out, c)
			state = st_norm
		case st_key:
			if c == '}' {
				out = append(out, []rune(vars[tpl[start:i]])...)
				state = st_norm
			}
		}
	}
	return string(out)
}

