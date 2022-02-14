package fs

import (
	"os"
	"errors"
)


func ReadFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	fileinfo, err := file.Stat()
	if err != nil {
		return "", err
	}
	bufsiz := fileinfo.Size()
	buf := make([]byte, bufsiz)
	n, err := file.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}


func ReadFileIfExists(filename string) (string, bool, error) {
	str, err := ReadFile(filename)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", false, err
		}
		return "", false, nil
	}
	return str, true, nil
}

