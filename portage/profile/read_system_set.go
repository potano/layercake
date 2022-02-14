package profile

import (
	"fmt"
	"path"
	"potano.layercake/fs"
	"potano.layercake/portage/depend"
)


func ReadSystemSet(profilePath string) (*depend.UserEnteredDependencies, error) {
	deps := depend.NewUserEnteredDependencies()
	err := readProfileDirectory(deps, profilePath)
	if err != nil {
		return nil, err
	}
	return deps, nil
}


func readProfileDirectory(deps *depend.UserEnteredDependencies, dirPath string) error {
	if !fs.IsDir(dirPath) {
		return fmt.Errorf("profile directory %s does not exist", dirPath)
	}
	packagesPath := path.Join(dirPath, "packages")
	if fs.IsFile(packagesPath) {
		err := readProfileFile(deps, packagesPath)
		if err != nil {
			return err
		}
	}

	parentFile := path.Join(dirPath, "parent")
	if fs.IsFile(parentFile) {
		err := readParentFile(deps, parentFile, dirPath)
		if err != nil {
			return err
		}
	}
	return nil
}


func readProfileFile(deps *depend.UserEnteredDependencies, filename string) error {
	cursor, err := fs.NewTextInputFileCursor(filename)
	if err != nil {
		return err
	}
	defer cursor.Close()
	var line string
	for cursor.ReadLine(&line) {
		if len(line) > 2 && line[0] == '*' {
			err = deps.Add(line[1:])
			if err != nil {
				return err
			}
		}
	}
	return cursor.Err()
}


func readParentFile(deps *depend.UserEnteredDependencies, filename, profilePath string) error {
	cursor, err := fs.NewTextInputFileCursor(filename)
	if err != nil {
		return err
	}
	defer cursor.Close()
	var line string
	for cursor.ReadLine(&line) {
		if len(line) > 0 {
			if fs.IsSymlink(profilePath) {
				pth, err := fs.Readlink(profilePath)
				if err != nil {
					return err
				}
				if pth[0] == '/' {
					profilePath = pth
				} else {
					profilePath = path.Join(path.Dir(profilePath), pth)
				}
			}
			profile := path.Join(profilePath, line)
			err := readProfileDirectory(deps, profile)
			if err != nil {
				return err
			}
		}
	}
	return cursor.Err()
}

