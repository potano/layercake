// Copyright © 2017, 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

// Generates the distribution tarball and versioned ebuild

package main

import (
	"os"
	"fmt"
	"flag"
	"bytes"
	"os/exec"
	"strings"
	"potano.layercake/fs"
	"potano.layercake/fns"
	"potano.layercake/defaults"
)


const ebuildDir = "ebuild/"
const distDir = "dist/"

const scriptTemplate =
`find . ! -name \\*.swp | \\
	grep -vP '^\\./(\\.$|\\.git|bin/\\w|dist|doc/.*\\.(texi|xml))' | \\
	xargs tar czf dist/{nameVer}.tar.gz --no-recursion --xform 's,^\\.,{nameVer},'
cp ebuild/{nameVer}.ebuild dist
`


func main() {
	var showScript bool
	flag.BoolVar(&showScript, "show", false, "show script")
	flag.Parse()

	atomVer := strings.ReplaceAll(defaults.Version, "-", "_")
	nameVer := "layercake-" + atomVer

	ebuildName := ebuildDir + "layercake.ebuild"
	versionedEbuildName := ebuildDir + "layercake-" + atomVer + ".ebuild"

	if !fs.IsFile(ebuildName) {
		fatal("Can't find %s", ebuildName)
	}
	if fs.IsFile(versionedEbuildName) {
		fatal("%s exists", versionedEbuildName)
	}

	if !fs.IsDir(distDir) {
		err := fs.Mkdir(distDir)
		if err != nil {
			fatal("can't create directory %s: %s", distDir, err)
		}
	}

	if err := fs.Rename(ebuildName, versionedEbuildName); err != nil {
		fatal("%s trying to rename %s to %s", err, ebuildName, versionedEbuildName)
	}
	defer func() {
		err := fs.Rename(versionedEbuildName, ebuildName)
		if err != nil {
			fatal("%s trying to rename %s to %s", err, versionedEbuildName, ebuildName)
		}
	}()

	script := fns.Template(scriptTemplate, map[string]string{"nameVer": nameVer})

	if showScript {
		fmt.Println(script)
	} else {
		cmd := exec.Command("/bin/bash")
		cmd.Stdin = strings.NewReader(script)
		var out bytes.Buffer
		cmd.Stdout = &out;
		err := cmd.Run()
		if err != nil {
			fatal(err.Error())
		}
		fmt.Println(out.String())
	}
}


func fatal(base string, params...interface{}) {
	fmt.Fprintf(os.Stderr, base, params...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

