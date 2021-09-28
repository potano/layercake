package config

import (
	"fmt"
	"flag"
)


type Opts struct {
	Verbose, Pretend, Debug, Force bool
	Writer func (string, ...interface{})
}


func NewOpts() *Opts {
	return &Opts{
		Writer: func (msg string, parms...interface{}) {
			fmt.Printf(msg + "\n", parms...)
		},
	}
}


func (o *Opts) MakeReaderOpts(keepDebug bool) *Opts {
	return &Opts{
		Verbose: o.Verbose,
		Pretend: false,
		Debug: o.Debug && keepDebug,
		Writer: o.Writer,
	}
}


func (o *Opts) AddFlagsToFlagset(fs *flag.FlagSet) {
	fs.BoolVar(&o.Verbose, "v", o.Verbose, "verbose output")
	fs.BoolVar(&o.Pretend, "p", o.Pretend, "pretend to undertake actions")
	fs.BoolVar(&o.Debug, "debug", o.Debug, "display debugging output")
	fs.BoolVar(&o.Force, "f", o.Force, "force actions")
}


func (o *Opts) MakePretender() func (string, ...interface{}) bool {
	doIt := !o.Pretend
	prefix := "action: "
	if o.Pretend {
		prefix = "would "
	}
	if o.Debug {
		writer := o.Writer
		return func (msg string, parms...interface{}) bool {
			writer(fmt.Sprintf(prefix + msg, parms...))
			return doIt
		}
	}
	return func (msg string, parms...interface{}) bool { return doIt }
}


func (o *Opts) DescribeIfVerbose(msg string, parms...interface{}) {
	if o.Verbose {
		o.Writer(msg, parms)
	}
}

