package repo

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"mvdan.cc/fdroidcl/adb"

	_ "embed"
)

//go:embed AuroraStore/app/build/outputs/apk/release/app-release-unsigned.apk
var comAuroraStoreApk []byte

type AuroraPackage struct {
	apk Apk
}

func (pkg AuroraPackage) Apk() Apk {
	return pkg.apk
}

func (pkg AuroraPackage) UpdateCache(device *adb.Device) (string, error) {
	apkDir, err := xdg.CacheFile("terraform-android/aurora")
	if err != nil {
		return "", err
	}

	err = os.MkdirAll(apkDir, 0775)
	if err != nil {
		return "", err
	}

	if pkg.apk.Name == "com.aurora.store" {
		apkPath := fmt.Sprintf("%s/%s.apk", apkDir, pkg)
		pkg.apk.Path = &apkPath
		if err = os.WriteFile(apkPath, comAuroraStoreApk, 0666); err != nil {
			return "", err
		}
		return apkPath, nil
	}

	cmd := device.AdbCmd(
		"shell",
		"am",
		"start",
		"-n", "com.aurora.store/com.aurora.store.view.ui.details.AppDetailsActivity",
		"-d", fmt.Sprintf("market://?id=%s\\&download", pkg),
	)

	stdouterr, err := cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if strings.Contains(string(stdouterr), "Activity class {com.aurora.store/com.aurora.store.view.ui.details.AppDetailsActivity} does not exist") {
		return "", fmt.Errorf("Failed to trigger download for %s: is `com.aurora.store` installed?", pkg)
	}
	if err != nil {
		return "", fmt.Errorf("Failed to trigger download for %s: %s", pkg, stdouterr)
	}

	var stdout []byte
	for !strings.Contains(string(stdout), pkg.apk.Name) {
		time.Sleep(3 * time.Second)
		cmd = device.AdbCmd("shell", "ls", "sdcard/Aurora/Store/Downloads/")
		stdout, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}

	cmd = device.AdbCmd("shell", "ls", fmt.Sprintf("sdcard/Aurora/Store/Downloads/%s", pkg))
	versionDownloaded, err := cmd.Output()
	if err != nil {
		return "", err
	}

	cmd = device.AdbCmd("pull", fmt.Sprintf("sdcard/Aurora/Store/Downloads/%s/%s", pkg, versionDownloaded), fmt.Sprintf("%s/%s/", apkDir, pkg))
	stdouterr, err = cmd.CombinedOutput()
	log.Println(string(stdouterr))
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve %s: %s", pkg, stdouterr)
	}

	apkPath := fmt.Sprintf("%s/%s/%s/%s.apk", apkDir, pkg, versionDownloaded, pkg)
	pkg.apk.Path = &apkPath
	return *pkg.apk.Path, nil
}
