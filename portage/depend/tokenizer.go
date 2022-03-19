// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package depend

import "potano.layercake/portage/parse"


// Types of token returned from getToken().
const (
	toktype_error = iota	// Illegal input before next whitespace.
	toktype_eof		// End of input.
	toktype_open		// Open paren
	toktype_close		// Close paren
	toktype_when_use_set	// When USE flag is set (flag?)
	toktype_when_use_unset	// When USE flag is not set (!flag?)
	toktype_any_of		// Intro to any_of group (||)
	toktype_exactly_one_of	// Intro to exactly-one-of group (^^)
	toktype_at_most_one_of	// Intro to at-most-one-of group (??)
	toktype_test_for_atom	// Unparsed package atom.
)

var byte_to_toktype_map map[byte]int = map[byte]int{
	'(': toktype_open,
	')': toktype_close,
	'|': toktype_any_of,
	'^': toktype_exactly_one_of,
	'?': toktype_at_most_one_of,
}

func getToken(ac *parse.AtomCursor) (start, toktype int, useFlag string) {
	if ac.Pos <= ac.Last {
		max := ac.Pos + 10
		if max > ac.Last {
			max = ac.Last
		}
	}
	start, toktype, useFlag = _getToken(ac)
	return start, toktype, useFlag
}

func _getToken(ac *parse.AtomCursor) (start, toktype int, useFlag string) {
	c := ac.Peek()
	for c <= ' ' {
		if ac.Pos > ac.Last {
			toktype = toktype_eof
			return
		}
		c = ac.Take()
	}
	start = ac.Pos
	end := ac.NextWhitespace()
	toklen := end - start
	var take int
	c0 := c
	if c0 == '(' || c0 == ')' {
		take = 1
	} else if c == '|' || c == '^' || c == '?' {
		take = 2
	}
	if take > 0 {
		if toklen != take || (take > 1 && ac.Peek1() != c0) {
			toktype = toktype_error
		} else {
			ac.Pos = end
			toktype = byte_to_toktype_map[c]
		}
		return
	}
	useStart := start
	if c == '!' {
		c = ac.Take()
		useStart++
		toklen--
	}
	if toklen > 1 && ac.Slice[end - 1] == '?' {
		for p := ac.Pos; p < end - 1; p++ {
			if !parse.IsUseFlagChar[ac.Slice[p]] {
				toktype = toktype_error
				return
			}
		}
		if c0 == '!' {
			toktype = toktype_when_use_unset
		} else {
			toktype = toktype_when_use_set
		}
		ac.Pos = end
		useFlag = string(ac.Slice[useStart:end-1])
	} else {
		ac.Pos = start
		toktype = toktype_test_for_atom
	}
	return
}

