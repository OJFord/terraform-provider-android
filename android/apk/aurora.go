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

type AuroraPackage string

func (pkg AuroraPackage) Name() string {
	return string(pkg)
}

func (pkg AuroraPackage) Source() string {
	return "Google Play Store via Aurora"
}

func (pkg AuroraPackage) UpdateCache(device *adb.Device) (string, error) {
	apkDir, err := xdg.CacheFile("terraform-android/aurora")
	if err != nil {
		return "", err
	}

	if pkg.Name() == "com.aurora.store" {
		apkFname := fmt.Sprintf("%s/%s.apk", apkDir, pkg)
		if err = os.WriteFile(apkFname, comAuroraStoreApk, 0666); err != nil {
			return "", err
		}
		return apkFname, nil
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
	for !strings.Contains(string(stdout), pkg.Name()) {
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

	return fmt.Sprintf("%s/%s/%s/%s.apk", apkDir, pkg, versionDownloaded, pkg), nil
}
