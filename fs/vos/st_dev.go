// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseMajorMinorString(str string) (int, int, error) {
	i := strings.IndexRune(str, ':')
	if i > 0 {
		v1, err1 := strconv.ParseUint(str[:i], 10, 64)
		v2, err2 := strconv.ParseUint(str[i+1:], 10, 64)
		if err1 == nil && err2 == nil {
			return int(v1), int(v2), nil
		}
	}
	return 0, 0, fmt.Errorf("bad major:minor format: %s", str)
}

func MajorMinorToString(major, minor int) string {
	return fmt.Sprintf("%d:%d", major, minor)
}

func MajorMinorToStDev(major, minor int) (uint64, error) {
	if major < 0 {
		return 0, fmt.Errorf("major device number %d is negative", major)
	}
	if minor < 0 || minor > 255 {
		return 0, fmt.Errorf("minor device number %d is not between 0 and 255", minor)
	}
	return uint64((major << 8) + minor), nil
}

func StDevToMajorMinor(st_dev uint64) (int, int) {
	return int(st_dev >> 8), int(st_dev & 0xFF)
}

func StDevToString(st_dev uint64) string {
	maj, min := StDevToMajorMinor(st_dev)
	return MajorMinorToString(maj, min)
}

func StDevForUnnamedDevice(st_dev uint64) bool {
	// From devices.txt: major numbers 0, 144, 145, and 146 are for unnamed devices
	return st_dev < 256 || (st_dev >= 144 * 256 && st_dev < 147 * 256)
}

