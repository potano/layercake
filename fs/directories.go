package fs

import (
	"os"
	"syscall"
)

func MakeDir(dirname string) (bool, error) {
	if IsDir(dirname) {
		return false, nil
	}
	err := os.Mkdir(dirname, 0755)
	return true, err
}

func PrepHtmlDir(dirname string) error {
	indexfile := dirname + "/index.html"
	if !IsFile(indexfile) {
		file, err := os.OpenFile(indexfile, os.O_RDWR|os.O_CREATE, 0644)
		if nil != err {
			return err
		}
		defer file.Close()
		_, err = file.Write([]byte(`<!DOCTYPE html>
<html>
   <head>
      <title>binpackager</title>
   </head>
   <body>
      <h1>binpackager</h1>
      <div>Serves prebuilt Gentoo packages</div>
   </body>
</html>
`))
		return err
	}
	return nil
}

func MakeEmptyLayersFile(pathname string) error {
	if !IsFile(pathname) {
		file, err := os.OpenFile(pathname, os.O_RDWR|os.O_CREATE, 0644)
		if nil != err {
			return err
		}
		file.Close()
	}
	return nil
}


func IsFile(filename string) bool {
	var stat syscall.Stat_t
	err := syscall.Stat(filename, &stat)
	return nil == err && (stat.Mode & syscall.S_IFMT) == syscall.S_IFREG
}

func IsDir(filename string) bool {
	var stat syscall.Stat_t
	err := syscall.Stat(filename, &stat)
	return nil == err && (stat.Mode & syscall.S_IFMT) == syscall.S_IFDIR
}

func IsSymlink(filename string) bool {
	var stat syscall.Stat_t
	err := syscall.Lstat(filename, &stat)
	return nil == err && (stat.Mode & syscall.S_IFMT) == syscall.S_IFLNK
}

func IsFileOrDir(filename string, wantFile bool) bool {
	mask := syscall.S_IFDIR
	if wantFile {
		mask = syscall.S_IFREG
	}
	var stat syscall.Stat_t
	err := syscall.Stat(filename, &stat)
	return nil == err && int(stat.Mode & syscall.S_IFMT) == mask
}

