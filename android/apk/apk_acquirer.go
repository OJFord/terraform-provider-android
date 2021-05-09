package repo

import (
	"fmt"
	aapt "github.com/shogo82148/androidbinary/apk"
	"log"
	"mvdan.cc/fdroidcl/adb"
)

type APKAcquirer interface {
	UpdateCache(*adb.Device) error
	GetApkPaths(*adb.Device, *int) ([]string, error)
	Apk() *Apk
}

type Apk struct {
	Name     string
	BasePath *string
	Paths    []string
}

func Package(method string, pkg string) (APKAcquirer, error) {
	apk := Apk{Name: pkg}
	var acq APKAcquirer

	switch method {
	case "aurora":
		acq = AuroraPackage{&apk}
	case "fdroid":
		acq = FDroidPackage{&apk}
	case "gplaycli":
		acq = GPlayCLIPackage{&apk}
	default:
		return nil, fmt.Errorf("Unknown APKAcquirer method: %s", method)
	}

	return acq, nil
}

func Version(apk APKAcquirer) (int, error) {
	if apk.Apk().BasePath == nil {
		return -1, fmt.Errorf("Expected %s to exist, but path unset", apk.Apk().Name)
	}

	pkg, err := aapt.OpenFile(*apk.Apk().BasePath)
	if err != nil {
		return -1, fmt.Errorf("Failed to read %s versionCode: %s", apk.Apk().Name, err)
	}

	v, err := pkg.Manifest().VersionCode.Int32()
	log.Printf("[INFO] %s versionCode is %d", apk.Apk().Name, v)
	return int(v), err
}

func VersionName(apk APKAcquirer) (string, error) {
	if apk.Apk().BasePath == nil {
		return "", fmt.Errorf("Expected %s to exist, but path unset", apk.Apk().Name)
	}

	pkg, err := aapt.OpenFile(*apk.Apk().BasePath)
	if err != nil {
		return "", fmt.Errorf("Failed to read %s versionName: %s", apk.Apk().Name, err)
	}

	v, err := pkg.Manifest().VersionName.String()
	log.Printf("[INFO] %s versionName is %s", apk.Apk().Name, v)
	return v, err
}
