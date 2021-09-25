package fs

import (
	"os"
	"syscall"
	"path/filepath"
)


func MakeDir(dirname string) (bool, error) {
	if IsDir(dirname) {
		return false, nil
	}
	err := os.Mkdir(dirname, 0755)
	return true, err
}


func WriteTextFile(filename, contents string) error {
	if !WriteOK("writing text file %s", filename) {
		return nil
	}
	if !IsFile(filename) {
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
		if nil != err {
			return err
		}
		defer file.Close()
		_, err = file.Write([]byte(contents))
		return err
	}
	return nil
}


func Readdirnames(directory string) ([]string, error) {
	fh, err := os.Open(directory)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	names, err := fh.Readdirnames(-1)
	if  err != nil {
		return nil, err
	}
	return names, nil
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


func Exists(filename string) bool {
	var stat syscall.Stat_t
	err := syscall.Stat(filename, &stat)
	return nil == err
}


func IsDescendant(dirpath, testpath string) (isDescendant bool, err error) {
	rel, err := filepath.Rel(dirpath, testpath)
	if err != nil {
		return
	}
	isDescendant = len(rel) > 0 && rel[0] != '.'
	return
}


func Readlink(linkname string) (string, error) {
	buf := make([]byte, 256)
	n, err := syscall.Readlink(linkname, buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

