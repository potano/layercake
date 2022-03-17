package main

import (
	"io"
	"os"
	"fmt"
	"path"
	"os/exec"
	"strings"
	"path/filepath"
	"potano.layercake/fs"
	"potano.layercake/defaults"
	"potano.layercake/portage/atom"
	"potano.layercake/portage/depend"
	"potano.layercake/portage/profile"
	"potano.layercake/portage/vdb"
	"potano.layercake/stage"
)


const (
	compression_none = iota
	compression_gzip
	compression_bzip2
	compression_xz
)


type stageData struct {
	rootDir, profileDir, outputPath string
	systemSet *depend.UserEnteredDependencies
	installedSet *atom.AtomSet
	tarFileName string
	fileListFiles []string
	tarCompression int
	includeBdepend bool
	includeVDB bool
	includeStaticDev bool
}


func setUpStageData(rootDir, profileDir, outputPath string) (data stageData, err error) {
	if len(rootDir) == 0 {
		rootDir, err = os.Getwd()
		if err != nil {
			return
		}
	}
	if !fs.IsDir(rootDir) {
		err = fmt.Errorf("root directory %s does not exist", rootDir)
		return
	}
	rootDir, err = filepath.Abs(rootDir)
	if err != nil {
		err = fmt.Errorf("%s resolving root directory %s", err, rootDir)
		return
	}
	data.rootDir = rootDir

	if len(profileDir) == 0 {
		profileDir = path.Join(rootDir, "/etc/portage/make.profile")
	}
	if !fs.IsDir(profileDir) {
		err = fmt.Errorf("profile directory %s does not exist", profileDir)
		return
	}
	data.profileDir = profileDir
	data.outputPath = outputPath

	systemSet, err := profile.ReadSystemSet(profileDir)
	if err != nil {
		return
	}
	data.systemSet = systemSet

	installedSet, err := vdb.GetInstalledPackageList(rootDir)
	if err != nil {
		return
	}
	data.installedSet = installedSet

	return
}


func addAtomListToSystemSet(data *stageData, str string) error {
	for _, atm := range strings.Fields(str) {
		err := data.systemSet.Add(atm)
		if err != nil {
			return err
		}
	}
	return nil
}


func addAtomsFileToSystemSet(data *stageData, filename string) error {
	cursor, err := fs.NewTextInputFileCursor(filename)
	if err != nil {
		return err
	}
	var line string
	for cursor.ReadLine(&line) {
		line = strings.TrimSpace(line)
		if len(line) < 1 || line[0] == '#' || (len(line) > 1 && line[:2] == "//") {
			continue
		}
		err := data.systemSet.Add(line)
		if err != nil {
			cursor.LogError(err.Error())
		}
	}
	return cursor.Err()
}


func decodeCompressionInput(input string) (int, error) {
	val := input
	if len(val) > 1 {
		val = strings.ToLower(val)
	}
	switch val {
	case "gzip", "gz", "z":
		return compression_gzip, nil
	case "bzip2", "bzip", "bz2", "bz", "j":
		return compression_bzip2, nil
	case "xz", "J":
		return compression_xz, nil
	case "none", "no", "0":
		return compression_none, nil
	}
	return 0, fmt.Errorf("unknown compression parameter %s", input)
}


func decodeFilenameExtension(pth string) (int, error) {
	for _, tst := range []struct {method int; exts string} {
		{compression_gzip, defaults.GzipExtensions},
		{compression_bzip2, defaults.BzipExtensions},
		{compression_xz, defaults.XzExtensions},
		{compression_none, defaults.NoCompressExtension},
	} {
		for _, ext := range strings.Fields(tst.exts) {
			if strings.HasSuffix(pth, ext) {
				return tst.method, nil
			}
		}
	}
	return 0, fmt.Errorf("cannot determine compression method from filename %s", pth)
}


func (data stageData) generateStageSet() (atom.AtomSlice, error) {
	solution, err := vdb.StartSolution(data.installedSet, data.includeBdepend)
	if err != nil {
		return nil, err
	}
	err = solution.ResolveUserDeps(data.systemSet)
	if err != nil {
		return nil, err
	}
	return solution.Resolution.SortedAtoms(), nil
}


func (data stageData) getStageFileList(atoms atom.AtomSlice) (*stage.FileList, error) {
	packageFiles, err := vdb.GetInstalledFileInfo(atoms)
	if err != nil {
		return nil, err
	}
	fileList, err := stage.GenerateFileList(packageFiles, data.rootDir)
	if err != nil {
		return nil, err
	}
	err = fileList.RecoverMissingLinks()
	if err != nil {
		return nil, err
	}
	if data.includeVDB {
		err = fileList.AddDirectoriesByName(vdb.GetDirectories(atoms))
		if err != nil {
			return nil, err
		}
	}
	if data.includeStaticDev {
		err = fileList.InsertStaticDev()
		if err != nil {
			return nil, err
		}
	}
	cursor := fs.NewTextInputCursor("StageMagic", strings.NewReader(defaults.StageMagic))
	err = fileList.ReadUserFileList(cursor)
	if err != nil {
		return nil, err
	}
	err = fileList.AddMissingStageDirs()
	if err != nil {
		return nil, err
	}
	for _, filename := range data.fileListFiles {
		cursor, err := fs.NewTextInputFileCursor(filename)
		if err != nil {
			return nil, err
		}
		err = fileList.ReadUserFileList(cursor)
		if err != nil {
			return nil, err
		}
	}

	fileList.Finalize()
	return fileList, err
}


func (data stageData) showListing(atomSlice atom.AtomSlice, listFiles, byPackage bool) error {
	cursor, err := fs.NewTextOutputFileCursor(data.outputPath)
	if err != nil {
		return err
	}
	if byPackage {
		for _, atm := range atomSlice {
			cursor.Println(atm.String())
			files, err := vdb.GetAtomFileInfo(atm)
			if err != nil {
				return err
			}
			for _, fe := range files {
				cursor.Printf("   %s\n", fe.Name)
			}
		}
	} else if listFiles {
		fileList, err := data.getStageFileList(atomSlice)
		if err != nil {
			return err
		}
		for _, name := range fileList.Names() {
			cursor.Println(name)
		}
	} else {
		for _, item := range atomSlice {
			cursor.Println(item.String())
		}
	}
	return nil
}


func (data stageData) writeTarFile(atomSlice atom.AtomSlice) error {
	fileList, err := data.getStageFileList(atomSlice)
	if err != nil {
		return err
	}
	fileWriter := os.Stdout
	if len(data.outputPath) > 0 {
		fileWriter, err = os.Create(data.outputPath)
		if err != nil {
			return err
		}
		defer fileWriter.Close()
	}
	tarWriter, deferred := data.makeTarWriter(fileWriter)
	err = fileList.MakeTar(tarWriter)
	if err != nil {
		return nil
	}
	tarWriter.Close()
	return <-deferred
}


func (data stageData) makeTarWriter(fileWriter io.WriteCloser) (io.WriteCloser, chan error) {
	deferred := make(chan error)
	var command string
	switch data.tarCompression {
	case compression_gzip:
		command = defaults.GzipExecutable
	case compression_bzip2:
		command = defaults.BzipExecutable
	case compression_xz:
		command = defaults.XzExecutable
	default:
		deferred <- nil
		return fileWriter, deferred
	}

	rpipe, wpipe := io.Pipe()
	cmd := exec.Command(command)
	cmd.Stdin = rpipe
	cmd.Stdout = fileWriter
	go func() {
		err := cmd.Run()
		rpipe.CloseWithError(err)
		deferred <- err
	}()
	return wpipe, deferred
}

