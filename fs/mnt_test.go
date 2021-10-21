package fs

import (
	"fmt"
	"strings"

	"testing"
)


// Images of /mnt/self/mountinfo from various systems

// Fresh Alpine-Linux system: devtmpfs and sysfs w/ submounts; portage in root filesystem
var alpine_fresh =
`16 21 0:4 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
17 21 0:16 / /sys rw,nosuid,nodev,noexec,relatime - sysfs sysfs rw
18 21 0:6 / /dev rw,nosuid,relatime - devtmpfs devtmpfs rw,size=10240k,nr_inodes=125935,mode=755
19 18 0:17 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=000
20 18 0:18 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw
21 0 8:3 / / rw,relatime - ext4 /dev/sda3 rw,data=ordered
22 21 0:19 / /run rw,nodev,relatime - tmpfs tmpfs rw,size=101660k,mode=755
23 18 0:15 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
24 17 0:20 / /sys/kernel/security rw,nosuid,nodev,noexec,relatime - securityfs securityfs rw
25 17 0:8 / /sys/kernel/debug rw,nosuid,nodev,noexec,relatime - debugfs debugfs rw
26 17 0:21 / /sys/fs/pstore rw,nosuid,nodev,noexec,relatime - pstore pstore rw
27 25 0:9 / /sys/kernel/debug/tracing rw,nosuid,nodev,noexec,relatime - tracefs tracefs rw
29 21 8:1 / /boot rw,relatime - ext4 /dev/sda1 rw,data=ordered
`
var alpine_fresh_replay =
`device 0:4 proc
device 0:6 devtmpfs
device 0:8 debugfs shadowed
device 0:9 tracefs shadowed
device 0:15 mqueue shadowed
device 0:16 sysfs
device 0:17 devpts shadowed
device 0:18 tmpfs shadowed
device 0:19 tmpfs
device 0:20 securityfs shadowed
device 0:21 pstore shadowed
device 8:1 ext4
device 8:3 ext4
mount 0:4 / /proc rw,nosuid,nodev,noexec,relatime
mount 0:6 / /dev rw,nosuid,relatime
mount 0:8 / /sys/kernel/debug rw,nosuid,nodev,noexec,relatime
mount 0:9 / /sys/kernel/debug/tracing rw,nosuid,nodev,noexec,relatime
mount 0:15 / /dev/mqueue rw,nosuid,nodev,noexec,relatime
mount 0:16 / /sys rw,nosuid,nodev,noexec,relatime
mount 0:17 / /dev/pts rw,nosuid,noexec,relatime
mount 0:18 / /dev/shm rw,nosuid,nodev,noexec,relatime
mount 0:19 / /run rw,nodev,relatime
mount 0:20 / /sys/kernel/security rw,nosuid,nodev,noexec,relatime
mount 0:21 / /sys/fs/pstore rw,nosuid,nodev,noexec,relatime
mount 8:1 / /boot rw,relatime
mount 8:3 / / rw,relatime`


