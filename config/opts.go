package config

import (
	"flag"
)


type Opts struct {
	Verbose, Pretend, Debug, Force bool
}


type CommandArgBuilder struct {
	Usage func ()
	Opts *Opts
	sw []localSwitch
}


type localSwitch struct {
	name string
	pt interface{}
}


func NewCommandArgBuilder() *CommandArgBuilder {
	cab := &CommandArgBuilder{Opts: &Opts{}}
	cab.AddSwitch("v", &cab.Opts.Verbose)
	cab.AddSwitch("p", &cab.Opts.Pretend)
	cab.AddSwitch("debug", &cab.Opts.Debug)
	cab.AddSwitch("force", &cab.Opts.Force)
	return cab
}


func (cab *CommandArgBuilder) AddSwitch(name string, pt interface{}) {
	cab.sw = append(cab.sw, localSwitch{name, pt})
}


func (cab *CommandArgBuilder) AddFlagsToFlagset(flgs *flag.FlagSet) {
	for _, sw := range cab.sw {
		switch sw.pt.(type) {
		case *bool:
			bp := sw.pt.(*bool)
			flgs.BoolVar(bp, sw.name, *bp, "")
		case *string:
			sp := sw.pt.(*string)
			flgs.StringVar(sp, sw.name, *sp, "")
		}
	}
	if cab.Usage != nil {
		flgs.Usage = cab.Usage
	}
}


func (cab *CommandArgBuilder) ParseArgsSetFlags(args []string) []string {
	cmdargs := make([]string, 0, len(args))
	firstPass := true
	for len(args) > 0 {
		if !firstPass {
			cmdargs = append(cmdargs, args[0])
		}
		firstPass = false
		flagset := flag.NewFlagSet("", flag.ExitOnError)
		cab.AddFlagsToFlagset(flagset)
		flagset.Parse(args[1:])
		args = flagset.Args()
	}
	return cmdargs
}

