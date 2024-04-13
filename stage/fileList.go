// Copyright © 2022 Michael Thompson
// SPDX-License-Identifier: GPL-2.0-or-later

package stage

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	unix "syscall"
	"potano.layercake/fs"
	"potano.layercake/fns"
	"potano.layercake/portage/vdb"
)


type FileList struct {
	Files []lineInfo
	rootDir string
	entryMap map[string]lineInfo
	inodes map[devIno]int32
}


func GenerateFileList(allFiles []vdb.FileInfo, rootDir string) (*FileList, error) {
	fl := &FileList{nil, path.Clean(rootDir), map[string]lineInfo{}, map[devIno]int32{}}
	for _, file := range allFiles {
		err := fl.addFiles(lineInfo{
			ltype: vdb.FileType_none,	// force type = "tbd"
			name: file.Name,
			skipIfAbsent: true})
		if err != nil {
			return nil, err
		}
	}
	return fl, nil
}


func (fl *FileList) Finalize() {
	allFiles := make([]lineInfo, 0, len(fl.entryMap))
	for _, entry := range fl.entryMap {
		allFiles = append(allFiles, entry)
	}
	sort.Slice(allFiles, func (i, j int) bool {
		return allFiles[i].name < allFiles[j].name
	})
	fl.Files = allFiles
	fl.entryMap = nil
	fl.fixHardlinks()
}


func (fl *FileList) Names() []string {
	names := make([]string, len(fl.Files))
	for i, entry := range fl.Files {
		names[i] = entry.name
	}
	return names
}


/* Format of the user-input file.  Contains one line per filesystem entry.  Comment lines begin
   with # or //

   Each non-blank, non-comment line contains a series of space-separated entries.  Spaces within
   names must be escaped with backslashes or by wrapping the line in quotes (single or double).
   General format of a line:  type name options

   The set of available or required options depends on the entry type.

   Type			 Notes
   ----			 -----

   file    Normal file	 Normally reads the contents to include from the named file in the source
			 tree but the user may specify a different source via the src= option.
			 Accepts the mod=, uid=, gid=, src=, and absent= options.
   dir     Directory	 Creates the directory in the stage tarball even if it does not exist
			 in the source tree.  Accepts the mod=, gid=, uid=, and absent= options.
   node    Device node	 Normally reads the node type and major/minor numbers from the named device
			 node in the source tree, but the user may specify a different node via the
			 src= option or supply these via the dev= option.  Accepts the mod=, uid=,
			 gid=, dev=, src=, and absent= options.
   symlink Symbolic link Unless the named link exists in the source tree, requires the targ= option
			 to specify the link target.  Accepts the targ= and absent= options.
   tbd     Undetermined  Creates a file, directory, or symlink in the stage tarball according to
			 the type of the existing object in the file system.  Accepts the absent=
			 option.
   omit    Omit file	 Indicatation to omit the named file from the stage tarball.


   The name argument indicates the name of the directory entry to be inserted into the stage
   tarball.  The base part of a name may contain an asterisk, in which case special rules apply
   for globbing.  If the entry type is 'dir', globbing applies recursively to that directory.
   For other entry types globbing is not recursive.  The src= option is unavailable when
   globbing is specified.

   Asterisks in the base part of names must be escaped with backslashes even if the names are
   wrapped in quotes.


   Options take the form name=value
   
   Name			Notes
   ----			-----

   mod    File mode	Permission bits of the file mode.  Value may take the forms the chmod(1)
			command allows:  an octal mask or a string of mode-setting characters.
   gid    Group ID	Integer group ID to apply to the file.  Note there is no option to use a
			group name instead:  there is no guaranteed correspondence between group
			names in the host system and those in the build root.
   uid    User ID	Integer user ID.  May be in the form N:N to specify both the GID and UID
			in the same operation.
   src    Source file	Name of source file, directory or device node to use as source data for
			the entry to be created.  Recursively copies source-directory entries if
			name is of a directory.  Names are relative to the current directory;
			absolute paths specify paths relative to the process' root directory.
			Prefix the path with $$stageroot to indicate paths relative to the root
			of the source tree.
   dev    Device	Type and major/minor numbers of the device in the form tN:N where t is
			either b (block) or c (character) and N is an integer.
   targ   Link target	Target of the symbolic link.  May be relative or absolute.  Absolute paths
			are relative to the root of the source tree.
   absent Excusable	Excusable absence:  ignore the item without error if the named object is
			not present in the filesystem.  Only legal value is 'skip'


   Examples:

   # Normal file with permissions and contents as they are in the source tree
   file /etc/portage/package.use

   # Normal file with contents taken from a file outside of the source tree
   file /etc/vim/vimrc src=/etc/vim/vimrc

   # Normal file with a space in the name wrapped in quote marks
   file "/usr/lib/python3.7/site-packages/setuptools/script (dev).tmpl"

   # Normal file with same name with backslash escape
   file /usr/lib/python3.7/site-packages/setuptools/script\ (dev).tmpl

   # Normal file which may be absent from the actual filesystem without causing fatal error
   file /etc/udev/rules.d/70-persistent-net.rules absent=skip

   # Zero-length normal file
   file /etc/udev/rules.d/70-persistent-net.rules src=/dev/null

   # All files in directory
   file /etc/portage/*

   # Filename containing an asterisk
   file /home/user/some\*name

   # Directory
   dir /dev

   # Directory in which contents are taken from a separate directory in the source tree
   dir /etc/portage src=$$stageroot/home/user/portage uid=0:0

   # Device nodes
   node /dev/sda dev=b8:0 uid=6
   node /dev/tty7 dev=c4:7 uid=5

   # Symbolic link
   symlink /usr/portage targ=/var/db/repos/gentoo

   # Filesystem entry of type to be determined at run time; skip if absent
   tbd /usr/tmp absent=skip
*/