// Same Alpine-Linux system with single layer mounted
var alpine_base1 =
`16 21 0:4 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
17 21 0:16 / /sys rw,nosuid,nodev,noexec,relatime - sysfs sysfs rw
18 21 0:6 / /dev rw,nosuid,relatime - devtmpfs devtmpfs rw,size=10240k,nr_inodes=125935,mode=755
19 18 0:17 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=000
20 18 0:18 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw
21 0 8:3 / / rw,relatime - ext4 /dev/sda3 rw,data=ordered
22 21 0:19 / /run rw,nodev,relatime - tmpfs tmpfs rw,size=101660k,mode=755
23 18 0:15 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
24 17 0:20 / /sys/kernel/security rw,nosuid,nodev,noexec,relatime - securityfs securityfs rw
25 17 0:8 / /sys/kernel/debug rw,nosuid,nodev,noexec,relatime - debugfs debugfs rw
26 17 0:21 / /sys/fs/pstore rw,nosuid,nodev,noexec,relatime - pstore pstore rw
27 25 0:9 / /sys/kernel/debug/tracing rw,nosuid,nodev,noexec,relatime - tracefs tracefs rw
29 21 8:1 / /boot rw,relatime - ext4 /dev/sda1 rw,data=ordered
30 21 0:6 / /var/lib/layercake/layers/base1/build/dev rw,nosuid,relatime - devtmpfs devtmpfs rw,size=10240k,nr_inodes=125935,mode=755
31 30 0:17 / /var/lib/layercake/layers/base1/build/dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=000
32 30 0:18 / /var/lib/layercake/layers/base1/build/dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw
33 30 0:15 / /var/lib/layercake/layers/base1/build/dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
34 21 0:4 / /var/lib/layercake/layers/base1/build/proc rw,relatime - proc proc rw
35 21 0:16 / /var/lib/layercake/layers/base1/build/sys rw,nosuid,nodev,noexec,relatime - sysfs sysfs rw
36 35 0:20 / /var/lib/layercake/layers/base1/build/sys/kernel/security rw,nosuid,nodev,noexec,relatime - securityfs securityfs rw
37 35 0:8 / /var/lib/layercake/layers/base1/build/sys/kernel/debug rw,nosuid,nodev,noexec,relatime - debugfs debugfs rw
38 37 0:9 / /var/lib/layercake/layers/base1/build/sys/kernel/debug/tracing rw,nosuid,nodev,noexec,relatime - tracefs tracefs rw
39 35 0:21 / /var/lib/layercake/layers/base1/build/sys/fs/pstore rw,nosuid,nodev,noexec,relatime - pstore pstore rw
40 21 8:3 /var/db/repos/gentoo /var/lib/layercake/layers/base1/build/var/db/repos/gentoo rw,relatime - ext4 /dev/sda3 rw,data=ordered
41 21 8:3 /var/cache/distfiles /var/lib/layercake/layers/base1/build/var/cache/distfiles rw,relatime - ext4 /dev/sda3 rw,data=ordered
`
var alpine_base1_replay =
`device 0:4 proc
device 0:6 devtmpfs
device 0:8 debugfs shadowed
device 0:9 tracefs shadowed
device 0:15 mqueue shadowed
device 0:16 sysfs
device 0:17 devpts shadowed
device 0:18 tmpfs shadowed
device 0:19 tmpfs
device 0:20 securityfs shadowed
device 0:21 pstore shadowed
device 8:1 ext4
device 8:3 ext4
mount 0:4 / /proc rw,nosuid,nodev,noexec,relatime
mount 0:4 / /var/lib/layercake/layers/base1/build/proc rw,relatime
mount 0:6 / /dev rw,nosuid,relatime
mount 0:6 / /var/lib/layercake/layers/base1/build/dev rw,nosuid,relatime
mount 0:8 / /sys/kernel/debug rw,nosuid,nodev,noexec,relatime
mount 0:8 / /var/lib/layercake/layers/base1/build/sys/kernel/debug rw,nosuid,nodev,noexec,relatime
mount 0:9 / /sys/kernel/debug/tracing rw,nosuid,nodev,noexec,relatime
mount 0:9 / /var/lib/layercake/layers/base1/build/sys/kernel/debug/tracing rw,nosuid,nodev,noexec,relatime
mount 0:15 / /dev/mqueue rw,nosuid,nodev,noexec,relatime
mount 0:15 / /var/lib/layercake/layers/base1/build/dev/mqueue rw,nosuid,nodev,noexec,relatime
mount 0:16 / /sys rw,nosuid,nodev,noexec,relatime
mount 0:16 / /var/lib/layercake/layers/base1/build/sys rw,nosuid,nodev,noexec,relatime
mount 0:17 / /dev/pts rw,nosuid,noexec,relatime
mount 0:17 / /var/lib/layercake/layers/base1/build/dev/pts rw,nosuid,noexec,relatime
mount 0:18 / /dev/shm rw,nosuid,nodev,noexec,relatime
mount 0:18 / /var/lib/layercake/layers/base1/build/dev/shm rw,nosuid,nodev,noexec,relatime
mount 0:19 / /run rw,nodev,relatime
mount 0:20 / /sys/kernel/security rw,nosuid,nodev,noexec,relatime
mount 0:20 / /var/lib/layercake/layers/base1/build/sys/kernel/security rw,nosuid,nodev,noexec,relatime
mount 0:21 / /sys/fs/pstore rw,nosuid,nodev,noexec,relatime
mount 0:21 / /var/lib/layercake/layers/base1/build/sys/fs/pstore rw,nosuid,nodev,noexec,relatime
mount 8:1 / /boot rw,relatime
mount 8:3 / / rw,relatime
mount 8:3 /var/db/repos/gentoo /var/lib/layercake/layers/base1/build/var/db/repos/gentoo rw,relatime
mount 8:3 /var/cache/distfiles /var/lib/layercake/layers/base1/build/var/cache/distfiles rw,relatime`


