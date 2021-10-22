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