func (fl *FileList) ReadUserFileList(cursor fs.LineReader) error {
	var line string
	for cursor.ReadLine(&line) {
		line = strings.TrimSpace(line)
		if len(line) < 1 || line[0] == '#' || (len(line) > 1 && line[:2] == "//") {
			continue
		}
		adding, entry, ok := parseLine(line, cursor)
		if ok {
			var err error
			if adding {
				err = fl.addFiles(entry)
			} else {
				err = fl.removeFiles(entry)
			}
			if err != nil {
				cursor.LogError(err.Error())
			}
		}
	}
	return cursor.Err()
}


type lineInfo struct {
	ltype uint8
	name, source, target string
	fsize, unixTime int64
	xattrs map[string]string
	gid, uid, andMask, orMask uint32
	devino int32
	major, minor uint32
	devtype byte
	hasWildcard, hasTarget, hasGid, hasUid, hasDev, hasPerm, skipIfAbsent bool
}


func parseLine(line string, cursor fs.LineReader) (bool, lineInfo, bool) {
	adding := true
	entry := lineInfo{}
	numErrors := len(cursor.GetMessages())
	fields, err := parseFields(line)

	var name, ltype string
	if len(fields) > 0 {
		ltype = fields[0]
	}
	if len(fields) > 1 {
		name = fields[1]
	}

	if len(ltype) < 1 {
		cursor.LogError("no file type")
	} else {
		switch ltype {
		case "file":
			entry.ltype = vdb.FileType_file
		case "dir":
			entry.ltype = vdb.FileType_dir
		case "node":
			entry.ltype = vdb.FileType_device
		case "symlink":
			entry.ltype = vdb.FileType_symlink
		case "tbd":
			entry.ltype = vdb.FileType_none
		case "omit":
			adding = false
		default:
			cursor.LogError("unknown file type " + ltype)
		}
	}

	if len(name) < 2 {
		err = fmt.Errorf("no file name")
	} else if name[0] != '/' {
		err = fmt.Errorf("name '%s' is not absolute", name)
	} else {
		entry.name, entry.hasWildcard, err = parseSource(name)
	}
	if err != nil {
		cursor.LogError(err.Error())
	}

	for i := 2; i < len(fields); i++ {
		str := fields[i]
		pos := strings.IndexByte(str, '=')
		if pos < 1 {
			cursor.LogError("could not parse option " + str)
			continue
		}
		name := str[:pos]
		val := str[pos+1:]
		var err error
		switch name {
		case "mod":
			err = processModOption(&entry, ltype, name, val)
		case "gid", "uid":
			err = processGidUidOption(&entry, ltype, name, val)
		case "src":
			err = processSourceOption(&entry, ltype, name, val)
		case "dev":
			err = processDevOption(&entry, ltype, name, val)
		case "targ":
			err = processTargOption(&entry, ltype, name, val)
		case "absent":
			err = processAbsentOption(&entry, ltype, name, val)
		default:
			err = fmt.Errorf("unknown option %s=", name)
		}
		if err != nil {
			cursor.LogError(err.Error())
		}
	}
	return adding, entry, len(cursor.GetMessages()) == numErrors
}