// Same Alpine-Linux system with base layer base1 and overlay layer der1 mounted
var alpine_der1 =
`16 21 0:4 / /proc rw,nosuid,nodev,noexec,relatime - proc proc rw
17 21 0:16 / /sys rw,nosuid,nodev,noexec,relatime - sysfs sysfs rw
18 21 0:6 / /dev rw,nosuid,relatime - devtmpfs devtmpfs rw,size=10240k,nr_inodes=125935,mode=755
19 18 0:17 / /dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=000
20 18 0:18 / /dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw
21 0 8:3 / / rw,relatime - ext4 /dev/sda3 rw,data=ordered
22 21 0:19 / /run rw,nodev,relatime - tmpfs tmpfs rw,size=101660k,mode=755
23 18 0:15 / /dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
24 17 0:20 / /sys/kernel/security rw,nosuid,nodev,noexec,relatime - securityfs securityfs rw
25 17 0:8 / /sys/kernel/debug rw,nosuid,nodev,noexec,relatime - debugfs debugfs rw
26 17 0:21 / /sys/fs/pstore rw,nosuid,nodev,noexec,relatime - pstore pstore rw
27 25 0:9 / /sys/kernel/debug/tracing rw,nosuid,nodev,noexec,relatime - tracefs tracefs rw
29 21 8:1 / /boot rw,relatime - ext4 /dev/sda1 rw,data=ordered
30 21 0:6 / /var/lib/layercake/layers/base1/build/dev rw,nosuid,relatime - devtmpfs devtmpfs rw,size=10240k,nr_inodes=125935,mode=755
31 30 0:17 / /var/lib/layercake/layers/base1/build/dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=000
32 30 0:18 / /var/lib/layercake/layers/base1/build/dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw
33 30 0:15 / /var/lib/layercake/layers/base1/build/dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
34 21 0:4 / /var/lib/layercake/layers/base1/build/proc rw,relatime - proc proc rw
35 21 0:16 / /var/lib/layercake/layers/base1/build/sys rw,nosuid,nodev,noexec,relatime - sysfs sysfs rw
36 35 0:20 / /var/lib/layercake/layers/base1/build/sys/kernel/security rw,nosuid,nodev,noexec,relatime - securityfs securityfs rw
37 35 0:8 / /var/lib/layercake/layers/base1/build/sys/kernel/debug rw,nosuid,nodev,noexec,relatime - debugfs debugfs rw
38 37 0:9 / /var/lib/layercake/layers/base1/build/sys/kernel/debug/tracing rw,nosuid,nodev,noexec,relatime - tracefs tracefs rw
39 35 0:21 / /var/lib/layercake/layers/base1/build/sys/fs/pstore rw,nosuid,nodev,noexec,relatime - pstore pstore rw
40 21 8:3 /var/db/repos/gentoo /var/lib/layercake/layers/base1/build/var/db/repos/gentoo rw,relatime - ext4 /dev/sda3 rw,data=ordered
41 21 8:3 /var/cache/distfiles /var/lib/layercake/layers/base1/build/var/cache/distfiles rw,relatime - ext4 /dev/sda3 rw,data=ordered
42 21 0:23 / /var/lib/layercake/layers/der1/build rw,relatime - overlay overlay rw,lowerdir=/var/lib/layercake/layers/base1/build,upperdir=/var/lib/layercake/layers/der1/overlayfs/upperdir,workdir=/var/lib/layercake/layers/der1/overlayfs/workdir
45 42 0:6 / /var/lib/layercake/layers/der1/build/dev rw,nosuid,relatime - devtmpfs devtmpfs rw,size=10240k,nr_inodes=125935,mode=755
46 45 0:17 / /var/lib/layercake/layers/der1/build/dev/pts rw,nosuid,noexec,relatime - devpts devpts rw,gid=5,mode=620,ptmxmode=000
47 45 0:18 / /var/lib/layercake/layers/der1/build/dev/shm rw,nosuid,nodev,noexec,relatime - tmpfs shm rw
48 45 0:15 / /var/lib/layercake/layers/der1/build/dev/mqueue rw,nosuid,nodev,noexec,relatime - mqueue mqueue rw
49 42 0:4 / /var/lib/layercake/layers/der1/build/proc rw,relatime - proc proc rw
50 42 0:16 / /var/lib/layercake/layers/der1/build/sys rw,nosuid,nodev,noexec,relatime - sysfs sysfs rw
51 50 0:20 / /var/lib/layercake/layers/der1/build/sys/kernel/security rw,nosuid,nodev,noexec,relatime - securityfs securityfs rw
52 50 0:8 / /var/lib/layercake/layers/der1/build/sys/kernel/debug rw,nosuid,nodev,noexec,relatime - debugfs debugfs rw
53 52 0:9 / /var/lib/layercake/layers/der1/build/sys/kernel/debug/tracing rw,nosuid,nodev,noexec,relatime - tracefs tracefs rw
54 50 0:21 / /var/lib/layercake/layers/der1/build/sys/fs/pstore rw,nosuid,nodev,noexec,relatime - pstore pstore rw
55 42 8:3 /var/db/repos/gentoo /var/lib/layercake/layers/der1/build/var/db/repos/gentoo rw,relatime - ext4 /dev/sda3 rw,data=ordered
56 42 8:3 /var/cache/distfiles /var/lib/layercake/layers/der1/build/var/cache/distfiles rw,relatime - ext4 /dev/sda3 rw,data=ordered
`
var alpine_der1_replay =
`device 0:4 proc
device 0:6 devtmpfs
device 0:8 debugfs shadowed
device 0:9 tracefs shadowed
device 0:15 mqueue shadowed
device 0:16 sysfs
device 0:17 devpts shadowed
device 0:18 tmpfs shadowed
device 0:19 tmpfs
device 0:20 securityfs shadowed
device 0:21 pstore shadowed
device 0:23 overlay
device 8:1 ext4
device 8:3 ext4
mount 0:4 / /proc rw,nosuid,nodev,noexec,relatime
mount 0:4 / /var/lib/layercake/layers/base1/build/proc rw,relatime
mount 0:4 / /var/lib/layercake/layers/der1/build/proc rw,relatime
mount 0:6 / /dev rw,nosuid,relatime
mount 0:6 / /var/lib/layercake/layers/base1/build/dev rw,nosuid,relatime
mount 0:6 / /var/lib/layercake/layers/der1/build/dev rw,nosuid,relatime
mount 0:8 / /sys/kernel/debug rw,nosuid,nodev,noexec,relatime
mount 0:8 / /var/lib/layercake/layers/base1/build/sys/kernel/debug rw,nosuid,nodev,noexec,relatime
mount 0:8 / /var/lib/layercake/layers/der1/build/sys/kernel/debug rw,nosuid,nodev,noexec,relatime
mount 0:9 / /sys/kernel/debug/tracing rw,nosuid,nodev,noexec,relatime
mount 0:9 / /var/lib/layercake/layers/base1/build/sys/kernel/debug/tracing rw,nosuid,nodev,noexec,relatime
mount 0:9 / /var/lib/layercake/layers/der1/build/sys/kernel/debug/tracing rw,nosuid,nodev,noexec,relatime
mount 0:15 / /dev/mqueue rw,nosuid,nodev,noexec,relatime
mount 0:15 / /var/lib/layercake/layers/base1/build/dev/mqueue rw,nosuid,nodev,noexec,relatime
mount 0:15 / /var/lib/layercake/layers/der1/build/dev/mqueue rw,nosuid,nodev,noexec,relatime
mount 0:16 / /sys rw,nosuid,nodev,noexec,relatime
mount 0:16 / /var/lib/layercake/layers/base1/build/sys rw,nosuid,nodev,noexec,relatime
mount 0:16 / /var/lib/layercake/layers/der1/build/sys rw,nosuid,nodev,noexec,relatime
mount 0:17 / /dev/pts rw,nosuid,noexec,relatime
mount 0:17 / /var/lib/layercake/layers/base1/build/dev/pts rw,nosuid,noexec,relatime
mount 0:17 / /var/lib/layercake/layers/der1/build/dev/pts rw,nosuid,noexec,relatime
mount 0:18 / /dev/shm rw,nosuid,nodev,noexec,relatime
mount 0:18 / /var/lib/layercake/layers/base1/build/dev/shm rw,nosuid,nodev,noexec,relatime
mount 0:18 / /var/lib/layercake/layers/der1/build/dev/shm rw,nosuid,nodev,noexec,relatime
mount 0:19 / /run rw,nodev,relatime
mount 0:20 / /sys/kernel/security rw,nosuid,nodev,noexec,relatime
mount 0:20 / /var/lib/layercake/layers/base1/build/sys/kernel/security rw,nosuid,nodev,noexec,relatime
mount 0:20 / /var/lib/layercake/layers/der1/build/sys/kernel/security rw,nosuid,nodev,noexec,relatime
mount 0:21 / /sys/fs/pstore rw,nosuid,nodev,noexec,relatime
mount 0:21 / /var/lib/layercake/layers/base1/build/sys/fs/pstore rw,nosuid,nodev,noexec,relatime
mount 0:21 / /var/lib/layercake/layers/der1/build/sys/fs/pstore rw,nosuid,nodev,noexec,relatime
mount 0:23 / /var/lib/layercake/layers/der1/build rw,relatime /var/lib/layercake/layers/base1/build /var/lib/layercake/layers/der1/overlayfs/upperdir /var/lib/layercake/layers/der1/overlayfs/workdir
mount 8:1 / /boot rw,relatime
mount 8:3 / / rw,relatime
mount 8:3 /var/db/repos/gentoo /var/lib/layercake/layers/base1/build/var/db/repos/gentoo rw,relatime
mount 8:3 /var/db/repos/gentoo /var/lib/layercake/layers/der1/build/var/db/repos/gentoo rw,relatime
mount 8:3 /var/cache/distfiles /var/lib/layercake/layers/base1/build/var/cache/distfiles rw,relatime
mount 8:3 /var/cache/distfiles /var/lib/layercake/layers/der1/build/var/cache/distfiles rw,relatime
`


