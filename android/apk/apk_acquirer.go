package repo

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
)

type APKAcquirer interface {
	Name() string
	Source() string
	UpdateCache(string) (string, error)
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

func Version(apk APKAcquirer, device_codename string) (int, error) {
	file, err := apk.UpdateCache(device_codename)
	if err != nil {
		return -1, err
	}

	cmd := exec.Command("aapt", "dump", "badging", file)
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

func VersionName(apk APKAcquirer, device_codename string) (string, error) {
	file, err := apk.UpdateCache(device_codename)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("aapt", "dump", "badging", file)
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
