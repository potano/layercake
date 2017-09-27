package config

import (
	"fmt"
	"flag"
)

type Opts struct {
	Verbose, Pretend, Force bool
}

func NewOpts() *Opts {
	return &Opts{}
}

func (o *Opts) AddFlagsToFlagset(fs *flag.FlagSet) {
	fs.BoolVar(&o.Verbose, "v", o.Verbose, "verbose output")
	fs.BoolVar(&o.Pretend, "p", o.Pretend, "pretend to undertake actions")
	fs.BoolVar(&o.Force, "f", o.Force, "force actions")
}

func (o *Opts) AfterParse() {
	if o.Pretend {
		o.Verbose = true
	}
}

func (o *Opts) Describe(msg string, params...interface{}) {
	var prefix string
	if o.Pretend {
		prefix = "Would "
	} else if o.Verbose {
		prefix = "Action: "
	}
	if o.Pretend || o.Verbose {
		if o.Force {
			prefix += "force "
		}
		fmt.Printf(prefix + msg + "\n", params...)
	}
}

