package manage

import "sort"

type LayerdefSorter struct {
	sortKeys map[string]string
	norm []string
}

func (a LayerdefSorter) Len() int {
	return len(a.norm)
}

func (a LayerdefSorter) Swap(i, j int) {
	a.norm[i], a.norm[j] = a.norm[j], a.norm[i]
}

func (a LayerdefSorter) Less(i, j int) bool {
	return a.sortKeys[a.norm[i]] < a.sortKeys[a.norm[j]]
}

func (layers *Layerdefs) normalizeOrder() {
	norm := []string{}
	sortKeys := map[string]string{}
	for name, layer := range layers.layermap {
		norm = append(norm, name)
		s := name
		for {
			base := layer.Base
			s = base + "/" + s
			if len(base) < 1 {
				break
			}
			layer = layers.layermap[base]
		}
		sortKeys[name] = s
	}

	sorter := &LayerdefSorter{sortKeys: sortKeys, norm: norm}
	sort.Sort(sorter)
	layers.normalizedOrder = sorter.norm
}

