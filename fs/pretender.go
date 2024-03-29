// Copyright © 2021 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package fs



type PretenderFn func (string, ...interface{}) bool
type DebugMessagePrintf func (string, ...interface{})


var WriteOK PretenderFn


func init() {
	WriteOK = MakePretender(false, false, nil)
}


func MakePretender(pretend, debug bool, writer DebugMessagePrintf) PretenderFn {
	doIt := !pretend
	prefix := "action: "
	if pretend {
		prefix = "would "
	}
	if debug && writer != nil {
		return func (msg string, parms...interface{}) bool {
			writer(prefix + msg, parms...)
			return doIt
		}
	}
	return func (msg string, parms...interface{}) bool { return doIt }
}

