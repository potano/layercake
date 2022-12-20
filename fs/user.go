// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package fs

import (
	"syscall"
)

func UserIsRoot() bool {
	return syscall.Geteuid() == 0
}

