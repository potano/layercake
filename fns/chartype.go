// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package fns



type CharTypeMap map[byte]bool


var IsDigit CharTypeMap


func init() {
	IsDigit = MakeCharTypeMap("0-9")
}


func MakeCharTypeMap(setup string) CharTypeMap {
	m := CharTypeMap{}
	i := 0
	for i < len(setup) {
		c := setup[i]
		i++
		if i < len(setup) - 1 && setup[i] == '-' {
			end := setup[i+1]
			i += 2
			for ; c <= end; c++ {
				m[c] = true
			}
		} else {
			m[c] = true
		}
	}
	return m
}

