// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package stage

import (
	"io"
	"fmt"
	"time"
	"archive/tar"
	"potano.layercake/fs"
	"potano.layercake/portage/vdb"
)


func (fl *FileList) MakeTar(writer io.Writer) error {
	wrt := tar.NewWriter(writer)
	defer wrt.Close()
	for _, info := range fl.Files {
		hdr := &tar.Header{
			Name: "." + info.name,
			Size: info.fsize,
			Mode: int64(info.orMask),
			Uid: int(info.uid),
			Gid: int(info.gid),
			ModTime: time.Unix(info.unixTime, 0),
			Format: tar.FormatPAX,
		}
		if info.xattrs != nil {
			hdr.Xattrs = info.xattrs
		}
		switch info.ltype {
		case vdb.FileType_dir:
			hdr.Typeflag = tar.TypeDir
		case vdb.FileType_file:
			hdr.Typeflag = tar.TypeReg
		case vdb.FileType_symlink:
			hdr.Typeflag = tar.TypeSymlink
			hdr.Linkname = info.target
		case vdb.FileType_hardlink:
			hdr.Typeflag = tar.TypeLink
			target := info.target
			if target[0] == '/' {
				target = "." + target
			}
			hdr.Linkname = target
		case vdb.FileType_device:
			if info.devtype == 'c' {
				hdr.Typeflag = tar.TypeChar
			} else {
				hdr.Typeflag = tar.TypeBlock
			}
			hdr.Devmajor = int64(info.major)
			hdr.Devminor = int64(info.minor)
		}
		if err := wrt.WriteHeader(hdr); err != nil {
			return err
		}
		if info.ltype == vdb.FileType_file && info.fsize > 0 {
			contents, err := fs.ReadFile(info.source)
			if err != nil {
				return err
			}
			if int64(len(contents)) != info.fsize {
				return fmt.Errorf("expected %s to have length %d, got %d",
					info.source, info.fsize, len(contents))
			}
			if _, err := wrt.Write([]byte(contents)); err != nil {
				return err
			}
		}
	}
	return nil
}


