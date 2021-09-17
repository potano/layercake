// Layercake: manager of layers of build chroots
// Mike Thompson 5/5/2017

package main

import (
	"os"
	"fmt"
	"flag"
	"path"
	"strings"
	"path/filepath"

	"potano.layercake/fs"
	"potano.layercake/fns"
	"potano.layercake/config"
	"potano.layercake/manage"
)

var (
	layers *manage.Layerdefs
)

const usageMessage = `Manages layers in the build chroot
Usage:
  {{myself}} [main-options] <command> [command-options]
These commands are available
  init             Establish the layer system in current directory
  status           Display the status of the build root
  list [-v]        Display list of layers showing status
                   Add -v for a more verbose listing
  add <layer> [base]  Add a layer and indicate layer it derives
  rename <layer> <newname>  Rename a layer
  rebase <layer> [newbase]  Change a layer's base layer
  remove <layer> [-files]   Remove a layer; use -files to remove
                            files and directories also
  shell [layer]   Starts a shell in the named layer or the first
                  base-level layer.  Useful for setting up base
                  layer before it can be made mountable.
  mkdirs [layer]  Create or recreate needed directories in named
                  layer or in all layers
  mount [layer]   Mount per-layer directories in named layer or
                  in all layers
  umount [layer]  Mount per-layer directories in named layer or
                  in all layers
  shake           Remount all current overlayfs mounts to ensure
                  that changes in lower layers propagate upward
                  to mounted layers
  chroot [layer]  Starts a chroot using named layer or first base-
                  level layer if none specified
Main options
  --config <file> Specify/override configuration-file location
  --basepath <path>  Specify/override build-root basepath
Global options
  -v              Verbose mode: show actions (to be) taken
  -p              Pretend to carry out actions; implies -v
  -f              Force action
`

