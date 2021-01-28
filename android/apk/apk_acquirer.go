package repo

import (
	"fmt"
	"github.com/adrg/xdg"
	aapt "github.com/shogo82148/androidbinary/apk"
	"log"
	"mvdan.cc/fdroidcl/adb"
)

type APKAcquirer interface {
	Name() string
	Source() string
	UpdateCache(*adb.Device) (string, error)
}

func Package(method string, pkg string) (APKAcquirer, error) {
	switch method {
	case "gplaycli":
		return GPlayCLIPackage(pkg), nil
	case "fdroid":
		return FDroidPackage(pkg), nil
	default:
		return nil, fmt.Errorf("Unknown APKAcquirer method: %s", method)
	}
}

func Path(apk APKAcquirer) string {
	path, err := xdg.CacheFile(fmt.Sprint("terraform-android/", apk.Name(), ".apk"))
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	return path
}

func Version(apk APKAcquirer) (int, error) {
	pkg, err := aapt.OpenFile(Path(apk))
	if err != nil {
		return -1, fmt.Errorf("Failed to read %s versionCode: %s", apk.Name(), err)
	}

	v, err := pkg.Manifest().VersionCode.Int32()
	log.Printf("%s versionCode is %d", apk.Name(), v)
	return int(v), err
}

func VersionName(apk APKAcquirer) (string, error) {
	pkg, err := aapt.OpenFile(Path(apk))
	if err != nil {
		return "", fmt.Errorf("Failed to read %s versionName: %s", apk.Name(), err)
	}

	v, err := pkg.Manifest().VersionName.String()
	log.Printf("%s versionName is %s", apk.Name(), v)
	return v, err
}