func makeInputCursor(filename, text string) *TextInputCursor {
	reader := strings.NewReader(text)
	return NewTextInputCursor(filename, reader)
}


func buildMountTree(filename, setup string) (Mounts, error) {
	mount_list := make([]MountType, 0, 100)
	device_list := make([]deviceType, 0, 20)
	device_devices := map[string]string{}
	mounts := map[string]*MountType{}
	devices := map[string]*deviceType{}
	shadowed_devices := map[string]bool{}

	cursor := makeInputCursor(filename, setup)
	defer cursor.Close()
	var line string
	for cursor.ReadLine(&line) {
		parts := strings.Split(line, " ")
		if len(parts) < 3 {
			return Mounts{}, fmt.Errorf("%s has too few fields: ", line)
		}
		op, st_dev, mp := parts[0], parts[1], parts[2]
		if op == "device" {
			device_list = append(device_list, deviceType{name: st_dev})
			device_devices[st_dev] = mp
			if len(parts) > 3 && parts[3] == "shadowed" {
				shadowed_devices[st_dev] = true
			}
		} else if op == "mount" {
			if len(parts) < 5 {
				return Mounts{}, fmt.Errorf("missing mount field(s): %s", line)
			}
			mnt := MountType{Mountpoint: parts[3], Options: parts[4], st_dev: st_dev,
				root: mp}
			if len(parts) > 5 {
				mnt.Source = parts[5]
				if len(parts) > 6 {
					mnt.Source2 = parts[6]
					if len(parts) > 7 {
						mnt.Workdir = parts[7]
					}
				}
			}
			mount_list = append(mount_list, mnt)
		} else {
			return Mounts{}, fmt.Errorf("unknown setup operation %s", op)
		}
	}
	for i, dev := range device_list {
		devices[dev.name] = &device_list[i]
	}
	for i, mnt := range mount_list {
		dev := devices[mnt.st_dev]
		if dev == nil {
			return Mounts{}, fmt.Errorf("no device %s for mountpoint %s", mnt.st_dev,
				mnt.Mountpoint)
		}
		mp := &mount_list[i]
		mounts[mnt.Mountpoint] = mp
		mp.Fstype = device_devices[mnt.st_dev]
		if mnt.root == "/" {
			dev.roots = append(dev.roots, mnt.Mountpoint)
			if shadowed_devices[mnt.st_dev] {
				mp.InShadow = true
			}
		}
	}
	return Mounts{mount_list, device_list, mounts, devices}, nil
}


