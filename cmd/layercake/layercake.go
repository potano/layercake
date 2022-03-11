// Layercake: manager of layers of build chroots
// Mike Thompson 5/5/2017

package main

import (
	"os"
	"fmt"
	"flag"
	"strings"

	"potano.layercake/fs"
	"potano.layercake/fns"
	"potano.layercake/config"
	"potano.layercake/manage"
	"potano.layercake/defaults"
)


const mainUsageMessage = `Manages layers in the build chroot
Usage:
  {myself} [main-options] <command> [command-options]
These commands are available
  init             Establish the layer system in configured directory
  status [layer]   Display the status of the build root or a single layer
  list [-v]        Display list of layers showing status
                   Add -v for a more verbose listing
  add <layer> [base]  Add a layer and indicate layer it derives
  rename <layer> <newname>  Rename a layer
  rebase <layer> [newbase]  Change a layer's base layer
  remove <layer> [-files]   Remove a layer; use -files to remove files and
                  directories also
  shell <layer>   Starts a shell in the named layer.  Useful for setting
                  up a base layer before it can be made mountable.
  mkdirs <layer>  Create or recreate needed directories in named layer
                  in all layers
  mount <layer>   Mount the named layer
  umount [layer]  Mount per-layer directories in named layer.  Use the
                  -all switch to unmount all non-busy layers
  shake           Remount all current overlayfs mounts to ensure that
                  changes in lower layers propagate upward to mounted
		  layers
  chroot <layer>  Starts a chroot using named layer

Main options
  --config <file> Specify/override configuration-file location
  --basepath <path>  Specify/override build-root basepath

Global options (may be specified anywhere in the command line)
  -v              Verbose mode: show actions (to be) taken
  -p              Pretend to carry out actions
  -force          Force action
  -debug          Show debugging output
`

const argumentHintMessage = `
Usage:
  {myself} [main-options] <command> [command-args-and-options]
  {myself} -h
  {myself} -version
`


func argumentHintMessageAndExit() {
	fns.TemplatedExitMessage(argumentHintMessage, 1, map[string]string{})
}


func main() {
	cab := config.NewCommandArgBuilder()
	cab.Usage = argumentHintMessageAndExit
	var configFile, basepath string
	var help, showVersion bool

	flag.StringVar(&configFile, "config", "", "specify configuration file")
	flag.StringVar(&basepath, "basepath", "", "specify a base path")
	flag.BoolVar(&help, "help", false, "help")
	flag.BoolVar(&help, "h", false, "help")
	flag.BoolVar(&showVersion, "version", false, "version")
	cab.AddFlagsToFlagset(flag.CommandLine)
	flag.Parse()
	if help {
		fns.TemplatedExitMessage(mainUsageMessage, 0, map[string]string{})
	}
	if showVersion {
		fmt.Println(defaults.Version)
		os.Exit(0)
	}

	cfg, err := config.Load(configFile, basepath)
	if err != nil {
		fatal(err.Error())
	}

	args := flag.Args()
	cmdinfo := commandInfo{
		cfg: cfg,
		isDefaultCommand: len(args) == 0,
		cab: cab,
	}

	command := defaults.DefaultCommand
	if len(args) > 0 {
		command = args[0]
	}

	fn := map[string]func(commandInfo){
		"init": initCommand,
		"status": statusCommand,
		"list": listCommand,
		"add": addCommand,
		"remove": removeCommand,
		"rename": renameCommand,
		"rebase": rebaseCommand,
		"shell": shellCommand,
		"mkdirs": mkdirsCommand,
		"mount": mountCommand,
		"unmount": unmountCommand,
		"umount": unmountCommand,
		"chroot": chrootCommand,
		"shake": shakeCommand,
	}[command]

	if fn == nil {
		fatal("Unknown command %s", command)
	}
	fn(cmdinfo)
}


type commandInfo struct {
	cfg *config.ConfigType
	isDefaultCommand bool
	cab *config.CommandArgBuilder
}


func (ci commandInfo) getArgs(minNeeded, maxNeeded int) []string {
	args := ci.cab.ParseArgsSetFlags(flag.Args())
	if len(args) < minNeeded {
		fatal("Not enough command-line arguments (need %d)", minNeeded)
	}
	if len(args) > maxNeeded {
		fatal("Too many command-line arguments (max %d)", maxNeeded)
	}
	for len(args) < maxNeeded {
		args = append(args, "")
	}
	fs.WriteOK = fs.MakePretender(ci.cab.Opts.Pretend, ci.cab.Opts.Debug, debugPrintf)
	return args
}


func (ci commandInfo) failOnMissingBaseSetup() {
	missing := manage.CheckBaseSetUp(ci.cfg)
	if len(missing) > 0 {
		fatal("Missing item(s):\n  %s\nCannot proceed unless all exist",
			strings.Join(missing, "\n  "))
	}
}


func (ci commandInfo) getLayers() *manage.Layerdefs {
	ci.failOnMissingBaseSetup()
	layers, err := manage.FindLayers(ci.cfg, ci.cab.Opts)
	if nil != err {
		fatal(err.Error())
	}
	inuse, err := fs.FindLayersInUse(ci.cfg.Layerdirs)
	if nil != err {
		fatal("%s finding users in buildroot", err)
	}
	err = layers.ProbeAllLayerstate(inuse)
	if nil != err {
		fatal("%s probing layers", err)
	}
	return layers
}





