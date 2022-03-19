// Copyright © 2021 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package fns

import "strings"

func AndSlice(slice []string) string {
	if len(slice) == 0 {
		return ""
	}
	right := slice[len(slice)-1]
	slice = slice[:len(slice)-1]
	left := strings.Join(slice, ", ")
	if len(slice) > 1 {
		left += ","
	}
	if len(left) > 0 {
		return left + " and " + right
	}
	return right
}

