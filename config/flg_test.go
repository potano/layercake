package config

import (
	"flag"
	"strings"
	"testing"
)



type switchExpectation struct {
	name string
	pt interface{}
	value interface{}
}


type augmentedCab struct {
	t *testing.T
	cab *CommandArgBuilder
	firstArgs []string
	locals []switchExpectation
}


func (ac *augmentedCab) addSwitch(name string, pt interface{}, value interface{}) {
	ac.cab.AddSwitch(name, pt)
	ac.locals = append(ac.locals, switchExpectation{name, pt, value})
}


func (ac *augmentedCab) check(cmd string, args[]string, verbose, pretend, debug, force bool) {
	command := ""
	firstArgs := ac.firstArgs
	if len(firstArgs) > 0 {
		command = firstArgs[0]
	}
	if cmd != command {
		ac.t.Errorf("Command is '%s'", command)
	}
	finalArgs := ac.cab.ParseArgsSetFlags(firstArgs)
	if !stringSlicesEqual(args, finalArgs) {
		ac.t.Errorf("Arguments '%s'", strings.Join(args, "', '"))
	}
	ac.checkOpts(verbose, pretend, debug, force)
	ac.checkSwitches()
}


func stringSlicesEqual(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i, v := range s1 {
		if s2[i] != v {
			return false
		}
	}
	return true
}


func (ac *augmentedCab) checkOpts(verbose, pretend, debug, force bool) {
	opts := ac.cab.Opts
	if opts.Verbose != verbose {
		ac.t.Errorf("Verbose=%v", opts.Verbose)
	}
	if opts.Pretend != pretend {
		ac.t.Errorf("Pretend=%v", opts.Pretend)
	}
	if opts.Debug != debug {
		ac.t.Errorf("Debug=%v", opts.Debug)
	}
	if opts.Force != force {
		ac.t.Errorf("Force=%v", opts.Force)
	}
}


func (ac *augmentedCab) checkSwitches() {
	for _, sw := range ac.locals {
		switch sw.pt.(type) {
		case *bool:
			bv := sw.pt.(*bool)
			if *bv != sw.value.(bool) {
				ac.t.Errorf("%s=%v", sw.name, *bv)
			}
		case *string:
			sv := sw.pt.(*string)
			if *sv != sw.value.(string) {
				ac.t.Errorf("%s=%s", sw.name, *sv)
			}
		}
	}
}


type optTest struct {
	name, cmdline string
	fn func (ac *augmentedCab)
}


func TestOpts(t *testing.T) {
	tests := []optTest{
		{"empty list", "c", func (ac *augmentedCab) {
			ac.check("", []string{}, false, false, false, false)
		}},
		{"simple command", "c init", func (ac *augmentedCab) {
			ac.check("init", []string{}, false, false, false, false)
		}},
		{"global flag before command", "c -v list", func (ac *augmentedCab) {
			ac.check("list", []string{}, true, false, false, false)
		}},
		{"global flag after command", "c list -v", func (ac *augmentedCab) {
			ac.check("list", []string{}, true, false, false, false)
		}},
		{"boolean command flag", "c unmount -all", func (ac *augmentedCab) {
			var all_mounts bool
			ac.addSwitch("all", &all_mounts, true)
			ac.check("unmount", []string{}, false, false, false, false)
		}},
		{"string command flag", "c write -o outfile", func (ac *augmentedCab) {
			var filename string
			ac.addSwitch("o", &filename, "outfile")
			ac.check("write", []string{}, false, false, false, false)
		}},
		{"command with argument", "c add base", func (ac *augmentedCab) {
			ac.check("add", []string{"base"}, false, false, false, false)
		}},
		{"multiple arguments, interspersed options", "c -p add derived -debug base",
			func (ac *augmentedCab) {
			ac.check("add", []string{"derived", "base"}, false, true, true, false)
		}},
		{"local switches with global switches", "c -v run -prog a b -force",
			func (ac *augmentedCab) {
			var program string
			ac.addSwitch("prog", &program, "a")
			ac.check("run", []string{"b"}, true, false, false, true)
		}},
	}
	for _, tst := range tests {
		t.Run(tst.name, func (t *testing.T) {
			cab := NewCommandArgBuilder()
			flagset := flag.NewFlagSet("", flag.ExitOnError)
			cab.AddFlagsToFlagset(flagset)
			flagset.Parse(strings.Split(tst.cmdline, " ")[1:])
			firstArgs := flagset.Args()
			ac := &augmentedCab{t: t, cab: cab, firstArgs: firstArgs}
			tst.fn(ac)
		})
	}
}