func stringSlicesHaveSameMembers(sl1, sl2 []string) bool {
	for _, v1 := range sl1 {
		ok := false
		for _, v2 := range sl2 {
			if v1 == v2 {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}


func compareMounts(mounts1, mounts2 Mounts) error {
	if len(mounts1.mount_list) != len(mounts2.mount_list) {
		return fmt.Errorf("expected mount list length %d, got %d",
			len(mounts1.mount_list), len(mounts2.mount_list))
	}
	if len(mounts1.device_list) != len(mounts2.device_list) {
		return fmt.Errorf("expected device list length %d, got %d",
			len(mounts1.device_list), len(mounts2.device_list))
	}
	if len(mounts1.mounts) != len(mounts2.mounts) {
		return fmt.Errorf("expected mount map length %d, got %d)",
			len(mounts1.mounts), len(mounts2.mounts))
	}
	if len(mounts1.devices) != len(mounts2.devices) {
		return fmt.Errorf("expected device map length %d, got %d",
			len(mounts1.mounts), len(mounts2.mounts))
	}
	for k, v1 := range mounts1.devices {
		v2, ok := mounts2.devices[k]
		if !ok {
			return fmt.Errorf("expected to find device %s", k)
		}
		if len(v1.roots) != len(v2.roots) {
			return fmt.Errorf("expected device %s to have %d roots, found %d", k,
				len(v1.roots), len(v2.roots))
		}
		if !stringSlicesHaveSameMembers(v1.roots, v2.roots) {
			return fmt.Errorf("expected device %s to have roots %s, found %s", k,
				strings.Join(v1.roots, " "), strings.Join(v2.roots, " "))
		}
	}
	for k, v1 := range mounts1.mounts {
		v2, ok := mounts2.mounts[k]
		if !ok {
			return fmt.Errorf("expected to find mountpoint %s", k)
		}
		if v1.Source != v2.Source {
			return fmt.Errorf("expected to find Source %s for mountpoint %s, found %s",
				v1.Source, k, v2.Source)
		}
		if v1.Mountpoint != v2.Mountpoint {
			return fmt.Errorf("expected to find mountpoint %s, found %s",
				k, v2.Mountpoint)
		}
		if v1.Source2 != v2.Source2 {
			return fmt.Errorf("expected to find Source2 %s for mountpoint %s, found %s",
				v1.Source2, k, v2.Source2)
		}
		if v1.Workdir != v2.Workdir {
			return fmt.Errorf("expected to find Workdir %s for mountpoint %s, found %s",
				v1.Workdir, k, v2.Workdir)
		}
		if v1.Fstype != v2.Fstype {
			return fmt.Errorf("expected to find Fstype %s for mountpoint %s, found %s",
				v1.Fstype, k, v2.Fstype)
		}
		if v1.Options != v2.Options {
			return fmt.Errorf("expected to find Options %s for mountpoint %s, found %s",
				v1.Options, k, v2.Options)
		}
		if v1.InShadow != v2.InShadow {
			return fmt.Errorf(
				"expected to find InShadow %t for mountpoint %s, found %t",
				v1.InShadow, k, v2.InShadow)
		}
		if v1.st_dev != v2.st_dev {
			return fmt.Errorf("expected to find st_dev %s for mountpoint %s, found %s",
				v1.st_dev, k, v2.st_dev)
		}
		if v1.root != v2.root {
			return fmt.Errorf("expected to find root %s for mountpoint %s, found %s",
				v1.root, k, v2.root)
		}
	}
	return nil
}


func displayDevices(devs map[string]*deviceType) {
	for mp, dev := range devs {
		fmt.Printf("   %s (%s): %s\n", mp, dev.name,
		strings.Join(dev.roots, ", "))
	}
}


func displayMounts(mnts []MountType) {
	for _, info := range mnts {
		displayMount(info)
	}
}

func displayMount(mnt MountType) {
	fmt.Printf("   %s (%s, %s)\n", mnt.Mountpoint, mnt.st_dev, mnt.Fstype)
	fmt.Printf("      Source: %s\n", mnt.Source)
	fmt.Printf("      Source2: %s\n", mnt.Source2)
	fmt.Printf("      Workdir: %s\n", mnt.Workdir)
	fmt.Printf("      Options: %s\n", mnt.Options)
	fmt.Printf("      InShadow: %t\n", mnt.InShadow)
	fmt.Printf("      st_dev: %s\n", mnt.st_dev)
	fmt.Printf("      Root: %s\n", mnt.root)
}


func TestMounts(t *testing.T) {
	for _, tst := range []struct{
		name, blob, setup string
	}{
		{"fresh alpine", alpine_fresh, alpine_fresh_replay},
		{"alpine single base mount", alpine_base1, alpine_base1_replay},
		{"alpine derived mount", alpine_der1, alpine_der1_replay},
	} {
		t.Run(tst.name, func (t *testing.T) {
			AlternateProbeMountsCursor = makeInputCursor(tst.name, tst.blob)
			mounts, err := ProbeMounts()
			if err != nil {
				t.Fatalf("while probing mouns: %s", err.Error())
			}
			wantmounts, err := buildMountTree(tst.name, tst.setup)
			if err != nil {
				t.Fatalf("while replaying setup: %s", err.Error())
			}
			err = compareMounts(wantmounts, mounts)
			if err != nil {
				t.Fatal(err.Error())
			}
			return
			fmt.Printf("For %s\n", tst.name)
			displayDevices(mounts.devices)
			displayMounts(mounts.mount_list)
		})
	}
}

