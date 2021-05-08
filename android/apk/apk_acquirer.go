package repo

import (
	"fmt"
	aapt "github.com/shogo82148/androidbinary/apk"
	"log"
	"mvdan.cc/fdroidcl/adb"
)

type APKAcquirer interface {
	UpdateCache(*adb.Device) (string, error)
	Apk() Apk
}

type Apk struct {
	Name string
	Path *string
}

func Package(method string, pkg string) (APKAcquirer, error) {
	apk := Apk{Name: pkg}
	switch method {
	case "aurora":
		return AuroraPackage{apk}, nil
	case "fdroid":
		return FDroidPackage{apk}, nil
	case "gplaycli":
		return GPlayCLIPackage{apk}, nil
	default:
		return nil, fmt.Errorf("Unknown APKAcquirer method: %s", method)
	}
}

func Version(apk APKAcquirer) (int, error) {
	pkg, err := aapt.OpenFile(*apk.Apk().Path)
	if err != nil {
		return -1, fmt.Errorf("Failed to read %s versionCode: %s", apk.Apk().Name, err)
	}

	v, err := pkg.Manifest().VersionCode.Int32()
	log.Printf("%s versionCode is %d", apk.Apk().Name, v)
	return int(v), err
}

func VersionName(apk APKAcquirer) (string, error) {
	pkg, err := aapt.OpenFile(*apk.Apk().Path)
	if err != nil {
		return "", fmt.Errorf("Failed to read %s versionName: %s", apk.Apk().Name, err)
	}

	v, err := pkg.Manifest().VersionName.String()
	log.Printf("%s versionName is %s", apk.Apk().Name, v)
	return v, err
}
