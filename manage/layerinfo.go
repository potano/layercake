package manage

import (
	"fmt"
	"path"
	"unicode"
)


func isLegalLayerName(name string) bool {
	for pos, c := range name {
		if !unicode.In(c, unicode.L, unicode.Nd) && c != '_' && (c != '-' || pos == 0) {
			return false
		}
	}
	return true
}


type nametest struct {
	name string
	mask int
	desc string
}

const (
	name_need = 1 << iota
	name_free
	name_optional
)

func (ld *Layerdefs) testName(tests...nametest) error {
	for _, test := range tests {
		name := test.name
		mask := test.mask
		desc := test.desc
		if len(name) < 1 {
			if (mask & name_optional) > 0 {
				continue
			}
			return fmt.Errorf("%s name is not set", desc)
		}
		if !isLegalLayerName(name) {
			return fmt.Errorf("%s name '%s' is not legal", desc, name)
		}
		if (mask & name_free) > 0 {
			if _, have := ld.layermap[name]; have {
				return fmt.Errorf("%s name '%s' already exists", desc, name)
			}
		} else if (mask & name_need) > 0 {
			if _, have := ld.layermap[name]; !have {
				return fmt.Errorf("%s name '%s' does not exist", desc, name)
			}
		}
	}
	return nil
}



func (ld *Layerdefs) getAncestorsAndSelf(name string) ([]*Layerinfo, error) {
	err := ld.testName(nametest{name, name_need, "Layer"})
	if nil != err {
		return nil, err
	}
	pos := len(ld.normalizedOrder)
	chain := make([]*Layerinfo, pos)
	for len(name) > 0 {
		layer := ld.layermap[name]
		pos--
		chain[pos] = layer
		name = layer.Base
	}
	return chain[pos:], nil
}


func (ld *Layerdefs) findLayerBase(layer *Layerinfo) *Layerinfo {
	for len(layer.Base) > 0 {
		layer = ld.layermap[layer.Base]
	}
	return layer
}


func (ld *Layerdefs) inAnyLayerDirectory(pathname string) bool {
	rootdir := ld.cfg.Layerdirs
	for len(pathname) >= len(rootdir) {
		if pathname == rootdir {
			return true
		}
		pathname = path.Dir(pathname)
	}
	return false
}

