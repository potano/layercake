package manage

import (
	"os"
	"fmt"
	"bufio"
	"regexp"
	"strings"
)

func ReadLayersfile(filename string) (*Layerdefs, error) {
	layers := NewLayerdefs()
	var errlist []string

	re, err := regexp.Compile("^\\s*(\\w+)\\s*(\\w*)\\s*$")
	if nil != err {
		return nil, err
	}

	file, err := os.OpenFile(filename, os.O_RDONLY, 0666)
	if nil != err {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	lineno := 0
	for scanner.Scan() {
		lineno++
		line := scanner.Text()
		if len(line) < 1 || line[0] == '#' || (len(line) > 1 && "//" == line[:2]) {
			continue
		}
		matches := re.FindStringSubmatch(line)
		if len(matches) < 1 {
			err = fmt.Errorf("bad format")
		} else {
			err = layers.addLayerinfo(Layerinfo{
				Name: matches[1],
				Base: matches[2],
				Defined: true,
			})
		}
		if nil != err {
			msg := fmt.Sprintf("%s at line %d", err.Error(), lineno)
			errlist = append(errlist, msg)
		}
	}
	if len(errlist) > 0 {
		err = fmt.Errorf("Error(s) found in %s:\n   %s", filename,
			strings.Join(errlist, "\n   "))
	}
	return layers, err
}

func (ld *Layerdefs) WriteLayersfile(filename string) error {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0666)
	if nil != err {
		return err
	}
	defer file.Close()
	for _, item := range ld.layers {
		if !item.Defined {
			continue
		}
		if len(item.Base) > 0 {
			fmt.Fprintf(file, "%s %s\n", item.Name, item.Base)
		} else {
			fmt.Fprintf(file, "%s\n", item.Name)
		}
	}
	return nil
}

