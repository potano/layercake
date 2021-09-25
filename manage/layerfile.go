package manage

import (
	"io"
	"strings"

	"potano.layercake/fs"
)


func ReadLayerFile(filename string, harderror bool) (*Layerinfo, error) {
	cursor, err := fs.NewTextInputFileCursor(filename)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	layer := &Layerinfo{
		ConfigMounts: []NeededMountType{},
		ConfigExports: []NeededMountType{},
	}

	for {
		line, err := cursor.GetLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) < 1 || line[0] == '#' || (len(line) > 1 && "//" == line[:2]) {
			continue
		}
		switch fields[0] {
		case "base":
			if len(fields) < 2 {
				cursor.LogError("No base specified")
			} else if len(layer.Base) > 0 && layer.Base != fields[1] {
				cursor.LogError("New conflicting setting of base property")
			} else {
				layer.Base = fields[1]
			}
		case "import":
			if len(fields) < 4 {
				cursor.LogError("Incomplete import specification")
			} else {
				layer.ConfigMounts = append(layer.ConfigMounts,
					NeededMountType{fields[3], fields[2], fields[1]})
			}
		case "export":
			if len(fields) < 4 {
				cursor.LogError("Incomplete export specification")
			} else {
				layer.ConfigExports = append(layer.ConfigExports,
					NeededMountType{fields[3], fields[2], fields[1]})
			}
		default:
			cursor.LogError("Unknown layerconf keyword '" + fields[0] + "'")
		}
	}
	if cursor.HaveError() {
		if harderror {
			return nil, cursor.GetError()
		}
		layer.Messages = append(layer.Messages, cursor.GetMessages()...)
		layer.State = Layerstate_error
	}
	return layer, nil
}


func WriteLayerfile(filename string, layer *Layerinfo) error {
	cursor, err := fs.NewTextOutputFileCursor(filename)
	if nil != err {
		return err
	}
	defer cursor.Close()
	if len(layer.Base) > 0 {
		cursor.Printf("base %s\n\n", layer.Base)
	}
	for _, mnt := range layer.ConfigMounts {
		cursor.Printf("import %s %s %s\n", mnt.Fstype, mnt.Source, mnt.Mount)
	}
	if len(layer.ConfigExports) > 0 {
		cursor.Printf("\n");
	}
	for _, mnt := range layer.ConfigExports {
		cursor.Printf("export %s %s %s\n", mnt.Fstype, mnt.Source, mnt.Mount)
	}
	return nil
}


func (layers *Layerdefs) writeLayerFile(layer *Layerinfo) error {
	return WriteLayerfile(layers.layerconfigFilePath(layer), layer)
}