func initCommand(cmdinfo commandInfo) {
	cmdinfo.getArgs(0, 0)
	err := manage.InitLayercakeBase(cmdinfo.cfg)
	if err != nil {
		fatal("init: %s", err.Error())
	}
}


func statusCommand(cmdinfo commandInfo) {
	args := cmdinfo.getArgs(0, 1)
	cmdinfo.failOnMissingBaseSetup()
	if len(args[0]) == 0 {
		fmt.Printf("Base directories set up OK at %s\n", cmdinfo.cfg.Basepath)
		if cmdinfo.isDefaultCommand {
			fmt.Println("Use --help switch for command usage")
		}
		return
	}

	name := args[0]
	layers := cmdinfo.getLayers()
	layer := layers.Layer(name)
	if layer == nil {
		fatal("Layer %s not found", name)
	}
	if len(layer.Base) > 0 {
		fmt.Printf("Layer: %s\nParent layer: %s\n", name, layer.Base)
	} else {
		fmt.Printf("Base layer: %s\n", name)
	}
	info := layers.DescribeState(layer, true)
	fmt.Printf("State: %s\n", info[0])

	if len(info) > 1 {
		fmt.Println("")
		for _, line := range info[1:] {
			fmt.Println(line)
		}
	}
}


func listCommand(cmdinfo commandInfo) {
	cmdinfo.getArgs(0, 0)
	layers := cmdinfo.getLayers()
	llist := layers.Layers()
	if len(llist) < 1 {
		fmt.Println("No layers found")
		return
	}
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
		if layer.Chroot {
			more = append(more, "chroot")
		} else if layer.Busy || layer.Overlain {
			more = append(more, "busy")
		}
		tbl.Print(layer.Name, basespec, strings.Join(more, ", "),
		layers.DescribeState(layer, cmdinfo.cab.Opts.Verbose))
	}
	tbl.Flush()
}


func addCommand(cmdinfo commandInfo) {
	var configFile string
	cmdinfo.cab.AddSwitch("configfile", &configFile)
	args := cmdinfo.getArgs(1, 2)
	layers := cmdinfo.getLayers()
	err := layers.AddLayer(args[0], args[1], configFile)
	if nil != err {
		fatal(err.Error())
	}
}


func removeCommand(cmdinfo commandInfo) {
	var removeFiles bool
	cmdinfo.cab.AddSwitch("files", &removeFiles)
	args := cmdinfo.getArgs(1, 1)
	layers := cmdinfo.getLayers()
	err := layers.RemoveLayer(args[0], removeFiles)
	if nil != err {
		fatal(err.Error())
	}
}


func renameCommand(cmdinfo commandInfo) {
	args := cmdinfo.getArgs(2, 2)
	layers := cmdinfo.getLayers()
	err := layers.RenameLayer(args[0], args[1])
	if nil != err {
		fatal(err.Error())
	}
}


func rebaseCommand(cmdinfo commandInfo) {
	args := cmdinfo.getArgs(1, 2)
	layers := cmdinfo.getLayers()
	err := layers.RebaseLayer(args[0], args[1])
	if nil != err {
		fatal(err.Error())
	}
}


func shellCommand(cmdinfo commandInfo) {
	args := cmdinfo.getArgs(1, 1)
	layers := cmdinfo.getLayers()
	err := layers.Shell(args[0])
	if nil != err {
		fatal(err.Error())
	}
}


func mkdirsCommand(cmdinfo commandInfo) {
	args := cmdinfo.getArgs(1, 1)
	layers := cmdinfo.getLayers()
	err := layers.Makedirs(args[0])
	if nil != err {
		fatal(err.Error())
	}
}


func mountCommand(cmdinfo commandInfo) {
	args := cmdinfo.getArgs(1, 1)
	layers := cmdinfo.getLayers()
	err := layers.Mount(args[0])
	if nil != err {
		fatal(err.Error())
	}
}


func unmountCommand(cmdinfo commandInfo) {
	var all bool
	cmdinfo.cab.AddSwitch("all", &all)
	args := cmdinfo.getArgs(0, 1)
	layers := cmdinfo.getLayers()
	err := layers.Unmount(args[0], all)
	if nil != err {
		fatal(err.Error())
	}
}


func chrootCommand(cmdinfo commandInfo) {
	args := cmdinfo.getArgs(1, 1)
	layers := cmdinfo.getLayers()
	err := layers.Chroot(args[0])
	if nil != err {
		fatal(err.Error())
	}
}


func shakeCommand(cmdinfo commandInfo) {
	layers := cmdinfo.getLayers()
	err := layers.Shake()
	if nil != err {
		fatal(err.Error())
	}
}






func fatal(base string, params...interface{}) {
	fmt.Fprintf(os.Stderr, base, params...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}


func debugPrintf(base string, params...interface{}) {
	fmt.Printf(base + "\n", params...)
}

