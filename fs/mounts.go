package fs

import (
	"path"
	"sort"
	"strings"

	"potano.layercake/defaults"
)


type MountType struct {
	Source, Mountpoint string
	Source2, Workdir string
	Fstype string
	Options string
	InShadow bool
	st_dev string
	root string
}

type deviceType struct {
	name string
	roots []string
}

type Mounts struct {
	mount_list []MountType
	device_list []deviceType
	mounts map[string]*MountType
	devices map[string]*deviceType
}

type mountList []*MountType

func (ml mountList) Len() int {
	return len(ml)
}

func (ml mountList) Swap(i, j int) {
	ml[j], ml[i] = ml[i], ml[j];
}

func (ml mountList) Less(i, j int) bool {
	return ml[i].Mountpoint < ml[j].Mountpoint
}


var AlternateProbeMountsCursor LineReader


/*  Extracts mount information from /proc/self/mountinfo
 *  Line format is documented in Documentation/filesystems/proc.txt in the Linux tarball:
 *    36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue
 *    (1)(2)(3)   (4)   (5)      (6)      (7)   (8) (9)   (10)         (11)

 *    (1) mount ID:  unique identifier of the mount (may be reused after umount)
 *    (2) parent ID:  ID of parent (or of self for the top of the mount tree)
 *    (3) major:minor:  value of st_dev for files on filesystem
 *    (4) root:  root of the mount within the filesystem
 *    (5) mount point:  mount point relative to the process's root
 *    (6) mount options:  per mount options
 *    (7) optional fields:  zero or more fields of the form "tag[:value]"
 *    (8) separator:  marks the end of the optional fields
 *    (9) filesystem type:  name of filesystem of the form "type[.subtype]"
 *    (10) mount source:  filesystem specific information or "none"
 *    (11) super options:  per super block options
 */
func ProbeMounts() (Mounts, error) {
	cursor := AlternateProbeMountsCursor
	var err error
	if cursor == nil {
		cursor, err = NewTextInputFileCursor(defaults.MountinfoPath)
		if err != nil {
			return Mounts{}, err
		}
	}

	mount_list := make([]MountType, 0, 100)
	device_list := make([]deviceType, 0, 20)
	mounts := map[string]*MountType{}
	devices := map[string]*deviceType{}
	shadowing_fs_types := map[string]bool{}
	shadowing_parents := map[string]bool{}

	for _, tp := range strings.Split(defaults.ShadowingFsTypes, " ") {
		shadowing_fs_types[tp] = true
	}

	defer cursor.Close()
	var line string
	for cursor.ReadLine(&line) {
		segments := strings.Split(line, " ")
		if len(segments) < 10 {
			continue
		}
		mountID := segments[0]
		parentID := segments[1]
		st_dev := segments[2]
		root := unescape(segments[3])
		mtpoint := unescape(segments[4])
		options := segments[5]
		var i int
		for i = 6; segments[i] != "-"; i++ {}
		fstype := segments[i + 1]
		fsname := unescape(segments[i + 2])

		var source, source2, workdir string
		var inShadow bool

		// Skip submounts of /dev and /sys
		if shadowing_fs_types[fstype] {
			shadowing_parents[mountID] = true
		} else if shadowing_parents[parentID] {
			shadowing_parents[mountID] = true
			inShadow = true
		}

		if fstype == "overlay" {
			for _, part := range strings.Split(segments[i + 3], ",") {
				kv := strings.SplitN(part, "=", 2)
				if len(kv) > 1 {
					switch kv[0] {
					case "lowerdir":
						source = unescape(kv[1])
					case "upperdir":
						source2 = unescape(kv[1])
					case "workdir":
						workdir = unescape(kv[1])
					}
				}
			}
		}
		mount_list = append(mount_list, MountType{source, mtpoint, source2, workdir,
			fstype, options, inShadow, st_dev, root})
		mounts[mtpoint] = &mount_list[len(mount_list) - 1]

		var device *deviceType
		device, ok := devices[st_dev]
		if !ok {
			device_list = append(device_list, deviceType{fsname, []string{}})
			device = &device_list[len(device_list) - 1]
			devices[st_dev] = device
		}
		if root == "/" {
			device.roots = append(device.roots, mtpoint)
		}

	}
	return Mounts{mount_list, device_list, mounts, devices}, nil
}


func (m Mounts) GetMount(path string) *MountType {
	return m.mounts[path]
}

func (m Mounts) GetMountAndSubmounts(path string) []*MountType {
	list := make(mountList, 0, 20)
	if mnt := m.mounts[path]; mnt != nil {
		list = append(list, mnt)
	}
	path += "/"
	for mtpoint, mnt := range m.mounts {
		if strings.HasPrefix(mtpoint, path) {
			list = append(list, mnt)
		}
	}
	sort.Sort(list)
	return list
}


func (m Mounts) GetMountSources(mnt *MountType) []string {
	device := m.devices[mnt.st_dev]
	out := make([]string, 0, len(device.roots) + 1)
	if len(mnt.Source) > 0 {
		out = append(out, mnt.Source)
	} else {
		root := mnt.root
		if root == "/" {
			out = append(out, device.name)
			root = ""
		}
		for _, mp := range device.roots {
			src := mp + root
			if src != mnt.Mountpoint {
				out = append(out, path.Join(mp, root))
			}
		}
	}
	return out
}


func (m Mounts) MountSourceIsExpected(mnt *MountType, test string) bool {
	for _, path := range m.GetMountSources(mnt) {
		if path == test {
			return true
		}
	}
	return false
}


func unescape(str string) string {
	out := []byte(str)
	outp := 0
	octal := -1
	for _, c := range out {
		if c == '\\' {
			octal = 0
			continue
		}
		if octal > -1 {
			if c >= '0' && c < '8' && octal < 32 {
				octal = octal * 8 + int(c - '0')
				continue
			}
			c = byte(octal)
			octal = -1
		}
		out[outp] = c
		outp++
	}
	return string(out[:outp])
}

