// Copyright © 2023 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package vos

import "fmt"

type Dtype int
var D Dtype

func (d Dtype) printf(patt string, args ...interface{}) {
	fmt.Printf(patt, args...)
}

func (d Dtype) print(msg string) {
	fmt.Println(msg)
}

