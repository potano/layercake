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
var ProlificFsTypes = map[string]bool{
	"devtmpfs": true,
	"sysfs": true,
}

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

var ExportDirEntries map[string]string = map[string]string{
	"packages": "packages",
	"builds": "builds",
	"generated": "generated",
}

var MinimalBuildDirs = []string{"bin", "etc", "lib", "opt", "root", "sbin", "usr"}

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
