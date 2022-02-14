package vdb

import (
	"fmt"
	"strconv"
	"strings"
	"encoding/hex"

	"potano.layercake/portage/atom"
)


/*
  Parses CONTENTS-file lines for installed package.  This file, which seems not to be documented
  anywhere, contains a line for each filesystem entry the package installs.  Each line begins
  with three characters which indicate the file type; the balance of the line is formatted
  according to the type.  File names may contain spaces :( and are unescaped when entered into
  the file.  This rules out an approach of a simple parsing of non-blank fields.
*/

const (
	FileType_none = iota
	FileType_dir		// type keyword "dir"
	FileType_file		// type keyword "obj"
	FileType_symlink	// type keyword "sym"
	FileType_hardlink	// not present in CONTENTS files
	FileType_device		// not present in CONTENTS files
)

const PermBits = 07777


type FileInfo struct {
	Name string
	UnixTime int64
	MD5 [16]byte
	Type uint8
}


func GetInstalledFileInfo(atoms atom.AtomSlice) ([]FileInfo, error) {
	var allFiles []FileInfo
	recorded := map[string]bool{}
	for _, atm := range atoms {
		files, err := GetAtomFileInfo(atm)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			name := file.Name
			if !recorded[name] {
				recorded[name] = true
				allFiles = append(allFiles, file)
			}
		}
	}
	return allFiles, nil
}


func GetAtomFileInfo(atm atom.Atom) ([]FileInfo, error) {
	ca, ok := atm.(*AvailableVersion)
	if !ok {
		return nil, fmt.Errorf("expected %s to be an AvailableVersion", atm)
	}
	blob, err := ca.readFile("CONTENTS")
	if err != nil {
		return nil, err
	}
	if len(blob) == 0 {
		return []FileInfo{}, err
	}
	lines := strings.Split(blob, "\n")
	out := make([]FileInfo, 0, len(lines))
	for lineno, line := range lines {
		lineno++
		entry := FileInfo{}
		typeInd := line[:4]
		tail := line[4:]
		switch typeInd {
		case "dir ":
			entry.Type = FileType_dir
			entry.Name = tail
		case "obj ":
			entry.Type = FileType_file
			ts, tail, err := parseOffTimestamp(tail, lineno, ca)
			if err != nil {
				return nil, err
			}
			entry.UnixTime = ts
			md5, tail, err := parseOffMd5(tail, lineno, ca)
			if err != nil {
				return nil, err
			}
			copy(entry.MD5[:], md5)
			entry.Name = tail
		case "sym ":
			entry.Type = FileType_symlink
			ts, tail, err := parseOffTimestamp(tail, lineno, ca)
			if err != nil {
				return nil, err
			}
			entry.UnixTime = ts
			pos := strings.Index(tail, " -> ")
			if pos < 0 {
				return nil, fmt.Errorf("missing -> in CONTENTS line %d of %s",
					lineno, ca)
			}
			entry.Name = tail[:pos]
		default:
			return nil, fmt.Errorf("unknown object type %sin CONTENTS line %d of %s",
				typeInd, lineno, ca)
		}
		out = append(out, entry)
	}
	return out, nil
}


func parseOffTimestamp(tail string, lineno int, ca *AvailableVersion) (int64, string, error) {
	tail, str, err := parseOffNonBlankField(tail, lineno, ca)
	if err != nil {
		return 0, "", err
	}
	ts, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("%s in CONTENTS line %d of %s", err, lineno, ca)
	}
	return ts, tail, nil
}


func parseOffMd5(tail string, lineno int, ca *AvailableVersion) ([]byte, string, error) {
	tail, str, err := parseOffNonBlankField(tail, lineno, ca)
	if err != nil {
		return nil, "", err
	}
	bs, err := hex.DecodeString(str)
	if err != nil {
		return nil, "", fmt.Errorf("%s in CONTENTS line %d of %s", err, lineno, ca)
	}
	return bs, tail, nil
}


func parseOffNonBlankField(tail string, lineno int, ca *AvailableVersion) (string, string, error) {
	if len(tail) < 1 {
		return "", "", fmt.Errorf("empty CONTENTS line %d of %s", lineno, ca)
	}
	pos := len(tail) - 1
	found := false
	for pos > 0 {
		if tail[pos] == ' ' {
			found = true
			break
		}
		pos--
	}
	right := tail[pos+1:]
	if !found || len(right) == 0 {
		return "", "", fmt.Errorf("parse error in CONTENTS line %d of %s", lineno, ca)
	}
	return tail[:pos], right, nil
}

