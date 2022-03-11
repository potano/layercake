package main


import (
	"os"
	"fmt"
	"flag"
	"strings"
	"potano.layercake/fs"
	"potano.layercake/fns"
	"potano.layercake/defaults"
)


const mainUsageMessage = `Builds Gentoo stage tarballs from working systems
Usage:
  {myself} -list [system|installed|stage] <options>
  {myself} -generate <output_path> <options>
  {myself} -help

Required switches: one of -list, -generate, or -help
  -list <set>        List atoms and/or files in given set:
                       system      the @system set for the profile
                       installed   all installed packages (@installed set)
                       stage       packages to include in stage tarball
  -generate          Generate stage tarball.  Writes to stdout when -o
                     option is not specified

Optional switches:
  -root <path>       Specify pathname of root of system from which to scrape
                     the stage file.  Defaults to the current direcory
  -profile <path>    Directory containing profile to use for stage file.
                     Defaults to etc/portage/make.profile relative to the
                     default or selected root directory
  -atoms <list>      Space-separated list of atoms to include in system set.
                     List may include blockers using the usual blocker
                     syntax (e.g. !sys-apps/make-kaos)
  -atomsfile <file>  File containing list of atoms to include in system set.
                     File has same format as /var/lib/portage/world
  -addfiles <file>   File containing list of additional files to insert
                     into tarball
  -recipe <file>     File whose contents may give settings for the -root,
                     -profile, -atoms, -atomsfile, -addfiles, -skinny,
                     -novdb, and -staticdev options
  -skinny            Exclude build dependencies
  -novdb             Exclude VDB (installed-package info at /var/db/pkg)
  -emptydev          Empty /dev:  omit static /dev entries
  -files             List files instead of packages in listing
  -filesbypackage    List of files grouped by package

  -o <file>          Specify output file.  When generating stage file, the
                     filename extension determines the compression method
                     unless the -compress option is specified.
  -compress <how>    Compress output tarball.  Parameter may be one of
                     'gzip', 'bzip2', 'xz', or 'none' (case-insensitive)
`

const argumentHintMessage = `
Usage:
  {myself} -list [system|installed|stage] <options>
  {myself} -generate <options>

  For more information:
  {myself} -h
  {myself} -version
`

