package repo

import (
	"fmt"
	"github.com/adrg/xdg"
	"log"
	"mvdan.cc/fdroidcl/adb"
	"os/exec"
	"regexp"
	"strconv"
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
	cmd := exec.Command("aapt2", "dump", "badging", Path(apk))
	stdouterr, err := cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if err != nil {
		return -1, fmt.Errorf("Failed to read %s versionCode: %s", apk.Name(), stdouterr)
	}

	re_vcode := regexp.MustCompile(`versionCode='(\d+)'`)
	matches := re_vcode.FindStringSubmatch(string(stdouterr))
	if len(matches) == 0 {
		return -1, fmt.Errorf("Failed to find %s's versionCode", apk.Name())
	}
	v, err := strconv.ParseInt(string(matches[1]), 10, 32)
	if err != nil {
		return -1, err
	}

	return int(v), nil
}

func VersionName(apk APKAcquirer) (string, error) {
	cmd := exec.Command("aapt2", "dump", "badging", Path(apk))
	stdouterr, err := cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if err != nil {
		return "", fmt.Errorf("Failed to read %s versionName: %s", apk.Name(), stdouterr)
	}

	re_vname := regexp.MustCompile(`versionName='([^']+)'`)
	matches := re_vname.FindStringSubmatch(string(stdouterr))
	if len(matches) == 0 {
		return "", fmt.Errorf("Failed to find %s's versionName", apk.Name())
	}

	return string(matches[1]), nil
}