func parseFields(line string) ([]string, error) {
	var fields []string
	var field []byte
	var quote byte
	inField := false
	for p := 0; p < len(line); p++ {
		c := line[p]
		if inField {
			if (c == ' ' || c == '\t') && quote == 0 {
				if len(field) > 0 {
					fields = append(fields, string(field))
					field = field[:0]
				}
				inField = false
			} else if c == quote {
				quote = 0
			} else if c == '\\' {
				if len(line) > p {
					c2 := line[p+1]
					if c2 == '*' {
						field = append(field, c)
					}
					p++
					field = append(field, c2)
				}
			} else {
				field = append(field, c)
			}
		} else if c != ' ' && c != '\t' {
			quote = 0
			if c == '"' || c == '\'' {
				quote = c
			} else {
				field = append(field, c)
			}
			inField = true
		}
	}

	if len(field) > 0 {
		fields = append(fields, string(field))
		if quote > 0 {
			return nil, fmt.Errorf("unclosed quoted string")
		}
	}
	return fields, nil
}


func processModOption(entry *lineInfo, ltype, name, val string) error {
	err := optErrorIf(ltype, name, "symlink", "omit")
	if err != nil {
		return err
	}
	if entry.hasPerm {
		return fmt.Errorf("multiple settings of file permissions")
	}
	andMask, orMask, err := parseModString(val)
	if err != nil {
		return err
	}
	entry.andMask, entry.orMask = uint32(andMask), uint32(orMask)
	entry.hasPerm = true
	return nil
}


func processGidUidOption(entry *lineInfo, ltype, name, val string) error {
	err := optErrorIf(ltype, name, "symlink", "omit")
	if err != nil {
		return err
	}
	v1, v2, err := parseUid(val)
	if v2 < 0 {
		if name == "gid" {
			entry.gid = uint32(v1)
			entry.hasGid = true
		} else {
			entry.uid = uint32(v1)
			entry.hasUid = true
		}
	} else {
		entry.gid, entry.uid = uint32(v1), uint32(v2)
		entry.hasGid, entry.hasUid = true, true
	}
	return err
}


func processSourceOption(entry *lineInfo, ltype, name, val string) error {
	err := optErrorIf(ltype, name, "symlink", "omit")
	if err != nil {
		return err
	}
	if len(entry.source) > 0 || entry.devtype != 0 {
		return fmt.Errorf("duplicate setting of source")
	}
	if entry.hasWildcard {
		return fmt.Errorf("filename cannot have wildcard when src= option given")
	}
	entry.source, entry.hasWildcard, err = parseSource(val)
	return err
}


func processDevOption(entry *lineInfo, ltype, name, val string) error {
	err := optErrorIf(ltype, name, "file", "dir", "symlink", "omit")
	if err != nil {
		return err
	}
	if len(entry.source) > 0 || entry.devtype != 0 {
		return fmt.Errorf("duplicate setting of source")
	}
	entry.devtype, entry.major, entry.minor, err = parseDev(val)
	entry.hasDev = true
	return err
}


func processTargOption(entry *lineInfo, ltype, name, val string) error {
	err := optErrorIf(ltype, name, "file", "dir", "node", "omit")
	if err != nil {
		return err
	}
	path, wildcard, err := parseSource(val)
	if err != nil {
		return err
	}
	if wildcard {
		return fmt.Errorf("symlink target %s is a wildcard", val)
	}
	entry.target = path
	return nil
}


func processAbsentOption(entry *lineInfo, ltype, name, val string) error {
	err := optErrorIf(ltype, name, "omit")
	if err != nil {
		return err
	}
	entry.skipIfAbsent, err = parseAbsent(val)
	return err
}



func optErrorIf(curLtype, option string, forbiddenLtypes ...string) error {
	for _, test := range forbiddenLtypes {
		if test == curLtype {
			return fmt.Errorf("cannot use %s option with type '%s'", option, test)
		}
	}
	return nil
}