func main() {
	opts := config.NewOpts()
	var configFile, basepath, command, arg1, arg2 string
	var removeFiles bool
	cfg := config.NewDefaultConfig()

	flag.StringVar(&configFile, "config", "", "specify configuration file")
	flag.StringVar(&basepath, "basepath", "", "specify a base path")
	opts.AddFlagsToFlagset(flag.CommandLine)
	flag.Usage = func () {
		subst := map[string]string{
			"myself": path.Base(os.Args[0]),
		}
		fmt.Fprintln(os.Stderr, fns.Template(usageMessage, subst))
		os.Exit(0)
	}
	flag.Parse()
	opts.AfterParse()

	if len(basepath) < 1 {
		basepath = os.Getenv("LAYERROOT")
	}
	if len(configFile) < 1 {
		var choices []string
		fromenv := os.Getenv("LAYERCONF")
		if len(fromenv) > 0 {
			choices = append(choices, fromenv)
		}
		home := os.Getenv("HOME")
		if len(home) > 0 {
			choices = append(choices, home + "/.layercake")
		}
		parentdir := path.Dir(path.Dir(os.Args[0]))
		if len(parentdir) > 0 {
			choices = append(choices, parentdir + "/etc/layercake")
			choices = append(choices, parentdir + "/etc/layercake.conf")
		}
		choices = append(choices, "/etc/layercake.conf")
		for _, filepath := range choices {
			if fs.IsFile(filepath) {
				configFile = filepath
				break
			}
		}
	}
	if len(configFile) > 0 {
		err := cfg.ReadConfigFile(configFile)
		if nil != err {
			fatal("Can't read configuration file: %s", err)
		}
		basepath = cfg.GetCfg("basepath")
		if len(basepath) < 1 {
			fatal("Configuration file has no valid basepath")
		}
	} else if len(basepath) == 0 {
		fatal("No configuration file found and no base path specified")
	}
	if len(basepath) > 0 {
		cfg.Set("basepath", basepath)
	}

	basepath, err := filepath.Abs(basepath)
	if nil != err {
		fatal("% in specified %s", err, basepath)
	}
	if !fs.IsDir(basepath) {
		fatal("Specified base path %s does not exist", basepath)
	}
	cfg.Set("basepath", basepath)

	missing, haveAbsPath := cfg.CheckConfigPaths()


	var flagset *flag.FlagSet
	var minNeeded, maxNeeded int
	sw := localSwitches{}
	command = "status"
	withHelpReminder := true
	args := flag.Args()
	if len(args) > 0 {
		command = args[0]
		withHelpReminder = false
		flagset = flag.NewFlagSet("command", flag.ContinueOnError)
		opts.AddFlagsToFlagset(flagset)
		flagset.Usage = flag.Usage
		switch command {
		case "add", "rebase":
			minNeeded, maxNeeded = 1, 2
		case "remove":
			minNeeded, maxNeeded = 1, 1
			sw.Add("files", &removeFiles)
		case "rename":
			minNeeded, maxNeeded = 2, 2
		case "shell", "mkdir", "mkdirs", "mount", "umount", "chroot":
			minNeeded, maxNeeded = 0, 1
		}
		sw.AddToFlagset(flagset)
		flagset.Parse(args[1:])
		opts.AfterParse()
		args = flagset.Args()
	}

	argv := []*string{&arg1, &arg2}
	for argX := range argv {
		if maxNeeded > argX {
			if len(args) < 1 {
				if minNeeded > argX {
					fatal("Not enough arguments to the %s command", command)
				}
				break
			}
			*argv[argX] = args[0]
			flagset = flag.NewFlagSet("arg1", flag.ContinueOnError)
			opts.AddFlagsToFlagset(flagset)
			flagset.Usage = flag.Usage
			sw.AddToFlagset(flagset)
			flagset.Parse(args[1:])
			opts.AfterParse()
			args = flagset.Args()
		}
	}


	switch command {
	case "init":
		if len(missing) == 0 {
			fmt.Println("Base directories already set up:  nothing to do")
			return
		}
		if haveAbsPath {
			fatal("At least one base element has an absolute path: need manual setup")
		}
		for _, dir := range cfg.GetDirPaths() {
			_, err = fs.MakeDir(dir)
			if nil != err {
				fatal("%s creating directory %s", err, dir)
			}
		}
		err = fs.PrepHtmlDir(cfg.GetPath("htmlroot"))
		if nil != err {
			fatal("%s setting up HTML directory", err)
		}
		err = fs.MakeEmptyLayersFile(cfg.GetPath("layersfile"))
		if nil != err {
			fatal("%s making empty layers file", err)
		}
	case "status":
		if len(missing) == 0 {
			fns.ShowStatus(cfg)
		} else if len(missing) < 3 {
			for _, name := range missing {
				val := cfg.GetPath(name)
				if len(val) > 0 {
					fmt.Printf("Base item %s (path %s) is missing\n", name, val)
				} else {
					fmt.Printf("Base item %s is not specified\n", name)
				}
			}
			fatal("Cannot proceed unless all base components exist")
		} else if haveAbsPath {
			fmt.Println("Base directories/layers file not set up; need manual setup")
		} else {
			fmt.Println("Base directory not set up; init command creates them")
		}
		if withHelpReminder {
			fmt.Println("Use --help switch for command usage")
		}
	default:
		if len(missing) > 0 {
			fatal("Base directory not set up--cannot proceed")
		}
		layersfileName := cfg.GetPath("layersfile")
		layers, err = manage.ReadLayersfile(layersfileName)
		if nil != err {
			fatal(err.Error())
			fatal("%s reading layers file %s", err, cfg.GetPath("layersfile"))
		}
		mounts, err := fs.ProbeMounts()
		if nil != err {
			fatal("%s probing mounts", err)
		}
		br := cfg.GetPath("buildroot")
		if br[len(br)-1] != '/' {
			br += "/"
		}
		inuse, err := fs.FindUses(br, -1)
		if nil != err {
			fatal("%s finding users in buildroot", err)
		}
		err = layers.ProbeLayerstate(cfg, mounts, inuse)
		if nil != err {
			fatal("%s probing layers", err)
		}

		switch command {
		case "list":
			llist := layers.Layers()
			if len(llist) < 1 {
				fmt.Println("No layers found")
				break
			}
			layers.CautionIfCaution(false, true)
			tbl := fns.NewAdaptiveTable("   l    l   c   l")
			tbl.SetLabels("Layer", "Parent", "Usage", "Setup State")
			for _, layer := range llist {
				var basespec string
				var more []string
				if len(layer.Base) > 0 {
					basespec = "<- " + layer.Base
				} else {
					basespec = "(base level)"
				}
				if !layer.Defined {
					more = []string{"undefined"}
				}
				if layer.Chroot {
					more = append(more, "chroot")
				} else if layer.Busy {
					more = append(more, "busy")
				}
				if layer.BaseError {
					more = append(more, "base error")
				}
				tbl.Print(layer.Name, basespec, strings.Join(more, ", "),
					layers.DescribeState(layer, opts.Verbose))
			}
			tbl.Flush()
		case "add":
			err = layers.AddLayer(arg1, arg2, opts)
			if nil != err {
				fatal(err.Error())
			}
			layers.CautionIfCaution(true, false)
			if !opts.Pretend {
				err = layers.WriteLayersfile(layersfileName)
				if nil != err {
					fatal("%s writing to %s", err, layersfileName)
				}
			}
		case "remove":
			err = layers.RemoveLayer(arg1, removeFiles, opts)
			if nil != err {
				fatal(err.Error())
			}
			layers.CautionIfCaution(true, false)
			if !opts.Pretend {
				err = layers.WriteLayersfile(layersfileName)
				if nil != err {
					fatal("%s writing to %s", err, layersfileName)
				}
			}
		case "rename":
			err = layers.RenameLayer(arg1, arg2, opts)
			if nil != err {
				fatal(err.Error())
			}
			layers.CautionIfCaution(true, false)
			if !opts.Pretend {
				err = layers.WriteLayersfile(layersfileName)
				if nil != err {
					fatal("%s writing to %s", err, layersfileName)
				}
			}
		case "rebase":
			err = layers.RebaseLayer(arg1, arg2, opts)
			if nil != err {
				fatal(err.Error())
			}
			layers.CautionIfCaution(true, false)
			if !opts.Pretend {
				err = layers.WriteLayersfile(layersfileName)
				if nil != err {
					fatal("%s writing to %s", err, layersfileName)
				}
			}
		case "shell":
			err = layers.Shell(arg1, opts)
			if nil != err {
				fatal(err.Error())
			}
			layers.CautionIfCaution(true, false)
		case "mkdir", "mkdirs":
			err = layers.Makedirs(arg1, opts)
			if nil != err {
				fatal(err.Error())
			}
			layers.CautionIfCaution(true, false)
		case "mount":
			err = layers.Mount(arg1, mounts, opts)
			if nil != err {
				fatal(err.Error())
			}
			layers.CautionIfCaution(true, false)
		case "umount":
			err = layers.Unmount(arg1, opts)
			if nil != err {
				fatal(err.Error())
			}
			layers.CautionIfCaution(true, false)
		case "chroot":
			err = layers.Chroot(arg1, mounts, cfg, opts)
			if nil != err {
				fatal(err.Error())
			}
		case "shake":
			err = layers.Shake(opts)
			if nil != err {
				fatal(err.Error())
			}
		default:
			fatal("Unknown command %s", command)
		}
	}
}



type localSwitches struct {
	sw []localSwitch
}

type localSwitch struct {
	name string
	pt interface{}
}

func (ls *localSwitches) Add(name string, pt interface{}) {
	ls.sw = append(ls.sw, localSwitch{name, pt})
}

func (ls *localSwitches) AddToFlagset(fs *flag.FlagSet) {
	for _, sw := range ls.sw {
		switch sw.pt.(type) {
		case *bool:
			bp := sw.pt.(*bool)
			fs.BoolVar(bp, sw.name, *bp, "")
		case *string:
			sp := sw.pt.(*string)
			fs.StringVar(sp, sw.name, *sp, "")
		}
	}
}


func fatal(base string, params...interface{}) {
	fmt.Fprintf(os.Stderr, base, params...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

