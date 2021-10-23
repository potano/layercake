package defaults


const DefaultCommand = "status"

const BasePath = "/var/lib/layercake"
const MainConfigFile = "config"
const Layerdirs = "layers"
const Builddir = "build"
const Workdir = "overlayfs/workdir"
const Upperdir = "overlayfs/upperdir"
const Generateddir = "generated"
const Exportdirs = "export"
const ChrootExec = "/usr/bin/chroot"

const MountinfoPath = "/proc/self/mountinfo"
const ShadowingFsTypes = "devtmpfs sysfs"

const LayerconfigFile = "layerconfig"
const SkeletonLayerconfigFile = "default_layerconfig.skel"
const SkeletonLayerconfigFileExt = ".skel"
const SkeletonLayerconfig =
`import rbind /dev /dev
import proc proc /proc
import rbind /sys /sys
import rbind /var/db/repos/gentoo /var/db/repos/gentoo
import rbind /var/cache/distfiles /var/cache/distfiles

export symlink /var/cache/binpkgs packages`

const ExportDirEntries = "packages:packages|builds:builds|generated:generated"

const MinimalBuildDirs = "bin etc lib opt root sbin usr"

const RemovedLayerSuffix = "~removed"

const ExportIndexHtmlName = "index.html"
const ExportIndexHtml = `<!DOCTYPE html>
<html>
   <head>
      <title>binpackager</title>
   </head>
   <body>
      <h1>binpackager</h1>
      <div>Serves prebuilt Gentoo packages</div>
   </body>
</html>

`

const BaseLayerRootBashrc = `#!/bin/bash

source /etc/profile
msg=chroot
if [ -n "$LAYERCAKE_LAYER" ]; then
        msg="chroot $LAYERCAKE_LAYER"
fi
export PS1="($msg) \[\033]0;\u@\h:\w\007\]\[\033[01;31m\]\h\[\033[01;34m\] \w \$\[\033[00m\] "

`