func parseModString(str string) (andMask int32, orMask int32, err error) {
	haveOctal := true
	for _, c := range []byte(str) {
		if !isOctal[c] {
			haveOctal = false
			break
		}
	}
	if haveOctal {
		var v int64
		v, err = strconv.ParseInt(str, 8, 32)
		orMask = int32(v)
		return
	}
	addOrRemove := 0
	andMask = vdb.PermBits
	var groupMask, settingMask int32
	for _, c := range []byte(str) {
		if mask, have := groupMasks[c]; have {
			if groupMask > 0 || settingMask > 0 || addOrRemove != 0 {
				goto maskError
			}
			groupMask = mask
		} else if c == '+' || c == '-' {
			if groupMask == 0 {
				groupMask = groupMasks['a']
			}
			if settingMask > 0 || addOrRemove != 0 {
				goto maskError
			}
			if c == '-' {
				addOrRemove = -1
			} else {
				addOrRemove = 1
			}
		} else if mask, have := settingMasks[c]; have {
			if groupMask == 0 {
				groupMask = groupMasks['a']
			}
			if addOrRemove == 0 {
				addOrRemove = 1
			}
			if settingMask > 0 {
				goto maskError
			}
			settingMask = mask & groupMask
			if addOrRemove > 0 {
				orMask |= settingMask
			} else {
				andMask &= vdb.PermBits ^ settingMask
			}
		} else if c == ',' {
			addOrRemove, groupMask, settingMask = 0, 0, 0
		} else {
			goto maskError
		}
	}
	return

	maskError:
	err = fmt.Errorf("bad mode setting %s", str)
	return
}


func parseUid(str string) (v1 int64, v2 int64, err error) {
	pos := strings.IndexByte(str, ':')
	if pos < 0 {
		v1, err = strconv.ParseInt(str, 10, 32)
		if err != nil || v1 < 0 {
			goto parseError
		}
		v2 = -1
	} else {
		v1, err = strconv.ParseInt(str[:pos], 10, 32)
		if err != nil || v1 < 0 {
			goto parseError
		}
		v2, err = strconv.ParseInt(str[pos+1:], 10, 32)
		if err != nil || v2 < 0 {
			goto parseError
		}
	}
	return

	parseError:
	err = fmt.Errorf("invalid UID/GID")
	return
}


func parseSource(str string) (string, bool, error) {
	if len(str) == 0 {
		return "", false, fmt.Errorf("zero-length file name")
	}
	// Detect an unescaped asterisk in a path element before the last one
	haveWildcard := false
	backslashPos := -2
	for i, c := range str {
		if c == '/' {
			if haveWildcard {
				return "", false, fmt.Errorf("%s has a wildcard parent directory",
					str)
			}
		} else if c == '\\' {
			backslashPos = i
		} else if c == '*' && i - 1 != backslashPos {
			haveWildcard = true
		}
	}
	return str, haveWildcard, nil
}


func parseDev(str string) (devType byte, major uint32, minor uint32, err error) {
	if len(str) == 0 {
		goto parseError
	}
	devType = str[0]
	if devType != 'c' && devType != 'b' {
		goto parseError
	} else {
		var v1, v2 uint64
		parts := strings.Split(str[1:], ":")
		if len(parts) != 2 {
			goto parseError
		}
		v1, err = strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			goto parseError
		}
		major = uint32(v1)
		v2, err = strconv.ParseUint(parts[1], 10, 8)
		if err != nil {
			goto parseError
		}
		major = uint32(v1)
		minor = uint32(v2)
	}
	return

	parseError:
	err = fmt.Errorf("could not parse device ID %s", str)
	return
}


func parseAbsent(str string) (allowAbsent bool, err error) {
	if str == "skip" {
		allowAbsent = true
	} else {
		err = fmt.Errorf("illegal value for absent= option")
	}
	return
}



var isOctal fns.CharTypeMap
var groupMasks, settingMasks map[byte]int32


func init() {
	isOctal = fns.MakeCharTypeMap("0-7")

	groupMasks = map[byte]int32 {
		'u': unix.S_IRWXU | unix.S_ISUID,	// user r/w/x + set UID
		'g': unix.S_IRWXG | unix.S_ISGID,	// group r/w/x + set GID
		'o': unix.S_IRWXO,			// others r/w/x
		'a': vdb.PermBits,			// all users
	}

	settingMasks = map[byte]int32 {
		'r': unix.S_IRUSR | unix.S_IRGRP | unix.S_IROTH,	// read permission
		'w': unix.S_IWUSR | unix.S_IWGRP | unix.S_IWOTH,	// write permission
		'x': unix.S_IXUSR | unix.S_IXGRP | unix.S_IXOTH,	// execute permission
		's': unix.S_ISUID | unix.S_ISGID,			// set UID or GID
		't': unix.S_ISVTX,					// sticky bit
	}
}

