package atom

import "strings"


func makeComparable(version string) string {
	var out string
	pos := 0
	for pos < len(version) {
		if isDigit(version[pos]) {
			start := pos
			for pos < len(version) && isDigit(version[pos]) {
				pos++
			}
			out += padNumericSegment(version[start:pos])
		} else {
			start := pos
			for pos < len(version) && !isDigit(version[pos]) {
				pos++
			}
			out += version[start:pos]
		}
	}
	return out
}


func MakeNextVer(version string) string {
	versionSegmentLoop:
	for {
		version = strings.TrimRight(version, ".-")
		if len(version) == 0 {
			return maxAlphaVersion
		}
		pos := len(version) - 1
		if isDigit(version[pos]) {
			for pos > 0 && isDigit(version[pos - 1]) {
				pos--
			}
			val, overflow := incrementDecimal(version[pos:])
			if overflow {
				continue versionSegmentLoop
			}
			return version[:pos] + val
		}
		numStrippedZs := 0
		for {
			c := version[pos]
			version = version[:pos]
			if c == '-' || c == '_' || c == '.' {
				continue versionSegmentLoop
			}
			if c < 'z' {
				if c == 'Z' {
					c = 'a' - 1
				}
				return version + string(c + 1)
			}
			if pos == 0 || isDigit(version[pos - 1]) {
				break
			}
			pos--
			numStrippedZs++
		}
		return version + strings.Repeat("z", numStrippedZs + 1)
	}
}


func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}


func padNumericSegment(segment string) string {
	pad := numericVersionSegmentWidth - len(segment)
	if pad <= 0 {
		return segment
	}
	return strings.Repeat("0", pad) + segment
}


func incrementDecimal(val string) (string, bool) {
	if len(val) == 8 {	// Could be a date
		yr, mo, dy := val[0:4], val[4:6], val[6:]
		if yr > "0000" && yr < "2100" && mo > "00" && mo < "13" && dy > "00" && dy < "32" {
			// Do a very simpleminded date increment
			dy, _ = incrementDecimal(dy)
			if dy > "31" {
				dy = "01"
				mo, _ = incrementDecimal(mo)
				if mo > "12" {
					mo = "01"
					yr, _ = incrementDecimal(yr)
				}
			}
			return yr + mo + dy, false
		}
	}
	slice := []byte(val)
	for p := len(slice) - 1; p >= 0; p-- {
		c := slice[p] + 1
		if c <= '9' {
			slice[p] = c
			return string(slice), false
		}
		slice[p] = '0'
	}
	return val, true
}

