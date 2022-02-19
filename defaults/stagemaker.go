package defaults

const Umask = 0022
const StageFileGID = 0
const StageFileUID = 0

const TreeRootDirPrefixName = "stageroot"

const MaxSymlinkChain = 5

const DoNotTraverse = "/boot /dev /home /media /mnt /proc /run /usr/portage /sys /var/db cache tmp"

const StageMagic =
`dir /boot
dir /dev
file /etc/csh.env
dir /etc/env.d/*
file /etc/fstab
file /etc/group
file /etc/gshadow
file /etc/ld.so.cache
file /etc/ld.so.conf
dir /etc/ld.so.conf.d/*
tbd /etc/localtime
symlink /etc/mtab targ=/proc/self/mounts
file /etc/passwd
dir /etc/portage/*
file /etc/profile.env
file /etc/shadow
file /etc/udev/hwdb.bin
dir /etc/xml/*
dir /home
dir /media
dir /mnt
dir /opt
dir /proc
dir /root
dir /run
dir /sys
dir /tmp
file /usr/bin/c89
file /usr/bin/c99
file /usr/lib/cracklib_dict.hwm
file /usr/lib/cracklib_dict.pwd
file /usr/lib/cracklib_dict.pwi
file /usr/lib64/gconv/gconv-modules.cache
file /usr/lib64/locale/locale-archive
dir /usr/local/*
file /usr/sbin/fix_libtool_files.sh absent=skip
dir /usr/share/binutils-data/*
dir /usr/share/gcc-data/*
file /usr/share/info/dir
dir /usr/src
tbd /usr/tmp
file /var/cache/*
dir /var/db/repos
dir /var/empty
dir /var/lib/gentoo/*
dir /var/lib/portage/*
tbd /var/lock
symlink /var/run
dir /var/spool
dir /var/tmp
`

const DevDirSetup =
`node /dev/console dev=c5:1 mod=0600
node /dev/core dev=c1:6 mod=0600
symlink /dev/fd targ=/proc/self/fd
node /dev/full dev=c1:7 mod=0666
node /dev/hda dev=b3:0 gid=6 mod=0640
dir /dev/input mod=0755
node /dev/input/event0 dev=c13:64 mod=0600
node /dev/input/js0 dev=c13:0 mod=0600
node /dev/input/keyboard dev=c10:150 mod=0600
node /dev/input/mice dev=c13:63 mod=0600
node /dev/input/mouse dev=c10:149 mod=0600
node /dev/input/mouse0 dev=c13:32 mod=0600
node /dev/input/uinput dev=c10:223 mod=0600
node /dev/mem dev=c1:1 gid=9 mod=0640
node /dev/null dev=c1:3 mod=0666
node /dev/port dev=c1:4 gid=9 mod=0640
node /dev/ptmx dev=c5:2 mod=0666
node /dev/random dev=c1:8 mod=0644
node /dev/sda dev=b8:0 gid=6 mod=0640
node /dev/sdb dev=b8:16 gid=6 mod=0640
node /dev/sdc dev=b8:32 gid=6 mod=0640
node /dev/sdd dev=b8:48 gid=6 mod=0640
symlink /dev/stderr targ=../proc/self/fd/2
symlink /dev/stdin targ=../proc/self/fd/0
symlink /dev/stdout targ=../proc/self/fd/1
node /dev/tty dev=c5:0 mod=0666
node /dev/tty0 dev=c4:0 gid=5 mod=0640
node /dev/urandom dev=c1:9 mod=0644
node /dev/zero dev=c1:5 mod=0666
`

const DevDirExtend =
`/dev/hda 32
/dev/input/event0 31
/dev/input/js0 31
/dev/input/mouse0 30
/dev/sda 15
/dev/sdb 15
/dev/sdc 15
/dev/sdd 15
/dev/tty0 63
`

const GzipExtensions = ".tar.gz .tgz"
const BzipExtensions = ".tar.bz2 .tbz2"
const XzExtensions = "tar.xz"
const NoCompressExtension = ".tar"

const GzipExecutable = "gzip"
const BzipExecutable = "bzip2"
const XzExecutable = "xz"