func main() {
	var root, profile, listSet, outputPath, compressionInput, includeAtoms, atomsFile string
	var fileListFile, recipeFile string
	var generate, help, listFiles, listFilesByPackage, skinny, noVDB, emptyDev, showVer bool

	flag.StringVar(&listSet, "list", "", "output list")
	flag.BoolVar(&generate, "generate", false, "generate stage file")
	flag.StringVar(&root, "root", "", "root of filesystem")
	flag.StringVar(&profile, "profile", "", "profile directory")
	flag.StringVar(&includeAtoms, "atoms", "", "atoms to include in system set")
	flag.StringVar(&atomsFile, "atomsfile", "", "name of file with atoms to include")
	flag.StringVar(&fileListFile, "addfiles", "", "name of file listing additional files")
	flag.StringVar(&recipeFile, "recipe", "", "name of file with atoms to include")
	flag.StringVar(&outputPath, "o", "", "output file")
	flag.StringVar(&compressionInput, "compress", "", "compression method")
	flag.BoolVar(&skinny, "skinny", false, "exclude build dependencies")
	flag.BoolVar(&noVDB, "novdb", false, "exclude /var/db/pkg files (VDB)")
	flag.BoolVar(&emptyDev, "emptydev", false, "do not prepopulate /dev with device nodes")
	flag.BoolVar(&listFiles, "files", false, "list files")
	flag.BoolVar(&listFilesByPackage, "filesbypackage", false, "list files by package")
	flag.BoolVar(&listFilesByPackage, "bypackage", false, "list files by package")
	flag.BoolVar(&help, "help", false, "help")
	flag.BoolVar(&help, "h", false, "help")
	flag.BoolVar(&showVer, "version", false, "version")
	flag.Usage = func () {
		fns.TemplatedExitMessage(argumentHintMessage, 1, map[string]string{})
	}
	flag.Parse()
	if help {
		fns.TemplatedExitMessage(mainUsageMessage, 0, map[string]string{})
	}
	if showVer {
		fns.TemplatedExitMessage("Stagemaker, part of {version}", 0,
			map[string]string{"version": defaults.Version})
	}

	if (len(listSet) > 0) == generate {
		fatal("Must specify exactly one of -list or -generate")
	}

	var atomFiles, fileListFiles []string
	if len(recipeFile) > 0 {
		cursor, err := fs.NewTextInputFileCursor(recipeFile)
		if err != nil {
			fatal(err.Error())
		}
		var line string
		for cursor.ReadNonBlankNonCommentLine(&line) {
			needValue := false
			key, value := parseRecipeLine(line)
			switch key {
			case "root":
				needValue = true
				root = value
			case "profile":
				needValue = true
				profile = value
			case "atoms":
				needValue = true
				includeAtoms = value + " " + includeAtoms
			case "atomsfile":
				needValue = true
				atomFiles = append(atomFiles, value)
			case "addfiles":
				needValue = true
				fileListFiles = append(fileListFiles, value)
			case "compress":
				needValue = true
				if len(compressionInput) == 0 {
					compressionInput= value
				}
			case "skinny":
				skinny = true
			case "novdb":
				noVDB = true
			case "emptydev":
				emptyDev = true
			default:
				cursor.LogError("unrecogized keyword " + key)
			}
			if needValue && len(value) == 0 {
				cursor.LogError(key + " input requires a value")
			}
		}
	}

	data, err := setUpStageData(root, profile, outputPath)
	if err != nil {
		fatal(err.Error())
	}
	data.includeBdepend = !skinny
	data.includeVDB = !noVDB
	data.includeStaticDev = ! emptyDev

	if len(atomsFile) > 0 {
		atomFiles = append(atomFiles, atomsFile)
	}
	for _, name := range atomFiles {
		err := addAtomsFileToSystemSet(&data, name)
		if err != nil {
			fatal(err.Error())
		}
	}
	if len(includeAtoms) > 0 {
		err := addAtomListToSystemSet(&data, includeAtoms)
		if err != nil {
			fatal(err.Error())
		}
	}

	if len(fileListFile) > 0 {
		fileListFiles = append(fileListFiles, fileListFile)
	}
	data.fileListFiles = fileListFiles


	if len(listSet) > 0 {
		switch listSet {
		case "system":
			err = data.outputSystemSet()
		case "installed":
			err = data.outputInstalledSet(listFiles, listFilesByPackage)
		case "stage":
			err = data.outputStageSet(listFiles, listFilesByPackage)
		default:
			fatal("unknown -list argument %s", listSet)
		}
	} else if generate {
		var method int
		if len(compressionInput) > 0 {
			method, err = decodeCompressionInput(compressionInput)
		} else if len(outputPath) > 0 {
			method, err = decodeFilenameExtension(outputPath)
		} else {
			method = compression_none
		}
		if err == nil {
			data.tarCompression = method
			err = data.generateStage()
		}
	}

	if err != nil {
		fatal(err.Error())
	}
}



func parseRecipeLine(line string) (key string, value string) {
	parts := strings.Fields(line)
	key = parts[0]
	value = strings.TrimSpace(line[len(key):])
	return
}


func (data stageData) outputSystemSet() error {
	cursor, err := fs.NewTextOutputFileCursor(data.outputPath)
	if err != nil {
		return err
	}
	for _, item := range data.systemSet.Atoms {
		cursor.Println(item.String())
	}
	return nil
}


func (data stageData) outputInstalledSet(listFiles, listFilesByPackage bool) error {
	return data.showListing(data.installedSet.SortedAtoms(), listFiles, listFilesByPackage)
}


func (data stageData) outputStageSet(listFiles, listFilesByPackage bool) error {
	stageSet, err := data.generateStageSet()
	if err != nil {
		return err
	}
	return data.showListing(stageSet, listFiles, listFilesByPackage)
}


func (data stageData) generateStage() error {
	stageSet, err := data.generateStageSet()
	if err != nil {
		return err
	}
	return data.writeTarFile(stageSet)
}



func fatal(base string, params...interface{}) {
	fmt.Fprintf(os.Stderr, base, params...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

