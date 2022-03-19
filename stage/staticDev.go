// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package stage

import (
	"fmt"
	"strings"
	"strconv"
	"potano.layercake/defaults"
	"potano.layercake/fs"
)


func (fl *FileList) InsertStaticDev() error {
	cursor := fs.NewTextInputCursor("Static Dev", strings.NewReader(defaults.DevDirSetup))
	err := fl.ReadUserFileList(cursor)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(defaults.DevDirExtend, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		name := fields[0]
		last := name[len(name)-1]
		count, err := strconv.ParseUint(fields[1], 10, 8)
		if err != nil {
			return fmt.Errorf("%s parsing %s", err, line)
		}
		entry, found := fl.entryMap[name]
		if !found {
			return fmt.Errorf("can't find device node %s", name)
		}
		if last >= '0' && last <= '9' {
			name = name[:len(name)-1]
		}
		nameNum := 1
		entry.source = ""
		for count > 0 {
			entry.name = fmt.Sprintf("%s%d", name, nameNum)
			nameNum++
			entry.minor++
			err = fl.addFiles(entry)
			if err != nil {
				return err
			}
			count--
		}
	}
	return nil
}

